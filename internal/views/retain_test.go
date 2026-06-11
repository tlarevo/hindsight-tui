package views

import "testing"

func TestRetainSubmitValidatesRequiredContent(t *testing.T) {
	t.Parallel()

	view := NewRetainView(newTestShared())
	view.bank.SetValue("default")
	view.content.SetValue("   ")

	cmd := view.submit()
	if cmd != nil {
		t.Fatal("submit() returned command for empty content")
	}
	if view.err == nil || view.err.Error() != "content is required" {
		t.Fatalf("err = %v, want content is required", view.err)
	}
}

func TestRetainSubmitValidatesMetadataLines(t *testing.T) {
	t.Parallel()

	view := NewRetainView(newTestShared())
	view.bank.SetValue("default")
	view.content.SetValue("Remember this")
	view.metadata.SetValue("broken-line")

	cmd := view.submit()
	if cmd != nil {
		t.Fatal("submit() returned command for invalid metadata")
	}
	if view.err == nil || view.err.Error() == "" {
		t.Fatal("submit() error = nil, want validation error")
	}
}
