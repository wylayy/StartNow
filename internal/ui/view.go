package ui

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
)

func (m Model) View() tea.View {
	if m.quitting {
		return tea.NewView("bye!\n")
	}
	var content string
	if m.screen == screenList {
		content = m.listView()
	} else {
		content = m.installView()
	}
	v := tea.NewView(content)
	v.AltScreen = true
	v.WindowTitle = "StartNow"
	return v
}

func (m Model) widthOr() int {
	if m.width > 0 {
		return m.width
	}
	return 80
}

func (m Model) listView() string {
	var b strings.Builder
	b.WriteString("StartNow — developer tool installer\n\n")
	for i, t := range m.tools {
		if i < m.scroll || i >= m.scroll+m.visibleRows() {
			continue
		}
		marker := " "
		if i == m.cursor {
			marker = ">"
		}
		check := " "
		if m.selected[t.Name] {
			check = "x"
		}
		status := "not installed"
		if st, ok := m.state[t.Name]; ok && st.found {
			status = firstLine(st.version)
			if status == "" {
				status = "installed"
			}
		}
		left := fmt.Sprintf("%s [%s] %-10s %s", marker, check, t.DisplayName, t.Description)
		right := truncate(status, 28)
		b.WriteString(pad(left, m.widthOr()-30))
		b.WriteString(right)
		b.WriteString("\n")
	}
	b.WriteString(fmt.Sprintf("\n%s\n%s\n",
		pad(fmt.Sprintf("%d/%d selected — enter: install", len(m.selected), len(m.tools)), m.widthOr()),
		"↑/↓ move · space toggle · a select all · q quit"))
	b.WriteString("Binaries go to ~/.startnow/bin — add it to your PATH\n")
	return b.String()
}

func (m Model) installView() string {
	var b strings.Builder
	b.WriteString("Installing\n\n")
	spinner := string("|/-\\"[m.spin%4])
	var done, failed, running int
	for _, t := range m.tools {
		j, ok := m.jobs[t.Name]
		if !ok {
			continue
		}
		icon := "·"
		switch j.status {
		case jobDone:
			icon = "✓"
			done++
		case jobFailed:
			icon = "✗"
			failed++
		case jobRunning:
			icon = spinner
			running++
		}
		left := fmt.Sprintf(" %s %-10s", icon, t.DisplayName)
		var rest string
		if j.status == jobFailed {
			rest = "failed: " + truncate(j.message, m.widthOr()-16)
		} else {
			bar := progressBar(j.progress, 18)
			msg := truncate(j.message, m.widthOr()-40)
			rest = fmt.Sprintf("%s %3.0f%%  %s", bar, j.progress*100, msg)
		}
		b.WriteString(pad(left, 14))
		b.WriteString(rest)
		b.WriteString("\n")
	}
	var sum float64
	for _, j := range m.jobs {
		if j.status == jobDone || j.status == jobFailed {
			sum += 1
		} else {
			sum += j.progress
		}
	}
	overall := 0.0
	if len(m.jobs) > 0 {
		overall = sum / float64(len(m.jobs))
	}
	b.WriteString("\noverall ")
	b.WriteString(progressBar(overall, m.widthOr()-8))
	b.WriteString(fmt.Sprintf(" %3.0f%%\n", overall*100))
	if m.allDone() {
		b.WriteString(fmt.Sprintf("\n%d installed, %d failed — enter: back to list\n", done, failed))
	} else {
		b.WriteString(fmt.Sprintf("\n%d running, %d failed\n", running, failed))
	}
	return b.String()
}

func progressBar(frac float64, width int) string {
	if width < 3 {
		return "[=]"
	}
	filled := int(frac * float64(width-2))
	if filled > width-2 {
		filled = width - 2
	}
	if filled < 0 {
		filled = 0
	}
	return "[" + strings.Repeat("=", filled) + strings.Repeat(" ", width-2-filled) + "]"
}

func pad(s string, w int) string {
	if len(s) >= w {
		return s
	}
	return s + strings.Repeat(" ", w-len(s))
}

func truncate(s string, w int) string {
	if len(s) <= w || w <= 0 {
		return s
	}
	return s[:w]
}

func firstLine(s string) string {
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return s[:i]
	}
	return s
}
