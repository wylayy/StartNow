package ui

import (
	"strings"
	"time"

	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"

	"startnow/internal/catalog"
	"startnow/internal/installer"
)

func (m Model) Init() tea.Cmd {
	return tick(time.Second)
}

func waitForEvents(ch chan installer.Event) tea.Cmd {
	return func() tea.Msg { return <-ch }
}

func tick(d time.Duration) tea.Cmd {
	return func() tea.Msg {
		time.Sleep(d)
		return tickMsg{}
	}
}

type downloadMsg struct {
	name string
	url  string
	err  error
}

func resolveDownloadCmd(t catalog.Tool, env *installer.Env) tea.Cmd {
	return func() tea.Msg {
		url, err := catalog.ResolveDownload(&t, env)
		return downloadMsg{name: t.Name, url: url, err: err}
	}
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		if msg.Width > 0 {
			m.tooSmall = msg.Width < minWidth || msg.Height < minHeight
		}
		m.table.SetWidth(m.widthOr() - 4)
		m.table.SetHeight(m.listHeight())
		m.verInput.SetWidth(m.widthOr() - 34)
		m.filterInput.SetWidth(m.widthOr() - 14)
	case tickMsg:
		if m.screen == screenInstall {
			return m, tick(120 * time.Millisecond)
		}
		if m.tab == tabUsage {
			m.spinner, _ = m.spinner.Update(spinner.TickMsg{Time: time.Now()})
			m.usage = m.sampler.Sample()
		}
		return m, tick(time.Second)
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	case tea.KeyPressMsg:
		return m.handleKey(msg)
	case tea.MouseClickMsg:
		return m.handleClick(msg)
	case downloadMsg:
		if msg.err != nil {
			m.downloadURL[msg.name] = "resolve failed: " + msg.err.Error()
		} else {
			m.downloadURL[msg.name] = msg.url
		}
		delete(m.resolving, msg.name)
		return m, nil
	case installer.Event:
		return m.updateEvent(msg)
	default:
		return m, nil
	}
	return m, nil
}

func (m Model) handleKey(k tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	if m.quitting {
		return m, nil
	}
	m.notice = ""
	if k.String() == "ctrl+c" {
		m.quitting = true
		return m, tea.Quit
	}
	if m.screen == screenInstall {
		return m.updateInstallKey(k)
	}
	if m.details != nil {
		switch k.String() {
		case "esc":
			m.details = nil
		}
		return m, nil
	}
	if m.verInput.Focused() {
		return m.updateVersionKey(k)
	}
	if m.filterInput.Focused() {
		return m.updateFilterKey(k)
	}
	switch k.String() {
	case "q":
		m.quitting = true
		return m, tea.Quit
	case "?":
		m.help.ShowAll = !m.help.ShowAll
	case "tab":
		m.tab = (m.tab + 1) % 3
	case "shift+tab":
		m.tab = (m.tab + 2) % 3
	case "1":
		m.tab = tabTools
	case "2":
		m.tab = tabMachine
	case "3":
		m.tab = tabUsage
	}
	if m.tab != tabTools {
		return m, nil
	}
	switch k.String() {
	case "/":
		return m, m.filterInput.Focus()
	case "i":
		if t, ok := m.selectedTool(); ok {
			m.details = &t
			if m.filterInput.Focused() {
				m.filterInput.Blur()
			}
			if m.verInput.Focused() {
				m.verInput.Blur()
			}
			if t.Kind == catalog.KindArchive && m.downloadURL[t.Name] == "" {
				m.resolving[t.Name] = true
				return m, resolveDownloadCmd(t, m.env)
			}
			return m, nil
		}
	case "v":
		if t, ok := m.selectedTool(); ok {
			m.verName = t.Name
			m.verInput.SetValue(m.version[t.Name])
			return m, m.verInput.Focus()
		}
	case "space":
		if t, ok := m.selectedTool(); ok {
			m.selected[t.Name] = !m.selected[t.Name]
			m.syncTableRows()
		}
		return m, nil
	case "a":
		if len(m.selected) == len(m.tools) {
			for k := range m.selected {
				delete(m.selected, k)
			}
		} else {
			for _, t := range m.tools {
				m.selected[t.Name] = true
			}
		}
		m.syncTableRows()
		return m, nil
	case "u":
		if t, ok := m.selectedTool(); ok {
			return m.updateTool(t)
		}
	case "x":
		if t, ok := m.selectedTool(); ok {
			return m.removeTool(t)
		}
	case "enter":
		var chosen []catalog.Tool
		for _, t := range m.tools {
			if m.selected[t.Name] {
				chosen = append(chosen, t)
			}
		}
		return m.startInstall(chosen)
	}
	var cmd tea.Cmd
	m.table, cmd = m.table.Update(k)
	m.syncTableRows()
	return m, cmd
}

