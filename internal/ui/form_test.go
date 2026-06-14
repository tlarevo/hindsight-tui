package ui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseTagsEmpty(t *testing.T) {
	t.Parallel()
	if got := ParseTags(""); got != nil {
		t.Fatalf("ParseTags(\"\") = %v, want nil", got)
	}
}

func TestParseTagsMultiple(t *testing.T) {
	t.Parallel()
	got := ParseTags("a,b,c")
	want := []string{"a", "b", "c"}
	if len(got) != len(want) {
		t.Fatalf("ParseTags returned %d tags, want %d", len(got), len(want))
	}
	for i, tag := range got {
		if tag != want[i] {
			t.Errorf("tag[%d] = %q, want %q", i, tag, want[i])
		}
	}
}

func TestParseTagsTrimsWhitespace(t *testing.T) {
	t.Parallel()
	got := ParseTags("a, ,b")
	want := []string{"a", "b"}
	if len(got) != len(want) {
		t.Fatalf("ParseTags returned %d tags, want %d", len(got), len(want))
	}
	for i, tag := range got {
		if tag != want[i] {
			t.Errorf("tag[%d] = %q, want %q", i, tag, want[i])
		}
	}
}

func TestParseTagsTrailingComma(t *testing.T) {
	t.Parallel()
	got := ParseTags("a,b,")
	want := []string{"a", "b"}
	if len(got) != len(want) {
		t.Fatalf("ParseTags returned %d tags, want %d", len(got), len(want))
	}
	for i, tag := range got {
		if tag != want[i] {
			t.Errorf("tag[%d] = %q, want %q", i, tag, want[i])
		}
	}
}

func TestParseMetadataLinesEmpty(t *testing.T) {
	t.Parallel()
	got, err := ParseMetadataLines("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != nil {
		t.Fatalf("ParseMetadataLines(\"\") = %v, want nil", got)
	}
}

func TestParseMetadataLinesValid(t *testing.T) {
	t.Parallel()
	got, err := ParseMetadataLines("key=value\nfoo=bar")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got["key"] != "value" || got["foo"] != "bar" {
		t.Fatalf("unexpected map: %v", got)
	}
}

func TestParseMetadataLinesBrokenLine(t *testing.T) {
	t.Parallel()
	_, err := ParseMetadataLines("broken-line")
	if err == nil {
		t.Fatal("expected error for broken line")
	}
	if !strings.Contains(err.Error(), "expected key=value") {
		t.Fatalf("error = %q, want substring 'expected key=value'", err.Error())
	}
}

func TestParseMetadataLinesEmptyKey(t *testing.T) {
	t.Parallel()
	_, err := ParseMetadataLines("=nokey")
	if err == nil {
		t.Fatal("expected error for empty key")
	}
	if !strings.Contains(err.Error(), "key is required") {
		t.Fatalf("error = %q, want substring 'key is required'", err.Error())
	}
}

func TestParseMetadataLinesSkipsBlankLines(t *testing.T) {
	t.Parallel()
	got, err := ParseMetadataLines("a=1\n\nb=2")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 entries, got %d: %v", len(got), got)
	}
}

func TestValidateBankIDEmpty(t *testing.T) {
	t.Parallel()
	err := ValidateBankID("")
	if err == nil {
		t.Fatal("expected error for empty bank_id")
	}
	if !strings.Contains(err.Error(), "bank_id is required") {
		t.Fatalf("error = %q, want 'bank_id is required'", err.Error())
	}
}

func TestValidateBankIDWhitespace(t *testing.T) {
	t.Parallel()
	err := ValidateBankID("   ")
	if err == nil {
		t.Fatal("expected error for whitespace-only bank_id")
	}
	if !strings.Contains(err.Error(), "bank_id is required") {
		t.Fatalf("error = %q, want 'bank_id is required'", err.Error())
	}
}

func TestValidateBankIDContainsWhitespace(t *testing.T) {
	t.Parallel()
	err := ValidateBankID("my bank")
	if err == nil {
		t.Fatal("expected error for bank_id with space")
	}
	if !strings.Contains(err.Error(), "must not contain whitespace") {
		t.Fatalf("error = %q, want 'must not contain whitespace'", err.Error())
	}
}

func TestValidateBankIDValid(t *testing.T) {
	t.Parallel()
	if err := ValidateBankID("valid-bank_1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestWritePrivateTextEmptyPath(t *testing.T) {
	t.Parallel()
	err := WritePrivateText("", []byte("content"))
	if err == nil {
		t.Fatal("expected error for empty path")
	}
	if !strings.Contains(err.Error(), "path is required") {
		t.Fatalf("error = %q, want 'path is required'", err.Error())
	}
}

func TestWritePrivateTextValid(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "secret.txt")
	content := []byte("hello world")

	if err := WritePrivateText(path, content); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read written file: %v", err)
	}
	if string(got) != string(content) {
		t.Fatalf("file content = %q, want %q", got, content)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat failed: %v", err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("file permissions = %o, want 0600", info.Mode().Perm())
	}
}
