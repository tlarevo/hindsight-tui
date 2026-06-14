package app

import (
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/tlarevo/hindsight-tui/internal/state"
	appviews "github.com/tlarevo/hindsight-tui/internal/views"
)

// TestSmokeRendersEveryRoute drives the demo model through a window resize and
// every route, rendering each, to catch render-time panics across views.
func TestSmokeRendersEveryRoute(t *testing.T) {
	t.Parallel()

	m := demoModel()
	m, _ = send(t, m, tea.WindowSizeMsg{Width: 120, Height: 40})

	routes := []state.Route{
		state.RouteDashboard,
		state.RouteBanks,
		state.RouteRetain,
		state.RouteRecall,
		state.RouteReflect,
		state.RouteExplorer,
		state.RouteOperations,
		state.RouteTraces,
		state.RouteConfig,
		state.RouteHelp,
	}
	for _, route := range routes {
		m, _ = send(t, m, appviews.NavigateMsg{Route: route})
		_ = m.View()
	}
}
