package app

import (
	"strings"
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
	return NewModel(cfg, hindsight.NewDemoClient(), nil, false)
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

func TestSidebarFocusedOnRouteSwitch(t *testing.T) {
	t.Parallel()

	m := demoModel()
	m, _ = send(t, m, tea.WindowSizeMsg{Width: 120, Height: 40})

	// Navigate via NavigateMsg — sidebar should be focused.
	m, _ = send(t, m, appviews.NavigateMsg{Route: state.RouteBanks})
	if !m.sidebarFocused {
		t.Fatal("sidebar not focused after NavigateMsg")
	}
	if m.appState.CurrentView != state.RouteBanks {
		t.Fatalf("route = %v, want RouteBanks", m.appState.CurrentView)
	}
}

func TestHelpRoundTripDoesNotTrap(t *testing.T) {
	t.Parallel()

	m := demoModel()
	m, _ = send(t, m, tea.WindowSizeMsg{Width: 120, Height: 40})
	// Sidebar is focused by default. '?' should still reach Help.
	m, _ = send(t, m, keyPress('?', "?"))
	if m.appState.CurrentView != state.RouteHelp {
		t.Fatalf("'?' -> %v, want RouteHelp", m.appState.CurrentView)
	}
	// switchRoute refocuses sidebar. Esc to unfocus for next key.
	m, _ = send(t, m, tea.KeyPressMsg{Code: tea.KeyEscape})
	// '?' while already on Help re-navigates to Help.
	m, _ = send(t, m, keyPress('?', "?"))
	m, _ = send(t, m, tea.KeyPressMsg{Code: tea.KeyEscape})
	if m.appState.CurrentView != state.RouteHelp {
		t.Fatalf("'?' on Help -> %v, want RouteHelp", m.appState.CurrentView)
	}
	// 'q' returns to where we came from (Dashboard).
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
	// Unfocus sidebar so ctrl+r reaches the view.
	m, _ = send(t, m, tea.KeyPressMsg{Code: tea.KeyEscape})

	m, cmd := send(t, m, tea.KeyPressMsg{Code: 'r', Mod: tea.ModCtrl})
	if m.appState.CurrentView != state.RouteExplorer {
		t.Fatalf("ctrl+r changed route to %v, want RouteExplorer", m.appState.CurrentView)
	}
	if cmd == nil {
		t.Fatal("ctrl+r returned nil cmd, want a refresh command reaching Explorer")
	}
}

func TestSidebarFocusAndNavigate(t *testing.T) {
	t.Parallel()

	m := demoModel()
	m, _ = send(t, m, tea.WindowSizeMsg{Width: 120, Height: 40})

	// NavigateMsg focuses sidebar at Banks (index 1).
	m, _ = send(t, m, appviews.NavigateMsg{Route: state.RouteBanks})
	if !m.sidebarFocused {
		t.Fatal("sidebar not focused after NavigateMsg")
	}

	startIdx := m.sidebar.Index()

	// Down arrow moves selection.
	m, _ = send(t, m, tea.KeyPressMsg{Code: tea.KeyDown})
	if m.sidebar.Index() != startIdx+1 {
		t.Fatalf("after Down: index %d, want %d", m.sidebar.Index(), startIdx+1)
	}

	// Enter navigates to the selected route (one below Banks = Retain).
	m, _ = send(t, m, keyPress(tea.KeyEnter, ""))
	if !m.sidebarFocused {
		t.Fatal("sidebar not focused after Enter (switchRoute keeps it focused)")
	}
	if m.appState.CurrentView != state.RouteRetain {
		t.Fatalf("after Enter: route %v, want RouteRetain", m.appState.CurrentView)
	}
}

func TestSidebarFocusBlocksGlobalNav(t *testing.T) {
	t.Parallel()

	m := demoModel()
	m, _ = send(t, m, tea.WindowSizeMsg{Width: 120, Height: 40})
	m, _ = send(t, m, appviews.NavigateMsg{Route: state.RouteBanks})
	if !m.sidebarFocused {
		t.Fatal("sidebar not focused after NavigateMsg")
	}

	// 'g' (formerly Banks shortcut) must be eaten while sidebar is focused.
	m, _ = send(t, m, keyPress('g', "g"))
	if m.appState.CurrentView != state.RouteBanks {
		t.Fatalf("'g' during sidebar focus -> %v, want RouteBanks (no nav)", m.appState.CurrentView)
	}
}

func TestEscUnfocusesSidebar(t *testing.T) {
	t.Parallel()

	m := demoModel()
	m, _ = send(t, m, tea.WindowSizeMsg{Width: 120, Height: 40})
	m, _ = send(t, m, appviews.NavigateMsg{Route: state.RouteBanks})
	if !m.sidebarFocused {
		t.Fatal("sidebar not focused after navigate")
	}
	m, _ = send(t, m, tea.KeyPressMsg{Code: tea.KeyEscape})
	if m.sidebarFocused {
		t.Fatal("Esc did not unfocus sidebar")
	}
}

func TestSidebarHiddenOnNarrowWidth(t *testing.T) {
	t.Parallel()

	m := demoModel()
	m, _ = send(t, m, tea.WindowSizeMsg{Width: 60, Height: 40})
	m, _ = send(t, m, appviews.NavigateMsg{Route: state.RouteBanks})
	// Sidebar is focused by switchRoute regardless of width.
	if !m.sidebarFocused {
		t.Fatal("sidebar not focused after NavigateMsg on narrow width")
	}
}

func TestFooterHints(t *testing.T) {
	t.Parallel()

	m := demoModel()
	m, _ = send(t, m, tea.WindowSizeMsg{Width: 120, Height: 40})
	// Sidebar mode after route switch
	m, _ = send(t, m, appviews.NavigateMsg{Route: state.RouteBanks})
	hints := m.footerHints(m.palette)
	if !strings.Contains(hints, "navigate") {
		t.Fatalf("sidebar footer missing 'navigate': %s", hints)
	}
	if !strings.Contains(hints, "back") {
		t.Fatalf("sidebar footer missing 'back': %s", hints)
	}
	// Esc to view mode
	m, _ = send(t, m, tea.KeyPressMsg{Code: tea.KeyEscape})
	hints = m.footerHints(m.palette)
	if !strings.Contains(hints, "tab") {
		t.Fatalf("view footer missing 'tab': %s", hints)
	}
	if !strings.Contains(hints, "sidebar") {
		t.Fatalf("view footer missing 'sidebar': %s", hints)
	}
}


func TestSKeyFocusesSidebar(t *testing.T) {
	t.Parallel()

	m := demoModel()
	m, _ = send(t, m, tea.WindowSizeMsg{Width: 120, Height: 40})
	// Navigate to Banks (sidebar focused by switchRoute)
	m, _ = send(t, m, appviews.NavigateMsg{Route: state.RouteBanks})
	// Unfocus sidebar
	m, _ = send(t, m, tea.KeyPressMsg{Code: tea.KeyEscape})
	if m.sidebarFocused {
		t.Fatal("sidebar still focused after Esc")
	}
	// s re-focuses sidebar
	m, _ = send(t, m, keyPress('s', "s"))
	if !m.sidebarFocused {
		t.Fatal("s did not focus sidebar")
	}
}

func TestQFromSidebarGoesToDashboard(t *testing.T) {
	t.Parallel()

	m := demoModel()
	m, _ = send(t, m, tea.WindowSizeMsg{Width: 120, Height: 40})
	m, _ = send(t, m, appviews.NavigateMsg{Route: state.RouteBanks})
	if !m.sidebarFocused {
		t.Fatal("sidebar not focused")
	}
	m, _ = send(t, m, keyPress('q', "q"))
	if m.appState.CurrentView != state.RouteDashboard {
		t.Fatalf("q from sidebar -> %v, want RouteDashboard", m.appState.CurrentView)
	}
}

func TestQFromSidebarOnDashboardQuits(t *testing.T) {
	t.Parallel()

	m := demoModel()
	m, _ = send(t, m, tea.WindowSizeMsg{Width: 120, Height: 40})
	// Navigate to a route and back to Dashboard to get sidebar focused.
	m, _ = send(t, m, appviews.NavigateMsg{Route: state.RouteBanks})
	m, _ = send(t, m, appviews.NavigateMsg{Route: state.RouteDashboard})
	if !m.sidebarFocused {
		t.Fatal("sidebar not focused after navigating to Dashboard")
	}
	_, cmd := send(t, m, keyPress('q', "q"))
	if cmd == nil {
		t.Fatal("q on Dashboard should return quit cmd")
	}
	// Execute the command to verify it's tea.Quit
	msg := cmd()
	if _, ok := msg.(tea.QuitMsg); !ok {
		t.Fatalf("expected QuitMsg, got %T", msg)
	}
}

func TestSidebarFocusedByDefault(t *testing.T) {
	t.Parallel()
	m := demoModel()
	m, _ = send(t, m, tea.WindowSizeMsg{Width: 120, Height: 40})
	if !m.sidebarFocused {
		t.Fatal("sidebar not focused on startup")
	}
}

func TestQuestionMarkWorksWithSidebarFocused(t *testing.T) {
	t.Parallel()
	m := demoModel()
	m, _ = send(t, m, tea.WindowSizeMsg{Width: 120, Height: 40})
	// Sidebar is focused by default after Step 2
	if !m.sidebarFocused {
		t.Fatal("sidebar not focused")
	}
	// '?' should switch to Help even while sidebar focused
	m, _ = send(t, m, keyPress('?', "?"))
	if m.appState.CurrentView != state.RouteHelp {
		t.Fatalf("'?' with sidebar focused -> %v, want RouteHelp", m.appState.CurrentView)
	}
}

func TestSetupRouteOnFirstRun(t *testing.T) {
	t.Parallel()
	cfg := config.Default()
	cfg.Backend = config.BackendDemo
	m := NewModel(cfg, hindsight.NewDemoClient(), nil, true)
	if m.appState.CurrentView != state.RouteSetup {
		t.Fatalf("CurrentView = %v, want RouteSetup", m.appState.CurrentView)
	}
	if !m.shared.State.SetupActive {
		t.Fatal("SetupActive = false, want true")
	}
}

func TestSetupRouteNotInSidebar(t *testing.T) {
	t.Parallel()
	entries := routeEntries()
	for _, e := range entries {
		if e.Route == state.RouteSetup {
			t.Fatal("RouteSetup should not appear in sidebar entries")
		}
	}
}