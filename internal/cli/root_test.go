package cli

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"hindsight-tui/internal/app"
)

func TestFlagsMapToOptions(t *testing.T) {
	t.Parallel()
	var got app.Options
	cmd := newRootCmd(func(o app.Options) error {
		got = o
		return nil
	})
	cmd.SetArgs([]string{"--config", "/c", "--backend", "http", "--api-url", "http://x", "--auth-token", "t", "--doctor"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error: %v", err)
	}
	want := app.Options{
		ConfigPath:        "/c",
		BackendOverride:   "http",
		APIURLOverride:    "http://x",
		AuthTokenOverride: "t",
		Doctor:            true,
	}
	if got != want {
		t.Errorf("got %+v, want %+v", got, want)
	}
}

func TestDemoFlagOverridesBackend(t *testing.T) {
	t.Parallel()
	var got app.Options
	cmd := newRootCmd(func(o app.Options) error {
		got = o
		return nil
	})
	cmd.SetArgs([]string{"--backend", "http", "--demo"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error: %v", err)
	}
	if got.BackendOverride != "demo" {
		t.Errorf("BackendOverride = %q, want %q", got.BackendOverride, "demo")
	}
}

func TestNoFlagsLeavesEmptyOverrides(t *testing.T) {
	t.Parallel()
	var got app.Options
	cmd := newRootCmd(func(o app.Options) error {
		got = o
		return nil
	})
	cmd.SetArgs([]string{})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error: %v", err)
	}
	if got != (app.Options{}) {
		t.Errorf("got %+v, want empty Options{}", got)
	}
}

func TestRunErrorPropagates(t *testing.T) {
	t.Parallel()
	sentinel := errors.New("boom")
	cmd := newRootCmd(func(o app.Options) error {
		return sentinel
	})
	cmd.SetArgs([]string{})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error from Execute(), got nil")
	}
	if !errors.Is(err, sentinel) {
		t.Errorf("error = %v, want errors.Is(err, sentinel)", err)
	}
}

func TestVersionFlagPrintsBuildVersion(t *testing.T) {
	t.Parallel()
	var out bytes.Buffer
	called := false
	cmd := newRootCmd(func(o app.Options) error {
		called = true
		return nil
	})
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"--version"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error: %v", err)
	}
	if called {
		t.Fatal("run callback called for --version")
	}
	if !strings.Contains(out.String(), version) {
		t.Fatalf("--version output %q does not contain %q", out.String(), version)
	}
}
