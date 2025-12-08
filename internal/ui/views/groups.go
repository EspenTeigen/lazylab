package views

import (
	"fmt"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/EspenTeigen/lazylab/internal/gitlab"
	"github.com/EspenTeigen/lazylab/internal/keymap"
	"github.com/EspenTeigen/lazylab/internal/ui/components"
)

// GroupItem wraps gitlab.Group to implement list.Item
type GroupItem struct {
	gitlab.Group
}

func (g GroupItem) Title() string       { return g.Name }
func (g GroupItem) Description() string { return g.FullPath }
func (g GroupItem) FilterValue() string { return g.Name }

// GroupsMsg carries loaded groups
type GroupsMsg struct {
	Groups []gitlab.Group
}

// Groups is the group browser view
type Groups struct {
	list    components.List
	keymap  keymap.KeyMap
	loading bool
	width   int
	height  int
}

// NewGroups creates a new groups view
func NewGroups() *Groups {
	l := components.NewList("Groups", []list.Item{}, 80, 24)

	return &Groups{
		list:    l,
		keymap:  keymap.DefaultKeyMap(),
		loading: true,
	}
}

// Title returns the view title
func (g *Groups) Title() string {
	return "Groups"
}

// ShortHelp returns context-specific help
func (g *Groups) ShortHelp() []key.Binding {
	return g.keymap.ShortHelp()
}

// Init loads the initial data
func (g *Groups) Init() tea.Cmd {
	return g.fetchGroups()
}

// fetchGroups returns a command that fetches groups
func (g *Groups) fetchGroups() tea.Cmd {
	return func() tea.Msg {
		// Use test data for now - will be replaced with real API
		groups, err := gitlab.LoadTestGroups()
		if err != nil {
			return ErrMsg{Err: err}
		}
		return GroupsMsg{Groups: groups}
	}
}

// Update handles messages
func (g *Groups) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		g.width = msg.Width
		g.height = msg.Height
		g.list.SetSize(msg.Width, msg.Height-4)
		return g, nil

	case GroupsMsg:
		g.loading = false
		items := make([]list.Item, len(msg.Groups))
		for i, group := range msg.Groups {
			items[i] = GroupItem{Group: group}
		}
		cmd := g.list.SetItems(items)
		return g, cmd

	case ErrMsg:
		g.loading = false
		return g, nil

	case tea.KeyMsg:
		if key.Matches(msg, g.keymap.Select) {
			if selected := g.list.SelectedItem(); selected != nil {
				if group, ok := selected.(GroupItem); ok {
					return g, Push(NewProjects(group.ID, group.Name))
				}
			}
		}

		if key.Matches(msg, g.keymap.Refresh) {
			g.loading = true
			return g, g.fetchGroups()
		}
	}

	var cmd tea.Cmd
	g.list, cmd = g.list.Update(msg)
	return g, cmd
}

// View renders the groups list
func (g *Groups) View() string {
	if g.loading {
		return "Loading groups..."
	}
	return g.list.View()
}

// ErrMsg represents an error
type ErrMsg struct {
	Err error
}

// ProjectItem wraps gitlab.Project to implement list.Item
type ProjectItem struct {
	gitlab.Project
}

func (p ProjectItem) Title() string       { return p.Name }
func (p ProjectItem) Description() string { return p.Project.Description }
func (p ProjectItem) FilterValue() string { return p.Name }

// ProjectsMsg carries loaded projects
type ProjectsMsg struct {
	Projects []gitlab.Project
}

// Projects is the projects view for a group
type Projects struct {
	groupID   int
	groupName string
	list      components.List
	keymap    keymap.KeyMap
	loading   bool
}

// NewProjects creates a projects view for a group
func NewProjects(groupID int, groupName string) *Projects {
	l := components.NewList(fmt.Sprintf("Projects in %s", groupName), []list.Item{}, 80, 24)

	return &Projects{
		groupID:   groupID,
		groupName: groupName,
		list:      l,
		keymap:    keymap.DefaultKeyMap(),
		loading:   true,
	}
}

func (p *Projects) Title() string            { return p.groupName }
func (p *Projects) ShortHelp() []key.Binding { return p.keymap.ShortHelp() }

func (p *Projects) Init() tea.Cmd {
	return p.fetchProjects()
}

func (p *Projects) fetchProjects() tea.Cmd {
	return func() tea.Msg {
		// Use test data for now - will be replaced with real API
		projects, err := gitlab.LoadTestProjects()
		if err != nil {
			return ErrMsg{Err: err}
		}
		return ProjectsMsg{Projects: projects}
	}
}

func (p *Projects) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		p.list.SetSize(msg.Width, msg.Height-4)
		return p, nil

	case ProjectsMsg:
		p.loading = false
		items := make([]list.Item, len(msg.Projects))
		for i, proj := range msg.Projects {
			items[i] = ProjectItem{Project: proj}
		}
		cmd := p.list.SetItems(items)
		return p, cmd

	case ErrMsg:
		p.loading = false
		return p, nil

	case tea.KeyMsg:
		if key.Matches(msg, p.keymap.Select) {
			if selected := p.list.SelectedItem(); selected != nil {
				if proj, ok := selected.(ProjectItem); ok {
					// TODO: Push project detail view
					_ = proj
				}
			}
		}

		if key.Matches(msg, p.keymap.Refresh) {
			p.loading = true
			return p, p.fetchProjects()
		}
	}

	var cmd tea.Cmd
	p.list, cmd = p.list.Update(msg)
	return p, cmd
}

func (p *Projects) View() string {
	if p.loading {
		return "Loading projects..."
	}
	return p.list.View()
}
