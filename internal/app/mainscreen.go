package app

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/espen/lazylab/internal/gitlab"
	"github.com/espen/lazylab/internal/keymap"
	"github.com/espen/lazylab/internal/ui/components"
	"github.com/espen/lazylab/internal/ui/styles"
)

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
	TabBranches
	TabCount
)

var contentTabNames = []string{"Files", "MRs", "Pipelines", "Branches"}

// MainScreen is the lazygit-style multi-panel interface
type MainScreen struct {
	// GitLab client
	client *gitlab.Client

	// Current group
	groupPath string

	// Data
	projects      []gitlab.Project
	files         []gitlab.TreeEntry
	mergeRequests []gitlab.MergeRequest
	pipelines     []gitlab.Pipeline
	branches      []gitlab.Branch

	// Selected project
	selectedProject *gitlab.Project

	// File browser state
	currentPath   []string
	fileContent   string
	readmeContent string

	// Selection indices
	selectedGroup   int
	selectedProjectIdx int
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
	readmeViewport  viewport.Model
	detailViewport  viewport.Model
	readmeReady     bool
	detailReady     bool

	// Scroll offset for file list (keeps selected item visible)
	fileScrollOffset int
}

// NewMainScreen creates a new main screen
func NewMainScreen() *MainScreen {
	// Check for GITLAB_TOKEN env var
	token := os.Getenv("GITLAB_TOKEN")
	host := os.Getenv("GITLAB_HOST")
	if host == "" {
		host = "https://gitlab.com"
	}

	var client *gitlab.Client
	if token != "" {
		client = gitlab.NewClient(host, token)
	} else {
		client = gitlab.NewPublicClient()
	}

	// Default to gitlab-org for testing (has MRs, pipelines, etc.)
	groupPath := os.Getenv("GITLAB_GROUP")
	if groupPath == "" {
		groupPath = "gitlab-org"
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
	m.loadingMsg = "Loading projects..."
	return m.loadProjects()
}

func (m *MainScreen) loadProjects() tea.Cmd {
	return func() tea.Msg {
		projects, err := m.client.ListGroupProjects(m.groupPath)
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
	projectID := fmt.Sprintf("%d", m.selectedProject.ID)
	ref := m.selectedProject.DefaultBranch
	if ref == "" {
		ref = "main"
	}

	return func() tea.Msg {
		entries, err := m.client.GetTree(projectID, ref, "")
		if err != nil {
			return errMsg{err: err}
		}

		// Try to load README
		var readme string
		for _, e := range entries {
			lower := strings.ToLower(e.Name)
			if strings.HasPrefix(lower, "readme") {
				content, err := m.client.GetFileContent(projectID, e.Path, ref)
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
	ref := m.selectedProject.DefaultBranch
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
	ref := m.selectedProject.DefaultBranch
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

// Messages
type errMsg struct{ err error }
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

// Update handles messages
func (m *MainScreen) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case errMsg:
		m.loading = false
		m.errMsg = msg.err.Error()
		return m, nil

	case projectsLoadedMsg:
		m.projects = msg.projects
		m.loading = false
		return m, nil

	case projectContentMsg:
		m.files = msg.entries
		m.readmeContent = msg.readme
		m.fileContent = ""
		m.selectedContent = 0
		m.fileScrollOffset = 0
		m.readmeReady = false // Reset to reinitialize viewport with new content
		m.loading = false
		return m, nil

	case treeLoadedMsg:
		m.files = msg.entries
		m.selectedContent = 0
		m.fileScrollOffset = 0
		m.fileContent = ""
		m.loading = false
		return m, nil

	case fileContentMsg:
		m.fileContent = msg.content
		m.detailReady = false // Reset to reinitialize viewport with new content
		m.loading = false
		return m, nil

	case mrsLoadedMsg:
		m.mergeRequests = msg.mrs
		m.selectedContent = 0
		m.fileScrollOffset = 0
		m.loading = false
		return m, nil

	case pipelinesLoadedMsg:
		m.pipelines = msg.pipelines
		m.selectedContent = 0
		m.fileScrollOffset = 0
		m.loading = false
		return m, nil

	case branchesLoadedMsg:
		m.branches = msg.branches
		m.selectedContent = 0
		m.fileScrollOffset = 0
		m.loading = false
		return m, nil

	case tea.KeyMsg:
		return m.handleKey(msg)
	}

	return m, nil
}

func (m *MainScreen) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if key.Matches(msg, m.keymap.Quit) {
		return m, tea.Quit
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
		if m.selectedGroup < 0 {
			m.selectedGroup++
		}
	case key.Matches(msg, m.keymap.Up):
		if m.selectedGroup > 0 {
			m.selectedGroup--
		}
	case key.Matches(msg, m.keymap.Right), key.Matches(msg, m.keymap.Select):
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
			return m, m.loadProjectContent()
		}
	}
	return m, nil
}

func (m *MainScreen) handleContentNav(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Handle escape for going back in directory
	if msg.String() == "esc" || msg.String() == "escape" {
		if m.contentTab == TabFiles && len(m.currentPath) > 0 {
			m.currentPath = m.currentPath[:len(m.currentPath)-1]
			m.loading = true
			m.loadingMsg = "Loading..."
			path := strings.Join(m.currentPath, "/")
			return m, m.loadDirectory(path)
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
		if m.contentTab < TabBranches {
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
				return m, m.loadDirectory(entry.Path)
			} else {
				m.loading = true
				m.loadingMsg = "Loading file..."
				return m, m.loadFile(entry.Path)
			}
		}
		// For other tabs, focus detail panel
		m.focusedPanel = PanelDetail

	case key.Matches(msg, m.keymap.Down):
		maxItems := m.getContentCount()
		if m.selectedContent < maxItems-1 {
			m.selectedContent++
			if m.contentTab == TabFiles {
				m.fileContent = ""
			}
			m.adjustScrollOffset()
		}
	case key.Matches(msg, m.keymap.Up):
		if m.selectedContent > 0 {
			m.selectedContent--
			if m.contentTab == TabFiles {
				m.fileContent = ""
			}
			m.adjustScrollOffset()
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
	// Also support vim-style half-page scrolling
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
			return m.loadProjectContent()
		}
	case TabMRs:
		if len(m.mergeRequests) == 0 {
			m.loading = true
			m.loadingMsg = "Loading merge requests..."
			return m.loadMRs()
		}
	case TabPipelines:
		if len(m.pipelines) == 0 {
			m.loading = true
			m.loadingMsg = "Loading pipelines..."
			return m.loadPipelines()
		}
	case TabBranches:
		if len(m.branches) == 0 {
			m.loading = true
			m.loadingMsg = "Loading branches..."
			return m.loadBranches()
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
	case TabBranches:
		return len(m.branches)
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

	return main + "\n" + statusBar
}

func (m *MainScreen) renderGroupsPanel(width, height int) string {
	var content strings.Builder

	// Just show the current group for now
	line := m.groupPath
	if m.selectedGroup == 0 {
		line = styles.SelectedItem.Render("> " + line)
	} else {
		line = styles.NormalItem.Render("  " + line)
	}
	content.WriteString(line + "\n")

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

	// Project header
	if m.selectedProject != nil {
		content.WriteString(styles.SelectedItem.Render(m.selectedProject.Name) + "\n")
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
				line := fmt.Sprintf("%s #%d %s", statusStyle.Render(icon), p.IID, p.Ref)
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
		case TabBranches:
			endIdx := m.fileScrollOffset + visibleLines
			if endIdx > len(m.branches) {
				endIdx = len(m.branches)
			}
			for i := m.fileScrollOffset; i < endIdx; i++ {
				b := m.branches[i]
				icon := "○"
				if b.Default {
					icon = "●"
				}
				line := fmt.Sprintf("%s %s", icon, b.Name)
				if i == m.selectedContent {
					line = styles.SelectedItem.Render("> ") + line
				} else {
					line = "  " + line
				}
				content.WriteString(line + "\n")
			}
			if len(m.branches) == 0 {
				content.WriteString(styles.DimmedText.Render("No branches"))
			} else if len(m.branches) > visibleLines {
				content.WriteString(styles.DimmedText.Render(fmt.Sprintf("\n[%d/%d]", m.selectedContent+1, len(m.branches))))
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

func (m *MainScreen) renderDetailPanel(width, height int) string {
	var content strings.Builder
	innerWidth := width - 4
	innerHeight := height - 3

	if m.fileContent != "" {
		// Use viewport for file content
		if !m.detailReady {
			m.detailViewport = viewport.New(innerWidth, innerHeight)
			m.detailViewport.SetContent(m.fileContent)
			m.detailReady = true
		} else {
			m.detailViewport.Width = innerWidth
			m.detailViewport.Height = innerHeight
		}
		content.WriteString(m.detailViewport.View())
		if m.detailViewport.TotalLineCount() > innerHeight {
			scrollPercent := int(m.detailViewport.ScrollPercent() * 100)
			content.WriteString(styles.DimmedText.Render(fmt.Sprintf(" [%d%%]", scrollPercent)))
		}
	} else if m.selectedProject != nil {
		switch m.contentTab {
		case TabFiles:
			if m.selectedContent < len(m.files) {
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
		case TabBranches:
			if m.selectedContent < len(m.branches) {
				b := m.branches[m.selectedContent]
				content.WriteString(styles.SelectedItem.Render(b.Name) + "\n\n")
				content.WriteString(styles.DimmedText.Render("Commit: ") + b.Commit.ShortID + "\n")
				content.WriteString(styles.DimmedText.Render("Author: ") + b.Commit.AuthorName + "\n")
				content.WriteString(styles.DimmedText.Render("Message: ") + b.Commit.Title + "\n")
				if b.Protected {
					content.WriteString("\n" + styles.PipelineStatus("running").Render("🔒 Protected"))
				}
			}
		}
	} else {
		content.WriteString(styles.DimmedText.Render("Select a project"))
	}

	return components.SimpleBorderedPanel("Details", content.String(), width, height, m.focusedPanel == PanelDetail)
}

func (m *MainScreen) renderStatusBar() string {
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
