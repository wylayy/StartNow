package ui

import (
	"fmt"
	"image/color"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"startnow/internal/machine"
)

const headerLines = 8

const logo = "    ____  _             _   _   _\n" +
	"   / ___|| |_ __ _ _ __| |_| \\ | | _____      __\n" +
	"   \\___ \\| __/ _` | '__| __|  \\| |/ _ \\ \\ /\\ / /\n" +
	"   ___) | || (_| | |  | |_| |\\  | (_) \\ V  V /\n" +
	"   |____/ \\__\\__,_|_|   \\__|_| \\_|\\___/ \\_/\\_/"

var (
	accent      = lipgloss.Color("#38BDF8")
	colorOK     = lipgloss.Color("#4ADE80")
	colorErr    = lipgloss.Color("#F87171")
	colorWarn   = lipgloss.Color("#FACC15")
	colorDim    = lipgloss.Color("#64748B")
	styleLogo   = lipgloss.NewStyle().Foreground(accent)
	styleDim    = lipgloss.NewStyle().Foreground(colorDim)
	styleCursor = lipgloss.NewStyle().Foreground(accent).Bold(true)
	styleName   = lipgloss.NewStyle().Bold(true)
	styleCat    = lipgloss.NewStyle().Foreground(colorDim)
	styleOK     = lipgloss.NewStyle().Foreground(colorOK)
	styleErr    = lipgloss.NewStyle().Foreground(colorErr)
	styleWarn   = lipgloss.NewStyle().Foreground(colorWarn)
	styleTabOn  = lipgloss.NewStyle().Foreground(lipgloss.Color("#0B1220")).Background(accent).Bold(true).Padding(0, 1)
	styleTabOff = lipgloss.NewStyle().Foreground(colorDim).Padding(0, 1)
	styleTitle  = lipgloss.NewStyle().Bold(true).Foreground(accent)
)

