package app

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"

	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/formatters"
	"github.com/alecthomas/chroma/v2/lexers"
	chromaStyles "github.com/alecthomas/chroma/v2/styles"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/espen/lazylab/internal/config"
	"github.com/espen/lazylab/internal/gitlab"
	"github.com/espen/lazylab/internal/keymap"
	"github.com/espen/lazylab/internal/ui/components"
	"github.com/espen/lazylab/internal/ui/styles"
)

// copyToClipboard copies text to the system clipboard
func copyToClipboard(text string) error {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("pbcopy")
	case "linux":
		// Try xclip first, fall back to xsel
		if _, err := exec.LookPath("xclip"); err == nil {
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

// getFileExtension returns the extension of a file path
func getFileExtension(path string) string {
	return filepath.Ext(path)
}

// stripANSI removes ANSI escape codes from a string
func stripANSI(s string) string {
	return ansiRegex.ReplaceAllString(s, "")
}

// wrapText wraps all lines in text to fit within maxWidth
func wrapText(text string, maxWidth int) string {
	if maxWidth <= 0 {
		return text
	}

	lines := strings.Split(text, "\n")
	var result []string

	for _, line := range lines {
		if len(line) <= maxWidth {
			result = append(result, line)
			continue
		}

		// Wrap long line
		remaining := line
		for len(remaining) > maxWidth {
			// Find break point
			breakAt := maxWidth
			for i := maxWidth; i > maxWidth/2; i-- {
				if remaining[i] == ' ' {
					breakAt = i
					break
				}
			}
			result = append(result, remaining[:breakAt])
			remaining = remaining[breakAt:]
			if len(remaining) > 0 && remaining[0] == ' ' {
				remaining = remaining[1:]
			}
		}
		if len(remaining) > 0 {
			result = append(result, remaining)
		}
	}

	return strings.Join(result, "\n")
}

// PanelID identifies panels in the UI
type PanelID int

const (
	PanelGroups PanelID = iota
	PanelProjects
	PanelContent
	PanelReadme
	PanelDetail
	PanelCount
)

// ContentTab identifies tabs in the content panel
type ContentTab int

const (
	TabFiles ContentTab = iota
	TabMRs
	TabPipelines
	TabCount
)

var contentTabNames = []string{"Files", "MRs", "Pipelines"}

// MainScreen is the lazygit-style multi-panel interface
type MainScreen struct {
	// GitLab client
	client *gitlab.Client

	// Current group
	groupPath string

	// Data
	groups        []gitlab.Group
	projects      []gitlab.Project
	files         []gitlab.TreeEntry
	mergeRequests []gitlab.MergeRequest
	pipelines     []gitlab.Pipeline
	branches      []gitlab.Branch
	jobs          []gitlab.Job
	jobLog        string

	// Jobs per pipeline (for showing stages in list)
	pipelineJobs map[int][]gitlab.Job

	// Selected project
	selectedProject *gitlab.Project

	// File browser state
	currentPath    []string
	fileContent    string
	readmeContent  string
	viewingFile    bool
	viewingFilePath string

	// Selection indices
	selectedGroupIdx   int
	selectedProjectIdx int
	selectedContent    int

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
	readmeViewport  viewport.Model
	detailViewport  viewport.Model
	jobLogViewport  viewport.Model
	fileViewport    viewport.Model
	readmeReady     bool
	detailReady     bool
	jobLogReady     bool
	fileViewReady   bool

	// Job selection for pipelines
	selectedJobIdx int

	// Scroll offset for file list (keeps selected item visible)
	fileScrollOffset int

	// Job log popup
	showJobLogPopup bool

	// Branch selector popup
	showBranchPopup  bool
	selectedBranchIdx int
	currentBranch    string

	// Status message (for clipboard feedback etc)
	statusMsg     string
	statusMsgTime int

	// Error handling
	lastError     string
	lastErrorTime int
	retryCmd      tea.Cmd // Command to retry on 'r' key
}

// NewMainScreen creates a new main screen
func NewMainScreen() *MainScreen {
	// Priority: env vars > glab config > defaults
	token := os.Getenv("GITLAB_TOKEN")
	host := os.Getenv("GITLAB_HOST")
	groupPath := os.Getenv("GITLAB_GROUP")

	// Try to load glab config if env vars not set
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
		host = "gitlab.com"
	}
	if groupPath == "" {
		groupPath = "gitlab-org"
	}

	// Build the full URL if needed
	if host != "" && !strings.HasPrefix(host, "http") {
		host = "https://" + host
	}

	var client *gitlab.Client
	if token != "" {
		client = gitlab.NewClient(host, token)
	} else {
		client = gitlab.NewPublicClient()
	}

	return &MainScreen{
		client:       client,
		groupPath:    groupPath,
		focusedPanel: PanelProjects,
		contentTab:   TabFiles,
		keymap:       keymap.DefaultKeyMap(),
	}
}

// Init initializes the screen
func (m *MainScreen) Init() tea.Cmd {
	m.loading = true
	m.loadingMsg = "Loading groups..."
	cmd := m.loadGroups()
	m.retryCmd = cmd
	return cmd
}

func (m *MainScreen) loadGroups() tea.Cmd {
	return func() tea.Msg {
		groups, err := m.client.ListGroups()
		if err != nil {
			return errMsg{err: err}
		}
		return groupsLoadedMsg{groups: groups}
	}
}

func (m *MainScreen) loadProjects() tea.Cmd {
	return func() tea.Msg {
		var projects []gitlab.Project
		var err error

		if m.groupPath != "" {
			projects, err = m.client.ListGroupProjects(m.groupPath)
		} else {
			projects, err = m.client.ListProjects()
		}

		if err != nil {
			return errMsg{err: err}
		}
		return projectsLoadedMsg{projects: projects}
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
	if m.selectedProject == nil {
		return nil
	}
	projectID := fmt.Sprintf("%d", m.selectedProject.ID)

	return func() tea.Msg {
		entries, err := m.client.GetTree(projectID, branch, "")
		if err != nil {
			return errMsg{err: err}
		}

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
	if m.selectedProject == nil {
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
		return treeLoadedMsg{entries: entries, path: path}
	}
}

func (m *MainScreen) loadFile(filePath string) tea.Cmd {
	if m.selectedProject == nil {
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
	if m.selectedProject == nil {
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
	if m.selectedProject == nil {
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

func (m *MainScreen) loadBranches() tea.Cmd {
	if m.selectedProject == nil {
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
	if m.selectedProject == nil {
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
	if m.selectedProject == nil {
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
	if m.selectedProject == nil {
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
type projectsLoadedMsg struct{ projects []gitlab.Project }
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
type branchesLoadedMsg struct{ branches []gitlab.Branch }
type jobsLoadedMsg struct{ jobs []gitlab.Job }
type jobLogLoadedMsg struct{ log string }
type pipelineJobsLoadedMsg struct {
	pipelineID int
	jobs       []gitlab.Job
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
		m.lastError = "" // Clear any previous error
		// If no groups, load all projects directly
		if len(m.groups) == 0 {
			m.loading = true
			m.loadingMsg = "Loading projects..."
			cmd := m.loadProjects()
			m.retryCmd = cmd
			return m, cmd
		}
		return m, nil

	case projectsLoadedMsg:
		m.projects = msg.projects
		m.loading = false
		m.lastError = ""
		return m, nil

	case projectContentMsg:
		m.files = msg.entries
		m.readmeContent = msg.readme
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
		m.fileContent = msg.content
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
		return m, tea.Batch(cmds...)

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
		return m, nil

	case tea.KeyMsg:
		return m.handleKey(msg)
	}

	return m, nil
}

func (m *MainScreen) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Handle popups first
	if m.showJobLogPopup {
		return m.handleJobLogPopup(msg)
	}
	if m.showBranchPopup {
		return m.handleBranchPopup(msg)
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
		if len(m.branches) == 0 {
			m.loading = true
			m.loadingMsg = "Loading branches..."
			cmd := m.loadBranches()
			m.retryCmd = cmd
			return m, cmd
		}
		return m, nil
	}

	// Panel navigation with Shift+HJKL
	// Layout:
	// [1 Groups  ] [3 Content ] [5 Detail]
	// [2 Projects] [4 README  ]
	switch msg.String() {
	case "H", "shift+left":
		switch m.focusedPanel {
		case PanelContent, PanelReadme:
			m.focusedPanel = PanelProjects
		case PanelDetail:
			m.focusedPanel = PanelContent
		}
		return m, nil
	case "L", "shift+right":
		switch m.focusedPanel {
		case PanelGroups, PanelProjects:
			m.focusedPanel = PanelContent
		case PanelContent, PanelReadme:
			m.focusedPanel = PanelDetail
		}
		return m, nil
	case "K", "shift+up":
		switch m.focusedPanel {
		case PanelProjects:
			m.focusedPanel = PanelGroups
		case PanelReadme:
			m.focusedPanel = PanelContent
		}
		return m, nil
	case "J", "shift+down":
		switch m.focusedPanel {
		case PanelGroups:
			m.focusedPanel = PanelProjects
		case PanelContent:
			m.focusedPanel = PanelReadme
		}
		return m, nil
	case "1":
		m.focusedPanel = PanelGroups
		return m, nil
	case "2":
		m.focusedPanel = PanelProjects
		return m, nil
	case "3":
		m.focusedPanel = PanelContent
		return m, nil
	case "4":
		m.focusedPanel = PanelReadme
		return m, nil
	case "5":
		m.focusedPanel = PanelDetail
		return m, nil
	}

	switch m.focusedPanel {
	case PanelGroups:
		return m.handleGroupsNav(msg)
	case PanelProjects:
		return m.handleProjectsNav(msg)
	case PanelContent:
		return m.handleContentNav(msg)
	case PanelReadme:
		return m.handleReadmeNav(msg)
	case PanelDetail:
		return m.handleDetailNav(msg)
	}

	return m, nil
}

func (m *MainScreen) handleGroupsNav(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keymap.Down):
		if m.selectedGroupIdx < len(m.groups)-1 {
			m.selectedGroupIdx++
		}
	case key.Matches(msg, m.keymap.Up):
		if m.selectedGroupIdx > 0 {
			m.selectedGroupIdx--
		}
	case key.Matches(msg, m.keymap.Right), key.Matches(msg, m.keymap.Select):
		// Select group and load its projects
		if m.selectedGroupIdx < len(m.groups) {
			m.groupPath = m.groups[m.selectedGroupIdx].FullPath
			m.projects = nil
			m.selectedProjectIdx = 0
			m.loading = true
			m.loadingMsg = "Loading projects..."
			m.focusedPanel = PanelProjects
			cmd := m.loadProjects()
			m.retryCmd = cmd
			return m, cmd
		}
		m.focusedPanel = PanelProjects
	}
	return m, nil
}

func (m *MainScreen) handleProjectsNav(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keymap.Down):
		if m.selectedProjectIdx < len(m.projects)-1 {
			m.selectedProjectIdx++
		}
	case key.Matches(msg, m.keymap.Up):
		if m.selectedProjectIdx > 0 {
			m.selectedProjectIdx--
		}
	case key.Matches(msg, m.keymap.Left):
		m.focusedPanel = PanelGroups
	case key.Matches(msg, m.keymap.Right), key.Matches(msg, m.keymap.Select):
		// Select project and load its content
		if m.selectedProjectIdx < len(m.projects) {
			m.selectedProject = &m.projects[m.selectedProjectIdx]
			m.currentPath = nil
			m.currentBranch = ""
			m.files = nil
			m.mergeRequests = nil
			m.pipelines = nil
			m.branches = nil
			m.fileContent = ""
			m.readmeContent = ""
			m.contentTab = TabFiles
			m.loading = true
			m.loadingMsg = "Loading repository..."
			m.focusedPanel = PanelContent
			cmd := m.loadProjectContent()
			m.retryCmd = cmd
			return m, cmd
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
			m.loading = true
			m.loadingMsg = "Loading..."
			path := strings.Join(m.currentPath, "/")
			cmd := m.loadDirectory(path)
			m.retryCmd = cmd
			return m, cmd
		}
		// If at root, go back to projects
		m.focusedPanel = PanelProjects
		return m, nil
	}

	switch {
	case key.Matches(msg, m.keymap.Left):
		// h - switch to previous tab
		if m.contentTab > TabFiles {
			return m, m.switchTab(m.contentTab - 1)
		}
		// At first tab, go to projects panel
		m.focusedPanel = PanelProjects

	case key.Matches(msg, m.keymap.Right):
		// l - switch to next tab
		if m.contentTab < TabPipelines {
			return m, m.switchTab(m.contentTab + 1)
		}
		// At last tab, go to detail panel
		m.focusedPanel = PanelDetail

	case key.Matches(msg, m.keymap.Select):
		// Enter - drill into directory or view file
		if m.contentTab == TabFiles && m.selectedContent < len(m.files) {
			entry := m.files[m.selectedContent]
			if entry.Type == "tree" {
				m.currentPath = append(m.currentPath, entry.Name)
				m.loading = true
				m.loadingMsg = "Loading..."
				cmd := m.loadDirectory(entry.Path)
				m.retryCmd = cmd
				return m, cmd
			} else {
				m.loading = true
				m.loadingMsg = "Loading file..."
				cmd := m.loadFile(entry.Path)
				m.retryCmd = cmd
				return m, cmd
			}
		}
		// Load jobs for selected pipeline and show popup
		if m.contentTab == TabPipelines && m.selectedContent < len(m.pipelines) {
			pipeline := m.pipelines[m.selectedContent]
			m.jobs = nil
			m.jobLog = ""
			m.showJobLogPopup = true
			m.loading = true
			m.loadingMsg = "Loading jobs..."
			cmd := m.loadPipelineJobs(pipeline.ID)
			m.retryCmd = cmd
			return m, cmd
		}
		// For other tabs, focus detail panel
		m.focusedPanel = PanelDetail

	case key.Matches(msg, m.keymap.Down):
		// If viewing file, scroll down
		if m.viewingFile {
			m.fileViewport.LineDown(1)
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
			m.fileViewport.LineUp(1)
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
			m.fileViewport.HalfViewDown()
		case "ctrl+u":
			m.fileViewport.HalfViewUp()
		case "g":
			m.fileViewport.GotoTop()
		case "G":
			m.fileViewport.GotoBottom()
		}
	}

	return m, nil
}

func (m *MainScreen) adjustScrollOffset() {
	// Calculate visible area (rough estimate, accounting for headers)
	visibleLines := (m.height / 2) - 6 // half height minus headers/borders
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
	switch {
	case key.Matches(msg, m.keymap.Left):
		m.focusedPanel = PanelProjects
	case key.Matches(msg, m.keymap.Right):
		m.focusedPanel = PanelDetail
	case key.Matches(msg, m.keymap.Up):
		m.readmeViewport.LineUp(1)
	case key.Matches(msg, m.keymap.Down):
		m.readmeViewport.LineDown(1)
	}
	switch msg.String() {
	case "ctrl+d":
		m.readmeViewport.HalfViewDown()
	case "ctrl+u":
		m.readmeViewport.HalfViewUp()
	}
	return m, nil
}

func (m *MainScreen) handleDetailNav(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keymap.Left):
		m.focusedPanel = PanelContent
	case key.Matches(msg, m.keymap.Down):
		m.detailViewport.LineDown(1)
	case key.Matches(msg, m.keymap.Up):
		m.detailViewport.LineUp(1)
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

func (m *MainScreen) handleJobLogPopup(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "esc", "escape":
		m.showJobLogPopup = false
		m.jobs = nil
		m.jobLog = ""
		m.statusMsg = ""
		m.lastError = ""
		return m, nil
	case "j", "down":
		// Next job
		if m.selectedJobIdx < len(m.jobs)-1 {
			m.selectedJobIdx++
			m.jobLog = ""
			m.jobLogReady = false
			m.loading = true
			m.loadingMsg = "Loading job log..."
			m.statusMsg = ""
			cmd := m.loadJobLog(m.jobs[m.selectedJobIdx].ID)
			m.retryCmd = cmd
			return m, cmd
		}
	case "k", "up":
		// Previous job
		if m.selectedJobIdx > 0 {
			m.selectedJobIdx--
			m.jobLog = ""
			m.jobLogReady = false
			m.loading = true
			m.loadingMsg = "Loading job log..."
			m.statusMsg = ""
			cmd := m.loadJobLog(m.jobs[m.selectedJobIdx].ID)
			m.retryCmd = cmd
			return m, cmd
		}
	case "h", "left":
		m.jobLogViewport.LineUp(3)
	case "l", "right":
		m.jobLogViewport.LineDown(3)
	case "ctrl+d":
		m.jobLogViewport.HalfViewDown()
	case "ctrl+u":
		m.jobLogViewport.HalfViewUp()
	case "g":
		m.jobLogViewport.GotoTop()
	case "G":
		m.jobLogViewport.GotoBottom()
	case "y":
		// Copy job log to clipboard
		if m.jobLog != "" {
			cleanLog := stripANSI(m.jobLog)
			if err := copyToClipboard(cleanLog); err != nil {
				m.statusMsg = "Copy failed: " + err.Error()
			} else {
				m.statusMsg = "Copied to clipboard!"
			}
		}
	}
	return m, nil
}

func (m *MainScreen) switchTab(tab ContentTab) tea.Cmd {
	m.contentTab = tab
	m.selectedContent = 0
	m.fileContent = ""

	if m.selectedProject == nil {
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

	// Calculate dimensions
	contentHeight := m.height - 1
	leftWidth := m.width / 4
	rightWidth := m.width - leftWidth

	groupsHeight := contentHeight / 3
	projectsHeight := contentHeight - groupsHeight

	contentWidth := rightWidth * 2 / 3
	detailWidth := rightWidth - contentWidth

	// Render panels
	groupsPanel := m.renderGroupsPanel(leftWidth, groupsHeight)
	projectsPanel := m.renderProjectsPanel(leftWidth, projectsHeight)
	contentPanel := m.renderContentPanel(contentWidth, contentHeight)
	detailPanel := m.renderDetailPanel(detailWidth, contentHeight)

	// Combine left column
	leftColumn := lipgloss.JoinVertical(lipgloss.Left, groupsPanel, projectsPanel)

	// Combine right side
	rightSide := lipgloss.JoinHorizontal(lipgloss.Top, contentPanel, detailPanel)

	// Combine all
	main := lipgloss.JoinHorizontal(lipgloss.Top, leftColumn, rightSide)
	statusBar := m.renderStatusBar()

	// If popup is shown, render only the popup
	if m.showJobLogPopup {
		return m.renderJobLogPopup()
	}
	if m.showBranchPopup {
		return m.renderBranchPopup()
	}

	return main + "\n" + statusBar
}

func (m *MainScreen) renderGroupsPanel(width, height int) string {
	var content strings.Builder

	if m.loading && len(m.groups) == 0 {
		content.WriteString(m.loadingMsg)
	} else if len(m.groups) == 0 {
		content.WriteString(styles.DimmedText.Render("No groups"))
	} else {
		for i, g := range m.groups {
			line := g.Name
			if i == m.selectedGroupIdx {
				line = styles.SelectedItem.Render("> " + line)
			} else {
				line = styles.NormalItem.Render("  " + line)
			}
			content.WriteString(line + "\n")
		}
	}

	return components.SimpleBorderedPanel("Groups", content.String(), width, height, m.focusedPanel == PanelGroups)
}

func (m *MainScreen) renderProjectsPanel(width, height int) string {
	var content strings.Builder

	if m.loading && len(m.projects) == 0 {
		content.WriteString(m.loadingMsg)
	} else {
		for i, p := range m.projects {
			line := p.Name
			if i == m.selectedProjectIdx {
				line = styles.SelectedItem.Render("> " + line)
			} else {
				line = styles.NormalItem.Render("  " + line)
			}
			content.WriteString(line + "\n")
		}
	}

	return components.SimpleBorderedPanel("Projects", content.String(), width, height, m.focusedPanel == PanelProjects)
}

func (m *MainScreen) renderContentPanel(width, height int) string {
	// Split: top half for file list, bottom half for README (only in Files tab at root)
	showReadme := m.contentTab == TabFiles && len(m.currentPath) == 0 && m.readmeContent != ""

	listHeight := height
	readmeHeight := 0
	if showReadme {
		listHeight = height / 2
		readmeHeight = height - listHeight
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

	// Project header with branch
	if m.selectedProject != nil {
		projectHeader := styles.SelectedItem.Render(m.selectedProject.Name)
		if m.currentBranch != "" {
			projectHeader += styles.DimmedText.Render(" (" + m.currentBranch + ")")
		}
		content.WriteString(projectHeader + "\n")
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
		content.WriteString(styles.DimmedText.Render("/" + strings.Join(m.currentPath, "/")) + "\n")
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
					line := fmt.Sprintf("%s %s", icon, f.Name)
					if i == m.selectedContent {
						line = styles.SelectedItem.Render("> " + line)
					} else {
						line = "  " + line
					}
					content.WriteString(line + "\n")
				}
				// Show scroll indicator
				if len(m.files) > visibleLines {
					content.WriteString(styles.DimmedText.Render(fmt.Sprintf("\n[%d/%d]", m.selectedContent+1, len(m.files))))
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
				line := fmt.Sprintf("%s !%d %s", icon, mr.IID, mr.Title)
				if i == m.selectedContent {
					line = styles.SelectedItem.Render("> ") + line
				} else {
					line = "  " + line
				}
				content.WriteString(line + "\n")
			}
			if len(m.mergeRequests) == 0 {
				content.WriteString(styles.DimmedText.Render("No open merge requests"))
			} else if len(m.mergeRequests) > visibleLines {
				content.WriteString(styles.DimmedText.Render(fmt.Sprintf("\n[%d/%d]", m.selectedContent+1, len(m.mergeRequests))))
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
					// Group jobs by stage and get stage status
					stageOrder := []string{}
					stageStatus := make(map[string]string)
					for _, job := range jobs {
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
					// Build stage icons
					for _, stage := range stageOrder {
						status := stageStatus[stage]
						stageIcon := styles.PipelineIcon(status)
						stageStyle := styles.PipelineStatus(status)
						stagesStr += stageStyle.Render(stageIcon) + " "
					}
				}

				line := fmt.Sprintf("%s #%d %s %s", statusStyle.Render(icon), p.IID, p.Ref, stagesStr)
				if i == m.selectedContent {
					line = styles.SelectedItem.Render("> ") + line
				} else {
					line = "  " + line
				}
				content.WriteString(line + "\n")
			}
			if len(m.pipelines) == 0 {
				content.WriteString(styles.DimmedText.Render("No pipelines"))
			} else if len(m.pipelines) > visibleLines {
				content.WriteString(styles.DimmedText.Render(fmt.Sprintf("\n[%d/%d]", m.selectedContent+1, len(m.pipelines))))
			}
		}
	}

	title := contentTabNames[m.contentTab]
	return components.SimpleBorderedPanel(title, content.String(), width, height, m.focusedPanel == PanelContent)
}

func (m *MainScreen) renderReadmeSection(width, height int) string {
	// Update viewport dimensions and content
	innerWidth := width - 4  // account for borders
	innerHeight := height - 3 // account for borders and title

	if !m.readmeReady {
		m.readmeViewport = viewport.New(innerWidth, innerHeight)
		m.readmeViewport.SetContent(m.readmeContent)
		m.readmeReady = true
	} else {
		m.readmeViewport.Width = innerWidth
		m.readmeViewport.Height = innerHeight
	}

	// Build the panel manually with viewport content
	var content strings.Builder
	content.WriteString(m.readmeViewport.View())

	// Add scroll indicator
	if m.readmeViewport.TotalLineCount() > innerHeight {
		scrollPercent := int(m.readmeViewport.ScrollPercent() * 100)
		content.WriteString(styles.DimmedText.Render(fmt.Sprintf(" [%d%%]", scrollPercent)))
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

		// Format: icon stage/name
		line := fmt.Sprintf("%s %s", icon, job.Name)
		if job.Stage != "" && job.Stage != job.Name {
			line = fmt.Sprintf("%s %s:%s", icon, job.Stage, job.Name)
		}

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

	jobPanel := components.SimpleBorderedPanel(
		fmt.Sprintf("Jobs (%d)", len(m.jobs)),
		jobList.String(),
		jobListWidth,
		popupHeight,
		true,
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
			cleanLog := stripANSI(m.jobLog)
			wrappedLog := wrapText(cleanLog, logInnerWidth)
			m.jobLogViewport.SetContent(wrappedLog)
			m.jobLogReady = true
		}
		logContent.WriteString(m.jobLogViewport.View())
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

	logPanel := components.SimpleBorderedPanel(logTitle, logContent.String(), logWidth, popupHeight, false)

	// Join panels horizontally
	jobLines := strings.Split(jobPanel, "\n")
	logLines := strings.Split(logPanel, "\n")

	var combined strings.Builder
	maxLines := len(jobLines)
	if len(logLines) > maxLines {
		maxLines = len(logLines)
	}

	for i := 0; i < maxLines; i++ {
		jobLine := ""
		logLine := ""
		if i < len(jobLines) {
			jobLine = jobLines[i]
		}
		if i < len(logLines) {
			logLine = logLines[i]
		}
		combined.WriteString(jobLine + logLine + "\n")
	}

	// Status bar
	scrollInfo := ""
	if m.jobLogReady && m.jobLogViewport.TotalLineCount() > logInnerHeight {
		scrollInfo = fmt.Sprintf(" [%d%%]", int(m.jobLogViewport.ScrollPercent()*100))
	}

	statusContent := styles.StatusBarKey.Render("Esc") + styles.StatusBarDesc.Render(" close") + " │ " +
		styles.StatusBarKey.Render("j/k") + styles.StatusBarDesc.Render(" jobs") + " │ " +
		styles.StatusBarKey.Render("h/l") + styles.StatusBarDesc.Render(" scroll") + " │ " +
		styles.StatusBarKey.Render("y") + styles.StatusBarDesc.Render(" copy") +
		scrollInfo

	if m.statusMsg != "" {
		statusContent = styles.SelectedItem.Render(m.statusMsg) + " │ " + statusContent
	}

	statusBar := styles.StatusBar.Width(m.width).Render(statusContent)

	return combined.String() + statusBar
}

func (m *MainScreen) renderDetailPanel(width, height int) string {
	var content strings.Builder

	if m.selectedProject != nil {
		switch m.contentTab {
		case TabFiles:
			if m.viewingFile {
				// Show file info when viewing
				content.WriteString(styles.SelectedItem.Render(m.viewingFilePath) + "\n\n")
				content.WriteString(styles.DimmedText.Render("Lines: ") + fmt.Sprintf("%d", strings.Count(m.fileContent, "\n")+1) + "\n")
			} else if m.selectedContent < len(m.files) {
				f := m.files[m.selectedContent]
				content.WriteString(styles.SelectedItem.Render(f.Name) + "\n\n")
				fileType := "File"
				if f.Type == "tree" {
					fileType = "Directory"
				}
				content.WriteString(styles.DimmedText.Render("Type: ") + fileType + "\n")
				content.WriteString(styles.DimmedText.Render("Path: ") + f.Path + "\n")
				if f.Type == "blob" {
					content.WriteString("\n" + styles.DimmedText.Render("Press Enter to view"))
				}
			}
		case TabMRs:
			if m.selectedContent < len(m.mergeRequests) {
				mr := m.mergeRequests[m.selectedContent]
				content.WriteString(styles.SelectedItem.Render(mr.Title) + "\n\n")
				content.WriteString(styles.DimmedText.Render("Author: ") + mr.Author.Name + "\n")
				content.WriteString(styles.DimmedText.Render("Branch: ") + mr.SourceBranch + "\n")
				content.WriteString(styles.DimmedText.Render("Target: ") + mr.TargetBranch + "\n")
				content.WriteString(styles.DimmedText.Render("Status: ") + mr.MergeStatus + "\n")
				if mr.Description != "" {
					content.WriteString("\n" + mr.Description)
				}
			}
		case TabPipelines:
			if m.selectedContent < len(m.pipelines) {
				p := m.pipelines[m.selectedContent]
				statusStyle := styles.PipelineStatus(p.Status)
				content.WriteString(fmt.Sprintf("#%d %s\n\n", p.IID, p.Ref))
				content.WriteString(styles.DimmedText.Render("Status: ") + statusStyle.Render(p.Status) + "\n")
				content.WriteString(styles.DimmedText.Render("SHA: ") + p.SHA[:8] + "\n")
				content.WriteString(styles.DimmedText.Render("Source: ") + p.Source + "\n")
			}
		}
	} else {
		content.WriteString(styles.DimmedText.Render("Select a project"))
	}

	return components.SimpleBorderedPanel("Details", content.String(), width, height, m.focusedPanel == PanelDetail)
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

func (m *MainScreen) renderStatusBar() string {
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
		{PanelGroups, "1", "groups"},
		{PanelProjects, "2", "projects"},
		{PanelContent, "3", "files"},
		{PanelReadme, "4", "readme"},
		{PanelDetail, "5", "detail"},
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

	help := styles.StatusBarKey.Render("j/k") + styles.StatusBarDesc.Render(" up/down") + " │ " +
		styles.StatusBarKey.Render("h/l") + styles.StatusBarDesc.Render(" tabs") + " │ " +
		styles.StatusBarKey.Render("Enter") + styles.StatusBarDesc.Render(" select") + " │ " +
		styles.StatusBarKey.Render("Esc") + styles.StatusBarDesc.Render(" back") + " │ " +
		styles.StatusBarKey.Render("q") + styles.StatusBarDesc.Render(" quit")

	leftWidth := lipgloss.Width(left)
	rightWidth := lipgloss.Width(help)
	padding := m.width - leftWidth - rightWidth - 2
	if padding < 0 {
		padding = 0
	}

	return styles.StatusBar.Width(m.width).Render(left + strings.Repeat(" ", padding) + help)
}
