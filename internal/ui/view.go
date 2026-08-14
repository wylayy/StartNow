package ui

import (
	"fmt"
	"image/color"
	"sort"
	"strings"

	"charm.land/bubbles/v2/progress"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	box "github.com/box-cli-maker/box-cli-maker/v3"

	"startnow/internal/catalog"
	"startnow/internal/machine"
)

const minWidth = 60
const minHeight = 15

// headerHeight returns the number of screen rows the header occupies:
// logo (when wide enough) + machine summary + tab pills (3 rows) + separator.
func (m Model) headerHeight() int {
	h := 1
	if m.widthOr() >= 70 {
		h += len(strings.Split(logo, "\n"))
	}
	h += 3
	h += 1
	return h
}

const logo = "  ____  _             _   _   _               \n" +
	" / ___|| |_ __ _ _ __| |_| \\ | | _____      __\n" +
	" \\___ \\| __/ _` | '__| __|  \\| |/ _ \\ \\ /\\ / /\n" +
	"  ___) | || (_| | |  | |_| |\\  | (_) \\ V  V / \n" +
	" |____/ \\__\\__,_|_|   \\__|_| \\_|\\___/ \\_/\\_/  "

var (
	accent      = lipgloss.Color("#38BDF8")
	colorOK     = lipgloss.Color("#4ADE80")
	colorErr    = lipgloss.Color("#F87171")
	colorWarn   = lipgloss.Color("#FACC15")
	colorDim    = lipgloss.Color("#64748B")
	styleDim    = lipgloss.NewStyle().Foreground(colorDim)
	styleCursor = lipgloss.NewStyle().Foreground(accent).Bold(true)
	styleOK     = lipgloss.NewStyle().Foreground(colorOK)
	styleErr    = lipgloss.NewStyle().Foreground(colorErr)
	styleWarn   = lipgloss.NewStyle().Foreground(colorWarn)
	styleTitle  = lipgloss.NewStyle().Bold(true).Foreground(accent)
)

func (m Model) View() tea.View {
	if m.quitting {
		return tea.NewView("bye!\n")
	}
	if m.tooSmall {
		v := tea.NewView(styleWarn.Render(fmt.Sprintf(
			"Terminal too small (%dx%d) — resize to at least %dx%d",
			m.width, m.height, minWidth, minHeight,
		)))
		v.AltScreen = true
		v.WindowTitle = "StartNow"
		return v
	}
	var content string
	if m.screen == screenInstall {
		content = m.installView()
	} else {
		content = m.header() + m.tabContent() + "\n" + m.footer()
	}
	v := tea.NewView(content)
	v.AltScreen = true
	v.WindowTitle = "StartNow"
	v.MouseMode = tea.MouseModeCellMotion
	return v
}

type pillRect struct {
	tab    tab
	x0, x1 int
	y0, y1 int
}

func (m Model) widthOr() int {
	if m.width > 0 {
		return m.width
	}
	return 80
}

var logoGradient = []color.Color{
	lipgloss.Color("#7DD3FC"),
	lipgloss.Color("#38BDF8"),
	lipgloss.Color("#0EA5E9"),
	lipgloss.Color("#0284C7"),
	lipgloss.Color("#0369A1"),
}

func (m Model) header() string {
	var b strings.Builder
	width := m.widthOr()
	if width >= 70 {
		lines := strings.Split(logo, "\n")
		for i, l := range lines {
			line := lipgloss.NewStyle().
				Foreground(logoGradient[i%len(logoGradient)]).
				Width(width).
				Align(lipgloss.Center).
				Render(l)
			b.WriteString(line)
			b.WriteString("\n")
		}
	}
	b.WriteString(m.machineSummaryLine(width))
	b.WriteString(m.tabBar(width))
	b.WriteString(styleDim.Render(strings.Repeat("─", width)))
	b.WriteString("\n")
	return b.String()
}

