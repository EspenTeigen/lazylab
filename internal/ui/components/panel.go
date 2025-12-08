package components

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/EspenTeigen/lazylab/internal/ui/styles"
)

// truncateToWidth truncates a string to fit within maxWidth visual characters
func truncateToWidth(s string, maxWidth int) string {
	if maxWidth <= 0 {
		return ""
	}
	width := lipgloss.Width(s)
	if width <= maxWidth {
		return s
	}
	// Truncate rune by rune until we fit
	runes := []rune(s)
	for len(runes) > 0 {
		runes = runes[:len(runes)-1]
		if lipgloss.Width(string(runes)) <= maxWidth-1 {
			return string(runes) + "…"
		}
	}
	return ""
}

// Panel represents a bordered panel with a title
type Panel struct {
	Title   string
	Content string
	Focused bool
	Width   int
	Height  int
}

// NewPanel creates a new panel
func NewPanel(title string) *Panel {
	return &Panel{
		Title: title,
	}
}

// SetSize updates the panel dimensions
func (p *Panel) SetSize(width, height int) {
	p.Width = width
	p.Height = height
}

// SetContent sets the panel content
func (p *Panel) SetContent(content string) {
	p.Content = content
}

// SetFocused sets the focus state
func (p *Panel) SetFocused(focused bool) {
	p.Focused = focused
}

// View renders the panel
func (p *Panel) View() string {
	if p.Width <= 0 || p.Height <= 0 {
		return ""
	}

	// Choose border style based on focus
	var borderStyle lipgloss.Style
	var titleStyle lipgloss.Style

	if p.Focused {
		borderStyle = styles.ActivePanelBorder
		titleStyle = styles.ActivePanelTitle
	} else {
		borderStyle = styles.InactivePanelBorder
		titleStyle = styles.InactivePanelTitle
	}

	// Calculate inner dimensions (accounting for border)
	innerWidth := p.Width - 2
	innerHeight := p.Height - 2

	if innerWidth <= 0 || innerHeight <= 0 {
		return ""
	}

	// Truncate and pad content to fit
	content := p.formatContent(innerWidth, innerHeight)

	// Create the bordered box
	box := borderStyle.
		Width(innerWidth).
		Height(innerHeight).
		Render(content)

	// Replace top border with title
	lines := strings.Split(box, "\n")
	if len(lines) > 0 && len(p.Title) > 0 {
		// Find where to insert the title (after the corner)
		topBorder := lines[0]
		if len(topBorder) > 4 {
			// Insert title after first corner character
			titleStr := " " + p.Title + " "
			if p.Focused {
				titleStr = titleStyle.Render(titleStr)
			} else {
				titleStr = titleStyle.Render(titleStr)
			}

			// Calculate position (after "╭─")
			runes := []rune(topBorder)
			if len(runes) > 3 {
				prefix := string(runes[0:2])               // "╭─"
				suffix := string(runes[2+len(p.Title)+2:]) // rest of border

				// Rebuild with colored title
				borderColor := styles.ColorActiveBorder
				if !p.Focused {
					borderColor = styles.ColorInactiveBorder
				}

				lines[0] = lipgloss.NewStyle().Foreground(borderColor).Render(prefix) +
					titleStr +
					lipgloss.NewStyle().Foreground(borderColor).Render(suffix)
			}
		}
		box = strings.Join(lines, "\n")
	}

	return box
}

// formatContent formats content to fit the panel dimensions
func (p *Panel) formatContent(width, height int) string {
	if p.Content == "" {
		return ""
	}

	lines := strings.Split(p.Content, "\n")
	result := make([]string, 0, height)

	for i, line := range lines {
		if i >= height {
			break
		}
		// Truncate line if too long
		if len(line) > width {
			line = line[:width-1] + "…"
		}
		result = append(result, line)
	}

	// Pad with empty lines if needed
	for len(result) < height {
		result = append(result, "")
	}

	return strings.Join(result, "\n")
}

// SimpleBorderedPanel creates a bordered panel with exact dimensions
func SimpleBorderedPanel(title string, content string, width, height int, focused bool) string {
	borderColor := styles.ColorInactiveBorder
	if focused {
		borderColor = styles.ColorActiveBorder
	}

	innerWidth := width - 2
	innerHeight := height - 2

	if innerWidth <= 0 || innerHeight <= 0 {
		return ""
	}

	borderStyle := lipgloss.NewStyle().Foreground(borderColor)

	// Prepare content lines - truncate/pad to fit exactly
	contentLines := strings.Split(content, "\n")
	paddedLines := make([]string, innerHeight)
	for i := 0; i < innerHeight; i++ {
		line := ""
		if i < len(contentLines) {
			line = contentLines[i]
		}
		// Truncate if too long - use rune-based truncation for accuracy
		line = truncateToWidth(line, innerWidth)
		// Pad to exact width
		padding := innerWidth - lipgloss.Width(line)
		if padding > 0 {
			line = line + strings.Repeat(" ", padding)
		}
		paddedLines[i] = line
	}

	// Build panel
	var result strings.Builder

	// Top border with title
	titleText := " " + title + " "
	titleLen := lipgloss.Width(titleText)
	leftLen := (innerWidth - titleLen) / 2
	rightLen := innerWidth - titleLen - leftLen
	if leftLen < 0 {
		leftLen = 0
	}
	if rightLen < 0 {
		rightLen = 0
	}

	result.WriteString(borderStyle.Render("╭" + strings.Repeat("─", leftLen)))
	result.WriteString(titleText)
	result.WriteString(borderStyle.Render(strings.Repeat("─", rightLen) + "╮"))
	result.WriteString("\n")

	// Content lines
	for _, line := range paddedLines {
		result.WriteString(borderStyle.Render("│"))
		result.WriteString(line)
		result.WriteString(borderStyle.Render("│"))
		result.WriteString("\n")
	}

	// Bottom border
	result.WriteString(borderStyle.Render("╰" + strings.Repeat("─", innerWidth) + "╯"))

	return result.String()
}
