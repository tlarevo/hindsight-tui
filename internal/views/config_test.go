package views

import (
	"fmt"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"hindsight-tui/internal/config"
	"hindsight-tui/internal/domain"
)

// --- Pure functions ---

func TestValidateConfigValid(t *testing.T) {
	t.Parallel()
	cfg := config.Default()
	if err := validateConfig(cfg); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateConfigInvalidBackend(t *testing.T) {
	t.Parallel()
	cfg := config.Default()
	cfg.Backend = "bogus"
	err := validateConfig(cfg)
	if err == nil {
		t.Fatal("expected error for invalid backend")
	}
	if !strings.Contains(err.Error(), "invalid backend") {
		t.Fatalf("error = %q, want 'invalid backend'", err.Error())
	}
}

func TestValidateConfigInvalidTheme(t *testing.T) {
	t.Parallel()
	cfg := config.Default()
	cfg.Theme = "rainbow"
	err := validateConfig(cfg)
	if err == nil {
		t.Fatal("expected error for invalid theme")
	}
	if !strings.Contains(err.Error(), "invalid theme") {
		t.Fatalf("error = %q, want 'invalid theme'", err.Error())
	}
}

func TestValidateConfigEmptyAPIURL(t *testing.T) {
	t.Parallel()
	cfg := config.Default()
	cfg.APIURL = ""
	err := validateConfig(cfg)
	if err == nil {
		t.Fatal("expected error for empty api_url")
	}
	if !strings.Contains(err.Error(), "api_url is required") {
		t.Fatalf("error = %q, want 'api_url is required'", err.Error())
	}
}

func TestValidateConfigEmptyDefaultBank(t *testing.T) {
	t.Parallel()
	cfg := config.Default()
	cfg.DefaultBank = ""
	err := validateConfig(cfg)
	if err == nil {
		t.Fatal("expected error for empty default_bank")
	}
	if !strings.Contains(err.Error(), "default_bank is required") {
		t.Fatalf("error = %q, want 'default_bank is required'", err.Error())
	}
}

func TestValidateConfigZeroTimeout(t *testing.T) {
	t.Parallel()
	cfg := config.Default()
	cfg.TimeoutMS = 0
	err := validateConfig(cfg)
	if err == nil {
		t.Fatal("expected error for zero timeout_ms")
	}
	if !strings.Contains(err.Error(), "timeout_ms must be greater than zero") {
		t.Fatalf("error = %q, want 'timeout_ms must be greater than zero'", err.Error())
	}
}

// --- State transitions ---

func TestConfigSaveMsgWithError(t *testing.T) {
	t.Parallel()
	shared := newTestShared()
	view := NewConfigView(shared)

	updated, _ := view.Update(configSaveMsg{Err: fmt.Errorf("disk full")})
	v := updated.(*ConfigView)
	if v.statusTitle != "Save failed" {
		t.Errorf("statusTitle = %q, want 'Save failed'", v.statusTitle)
	}
	if v.loading {
		t.Error("loading should be false after save error")
	}
}

func TestConfigSaveMsgSuccess(t *testing.T) {
	t.Parallel()
	shared := newTestShared()
	view := NewConfigView(shared)

	newCfg := config.Default()
	newCfg.Backend = config.BackendDemo
	updated, _ := view.Update(configSaveMsg{
		Config: newCfg,
		Path:   "/tmp/config.yaml",
	})
	v := updated.(*ConfigView)
	if v.statusTitle != "Saved" {
		t.Errorf("statusTitle = %q, want 'Saved'", v.statusTitle)
	}
	if shared.Config.Backend != config.BackendDemo {
		t.Errorf("shared.Config.Backend = %q, want 'demo'", shared.Config.Backend)
	}
	if shared.State.Backend != config.BackendDemo {
		t.Errorf("shared.State.Backend = %q, want 'demo'", shared.State.Backend)
	}
}

func TestConfigActionResultMsg(t *testing.T) {
	t.Parallel()
	shared := newTestShared()
	view := NewConfigView(shared)

	updated, _ := view.Update(configActionResultMsg{
		Title: "Doctor",
		Body:  "All good",
	})
	v := updated.(*ConfigView)
	if v.statusTitle != "Doctor" {
		t.Errorf("statusTitle = %q, want 'Doctor'", v.statusTitle)
	}
	if v.statusBody != "All good" {
		t.Errorf("statusBody = %q, want 'All good'", v.statusBody)
	}
}

func TestConfigActionResultClearHealth(t *testing.T) {
	t.Parallel()
	shared := newTestShared()
	shared.Health = &domain.HealthStatus{OK: true}
	view := NewConfigView(shared)

	updated, _ := view.Update(configActionResultMsg{
		Title:       "Stop",
		ClearHealth: true,
	})
	_ = updated.(*ConfigView)
	if shared.Health != nil {
		t.Error("shared.Health should be nil after ClearHealth")
	}
}

func TestConfigCtrlSStartsSave(t *testing.T) {
	t.Parallel()
	shared := newTestShared()
	view := NewConfigView(shared)
	updated, cmd := view.Update(tea.KeyPressMsg(tea.Key{Mod: tea.ModCtrl, Code: 's'}))
	v := updated.(*ConfigView)
	if !v.loading {
		t.Error("loading should be true after Ctrl+S")
	}
	if cmd == nil {
		t.Fatal("expected command from save")
	}
}

func TestConfigCtrlRReloadsFields(t *testing.T) {
	t.Parallel()
	shared := newTestShared()
	view := NewConfigView(shared)

	// Modify a field value
	view.fields[0].value = "bogus"

	updated, _ := view.Update(tea.KeyPressMsg(tea.Key{Mod: tea.ModCtrl, Code: 'r'}))
	v := updated.(*ConfigView)
	if v.statusTitle != "Reloaded" {
		t.Errorf("statusTitle = %q, want 'Reloaded'", v.statusTitle)
	}
	// Field should be resynced from shared config
	if v.fields[0].value != string(shared.Config.Backend) {
		t.Errorf("field[0].value = %q, want %q", v.fields[0].value, shared.Config.Backend)
	}
}

func TestConfigPaneCycling(t *testing.T) {
	t.Parallel()
	shared := newTestShared()
	view := NewConfigView(shared)

	// Tab cycles forward through 3 panes
	for i := 0; i < 3; i++ {
		updated, _ := view.Update(keyPress("", tea.KeyTab))
		view = updated.(*ConfigView)
	}
	if view.pane != 0 {
		t.Errorf("pane = %d after 3 tabs, want 0", view.pane)
	}

	// Shift+Tab cycles backward
	updated, _ := view.Update(tea.KeyPressMsg(tea.Key{Mod: tea.ModShift, Code: tea.KeyTab}))
	view = updated.(*ConfigView)
	if view.pane != 2 {
		t.Errorf("pane = %d after shift-tab, want 2", view.pane)
	}
}

func TestConfigFormEditorStartStop(t *testing.T) {
	t.Parallel()
	shared := newTestShared()
	view := NewConfigView(shared)
	view.pane = configPaneForm

	// Enter starts editor
	updated, cmd := view.Update(keyPress("", tea.KeyEnter))
	view = updated.(*ConfigView)
	if !view.editor.active {
		t.Fatal("editor should be active after enter")
	}
	if cmd == nil {
		t.Fatal("expected command from editor.Start")
	}

	// Escape stops editor
	updated, _ = view.Update(keyPress("", tea.KeyEscape))
	view = updated.(*ConfigView)
	if view.editor.active {
		t.Fatal("editor should be inactive after escape")
	}
}
