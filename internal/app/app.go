package app

import (
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/espen/lazylab/internal/keymap"
	"github.com/espen/lazylab/internal/ui/styles"
	"github.com/espen/lazylab/internal/ui/views"
)

// App is the root bubbletea model
type App struct {
	stack    *ViewStack
	keymap   keymap.KeyMap
	width    int
	height   int
	showHelp bool
	err      error
}

// New creates a new App with an initial view
func New(initialView views.View) *App {
	stack := NewViewStack()
	stack.Push(initialView)

	return &App{
		stack:  stack,
		keymap: keymap.DefaultKeyMap(),
	}
}

// Init initializes the app
func (a *App) Init() tea.Cmd {
	if current := a.stack.Current(); current != nil {
		return current.Init()
	}
	return nil
}

// Update handles messages
func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height

	case tea.KeyMsg:
		// Global keybindings
		if key.Matches(msg, a.keymap.Quit) {
			if a.stack.Len() <= 1 {
				return a, tea.Quit
			}
			// Pop view instead of quitting
			a.stack.Pop()
			return a, nil
		}

		if key.Matches(msg, a.keymap.Help) {
			a.showHelp = !a.showHelp
			return a, nil
		}

		if key.Matches(msg, a.keymap.Back) {
			if a.stack.Len() > 1 {
				a.stack.Pop()
				return a, nil
			}
		}

	case views.PushViewMsg:
		a.stack.Push(msg.View)
		return a, msg.View.Init()

	case views.PopViewMsg:
		if a.stack.Len() > 1 {
			a.stack.Pop()
		}
		return a, nil

	case ErrMsg:
		a.err = msg.Err
		return a, nil
	}

	// Pass message to current view
	if current := a.stack.Current(); current != nil {
		updatedView, cmd := current.Update(msg)
		if v, ok := updatedView.(views.View); ok {
			a.stack.Pop()
			a.stack.Push(v)
		}
		return a, cmd
	}

	return a, nil
}

// View renders the app
func (a *App) View() string {
	if a.width == 0 {
		return "Loading..."
	}

	var content string
	if current := a.stack.Current(); current != nil {
		content = current.View()
	}

	// Build status bar with breadcrumbs
	breadcrumbs := a.stack.Breadcrumbs()
	breadcrumbStr := strings.Join(breadcrumbs, " > ")
	statusBar := styles.StatusBar.Render(breadcrumbStr + " | ? help | q quit")

	// Show error if present
	if a.err != nil {
		errStyle := lipgloss.NewStyle().Foreground(styles.ColorRed)
		content = errStyle.Render("Error: " + a.err.Error())
	}

	// Show help overlay if toggled
	if a.showHelp {
		content = a.renderHelp()
	}

	// Layout: content + status bar
	return lipgloss.JoinVertical(
		lipgloss.Left,
		lipgloss.NewStyle().Height(a.height-1).Render(content),
		statusBar,
	)
}

func (a *App) renderHelp() string {
	help := a.keymap.FullHelp()
	var lines []string

	lines = append(lines, styles.ActivePanelTitle.Render("Keyboard Shortcuts"))
	lines = append(lines, "")

	for _, column := range help {
		for _, binding := range column {
			helpStr := binding.Help()
			line := styles.NormalItem.Render(helpStr.Key) + "  " + styles.DimmedText.Render(helpStr.Desc)
			lines = append(lines, line)
		}
		lines = append(lines, "")
	}

	lines = append(lines, styles.DimmedText.Render("Press ? to close"))

	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}
