package ui

import (
	"fmt"
	"runtime"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"startnow/internal/installer"
)

func newTestModel(t *testing.T) Model {
	t.Helper()
	env, err := installer.NewEnv(nil)
	if err != nil {
		t.Fatal(err)
	}
	m := NewModel(env)
	return m.update(tea.WindowSizeMsg{Width: 100, Height: 30})
}

func (m Model) update(msg tea.Msg) Model {
	model, cmd := m.Update(msg)
	return model.(Model).drain(cmd)
}

func (m Model) drain(cmd tea.Cmd) Model {
	model := m
	for i := 0; i < 5 && cmd != nil; i++ {
		msg := cmd()
		if batch, ok := msg.(tea.BatchMsg); ok {
			for _, sub := range batch {
				model = model.drain(sub)
			}
			return model
		}
		var next tea.Cmd
		nextModel, next := model.Update(msg)
		model = nextModel.(Model)
		cmd = next
	}
	return model
}

func TestListViewRenders(t *testing.T) {
	m := newTestModel(t)
	content := m.View().Content
	for _, want := range []string{"|____/", "Go", "Node.js", "Rust", "Bun", "LazyGit", "Ripgrep", "Htop", "Tools", "Machine", "Usage", "Category", "Support", "support", "Languages"} {
		if !strings.Contains(content, want) {
			t.Errorf("list view missing %q", want)
		}
	}
	if !strings.Contains(content, "not installed") {
		t.Error("expected at least one 'not installed' status")
	}
}

func TestTabSwitching(t *testing.T) {
	m := newTestModel(t)
	if m.tab != tabTools {
		t.Fatalf("initial tab = %d", m.tab)
	}
	m = m.update(tea.KeyPressMsg{Code: '2'})
	if m.tab != tabMachine {
		t.Fatalf("tab after '2' = %d", m.tab)
	}
	if !strings.Contains(m.View().Content, "Hostname") {
		t.Error("machine view missing Hostname")
	}
	m = m.update(tea.KeyPressMsg{Code: '3'})
	if !strings.Contains(m.View().Content, "Top processes") {
		t.Error("usage view missing top processes")
	}
	m = m.update(tea.KeyPressMsg{Code: '1'})
	if m.tab != tabTools {
		t.Fatalf("tab after '1' = %d", m.tab)
	}
}

func TestInstallViewRenders(t *testing.T) {
	m := newTestModel(t)
	m.screen = screenInstall
	m.jobs = map[string]*job{
		"go":   {status: jobRunning, progress: 0.5, message: "downloading go1.27.4.linux-amd64.tar.gz (30/70 MB)"},
		"node": {status: jobDone, progress: 1, message: "linked 3 binaries"},
		"rust": {status: jobFailed, progress: 0, message: "installer script failed: boom"},
	}
	content := m.View().Content
	for _, want := range []string{"Installing", "50%", "downloading", "✗", "failed"} {
		if !strings.Contains(content, want) {
			t.Errorf("install view missing %q", want)
		}
	}
}

func TestSelectAndListNav(t *testing.T) {
	m := newTestModel(t)
	m = m.update(tea.KeyPressMsg{Code: ' '})
	if !m.selected["go"] {
		t.Error("space should select the cursor row")
	}
	m = m.update(tea.KeyPressMsg{Code: 'j'})
	m = m.update(tea.KeyPressMsg{Code: ' '})
	if !m.selected["node"] {
		t.Error("down+space should select node")
	}
	m = m.update(tea.KeyPressMsg{Code: 'a'})
	if len(m.selected) != len(m.tools) {
		t.Errorf("select all: got %d of %d", len(m.selected), len(m.tools))
	}
	m = m.update(tea.KeyPressMsg{Code: 'a'})
	if len(m.selected) != 0 {
		t.Errorf("select all again should clear: got %d", len(m.selected))
	}
}

