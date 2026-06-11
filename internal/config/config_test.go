package config

import (
	stderrors "errors"
	"os"
	"path/filepath"
	"testing"
)

func TestLoadMissingReturnsDefaults(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "missing.yaml")

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg != Default() {
		t.Fatalf("Load() = %#v, want %#v", cfg, Default())
	}
}

func TestLoadMalformedYAMLReturnsError(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte("backend: ["), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	_, err := Load(path)
	if err == nil {
		t.Fatal("Load() error = nil, want malformed config error")
	}

	var malformed *MalformedConfigError
	if !stderrors.As(err, &malformed) {
		t.Fatalf("Load() error = %T, want *MalformedConfigError", err)
	}
}

func TestSaveLoadRoundTrip(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "config.yaml")
	want := Config{
		Backend:     BackendHTTP,
		APIURL:      "http://localhost:9999",
		DefaultBank: "research",
		Theme:       "light",
		Compact:     true,
		TimeoutMS:   1234,
	}

	if err := Save(path, want); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	loaded, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if loaded != want {
		t.Fatalf("Load() = %#v, want %#v", loaded, want)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat() error = %v", err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("mode = %o, want 600", info.Mode().Perm())
	}
}

func TestLoadInvalidBackendReturnsError(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte("backend: bogus\n"), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	_, err := Load(path)
	if err == nil {
		t.Fatal("Load() error = nil, want invalid backend error")
	}
	var malformed *MalformedConfigError
	if !stderrors.As(err, &malformed) {
		t.Fatalf("Load() error = %T, want *MalformedConfigError", err)
	}
}
