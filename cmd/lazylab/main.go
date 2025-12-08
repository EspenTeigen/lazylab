package main

import (
	"flag"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/EspenTeigen/lazylab/internal/app"
)

func main() {
	setup := flag.Bool("setup", false, "Configure GitLab connection (add/change host and token)")
	flag.Parse()

	// Check for credentials and show appropriate screen
	var screen tea.Model
	if *setup || !app.HasCredentials() {
		screen = app.NewLauncher()
	} else {
		screen = app.NewMainScreen()
	}

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
