package views

import (
	"fmt"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/tlarevo/hindsight-tui/internal/domain"
)

func TestOperationsLoadedWithError(t *testing.T) {
	t.Parallel()
	shared := newTestShared()
	view := NewOperationsView(shared)

	gen, _ := view.pt.BeginLoad()

	updated, _ := view.Update(operationsLoadedMsg{gen: gen, err: fmt.Errorf("timeout")})
	v := updated.(*OperationsView)
	if v.err == nil {
		t.Fatal("expected error to be set")
	}
	if !strings.Contains(v.err.Error(), "timeout") {
		t.Errorf("err = %q, want 'timeout'", v.err.Error())
	}
}

func TestOperationsLoadedWithPage(t *testing.T) {
	t.Parallel()
	shared := newTestShared()
	view := NewOperationsView(shared)

	gen, _ := view.pt.BeginLoad()
	page := &domain.Page[map[string]any]{
		Items: []map[string]any{
			{"id": "op-1", "status": "completed", "type": "retain"},
			{"id": "op-2", "status": "pending", "type": "recall"},
		},
		Total: 2,
	}

	updated, _ := view.Update(operationsLoadedMsg{gen: gen, page: page})
	v := updated.(*OperationsView)
	if v.err != nil {
		t.Errorf("unexpected error: %v", v.err)
	}
	if len(v.rows) != 2 {
		t.Fatalf("rows len = %d, want 2", len(v.rows))
	}
	if v.rows[0]["id"] != "op-1" {
		t.Errorf("rows[0][id] = %v, want 'op-1'", v.rows[0]["id"])
	}
}

func TestOperationsCopyKeyWhenTableFocused(t *testing.T) {
	t.Parallel()
	shared := newTestShared()
	view := NewOperationsView(shared)

	// Move focus to the table so TextEntryFocused() is false
	view.pt.FocusTable()

	updated, cmd := view.Update(keyPress("c", 'c'))
	v := updated.(*OperationsView)
	if v.notice != "Copied JSON to clipboard" {
		t.Errorf("notice = %q, want 'Copied JSON to clipboard'", v.notice)
	}
	if cmd == nil {
		t.Fatal("expected clipboard command")
	}
}

func TestOperationsCopyKeyIgnoredWhenInputFocused(t *testing.T) {
	t.Parallel()
	shared := newTestShared()
	view := NewOperationsView(shared)

	// Focus is on first input by default, so copy should be ignored
	updated, _ := view.Update(keyPress("c", 'c'))
	v := updated.(*OperationsView)
	if v.notice != "" {
		t.Errorf("notice should be empty when input focused, got %q", v.notice)
	}
}

func TestOperationsRefreshKey(t *testing.T) {
	t.Parallel()
	shared := newTestShared()
	view := NewOperationsView(shared)

	updated, cmd := view.Update(tea.KeyPressMsg(tea.Key{Mod: tea.ModCtrl, Code: 'r'}))
	_ = updated.(*OperationsView)
	if cmd == nil {
		t.Fatal("expected load command from refresh")
	}
}

func TestOperationsPaginationNext(t *testing.T) {
	t.Parallel()
	shared := newTestShared()
	view := NewOperationsView(shared)

	// Load data with total > limit so CanNext() returns true
	gen, _ := view.pt.BeginLoad()
	page := &domain.Page[map[string]any]{
		Items: []map[string]any{{"id": "op-1"}},
		Total: 50,
	}
	updated, _ := view.Update(operationsLoadedMsg{gen: gen, page: page})
	view = updated.(*OperationsView)

	// Move focus to table first (pagination key handling doesn't check focus,
	// but the ] key handler runs before the focus check)
	updated, cmd := view.Update(keyPress("]", ']'))
	_ = updated.(*OperationsView)
	if cmd == nil {
		t.Fatal("expected load command from next page")
	}
}

func TestOperationsPaginationPrev(t *testing.T) {
	t.Parallel()
	shared := newTestShared()
	view := NewOperationsView(shared)

	// Load data and move to page 2
	gen, _ := view.pt.BeginLoad()
	page := &domain.Page[map[string]any]{
		Items: []map[string]any{{"id": "op-1"}},
		Total: 50,
	}
	updated, _ := view.Update(operationsLoadedMsg{gen: gen, page: page})
	view = updated.(*OperationsView)
	view.pt.NextPage()

	// [ key should go back to page 1
	updated, cmd := view.Update(keyPress("[", '['))
	_ = updated.(*OperationsView)
	if cmd == nil {
		t.Fatal("expected load command from prev page")
	}
}

func TestOperationsStaleGenerationDiscarded(t *testing.T) {
	t.Parallel()
	shared := newTestShared()
	view := NewOperationsView(shared)

	gen, _ := view.pt.BeginLoad()
	view.pt.BeginLoad() // gen=2, making gen=1 stale

	page := &domain.Page[map[string]any]{
		Items: []map[string]any{{"id": "op-stale"}},
		Total: 1,
	}
	updated, _ := view.Update(operationsLoadedMsg{gen: gen, page: page})
	v := updated.(*OperationsView)

	if len(v.rows) != 0 {
		t.Errorf("rows should be empty for stale gen, got %d", len(v.rows))
	}
}

func TestOperationsTextEntryFocusedInitially(t *testing.T) {
	t.Parallel()
	shared := newTestShared()
	view := NewOperationsView(shared)

	// Focus starts on first input (index 0), so TextEntryFocused should be true
	if !view.TextEntryFocused() {
		t.Error("TextEntryFocused should be true initially (input focused)")
	}

	// After focusing the table, it should be false
	view.pt.FocusTable()
	if view.TextEntryFocused() {
		t.Error("TextEntryFocused should be false when table focused")
	}
}
