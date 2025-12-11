package app

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/formatters"
	"github.com/alecthomas/chroma/v2/lexers"
	chromaStyles "github.com/alecthomas/chroma/v2/styles"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
	"github.com/EspenTeigen/lazylab/internal/config"
	"github.com/EspenTeigen/lazylab/internal/gitlab"
	"github.com/EspenTeigen/lazylab/internal/keymap"
	"github.com/EspenTeigen/lazylab/internal/ui/components"
	"github.com/EspenTeigen/lazylab/internal/ui/styles"
)

// copyToClipboard copies text to the system clipboard
func copyToClipboard(text string) error {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("pbcopy")
	case "linux":
		// Try wl-copy for Wayland, then xclip/xsel for X11
		if _, err := exec.LookPath("wl-copy"); err == nil {
			cmd = exec.Command("wl-copy")
		} else if _, err := exec.LookPath("xclip"); err == nil {
			cmd = exec.Command("xclip", "-selection", "clipboard")
		} else {
			cmd = exec.Command("xsel", "--clipboard", "--input")
		}
	case "windows":
		cmd = exec.Command("clip")
	default:
		return fmt.Errorf("unsupported platform")
	}

	pipe, err := cmd.StdinPipe()
	if err != nil {
		return err
	}

	if err := cmd.Start(); err != nil {
		return err
	}

	_, err = pipe.Write([]byte(text))
	if err != nil {
		return err
	}

	pipe.Close()
	return cmd.Wait()
}