func (m Model) View() tea.View {
	if m.quitting {
		return tea.NewView("bye!\n")
	}
	var content string
	if m.screen == screenInstall {
		content = m.installView()
	} else {
		content = m.header() + m.tabContent() + m.footer()
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

func (m Model) header() string {
	var b strings.Builder
	b.WriteString(styleLogo.Render(logo))
	b.WriteString("\n")
	b.WriteString(styleDim.Render(m.machineSummary()))
	b.WriteString("\n")
	b.WriteString(m.tabBar())
	b.WriteString("\n")
	b.WriteString(styleDim.Render(strings.Repeat("─", m.widthOr())))
	b.WriteString("\n")
	return b.String()
}

func (m Model) machineSummary() string {
	mem := machine.HumanBytes(m.machine.MemTotal)
	parts := []string{
		m.machine.User + "@" + m.machine.Hostname,
		m.machine.OS + "/" + m.machine.Arch,
		fmt.Sprintf("%d cores", m.machine.CPUs),
		mem + " RAM",
	}
	if m.machine.Uptime > 0 {
		parts = append(parts, "up "+machine.HumanDuration(m.machine.Uptime))
	}
	return strings.Join(parts, "  ·  ")
}

func (m Model) tabBar() string {
	tabs := []struct {
		t     tab
		label string
	}{
		{tabTools, "Tools"},
		{tabMachine, "Machine"},
		{tabUsage, "Usage"},
	}
	var parts []string
	for _, t := range tabs {
		if m.tab == t.t {
			parts = append(parts, styleTabOn.Render(t.label))
		} else {
			parts = append(parts, styleTabOff.Render(t.label))
		}
	}
	return "  " + strings.Join(parts, "  ")
}

func (m Model) tabContent() string {
	switch m.tab {
	case tabMachine:
		return m.machineView()
	case tabUsage:
		return m.usageView()
	default:
		return m.listView()
	}
}

func (m Model) listView() string {
	var b strings.Builder
	width := m.widthOr()
	for i, t := range m.tools {
		if i < m.scroll || i >= m.scroll+m.visibleRows() {
			continue
		}
		marker := " "
		if i == m.cursor {
			marker = styleCursor.Render(">")
		}
		check := " "
		if m.selected[t.Name] {
			check = "✓"
		}
		status := "not installed"
		statusStyle := styleWarn
		if st, ok := m.state[t.Name]; ok && st.found {
			status = firstLine(st.version)
			if status == "" {
				status = "installed"
			}
			statusStyle = styleOK
		}
		badge := ""
		if v, ok := m.version[t.Name]; ok && v != "" {
			badge = styleWarn.Render(" → " + v)
		}
		left := fmt.Sprintf("%s [%s] %s%s  %s  %s", marker, check, t.DisplayName, badge, t.Category, t.Description)
		left = truncate(left, width-statusWidth-1)
		b.WriteString(pad(left, width-statusWidth-1))
		b.WriteString(pad(statusStyle.Render(truncate(status, statusWidth)), statusWidth))
		b.WriteString("\n")
	}
	b.WriteString("\n")
	b.WriteString(styleDim.Render(fmt.Sprintf("%d/%d selected", len(m.selected), len(m.tools))))
	b.WriteString("\n")
	b.WriteString(styleDim.Render("Binaries go to ~/.startnow/bin — add it to your PATH"))
	if m.verActive {
		t := m.tools[m.verTool]
		b.WriteString("\n\n")
		b.WriteString(styleTitle.Render("version for " + t.DisplayName))
		b.WriteString(" (empty = latest): " + m.verBuf + "▍")
	}
	return b.String()
}

const statusWidth = 30

func (m Model) machineView() string {
	width := m.widthOr()
	inf := m.machine
	mem := machine.HumanBytes(inf.MemTotal)
	rows := [][2]string{
		{"Hostname", inf.Hostname},
		{"User", inf.User},
		{"OS", inf.OS},
		{"Kernel", inf.Kernel},
		{"Arch", inf.Arch},
		{"CPU", fmt.Sprintf("%dx %s", inf.CPUs, inf.CPUModel)},
		{"Memory", fmt.Sprintf("%s total / %s available", mem, machine.HumanBytes(inf.MemAvail))},
		{"Uptime", machine.HumanDuration(inf.Uptime)},
		{"Shell", inf.Shell},
		{"Go", inf.GoVer},
	}
	var lines []string
	for _, r := range rows {
		key := styleTitle.Render(pad(r[0], 10))
		val := truncate(r[1], width-14)
		lines = append(lines, key+val)
	}
	return m.scrollable(strings.Join(lines, "\n"), width)
}

func (m Model) usageView() string {
	width := m.widthOr()
	u := m.usage
	var b strings.Builder
	cpu := usageBar(u.CPU/100, 18, accent)
	b.WriteString("CPU     " + cpu + fmt.Sprintf(" %3.0f%%   %d cores\n", u.CPU, m.machine.CPUs))
	var memPct float64
	if u.MemTotal > 0 {
		memPct = 100 - float64(u.MemAvail)/float64(u.MemTotal)*100
	}
	b.WriteString("Memory  " + usageBar(memPct/100, 18, memColor(memPct)))
	b.WriteString(fmt.Sprintf(" %3.0f%%   %s used / %s total\n", memPct, machine.HumanBytes(u.MemTotal-u.MemAvail), machine.HumanBytes(u.MemTotal)))
	var swapPct float64
	if u.SwapTotal > 0 {
		swapPct = float64(u.SwapUsed) / float64(u.SwapTotal) * 100
	}
	b.WriteString("Swap    " + usageBar(swapPct/100, 18, accent))
	b.WriteString(fmt.Sprintf(" %3.0f%%   %s used / %s total\n", swapPct, machine.HumanBytes(u.SwapUsed), machine.HumanBytes(u.SwapTotal)))
	b.WriteString(fmt.Sprintf("Load    %.2f  %.2f  %.2f (1m 5m 15m)\n", u.Load1, u.Load5, u.Load15))
	if u.DiskTotal > 0 {
		diskPct := float64(u.DiskUsed) / float64(u.DiskTotal) * 100
		b.WriteString("Disk /  " + usageBar(diskPct/100, 18, accent))
		b.WriteString(fmt.Sprintf(" %3.0f%%   %s used / %s total\n", diskPct, machine.HumanBytes(u.DiskUsed), machine.HumanBytes(u.DiskTotal)))
	}
	b.WriteString("\n")
	b.WriteString(styleTitle.Render("Top processes by memory"))
	b.WriteString("\n")
	if len(u.Procs) == 0 {
		b.WriteString("  —")
	}
	for _, p := range u.Procs {
		b.WriteString(fmt.Sprintf("  %6d  %-24s %s\n", p.PID, truncate(p.Name, 24), machine.HumanBytes(p.Mem)))
	}
	out := strings.Split(b.String(), "\n")
	for i, l := range out {
		out[i] = truncate(l, width)
	}
	return strings.Join(out, "\n")
}

func memColor(pct float64) color.Color {
	switch {
	case pct > 90:
		return colorErr
	case pct > 70:
		return colorWarn
	default:
		return colorOK
	}
}

func usageBar(frac float64, width int, c color.Color) string {
	if width < 3 {
		width = 3
	}
	n := int(frac * float64(width-2))
	if n > width-2 {
		n = width - 2
	}
	if n < 0 {
		n = 0
	}
	fill := lipgloss.NewStyle().Foreground(c).Render(strings.Repeat("=", n))
	return "[" + fill + strings.Repeat(" ", width-2-n) + "]"
}

func progressBar(frac float64, width int) string {
	return usageBar(frac, width, accent)
}

func (m Model) scrollable(content string, width int) string {
	lines := strings.Split(content, "\n")
	rows := m.visibleRows()
	offset := m.tabScroll
	if max := len(lines) - rows; offset > max {
		offset = max
	}
	if offset < 0 {
		offset = 0
	}
	out := make([]string, 0, rows)
	for i := offset; i < len(lines) && i < offset+rows; i++ {
		out = append(out, truncate(lines[i], width))
	}
	return strings.Join(out, "\n")
}

func (m Model) footer() string {
	keys := "↑/↓ move · space toggle · a select all · v version · enter install · tab next · q quit"
	if m.tab != tabTools {
		keys = "↑/↓ scroll · g/G jump · tab next · q quit"
	}
	return "\n" + styleDim.Render(pad(keys, m.widthOr()))
}

func (m Model) installView() string {
	var b strings.Builder
	width := m.widthOr()
	b.WriteString(styleTitle.Render("Installing"))
	b.WriteString("\n\n")
	spinner := string("|/-\\"[m.spin%4])
	var done, failed, running int
	for _, t := range m.tools {
		j, ok := m.jobs[t.Name]
		if !ok {
			continue
		}
		icon := styleDim.Render("·")
		switch j.status {
		case jobDone:
			icon = styleOK.Render("✓")
			done++
		case jobFailed:
			icon = styleErr.Render("✗")
			failed++
		case jobRunning:
			icon = styleCursor.Render(spinner)
			running++
		}
		left := fmt.Sprintf("%s %s", icon, t.DisplayName)
		var rest string
		if j.status == jobFailed {
			rest = "failed: " + truncate(j.message, width-16)
		} else {
			bar := progressBar(j.progress, 18)
			msg := styleDim.Render(truncate(j.message, width-42))
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
	b.WriteString(progressBar(overall, width-10))
	b.WriteString(fmt.Sprintf(" %3.0f%%\n", overall*100))
	if m.allDone() {
		summary := fmt.Sprintf("%d installed, %d failed — enter: back to list", done, failed)
		if failed > 0 {
			b.WriteString(styleErr.Render(summary))
		} else {
			b.WriteString(styleOK.Render(summary))
		}
		b.WriteString("\n")
	} else {
		b.WriteString(styleDim.Render(fmt.Sprintf("%d running, %d failed", running, failed)))
		b.WriteString("\n")
	}
	return b.String()
}

func pad(s string, w int) string {
	if lipgloss.Width(s) >= w {
		return s
	}
	return s + strings.Repeat(" ", w-lipgloss.Width(s))
}

func truncate(s string, w int) string {
	if w <= 0 {
		return ""
	}
	if lipgloss.Width(s) <= w {
		return s
	}
	runes := []rune(s)
	var b strings.Builder
	for _, r := range runes {
		if lipgloss.Width(b.String()+string(r)) > w {
			break
		}
		b.WriteRune(r)
	}
	return b.String()
}

func firstLine(s string) string {
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return s[:i]
	}
	return s
}
