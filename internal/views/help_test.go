package views

import (
	"strings"
	"testing"
)

func TestHelpViewRenderContainsSections(t *testing.T) {
	t.Parallel()
	shared := newTestShared()
	view := NewHelpView(shared)

	rendered := view.View(120, 40)

	expected := []string{
		"Core concepts",
		"Global keybindings",
		"Help bubble",
		"Bootstrap commands",
		"Troubleshooting",
	}
	for _, section := range expected {
		if !strings.Contains(rendered, section) {
			t.Errorf("View() missing section %q", section)
		}
	}
}

func TestHelpViewTextEntryFocusedAlwaysFalse(t *testing.T) {
	t.Parallel()
	shared := newTestShared()
	view := NewHelpView(shared)

	if view.TextEntryFocused() {
		t.Error("TextEntryFocused should always be false")
	}
}

func TestHelpViewSmallDimensions(t *testing.T) {
	t.Parallel()
	shared := newTestShared()
	view := NewHelpView(shared)

	// Zero dimensions should return empty
	rendered := view.View(0, 0)
	if rendered != "" {
		t.Errorf("View(0,0) should be empty, got %d chars", len(rendered))
	}

	// Negative dimensions should return empty
	rendered = view.View(-1, -1)
	if rendered != "" {
		t.Errorf("View(-1,-1) should be empty, got %d chars", len(rendered))
	}
}

func TestHelpViewUpdateIsNoop(t *testing.T) {
	t.Parallel()
	shared := newTestShared()
	view := NewHelpView(shared)

	// Help view ignores all messages
	updated, cmd := view.Update(keyPress("a", 'a'))
	_ = updated.(*HelpView)
	if cmd != nil {
		t.Error("Update should return nil command")
	}
}