// ansiRegex matches ANSI escape sequences
var ansiRegex = regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]`)

// highlightCode applies syntax highlighting to code based on filename
// truncateString truncates a string to maxLen, adding "..." if truncated
func truncateString(s string, maxLen int) string {
	if maxLen <= 3 {
		maxLen = 10
	}
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

// timeAgo formats a time as a human-readable relative time
func timeAgo(t time.Time) string {
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		m := int(d.Minutes())
		if m == 1 {
			return "1m ago"
		}
		return fmt.Sprintf("%dm ago", m)
	case d < 24*time.Hour:
		h := int(d.Hours())
		if h == 1 {
			return "1h ago"
		}
		return fmt.Sprintf("%dh ago", h)
	case d < 7*24*time.Hour:
		days := int(d.Hours() / 24)
		if days == 1 {
			return "1d ago"
		}
		return fmt.Sprintf("%dd ago", days)
	case d < 30*24*time.Hour:
		weeks := int(d.Hours() / 24 / 7)
		if weeks == 1 {
			return "1w ago"
		}
		return fmt.Sprintf("%dw ago", weeks)
	default:
		months := int(d.Hours() / 24 / 30)
		if months == 1 {
			return "1mo ago"
		}
		return fmt.Sprintf("%dmo ago", months)
	}
}

func highlightCode(code, filename string) string {
	// Get lexer based on filename
	lexer := lexers.Match(filename)
	if lexer == nil {
		lexer = lexers.Analyse(code)
	}
	if lexer == nil {
		lexer = lexers.Fallback
	}
	lexer = chroma.Coalesce(lexer)

	// Use a dark terminal-friendly style
	style := chromaStyles.Get("monokai")
	if style == nil {
		style = chromaStyles.Fallback
	}

	// Use terminal256 formatter for ANSI output
	formatter := formatters.Get("terminal256")
	if formatter == nil {
		formatter = formatters.Fallback
	}

	iterator, err := lexer.Tokenise(nil, code)
	if err != nil {
		return code
	}

	var buf bytes.Buffer
	err = formatter.Format(&buf, style, iterator)
	if err != nil {
		return code
	}

	return buf.String()
}

// stripANSI removes ANSI escape codes from a string
func stripANSI(s string) string {
	return ansiRegex.ReplaceAllString(s, "")
}

// Custom markdown style based on dark theme
var markdownStyle = []byte(`{
	"document": {
		"block_prefix": "",
		"block_suffix": "",
		"margin": 0
	},
	"block_quote": {
		"indent": 2,
		"color": "244"
	},
	"list": {
		"level_indent": 2
	},
	"heading": {
		"block_suffix": "\n",
		"bold": true
	},
	"h1": {
		"bold": true,
		"color": "39",
		"block_suffix": "\n"
	},
	"h2": {
		"bold": true,
		"color": "39",
		"block_suffix": "\n"
	},
	"h3": {
		"bold": true,
		"color": "39",
		"block_suffix": "\n"
	},
	"h4": {
		"bold": true,
		"color": "39",
		"block_suffix": "\n"
	},
	"h5": {
		"bold": true,
		"color": "39",
		"block_suffix": "\n"
	},
	"h6": {
		"bold": true,
		"color": "39",
		"block_suffix": "\n"
	},
	"text": {},
	"strikethrough": {
		"crossed_out": true
	},
	"emph": {
		"italic": true
	},
	"strong": {
		"bold": true
	},
	"hr": {
		"color": "240",
		"format": "\n--------\n"
	},
	"item": {
		"block_prefix": "• "
	},
	"enumeration": {
		"block_prefix": ". "
	},
	"task": {
		"ticked": "[✓] ",
		"unticked": "[ ] "
	},
	"link": {
		"color": "30",
		"underline": true
	},
	"link_text": {
		"color": "35",
		"bold": true
	},
	"image": {
		"color": "212",
		"underline": true
	},
	"image_text": {
		"color": "243",
		"format": "Image: {{.text}} →"
	},
	"code": {
		"color": "203"
	},
	"code_block": {
		"color": "244",
		"margin": 1,
		"chroma": {
			"text": { "color": "#C4C4C4" },
			"keyword": { "color": "#F92672" },
			"name": { "color": "#F8F8F2" },
			"literal": { "color": "#E6DB74" },
			"string": { "color": "#E6DB74" },
			"number": { "color": "#AE81FF" },
			"operator": { "color": "#F92672" },
			"comment": { "color": "#75715E" }
		}
	},
	"table": {
		"center_separator": "┼",
		"column_separator": "│",
		"row_separator": "─"
	},
	"definition_list": {},
	"definition_term": {},
	"definition_description": {
		"block_prefix": "\n"
	},
	"html_block": {},
	"html_span": {}
}`)

// renderMarkdown renders markdown content for terminal display
func renderMarkdown(content string, width int) string {
	if content == "" {
		return ""
	}

	if width <= 0 {
		width = 80
	}

	renderer, err := glamour.NewTermRenderer(
		glamour.WithStylesFromJSONBytes(markdownStyle),
		glamour.WithWordWrap(width),
	)
	if err != nil {
		return content // Fall back to raw content
	}

	rendered, err := renderer.Render(content)
	if err != nil {
		return content // Fall back to raw content
	}

	return strings.TrimSpace(rendered)
}

// isBinaryContent checks if content appears to be binary
func isBinaryContent(content string) bool {
	// Check first 8KB for null bytes (strong indicator of binary)
	checkLen := 8192
	if len(content) < checkLen {
		checkLen = len(content)
	}
	for i := 0; i < checkLen; i++ {
		if content[i] == 0 {
			return true
		}
	}
	return false
}

// isBinaryExtension checks if file extension indicates binary
func isBinaryExtension(path string) bool {
	binaryExts := map[string]bool{
		".exe": true, ".dll": true, ".so": true, ".dylib": true, ".a": true, ".o": true,
		".bin": true, ".dat": true, ".db": true, ".sqlite": true,
		".zip": true, ".tar": true, ".gz": true, ".bz2": true, ".xz": true, ".7z": true, ".rar": true,
		".png": true, ".jpg": true, ".jpeg": true, ".gif": true, ".bmp": true, ".ico": true, ".webp": true,
		".pdf": true, ".doc": true, ".docx": true, ".xls": true, ".xlsx": true, ".ppt": true, ".pptx": true,
		".mp3": true, ".mp4": true, ".avi": true, ".mkv": true, ".mov": true, ".wav": true, ".flac": true,
		".ttf": true, ".otf": true, ".woff": true, ".woff2": true, ".eot": true,
		".pyc": true, ".pyo": true, ".class": true, ".jar": true, ".war": true,
		".wasm": true, ".node": true,
	}
	path = strings.ToLower(path)
	for ext := range binaryExts {
		if strings.HasSuffix(path, ext) {
			return true
		}
	}
	return false
}

// hardTruncate cuts a string to fit within maxWidth visual characters
func hardTruncate(s string, maxWidth int) string {
	if maxWidth <= 0 {
		return ""
	}
	width := lipgloss.Width(s)
	if width <= maxWidth {
		return s
	}
	// Cut rune by rune until we fit
	runes := []rune(s)
	for len(runes) > 0 {
		runes = runes[:len(runes)-1]
		if lipgloss.Width(string(runes)) <= maxWidth {
			return string(runes)
		}
	}
	return ""
}

// sliceByWidth returns a substring starting at visual offset and fitting within maxWidth
func sliceByWidth(s string, offset, maxWidth int) string {
	if maxWidth <= 0 {
		return ""
	}
	if offset <= 0 {
		return hardTruncate(s, maxWidth)
	}

	// Strip ANSI codes for width calculation, but we need to preserve them
	// For simplicity, strip them entirely when scrolling horizontally
	clean := stripANSI(s)

	// Skip 'offset' visual characters
	runes := []rune(clean)
	skipped := 0
	startIdx := 0
	for i, r := range runes {
		if skipped >= offset {
			startIdx = i
			break
		}
		skipped += lipgloss.Width(string(r))
		startIdx = i + 1
	}

	if startIdx >= len(runes) {
		return ""
	}

	// Take up to maxWidth characters
	result := []rune{}
	currentWidth := 0
	for i := startIdx; i < len(runes); i++ {
		runeWidth := lipgloss.Width(string(runes[i]))
		if currentWidth+runeWidth > maxWidth {
			break
		}
		result = append(result, runes[i])
		currentWidth += runeWidth
	}

	return string(result)
}

// wrapText wraps all lines in text to fit within maxWidth
func wrapText(text string, maxWidth int) string {
	if maxWidth <= 0 {
		return text
	}

	lines := strings.Split(text, "\n")
	var result []string

	for _, line := range lines {
		if lipgloss.Width(line) <= maxWidth {
			result = append(result, line)
			continue
		}

		// Wrap long line using rune-based approach for proper width handling
		var currentLine []rune
		currentWidth := 0

		for _, r := range line {
			runeWidth := lipgloss.Width(string(r))
			if currentWidth+runeWidth > maxWidth {
				// Line would exceed maxWidth, wrap here
				if len(currentLine) > 0 {
					result = append(result, string(currentLine))
				}
				currentLine = []rune{r}
				currentWidth = runeWidth
			} else {
				currentLine = append(currentLine, r)
				currentWidth += runeWidth
			}
		}
		if len(currentLine) > 0 {
			result = append(result, string(currentLine))
		}
	}

	return strings.Join(result, "\n")
}

// PanelID identifies panels in the UI
type PanelID int

const (
	PanelNavigator PanelID = iota
	PanelContent
	PanelReadme
)

// TreeNode represents an item in the navigator tree
type TreeNode struct {
	Type     string // "group" or "project"
	Name     string
	FullPath string
	ID       int
	Depth    int
	Expanded bool
	Group    *gitlab.Group
	Project  *gitlab.Project
}

// ContentTab identifies tabs in the content panel
type ContentTab int

const (
	TabFiles ContentTab = iota
	TabMRs
	TabPipelines
	TabReleases
	TabCount
)

var contentTabNames = []string{"Files", "MRs", "Pipelines", "Releases"}

// MainScreen is the lazygit-style multi-panel interface
type MainScreen struct {
	// GitLab client
	client *gitlab.Client

	// Navigator tree
	treeNodes       []TreeNode
	selectedNodeIdx int
	expandedGroups  map[int]bool             // group ID -> expanded
	groupProjects   map[int][]gitlab.Project // group ID -> projects (cache)

	// Raw data
	groups        []gitlab.Group
	files         []gitlab.TreeEntry
	mergeRequests []gitlab.MergeRequest
	pipelines     []gitlab.Pipeline
	releases      []gitlab.Release
	branches      []gitlab.Branch
	jobs          []gitlab.Job
	jobLog        string

	// Jobs per pipeline (for showing stages in list)
	pipelineJobs map[int][]gitlab.Job

	// Selected project
	selectedProject *gitlab.Project

	// File browser state
	currentPath     []string
	fileContent     string
	readmeContent   string
	readmeRendered  string
	viewingFile     bool
	viewingFilePath string

	// Selection indices
	selectedContent int

	// Focus
	focusedPanel PanelID

	// Content tab
	contentTab ContentTab

	// Dimensions
	width  int
	height int

	// Keymaps
	keymap keymap.KeyMap

	// Loading states
	loading    bool
	loadingMsg string
	errMsg     string

	// Viewports for scrolling
	readmeViewport viewport.Model
	jobLogViewport viewport.Model
	fileViewport   viewport.Model
	readmeReady    bool
	jobLogReady    bool
	fileViewReady  bool

	// README visual mode
	readmeCursor       int
	readmeLastKey      string
	readmeVisualMode   bool
	readmeVisualStart  int
	readmeVisualEnd    int

	// Job selection for pipelines
	selectedJobIdx int

	// Scroll offset for file list (keeps selected item visible)
	fileScrollOffset int

	// Job log popup
	showJobLogPopup   bool
	currentPipelineID int // Pipeline ID for job refresh

	// Branch selector popup
	showBranchPopup   bool
	selectedBranchIdx int
	currentBranch     string

	// Status message (for clipboard feedback etc)
	statusMsg string

	// Error handling
	lastError string
	retryCmd  tea.Cmd // Command to retry on 'r' key

	// Job log popup focus (true = log panel, false = job list)
	jobLogFocused    bool
	jobLogCursor     int    // Current cursor line in log
	jobLogHScroll    int    // Horizontal scroll offset
	jobLogLastKey    string // Last key pressed (for sequences like yy, gg)
	visualLineMode   bool   // Visual line selection active
	visualStartLine  int    // Start of visual selection
	visualEndLine    int    // End of visual selection (follows cursor)

	// Runners popup (shows all running/pending jobs across projects)
	showRunnersPopup bool
	runningJobs      []gitlab.Job
	pendingJobs      []gitlab.Job
	runnersLoading   bool
	runnersLastKey   string
	runnersCursor    int
	runnersTab       int // 0 = running, 1 = pending

	// Release assets popup
	showReleasePopup    bool
	selectedReleaseIdx  int // Index of selected release for popup
	releaseAssetCursor  int // Cursor position in assets list
	releaseScrollOffset int // Scroll offset for assets list

	// Folder browser for downloads
	showFolderBrowser    bool
	folderBrowserPath    string   // Current path in folder browser
	folderBrowserEntries []string // Directory entries (folders only)
	folderBrowserCursor  int      // Selected entry index
	folderBrowserScroll  int      // Scroll offset
	downloadURL          string   // URL to download after folder selection
	downloadFilename     string   // Filename for the download

	// Demo mode (no API calls)
	isDemo bool
}

// NewMainScreen creates a new main screen
func NewMainScreen() *MainScreen {
	token, host := loadCredentials()
	client := createClient(host, token)

	return &MainScreen{
		client:         client,
		focusedPanel:   PanelNavigator,
		contentTab:     TabFiles,
		keymap:         keymap.DefaultKeyMap(),
		expandedGroups: make(map[int]bool),
		groupProjects:  make(map[int][]gitlab.Project),
	}
}

// loadCredentials loads GitLab credentials from env vars, lazylab config, or glab config
func loadCredentials() (token, host string) {
	// 1. Check environment variables (highest priority)
	token = os.Getenv(config.EnvGitLabToken)
	host = os.Getenv(config.EnvGitLabHost)

	// 2. Fall back to lazylab config
	if token == "" || host == "" {
		if lazylabConfig, err := config.LoadLazyLabConfig(); err == nil {
			if host == "" {
				host = lazylabConfig.GetDefaultHost()
			}
			if hostConfig := lazylabConfig.GetHostConfig(host); hostConfig != nil {
				if token == "" {
					token = hostConfig.Token
				}
			}
		}
	}

	// 3. Fall back to glab config
	if token == "" || host == "" {
		if glabConfig, err := config.LoadGlabConfig(); err == nil {
			if host == "" {
				host = glabConfig.GetDefaultHost()
			}
			if hostConfig := glabConfig.GetHostConfig(host); hostConfig != nil {
				if token == "" {
					token = hostConfig.Token
				}
			}
		}
	}

	// Apply defaults
	if host == "" {
		host = config.DefaultHost
	}

	// Ensure host has protocol
	if !strings.HasPrefix(host, "http") {
		host = "https://" + host
	}

	return token, host
}

// HasCredentials checks if valid credentials are available
func HasCredentials() bool {
	token, _ := loadCredentials()
	return token != ""
}

// createClient creates a GitLab client with the given credentials
func createClient(host, token string) *gitlab.Client {
	if token != "" {
		return gitlab.NewClient(host, token)
	}
	return gitlab.NewPublicClient()
}

// rebuildNavTree rebuilds the flat tree representation from groups and their projects
func (m *MainScreen) rebuildNavTree() {
	m.treeNodes = nil

	for _, g := range m.groups {
		// Add group node
		groupNode := TreeNode{
			Type:     "group",
			Name:     g.Name,
			FullPath: g.FullPath,
			ID:       g.ID,
			Depth:    0,
			Expanded: m.expandedGroups[g.ID],
			Group:    &g,
		}
		m.treeNodes = append(m.treeNodes, groupNode)

		// If expanded, add projects
		if m.expandedGroups[g.ID] {
			if projects, ok := m.groupProjects[g.ID]; ok {
				for _, p := range projects {
					projectNode := TreeNode{
						Type:     "project",
						Name:     p.Name,
						FullPath: p.PathWithNamespace,
						ID:       p.ID,
						Depth:    1,
						Project:  &p,
					}
					m.treeNodes = append(m.treeNodes, projectNode)
				}
			}
		}
	}
}

// Init initializes the screen
func (m *MainScreen) Init() tea.Cmd {
	// Demo mode has pre-loaded data, no API calls needed
	if m.isDemo {
		return nil
	}
	m.loading = true
	m.loadingMsg = "Loading groups..."
	cmd := m.loadGroups()
	m.retryCmd = cmd
	return cmd
}

func (m *MainScreen) loadGroups() tea.Cmd {
	if m.isDemo {
		return nil
	}
	return func() tea.Msg {
		groups, err := m.client.ListGroups()
		if err != nil {
			return errMsg{err: err}
		}
		return groupsLoadedMsg{groups: groups}
	}
}

func (m *MainScreen) loadGroupProjects(groupID int, groupPath string) tea.Cmd {
	if m.isDemo {
		return nil
	}
	return func() tea.Msg {
		projects, err := m.client.ListGroupProjects(groupPath)
		if err != nil {
			return errMsg{err: err}
		}
		return groupProjectsLoadedMsg{groupID: groupID, projects: projects}
	}
}

func (m *MainScreen) loadAllProjects() tea.Cmd {
	if m.isDemo {
		return nil
	}
	return func() tea.Msg {
		projects, err := m.client.ListProjects()
		if err != nil {
			return errMsg{err: err}
		}
		return allProjectsLoadedMsg{projects: projects}
	}
}

func (m *MainScreen) loadProjectContent() tea.Cmd {
	if m.selectedProject == nil {
		return nil
	}
	ref := m.selectedProject.DefaultBranch
	if ref == "" {
		ref = "main"
	}
	return m.loadProjectContentForBranch(ref)
}

func (m *MainScreen) loadProjectContentForBranch(branch string) tea.Cmd {
	if m.selectedProject == nil || m.isDemo {
		return nil
	}
	projectID := fmt.Sprintf("%d", m.selectedProject.ID)

	return func() tea.Msg {
		entries, err := m.client.GetTree(projectID, branch, "")
		if err != nil {
			return errMsg{err: err}
		}

		// Fetch last commit for each entry in parallel
		m.fetchLastCommits(projectID, branch, entries)

		// Try to load README
		var readme string
		for _, e := range entries {
			lower := strings.ToLower(e.Name)
			if strings.HasPrefix(lower, "readme") {
				content, err := m.client.GetFileContent(projectID, e.Path, branch)
				if err == nil {
					readme = content
				}
				break
			}
		}

		return projectContentMsg{entries: entries, readme: readme}
	}
}

func (m *MainScreen) loadDirectory(path string) tea.Cmd {
	if m.selectedProject == nil || m.isDemo {
		return nil
	}
	projectID := fmt.Sprintf("%d", m.selectedProject.ID)
	ref := m.currentBranch
	if ref == "" {
		ref = m.selectedProject.DefaultBranch
	}
	if ref == "" {
		ref = "main"
	}

	return func() tea.Msg {
		entries, err := m.client.GetTree(projectID, ref, path)
		if err != nil {
			return errMsg{err: err}
		}

		// Fetch last commit for each entry in parallel
		m.fetchLastCommits(projectID, ref, entries)

		return treeLoadedMsg{entries: entries, path: path}
	}
}

// fetchLastCommits fetches the last commit for each entry in parallel
func (m *MainScreen) fetchLastCommits(projectID, ref string, entries []gitlab.TreeEntry) {
	if m.client == nil || len(entries) == 0 {
		return
	}

	var wg sync.WaitGroup
	// Limit concurrent requests
	sem := make(chan struct{}, 10)

	for i := range entries {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			commit, err := m.client.GetLastCommitForPath(projectID, ref, entries[idx].Path)
			if err == nil && commit != nil {
				entries[idx].LastCommit = commit
			}
		}(i)
	}
	wg.Wait()
}

func (m *MainScreen) loadFile(filePath string) tea.Cmd {
	if m.selectedProject == nil || m.isDemo {
		return nil
	}
	projectID := fmt.Sprintf("%d", m.selectedProject.ID)
	ref := m.currentBranch
	if ref == "" {
		ref = m.selectedProject.DefaultBranch
	}
	if ref == "" {
		ref = "main"
	}

	return func() tea.Msg {
		content, err := m.client.GetFileContent(projectID, filePath, ref)
		if err != nil {
			return errMsg{err: err}
		}
		return fileContentMsg{content: content, path: filePath}
	}
}

func (m *MainScreen) loadMRs() tea.Cmd {
	if m.selectedProject == nil || m.isDemo {
		return nil
	}
	projectID := fmt.Sprintf("%d", m.selectedProject.ID)
	return func() tea.Msg {
		mrs, err := m.client.ListMergeRequests(projectID)
		if err != nil {
			return errMsg{err: err}
		}
		return mrsLoadedMsg{mrs: mrs}
	}
}

func (m *MainScreen) loadPipelines() tea.Cmd {
	if m.selectedProject == nil || m.isDemo {
		return nil
	}
	projectID := fmt.Sprintf("%d", m.selectedProject.ID)
	return func() tea.Msg {
		pipelines, err := m.client.ListPipelines(projectID)
		if err != nil {
			return errMsg{err: err}
		}
		return pipelinesLoadedMsg{pipelines: pipelines}
	}
}

func (m *MainScreen) loadReleases() tea.Cmd {
	if m.selectedProject == nil || m.isDemo {
		return nil
	}
	projectID := fmt.Sprintf("%d", m.selectedProject.ID)
	return func() tea.Msg {
		releases, err := m.client.ListReleases(projectID)
		if err != nil {
			return errMsg{err: err}
		}
		return releasesLoadedMsg{releases: releases}
	}
}

func (m *MainScreen) loadBranches() tea.Cmd {
	if m.selectedProject == nil || m.isDemo {
		return nil
	}
	projectID := fmt.Sprintf("%d", m.selectedProject.ID)
	return func() tea.Msg {
		branches, err := m.client.ListBranches(projectID)
		if err != nil {
			return errMsg{err: err}
		}
		return branchesLoadedMsg{branches: branches}
	}
}

func (m *MainScreen) loadPipelineJobs(pipelineID int) tea.Cmd {
	if m.selectedProject == nil || m.isDemo {
		return nil
	}
	projectID := fmt.Sprintf("%d", m.selectedProject.ID)
	return func() tea.Msg {
		jobs, err := m.client.ListPipelineJobs(projectID, pipelineID)
		if err != nil {
			return errMsg{err: err}
		}
		return jobsLoadedMsg{jobs: jobs}
	}
}

func (m *MainScreen) loadPipelineJobsForList(pipelineID int) tea.Cmd {
	if m.selectedProject == nil || m.isDemo {
		return nil
	}
	projectID := fmt.Sprintf("%d", m.selectedProject.ID)
	return func() tea.Msg {
		jobs, err := m.client.ListPipelineJobs(projectID, pipelineID)
		if err != nil {
			// Silently ignore errors for list view
			return pipelineJobsLoadedMsg{pipelineID: pipelineID, jobs: nil}
		}
		return pipelineJobsLoadedMsg{pipelineID: pipelineID, jobs: jobs}
	}
}

func (m *MainScreen) loadJobLog(jobID int) tea.Cmd {
	if m.selectedProject == nil || m.isDemo {
		return nil
	}
	projectID := fmt.Sprintf("%d", m.selectedProject.ID)
	return func() tea.Msg {
		log, err := m.client.GetJobLog(projectID, jobID)
		if err != nil {
			return errMsg{err: err}
		}
		return jobLogLoadedMsg{log: log}
	}
}

// Messages
type errMsg struct{ err error }
type groupsLoadedMsg struct{ groups []gitlab.Group }
type groupProjectsLoadedMsg struct {
	groupID  int
	projects []gitlab.Project
}
type allProjectsLoadedMsg struct{ projects []gitlab.Project }
type projectContentMsg struct {
	entries []gitlab.TreeEntry
	readme  string
}
type treeLoadedMsg struct {
	entries []gitlab.TreeEntry
	path    string
}
type fileContentMsg struct {
	content string
	path    string
}
type mrsLoadedMsg struct{ mrs []gitlab.MergeRequest }
type pipelinesLoadedMsg struct{ pipelines []gitlab.Pipeline }
type releasesLoadedMsg struct{ releases []gitlab.Release }
type downloadCompleteMsg struct {
	filename string
	bytes    int64
	err      error
}
type branchesLoadedMsg struct{ branches []gitlab.Branch }
type jobsLoadedMsg struct{ jobs []gitlab.Job }
type jobLogLoadedMsg struct{ log string }
type pipelineJobsLoadedMsg struct {
	pipelineID int
	jobs       []gitlab.Job
}

// pipelineTickMsg triggers auto-refresh of pipelines
type pipelineTickMsg time.Time

// pipelinesRefreshedMsg is like pipelinesLoadedMsg but preserves selection
type pipelinesRefreshedMsg struct{ pipelines []gitlab.Pipeline }

// pipelineTickCmd returns a command that sends a tick after the configured interval
func pipelineTickCmd() tea.Cmd {
	return tea.Tick(config.PipelineRefreshInterval, func(t time.Time) tea.Msg {
		return pipelineTickMsg(t)
	})
}

// refreshPipelines fetches pipelines without resetting selection
func (m *MainScreen) refreshPipelines() tea.Cmd {
	if m.selectedProject == nil || m.isDemo {
		return nil
	}
	projectID := fmt.Sprintf("%d", m.selectedProject.ID)
	return func() tea.Msg {
		pipelines, err := m.client.ListPipelines(projectID)
		if err != nil {
			// Silently ignore errors on auto-refresh
			return nil
		}
		return pipelinesRefreshedMsg{pipelines: pipelines}
	}
}

// jobLogTickMsg triggers auto-refresh of job log
type jobLogTickMsg time.Time

// jobLogRefreshedMsg carries refreshed log content
type jobLogRefreshedMsg struct{ log string }

// jobsRefreshedMsg carries refreshed job statuses
type jobsRefreshedMsg struct{ jobs []gitlab.Job }

// runnersLoadedMsg carries all running and pending jobs
type runnersLoadedMsg struct {
	running []gitlab.Job
	pending []gitlab.Job
}

// runnersTickMsg triggers auto-refresh of runners popup
type runnersTickMsg time.Time

// runnersTickCmd returns a command that sends a tick for runners refresh
func runnersTickCmd() tea.Cmd {
	return tea.Tick(config.PipelineRefreshInterval, func(t time.Time) tea.Msg {
		return runnersTickMsg(t)
	})
}

// loadAllJobs fetches all running and pending jobs across projects
func (m *MainScreen) loadAllJobs() tea.Cmd {
	if m.isDemo {
		return nil
	}
	return func() tea.Msg {
		running, _ := m.client.ListRunningJobs()
		pending, _ := m.client.ListPendingJobs()
		return runnersLoadedMsg{running: running, pending: pending}
	}
}

// jobLogTickCmd returns a command that sends a tick after the configured interval
func jobLogTickCmd() tea.Cmd {
	return tea.Tick(config.JobLogRefreshInterval, func(t time.Time) tea.Msg {
		return jobLogTickMsg(t)
	})
}

// refreshJobLog fetches job log without resetting viewport position
func (m *MainScreen) refreshJobLog() tea.Cmd {
	if m.selectedProject == nil || m.isDemo || m.selectedJobIdx < 0 || m.selectedJobIdx >= len(m.jobs) {
		return nil
	}
	job := m.jobs[m.selectedJobIdx]
	projectID := fmt.Sprintf("%d", m.selectedProject.ID)
	return func() tea.Msg {
		log, err := m.client.GetJobLog(projectID, job.ID)
		if err != nil {
			// Silently ignore errors on auto-refresh
			return nil
		}
		return jobLogRefreshedMsg{log: log}
	}
}

// refreshJobs fetches updated job statuses for the current pipeline
func (m *MainScreen) refreshJobs() tea.Cmd {
	if m.selectedProject == nil || m.isDemo || m.currentPipelineID == 0 {
		return nil
	}
	projectID := fmt.Sprintf("%d", m.selectedProject.ID)
	pipelineID := m.currentPipelineID
	return func() tea.Msg {
		jobs, err := m.client.ListPipelineJobs(projectID, pipelineID)
		if err != nil {
			return nil
		}
		return jobsRefreshedMsg{jobs: jobs}
	}
}

// Update handles messages
func (m *MainScreen) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case errMsg:
		m.loading = false
		m.lastError = msg.err.Error()
		// Don't set m.errMsg - that would crash the UI
		// Instead show error in status bar and allow retry
		return m, nil

	case groupsLoadedMsg:
		m.groups = msg.groups
		m.loading = false
		m.lastError = ""
		m.rebuildNavTree()
		// If no groups, load all projects directly
		if len(m.groups) == 0 {
			m.loading = true
			m.loadingMsg = "Loading projects..."
			cmd := m.loadAllProjects()
			m.retryCmd = cmd
			return m, cmd
		}
		return m, nil

	case groupProjectsLoadedMsg:
		m.groupProjects[msg.groupID] = msg.projects
		m.loading = false
		m.lastError = ""
		m.rebuildNavTree()
		return m, nil

	case allProjectsLoadedMsg:
		// When no groups exist, show all projects as root nodes
		for _, p := range msg.projects {
			projectNode := TreeNode{
				Type:     "project",
				Name:     p.Name,
				FullPath: p.PathWithNamespace,
				ID:       p.ID,
				Depth:    0,
				Project:  &p,
			}
			m.treeNodes = append(m.treeNodes, projectNode)
		}
		m.loading = false
		m.lastError = ""
		return m, nil

	case projectContentMsg:
		m.files = msg.entries
		m.readmeContent = msg.readme
		// Calculate content width for markdown rendering
		contentWidth := int(float64(m.width) * (1 - config.NavigatorWidthRatio)) - 4
		if contentWidth < 40 {
			contentWidth = 80
		}
		m.readmeRendered = renderMarkdown(msg.readme, contentWidth)
		m.fileContent = ""
		m.selectedContent = 0
		m.fileScrollOffset = 0
		m.readmeReady = false // Reset to reinitialize viewport with new content
		m.loading = false
		m.lastError = ""
		// Set current branch if not set
		if m.currentBranch == "" && m.selectedProject != nil {
			m.currentBranch = m.selectedProject.DefaultBranch
			if m.currentBranch == "" {
				m.currentBranch = "main"
			}
		}
		return m, nil

	case treeLoadedMsg:
		m.files = msg.entries
		m.selectedContent = 0
		m.fileScrollOffset = 0
		m.fileContent = ""
		m.loading = false
		m.lastError = ""
		return m, nil

	case fileContentMsg:
		// Check for binary content
		if isBinaryExtension(msg.path) || isBinaryContent(msg.content) {
			m.fileContent = "[Binary file - cannot display]"
		} else {
			m.fileContent = msg.content
		}
		m.viewingFile = true
		m.viewingFilePath = msg.path
		m.fileViewReady = false // Reset to reinitialize viewport with new content
		m.loading = false
		m.lastError = ""
		return m, nil

	case mrsLoadedMsg:
		m.mergeRequests = msg.mrs
		m.selectedContent = 0
		m.fileScrollOffset = 0
		m.loading = false
		m.lastError = ""
		return m, nil

	case pipelinesLoadedMsg:
		m.pipelines = msg.pipelines
		m.selectedContent = 0
		m.fileScrollOffset = 0
		m.pipelineJobs = make(map[int][]gitlab.Job)
		m.loading = false
		m.lastError = ""
		// Load jobs for each pipeline to show stages
		var cmds []tea.Cmd
		for _, p := range m.pipelines {
			cmds = append(cmds, m.loadPipelineJobsForList(p.ID))
		}
		// Start auto-refresh ticker
		cmds = append(cmds, pipelineTickCmd())
		return m, tea.Batch(cmds...)

	case releasesLoadedMsg:
		m.releases = msg.releases
		m.selectedContent = 0
		m.fileScrollOffset = 0
		m.loading = false
		m.lastError = ""
		return m, nil

	case downloadCompleteMsg:
		m.loading = false
		if msg.err != nil {
			m.statusMsg = "Download failed: " + msg.err.Error()
		} else {
			m.statusMsg = fmt.Sprintf("Downloaded %s (%d bytes)", msg.filename, msg.bytes)
		}
		return m, nil

	case pipelinesRefreshedMsg:
		// Preserve selection when auto-refreshing
		selectedPipelineID := 0
		if m.selectedContent < len(m.pipelines) {
			selectedPipelineID = m.pipelines[m.selectedContent].ID
		}
		m.pipelines = msg.pipelines
		// Restore selection by finding the same pipeline ID
		if selectedPipelineID != 0 {
			for i, p := range m.pipelines {
				if p.ID == selectedPipelineID {
					m.selectedContent = i
					break
				}
			}
		}
		// Clamp selection to valid range
		if m.selectedContent >= len(m.pipelines) && len(m.pipelines) > 0 {
			m.selectedContent = len(m.pipelines) - 1
		}
		// Refresh jobs for pipelines
		var cmds []tea.Cmd
		for _, p := range m.pipelines {
			cmds = append(cmds, m.loadPipelineJobsForList(p.ID))
		}
		// Continue ticker
		cmds = append(cmds, pipelineTickCmd())
		return m, tea.Batch(cmds...)

	case pipelineTickMsg:
		// Only refresh if we're viewing pipelines tab and have a project
		if m.contentTab == TabPipelines && m.selectedProject != nil && !m.loading {
			return m, m.refreshPipelines()
		}
		// Keep ticker running even if we're not on pipelines tab
		if m.selectedProject != nil {
			return m, pipelineTickCmd()
		}
		return m, nil

	case pipelineJobsLoadedMsg:
		if m.pipelineJobs == nil {
			m.pipelineJobs = make(map[int][]gitlab.Job)
		}
		m.pipelineJobs[msg.pipelineID] = msg.jobs
		return m, nil

	case branchesLoadedMsg:
		m.branches = msg.branches
		m.selectedContent = 0
		m.fileScrollOffset = 0
		m.loading = false
		m.lastError = ""
		// If branch popup is open, keep it open
		if m.showBranchPopup {
			// Find current branch in list
			for i, br := range m.branches {
				if br.Name == m.currentBranch {
					m.selectedBranchIdx = i
					break
				}
			}
		}
		return m, nil

	case jobsLoadedMsg:
		m.jobs = msg.jobs
		m.selectedJobIdx = 0
		m.jobLog = ""
		m.jobLogReady = false
		m.loading = false
		m.lastError = ""
		// Auto-load first job's log if available
		if len(m.jobs) > 0 {
			m.loading = true
			m.loadingMsg = "Loading job log..."
			cmd := m.loadJobLog(m.jobs[0].ID)
			m.retryCmd = cmd
			return m, cmd
		}
		return m, nil

	case jobLogLoadedMsg:
		m.jobLog = msg.log
		m.jobLogReady = false
		m.loading = false
		m.lastError = ""
		// Start at bottom where errors usually are
		m.jobLogCursor = strings.Count(msg.log, "\n")
		// Start auto-refresh for live log viewing
		return m, jobLogTickCmd()

	case jobLogTickMsg:
		// Only refresh if job popup is still open
		if m.showJobLogPopup {
			// Refresh both jobs (for status updates) and log, then schedule next tick
			return m, tea.Batch(m.refreshJobs(), m.refreshJobLog(), jobLogTickCmd())
		}
		return m, nil

	case jobsRefreshedMsg:
		if m.showJobLogPopup && msg.jobs != nil {
			// Preserve selection by job ID
			selectedJobID := 0
			if m.selectedJobIdx >= 0 && m.selectedJobIdx < len(m.jobs) {
				selectedJobID = m.jobs[m.selectedJobIdx].ID
			}
			m.jobs = msg.jobs
			// Restore selection
			if selectedJobID != 0 {
				for i, job := range m.jobs {
					if job.ID == selectedJobID {
						m.selectedJobIdx = i
						break
					}
				}
			}
		}
		return m, nil

	case jobLogRefreshedMsg:
		if msg.log != "" && m.showJobLogPopup {
			// Save current scroll position
			currentLine := m.jobLogViewport.YOffset
			wasAtBottom := m.jobLogViewport.ScrollPercent() >= 0.99

			// Update log content
			m.jobLog = msg.log

			// Update viewport content directly without recreating it
			cleanLog := msg.log
			cleanLog = strings.ReplaceAll(cleanLog, "\t", "    ")
			cleanLog = strings.ReplaceAll(cleanLog, "\r", "")
			// Don't wrap - preserve line numbers for visual selection
			m.jobLogViewport.SetContent(cleanLog)

			// Auto-scroll to bottom when not focused on log panel, or was already at bottom
			if !m.jobLogFocused || wasAtBottom {
				m.jobLogViewport.GotoBottom()
			} else {
				m.jobLogViewport.SetYOffset(currentLine)
			}
		}
		return m, nil

	case runnersLoadedMsg:
		m.runningJobs = msg.running
		m.pendingJobs = msg.pending
		m.runnersLoading = false
		if m.showRunnersPopup {
			return m, runnersTickCmd()
		}
		return m, nil

	case runnersTickMsg:
		if m.showRunnersPopup {
			return m, m.loadAllJobs()
		}
		return m, nil

	case tea.KeyMsg:
		return m.handleKey(msg)
	}

	return m, nil
}

func (m *MainScreen) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Clear status message on any keypress
	m.statusMsg = ""

	// Handle popups first
	if m.showJobLogPopup {
		return m.handleJobLogPopup(msg)
	}
	if m.showBranchPopup {
		return m.handleBranchPopup(msg)
	}
	if m.showRunnersPopup {
		return m.handleRunnersPopup(msg)
	}
	if m.showReleasePopup {
		return m.handleReleasePopup(msg)
	}
	if m.showFolderBrowser {
		return m.handleFolderBrowser(msg)
	}

	if key.Matches(msg, m.keymap.Quit) {
		return m, tea.Quit
	}

	// Clear error on Escape
	if msg.String() == "esc" || msg.String() == "escape" {
		if m.lastError != "" {
			m.lastError = ""
			return m, nil
		}
	}

	// Retry on 'r' key if there's an error
	if msg.String() == "r" && m.lastError != "" && m.retryCmd != nil {
		m.lastError = ""
		m.loading = true
		m.loadingMsg = "Retrying..."
		cmd := m.retryCmd
		return m, cmd
	}

	// Yank clone URLs when project is selected
	if m.selectedProject != nil {
		switch msg.String() {
		case "S":
			// Yank SSH URL
			if m.selectedProject.SSHURLToRepo != "" {
				if err := copyToClipboard(m.selectedProject.SSHURLToRepo); err != nil {
					m.statusMsg = "Copy failed: " + err.Error()
				} else {
					m.statusMsg = "SSH: " + m.selectedProject.SSHURLToRepo
				}
				return m, nil
			}
		case "U":
			// Yank HTTPS URL
			if m.selectedProject.HTTPURLToRepo != "" {
				if err := copyToClipboard(m.selectedProject.HTTPURLToRepo); err != nil {
					m.statusMsg = "Copy failed: " + err.Error()
				} else {
					m.statusMsg = "HTTPS: " + m.selectedProject.HTTPURLToRepo
				}
				return m, nil
			}
		}
	}

	// 'b' to open branch selector (when viewing files)
	if msg.String() == "b" && m.selectedProject != nil && m.contentTab == TabFiles {
		m.showBranchPopup = true
		m.selectedBranchIdx = 0
		// Find current branch in list
		for i, br := range m.branches {
			if br.Name == m.currentBranch {
				m.selectedBranchIdx = i
				break
			}
		}
		if len(m.branches) == 0 && !m.isDemo {
			m.loading = true
			m.loadingMsg = "Loading branches..."
			cmd := m.loadBranches()
			m.retryCmd = cmd
			return m, cmd
		}
		return m, nil
	}

	// 'R' to open runners/jobs popup (shows all running/pending jobs)
	if msg.String() == "R" {
		m.showRunnersPopup = true
		m.runnersCursor = 0
		m.runnersTab = 0
		m.runnersLoading = true
		return m, m.loadAllJobs()
	}

	// Panel navigation with Shift+HJKL
	// Layout:
	// [1 Navigator] [2 Content ]
	//               [3 README  ]
	switch msg.String() {
	case "H", "shift+left":
		switch m.focusedPanel {
		case PanelContent, PanelReadme:
			m.focusedPanel = PanelNavigator
		}
		return m, nil
	case "L", "shift+right":
		switch m.focusedPanel {
		case PanelNavigator:
			m.focusedPanel = PanelContent
		}
		return m, nil
	case "K", "shift+up":
		switch m.focusedPanel {
		case PanelReadme:
			m.focusedPanel = PanelContent
		}
		return m, nil
	case "J", "shift+down":
		switch m.focusedPanel {
		case PanelContent:
			m.focusedPanel = PanelReadme
		}
		return m, nil
	case "1":
		m.focusedPanel = PanelNavigator
		return m, nil
	case "2":
		m.focusedPanel = PanelContent
		return m, nil
	case "3":
		m.focusedPanel = PanelReadme
		return m, nil
	}

	switch m.focusedPanel {
	case PanelNavigator:
		return m.handleNavigatorNav(msg)
	case PanelContent:
		return m.handleContentNav(msg)
	case PanelReadme:
		return m.handleReadmeNav(msg)
	}

	return m, nil
}

func (m *MainScreen) handleNavigatorNav(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if len(m.treeNodes) == 0 {
		return m, nil
	}

	switch {
	case key.Matches(msg, m.keymap.Down):
		if m.selectedNodeIdx < len(m.treeNodes)-1 {
			m.selectedNodeIdx++
		}
	case key.Matches(msg, m.keymap.Up):
		if m.selectedNodeIdx > 0 {
			m.selectedNodeIdx--
		}
	case key.Matches(msg, m.keymap.Right), key.Matches(msg, m.keymap.Select):
		if m.selectedNodeIdx >= len(m.treeNodes) {
			return m, nil
		}

		node := m.treeNodes[m.selectedNodeIdx]

		if node.Type == "group" {
			// Toggle group expansion
			if m.expandedGroups[node.ID] {
				// Collapse
				m.expandedGroups[node.ID] = false
				m.rebuildNavTree()
			} else {
				// Expand - check if we have projects cached
				m.expandedGroups[node.ID] = true
				if _, ok := m.groupProjects[node.ID]; !ok {
					// Need to load projects
					m.loading = true
					m.loadingMsg = "Loading projects..."
					cmd := m.loadGroupProjects(node.ID, node.FullPath)
					m.retryCmd = cmd
					return m, cmd
				}
				m.rebuildNavTree()
			}
		} else if node.Type == "project" && node.Project != nil {
			// Select project and load its content
			m.selectedProject = node.Project
			m.currentPath = nil
			m.currentBranch = ""
			m.contentTab = TabFiles
			m.focusedPanel = PanelContent

			// In demo mode, data is pre-populated - don't clear or reload
			if m.isDemo {
				return m, nil
			}

			m.files = nil
			m.mergeRequests = nil
			m.pipelines = nil
			m.releases = nil
			m.branches = nil
			m.fileContent = ""
			m.readmeContent = ""
			m.loading = true
			m.loadingMsg = "Loading repository..."
			cmd := m.loadProjectContent()
			m.retryCmd = cmd
			return m, cmd
		}
	case key.Matches(msg, m.keymap.Left):
		if m.selectedNodeIdx >= len(m.treeNodes) {
			return m, nil
		}

		node := m.treeNodes[m.selectedNodeIdx]
		if node.Type == "group" && m.expandedGroups[node.ID] {
			// Collapse the group
			m.expandedGroups[node.ID] = false
			m.rebuildNavTree()
		}
	}
	return m, nil
}

func (m *MainScreen) handleContentNav(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Handle escape for going back
	if msg.String() == "esc" || msg.String() == "escape" {
		// If viewing a file, go back to file list
		if m.viewingFile {
			m.viewingFile = false
			m.fileContent = ""
			m.viewingFilePath = ""
			return m, nil
		}
		// If in a directory, go up
		if m.contentTab == TabFiles && len(m.currentPath) > 0 {
			m.currentPath = m.currentPath[:len(m.currentPath)-1]
			// Demo mode doesn't support directory navigation
			if m.isDemo {
				return m, nil
			}
			m.loading = true
			m.loadingMsg = "Loading..."
			path := strings.Join(m.currentPath, "/")
			cmd := m.loadDirectory(path)
			m.retryCmd = cmd
			return m, cmd
		}
		// If at root, go back to navigator
		m.focusedPanel = PanelNavigator
		return m, nil
	}

	switch {
	case key.Matches(msg, m.keymap.Left):
		// h - switch to previous tab
		if m.contentTab > TabFiles {
			return m, m.switchTab(m.contentTab - 1)
		}
		// At first tab, go to navigator panel
		m.focusedPanel = PanelNavigator

	case key.Matches(msg, m.keymap.Right):
		// l - switch to next tab
		if m.contentTab < TabReleases {
			return m, m.switchTab(m.contentTab + 1)
		}

	case key.Matches(msg, m.keymap.Select):
		// Enter - drill into directory or view file
		if m.contentTab == TabFiles && m.selectedContent < len(m.files) {
			entry := m.files[m.selectedContent]
			if entry.Type == "tree" {
				// Demo mode doesn't support directory navigation
				if m.isDemo {
					return m, nil
				}
				m.currentPath = append(m.currentPath, entry.Name)
				m.loading = true
				m.loadingMsg = "Loading..."
				cmd := m.loadDirectory(entry.Path)
				m.retryCmd = cmd
				return m, cmd
			} else {
				// Demo mode uses mock file content
				if m.isDemo {
					if content, ok := MockFileContent[entry.Name]; ok {
						m.fileContent = content
						m.viewingFile = true
						m.viewingFilePath = entry.Path
					}
					return m, nil
				}
				m.loading = true
				m.loadingMsg = "Loading file..."
				cmd := m.loadFile(entry.Path)
				m.retryCmd = cmd
				return m, cmd
			}
		}
		// Load jobs for selected pipeline and show popup
		if m.contentTab == TabPipelines && m.selectedContent < len(m.pipelines) {
			// Demo mode doesn't support job log viewing
			if m.isDemo {
				return m, nil
			}
			pipeline := m.pipelines[m.selectedContent]
			m.jobs = nil
			m.jobLog = ""
			m.showJobLogPopup = true
			m.jobLogFocused = false // Start focused on job list
			m.jobLogCursor = 0
			m.jobLogHScroll = 0
			m.currentPipelineID = pipeline.ID
			m.loading = true
			m.loadingMsg = "Loading jobs..."
			cmd := m.loadPipelineJobs(pipeline.ID)
			m.retryCmd = cmd
			return m, cmd
		}
		// Show release assets popup
		if m.contentTab == TabReleases && m.selectedContent < len(m.releases) {
			m.selectedReleaseIdx = m.selectedContent
			m.releaseAssetCursor = 0
			m.releaseScrollOffset = 0
			m.showReleasePopup = true
			return m, nil
		}

	case key.Matches(msg, m.keymap.Down):
		// If viewing file, scroll down
		if m.viewingFile {
			m.fileViewport.ScrollDown(1)
			return m, nil
		}
		maxItems := m.getContentCount()
		if m.selectedContent < maxItems-1 {
			m.selectedContent++
			if m.contentTab == TabFiles {
				m.fileContent = ""
				m.viewingFile = false
			}
			m.adjustScrollOffset()
		}
	case key.Matches(msg, m.keymap.Up):
		// If viewing file, scroll up
		if m.viewingFile {
			m.fileViewport.ScrollUp(1)
			return m, nil
		}
		if m.selectedContent > 0 {
			m.selectedContent--
			if m.contentTab == TabFiles {
				m.fileContent = ""
				m.viewingFile = false
			}
			m.adjustScrollOffset()
		}
	}

	// Additional scroll keys when viewing file
	if m.viewingFile {
		switch msg.String() {
		case "ctrl+d":
			m.fileViewport.HalfPageDown()
		case "ctrl+u":
			m.fileViewport.HalfPageUp()
		case "g":
			m.fileViewport.GotoTop()
		case "G":
			m.fileViewport.GotoBottom()
		}
	}

	return m, nil
}

func (m *MainScreen) adjustScrollOffset() {
	// Calculate visible area matching renderContentPanel calculation
	// contentHeight = m.height - StatusBarHeight (1)
	// visibleLines = height - 6 (in renderContentPanel)
	contentHeight := m.height - config.StatusBarHeight
	visibleLines := contentHeight - 6
	if visibleLines < 1 {
		visibleLines = 1
	}

	// Adjust offset to keep selected item visible
	if m.selectedContent < m.fileScrollOffset {
		m.fileScrollOffset = m.selectedContent
	} else if m.selectedContent >= m.fileScrollOffset+visibleLines {
		m.fileScrollOffset = m.selectedContent - visibleLines + 1
	}
}

func (m *MainScreen) handleReadmeNav(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	// Clear key sequence unless it's a sequence key (g, y)
	if key != "g" && key != "y" {
		m.readmeLastKey = ""
	}

	// Get line count from raw content
	maxLine := strings.Count(m.readmeContent, "\n")

	switch key {
	case "H", "shift+left":
		m.focusedPanel = PanelNavigator
		return m, nil
	case "L", "shift+right":
		m.focusedPanel = PanelContent
		return m, nil
	case "h", "left":
		m.focusedPanel = PanelNavigator
		return m, nil
	case "l", "right":
		m.focusedPanel = PanelContent
		return m, nil
	case "j", "down":
		if m.readmeCursor < maxLine {
			m.readmeCursor++
			if m.readmeVisualMode {
				m.readmeVisualEnd = m.readmeCursor
			}
		}
		// Keep cursor in view
		viewportBottom := m.readmeViewport.YOffset + m.readmeViewport.Height - 1
		if m.readmeCursor > viewportBottom {
			m.readmeViewport.ScrollDown(1)
		}
	case "k", "up":
		if m.readmeCursor > 0 {
			m.readmeCursor--
			if m.readmeVisualMode {
				m.readmeVisualEnd = m.readmeCursor
			}
		}
		// Keep cursor in view
		if m.readmeCursor < m.readmeViewport.YOffset {
			m.readmeViewport.ScrollUp(1)
		}
	case "ctrl+d":
		m.readmeViewport.HalfPageDown()
		m.readmeCursor += m.readmeViewport.Height / 2
		if m.readmeCursor > maxLine {
			m.readmeCursor = maxLine
		}
		if m.readmeVisualMode {
			m.readmeVisualEnd = m.readmeCursor
		}
	case "ctrl+u":
		m.readmeViewport.HalfPageUp()
		m.readmeCursor -= m.readmeViewport.Height / 2
		if m.readmeCursor < 0 {
			m.readmeCursor = 0
		}
		if m.readmeVisualMode {
			m.readmeVisualEnd = m.readmeCursor
		}
	case "g":
		if m.readmeLastKey == "g" {
			// gg - go to top
			m.readmeViewport.GotoTop()
			m.readmeCursor = 0
			if m.readmeVisualMode {
				m.readmeVisualEnd = m.readmeCursor
			}
			m.readmeLastKey = "gg"
			return m, nil
		}
		m.readmeLastKey = "g"
		return m, nil
	case "G":
		m.readmeViewport.GotoBottom()
		m.readmeCursor = maxLine
		if m.readmeVisualMode {
			m.readmeVisualEnd = m.readmeCursor
		}
	case "V":
		// Toggle visual line mode
		if m.readmeVisualMode {
			m.readmeVisualMode = false
		} else {
			m.readmeVisualMode = true
			m.readmeVisualStart = m.readmeCursor
			m.readmeVisualEnd = m.readmeCursor
		}
	case "y":
		if m.readmeContent == "" {
			m.readmeLastKey = ""
			return m, nil
		}
		lines := strings.Split(m.readmeContent, "\n")
		if m.readmeVisualMode {
			// Copy selected lines
			startLine := m.readmeVisualStart
			endLine := m.readmeVisualEnd
			if startLine > endLine {
				startLine, endLine = endLine, startLine
			}
			if startLine < 0 {
				startLine = 0
			}
			if endLine >= len(lines) {
				endLine = len(lines) - 1
			}
			selected := strings.Join(lines[startLine:endLine+1], "\n")
			if err := copyToClipboard(selected); err != nil {
				m.statusMsg = "Copy failed: " + err.Error()
			} else {
				m.statusMsg = fmt.Sprintf("Copied %d lines!", endLine-startLine+1)
			}
			m.readmeVisualMode = false
		} else if m.readmeLastKey == "gg" {
			// ggy - yank entire readme
			if err := copyToClipboard(m.readmeContent); err != nil {
				m.statusMsg = "Copy failed: " + err.Error()
			} else {
				m.statusMsg = fmt.Sprintf("Yanked all %d lines!", len(lines))
			}
		} else if m.readmeLastKey == "y" {
			// yy - yank current line
			if m.readmeCursor >= 0 && m.readmeCursor < len(lines) {
				if err := copyToClipboard(lines[m.readmeCursor]); err != nil {
					m.statusMsg = "Copy failed: " + err.Error()
				} else {
					m.statusMsg = "Yanked line!"
				}
			}
		} else {
			m.readmeLastKey = "y"
			return m, nil
		}
		m.readmeLastKey = ""
	case "esc", "escape":
		if m.readmeVisualMode {
			m.readmeVisualMode = false
			return m, nil
		}
	}
	return m, nil
}

func (m *MainScreen) handleBranchPopup(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "esc", "escape":
		m.showBranchPopup = false
		return m, nil
	case "j", "down":
		if m.selectedBranchIdx < len(m.branches)-1 {
			m.selectedBranchIdx++
		}
	case "k", "up":
		if m.selectedBranchIdx > 0 {
			m.selectedBranchIdx--
		}
	case "enter":
		if m.selectedBranchIdx < len(m.branches) {
			m.currentBranch = m.branches[m.selectedBranchIdx].Name
			m.showBranchPopup = false
			// Demo mode doesn't support branch switching
			if m.isDemo {
				return m, nil
			}
			// Reload files for new branch
			m.files = nil
			m.currentPath = nil
			m.fileContent = ""
			m.viewingFile = false
			m.readmeContent = ""
			m.loading = true
			m.loadingMsg = "Loading files..."
			cmd := m.loadProjectContentForBranch(m.currentBranch)
			m.retryCmd = cmd
			return m, cmd
		}
	}
	return m, nil
}

func (m *MainScreen) handleRunnersPopup(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Get current job list based on tab
	jobs := m.runningJobs
	if m.runnersTab == 1 {
		jobs = m.pendingJobs
	}

	switch msg.String() {
	case "q", "esc", "escape":
		m.showRunnersPopup = false
		return m, nil
	case "j", "down":
		if m.runnersCursor < len(jobs)-1 {
			m.runnersCursor++
		}
	case "k", "up":
		if m.runnersCursor > 0 {
			m.runnersCursor--
		}
	case "tab", "l", "right":
		// Switch between running/pending tabs
		m.runnersTab = (m.runnersTab + 1) % 2
		m.runnersCursor = 0
	case "shift+tab", "h", "left":
		m.runnersTab = (m.runnersTab + 1) % 2
		m.runnersCursor = 0
	case "r":
		// Manual refresh
		m.runnersLoading = true
		return m, m.loadAllJobs()
	case "g":
		if m.runnersLastKey == "g" {
			m.runnersCursor = 0
			m.runnersLastKey = ""
			return m, nil
		}
		m.runnersLastKey = "g"
		return m, nil
	case "G":
		m.runnersCursor = len(jobs) - 1
		if m.runnersCursor < 0 {
			m.runnersCursor = 0
		}
	}
	// Clear key sequence for non-sequence keys
	if msg.String() != "g" {
		m.runnersLastKey = ""
	}
	return m, nil
}

func (m *MainScreen) handleJobLogPopup(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	// Clear key sequence unless it's a sequence key (g, y)
	if key != "g" && key != "y" {
		m.jobLogLastKey = ""
	}

	switch key {
	case "q":
		m.showJobLogPopup = false
		m.jobs = nil
		m.jobLog = ""
		m.statusMsg = ""
		m.lastError = ""
		m.jobLogFocused = false
		return m, nil
	case "esc", "escape":
		// Cancel visual mode first
		if m.visualLineMode {
			m.visualLineMode = false
			return m, nil
		}
		// Switch to job list first, then close
		if m.jobLogFocused {
			m.jobLogFocused = false
			return m, nil
		}
		m.showJobLogPopup = false
		m.jobs = nil
		m.jobLog = ""
		m.statusMsg = ""
		m.lastError = ""
		m.visualLineMode = false
		return m, nil
	case "H", "shift+left":
		// Switch to job list panel
		if m.jobLogFocused {
			m.jobLogFocused = false
		}
		return m, nil
	case "L", "shift+right", "enter":
		// Switch to log panel
		if !m.jobLogFocused {
			m.jobLogFocused = true
		}
		return m, nil
	case "h", "left":
		// Scroll left
		if m.jobLogFocused && m.jobLogHScroll > 0 {
			m.jobLogHScroll -= 20
			if m.jobLogHScroll < 0 {
				m.jobLogHScroll = 0
			}
		}
		return m, nil
	case "l", "right":
		// Scroll right
		if m.jobLogFocused {
			m.jobLogHScroll += 20
		}
		return m, nil
	case "j", "down":
		if m.jobLogFocused {
			maxLine := strings.Count(m.jobLog, "\n")
			if m.jobLogCursor < maxLine {
				m.jobLogCursor++
				if m.visualLineMode {
					m.visualEndLine = m.jobLogCursor
				}
			}
			// Keep cursor in view
			viewportBottom := m.jobLogViewport.YOffset + m.jobLogViewport.Height - 1
			if m.jobLogCursor > viewportBottom {
				m.jobLogViewport.ScrollDown(1)
			}
		} else {
			// Next job in list
			if m.selectedJobIdx < len(m.jobs)-1 {
				m.selectedJobIdx++
				if !m.isDemo {
					m.jobLog = ""
					m.jobLogReady = false
					m.jobLogHScroll = 0
					m.visualLineMode = false
					m.loading = true
					m.loadingMsg = "Loading job log..."
					m.statusMsg = ""
					cmd := m.loadJobLog(m.jobs[m.selectedJobIdx].ID)
					m.retryCmd = cmd
					return m, cmd
				}
			}
		}
	case "k", "up":
		if m.jobLogFocused {
			if m.jobLogCursor > 0 {
				m.jobLogCursor--
				if m.visualLineMode {
					m.visualEndLine = m.jobLogCursor
				}
			}
			// Keep cursor in view
			if m.jobLogCursor < m.jobLogViewport.YOffset {
				m.jobLogViewport.ScrollUp(1)
			}
		} else {
			// Previous job in list
			if m.selectedJobIdx > 0 {
				m.selectedJobIdx--
				if !m.isDemo {
					m.jobLog = ""
					m.jobLogReady = false
					m.jobLogHScroll = 0
					m.visualLineMode = false
					m.loading = true
					m.loadingMsg = "Loading job log..."
					m.statusMsg = ""
					cmd := m.loadJobLog(m.jobs[m.selectedJobIdx].ID)
					m.retryCmd = cmd
					return m, cmd
				}
			}
		}
	case "ctrl+d":
		if m.jobLogFocused {
			m.jobLogViewport.HalfPageDown()
			maxLine := strings.Count(m.jobLog, "\n")
			m.jobLogCursor += m.jobLogViewport.Height / 2
			if m.jobLogCursor > maxLine {
				m.jobLogCursor = maxLine
			}
			if m.visualLineMode {
				m.visualEndLine = m.jobLogCursor
			}
		}
	case "ctrl+u":
		if m.jobLogFocused {
			m.jobLogViewport.HalfPageUp()
			m.jobLogCursor -= m.jobLogViewport.Height / 2
			if m.jobLogCursor < 0 {
				m.jobLogCursor = 0
			}
			if m.visualLineMode {
				m.visualEndLine = m.jobLogCursor
			}
		}
	case "g":
		if m.jobLogFocused {
			if m.jobLogLastKey == "g" {
				// gg - go to top
				m.jobLogViewport.GotoTop()
				m.jobLogCursor = 0
				if m.visualLineMode {
					m.visualEndLine = m.jobLogCursor
				}
				m.jobLogLastKey = "gg" // Mark that we did gg
				return m, nil
			}
			m.jobLogLastKey = "g"
			return m, nil
		}
	case "G":
		if m.jobLogFocused {
			m.jobLogViewport.GotoBottom()
			m.jobLogCursor = strings.Count(m.jobLog, "\n")
			if m.visualLineMode {
				m.visualEndLine = m.jobLogCursor
			}
		}
	case "V":
		// Toggle visual line mode
		if m.jobLogFocused {
			if m.visualLineMode {
				m.visualLineMode = false
			} else {
				m.visualLineMode = true
				m.visualStartLine = m.jobLogCursor
				m.visualEndLine = m.jobLogCursor
			}
		}
	case "y":
		if m.jobLog == "" {
			m.jobLogLastKey = ""
			return m, nil
		}
		lines := strings.Split(m.jobLog, "\n")
		if m.visualLineMode {
			// Copy selected lines
			startLine := m.visualStartLine
			endLine := m.visualEndLine
			if startLine > endLine {
				startLine, endLine = endLine, startLine
			}
			// Clamp to valid range
			if startLine < 0 {
				startLine = 0
			}
			if endLine >= len(lines) {
				endLine = len(lines) - 1
			}
			selected := strings.Join(lines[startLine:endLine+1], "\n")
			cleanLog := stripANSI(selected)
			if err := copyToClipboard(cleanLog); err != nil {
				m.statusMsg = "Copy failed: " + err.Error()
			} else {
				m.statusMsg = fmt.Sprintf("Copied %d lines!", endLine-startLine+1)
			}
			m.visualLineMode = false
		} else if m.jobLogLastKey == "gg" {
			// ggy - yank entire log
			cleanLog := stripANSI(m.jobLog)
			if err := copyToClipboard(cleanLog); err != nil {
				m.statusMsg = "Copy failed: " + err.Error()
			} else {
				m.statusMsg = fmt.Sprintf("Yanked all %d lines!", len(lines))
			}
		} else if m.jobLogLastKey == "y" {
			// yy - yank current line
			if m.jobLogCursor >= 0 && m.jobLogCursor < len(lines) {
				cleanLine := stripANSI(lines[m.jobLogCursor])
				if err := copyToClipboard(cleanLine); err != nil {
					m.statusMsg = "Copy failed: " + err.Error()
				} else {
					m.statusMsg = "Yanked line!"
				}
			}
		} else {
			// First y - wait for second key
			m.jobLogLastKey = "y"
			return m, nil
		}
		m.jobLogLastKey = ""
	case "0":
		// Go to start of line
		if m.jobLogFocused {
			m.jobLogHScroll = 0
		}
	case "$":
		// Go to end of line (find max line width)
		if m.jobLogFocused && m.jobLog != "" {
			lines := strings.Split(m.jobLog, "\n")
			maxWidth := 0
			for _, line := range lines {
				w := lipgloss.Width(stripANSI(line))
				if w > maxWidth {
					maxWidth = w
				}
			}
			// Scroll to show end of longest line
			if maxWidth > 80 {
				m.jobLogHScroll = maxWidth - 80
			}
		}
	}
	return m, nil
}

func (m *MainScreen) switchTab(tab ContentTab) tea.Cmd {
	m.contentTab = tab
	m.selectedContent = 0
	m.fileContent = ""

	if m.selectedProject == nil || m.isDemo {
		return nil
	}

	switch tab {
	case TabFiles:
		if len(m.files) == 0 {
			m.loading = true
			m.loadingMsg = "Loading files..."
			m.currentPath = nil
			cmd := m.loadProjectContent()
			m.retryCmd = cmd
			return cmd
		}
	case TabMRs:
		if len(m.mergeRequests) == 0 {
			m.loading = true
			m.loadingMsg = "Loading merge requests..."
			cmd := m.loadMRs()
			m.retryCmd = cmd
			return cmd
		}
	case TabPipelines:
		if len(m.pipelines) == 0 {
			m.loading = true
			m.loadingMsg = "Loading pipelines..."
			cmd := m.loadPipelines()
			m.retryCmd = cmd
			return cmd
		}
	case TabReleases:
		if len(m.releases) == 0 {
			m.loading = true
			m.loadingMsg = "Loading releases..."
			cmd := m.loadReleases()
			m.retryCmd = cmd
			return cmd
		}
	}
	return nil
}

func (m *MainScreen) getContentCount() int {
	switch m.contentTab {
	case TabFiles:
		return len(m.files)
	case TabMRs:
		return len(m.mergeRequests)
	case TabPipelines:
		return len(m.pipelines)
	case TabReleases:
		return len(m.releases)
	}
	return 0
}

// View renders the screen
func (m *MainScreen) View() string {
	if m.width == 0 || m.height == 0 {
		return "Loading..."
	}

	if m.errMsg != "" {
		return fmt.Sprintf("Error: %s\n\nPress q to quit", m.errMsg)
	}

	// If popup is shown, render only the popup
	if m.showJobLogPopup {
		return m.renderJobLogPopup()
	}
	if m.showBranchPopup {
		return m.renderBranchPopup()
	}
	if m.showRunnersPopup {
		return m.renderRunnersPopup()
	}
	if m.showReleasePopup {
		return m.renderReleasePopup()
	}
	if m.showFolderBrowser {
		return m.renderFolderBrowser()
	}

	// Calculate dimensions using config ratios
	contentHeight := m.height - config.StatusBarHeight
	navWidth := int(float64(m.width) * config.NavigatorWidthRatio)
	contentWidth := m.width - navWidth

	// Render panels
	navPanel := m.renderNavigatorPanel(navWidth, contentHeight)
	contentPanel := m.renderContentPanel(contentWidth, contentHeight)

	// Combine all
	main := lipgloss.JoinHorizontal(lipgloss.Top, navPanel, contentPanel)
	statusBar := m.renderStatusBar()

	return main + "\n" + statusBar
}

func (m *MainScreen) renderNavigatorPanel(width, height int) string {
	var content strings.Builder

	if m.loading && len(m.treeNodes) == 0 {
		content.WriteString(m.loadingMsg)
	} else if len(m.treeNodes) == 0 {
		content.WriteString(styles.DimmedText.Render("No groups or projects"))
	} else {
		// Calculate visible area for scrolling
		visibleLines := height - config.BorderSize - 2 // account for borders and padding
		if visibleLines < 1 {
			visibleLines = 10
		}

		// Calculate scroll offset to keep selected item visible
		scrollOffset := 0
		if m.selectedNodeIdx >= visibleLines {
			scrollOffset = m.selectedNodeIdx - visibleLines + 1
		}
		endIdx := scrollOffset + visibleLines
		if endIdx > len(m.treeNodes) {
			endIdx = len(m.treeNodes)
		}

		for i := scrollOffset; i < endIdx; i++ {
			node := m.treeNodes[i]

			// Build indent based on depth
			indent := strings.Repeat("  ", node.Depth)

			// Build icon
			icon := ""
			if node.Type == "group" {
				if m.expandedGroups[node.ID] {
					icon = "▼ "
				} else {
					icon = "▶ "
				}
			} else {
				icon = "  📦 "
			}

			line := indent + icon + node.Name

			// Truncate if too long
			maxLineLen := width - config.BorderSize - 4
			if maxLineLen > 0 && len(line) > maxLineLen {
				line = line[:maxLineLen-1] + "…"
			}

			if i == m.selectedNodeIdx {
				line = styles.SelectedItem.Render("> " + line)
			} else {
				line = styles.NormalItem.Render("  " + line)
			}
			content.WriteString(line + "\n")
		}

		// Show scroll indicator
		if len(m.treeNodes) > visibleLines {
			content.WriteString(styles.DimmedText.Render(fmt.Sprintf("\n[%d/%d]", m.selectedNodeIdx+1, len(m.treeNodes))))
		}
	}

	return components.SimpleBorderedPanel("Navigator", content.String(), width, height, m.focusedPanel == PanelNavigator)
}

func (m *MainScreen) renderContentPanel(width, height int) string {
	// Split: top half for file list, bottom half for README (only in Files tab at root)
	showReadme := m.contentTab == TabFiles && len(m.currentPath) == 0 && m.readmeContent != ""

	listHeight := height
	readmeHeight := 0
	if showReadme {
		readmeHeight = int(float64(height) * config.ReadmeHeightRatio)
		listHeight = height - readmeHeight
	}

	// Build the file/content list panel
	listPanel := m.renderListSection(width, listHeight)

	if !showReadme {
		return listPanel
	}

	// Build the README panel
	readmePanel := m.renderReadmeSection(width, readmeHeight)
	return lipgloss.JoinVertical(lipgloss.Left, listPanel, readmePanel)
}

func (m *MainScreen) renderListSection(width, height int) string {
	var content strings.Builder

	// Project header with branch and last commit
	if m.selectedProject != nil {
		projectHeader := styles.SelectedItem.Render(m.selectedProject.Name)
		if m.currentBranch != "" {
			projectHeader += styles.DimmedText.Render(" (" + m.currentBranch + ")")
		}
		content.WriteString(projectHeader + "\n")

		// Show last commit from current branch
		for _, b := range m.branches {
			if b.Name == m.currentBranch && b.Commit.Title != "" {
				commitInfo := styles.DimmedText.Render("Last commit: ") + truncateString(b.Commit.Title, width-20)
				if b.Commit.AuthorName != "" {
					commitInfo += styles.DimmedText.Render(" by " + b.Commit.AuthorName)
				}
				content.WriteString(commitInfo + "\n")
				break
			}
		}
	}

	// Tab header
	for i, name := range contentTabNames {
		if ContentTab(i) == m.contentTab {
			content.WriteString(styles.StatusBarKey.Render("[" + name + "]"))
		} else {
			content.WriteString(styles.DimmedText.Render(" " + name + " "))
		}
		content.WriteString(" ")
	}
	content.WriteString("\n")

	// Path breadcrumb for files
	if m.contentTab == TabFiles && len(m.currentPath) > 0 {
		content.WriteString(styles.DimmedText.Render("/"+strings.Join(m.currentPath, "/")) + "\n")
	}
	content.WriteString("\n")

	if m.selectedProject == nil {
		content.WriteString(styles.DimmedText.Render("Select a project"))
	} else if m.loading {
		content.WriteString(m.loadingMsg)
	} else {
		// Calculate visible lines for scrolling
		visibleLines := height - 6 // account for headers and borders
		if visibleLines < 1 {
			visibleLines = 10
		}

		switch m.contentTab {
		case TabFiles:
			// If viewing a file, show its content
			if m.viewingFile && m.fileContent != "" {
				// Show file path
				content.WriteString(styles.DimmedText.Render(m.viewingFilePath) + "\n")
				content.WriteString(styles.DimmedText.Render("Esc: back | j/k: scroll | g/G: top/bottom") + "\n\n")

				// Use viewport for file content
				fileViewHeight := visibleLines - 3
				innerWidth := width - 4
				if !m.fileViewReady {
					m.fileViewport = viewport.New(innerWidth, fileViewHeight)
					// Apply syntax highlighting
					highlighted := highlightCode(m.fileContent, m.viewingFilePath)
					m.fileViewport.SetContent(highlighted)
					m.fileViewReady = true
				}
				content.WriteString(m.fileViewport.View())

				// Scroll indicator
				if m.fileViewport.TotalLineCount() > fileViewHeight {
					content.WriteString(styles.DimmedText.Render(fmt.Sprintf("\n[%d%%]", int(m.fileViewport.ScrollPercent()*100))))
				}
			} else {
				// Show file list
				endIdx := m.fileScrollOffset + visibleLines
				if endIdx > len(m.files) {
					endIdx = len(m.files)
				}
				for i := m.fileScrollOffset; i < endIdx; i++ {
					f := m.files[i]
					icon := "📄"
					if f.Type == "tree" {
						icon = "📁"
					}
					// Build commit info
					commitInfo := ""
					if f.LastCommit != nil {
						commitInfo = fmt.Sprintf(" %s @%s", timeAgo(f.LastCommit.AuthoredDate), f.LastCommit.AuthorName)
					}
					line := fmt.Sprintf("%s %s", icon, f.Name)
					meta := styles.DimmedText.Render(commitInfo)
					if i == m.selectedContent {
						line = styles.SelectedItem.Render("> "+line) + meta
					} else {
						line = "  " + line + meta
					}
					content.WriteString(line + "\n")
				}
				// Show scroll indicator
				if len(m.files) > visibleLines {
					content.WriteString(styles.DimmedText.Render(fmt.Sprintf("\n[%d/%d]", m.selectedContent+1, len(m.files))))
				}
				// Show selected file info
				if m.selectedContent < len(m.files) {
					f := m.files[m.selectedContent]
					fileType := "File"
					if f.Type == "tree" {
						fileType = "Directory"
					}
					infoLine := fileType + ": " + f.Path
					if f.LastCommit != nil && f.LastCommit.Title != "" {
						infoLine += " | " + truncateString(f.LastCommit.Title, width-len(infoLine)-10)
					}
					content.WriteString("\n" + styles.DimmedText.Render(infoLine))
				}
			}
		case TabMRs:
			endIdx := m.fileScrollOffset + visibleLines
			if endIdx > len(m.mergeRequests) {
				endIdx = len(m.mergeRequests)
			}
			for i := m.fileScrollOffset; i < endIdx; i++ {
				mr := m.mergeRequests[i]
				icon := "○"
				if mr.Draft {
					icon = "◐"
				}
				// Build reviewer string
				reviewerStr := ""
				if len(mr.Reviewers) > 0 {
					reviewerStr = " → " + mr.Reviewers[0].Username
					if len(mr.Reviewers) > 1 {
						reviewerStr += fmt.Sprintf(" +%d", len(mr.Reviewers)-1)
					}
				}
				line := fmt.Sprintf("%s !%d %s", icon, mr.IID, truncateString(mr.Title, width-45))
				meta := styles.DimmedText.Render(fmt.Sprintf(" @%s%s %s", mr.Author.Username, reviewerStr, timeAgo(mr.CreatedAt)))
				if i == m.selectedContent {
					line = styles.SelectedItem.Render("> ") + line + meta
				} else {
					line = "  " + line + meta
				}
				content.WriteString(line + "\n")
			}
			if len(m.mergeRequests) == 0 {
				content.WriteString(styles.DimmedText.Render("No open merge requests"))
			} else {
				if len(m.mergeRequests) > visibleLines {
					content.WriteString(styles.DimmedText.Render(fmt.Sprintf("\n[%d/%d]", m.selectedContent+1, len(m.mergeRequests))))
				}
				// Show selected MR info
				if m.selectedContent < len(m.mergeRequests) {
					mr := m.mergeRequests[m.selectedContent]
					mrInfo := fmt.Sprintf("%s → %s", mr.SourceBranch, mr.TargetBranch)
					if mr.HasConflicts {
						mrInfo += " (conflicts)"
					}
					content.WriteString("\n" + styles.DimmedText.Render(mrInfo))
				}
			}
		case TabPipelines:
			endIdx := m.fileScrollOffset + visibleLines
			if endIdx > len(m.pipelines) {
				endIdx = len(m.pipelines)
			}
			for i := m.fileScrollOffset; i < endIdx; i++ {
				p := m.pipelines[i]
				icon := styles.PipelineIcon(p.Status)
				statusStyle := styles.PipelineStatus(p.Status)

				// Build job stages icons
				stagesStr := ""
				if jobs, ok := m.pipelineJobs[p.ID]; ok && len(jobs) > 0 {
					// Sort jobs by ID to get correct stage order (earlier stages have lower IDs)
					sortedJobs := make([]gitlab.Job, len(jobs))
					copy(sortedJobs, jobs)
					sort.Slice(sortedJobs, func(i, j int) bool {
						return sortedJobs[i].ID < sortedJobs[j].ID
					})
					// Group jobs by stage and get stage status
					stageOrder := []string{}
					stageStatus := make(map[string]string)
					for _, job := range sortedJobs {
						if _, exists := stageStatus[job.Stage]; !exists {
							stageOrder = append(stageOrder, job.Stage)
							stageStatus[job.Stage] = job.Status
						} else {
							// If any job in stage failed, stage is failed
							current := stageStatus[job.Stage]
							if job.Status == "failed" {
								stageStatus[job.Stage] = "failed"
							} else if job.Status == "running" && current != "failed" {
								stageStatus[job.Stage] = "running"
							} else if job.Status == "pending" && current != "failed" && current != "running" {
								stageStatus[job.Stage] = "pending"
							}
						}
					}
					// Build stage icons with names
					for _, stage := range stageOrder {
						status := stageStatus[stage]
						stageIcon := styles.PipelineIcon(status)
						stageStyle := styles.PipelineStatus(status)
						stagesStr += stageStyle.Render(stageIcon) + styles.DimmedText.Render("("+stage+")") + " "
					}
				} else {
					// No jobs loaded yet - show status text for pending/created pipelines
					stagesStr = statusStyle.Render("(" + p.Status + ")")
				}

				// Build meta info: user, time, source
				userStr := ""
				if p.User.Username != "" {
					userStr = "@" + p.User.Username
				}
				meta := styles.DimmedText.Render(fmt.Sprintf(" %s %s %s", userStr, p.Source, timeAgo(p.CreatedAt)))

				line := fmt.Sprintf("%s #%d %s %s", statusStyle.Render(icon), p.IID, p.Ref, stagesStr)
				if i == m.selectedContent {
					line = styles.SelectedItem.Render("> ") + line + meta
				} else {
					line = "  " + line + meta
				}
				content.WriteString(line + "\n")
			}
			if len(m.pipelines) == 0 {
				content.WriteString(styles.DimmedText.Render("No pipelines"))
			} else {
				if len(m.pipelines) > visibleLines {
					content.WriteString(styles.DimmedText.Render(fmt.Sprintf("\n[%d/%d]", m.selectedContent+1, len(m.pipelines))))
				}
				// Show selected pipeline info
				if m.selectedContent < len(m.pipelines) {
					p := m.pipelines[m.selectedContent]
					sha := p.SHA
					if len(sha) > 8 {
						sha = sha[:8]
					}
					pInfo := fmt.Sprintf("%s | %s", p.Status, sha)
					content.WriteString("\n" + styles.DimmedText.Render(pInfo))
				}
			}
		case TabReleases:
			endIdx := m.fileScrollOffset + visibleLines
			if endIdx > len(m.releases) {
				endIdx = len(m.releases)
			}
			for i := m.fileScrollOffset; i < endIdx; i++ {
				rel := m.releases[i]
				// Count downloadable assets (links + source archives)
				assetCount := len(rel.Assets.Links) + len(rel.Assets.Sources)
				assetStr := ""
				if assetCount > 0 {
					assetStr = fmt.Sprintf(" [%d]", assetCount)
				}

				// Format release time
				relTime := timeAgo(rel.CreatedAt)
				if rel.ReleasedAt != nil {
					relTime = timeAgo(*rel.ReleasedAt)
				}

				line := fmt.Sprintf("📦 %s%s", rel.TagName, assetStr)
				meta := styles.DimmedText.Render(fmt.Sprintf(" @%s %s", rel.Author.Username, relTime))
				if i == m.selectedContent {
					line = styles.SelectedItem.Render("> ") + line + meta
				} else {
					line = "  " + line + meta
				}
				content.WriteString(line + "\n")
			}
			if len(m.releases) == 0 {
				content.WriteString(styles.DimmedText.Render("No releases"))
			} else {
				if len(m.releases) > visibleLines {
					content.WriteString(styles.DimmedText.Render(fmt.Sprintf("\n[%d/%d]", m.selectedContent+1, len(m.releases))))
				}
				// Show selected release info
				if m.selectedContent < len(m.releases) {
					rel := m.releases[m.selectedContent]
					name := rel.Name
					if name == "" {
						name = rel.TagName
					}
					relInfo := fmt.Sprintf("%s | commit: %s", name, rel.Commit.ShortID)
					content.WriteString("\n" + styles.DimmedText.Render(relInfo))
				}
			}
		}
	}

	title := contentTabNames[m.contentTab]
	return components.SimpleBorderedPanel(title, content.String(), width, height, m.focusedPanel == PanelContent)
}

func (m *MainScreen) renderReadmeSection(width, height int) string {
	// Update viewport dimensions and content
	innerWidth := width - 4   // account for borders
	innerHeight := height - 3 // account for borders and title

	if !m.readmeReady {
		m.readmeViewport = viewport.New(innerWidth, innerHeight)
		m.readmeViewport.SetContent(m.readmeRendered)
		m.readmeReady = true
	} else {
		m.readmeViewport.Width = innerWidth
		m.readmeViewport.Height = innerHeight
	}

	// Build the panel manually with viewport content
	var content strings.Builder

	// Apply cursor and visual selection highlighting
	viewContent := m.readmeViewport.View()
	lines := strings.Split(viewContent, "\n")

	// Calculate visual selection range
	selStart := m.readmeVisualStart
	selEnd := m.readmeVisualEnd
	if selStart > selEnd {
		selStart, selEnd = selEnd, selStart
	}

	for i, line := range lines {
		viewportLine := m.readmeViewport.YOffset + i

		// Highlight visual selection
		if m.readmeVisualMode && viewportLine >= selStart && viewportLine <= selEnd {
			line = lipgloss.NewStyle().Background(lipgloss.Color("238")).Render(line)
		}

		// Show cursor line when focused
		if m.focusedPanel == PanelReadme && viewportLine == m.readmeCursor {
			line = lipgloss.NewStyle().Reverse(true).Render(line)
		}
		lines[i] = line
	}
	content.WriteString(strings.Join(lines, "\n"))

	// Add scroll indicator and visual mode status
	var statusParts []string
	if m.readmeViewport.TotalLineCount() > innerHeight {
		scrollPercent := int(m.readmeViewport.ScrollPercent() * 100)
		statusParts = append(statusParts, fmt.Sprintf("[%d%%]", scrollPercent))
	}
	if m.readmeVisualMode {
		lineCount := m.readmeVisualEnd - m.readmeVisualStart
		if lineCount < 0 {
			lineCount = -lineCount
		}
		lineCount++
		statusParts = append(statusParts, fmt.Sprintf("VISUAL(%d)", lineCount))
	}
	if len(statusParts) > 0 {
		content.WriteString(styles.DimmedText.Render(" " + strings.Join(statusParts, " ")))
	}

	return components.SimpleBorderedPanel("README", content.String(), width, height, m.focusedPanel == PanelReadme)
}

func (m *MainScreen) renderJobLogPopup() string {
	// Use full screen
	popupWidth := m.width
	popupHeight := m.height - 1

	// Split: left panel for job list, right panel for log
	jobListWidth := 30
	logWidth := popupWidth - jobListWidth

	// Render job list panel
	var jobList strings.Builder
	for i, job := range m.jobs {
		icon := styles.PipelineIcon(job.Status)
		statusStyle := styles.PipelineStatus(job.Status)

		// Format: icon name (status)
		line := fmt.Sprintf("%s %s (%s)", icon, job.Name, job.Status)

		// Truncate if too long
		if len(line) > jobListWidth-4 {
			line = line[:jobListWidth-5] + "…"
		}

		if i == m.selectedJobIdx {
			jobList.WriteString(styles.SelectedItem.Render("> " + statusStyle.Render(line)))
		} else {
			jobList.WriteString("  " + statusStyle.Render(line))
		}
		jobList.WriteString("\n")
	}

	// Job panel - focused when not in log
	jobPanel := components.SimpleBorderedPanel(
		fmt.Sprintf("Jobs (%d)", len(m.jobs)),
		jobList.String(),
		jobListWidth,
		popupHeight,
		!m.jobLogFocused,
	)

	// Render log panel
	logInnerWidth := logWidth - 2
	logInnerHeight := popupHeight - 2

	var logContent strings.Builder
	if m.jobLog == "" {
		if m.loading {
			logContent.WriteString(m.loadingMsg)
		} else {
			logContent.WriteString(styles.DimmedText.Render("Select a job to view log"))
		}
	} else {
		if !m.jobLogReady || m.jobLogViewport.Width != logInnerWidth || m.jobLogViewport.Height != logInnerHeight {
			m.jobLogViewport = viewport.New(logInnerWidth, logInnerHeight)
			// Keep ANSI colors but clean up problematic characters
			cleanLog := m.jobLog
			// Replace tabs with spaces (tabs mess up width calculation)
			cleanLog = strings.ReplaceAll(cleanLog, "\t", "    ")
			// Remove carriage returns (CI logs use these for progress updates)
			cleanLog = strings.ReplaceAll(cleanLog, "\r", "")
			// Don't wrap - truncate lines to preserve line numbers for visual selection
			m.jobLogViewport.SetContent(cleanLog)
			// Start at bottom where errors usually are
			m.jobLogViewport.GotoBottom()
			m.jobLogReady = true
		}
		// Get viewport content and apply cursor/selection highlighting + horizontal scroll
		viewContent := m.jobLogViewport.View()
		lines := strings.Split(viewContent, "\n")

		// Calculate visual selection range
		selStart := m.visualStartLine
		selEnd := m.visualEndLine
		if selStart > selEnd {
			selStart, selEnd = selEnd, selStart
		}

		for i, line := range lines {
			line = strings.ReplaceAll(line, "\t", "    ")
			// Apply horizontal scroll
			line = sliceByWidth(line, m.jobLogHScroll, logInnerWidth)

			viewportLine := m.jobLogViewport.YOffset + i

			// Highlight visual selection
			if m.visualLineMode && viewportLine >= selStart && viewportLine <= selEnd {
				line = lipgloss.NewStyle().Background(lipgloss.Color("238")).Render(line)
			}

			// Show cursor line when focused (on top of selection)
			if m.jobLogFocused && viewportLine == m.jobLogCursor {
				line = lipgloss.NewStyle().Reverse(true).Render(line)
			}
			lines[i] = line
		}
		logContent.WriteString(strings.Join(lines, "\n"))
	}

	// Build log title with job info
	logTitle := "Log"
	if m.selectedJobIdx < len(m.jobs) {
		job := m.jobs[m.selectedJobIdx]
		duration := ""
		if job.Duration > 0 {
			duration = fmt.Sprintf(" (%.1fs)", job.Duration)
		}
		logTitle = fmt.Sprintf("%s - %s%s", job.Name, job.Status, duration)
	}

	// Log panel - focused when in log
	logPanel := components.SimpleBorderedPanel(logTitle, logContent.String(), logWidth, popupHeight, m.jobLogFocused)

	// Join panels horizontally
	combined := lipgloss.JoinHorizontal(lipgloss.Top, jobPanel, logPanel)

	// Status bar
	scrollInfo := ""
	if m.jobLogReady && m.jobLogViewport.TotalLineCount() > logInnerHeight {
		scrollInfo = fmt.Sprintf(" [%d%%]", int(m.jobLogViewport.ScrollPercent()*100))
	}
	if m.jobLogHScroll > 0 {
		scrollInfo += fmt.Sprintf(" [→%d]", m.jobLogHScroll)
	}

	statusContent := styles.StatusBarKey.Render("H/L") + styles.StatusBarDesc.Render(" panels") + " │ " +
		styles.StatusBarKey.Render("hjkl") + styles.StatusBarDesc.Render(" nav") + " │ " +
		styles.StatusBarKey.Render("V") + styles.StatusBarDesc.Render(" select") + " │ " +
		styles.StatusBarKey.Render("yy") + styles.StatusBarDesc.Render(" yank") + " │ " +
		styles.StatusBarKey.Render("ggy") + styles.StatusBarDesc.Render(" all") + " │ " +
		styles.StatusBarKey.Render("q") + styles.StatusBarDesc.Render(" close") +
		scrollInfo

	if m.visualLineMode {
		lineCount := m.visualEndLine - m.visualStartLine
		if lineCount < 0 {
			lineCount = -lineCount
		}
		lineCount++
		statusContent = styles.SelectedItem.Render(fmt.Sprintf("VISUAL LINE (%d)", lineCount)) + " │ " + statusContent
	}

	if m.statusMsg != "" {
		statusContent = styles.SelectedItem.Render(m.statusMsg) + " │ " + statusContent
	}

	statusBar := styles.StatusBar.Width(m.width).Render(statusContent)

	return combined + "\n" + statusBar
}

func (m *MainScreen) renderBranchPopup() string {
	// Centered popup for branch selection
	popupWidth := 50
	popupHeight := 20

	if popupWidth > m.width-4 {
		popupWidth = m.width - 4
	}
	if popupHeight > m.height-4 {
		popupHeight = m.height - 4
	}

	var content strings.Builder

	// Header
	content.WriteString(styles.DimmedText.Render("Current: ") + styles.SelectedItem.Render(m.currentBranch) + "\n\n")

	if len(m.branches) == 0 {
		if m.loading {
			content.WriteString(m.loadingMsg)
		} else {
			content.WriteString(styles.DimmedText.Render("No branches found"))
		}
	} else {
		visibleLines := popupHeight - 6
		if visibleLines < 5 {
			visibleLines = 5
		}

		// Calculate scroll offset for branch list
		startIdx := 0
		if m.selectedBranchIdx >= visibleLines {
			startIdx = m.selectedBranchIdx - visibleLines + 1
		}
		endIdx := startIdx + visibleLines
		if endIdx > len(m.branches) {
			endIdx = len(m.branches)
		}

		for i := startIdx; i < endIdx; i++ {
			b := m.branches[i]
			icon := "○"
			if b.Default {
				icon = "●"
			}
			if b.Name == m.currentBranch {
				icon = "✓"
			}

			line := fmt.Sprintf("%s %s", icon, b.Name)
			if i == m.selectedBranchIdx {
				line = styles.SelectedItem.Render("> " + line)
			} else {
				line = "  " + line
			}
			content.WriteString(line + "\n")
		}

		// Scroll indicator
		if len(m.branches) > visibleLines {
			content.WriteString(styles.DimmedText.Render(fmt.Sprintf("\n[%d/%d]", m.selectedBranchIdx+1, len(m.branches))))
		}
	}

	// Build popup panel
	popup := components.SimpleBorderedPanel("Switch Branch", content.String(), popupWidth, popupHeight, true)

	// Center the popup
	popupLines := strings.Split(popup, "\n")
	topPadding := (m.height - len(popupLines)) / 2
	leftPadding := (m.width - popupWidth) / 2
	if topPadding < 0 {
		topPadding = 0
	}
	if leftPadding < 0 {
		leftPadding = 0
	}

	var result strings.Builder
	for i := 0; i < topPadding; i++ {
		result.WriteString("\n")
	}
	for _, line := range popupLines {
		result.WriteString(strings.Repeat(" ", leftPadding) + line + "\n")
	}

	// Status bar at bottom
	statusContent := styles.StatusBarKey.Render("Esc") + styles.StatusBarDesc.Render(" cancel") + " │ " +
		styles.StatusBarKey.Render("j/k") + styles.StatusBarDesc.Render(" navigate") + " │ " +
		styles.StatusBarKey.Render("Enter") + styles.StatusBarDesc.Render(" switch")

	// Pad to bottom
	currentLines := topPadding + len(popupLines)
	for i := currentLines; i < m.height-1; i++ {
		result.WriteString("\n")
	}

	result.WriteString(styles.StatusBar.Width(m.width).Render(statusContent))

	return result.String()
}

func (m *MainScreen) renderRunnersPopup() string {
	// Larger popup for runners view
	popupWidth := int(float64(m.width) * 0.8)
	popupHeight := int(float64(m.height) * 0.8)

	if popupWidth < 60 {
		popupWidth = 60
	}
	if popupHeight < 15 {
		popupHeight = 15
	}
	if popupWidth > m.width-4 {
		popupWidth = m.width - 4
	}
	if popupHeight > m.height-4 {
		popupHeight = m.height - 4
	}

	var content strings.Builder

	// Tab headers
	runningTab := fmt.Sprintf("Running (%d)", len(m.runningJobs))
	pendingTab := fmt.Sprintf("Pending (%d)", len(m.pendingJobs))

	if m.runnersTab == 0 {
		content.WriteString(styles.SelectedItem.Render("["+runningTab+"]") + " " + styles.DimmedText.Render(pendingTab))
	} else {
		content.WriteString(styles.DimmedText.Render(runningTab) + " " + styles.SelectedItem.Render("["+pendingTab+"]"))
	}
	content.WriteString("\n\n")

	// Get current job list
	jobs := m.runningJobs
	if m.runnersTab == 1 {
		jobs = m.pendingJobs
	}

	if m.runnersLoading {
		content.WriteString(styles.DimmedText.Render("Loading jobs..."))
	} else if len(jobs) == 0 {
		if m.runnersTab == 0 {
			content.WriteString(styles.DimmedText.Render("No running jobs"))
		} else {
			content.WriteString(styles.DimmedText.Render("No pending jobs"))
		}
	} else {
		visibleLines := popupHeight - 8
		if visibleLines < 5 {
			visibleLines = 5
		}

		// Calculate scroll offset
		startIdx := 0
		if m.runnersCursor >= visibleLines {
			startIdx = m.runnersCursor - visibleLines + 1
		}
		endIdx := startIdx + visibleLines
		if endIdx > len(jobs) {
			endIdx = len(jobs)
		}

		// Column header
		header := fmt.Sprintf("%-20s %-30s %-15s %s", "PROJECT", "JOB", "RUNNER", "DURATION")
		content.WriteString(styles.DimmedText.Render(header) + "\n")
		content.WriteString(styles.DimmedText.Render(strings.Repeat("─", popupWidth-4)) + "\n")

		for i := startIdx; i < endIdx; i++ {
			job := jobs[i]
			icon := styles.PipelineIcon(job.Status)
			statusStyle := styles.PipelineStatus(job.Status)

			// Project name (truncate if needed)
			project := job.Project.Name
			if len(project) > 18 {
				project = project[:17] + "…"
			}

			// Job name (truncate if needed)
			jobName := job.Name
			if job.Stage != "" && job.Stage != job.Name {
				jobName = job.Stage + "/" + job.Name
			}
			if len(jobName) > 28 {
				jobName = jobName[:27] + "…"
			}

			// Runner info
			runnerName := "-"
			if job.Runner != nil {
				runnerName = job.Runner.Description
				if runnerName == "" {
					runnerName = job.Runner.Name
				}
				if runnerName == "" {
					runnerName = fmt.Sprintf("#%d", job.Runner.ID)
				}
			}
			if len(runnerName) > 13 {
				runnerName = runnerName[:12] + "…"
			}

			// Duration
			duration := "-"
			if job.Duration > 0 {
				duration = fmt.Sprintf("%.0fs", job.Duration)
			} else if job.StartedAt != nil {
				duration = timeAgo(*job.StartedAt)
			}

			line := fmt.Sprintf("%s %-20s %-30s %-15s %s",
				statusStyle.Render(icon),
				project,
				jobName,
				runnerName,
				duration)

			if i == m.runnersCursor {
				line = styles.SelectedItem.Render("> ") + line
			} else {
				line = "  " + line
			}
			content.WriteString(line + "\n")
		}

		// Scroll indicator
		if len(jobs) > visibleLines {
			content.WriteString(styles.DimmedText.Render(fmt.Sprintf("\n[%d/%d]", m.runnersCursor+1, len(jobs))))
		}
	}

	// Build popup panel
	title := "CI/CD Jobs"
	if m.runnersLoading {
		title += " (loading...)"
	}
	popup := components.SimpleBorderedPanel(title, content.String(), popupWidth, popupHeight, true)

	// Center the popup
	popupLines := strings.Split(popup, "\n")
	topPadding := (m.height - len(popupLines)) / 2
	leftPadding := (m.width - popupWidth) / 2
	if topPadding < 0 {
		topPadding = 0
	}
	if leftPadding < 0 {
		leftPadding = 0
	}

	var result strings.Builder
	for i := 0; i < topPadding; i++ {
		result.WriteString("\n")
	}
	for _, line := range popupLines {
		result.WriteString(strings.Repeat(" ", leftPadding) + line + "\n")
	}

	// Status bar at bottom
	statusContent := styles.StatusBarKey.Render("Esc") + styles.StatusBarDesc.Render(" close") + " │ " +
		styles.StatusBarKey.Render("Tab") + styles.StatusBarDesc.Render(" switch") + " │ " +
		styles.StatusBarKey.Render("j/k") + styles.StatusBarDesc.Render(" navigate") + " │ " +
		styles.StatusBarKey.Render("r") + styles.StatusBarDesc.Render(" refresh")

	// Pad to bottom
	currentLines := topPadding + len(popupLines)
	for i := currentLines; i < m.height-1; i++ {
		result.WriteString("\n")
	}

	result.WriteString(styles.StatusBar.Width(m.width).Render(statusContent))

	return result.String()
}

func (m *MainScreen) renderStatusBar() string {
	// If there's a status message, show it prominently
	if m.statusMsg != "" {
		msgStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("42")).Bold(true) // Green
		return styles.StatusBar.Width(m.width).Render(msgStyle.Render(m.statusMsg))
	}

	// If there's an error, show it prominently with retry hint
	if m.lastError != "" {
		errorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Bold(true) // Red
		errorMsg := m.lastError
		// Truncate long error messages
		maxLen := m.width - 30
		if maxLen > 0 && len(errorMsg) > maxLen {
			errorMsg = errorMsg[:maxLen] + "..."
		}
		errText := errorStyle.Render("Error: " + errorMsg)
		retryHint := styles.StatusBarKey.Render(" r") + styles.StatusBarDesc.Render(" retry") + " │ " +
			styles.StatusBarKey.Render("Esc") + styles.StatusBarDesc.Render(" dismiss")
		return styles.StatusBar.Width(m.width).Render(errText + " " + retryHint)
	}

	panels := []struct {
		id   PanelID
		key  string
		name string
	}{
		{PanelNavigator, "1", "navigator"},
		{PanelContent, "2", "content"},
		{PanelReadme, "3", "readme"},
	}

	var parts []string
	for _, p := range panels {
		if p.id == m.focusedPanel {
			parts = append(parts, styles.StatusBarKey.Render("["+p.key+"]")+styles.StatusBarDesc.Render(" "+p.name))
		} else {
			parts = append(parts, styles.DimmedText.Render(" "+p.key+" "+p.name))
		}
	}

	left := strings.Join(parts, " ")

	var help string
	if m.focusedPanel == PanelReadme {
		// README-specific keybindings
		help = styles.StatusBarKey.Render("j/k") + styles.StatusBarDesc.Render(" scroll") + " │ " +
			styles.StatusBarKey.Render("V") + styles.StatusBarDesc.Render(" select") + " │ " +
			styles.StatusBarKey.Render("yy") + styles.StatusBarDesc.Render(" yank") + " │ " +
			styles.StatusBarKey.Render("ggy") + styles.StatusBarDesc.Render(" all") + " │ " +
			styles.StatusBarKey.Render("q") + styles.StatusBarDesc.Render(" quit")
	} else {
		help = styles.StatusBarKey.Render("j/k") + styles.StatusBarDesc.Render(" nav") + " │ " +
			styles.StatusBarKey.Render("Enter") + styles.StatusBarDesc.Render(" select") + " │ " +
			styles.StatusBarKey.Render("S") + styles.StatusBarDesc.Render(" ssh") + " " +
			styles.StatusBarKey.Render("U") + styles.StatusBarDesc.Render(" https") + " │ " +
			styles.StatusBarKey.Render("R") + styles.StatusBarDesc.Render(" jobs") + " │ " +
			styles.StatusBarKey.Render("q") + styles.StatusBarDesc.Render(" quit")
	}

	leftWidth := lipgloss.Width(left)
	rightWidth := lipgloss.Width(help)
	padding := m.width - leftWidth - rightWidth - 2
	if padding < 0 {
		padding = 0
	}

	return styles.StatusBar.Width(m.width).Render(left + strings.Repeat(" ", padding) + help)
}

// handleReleasePopup handles keyboard input for the release assets popup
func (m *MainScreen) handleReleasePopup(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.selectedReleaseIdx >= len(m.releases) {
		m.showReleasePopup = false
		return m, nil
	}

	rel := m.releases[m.selectedReleaseIdx]
	totalAssets := len(rel.Assets.Sources) + len(rel.Assets.Links)

	switch msg.String() {
	case "esc", "escape", "q":
		m.showReleasePopup = false
		return m, nil
	case "j", "down":
		if m.releaseAssetCursor < totalAssets-1 {
			m.releaseAssetCursor++
		}
	case "k", "up":
		if m.releaseAssetCursor > 0 {
			m.releaseAssetCursor--
		}
	case "g":
		m.releaseAssetCursor = 0
	case "G":
		if totalAssets > 0 {
			m.releaseAssetCursor = totalAssets - 1
		}
	case "y", "enter":
		// Copy the URL of selected asset to clipboard
		url := m.getSelectedReleaseAssetURL()
		if url != "" {
			if err := copyToClipboard(url); err != nil {
				m.statusMsg = "Copy failed: " + err.Error()
			} else {
				m.statusMsg = "Copied: " + truncateString(url, 60)
			}
		}
		return m, nil
	case "o":
		// Open release web URL in browser
		if rel.Links.Self != "" {
			m.statusMsg = "Open: " + rel.Links.Self
			// Just copy for now - could open browser in future
			if err := copyToClipboard(rel.Links.Self); err != nil {
				m.statusMsg = "Copy failed: " + err.Error()
			} else {
				m.statusMsg = "Copied release URL: " + truncateString(rel.Links.Self, 50)
			}
		}
		return m, nil
	case "d":
		// Download the selected asset - open folder browser
		url := m.getSelectedReleaseAssetURL()
		filename := m.getSelectedReleaseAssetFilename()
		if url != "" && filename != "" {
			m.downloadURL = url
			m.downloadFilename = filename
			m.showReleasePopup = false
			m.openFolderBrowser()
		}
		return m, nil
	}

	return m, nil
}

// getSelectedReleaseAssetFilename returns the filename of the currently selected asset
func (m *MainScreen) getSelectedReleaseAssetFilename() string {
	if m.selectedReleaseIdx >= len(m.releases) {
		return ""
	}

	rel := m.releases[m.selectedReleaseIdx]
	cursor := m.releaseAssetCursor

	// First, source archives
	if cursor < len(rel.Assets.Sources) {
		src := rel.Assets.Sources[cursor]
		return fmt.Sprintf("%s-%s.%s", rel.TagName, "source", src.Format)
	}

	// Then, asset links
	linkIdx := cursor - len(rel.Assets.Sources)
	if linkIdx < len(rel.Assets.Links) {
		return rel.Assets.Links[linkIdx].Name
	}

	return ""
}

// openFolderBrowser initializes and shows the folder browser
func (m *MainScreen) openFolderBrowser() {
	// Start from home directory
	home, err := os.UserHomeDir()
	if err != nil {
		home = "/"
	}
	m.folderBrowserPath = home
	m.folderBrowserCursor = 0
	m.folderBrowserScroll = 0
	m.loadFolderEntries()
	m.showFolderBrowser = true
}

// loadFolderEntries loads directory entries for the current folder browser path
func (m *MainScreen) loadFolderEntries() {
	entries, err := os.ReadDir(m.folderBrowserPath)
	if err != nil {
		m.folderBrowserEntries = []string{}
		return
	}

	m.folderBrowserEntries = []string{}
	for _, entry := range entries {
		name := entry.Name()
		// Skip hidden files/directories (starting with .)
		if strings.HasPrefix(name, ".") {
			continue
		}
		if entry.IsDir() {
			m.folderBrowserEntries = append(m.folderBrowserEntries, name)
		}
	}
	sort.Strings(m.folderBrowserEntries)
}

// getSelectedReleaseAssetURL returns the URL of the currently selected asset
func (m *MainScreen) getSelectedReleaseAssetURL() string {
	if m.selectedReleaseIdx >= len(m.releases) {
		return ""
	}

	rel := m.releases[m.selectedReleaseIdx]
	cursor := m.releaseAssetCursor

	// First, source archives
	if cursor < len(rel.Assets.Sources) {
		return rel.Assets.Sources[cursor].URL
	}

	// Then, asset links
	linkIdx := cursor - len(rel.Assets.Sources)
	if linkIdx < len(rel.Assets.Links) {
		return rel.Assets.Links[linkIdx].URL
	}

	return ""
}

// renderReleasePopup renders the release assets popup
func (m *MainScreen) renderReleasePopup() string {
	if m.selectedReleaseIdx >= len(m.releases) {
		return ""
	}

	rel := m.releases[m.selectedReleaseIdx]

	// Popup dimensions
	popupWidth := min(m.width-4, 80)
	popupHeight := min(m.height-4, 30)

	var content strings.Builder

	// Release info header
	name := rel.Name
	if name == "" {
		name = rel.TagName
	}
	content.WriteString(styles.ActivePanelTitle.Render("Release: "+name) + "\n")
	content.WriteString(styles.DimmedText.Render("Tag: "+rel.TagName) + "\n")
	if rel.Commit.ShortID != "" {
		content.WriteString(styles.DimmedText.Render("Commit: "+rel.Commit.ShortID) + "\n")
	}
	content.WriteString("\n")

	// Downloadable assets
	content.WriteString(styles.ActivePanelTitle.Render("Downloads:") + "\n")

	visibleLines := popupHeight - 10
	cursor := 0

	// Source archives first
	for i, src := range rel.Assets.Sources {
		icon := "📦"
		line := fmt.Sprintf("%s Source code (%s)", icon, src.Format)

		if cursor == m.releaseAssetCursor {
			content.WriteString(styles.SelectedItem.Render("> ") + line + "\n")
		} else {
			content.WriteString("  " + line + "\n")
		}
		cursor++

		if i >= visibleLines {
			break
		}
	}

	// Asset links
	for i, link := range rel.Assets.Links {
		icon := "📎"
		switch link.LinkType {
		case "package":
			icon = "📦"
		case "image":
			icon = "🖼️"
		case "runbook":
			icon = "📖"
		}

		line := fmt.Sprintf("%s %s", icon, link.Name)
		if len(line) > popupWidth-6 {
			line = line[:popupWidth-7] + "…"
		}

		if cursor == m.releaseAssetCursor {
			content.WriteString(styles.SelectedItem.Render("> ") + line + "\n")
		} else {
			content.WriteString("  " + line + "\n")
		}
		cursor++

		if i+len(rel.Assets.Sources) >= visibleLines {
			break
		}
	}

	totalAssets := len(rel.Assets.Sources) + len(rel.Assets.Links)
	if totalAssets == 0 {
		content.WriteString(styles.DimmedText.Render("  No downloadable assets") + "\n")
	} else {
		content.WriteString(styles.DimmedText.Render(fmt.Sprintf("\n[%d/%d]", m.releaseAssetCursor+1, totalAssets)) + "\n")
	}

	// Show selected URL
	selectedURL := m.getSelectedReleaseAssetURL()
	if selectedURL != "" {
		content.WriteString("\n" + styles.DimmedText.Render("URL: "+truncateString(selectedURL, popupWidth-8)) + "\n")
	}

	// Build popup panel
	popup := components.SimpleBorderedPanel(
		fmt.Sprintf("Release: %s", rel.TagName),
		content.String(),
		popupWidth,
		popupHeight,
		true,
	)

	// Center the popup
	popupLines := strings.Split(popup, "\n")
	topPadding := (m.height - len(popupLines)) / 2
	leftPadding := (m.width - popupWidth) / 2
	if topPadding < 0 {
		topPadding = 0
	}
	if leftPadding < 0 {
		leftPadding = 0
	}

	var result strings.Builder
	for i := 0; i < topPadding; i++ {
		result.WriteString("\n")
	}
	for _, line := range popupLines {
		result.WriteString(strings.Repeat(" ", leftPadding) + line + "\n")
	}

	// Status bar at bottom
	statusContent := styles.StatusBarKey.Render("Esc") + styles.StatusBarDesc.Render(" close") + " │ " +
		styles.StatusBarKey.Render("j/k") + styles.StatusBarDesc.Render(" navigate") + " │ " +
		styles.StatusBarKey.Render("y/Enter") + styles.StatusBarDesc.Render(" copy URL") + " │ " +
		styles.StatusBarKey.Render("d") + styles.StatusBarDesc.Render(" download") + " │ " +
		styles.StatusBarKey.Render("o") + styles.StatusBarDesc.Render(" copy release URL")

	// Pad to bottom
	currentLines := topPadding + len(popupLines)
	for i := currentLines; i < m.height-1; i++ {
		result.WriteString("\n")
	}

	result.WriteString(styles.StatusBar.Width(m.width).Render(statusContent))

	return result.String()
}

// handleFolderBrowser handles keyboard input for the folder browser popup
func (m *MainScreen) handleFolderBrowser(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "escape", "q":
		m.showFolderBrowser = false
		m.downloadURL = ""
		m.downloadFilename = ""
		return m, nil

	case "j", "down":
		if m.folderBrowserCursor < len(m.folderBrowserEntries)-1 {
			m.folderBrowserCursor++
			// Adjust scroll - match the visibleLines calculation in renderFolderBrowser
			popupHeight := min(m.height-4, 25)
			visibleLines := popupHeight - 12
			if visibleLines < 3 {
				visibleLines = 3
			}
			if m.folderBrowserCursor >= m.folderBrowserScroll+visibleLines {
				m.folderBrowserScroll = m.folderBrowserCursor - visibleLines + 1
			}
		}

	case "k", "up":
		if m.folderBrowserCursor > 0 {
			m.folderBrowserCursor--
			if m.folderBrowserCursor < m.folderBrowserScroll {
				m.folderBrowserScroll = m.folderBrowserCursor
			}
		}

	case "g":
		m.folderBrowserCursor = 0
		m.folderBrowserScroll = 0

	case "G":
		if len(m.folderBrowserEntries) > 0 {
			m.folderBrowserCursor = len(m.folderBrowserEntries) - 1
			popupHeight := min(m.height-4, 25)
			visibleLines := popupHeight - 12
			if visibleLines < 3 {
				visibleLines = 3
			}
			if m.folderBrowserCursor >= visibleLines {
				m.folderBrowserScroll = m.folderBrowserCursor - visibleLines + 1
			}
		}

	case "l", "enter":
		// Enter selected directory
		if m.folderBrowserCursor < len(m.folderBrowserEntries) {
			selectedDir := m.folderBrowserEntries[m.folderBrowserCursor]
			newPath := filepath.Join(m.folderBrowserPath, selectedDir)
			// Verify it's accessible
			if _, err := os.ReadDir(newPath); err == nil {
				m.folderBrowserPath = newPath
				m.folderBrowserCursor = 0
				m.folderBrowserScroll = 0
				m.loadFolderEntries()
			}
		}

	case "h", "backspace":
		// Go up one directory
		parent := filepath.Dir(m.folderBrowserPath)
		if parent != m.folderBrowserPath {
			m.folderBrowserPath = parent
			m.folderBrowserCursor = 0
			m.folderBrowserScroll = 0
			m.loadFolderEntries()
		}

	case "~":
		// Go to home directory
		if home, err := os.UserHomeDir(); err == nil {
			m.folderBrowserPath = home
			m.folderBrowserCursor = 0
			m.folderBrowserScroll = 0
			m.loadFolderEntries()
		}

	case "d", " ":
		// Download to current directory
		if m.downloadURL != "" && m.downloadFilename != "" {
			destPath := filepath.Join(m.folderBrowserPath, m.downloadFilename)
			m.showFolderBrowser = false
			m.loading = true
			m.loadingMsg = "Downloading " + m.downloadFilename + "..."

			// Start download in background
			url := m.downloadURL
			filename := m.downloadFilename
			client := m.client

			return m, func() tea.Msg {
				bytes, err := client.DownloadFile(url, destPath)
				return downloadCompleteMsg{
					filename: filename,
					bytes:    bytes,
					err:      err,
				}
			}
		}
	}

	return m, nil
}

// renderFolderBrowser renders the folder browser popup
func (m *MainScreen) renderFolderBrowser() string {
	popupWidth := min(m.width-4, 70)
	popupHeight := min(m.height-4, 25)

	var content strings.Builder

	// Current path
	content.WriteString(styles.ActivePanelTitle.Render("Location:") + "\n")
	displayPath := m.folderBrowserPath
	if len(displayPath) > popupWidth-6 {
		displayPath = "..." + displayPath[len(displayPath)-popupWidth+9:]
	}
	content.WriteString(styles.DimmedText.Render(displayPath) + "\n\n")

	// File to download
	content.WriteString(styles.ActivePanelTitle.Render("File:") + " " + m.downloadFilename + "\n\n")

	// Directory listing
	content.WriteString(styles.ActivePanelTitle.Render("Folders:") + "\n")

	visibleLines := popupHeight - 12
	if visibleLines < 3 {
		visibleLines = 3
	}

	endIdx := m.folderBrowserScroll + visibleLines
	if endIdx > len(m.folderBrowserEntries) {
		endIdx = len(m.folderBrowserEntries)
	}

	if len(m.folderBrowserEntries) == 0 {
		content.WriteString(styles.DimmedText.Render("  (empty directory)") + "\n")
	} else {
		for i := m.folderBrowserScroll; i < endIdx; i++ {
			entry := m.folderBrowserEntries[i]
			icon := "📁"

			line := fmt.Sprintf("%s %s", icon, entry)
			if len(line) > popupWidth-6 {
				line = line[:popupWidth-7] + "…"
			}

			if i == m.folderBrowserCursor {
				content.WriteString(styles.SelectedItem.Render("> ") + line + "\n")
			} else {
				content.WriteString("  " + line + "\n")
			}
		}
	}

	// Scroll indicator
	if len(m.folderBrowserEntries) > visibleLines {
		content.WriteString(styles.DimmedText.Render(fmt.Sprintf("\n[%d/%d]", m.folderBrowserCursor+1, len(m.folderBrowserEntries))) + "\n")
	}

	// Build popup panel
	popup := components.SimpleBorderedPanel(
		"Select Download Location",
		content.String(),
		popupWidth,
		popupHeight,
		true,
	)

	// Center the popup
	popupLines := strings.Split(popup, "\n")
	topPadding := (m.height - len(popupLines)) / 2
	leftPadding := (m.width - popupWidth) / 2
	if topPadding < 0 {
		topPadding = 0
	}
	if leftPadding < 0 {
		leftPadding = 0
	}

	var result strings.Builder
	for i := 0; i < topPadding; i++ {
		result.WriteString("\n")
	}
	for _, line := range popupLines {
		result.WriteString(strings.Repeat(" ", leftPadding) + line + "\n")
	}

	// Status bar at bottom
	folderStatusContent := styles.StatusBarKey.Render("Esc") + styles.StatusBarDesc.Render(" cancel") + " │ " +
		styles.StatusBarKey.Render("j/k") + styles.StatusBarDesc.Render(" navigate") + " │ " +
		styles.StatusBarKey.Render("l/Enter") + styles.StatusBarDesc.Render(" open") + " │ " +
		styles.StatusBarKey.Render("h/Bksp") + styles.StatusBarDesc.Render(" up") + " │ " +
		styles.StatusBarKey.Render("~") + styles.StatusBarDesc.Render(" home") + " │ " +
		styles.StatusBarKey.Render("d/Space") + styles.StatusBarDesc.Render(" download here")

	// Pad to bottom
	folderCurrentLines := topPadding + len(popupLines)
	for i := folderCurrentLines; i < m.height-1; i++ {
		result.WriteString("\n")
	}

	result.WriteString(styles.StatusBar.Width(m.width).Render(folderStatusContent))

	return result.String()
}
