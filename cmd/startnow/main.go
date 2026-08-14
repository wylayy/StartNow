package main

import (
	"fmt"
	"os"
	"runtime"

	tea "charm.land/bubbletea/v2"

	"startnow/internal/catalog"
	"startnow/internal/installer"
	"startnow/internal/ui"
)

func main() {
	if runtime.GOOS != "linux" {
		fmt.Fprintf(os.Stderr, "startnow: only Linux is supported (found %s)\n", runtime.GOOS)
		os.Exit(1)
	}
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
