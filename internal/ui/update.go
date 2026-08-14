package ui

import (
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"

	"startnow/internal/catalog"
	"startnow/internal/installer"
)

func (m Model) Init() tea.Cmd { return nil }

func waitForEvents(ch chan installer.Event) tea.Cmd {
	return func() tea.Msg { return <-ch }
}

func tick(d time.Duration) tea.Cmd {
	return func() tea.Msg {
		time.Sleep(d)
		return tickMsg{}
	}
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
	case tickMsg:
		m.spin++
		if m.screen == screenInstall {
			return m, tick(120 * time.Millisecond)
		}
		if m.tab == tabUsage {
			m.usage = m.sampler.Sample()
			return m, tick(time.Second)
		}
	case tea.KeyPressMsg:
		if m.quitting {
			return m, nil
		}
		switch msg.String() {
		case "ctrl+c", "q":
			m.quitting = true
			return m, tea.Quit
		}
		if m.screen == screenInstall {
			return m.updateInstallKey(msg)
		}
		if m.tab == tabTools && m.verActive {
			return m.updateListKey(msg)
		}
		switch msg.String() {
		case "tab":
			m.tab = (m.tab + 1) % 3
			m.tabScroll = 0
		case "shift+tab":
			m.tab = (m.tab + 2) % 3
			m.tabScroll = 0
		case "1":
			m.tab = tabTools
			m.tabScroll = 0
		case "2":
			m.tab = tabMachine
			m.tabScroll = 0
		case "3":
			m.tab = tabUsage
			m.tabScroll = 0
		}
		if m.tab == tabTools {
			return m.updateListKey(msg)
		}
		return m.updateScrollKey(msg)
	case installer.Event:
		return m.updateEvent(msg)
	}
	return m, nil
}

func (m Model) updateListKey(k tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	if m.verActive {
		return m.updateVersionKey(k)
	}
	switch k.String() {
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}
		m.clampScroll()
	case "down", "j":
		if m.cursor < len(m.tools)-1 {
			m.cursor++
		}
		m.clampScroll()
	case "g":
		m.cursor = 0
		m.clampScroll()
	case "G":
		m.cursor = len(m.tools) - 1
		m.clampScroll()
	case "space":
		name := m.tools[m.cursor].Name
		m.selected[name] = !m.selected[name]
	case "a":
		if len(m.selected) == len(m.tools) {
			m.selected = map[string]bool{}
		} else {
			for _, t := range m.tools {
				m.selected[t.Name] = true
			}
		}
	case "v":
		m.verTool = m.cursor
		m.verBuf = m.version[m.tools[m.cursor].Name]
		m.verActive = true
	case "enter":
		var chosen []catalog.Tool
		for _, t := range m.tools {
			if m.selected[t.Name] {
				chosen = append(chosen, t)
			}
		}
		if len(chosen) == 0 {
			return m, nil
		}
		m.screen = screenInstall
		m.jobs = map[string]*job{}
		for _, t := range chosen {
			m.jobs[t.Name] = &job{status: jobPending, message: "queued"}
			go m.runInstall(t)
		}
		return m, waitForEvents(m.events)
	}
	return m, nil
}

func (m Model) updateVersionKey(k tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch k.String() {
	case "enter":
		name := m.tools[m.verTool].Name
		m.version[name] = strings.TrimSpace(m.verBuf)
		m.verActive = false
	case "esc", "\x1b":
		m.verActive = false
	case "backspace", "\b":
		if len(m.verBuf) > 0 {
			m.verBuf = m.verBuf[:len(m.verBuf)-1]
		}
	case "space":
		m.verBuf += " "
	default:
		for _, r := range k.Text {
			if r >= 32 {
				m.verBuf += string(r)
			}
		}
	}
	return m, nil
}

func (m Model) updateScrollKey(k tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch k.String() {
	case "up", "k":
		if m.tabScroll > 0 {
			m.tabScroll--
		}
	case "down", "j":
		m.tabScroll++
	case "g":
		m.tabScroll = 0
	case "G":
		m.tabScroll = 1 << 30
	}
	return m, nil
}

func (m Model) updateInstallKey(k tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	if !m.allDone() {
		return m, nil
	}
	switch k.String() {
	case "enter", "esc", "space":
		m.screen = screenList
		m.probeAll()
		m.jobs = map[string]*job{}
	}
	return m, nil
}

func (m Model) updateEvent(ev installer.Event) (tea.Model, tea.Cmd) {
	j, ok := m.jobs[ev.Tool]
	if !ok {
		return m, waitForEvents(m.events)
	}
	switch ev.Step {
	case installer.StepDone:
		j.status, j.progress = jobDone, 1
		if j.message == "" {
			j.message = "done"
		}
	case installer.StepFailed:
		j.status, j.message = jobFailed, ev.Message
	default:
		j.status = jobRunning
		if ev.Progress > j.progress {
			j.progress = ev.Progress
		}
		if ev.Message != "" {
			j.message = ev.Message
		}
	}
	return m, waitForEvents(m.events)
}

func (m Model) allDone() bool {
	if len(m.jobs) == 0 {
		return false
	}
	for _, j := range m.jobs {
		if j.status != jobDone && j.status != jobFailed {
			return false
		}
	}
	return true
}

func (m *Model) clampScroll() {
	rows := m.visibleRows()
	if m.cursor < m.scroll {
		m.scroll = m.cursor
	}
	if m.cursor >= m.scroll+rows {
		m.scroll = m.cursor - rows + 1
	}
	if m.scroll < 0 {
		m.scroll = 0
	}
}

func (m Model) visibleRows() int {
	rows := m.height - headerLines - 2
	if rows < 3 {
		rows = 3
	}
	return rows
}
