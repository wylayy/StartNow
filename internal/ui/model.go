package ui

import (
	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/spinner"
	"charm.land/bubbles/v2/table"
	"charm.land/bubbles/v2/textinput"
	"charm.land/lipgloss/v2"

	"startnow/internal/catalog"
	"startnow/internal/installer"
	"startnow/internal/machine"
)

type screen int

const (
	screenList screen = iota
	screenInstall
)

type tab int

const (
	tabTools tab = iota
	tabMachine
	tabUsage
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
	version  map[string]string

	managed     map[string]bool
	manifestVer map[string]string
	latest      map[string]string
	updateAvail map[string]bool
	notice      string

	table       table.Model
	rows        []catalog.Tool
	filterInput textinput.Model
	verName     string
	verInput    textinput.Model
	spinner     spinner.Model
	help        help.Model

	details *catalog.Tool

	downloadURL map[string]string
	resolving   map[string]bool

	tooSmall bool

	tab tab

	machine machine.Info
	sampler *machine.Sampler
	usage   machine.Usage

	screen screen
	jobs   map[string]*job

	width, height int
	quitting      bool
}

func NewModel(env *installer.Env) Model {
	m := Model{
		env:         env,
		events:      make(chan installer.Event, 256),
		tools:       catalog.Tools(),
		state:       map[string]toolState{},
		selected:    map[string]bool{},
		version:     map[string]string{},
		managed:     map[string]bool{},
		manifestVer: map[string]string{},
		latest:      map[string]string{},
		updateAvail: map[string]bool{},
		downloadURL: map[string]string{},
		resolving:   map[string]bool{},
		jobs:        map[string]*job{},
		machine:     machine.Collect(),
		sampler:     machine.NewSampler(),
	}
	env.Send = func(ev installer.Event) { m.events <- ev }
	if manifest, err := env.LoadManifest(); err == nil {
		for name, entry := range manifest.Installs {
			m.managed[name] = true
			m.manifestVer[name] = entry.Version
		}
	}
	m.usage = m.sampler.Sample()
	m.probeAll()

	m.rows = append([]catalog.Tool(nil), m.tools...)
	m.table = newToolsTable()
	m.syncTableRows()

	m.filterInput = textinput.New()
	m.filterInput.Placeholder = "filter…"
	m.filterInput.CharLimit = 40
	m.filterInput.Prompt = ""

	m.verInput = textinput.New()
	m.verInput.Placeholder = "e.g. 1.26.5 — empty for latest"
	m.verInput.CharLimit = 40
	m.verInput.Prompt = ""

	m.spinner = spinner.New(spinner.WithSpinner(spinner.Line), spinner.WithStyle(lipgloss.NewStyle().Foreground(accent)))
	m.help = help.New()
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
	t.Version = m.version[t.Name]
	version, err := catalog.Install(&t, m.env)
	if err != nil {
		m.env.Send(installer.Event{Tool: t.Name, Step: installer.StepFailed, Message: err.Error()})
		return
	}
	if err := m.env.RecordInstall(t.Name, version); err != nil {
		m.env.Send(installer.Event{Tool: t.Name, Step: installer.StepFailed, Message: "manifest update failed: " + err.Error()})
		return
	}
	m.env.Send(installer.Event{Tool: t.Name, Step: installer.StepDone, Progress: 1, Message: "installed", Version: version})
}
