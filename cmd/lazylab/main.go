package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/espen/lazylab/internal/app"
)

func main() {
	// Create main screen (lazygit-style multi-panel)
	screen := app.NewMainScreen()

	// Run the TUI
	p := tea.NewProgram(
		screen,
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
