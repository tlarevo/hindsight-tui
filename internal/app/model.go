package app

import (
	"context"
	"fmt"
	"strings"
	"time"

	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
	gloss "charm.land/lipgloss/v2"

	"hindsight-tui/internal/config"
	"hindsight-tui/internal/domain"
	"hindsight-tui/internal/hindsight"
	appkeymap "hindsight-tui/internal/keymap"
	"hindsight-tui/internal/state"
	"hindsight-tui/internal/theme"
	"hindsight-tui/internal/ui"
	appviews "hindsight-tui/internal/views"
)

type Options struct {
	ConfigPath        string
	BackendOverride   string
	APIURLOverride    string
	AuthTokenOverride string
	Doctor            bool
}

type Model struct {
	cfg       config.Config
	appState  state.AppState
	client    hindsight.Client
	embed     *hindsight.EmbedManager
	width     int
	height    int
	version   *domain.VersionInfo
	health    *domain.HealthStatus
	loading   bool
	err       error
	sidebar   list.Model
	help      help.Model
	keymap    appkeymap.KeyMap
	views     map[state.Route]appviews.View
	shared    *appviews.Shared
	previous  state.Route
	palette   theme.Palette
	themeName string
}

type routeItem struct {
	title string
}

func (i routeItem) Title() string       { return i.title }
func (i routeItem) Description() string { return "" }
func (i routeItem) FilterValue() string { return i.title }

func Run(opts Options) error {
	cfg, err := loadConfigForRun(opts)
	if err != nil {
		return err
	}

	client, embed := hindsight.NewFromConfig(cfg)
	if opts.Doctor {
		return runDoctor(cfg, client)
	}

	model := NewModel(cfg, client, embed)
	_, err = tea.NewProgram(model).Run()
	return err
}

func NewModel(cfg config.Config, client hindsight.Client, embed *hindsight.EmbedManager) Model {
	km := appkeymap.Default()
	shared := &appviews.Shared{
		Config: &cfg,
		State: &state.AppState{
			ActiveBank:  cfg.DefaultBank,
			Backend:     cfg.Backend,
			CurrentView: state.RouteDashboard,
		},
		Client: client,
		Embed:  embed,
		KeyMap: km,
	}

	model := Model{
		cfg:       cfg,
		appState:  *shared.State,
		client:    client,
		embed:     embed,
		keymap:    km,
		help:      help.New(),
		shared:    shared,
		previous:  state.RouteDashboard,
		palette:   theme.Resolve(cfg.Theme),
		themeName: cfg.Theme,
	}
	model.views = map[state.Route]appviews.View{
		state.RouteDashboard:  appviews.NewDashboardView(shared),
		state.RouteBanks:      appviews.NewBanksView(shared),
		state.RouteRetain:     appviews.NewRetainView(shared),
		state.RouteRecall:     appviews.NewRecallView(shared),
		state.RouteReflect:    appviews.NewReflectView(shared),
		state.RouteExplorer:   appviews.NewExplorerView(shared),
		state.RouteOperations: appviews.NewOperationsView(shared),
		state.RouteTraces:     appviews.NewTracesView(shared),
		state.RouteConfig:     appviews.NewConfigView(shared),
		state.RouteHelp:       appviews.NewHelpView(shared),
	}
	model.sidebar = newSidebar(model.appState.CurrentView)
	return model
}

func newSidebar(current state.Route) list.Model {
	items := make([]list.Item, 0, len(routeEntries()))
	selected := 0
	for i, entry := range routeEntries() {
		items = append(items, routeItem{title: entry.Label})
		if entry.Route == current {
			selected = i
		}
	}

	delegate := list.NewDefaultDelegate()
	delegate.ShowDescription = false
	delegate.SetSpacing(0)

	sidebar := list.New(items, delegate, 18, 12)
	sidebar.SetShowTitle(false)
	sidebar.SetShowFilter(false)
	sidebar.SetShowHelp(false)
	sidebar.SetShowPagination(false)
	sidebar.SetShowStatusBar(false)
	sidebar.Select(selected)
	return sidebar
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(m.currentView().Init(), m.loadStatusCmd())
}

type appStatusMsg struct {
	health  *domain.HealthStatus
	version *domain.VersionInfo
}

