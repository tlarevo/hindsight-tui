package app

import "github.com/tlarevo/hindsight-tui/internal/state"

type routeEntry struct {
	Route state.Route
	Label string
}

func routeEntries() []routeEntry {
	return []routeEntry{
		{Route: state.RouteDashboard, Label: "Dashboard"},
		{Route: state.RouteBanks, Label: "Banks"},
		{Route: state.RouteRetain, Label: "Retain"},
		{Route: state.RouteRecall, Label: "Recall"},
		{Route: state.RouteReflect, Label: "Reflect"},
		{Route: state.RouteExplorer, Label: "Explorer"},
		{Route: state.RouteOperations, Label: "Operations"},
		{Route: state.RouteTraces, Label: "Traces"},
		{Route: state.RouteConfig, Label: "Config"},
		{Route: state.RouteHelp, Label: "Help"},
	}
}

func routeLabel(route state.Route) string {
	for _, entry := range routeEntries() {
		if entry.Route == route {
			return entry.Label
		}
	}
	return "Dashboard"
}
