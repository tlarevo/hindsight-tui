package views

import (
	"encoding/json"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"hindsight-tui/internal/domain"
)

// --- Existing test ---

func TestExplorerViewShowsAllTabs(t *testing.T) {
	t.Parallel()

	view := NewExplorerView(newTestShared())
	rendered := view.View(120, 40)

	for _, label := range []string{"Facts", "Entities", "Relationships", "Documents", "Tags"} {
		if !strings.Contains(rendered, label) {
			t.Fatalf("View() missing %q tab label", label)
		}
	}
}

// --- Tab switching ---

func TestExplorerTabSwitchRightAndLeft(t *testing.T) {
	t.Parallel()
	shared := newTestShared()
	view := NewExplorerView(shared)
	view.pt.FocusTable()

	if view.tab != explorerFacts {
		t.Fatalf("initial tab = %d, want explorerFacts", view.tab)
	}

	// right arrow → next tab
	updated, _ := view.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyRight}))
	v := updated.(*ExplorerView)
	if v.tab != explorerEntities {
		t.Errorf("tab = %d after right, want explorerEntities", v.tab)
	}

	// left arrow → prev tab
	updated, _ = v.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyLeft}))
	v = updated.(*ExplorerView)
	if v.tab != explorerFacts {
		t.Errorf("tab = %d after left, want explorerFacts", v.tab)
	}
}

func TestExplorerTabSwitchWraps(t *testing.T) {
	t.Parallel()
	shared := newTestShared()
	view := NewExplorerView(shared)
	view.pt.FocusTable()

	// Go backwards from facts (0) → should wrap to last tab (4 = Tags)
	updated, _ := view.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyLeft}))
	v := updated.(*ExplorerView)
	if v.tab != explorerTags {
		t.Errorf("tab = %d after left from facts, want explorerTags", v.tab)
	}

	// Go forward from Tags (4) → should wrap to Facts (0)
	updated, _ = v.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyRight}))
	v = updated.(*ExplorerView)
	if v.tab != explorerFacts {
		t.Errorf("tab = %d after right from tags, want explorerFacts", v.tab)
	}
}

func TestExplorerTabSwitchViaArrowKeys(t *testing.T) {
	t.Parallel()
	shared := newTestShared()
	view := NewExplorerView(shared)
	view.pt.FocusTable()

	// right arrow
	updated, _ := view.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyRight}))
	v := updated.(*ExplorerView)
	if v.tab != explorerEntities {
		t.Errorf("tab = %d after right arrow, want explorerEntities", v.tab)
	}

	// left arrow
	updated, _ = v.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyLeft}))
	v = updated.(*ExplorerView)
	if v.tab != explorerFacts {
		t.Errorf("tab = %d after left arrow, want explorerFacts", v.tab)
	}
}

// --- Messages ---

func TestExplorerLoadedMsgUnsupported(t *testing.T) {
	t.Parallel()
	shared := newTestShared()
	view := NewExplorerView(shared)
	view.pt.FocusTable()

	gen, _ := view.pt.BeginLoad()
	updated, _ := view.Update(explorerLoadedMsg{
		gen:         gen,
		tab:         explorerFacts,
		unsupported: "This Hindsight API version does not expose facts listing",
	})
	v := updated.(*ExplorerView)

	if v.notice != "This Hindsight API version does not expose facts listing" {
		t.Errorf("notice = %q", v.notice)
	}
	if v.rows != nil {
		t.Errorf("rows should be nil, got %d", len(v.rows))
	}
}

func TestExplorerLoadedMsgWithPage(t *testing.T) {
	t.Parallel()
	shared := newTestShared()
	view := NewExplorerView(shared)

	gen, _ := view.pt.BeginLoad()
	page := &domain.Page[map[string]any]{
		Items: []map[string]any{
			{"id": "f1", "type": "fact", "text": "hello"},
		},
		Total: 1,
	}
	updated, _ := view.Update(explorerLoadedMsg{
		gen:  gen,
		tab:  explorerFacts,
		page: page,
	})
	v := updated.(*ExplorerView)

	if len(v.rows) != 1 {
		t.Fatalf("rows len = %d, want 1", len(v.rows))
	}
	if v.rows[0]["id"] != "f1" {
		t.Errorf("rows[0][id] = %v", v.rows[0]["id"])
	}
}

func TestExplorerLoadedMsgStaleTabDiscarded(t *testing.T) {
	t.Parallel()
	shared := newTestShared()
	view := NewExplorerView(shared)

	// View is on Facts tab, send a message for Entities tab
	gen, _ := view.pt.BeginLoad()
	page := &domain.Page[map[string]any]{
		Items: []map[string]any{{"entity": "e1"}},
		Total: 1,
	}
	updated, _ := view.Update(explorerLoadedMsg{
		gen:  gen,
		tab:  explorerEntities,
		page: page,
	})
	v := updated.(*ExplorerView)

	// Rows should not be updated (wrong tab)
	if v.rows != nil {
		t.Errorf("rows should be nil for stale tab, got %d", len(v.rows))
	}
}

func TestExplorerCopyKeyOnTableTab(t *testing.T) {
	t.Parallel()
	shared := newTestShared()
	view := NewExplorerView(shared)
	view.pt.FocusTable()

	updated, cmd := view.Update(keyPress("c", 'c'))
	v := updated.(*ExplorerView)
	if v.notice != "Copied JSON to clipboard" {
		t.Errorf("notice = %q, want 'Copied JSON to clipboard'", v.notice)
	}
	if cmd == nil {
		t.Fatal("expected clipboard command")
	}
}

// --- Pure functions ---

