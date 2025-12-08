package app

import (
	tea "github.com/charmbracelet/bubbletea"
)

// Launcher handles the login flow and transitions to main screen
type Launcher struct {
	loginScreen *LoginScreen
	mainScreen  *MainScreen
	loggedIn    bool
	width       int
	height      int
}

// NewLauncher creates a new Launcher starting with the login screen
func NewLauncher() *Launcher {
	return &Launcher{
		loginScreen: NewLoginScreen(),
		loggedIn:    false,
	}
}

// Init initializes the launcher
func (l *Launcher) Init() tea.Cmd {
	return l.loginScreen.Init()
}

// Update handles messages
func (l *Launcher) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Store window size
	if wsm, ok := msg.(tea.WindowSizeMsg); ok {
		l.width = wsm.Width
		l.height = wsm.Height
	}

	// If logged in, forward to main screen
	if l.loggedIn {
		model, cmd := l.mainScreen.Update(msg)
		l.mainScreen = model.(*MainScreen)
		return l, cmd
	}

	// Check for successful login
	if _, ok := msg.(loginSuccessMsg); ok {
		l.loggedIn = true
		l.mainScreen = NewMainScreen()
		l.loginScreen = nil

		// Send window size and init to main screen
		return l, tea.Batch(
			func() tea.Msg { return tea.WindowSizeMsg{Width: l.width, Height: l.height} },
			l.mainScreen.Init(),
		)
	}

	// Forward to login screen
	model, cmd := l.loginScreen.Update(msg)
	l.loginScreen = model.(*LoginScreen)
	return l, cmd
}

// View renders the current screen
func (l *Launcher) View() string {
	if l.loggedIn {
		return l.mainScreen.View()
	}
	return l.loginScreen.View()
}
