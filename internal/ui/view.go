package ui

import (
	"fmt"

	tea "charm.land/bubbletea/v2"
)

func (m Model) View() tea.View {
	if m.quitting {
		return tea.NewView("bye!\n")
	}
	v := tea.NewView(fmt.Sprintf("Hello, TUI!\n\nspace: count up (%d)   q / ctrl+c: quit\n", m.count))
	v.AltScreen = true
	return v
}