func (m Model) loadStatusCmd() tea.Cmd {
	client := m.client
	timeout := time.Duration(m.cfg.TimeoutMS) * time.Millisecond
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()
		msg := appStatusMsg{}
		if health, err := client.Health(ctx); err == nil {
			msg.health = health
		}
		if version, err := client.Version(ctx); err == nil {
			msg.version = version
		}
		return msg
	}
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.help.SetWidth(max(0, msg.Width))
		if msg.Width >= 80 {
			m.sidebar.SetSize(18, max(3, msg.Height-4))
		}
		m.syncShared()
		return m, nil
	case appviews.NavigateMsg:
		return m.switchRoute(msg.Route)
	case appviews.OpenReflectMsg:
		if reflectView, ok := m.views[state.RouteReflect].(*appviews.ReflectView); ok {
			reflectView.ApplyPrefill(msg.Bank, msg.Query)
		}
		return m.switchRoute(state.RouteReflect)
	case appviews.OpenRetainMsg:
		if retainView, ok := m.views[state.RouteRetain].(*appviews.RetainView); ok {
			retainView.ApplyPrefill(msg.Bank, msg.Content, msg.Context, msg.Tags)
		}
		return m.switchRoute(state.RouteRetain)
	case appStatusMsg:
		if msg.health != nil {
			m.shared.Health = msg.health
		}
		if msg.version != nil {
			m.shared.Version = msg.version
		}
		m.syncShared()
		return m, nil
	case tea.KeyPressMsg:
		if key.Matches(msg, m.keymap.Quit) {
			return m, tea.Quit
		}
		if !m.currentView().TextEntryFocused() {
			switch {
			case key.Matches(msg, m.keymap.Back):
				if m.appState.CurrentView == state.RouteDashboard {
					return m, tea.Quit
				}
				if m.appState.CurrentView == state.RouteHelp {
					return m.switchRoute(m.previous)
				}
				return m.switchRoute(state.RouteDashboard)
			case key.Matches(msg, m.keymap.Help):
				if m.appState.CurrentView != state.RouteHelp {
					m.previous = m.appState.CurrentView
				}
				return m.switchRoute(state.RouteHelp)
			case key.Matches(msg, m.keymap.Banks):
				return m.switchRoute(state.RouteBanks)
			case key.Matches(msg, m.keymap.Recall):
				return m.switchRoute(state.RouteRecall)
			case key.Matches(msg, m.keymap.Retain):
				return m.switchRoute(state.RouteRetain)
			case key.Matches(msg, m.keymap.Reflect):
				if m.appState.CurrentView != state.RouteRecall {
					return m.switchRoute(state.RouteReflect)
				}
			}
		}
	}

	current := m.currentView()
	next, cmd := current.Update(msg)
	m.views[m.appState.CurrentView] = next
	m.syncShared()
	return m, cmd
}

func (m Model) View() tea.View {
	if m.width <= 0 || m.height <= 0 {
		return tea.NewView("hindsight-tui")
	}

	bodyWidth := m.width
	contentTitle := routeLabel(m.appState.CurrentView)
	if current := m.currentView(); current != nil {
		contentTitle = current.Title()
	}

	content := m.currentView().View(max(0, bodyWidth-2), max(0, m.height-4))
	if m.width >= 80 {
		sidebarWidth := 18
		bodyWidth -= sidebarWidth + 1
		content = ui.Panel(contentTitle, content, bodyWidth)
		left := ui.Panel("Routes", m.sidebar.View(), sidebarWidth)
		content = gloss.JoinHorizontal(gloss.Top, left, content)
	} else {
		content = ui.Panel(contentTitle, content, bodyWidth)
	}

	header := m.palette.Header.Render(fmt.Sprintf("hindsight-tui | Bank: %s | Backend: %s | Status: %s", blank(m.appState.ActiveBank, "default"), m.appState.Backend, m.statusLine()))
	footer := m.help.View(m.keymap)
	view := tea.NewView(strings.Join([]string{header, content, footer}, "\n"))
	view.AltScreen = true
	return view
}

func (m Model) currentView() appviews.View {
	return m.views[m.appState.CurrentView]
}

func (m *Model) switchRoute(route state.Route) (tea.Model, tea.Cmd) {
	m.appState.CurrentView = route
	m.shared.State.CurrentView = route
	for i, entry := range routeEntries() {
		if entry.Route == route {
			m.sidebar.Select(i)
			break
		}
	}
	m.syncShared()
	return *m, m.currentView().Init()
}

func (m *Model) syncShared() {
	m.cfg = *m.shared.Config
	m.appState = *m.shared.State
	m.client = m.shared.Client
	m.embed = m.shared.Embed
	m.version = m.shared.Version
	m.health = m.shared.Health
	if m.cfg.Theme != m.themeName {
		m.palette = theme.Resolve(m.cfg.Theme)
		m.themeName = m.cfg.Theme
	}
}

func (m Model) statusLine() string {
	if m.appState.Backend == config.BackendDemo {
		return "demo"
	}
	if m.health != nil && m.health.OK {
		return "running"
	}
	return "degraded"
}

func blank(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}