func TestTableFiltering(t *testing.T) {
	m := newTestModel(t)
	m = m.update(tea.KeyPressMsg{Code: '/'})
	if !m.filterInput.Focused() {
		t.Fatal("filter input should be focused after /")
	}
	m = m.update(tea.KeyPressMsg{Text: "lazygit"})
	if len(m.rows) != 1 {
		t.Fatalf("expected 1 filtered row, got %d", len(m.rows))
	}
	m = m.update(tea.KeyPressMsg{Code: '\r'})
	if m.filterInput.Focused() {
		t.Fatal("enter should blur the filter")
	}
	if len(m.rows) != 1 {
		t.Errorf("filter should stay applied after enter, got %d rows", len(m.rows))
	}
	m = m.update(tea.KeyPressMsg{Code: '/'})
	m = m.update(tea.KeyPressMsg{Code: '\x1b'})
	if m.filterInput.Focused() {
		t.Error("esc should blur the filter")
	}
	if len(m.rows) != len(m.tools) {
		t.Errorf("esc should clear the filter, got %d rows", len(m.rows))
	}
}

func TestFilterClick(t *testing.T) {
	m := newTestModel(t)
	// click the filter row inside the Tools box
	m = m.update(tea.MouseClickMsg{X: 30, Y: m.filterRowY(), Button: tea.MouseLeft})
	if !m.filterInput.Focused() {
		t.Fatal("clicking the filter row should focus the input")
	}
	m = m.update(tea.KeyPressMsg{Text: "bun"})
	if len(m.rows) != 1 || m.rows[0].Name != "bun" {
		t.Errorf("typing bun should filter to bun, got %d rows", len(m.rows))
	}
	// clicking elsewhere blurs
	m = m.update(tea.MouseClickMsg{X: 5, Y: m.headerHeight() + 3, Button: tea.MouseLeft})
	if m.filterInput.Focused() {
		t.Error("clicking outside should blur the filter")
	}
}

func TestUpdateRemoveUnmanaged(t *testing.T) {
	m := newTestModel(t)
	m = m.update(tea.KeyPressMsg{Code: 'u'})
	if m.notice == "" {
		t.Error("u on unmanaged tool should set a notice")
	}
	m = m.update(tea.KeyPressMsg{Code: 'x'})
	if m.notice == "" {
		t.Error("x on unmanaged tool should set a notice")
	}
	if m.managed["go"] {
		t.Error("go should not be managed")
	}
}

func TestNoticeClears(t *testing.T) {
	m := newTestModel(t)
	m = m.update(tea.KeyPressMsg{Code: 'x'})
	if m.notice == "" {
		t.Fatal("expected notice")
	}
	m = m.update(tea.KeyPressMsg{Code: 'j'})
	if m.notice != "" {
		t.Errorf("notice should clear on next key, got %q", m.notice)
	}
}

func TestTabClick(t *testing.T) {
	m := newTestModel(t)
	_, rects := m.tabLayout()
	// click the middle of the Machine pill
	for _, r := range rects {
		if r.tab == tabMachine {
			m = m.update(tea.MouseClickMsg{X: (r.x0 + r.x1) / 2, Y: (r.y0 + r.y1) / 2, Button: tea.MouseLeft})
			if m.tab != tabMachine {
				t.Fatalf("click on Machine should switch tab, got %d", m.tab)
			}
		}
	}
	// click outside any pill: no change
	m = m.update(tea.MouseClickMsg{X: 0, Y: 0, Button: tea.MouseLeft})
	if m.tab != tabMachine {
		t.Errorf("click outside should not change tab")
	}
	// right-click on Tools does nothing
	_, rects = m.tabLayout()
	for _, r := range rects {
		if r.tab == tabTools {
			m = m.update(tea.MouseClickMsg{X: (r.x0 + r.x1) / 2, Y: (r.y0 + r.y1) / 2, Button: tea.MouseRight})
			if m.tab != tabMachine {
				t.Errorf("right-click should not switch tabs")
			}
		}
	}
}

func TestUsageTick(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("linux only")
	}
	m := newTestModel(t)
	m = m.update(tea.KeyPressMsg{Code: '3'}) // usage tab samples on tick
	time.Sleep(30 * time.Millisecond)
	model, _ := m.Update(tickMsg{})
	m = model.(Model)
	time.Sleep(30 * time.Millisecond)
	model, _ = m.Update(tickMsg{})
	m = model.(Model)
	if m.usage.ProcTotal == 0 {
		t.Error("usage should be sampled on tick")
	}
	if len(m.usage.Cores) == 0 {
		t.Error("per-core usage should populate after two ticks")
	}
	if len(m.usage.Procs) == 0 {
		t.Error("top processes should be populated")
	}
}

