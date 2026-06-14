package app

import (
	stderrors "errors"
	"os"
	"path/filepath"
	"testing"

	"hindsight-tui/internal/config"
)

func TestParseBackend(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input     string
		want      config.Backend
		wantErr   bool
	}{
		{"embed", config.BackendEmbed, false},
		{"http", config.BackendHTTP, false},
		{"demo", config.BackendDemo, false},
		{" demo ", config.BackendDemo, false},
		{"bogus", "", true},
		{"", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			t.Parallel()
			got, err := parseBackend(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseBackend(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("parseBackend(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestLoadConfigForRunAppliesOverrides(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")
	content := "backend: embed\napi_url: http://seed:1\ndefault_bank: seed\ntheme: dark\ntimeout_ms: 1000\n"
	if err := os.WriteFile(cfgPath, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := loadConfigForRun(Options{
		ConfigPath:        cfgPath,
		BackendOverride:   "http",
		APIURLOverride:    "http://override:2",
		AuthTokenOverride: "tok",
	})
	if err != nil {
		t.Fatalf("loadConfigForRun error: %v", err)
	}
	if cfg.Backend != config.BackendHTTP {
		t.Errorf("Backend = %q, want %q", cfg.Backend, config.BackendHTTP)
	}
	if cfg.APIURL != "http://override:2" {
		t.Errorf("APIURL = %q, want %q", cfg.APIURL, "http://override:2")
	}
	if cfg.AuthToken != "tok" {
		t.Errorf("AuthToken = %q, want %q", cfg.AuthToken, "tok")
	}
	if cfg.DefaultBank != "seed" {
		t.Errorf("DefaultBank = %q, want %q", cfg.DefaultBank, "seed")
	}
}

func TestLoadConfigForRunInvalidBackendOverrideErrors(t *testing.T) {
	t.Parallel()
	_, err := loadConfigForRun(Options{BackendOverride: "bogus"})
	if err == nil {
		t.Fatal("expected error for invalid backend override, got nil")
	}
}

func TestLoadConfigForRunReadsAuthTokenFromEnv(t *testing.T) {
	// No t.Parallel() — uses t.Setenv which forbids parallel.

	t.Setenv("HINDSIGHT_TUI_AUTH_TOKEN", "envtok")

	// When AuthTokenOverride is empty, env var is used.
	cfg, err := loadConfigForRun(Options{})
	if err != nil {
		t.Fatalf("loadConfigForRun error: %v", err)
	}
	if cfg.AuthToken != "envtok" {
		t.Errorf("AuthToken = %q, want %q (from env)", cfg.AuthToken, "envtok")
	}

	// When AuthTokenOverride is set, it wins over the env var.
	cfg, err = loadConfigForRun(Options{AuthTokenOverride: "flagtok"})
	if err != nil {
		t.Fatalf("loadConfigForRun error: %v", err)
	}
	if cfg.AuthToken != "flagtok" {
		t.Errorf("AuthToken = %q, want %q (flag wins over env)", cfg.AuthToken, "flagtok")
	}
}

func TestLoadConfigForRunPropagatesMalformedConfig(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(cfgPath, []byte("backend: [\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := loadConfigForRun(Options{ConfigPath: cfgPath})
	if err == nil {
		t.Fatal("expected error for malformed config, got nil")
	}
	var malformed *config.MalformedConfigError
	if !stderrors.As(err, &malformed) {
		t.Errorf("error = %v, want errors.As to *config.MalformedConfigError", err)
	}
}