func (m Model) machineSummaryLine(width int) string {
	host := m.machine.User + "@" + m.machine.Hostname
	line := strings.Replace(m.machineSummary(), host, styleCursor.Render(host), 1)
	if width < 80 {
		line = truncate(line, width)
	}
	return lipgloss.NewStyle().Width(width).Align(lipgloss.Center).Render(line) + "\n"
}

// tabPills renders the tab pills and their cell widths (including margins).
func (m Model) tabPills() ([]string, []int) {
	tabs := []struct {
		t     tab
		label string
	}{
		{tabTools, "Tools"},
		{tabMachine, "Machine"},
		{tabUsage, "Usage"},
	}
	styleOn := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(accent).
		Foreground(accent).
		Bold(true).
		Padding(0, 2).
		MarginRight(2)
	styleOff := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colorDim).
		Foreground(colorDim).
		Padding(0, 2).
		MarginRight(2)
	pills := make([]string, 0, len(tabs))
	widths := make([]int, 0, len(tabs))
	for _, t := range tabs {
		if m.tab == t.t {
			pills = append(pills, styleOn.Render(t.label))
		} else {
			pills = append(pills, styleOff.Render(t.label))
		}
		widths = append(widths, lipgloss.Width(pills[len(pills)-1]))
	}
	return pills, widths
}

func (m Model) tabBar(width int) string {
	pills, _ := m.tabPills()
	bar := lipgloss.JoinHorizontal(lipgloss.Top, pills...)
	return lipgloss.NewStyle().Width(width).Align(lipgloss.Center).Render(bar) + "\n"
}

// tabLayout returns the screen Y of the tab bar block and the clickable
// rectangle of each pill, mirroring header()'s rendering.
func (m Model) tabLayout() (barTop int, rects []pillRect) {
	y := 0
	if m.widthOr() >= 70 {
		y += len(strings.Split(logo, "\n")) // logo
	}
	y += 1 // machine summary
	barTop = y
	width := m.widthOr()
	pills, widths := m.tabPills()
	barWidth := 0
	for _, w := range widths {
		barWidth += w
	}
	x := (width - barWidth) / 2
	tabs := []tab{tabTools, tabMachine, tabUsage}
	for i, p := range pills {
		rects = append(rects, pillRect{tab: tabs[i], x0: x, x1: x + widths[i], y0: barTop, y1: barTop + lipgloss.Height(p)})
		x += widths[i]
	}
	return barTop, rects
}

// boxify renders content in a rounded box sized to the terminal width.
// targetLines (0 = content-sized) pads the box to fill the terminal height.
func (m Model) boxify(title, content string, targetLines int) string {
	inner := m.widthOr() - 4
	if inner < 10 {
		inner = 10
	}
	lines := strings.Split(content, "\n")
	for i, l := range lines {
		lines[i] = pad(l, inner)
	}
	for len(lines) < targetLines {
		lines = append(lines, "")
	}
	b := box.NewBox().
		Style(box.Round).
		Padding(1, 0).
		TitlePosition(box.Top).
		ContentAlign(box.Left).
		Color("#38BDF8").
		WrapContent(true).
		WrapLimit(inner)
	out, err := b.Render(title, strings.Join(lines, "\n"))
	if err != nil {
		return content
	}
	return strings.TrimSuffix(out, "\n")
}

// footerLines returns the number of screen rows the footer occupies.
func (m Model) footerLines() int {
	if !m.help.ShowAll {
		return 1
	}
	keys := toolKeys()
	if m.tab != tabTools {
		keys = navKeys()
	}
	max := 1
	for _, g := range keys.FullHelp() {
		if len(g) > max {
			max = len(g)
		}
	}
	return max
}

// boxTarget is the content-line count that makes the box reach the footer.
func (m Model) boxTarget() int {
	if m.height == 0 {
		return 0
	}
	return m.height - m.headerHeight() - 2 - m.footerLines()
}

