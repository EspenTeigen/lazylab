package components

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/espen/lazylab/internal/ui/styles"
)

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
				prefix := string(runes[0:2])         // "╭─"
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

// SimpleBorderedPanel creates a simple bordered panel without all the complexity
func SimpleBorderedPanel(title string, content string, width, height int, focused bool) string {
	borderColor := styles.ColorInactiveBorder
	titleColor := styles.ColorInactiveTitle
	if focused {
		borderColor = styles.ColorActiveBorder
		titleColor = styles.ColorActiveTitle
	}

	innerWidth := width - 2
	innerHeight := height - 2

	if innerWidth <= 0 || innerHeight <= 0 {
		return ""
	}

	// Format content
	contentLines := strings.Split(content, "\n")
	formattedLines := make([]string, innerHeight)
	for i := 0; i < innerHeight; i++ {
		if i < len(contentLines) {
			line := contentLines[i]
			// Truncate if needed
			lineRunes := []rune(line)
			if len(lineRunes) > innerWidth {
				line = string(lineRunes[:innerWidth-1]) + "…"
			}
			formattedLines[i] = line
		}
	}

	// Build the box manually for better control
	borderStyle := lipgloss.NewStyle().Foreground(borderColor)
	titleStyle := lipgloss.NewStyle().Foreground(titleColor).Bold(true)

	// Top border with title
	titleText := " " + title + " "
	topBorderLen := innerWidth - len(titleText)
	leftBorder := topBorderLen / 2
	rightBorder := topBorderLen - leftBorder

	top := borderStyle.Render("╭" + strings.Repeat("─", leftBorder)) +
		titleStyle.Render(titleText) +
		borderStyle.Render(strings.Repeat("─", rightBorder) + "╮")

	// Content lines with borders
	var middle strings.Builder
	for _, line := range formattedLines {
		padding := innerWidth - len([]rune(line))
		middle.WriteString(borderStyle.Render("│"))
		middle.WriteString(line)
		if padding > 0 {
			middle.WriteString(strings.Repeat(" ", padding))
		}
		middle.WriteString(borderStyle.Render("│"))
		middle.WriteString("\n")
	}

	// Bottom border
	bottom := borderStyle.Render("╰" + strings.Repeat("─", innerWidth) + "╯")

	return top + "\n" + middle.String() + bottom
}