func TestExplorerRowsFactsTab(t *testing.T) {
	t.Parallel()
	rows := []map[string]any{
		{"id": "f1", "type": "observation", "tags": []any{"tag1", "tag2"}, "text": "some fact"},
	}
	result := explorerRows(explorerFacts, rows)
	if len(result) != 1 {
		t.Fatalf("expected 1 row, got %d", len(result))
	}
	if result[0][0] != "f1" {
		t.Errorf("col[0] = %v, want 'f1'", result[0][0])
	}
	if result[0][1] != "observation" {
		t.Errorf("col[1] = %v, want 'observation'", result[0][1])
	}
	if result[0][3] != "some fact" {
		t.Errorf("col[3] = %v, want 'some fact'", result[0][3])
	}
}

func TestExplorerRowsEntitiesTab(t *testing.T) {
	t.Parallel()
	rows := []map[string]any{
		{"entity": "Alice", "type": "person", "count": 5, "memory_ids": []any{"m1", "m2"}},
	}
	result := explorerRows(explorerEntities, rows)
	if len(result) != 1 {
		t.Fatalf("expected 1 row, got %d", len(result))
	}
	if result[0][0] != "Alice" {
		t.Errorf("col[0] = %v, want 'Alice'", result[0][0])
	}
	if result[0][1] != "person" {
		t.Errorf("col[1] = %v, want 'person'", result[0][1])
	}
}

func TestExplorerRowsDocumentsTab(t *testing.T) {
	t.Parallel()
	rows := []map[string]any{
		{"document_id": "d1", "title": "My Doc", "tags": "a,b", "updated_at": "2025-01-01"},
	}
	result := explorerRows(explorerDocuments, rows)
	if len(result) != 1 {
		t.Fatalf("expected 1 row, got %d", len(result))
	}
	if result[0][0] != "d1" {
		t.Errorf("col[0] = %v, want 'd1'", result[0][0])
	}
	if result[0][1] != "My Doc" {
		t.Errorf("col[1] = %v, want 'My Doc'", result[0][1])
	}
}

func TestExplorerRowsTagsTab(t *testing.T) {
	t.Parallel()
	rows := []map[string]any{
		{"tag": "important", "count": 10, "source": "manual", "sample": "example text"},
	}
	result := explorerRows(explorerTags, rows)
	if len(result) != 1 {
		t.Fatalf("expected 1 row, got %d", len(result))
	}
	if result[0][0] != "important" {
		t.Errorf("col[0] = %v, want 'important'", result[0][0])
	}
}

func TestFirstNonEmpty(t *testing.T) {
	t.Parallel()
	row := map[string]any{"a": "", "b": "value", "c": "other"}
	got := firstNonEmpty(row, "a", "b", "c")
	if got != "value" {
		t.Errorf("firstNonEmpty = %q, want 'value'", got)
	}
}

func TestFirstNonEmptyAllEmpty(t *testing.T) {
	t.Parallel()
	row := map[string]any{"a": "", "b": ""}
	got := firstNonEmpty(row, "a", "b")
	if got != "" {
		t.Errorf("firstNonEmpty = %q, want ''", got)
	}
}

func TestFirstNonEmptyMissingKey(t *testing.T) {
	t.Parallel()
	row := map[string]any{"x": "val"}
	got := firstNonEmpty(row, "missing", "x")
	if got != "val" {
		t.Errorf("firstNonEmpty = %q, want 'val'", got)
	}
}

func TestFirstExistingFound(t *testing.T) {
	t.Parallel()
	row := map[string]any{"a": "val-a", "b": "val-b"}
	got := firstExisting(row, "a", "b")
	if got != "val-a" {
		t.Errorf("firstExisting = %v, want 'val-a'", got)
	}
}

func TestFirstExistingEmptyValue(t *testing.T) {
	t.Parallel()
	row := map[string]any{"a": "", "b": "val-b"}
	got := firstExisting(row, "a", "b")
	// firstExisting returns first EXISTING key, even if empty
	if got != "" {
		t.Errorf("firstExisting = %v, want '' (empty string exists)", got)
	}
}

func TestFirstExistingNone(t *testing.T) {
	t.Parallel()
	row := map[string]any{"x": "val"}
	got := firstExisting(row, "a", "b")
	if got != nil {
		t.Errorf("firstExisting = %v, want nil", got)
	}
}

func TestGraphCountsValid(t *testing.T) {
	t.Parallel()
	raw := json.RawMessage(`{"nodes":[{"id":"n1"},{"id":"n2"}],"edges":[{"source":"n1","target":"n2"}]}`)
	nodes, edges := graphCounts(raw)
	if nodes != 2 {
		t.Errorf("nodes = %d, want 2", nodes)
	}
	if edges != 1 {
		t.Errorf("edges = %d, want 1", edges)
	}
}

func TestGraphCountsEmpty(t *testing.T) {
	t.Parallel()
	nodes, edges := graphCounts(nil)
	if nodes != 0 || edges != 0 {
		t.Errorf("nodes=%d edges=%d, want 0,0", nodes, edges)
	}
}

func TestGraphCountsInvalid(t *testing.T) {
	t.Parallel()
	nodes, edges := graphCounts(json.RawMessage(`not json`))
	if nodes != 0 || edges != 0 {
		t.Errorf("nodes=%d edges=%d, want 0,0", nodes, edges)
	}
}

func TestUnsupportedListingMessage(t *testing.T) {
	t.Parallel()
	msg := unsupportedListingMessage("facts")
	want := "This Hindsight API version does not expose facts listing"
	if msg != want {
		t.Errorf("unsupportedListingMessage = %q, want %q", msg, want)
	}
}
