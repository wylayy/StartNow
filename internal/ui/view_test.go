package ui

import (
	"strings"
	"testing"

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
	return m.update(tea.WindowSizeMsg{Width: 100, Height: 24})
}

func (m Model) update(msg tea.Msg) Model {
	model, _ := m.Update(msg)
	return model.(Model)
}

func TestListViewRenders(t *testing.T) {
	m := newTestModel(t)
	content := m.View().Content
	for _, want := range []string{"|____/", "Go", "Node.js", "Rust", "Bun", "LazyGit", "Tools", "Machine", "Usage"} {
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

func TestScrollAndSelect(t *testing.T) {
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
}

func TestVersionInput(t *testing.T) {
	m := newTestModel(t)
	m = m.update(tea.KeyPressMsg{Code: 'v'})
	if !m.verActive {
		t.Fatal("v should open version input")
	}
	if !strings.Contains(m.View().Content, "version for Go") {
		t.Error("version prompt not rendered")
	}
	m = m.update(tea.KeyPressMsg{Text: "1"})
	m = m.update(tea.KeyPressMsg{Text: "."})
	m = m.update(tea.KeyPressMsg{Text: "2"})
	m = m.update(tea.KeyPressMsg{Text: "6"})
	if m.verBuf != "1.26" {
		t.Fatalf("verBuf = %q", m.verBuf)
	}
	m = m.update(tea.KeyPressMsg{Code: '\b'})
	if m.verBuf != "1.2" {
		t.Fatalf("verBuf after backspace = %q", m.verBuf)
	}
	m = m.update(tea.KeyPressMsg{Code: ' '})
	if m.verBuf != "1.2 " {
		t.Errorf("space while typing should append, got %q", m.verBuf)
	}
	m = m.update(tea.KeyPressMsg{Code: '\b'})
	m = m.update(tea.KeyPressMsg{Code: '\r'})
	if m.verActive {
		t.Error("enter should confirm")
	}
	if got := m.version["go"]; got != "1.2" {
		t.Errorf("version = %q", got)
	}
	if !strings.Contains(m.View().Content, "→ 1.2") {
		t.Error("version badge not rendered")
	}
	m = m.update(tea.KeyPressMsg{Code: 'v'})
	if m.verBuf != "1.2" {
		t.Errorf("input should prefill current version, got %q", m.verBuf)
	}
	m = m.update(tea.KeyPressMsg{Code: '\x1b'})
	if m.verActive {
		t.Error("esc should cancel")
	}
	if got := m.version["go"]; got != "1.2" {
		t.Errorf("version changed after cancel: %q", got)
	}
}
