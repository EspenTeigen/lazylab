package views

import (
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

// View represents a navigable view in the TUI
type View interface {
	tea.Model

	// Title returns the view title for breadcrumbs/status bar
	Title() string

	// ShortHelp returns context-specific help bindings
	ShortHelp() []key.Binding
}

// PushViewMsg signals the app to push a new view onto the stack
type PushViewMsg struct {
	View View
}

// PopViewMsg signals the app to pop the current view
type PopViewMsg struct{}

// Push creates a message to push a view
func Push(v View) tea.Cmd {
	return func() tea.Msg {
		return PushViewMsg{View: v}
	}
}

// Pop creates a message to pop the current view
func Pop() tea.Cmd {
	return func() tea.Msg {
		return PopViewMsg{}
	}
}
