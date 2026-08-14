package ui

import (
	tea "charm.land/bubbletea/v2"
)

type Model struct {
	width    int
	height   int
	quitting bool
	count    int
}

func NewModel() Model {
	return Model{}
}

func (m Model) Init() tea.Cmd {
	return nil
}
