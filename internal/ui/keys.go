package ui

import "charm.land/bubbles/v2/key"

type keyMap struct {
	Move, Toggle, SelectAll, Version, Update, Remove, Filter, Info, Install, Tab, Quit key.Binding
}

// ShortHelp returns the bindings that fit on one footer line; the rest are
// available via the full help (toggle with "?").
func (k keyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Move, k.Toggle, k.Filter, k.Install, k.Tab, k.Quit}
}

func (k keyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Move, k.Toggle, k.SelectAll, k.Version},
		{k.Update, k.Remove, k.Filter, k.Info},
		{k.Install, k.Tab, k.Quit},
	}
}

func toolKeys() keyMap {
	return keyMap{
		Move:      key.NewBinding(key.WithKeys("up", "k", "down", "j"), key.WithHelp("↑/↓", "move")),
		Toggle:    key.NewBinding(key.WithKeys("space"), key.WithHelp("space", "toggle")),
		SelectAll: key.NewBinding(key.WithKeys("a"), key.WithHelp("a", "select all")),
		Version:   key.NewBinding(key.WithKeys("v"), key.WithHelp("v", "version")),
		Update:    key.NewBinding(key.WithKeys("u"), key.WithHelp("u", "update")),
		Remove:    key.NewBinding(key.WithKeys("x"), key.WithHelp("x", "remove")),
		Filter:    key.NewBinding(key.WithKeys("/"), key.WithHelp("/", "filter")),
		Info:      key.NewBinding(key.WithKeys("i"), key.WithHelp("i", "details")),
		Install:   key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "install")),
		Tab:       key.NewBinding(key.WithKeys("tab"), key.WithHelp("tab", "switch tab")),
		Quit:      key.NewBinding(key.WithKeys("q", "ctrl+c"), key.WithHelp("q", "quit")),
	}
}

func navKeys() keyMap {
	return keyMap{
		Tab:  key.NewBinding(key.WithKeys("tab"), key.WithHelp("tab", "switch tab")),
		Quit: key.NewBinding(key.WithKeys("q", "ctrl+c"), key.WithHelp("q", "quit")),
	}
}
