package views

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	tea "charm.land/bubbletea/v2"

	"hindsight-tui/internal/domain"
)

func TestReflectCopyActionEmitsClipboardCommand(t *testing.T) {
	t.Parallel()

	view := NewReflectView(newTestShared())
	view.response = &domain.ReflectResponse{Text: "reflection text"}
	view.focus = reflectFocusResponse

	next, cmd := view.Update(keyPress("c", 'c'))
	if cmd == nil {
		t.Fatal("Update() returned nil command")
	}
	if next.(*ReflectView).status != "Sent reflection to terminal clipboard" {
		t.Fatalf("status = %q", next.(*ReflectView).status)
	}
	if got := fmt.Sprintf("%T", cmd()); got != "tea.setClipboardMsg" {
		t.Fatalf("command type = %s, want tea.setClipboardMsg", got)
	}
}

func TestReflectExportWritesResponseText(t *testing.T) {
	t.Parallel()

	view := NewReflectView(newTestShared())
	view.response = &domain.ReflectResponse{Text: "# Reflection\nKeep this."}
	target := filepath.Join(t.TempDir(), "reflection.md")
	view.exportPath.SetValue(target)

	cmd := view.exportReflection()
	if cmd == nil {
		t.Fatal("exportReflection() returned nil command")
	}

	msg := cmd()
	if _, ok := msg.(ExportedFileMsg); !ok {
		t.Fatalf("command returned %T, want ExportedFileMsg", msg)
	}

	data, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if string(data) != view.response.Text {
		t.Fatalf("file contents = %q, want %q", string(data), view.response.Text)
	}

	info, err := os.Stat(target)
	if err != nil {
		t.Fatalf("Stat() error = %v", err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("mode = %o, want 600", info.Mode().Perm())
	}
}

var _ tea.Msg = ExportedFileMsg{}
