package app

import (
	"testing"

	tea "charm.land/bubbletea/v2"

	"hindsight-tui/internal/config"
	"hindsight-tui/internal/hindsight"
	"hindsight-tui/internal/state"
	appviews "hindsight-tui/internal/views"
)

func demoModel() Model {
	cfg := config.Default()
	cfg.Backend = config.BackendDemo
	return NewModel(cfg, hindsight.NewDemoClient(), nil)
}

func keyPress(code rune, text string) tea.KeyPressMsg {
	return tea.KeyPressMsg{Code: code, Text: text}
}

func send(t *testing.T, m Model, msg tea.Msg) (Model, tea.Cmd) {
	t.Helper()
	next, cmd := m.Update(msg)
	model, ok := next.(Model)
	if !ok {
		t.Fatalf("Update returned %T, want Model", next)
	}
	return model, cmd
}

func TestGlobalNavBlockedDuringTextEntry(t *testing.T) {
	t.Parallel()

	m := demoModel()

	// 'g' on the dashboard (no text entry) navigates to Banks.
	probe, _ := send(t, m, keyPress('g', "g"))
	if probe.appState.CurrentView != state.RouteBanks {
		t.Fatalf("'g' on dashboard -> %v, want RouteBanks", probe.appState.CurrentView)
	}

	// 'r' opens Recall with the query input focused.
	m, _ = send(t, m, keyPress('r', "r"))
	if m.appState.CurrentView != state.RouteRecall {
		t.Fatalf("'r' -> %v, want RouteRecall", m.appState.CurrentView)
	}
	if !m.currentView().TextEntryFocused() {
		t.Fatal("recall query is not text-entry focused after navigation")
	}

	// 'g' while typing must NOT navigate; it is consumed by the query input.
	m, _ = send(t, m, keyPress('g', "g"))
	if m.appState.CurrentView != state.RouteRecall {
		t.Fatalf("'g' during text entry -> %v, want RouteRecall (no navigation)", m.appState.CurrentView)
	}
}

func TestHelpRoundTripDoesNotTrap(t *testing.T) {
	t.Parallel()

	m := demoModel()
	m, _ = send(t, m, keyPress('?', "?"))
	if m.appState.CurrentView != state.RouteHelp {
		t.Fatalf("'?' -> %v, want RouteHelp", m.appState.CurrentView)
	}
	// '?' while already on Help must not overwrite the remembered previous view.
	m, _ = send(t, m, keyPress('?', "?"))
	if m.appState.CurrentView != state.RouteHelp {
		t.Fatalf("'?' on Help -> %v, want RouteHelp", m.appState.CurrentView)
	}
	// 'q' returns to where we came from (Dashboard), not back into Help.
	m, _ = send(t, m, keyPress('q', "q"))
	if m.appState.CurrentView != state.RouteDashboard {
		t.Fatalf("'q' from Help -> %v, want RouteDashboard", m.appState.CurrentView)
	}
}

func TestCtrlRReachesFocusedView(t *testing.T) {
	t.Parallel()

	m := demoModel()
	m, _ = send(t, m, appviews.NavigateMsg{Route: state.RouteExplorer})
	if m.appState.CurrentView != state.RouteExplorer {
		t.Fatalf("navigate -> %v, want RouteExplorer", m.appState.CurrentView)
	}

	m, cmd := send(t, m, tea.KeyPressMsg{Code: 'r', Mod: tea.ModCtrl})
	if m.appState.CurrentView != state.RouteExplorer {
		t.Fatalf("ctrl+r changed route to %v, want RouteExplorer", m.appState.CurrentView)
	}
	if cmd == nil {
		t.Fatal("ctrl+r returned nil cmd, want a refresh command reaching Explorer")
	}
}
