package views

import (
	"fmt"
	"testing"

	tea "charm.land/bubbletea/v2"

	"hindsight-tui/internal/domain"
)

func TestDashboardTextEntryFocusedAlwaysFalse(t *testing.T) {
	t.Parallel()
	shared := newTestShared()
	view := NewDashboardView(shared)

	if view.TextEntryFocused() {
		t.Error("TextEntryFocused should always be false")
	}
}


func TestDashboardLoadMsgSyncsShared(t *testing.T) {
	t.Parallel()
	shared := newTestShared()
	view := NewDashboardView(shared)

	health := &domain.HealthStatus{OK: true, Version: &domain.VersionInfo{APIVersion: "0.9.0"}}
	version := &domain.VersionInfo{APIVersion: "0.9.0"}
	banks := []domain.BankSummary{{BankID: "test-bank"}}
	stats := map[string]any{"memory_count": float64(100)}

	updated, _ := view.Update(dashboardLoadMsg{
		Health:  health,
		Version: version,
		Banks:   banks,
		Stats:   stats,
	})
	v := updated.(*DashboardView)

	if v.loading {
		t.Error("loading should be false after load")
	}
	if v.health != health {
		t.Error("health not synced to view")
	}
	if shared.Health != health {
		t.Error("health not synced to shared")
	}
	if v.version != version {
		t.Error("version not synced to view")
	}
	if shared.Version != version {
		t.Error("version not synced to shared")
	}
	if len(v.banks) != 1 {
		t.Errorf("banks len = %d, want 1", len(v.banks))
	}
	if v.stats["memory_count"] != float64(100) {
		t.Errorf("stats = %v", v.stats)
	}
}

func TestDashboardLoadMsgSetsActiveBank(t *testing.T) {
	t.Parallel()
	shared := newTestShared()
	view := NewDashboardView(shared)

	updated, _ := view.Update(dashboardLoadMsg{
		ActiveBank: "new-bank",
	})
	v := updated.(*DashboardView)

	if v.activeBank != "new-bank" {
		t.Errorf("activeBank = %q, want 'new-bank'", v.activeBank)
	}
	if shared.State.ActiveBank != "new-bank" {
		t.Errorf("shared.ActiveBank = %q, want 'new-bank'", shared.State.ActiveBank)
	}
}

func TestDashboardRefreshTriggersLoad(t *testing.T) {
	t.Parallel()
	shared := newTestShared()
	view := NewDashboardView(shared)
	updated, cmd := view.Update(tea.KeyPressMsg(tea.Key{Mod: tea.ModCtrl, Code: 'r'}))
	v := updated.(*DashboardView)
	if !v.loading {
		t.Error("loading should be true after refresh")
	}
	if cmd == nil {
		t.Fatal("expected load command from refresh")
	}
}

func TestDashboardLoadMsgWithError(t *testing.T) {
	t.Parallel()
	shared := newTestShared()
	view := NewDashboardView(shared)

	updated, _ := view.Update(dashboardLoadMsg{
		Err: fmt.Errorf("connection refused"),
	})
	v := updated.(*DashboardView)

	if v.loading {
		t.Error("loading should be false after error")
	}
	if v.err == nil {
		t.Fatal("expected error to be set")
	}
}
