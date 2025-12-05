package styles

import "github.com/charmbracelet/lipgloss"

// Lazygit-inspired color palette
var (
	// Core colors
	ColorCyan    = lipgloss.Color("#00ffff")
	ColorGreen   = lipgloss.Color("#00ff00")
	ColorYellow  = lipgloss.Color("#ffff00")
	ColorRed     = lipgloss.Color("#ff0000")
	ColorMagenta = lipgloss.Color("#ff00ff")
	ColorBlue    = lipgloss.Color("#5f87ff")
	ColorWhite   = lipgloss.Color("#ffffff")
	ColorGray    = lipgloss.Color("#808080")
	ColorDimGray = lipgloss.Color("#4a4a4a")

	// Panel colors
	ColorActiveBorder   = ColorCyan
	ColorInactiveBorder = ColorDimGray
	ColorActiveTitle    = ColorCyan
	ColorInactiveTitle  = ColorGray

	// Status colors
	ColorSuccess = ColorGreen
	ColorRunning = ColorYellow
	ColorFailed  = ColorRed
	ColorPending = ColorGray

	// MR status
	ColorMROpen   = ColorGreen
	ColorMRMerged = ColorMagenta
	ColorMRClosed = ColorRed
	ColorMRDraft  = ColorGray
)

// Panel styles
var (
	// Active panel border
	ActivePanelBorder = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(ColorActiveBorder)

	// Inactive panel border
	InactivePanelBorder = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(ColorInactiveBorder)

	// Panel title (active)
	ActivePanelTitle = lipgloss.NewStyle().
				Bold(true).
				Foreground(ColorActiveBorder).
				Padding(0, 1)

	// Panel title (inactive)
	InactivePanelTitle = lipgloss.NewStyle().
				Foreground(ColorInactiveTitle).
				Padding(0, 1)

	// Selected item in list
	SelectedItem = lipgloss.NewStyle().
			Foreground(ColorCyan).
			Bold(true)

	// Normal item
	NormalItem = lipgloss.NewStyle().
			Foreground(ColorWhite)

	// Dimmed/secondary text
	DimmedText = lipgloss.NewStyle().
			Foreground(ColorGray)

	// Status bar at bottom
	StatusBar = lipgloss.NewStyle().
			Foreground(ColorGray).
			Background(lipgloss.Color("#1a1a1a")).
			Padding(0, 1)

	// Status bar keys
	StatusBarKey = lipgloss.NewStyle().
			Foreground(ColorCyan).
			Bold(true)

	// Status bar description
	StatusBarDesc = lipgloss.NewStyle().
			Foreground(ColorGray)
)

// Pipeline status styles
func PipelineStatus(status string) lipgloss.Style {
	switch status {
	case "success":
		return lipgloss.NewStyle().Foreground(ColorSuccess)
	case "running":
		return lipgloss.NewStyle().Foreground(ColorRunning)
	case "failed":
		return lipgloss.NewStyle().Foreground(ColorFailed)
	case "pending", "created":
		return lipgloss.NewStyle().Foreground(ColorPending)
	case "canceled":
		return lipgloss.NewStyle().Foreground(ColorGray)
	default:
		return lipgloss.NewStyle().Foreground(ColorWhite)
	}
}

// Pipeline status icon
func PipelineIcon(status string) string {
	switch status {
	case "success":
		return "✓"
	case "running":
		return "●"
	case "failed":
		return "✗"
	case "pending", "created":
		return "○"
	case "canceled":
		return "⊘"
	default:
		return "?"
	}
}

// MR status style
func MRStatus(state string, draft bool) lipgloss.Style {
	if draft {
		return lipgloss.NewStyle().Foreground(ColorMRDraft)
	}
	switch state {
	case "opened":
		return lipgloss.NewStyle().Foreground(ColorMROpen)
	case "merged":
		return lipgloss.NewStyle().Foreground(ColorMRMerged)
	case "closed":
		return lipgloss.NewStyle().Foreground(ColorMRClosed)
	default:
		return lipgloss.NewStyle().Foreground(ColorWhite)
	}
}
