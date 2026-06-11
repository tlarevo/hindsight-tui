package keymap

import "charm.land/bubbles/v2/key"

type KeyMap struct {
	Quit     key.Binding
	Back     key.Binding
	NextPane key.Binding
	PrevPane key.Binding
	Down     key.Binding
	Up       key.Binding
	Select   key.Binding
	Search   key.Binding
	Help     key.Binding
	Refresh  key.Binding
	Save     key.Binding
	Banks    key.Binding
	Recall   key.Binding
	Retain   key.Binding
	Reflect  key.Binding
	Advanced key.Binding
	Copy     key.Binding
	Export   key.Binding
}

func Default() KeyMap {
	return KeyMap{
		Quit:     key.NewBinding(key.WithKeys("ctrl+c"), key.WithHelp("ctrl+c", "quit")),
		Back:     key.NewBinding(key.WithKeys("q"), key.WithHelp("q", "back")),
		NextPane: key.NewBinding(key.WithKeys("tab"), key.WithHelp("tab", "next pane")),
		PrevPane: key.NewBinding(key.WithKeys("shift+tab"), key.WithHelp("shift+tab", "prev pane")),
		Down:     key.NewBinding(key.WithKeys("j", "down"), key.WithHelp("j/↓", "down")),
		Up:       key.NewBinding(key.WithKeys("k", "up"), key.WithHelp("k/↑", "up")),
		Select:   key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "select")),
		Search:   key.NewBinding(key.WithKeys("/"), key.WithHelp("/", "search")),
		Help:     key.NewBinding(key.WithKeys("?"), key.WithHelp("?", "help")),
		Refresh:  key.NewBinding(key.WithKeys("ctrl+r"), key.WithHelp("ctrl+r", "refresh")),
		Save:     key.NewBinding(key.WithKeys("ctrl+s"), key.WithHelp("ctrl+s", "save")),
		Banks:    key.NewBinding(key.WithKeys("g"), key.WithHelp("g", "banks")),
		Recall:   key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "recall")),
		Retain:   key.NewBinding(key.WithKeys("R"), key.WithHelp("R", "retain")),
		Reflect:  key.NewBinding(key.WithKeys("f"), key.WithHelp("f", "reflect")),
		Advanced: key.NewBinding(key.WithKeys("a"), key.WithHelp("a", "advanced")),
		Copy:     key.NewBinding(key.WithKeys("c"), key.WithHelp("c", "copy")),
		Export:   key.NewBinding(key.WithKeys("e"), key.WithHelp("e", "export")),
	}
}

func (k KeyMap) ShortHelp() []key.Binding {
	return []key.Binding{
		k.Help,
		k.Banks,
		k.Recall,
		k.Retain,
		k.Reflect,
		k.Refresh,
		k.Back,
		k.Quit,
	}
}

func (k KeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Help, k.Refresh, k.Save, k.Select, k.Back, k.Quit},
		{k.Banks, k.Recall, k.Retain, k.Reflect, k.Search, k.Advanced},
		{k.Copy, k.Export, k.NextPane, k.PrevPane, k.Down, k.Up},
	}
}
