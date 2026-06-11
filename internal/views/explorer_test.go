package views

import (
	"strings"
	"testing"
)

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
