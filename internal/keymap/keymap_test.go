package keymap

import (
	"fmt"
	"testing"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
)

func TestDefaultBindingsMatchExpectedKeys(t *testing.T) {
	t.Parallel()
	km := Default()

	tests := []struct {
		name      string
		event     tea.KeyPressMsg
		binding   key.Binding
		wantMatch bool
	}{
		// Printable keys
	{"s matches FocusSidebar", tea.KeyPressMsg(tea.Key{Text: "s", Code: 's'}), km.FocusSidebar, true},
		{"f matches Reflect", tea.KeyPressMsg(tea.Key{Text: "f", Code: 'f'}), km.Reflect, true},
		{"? matches Help", tea.KeyPressMsg(tea.Key{Text: "?", Code: '?'}), km.Help, true},
		{"a matches Advanced", tea.KeyPressMsg(tea.Key{Text: "a", Code: 'a'}), km.Advanced, true},
		{"c matches Copy", tea.KeyPressMsg(tea.Key{Text: "c", Code: 'c'}), km.Copy, true},
		{"e matches Export", tea.KeyPressMsg(tea.Key{Text: "e", Code: 'e'}), km.Export, true},
		{"/ matches Search", tea.KeyPressMsg(tea.Key{Text: "/", Code: '/'}), km.Search, true},
		// Special keys
		{"enter matches Select", tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}), km.Select, true},
		{"tab matches NextPane", tea.KeyPressMsg(tea.Key{Code: tea.KeyTab}), km.NextPane, true},
		{"shift+tab matches PrevPane", tea.KeyPressMsg(tea.Key{Mod: tea.ModShift, Code: tea.KeyTab}), km.PrevPane, true},
		// Modifier keys
		{"ctrl+c matches Quit", tea.KeyPressMsg(tea.Key{Mod: tea.ModCtrl, Code: 'c'}), km.Quit, true},
		{"ctrl+r matches Refresh", tea.KeyPressMsg(tea.Key{Mod: tea.ModCtrl, Code: 'r'}), km.Refresh, true},
		{"ctrl+s matches Save", tea.KeyPressMsg(tea.Key{Mod: tea.ModCtrl, Code: 's'}), km.Save, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := key.Matches(tt.event, tt.binding)
			if got != tt.wantMatch {
				t.Errorf("key.Matches = %v, want %v", got, tt.wantMatch)
			}
		})
	}
}

func TestBindingKeysAreUnique(t *testing.T) {
	t.Parallel()
	km := Default()

	// All 16 fields of KeyMap. Down/Up listed explicitly alongside others.
	// (which share keys with nothing else). Explicit list of (name, keys).
	type bindingInfo struct {
		name string
		keys []string
	}
	bindings := []bindingInfo{
		{"FocusSidebar", km.FocusSidebar.Keys()},
		{"Quit", km.Quit.Keys()},
		{"Back", km.Back.Keys()},
		{"NextPane", km.NextPane.Keys()},
		{"PrevPane", km.PrevPane.Keys()},
		{"Down", km.Down.Keys()},
		{"Up", km.Up.Keys()},
		{"Select", km.Select.Keys()},
		{"Search", km.Search.Keys()},
		{"Help", km.Help.Keys()},
		{"Refresh", km.Refresh.Keys()},
		{"Save", km.Save.Keys()},
		{"Reflect", km.Reflect.Keys()},
		{"Advanced", km.Advanced.Keys()},
		{"Copy", km.Copy.Keys()},
		{"Export", km.Export.Keys()},
	}

	seen := make(map[string]string) // key string → binding name
	for _, b := range bindings {
		for _, k := range b.keys {
			if prev, ok := seen[k]; ok {
				t.Errorf("duplicate key %q in bindings %q and %q", k, prev, b.name)
			}
			seen[k] = b.name
		}
	}
}

func TestShortHelpAndFullHelpShape(t *testing.T) {
	t.Parallel()
	km := Default()
	short := km.ShortHelp()
	if got := len(short); got != 6 {
		t.Errorf("ShortHelp() returned %d bindings, want 6", got)
	}
	for i, b := range short {
		if b.Help().Key == "" {
			t.Errorf("ShortHelp()[%d].Help().Key is empty", i)
		}
		if b.Help().Desc == "" {
			t.Errorf("ShortHelp()[%d].Help().Desc is empty", i)
		}
	}

	full := km.FullHelp()
	if got := len(full); got != 3 {
		t.Fatalf("FullHelp() returned %d rows, want 3", got)
	}
	for i, row := range full {
		if got := len(row); got == 0 {
			t.Errorf("FullHelp()[%d] has 0 bindings, want > 0", i)
		}
		for j, b := range row {
			if b.Help().Key == "" {
				t.Errorf("FullHelp()[%d][%d].Help().Key is empty", i, j)
			}
			if b.Help().Desc == "" {
				t.Errorf("FullHelp()[%d][%d].Help().Desc is empty", i, j)
			}
		}
	}
}

// Ensure fmt.Stringer is satisfied for key.Matches constraint.
var _ fmt.Stringer = tea.KeyPressMsg{}
