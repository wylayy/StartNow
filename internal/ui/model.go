package ui

import (
	"startnow/internal/catalog"
	"startnow/internal/installer"
)

type screen int

const (
	screenList screen = iota
	screenInstall
)

type toolState struct {
	found   bool
	version string
}

type jobStatus int

const (
	jobPending jobStatus = iota
	jobRunning
	jobDone
	jobFailed
)

type job struct {
	status   jobStatus
	progress float64
	message  string
}

type tickMsg struct{}

type Model struct {
	env    *installer.Env
	events chan installer.Event

	tools    []catalog.Tool
	state    map[string]toolState
	selected map[string]bool

	cursor int
	scroll int

	screen screen
	jobs   map[string]*job
	spin   int

	width, height int
	quitting      bool
}

func NewModel(env *installer.Env) Model {
	m := Model{
		env:      env,
		events:   make(chan installer.Event, 256),
		tools:    catalog.Tools(),
		state:    map[string]toolState{},
		selected: map[string]bool{},
		jobs:     map[string]*job{},
	}
	env.Send = func(ev installer.Event) { m.events <- ev }
	m.probeAll()
	return m
}

func (m *Model) probeAll() {
	for _, t := range m.tools {
		if len(t.VersionCmd) == 0 {
			continue
		}
		v, ok := m.env.Probe(t.VersionCmd[0], t.VersionCmd[1:]...)
		m.state[t.Name] = toolState{found: ok, version: v}
	}
}

func (m *Model) runInstall(t catalog.Tool) {
	if err := catalog.Install(&t, m.env); err != nil {
		m.env.Send(installer.Event{Tool: t.Name, Step: installer.StepFailed, Message: err.Error()})
		return
	}
	m.env.Send(installer.Event{Tool: t.Name, Step: installer.StepDone, Progress: 1, Message: "installed"})
}