// bar renders a bubbles progress bar with an optional color blend.
func bar(frac float64, width int, colors ...color.Color) string {
	opts := []progress.Option{progress.WithWidth(width), progress.WithScaled(true)}
	if len(colors) > 0 {
		opts = append(opts, progress.WithColors(colors...))
	}
	return progress.New(opts...).ViewAs(frac)
}

func (m Model) machineSummary() string {
	mem := machine.HumanBytes(m.machine.MemTotal)
	parts := []string{
		m.machine.User + "@" + m.machine.Hostname,
		m.machine.OS + "/" + m.machine.Arch,
		fmt.Sprintf("%d cores", m.machine.CPUs),
		mem + " RAM",
	}
	if d := m.machine.Distro.Pretty; d != "" {
		parts = append([]string{m.machine.User + "@" + m.machine.Hostname, d}, parts[1:]...)
	}
	if m.machine.Uptime > 0 {
		parts = append(parts, "up "+machine.HumanDuration(m.machine.Uptime))
	}
	return strings.Join(parts, "  ·  ")
}

func (m Model) tabContent() string {
	if m.details != nil {
		return m.detailsView(m.details)
	}
	switch m.tab {
	case tabMachine:
		return m.machineView()
	case tabUsage:
		return m.usageView()
	default:
		return m.listView()
	}
}

func (m Model) detailsView(t *catalog.Tool) string {
	var rows [][2]string
	rows = append(rows,
		[2]string{"Name", t.Name},
		[2]string{"Category", t.Category},
		[2]string{"Kind", string(t.Kind)},
	)
	if t.Source != "" {
		rows = append(rows, [2]string{"Source", string(t.Source)})
	}
	rows = append(rows, [2]string{"Description", t.Description})
	status := "not installed"
	if st, ok := m.state[t.Name]; ok && st.found {
		status = firstLine(st.version)
		if status == "" {
			status = "installed"
		}
	}
	if m.updateAvail[t.Name] {
		status = "update → " + m.latest[t.Name]
	}
	rows = append(rows, [2]string{"Status", status})
	if m.managed[t.Name] {
		rows = append(rows, [2]string{"Managed", "yes (v" + m.manifestVer[t.Name] + ")"})
	}
	if v, ok := m.version[t.Name]; ok && v != "" {
		rows = append(rows, [2]string{"Pinned", v})
	}
	support := "support"
	if !t.Supported(m.env) {
		support = "unsupport"
	}
	rows = append(rows, [2]string{"Support", support})
	switch t.Kind {
	case catalog.KindArchive:
		url := m.downloadURL[t.Name]
		if url == "" {
			if m.resolving[t.Name] {
				url = "resolving…"
			} else {
				url = t.ArchiveURL
			}
		}
		rows = append(rows, [2]string{"Download", url})
		if latest := m.latest[t.Name]; latest != "" {
			rows = append(rows, [2]string{"Latest", latest})
		}
	case catalog.KindScript:
		rows = append(rows, [2]string{"Script", t.ScriptURL})
	case catalog.KindPkg:
		pkgs := make([]string, 0, len(t.Pkg))
		for id, p := range t.Pkg {
			pkgs = append(pkgs, id+": "+p)
		}
		sort.Strings(pkgs)
		rows = append(rows, [2]string{"Packages", strings.Join(pkgs, ", ")})
	}
	rows = append(rows, [2]string{"Probe", strings.Join(t.VersionCmd, " ")})
	var lines []string
	for _, r := range rows {
		lines = append(lines, styleTitle.Render(pad(r[0], 12))+truncate(r[1], m.widthOr()-18))
	}
	lines = append(lines, "")
	lines = append(lines, styleDim.Render("right-click or esc: close"))
	return m.boxify(t.DisplayName+" — details", strings.Join(lines, "\n"), m.boxTarget())
}

