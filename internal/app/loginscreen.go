package app

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/espen/lazylab/internal/config"
	"github.com/espen/lazylab/internal/ui/styles"
)

// LoginField identifies which input field is focused
type LoginField int

const (
	FieldHost LoginField = iota
	FieldToken
)

// LoginScreen handles the login/setup flow
type LoginScreen struct {
	hostInput  textinput.Model
	tokenInput textinput.Model
	focused    LoginField
	width      int
	height     int
	errMsg     string
}

// NewLoginScreen creates a new login screen
func NewLoginScreen() *LoginScreen {
	hostInput := textinput.New()
	hostInput.Placeholder = "gitlab.com"
	hostInput.SetValue(config.DefaultHost)
	hostInput.CharLimit = 100
	hostInput.Width = 40

	tokenInput := textinput.New()
	tokenInput.Placeholder = "glpat-xxxxxxxxxxxxxxxxxxxx"
	tokenInput.CharLimit = 100
	tokenInput.Width = 40
	tokenInput.EchoMode = textinput.EchoPassword
	tokenInput.EchoCharacter = '•'
	tokenInput.Focus()

	return &LoginScreen{
		hostInput:  hostInput,
		tokenInput: tokenInput,
		focused:    FieldToken,
	}
}

// Init initializes the login screen
func (m *LoginScreen) Init() tea.Cmd {
	return textinput.Blink
}

// loginSuccessMsg signals successful login
type loginSuccessMsg struct{}

// Update handles messages
func (m *LoginScreen) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			return m, tea.Quit

		case "tab", "shift+tab", "up", "down":
			// Toggle between fields
			if m.focused == FieldToken {
				m.focused = FieldHost
				m.tokenInput.Blur()
				m.hostInput.Focus()
			} else {
				m.focused = FieldToken
				m.hostInput.Blur()
				m.tokenInput.Focus()
			}
			return m, nil

		case "enter":
			return m.handleSubmit()
		}
	}

	// Update the focused input
	var cmd tea.Cmd
	if m.focused == FieldToken {
		m.tokenInput, cmd = m.tokenInput.Update(msg)
	} else {
		m.hostInput, cmd = m.hostInput.Update(msg)
	}

	return m, cmd
}

func (m *LoginScreen) handleSubmit() (tea.Model, tea.Cmd) {
	token := strings.TrimSpace(m.tokenInput.Value())
	host := strings.TrimSpace(m.hostInput.Value())

	if token == "" {
		m.errMsg = "Token is required"
		return m, nil
	}

	if host == "" {
		host = config.DefaultHost
	}

	// Save to config
	cfg := &config.LazyLabConfig{
		DefaultHost: host,
	}
	cfg.SetHostToken(host, token)

	if err := config.SaveLazyLabConfig(cfg); err != nil {
		m.errMsg = fmt.Sprintf("Failed to save config: %v", err)
		return m, nil
	}

	// Signal success - the main app will handle the transition
	return m, func() tea.Msg { return loginSuccessMsg{} }
}

// View renders the login screen
func (m *LoginScreen) View() string {
	if m.width == 0 || m.height == 0 {
		return "Loading..."
	}

	var b strings.Builder

	// Title
	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("212")).
		Render("LazyLab Setup")

	subtitle := styles.DimmedText.Render("Configure your GitLab connection")

	// Form
	hostLabel := "Host:"
	tokenLabel := "Token:"

	if m.focused == FieldHost {
		hostLabel = styles.SelectedItem.Render("> " + hostLabel)
	} else {
		hostLabel = "  " + hostLabel
	}

	if m.focused == FieldToken {
		tokenLabel = styles.SelectedItem.Render("> " + tokenLabel)
	} else {
		tokenLabel = "  " + tokenLabel
	}

	form := fmt.Sprintf(
		"%s\n%s\n\n%s\n%s\n\n%s\n%s",
		tokenLabel,
		m.tokenInput.View(),
		hostLabel,
		m.hostInput.View(),
		"",
		styles.DimmedText.Render("Tab: switch field | Enter: save | Esc: quit"),
	)

	// Error message
	errView := ""
	if m.errMsg != "" {
		errStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
		errView = "\n\n" + errStyle.Render("Error: "+m.errMsg)
	}

	// Help text
	help := styles.DimmedText.Render("\nToken: GitLab → Settings → Access Tokens (needs read_api scope)")

	content := fmt.Sprintf("%s\n%s\n\n%s%s%s", title, subtitle, form, errView, help)

	// Center the content
	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("62")).
		Padding(1, 2).
		Width(60)

	box := boxStyle.Render(content)

	// Center vertically and horizontally
	boxHeight := lipgloss.Height(box)
	boxWidth := lipgloss.Width(box)

	topPadding := (m.height - boxHeight) / 2
	leftPadding := (m.width - boxWidth) / 2

	if topPadding < 0 {
		topPadding = 0
	}
	if leftPadding < 0 {
		leftPadding = 0
	}

	for i := 0; i < topPadding; i++ {
		b.WriteString("\n")
	}

	for _, line := range strings.Split(box, "\n") {
		b.WriteString(strings.Repeat(" ", leftPadding))
		b.WriteString(line)
		b.WriteString("\n")
	}

	return b.String()
}
