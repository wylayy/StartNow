package main

import (
	"fmt"
	"os"

	tea "charm.land/bubbletea/v2"

	"startnow/internal/catalog"
	"startnow/internal/installer"
	"startnow/internal/ui"
)

func main() {
	env, err := installer.NewEnv(nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "startnow: %v\n", err)
		os.Exit(1)
	}
	if err := catalog.ValidateAll(); err != nil {
		fmt.Fprintf(os.Stderr, "startnow: invalid catalog: %v\n", err)
		os.Exit(1)
	}
	p := tea.NewProgram(ui.NewModel(env))
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "startnow: %v\n", err)
		os.Exit(1)
	}
}
