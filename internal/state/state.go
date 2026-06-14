package state

import "hindsight-tui/internal/config"

type Route int

const (
	RouteDashboard Route = iota
	RouteBanks
	RouteRetain
	RouteRecall
	RouteReflect
	RouteExplorer
	RouteOperations
	RouteTraces
	RouteConfig
	RouteHelp
	RouteSetup
)

type AppState struct {
	ActiveBank  string
	Backend     config.Backend
	CurrentView Route
	SetupActive bool
}