func (m Model) listView() string {
	var rows []string
	indicator := styleDim.Render("filter  ")
	if m.filterInput.Value() != "" {
		indicator = styleTitle.Render("filter  ")
	}
	rows = append(rows, indicator+m.filterInput.View())
	rows = append(rows, m.table.View())
	rows = append(rows, styleDim.Render(fmt.Sprintf("%d/%d selected", len(m.selected), len(m.tools))))
	if m.onPath {
		rows = append(rows, styleDim.Render("Binaries go to ~/.startnow/bin"))
	} else {
		rows = append(rows, styleWarn.Render("~/.startnow/bin is NOT on PATH — export PATH=\"$HOME/.startnow/bin:$PATH\""))
	}
	if m.notice != "" {
		rows = append(rows, styleWarn.Render(m.notice))
	}
	if m.verInput.Focused() {
		for _, t := range m.tools {
			if t.Name == m.verName {
				rows = append(rows, "")
				rows = append(rows, styleTitle.Render("version for "+t.DisplayName)+"  "+m.verInput.View())
				break
			}
		}
	}
	return m.boxify("Tools", strings.Join(rows, "\n"), m.boxTarget())
}

const statusWidth = 30

func (m Model) machineView() string {
	inf := m.machine
	mem := machine.HumanBytes(inf.MemTotal)
	distro := inf.Distro.Pretty
	if distro == "" {
		distro = "unknown"
	}
	rows := [][2]string{
		{"Hostname", inf.Hostname},
		{"User", inf.User},
		{"Distro", distro},
		{"Kernel", inf.Kernel},
		{"Arch", inf.Arch},
		{"CPU", fmt.Sprintf("%dx %s", inf.CPUs, inf.CPUModel)},
		{"Memory", fmt.Sprintf("%s total / %s available", mem, machine.HumanBytes(inf.MemAvail))},
		{"Uptime", machine.HumanDuration(inf.Uptime)},
		{"Shell", inf.Shell},
		{"Pkg Mgr", m.pkgMgrName()},
		{"Sudo", m.env.Sudo},
		{"Go", inf.GoVer},
	}
	var info []string
	inner := m.widthOr() - 4
	logoLines := strings.Split(logo, "\n")
	logoW := 0
	for _, l := range logoLines {
		if w := lipgloss.Width(l); w > logoW {
			logoW = w
		}
	}
	valW := inner - logoW - 15
	if valW < 10 {
		valW = 10
	}
	for _, r := range rows {
		info = append(info, styleTitle.Render(pad(r[0], 10))+truncate(r[1], valW))
	}
	n := max(len(info), len(logoLines))
	var out []string
	for i := 0; i < n; i++ {
		var right string
		if i < len(info) {
			right = info[i]
		}
		left := ""
		if i < len(logoLines) {
			left = lipgloss.NewStyle().Foreground(logoGradient[i%len(logoGradient)]).Render(logoLines[i])
		}
		out = append(out, pad(left, logoW)+"  "+truncate(right, inner-logoW-3))
	}
	return m.boxify("Machine", strings.Join(out, "\n"), m.boxTarget())
}

func (m Model) pkgMgrName() string {
	if m.env.PkgMgr != nil {
		return m.env.PkgMgr.Name
	}
	if m.env.Distro.PackageManager != "" {
		return m.env.Distro.PackageManager
	}
	return "none"
}

