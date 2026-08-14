package ui

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/table"
	"charm.land/lipgloss/v2"

	"startnow/internal/catalog"
)

func toolColumns() []table.Column {
	return []table.Column{
		{Title: " ", Width: 5},
		{Title: "Tool", Width: 18},
		{Title: "Category", Width: 10},
		{Title: "Status", Width: 30},
		{Title: "Support", Width: 8},
		{Title: "Description", Width: 20},
	}
}

func toolTableStyles() table.Styles {
	st := table.DefaultStyles()
	st.Header = lipgloss.NewStyle().Bold(true).Foreground(accent).Padding(0, 1)
	st.Cell = lipgloss.NewStyle().Padding(0, 1)
	st.Selected = lipgloss.NewStyle().Foreground(accent).Bold(true)
	return st
}

func newToolsTable() table.Model {
	return table.New(
		table.WithColumns(toolColumns()),
		table.WithRows(nil),
		table.WithWidth(80),
		table.WithHeight(10),
		table.WithFocused(true),
		table.WithStyles(toolTableStyles()),
	)
}

func (m Model) buildRows() []table.Row {
	rows := make([]table.Row, 0, len(m.rows))
	for i, t := range m.rows {
		check := " "
		if m.selected[t.Name] {
			check = "✓"
		}
		marker := " "
		if i == m.table.Cursor() {
			marker = ">"
		}
		name := t.DisplayName
		if v, ok := m.version[t.Name]; ok && v != "" {
			name += " → " + v
		}
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
		support := "support"
		if !t.Supported(m.env) {
			support = "unsupport"
		}
		rows = append(rows, table.Row{
			marker + "[" + check + "]",
			truncate(name, 18),
			t.Category,
			truncate(status, 30),
			support,
			truncate(t.Description, 20),
		})
	}
	return rows
}

func (m *Model) syncTableRows() {
	m.table.SetRows(m.buildRows())
}

// selectedTool returns the tool under the table cursor, if any.
func (m Model) selectedTool() (catalog.Tool, bool) {
	if len(m.rows) == 0 {
		return catalog.Tool{}, false
	}
	idx := m.table.Cursor()
	if idx < 0 || idx >= len(m.rows) {
		return catalog.Tool{}, false
	}
	return m.rows[idx], true
}

func (m Model) filterTools() []catalog.Tool {
	q := strings.ToLower(m.filterInput.Value())
	if q == "" {
		return m.tools
	}
	var out []catalog.Tool
	for _, t := range m.tools {
		hay := strings.ToLower(fmt.Sprintf("%s %s %s %s", t.DisplayName, t.Name, t.Description, t.Category))
		if strings.Contains(hay, q) {
			out = append(out, t)
		}
	}
	return out
}