func TestRightClickDetails(t *testing.T) {
	m := newTestModel(t)
	// seed the download cache so the click doesn't trigger a network resolve
	m.downloadURL["node"] = "https://nodejs.org/dist/v24.19.0/node-v24.19.0-linux-x64.tar.xz"
	// right-click the second table row
	m = m.update(tea.MouseClickMsg{X: 40, Y: m.headerHeight() + 3 + 1, Button: tea.MouseRight})
	if m.details == nil {
		t.Fatal("right-click on a row should open details")
	}
	if m.details.Name != "node" {
		t.Errorf("details for %q, want node", m.details.Name)
	}
	if !strings.Contains(m.View().Content, "Node.js — details") {
		t.Error("details panel not rendered")
	}
	if !strings.Contains(m.View().Content, "JavaScript runtime (LTS)") {
		t.Error("details should include the description")
	}
	if !strings.Contains(m.View().Content, "node-v24.19.0-linux-x64.tar.xz") {
		t.Error("details should show the resolved download URL")
	}
	// esc closes
	m = m.update(tea.KeyPressMsg{Code: '\x1b'})
	if m.details != nil {
		t.Error("esc should close details")
	}
	// 'i' key opens details for the selected row
	m = m.update(tea.KeyPressMsg{Code: 'i'})
	if m.details == nil || m.details.Name != "go" {
		t.Errorf("i should open details for selected tool, got %v", m.details)
	}
	// right-click off the rows closes
	m = m.update(tea.MouseClickMsg{X: 40, Y: m.headerHeight() + 1, Button: tea.MouseRight})
	if m.details != nil {
		t.Error("right-click outside rows should close details")
	}
}

func TestDownloadResolveMsg(t *testing.T) {
	m := newTestModel(t)
	m = m.update(downloadMsg{name: "go", url: "https://go.dev/dl/go1.27.4.linux-amd64.tar.gz"})
	if m.downloadURL["go"] != "https://go.dev/dl/go1.27.4.linux-amd64.tar.gz" {
		t.Errorf("download url not cached: %q", m.downloadURL["go"])
	}
	m = m.update(downloadMsg{name: "lazygit", err: fmt.Errorf("network down")})
	if !strings.Contains(m.downloadURL["lazygit"], "resolve failed") {
		t.Errorf("resolve error should be recorded, got %q", m.downloadURL["lazygit"])
	}
	if m.resolving["go"] {
		t.Error("resolving flag should be cleared")
	}
}

func TestVersionInput(t *testing.T) {
	m := newTestModel(t)
	m = m.update(tea.KeyPressMsg{Code: 'v'})
	if !m.verInput.Focused() {
		t.Fatal("v should open version input")
	}
	if !strings.Contains(m.View().Content, "version for Go") {
		t.Error("version prompt not rendered")
	}
	m = m.update(tea.KeyPressMsg{Text: "1"})
	m = m.update(tea.KeyPressMsg{Text: "."})
	m = m.update(tea.KeyPressMsg{Text: "2"})
	m = m.update(tea.KeyPressMsg{Text: "6"})
	if m.verInput.Value() != "1.26" {
		t.Fatalf("value = %q", m.verInput.Value())
	}
	m = m.update(tea.KeyPressMsg{Code: '\r'})
	if m.verInput.Focused() {
		t.Error("enter should confirm")
	}
	if got := m.version["go"]; got != "1.26" {
		t.Errorf("version = %q", got)
	}
	if !strings.Contains(m.View().Content, "→ 1.26") {
		t.Error("version badge not rendered")
	}
	m = m.update(tea.KeyPressMsg{Code: 'v'})
	if m.verInput.Value() != "1.26" {
		t.Errorf("input should prefill current version, got %q", m.verInput.Value())
	}
	m = m.update(tea.KeyPressMsg{Code: '\x1b'})
	if m.verInput.Focused() {
		t.Error("esc should cancel")
	}
	if got := m.version["go"]; got != "1.26" {
		t.Errorf("version changed after cancel: %q", got)
	}
}