func (m Model) usageView() string {
	u := m.usage
	var b strings.Builder
	b.WriteString(m.spinner.View())
	b.WriteString("  live · refreshing every second\n")
	cpu := bar(u.CPU/100, 18, accent)
	b.WriteString("CPU     ")
	b.WriteString(cpu)
	b.WriteString(fmt.Sprintf("   %d cores\n", m.machine.CPUs))
	if len(u.Cores) > 0 {
		for i := 0; i < len(u.Cores); i += 2 {
			b.WriteString("  core ")
			b.WriteString(bar(u.Cores[i]/100, 12, accent))
			b.WriteString(fmt.Sprintf(" %3.0f%%", u.Cores[i]))
			if i+1 < len(u.Cores) {
				b.WriteString("   core ")
				b.WriteString(bar(u.Cores[i+1]/100, 12, accent))
				b.WriteString(fmt.Sprintf(" %3.0f%%", u.Cores[i+1]))
			}
			b.WriteString("\n")
		}
	}
	var memPct float64
	if u.MemTotal > 0 {
		memPct = 100 - float64(u.MemAvail)/float64(u.MemTotal)*100
	}
	b.WriteString("Memory  ")
	b.WriteString(bar(memPct/100, 18, memColor(memPct)))
	b.WriteString(fmt.Sprintf("   %s used / %s total\n", machine.HumanBytes(u.MemTotal-u.MemAvail), machine.HumanBytes(u.MemTotal)))
	var swapPct float64
	if u.SwapTotal > 0 {
		swapPct = float64(u.SwapUsed) / float64(u.SwapTotal) * 100
	}
	b.WriteString("Swap    ")
	b.WriteString(bar(swapPct/100, 18, accent))
	b.WriteString(fmt.Sprintf("   %s used / %s total\n", machine.HumanBytes(u.SwapUsed), machine.HumanBytes(u.SwapTotal)))
	b.WriteString(fmt.Sprintf("Load    %.2f  %.2f  %.2f (1m 5m 15m)   %d procs (%d running)\n", u.Load1, u.Load5, u.Load15, u.ProcTotal, u.ProcRunning))
	b.WriteString(fmt.Sprintf("Net     ↓ %s/s   ↑ %s/s\n", machine.HumanBytes(uint64(u.NetRx)), machine.HumanBytes(uint64(u.NetTx))))
	if u.DiskTotal > 0 {
		diskPct := float64(u.DiskUsed) / float64(u.DiskTotal) * 100
		b.WriteString("Disk /  ")
		b.WriteString(bar(diskPct/100, 18, accent))
		b.WriteString(fmt.Sprintf("   %s used / %s total\n", machine.HumanBytes(u.DiskUsed), machine.HumanBytes(u.DiskTotal)))
	}
	b.WriteString("\n")
	b.WriteString(styleTitle.Render("Top processes by CPU"))
	b.WriteString("\n")
	if len(u.Procs) == 0 {
		b.WriteString("  —")
	}
	for _, p := range u.Procs {
		b.WriteString(fmt.Sprintf("  %6d  %5.1f%% cpu  %9s  %s\n", p.PID, p.CPU, machine.HumanBytes(p.Mem), truncate(p.Name, 24)))
	}
	return m.boxify("Usage", b.String(), m.boxTarget())
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

func (m Model) footer() string {
	keys := toolKeys()
	if m.tab != tabTools {
		keys = navKeys()
	}
	h := m.help
	h.Styles.ShortKey = lipgloss.NewStyle().Foreground(accent)
	h.Styles.ShortDesc = styleDim
	h.Styles.ShortSeparator = styleDim
	return "\n" + truncate(styleDim.Render(h.View(keys)), m.widthOr())
}

func (m Model) installView() string {
	var b strings.Builder
	inner := m.widthOr() - 4
	if inner < 10 {
		inner = 10
	}
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
			icon = m.spinner.View()
			running++
		}
		left := fmt.Sprintf("%s %s", icon, t.DisplayName)
		var rest string
		if j.status == jobFailed {
			rest = "failed: " + truncate(j.message, inner-16)
		} else {
			bar := bar(j.progress, 18, accent, colorOK)
			msg := styleDim.Render(truncate(j.message, inner-42))
			rest = fmt.Sprintf("%s  %s", bar, msg)
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
	b.WriteString(bar(overall, inner-14, accent, colorOK))
	b.WriteString("\n")
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
	target := 0
	if m.height > 0 {
		target = m.height - 2
	}
	return m.boxify("Installing", b.String(), target)
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