func (m Model) handleClick(ev tea.MouseClickMsg) (tea.Model, tea.Cmd) {
	if m.screen != screenList {
		return m, nil
	}
	if ev.Button == tea.MouseRight {
		if m.tab == tabTools {
			if idx, ok := m.toolRowAt(ev.Y); ok {
				t := m.rows[idx]
				m.details = &t
				if m.filterInput.Focused() {
					m.filterInput.Blur()
				}
				if m.verInput.Focused() {
					m.verInput.Blur()
				}
				if t.Kind == catalog.KindArchive && m.downloadURL[t.Name] == "" {
					m.resolving[t.Name] = true
					return m, resolveDownloadCmd(t, m.env)
				}
				return m, nil
			}
		}
		if m.details != nil {
			m.details = nil
		}
		return m, nil
	}
	if m.details != nil {
		m.details = nil
	}
	_, rects := m.tabLayout()
	for _, r := range rects {
		if ev.X >= r.x0 && ev.X < r.x1 && ev.Y >= r.y0 && ev.Y < r.y1 {
			m.tab = r.tab
			m.notice = ""
			if m.filterInput.Focused() {
				m.filterInput.Blur()
			}
			return m, nil
		}
	}
	if m.tab == tabTools && ev.Y == m.filterRowY() && ev.X > 0 && ev.X < m.widthOr()-1 {
		if !m.filterInput.Focused() {
			return m, m.filterInput.Focus()
		}
		return m, nil
	}
	if m.filterInput.Focused() {
		m.filterInput.Blur()
	}
	return m, nil
}

// toolRowAt maps a screen Y to a tools-table row index within the current
// (possibly filtered) row set. The table header sits at headerHeight+3.
func (m Model) toolRowAt(y int) (int, bool) {
	idx := y - (m.headerHeight() + 3)
	if idx < 0 || idx >= len(m.rows) {
		return 0, false
	}
	return idx, true
}

// filterRowY returns the screen row of the filter input inside the Tools box.
func (m Model) filterRowY() int {
	return m.headerHeight() + 1
}

func (m Model) updateFilterKey(k tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch k.String() {
	case "enter":
		m.filterInput.Blur()
		return m, nil
	case "esc":
		m.filterInput.Blur()
		m.filterInput.SetValue("")
		m.rows = m.filterTools()
		m.syncTableRows()
		return m, nil
	}
	prev := m.filterInput.Value()
	var cmd tea.Cmd
	m.filterInput, cmd = m.filterInput.Update(k)
	if m.filterInput.Value() != prev {
		m.rows = m.filterTools()
		m.syncTableRows()
		m.table.GotoTop()
	}
	return m, cmd
}

func (m Model) updateTool(t catalog.Tool) (tea.Model, tea.Cmd) {
	name := t.Name
	if !m.managed[name] {
		m.notice = name + " is not managed by startnow — select it and press enter to install"
		return m, nil
	}
	latest, err := catalog.ResolveVersion(&t, m.env)
	if err != nil {
		m.notice = "update check failed: " + err.Error()
		return m, nil
	}
	m.latest[name] = latest
	if latest == "" || catalog.CompareVersions(m.manifestVer[name], latest) >= 0 {
		m.updateAvail[name] = false
		m.notice = name + " is up to date"
		m.syncTableRows()
		return m, nil
	}
	m.updateAvail[name] = true
	pinned := t
	pinned.Version = latest
	m.notice = "updating " + name + " to " + latest
	m.syncTableRows()
	return m.startInstall([]catalog.Tool{pinned})
}

func (m Model) removeTool(t catalog.Tool) (tea.Model, tea.Cmd) {
	name := t.Name
	if !m.managed[name] {
		m.notice = name + " is not managed by startnow"
		return m, nil
	}
	if err := catalog.Uninstall(&t, m.env); err != nil {
		m.notice = "uninstall failed: " + err.Error()
		return m, nil
	}
	delete(m.managed, name)
	delete(m.manifestVer, name)
	delete(m.updateAvail, name)
	m.notice = "removed " + name
	m.probeAll()
	m.syncTableRows()
	return m, nil
}

func (m Model) startInstall(chosen []catalog.Tool) (tea.Model, tea.Cmd) {
	if len(chosen) == 0 {
		return m, nil
	}
	m.screen = screenInstall
	m.jobs = map[string]*job{}
	for _, t := range chosen {
		m.jobs[t.Name] = &job{status: jobPending, message: "queued"}
		go m.runInstall(t)
	}
	return m, tea.Batch(waitForEvents(m.events), m.spinner.Tick)
}

func (m Model) updateVersionKey(k tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch k.String() {
	case "enter":
		m.version[m.verName] = strings.TrimSpace(m.verInput.Value())
		m.verInput.Blur()
		m.syncTableRows()
		return m, nil
	case "esc":
		m.verInput.Blur()
		return m, nil
	}
	var cmd tea.Cmd
	m.verInput, cmd = m.verInput.Update(k)
	return m, cmd
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
		m.syncTableRows()
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
		m.managed[ev.Tool] = true
		if ev.Version != "" {
			m.manifestVer[ev.Tool] = ev.Version
		}
		if latest, ok := m.latest[ev.Tool]; ok {
			m.updateAvail[ev.Tool] = catalog.CompareVersions(m.manifestVer[ev.Tool], latest) < 0
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

func (m Model) listHeight() int {
	h := m.height - m.headerHeight() - 6
	if n := len(m.tools) + 2; h > n {
		h = n
	}
	if h < 3 {
		h = 3
	}
	return h
}
