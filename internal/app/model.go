package app

import (
	"context"
	"strings"
	"time"

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
	appviews "hindsight-tui/internal/views"
)

type Options struct {
	ConfigPath        string
	BackendOverride   string
	APIURLOverride    string
	AuthTokenOverride string
	Doctor            bool
	Setup             bool
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
	sidebarFocused bool
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

	// First-run detection: no config file and no CLI overrides → setup
	if !opts.Setup && !opts.Doctor && !config.FileExists(opts.ConfigPath) &&
		opts.BackendOverride == "" && opts.APIURLOverride == "" {
		opts.Setup = true
	}

	model := NewModel(cfg, client, embed, opts.Setup)
	_, err = tea.NewProgram(model).Run()
	return err
}

func NewModel(cfg config.Config, client hindsight.Client, embed *hindsight.EmbedManager, setup bool) Model {
	km := appkeymap.Default()
	pal := theme.Resolve(cfg.Theme)
	shared := &appviews.Shared{
		Config: &cfg,
		State: &state.AppState{
			ActiveBank:  cfg.DefaultBank,
			Backend:     cfg.Backend,
			CurrentView: state.RouteDashboard,
			SetupActive: setup,
		},
		Client:  client,
		Embed:   embed,
		KeyMap:  km,
		Palette: pal,
	}

	initialRoute := state.RouteDashboard
	if setup {
		initialRoute = state.RouteSetup
		shared.State.CurrentView = initialRoute
	}

	model := Model{
		cfg:            cfg,
		appState:       *shared.State,
		client:         client,
		embed:          embed,
		keymap:         km,
		shared:         shared,
		previous:       state.RouteDashboard,
		palette:        pal,
		themeName:      cfg.Theme,
		sidebarFocused: !setup,
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
		state.RouteSetup:      appviews.NewBootstrapView(shared),
	}
	model.appState.CurrentView = initialRoute
	model.sidebar = newSidebar(initialRoute, pal)
	return model
}

func newSidebar(current state.Route, pal theme.Palette) list.Model {
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
	delegate.Styles.SelectedTitle = pal.SidebarSelected
	delegate.Styles.SelectedDesc = pal.SidebarSelected

	sidebar := list.New(items, delegate, 18, 12)
	sidebar.SetShowTitle(false)
	sidebar.SetShowFilter(false)
	sidebar.SetShowHelp(false)
	sidebar.SetShowPagination(false)
	sidebar.SetShowStatusBar(false)
	sidebar.Select(selected)

	// Restrict to arrow keys; j/k and / are handled at the model level.
	sidebar.KeyMap.CursorUp = key.NewBinding(key.WithKeys("up"))
	sidebar.KeyMap.CursorDown = key.NewBinding(key.WithKeys("down"))
	sidebar.DisableQuitKeybindings()  // q/esc and ctrl+c — handled by model
	sidebar.KeyMap.ClearFilter = key.NewBinding()  // esc — handled by model
	sidebar.KeyMap.Filter = key.NewBinding()       // / — not relevant in sidebar
	sidebar.KeyMap.ShowFullHelp = key.NewBinding()  // ? — handled by model
	sidebar.KeyMap.CloseFullHelp = key.NewBinding() // ? — handled by model
	sidebar.KeyMap.PrevPage = key.NewBinding()      // h/pgup — not relevant
	sidebar.KeyMap.NextPage = key.NewBinding()      // l/pgdn — not relevant
	sidebar.KeyMap.GoToStart = key.NewBinding()     // g — not relevant
	sidebar.KeyMap.GoToEnd = key.NewBinding()       // G — not relevant

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
		if msg.Width >= 80 {
			m.sidebar.SetSize(18, max(3, msg.Height-6))
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

		// Sidebar-focused mode: route keys to sidebar
		if m.sidebarFocused {
			switch {
			case key.Matches(msg, m.keymap.PrevPane), msg.Key().Code == tea.KeyEscape:
				m.sidebarFocused = false
				return m, nil
			case key.Matches(msg, m.keymap.Up), key.Matches(msg, m.keymap.Down):
				updated, cmd := m.sidebar.Update(msg)
				m.sidebar = updated
				return m, cmd
			case key.Matches(msg, m.keymap.Select):
				idx := m.sidebar.Index()
				entries := routeEntries()
				if idx >= 0 && idx < len(entries) {
					return m.switchRoute(entries[idx].Route)
				}
				return m, nil
			case key.Matches(msg, m.keymap.Back):
				if m.appState.CurrentView == state.RouteDashboard || m.appState.CurrentView == state.RouteSetup {
					return m, tea.Quit
				}
				return m.switchRoute(state.RouteDashboard)
			case key.Matches(msg, m.keymap.Help):
				if m.appState.CurrentView != state.RouteHelp {
					m.previous = m.appState.CurrentView
				}
				return m.switchRoute(state.RouteHelp)
			default:
				return m, nil // eat all other keys while sidebar focused
			}
		}

		if !m.currentView().TextEntryFocused() {
			switch {
			case key.Matches(msg, m.keymap.Back):
				if m.appState.CurrentView == state.RouteDashboard || m.appState.CurrentView == state.RouteSetup {
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
			case key.Matches(msg, m.keymap.FocusSidebar):
				if m.width >= 80 {
					m.sidebarFocused = true
					return m, nil
				}
				return m, nil
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

	p := m.palette
	sidebarWidth := 0
	if m.width >= 80 {
		sidebarWidth = 20
	}
	bodyWidth := m.width - sidebarWidth

	contentTitle := routeLabel(m.appState.CurrentView)
	if current := m.currentView(); current != nil {
		contentTitle = current.Title()
	}

	content := m.currentView().View(max(0, bodyWidth-4), max(0, m.height-6))
	if sidebarWidth > 0 {
		content = p.Panel(contentTitle, content, bodyWidth)
		var left string
		if m.sidebarFocused {
			left = p.FocusedPanel("Routes", m.sidebar.View(), sidebarWidth)
		} else {
			left = p.Panel("Routes", m.sidebar.View(), sidebarWidth)
		}
		content = gloss.JoinHorizontal(gloss.Top, left, content)
	} else {
		content = p.Panel(contentTitle, content, bodyWidth)
	}

	// Styled header with status indicators
	bankName := blank(m.appState.ActiveBank, "default")
	backendName := string(m.appState.Backend)
	statusKind := "good"
	if m.statusLine() == "degraded" {
		statusKind = "bad"
	} else if m.statusLine() == "demo" {
		statusKind = "neutral"
	}

	headerLeft := p.Primary.Render("hindsight-tui")
	headerSep := p.Muted.Render(" │ ")
	headerBank := p.StatusLabel("Bank", bankName, "neutral")
	headerBackend := p.StatusLabel("Backend", backendName, "neutral")
	headerStatus := p.StatusLabelWithIcon("Status", m.statusLine(), statusKind)
	modeLabel := "view"
	if m.sidebarFocused {
		modeLabel = "sidebar"
	} else if m.currentView().TextEntryFocused() {
		modeLabel = "text"
	}
	headerMode := p.StatusLabel("Mode", modeLabel, "neutral")
	header := p.Header.Render(headerLeft + headerSep + headerBank + headerSep + headerBackend + headerSep + headerStatus + headerSep + headerMode)

	footer := p.Footer.Render(m.footerHints(p))
	view := tea.NewView(strings.Join([]string{header, content, footer}, "\n"))
	view.AltScreen = true
	return view
}

func (m Model) currentView() appviews.View {
	return m.views[m.appState.CurrentView]
}

func (m Model) footerHints(p theme.Palette) string {
	render := func(parts ...string) string {
		out := make([]string, 0, len(parts)/2)
		for i := 0; i < len(parts)-1; i += 2 {
			out = append(out, p.Muted.Render(parts[i]+" ")+parts[i+1])
		}
		return strings.Join(out, p.Muted.Render(" · "))
	}
	if m.sidebarFocused {
		return render(
			"↑↓", "navigate",
			"enter", "select",
			"tab/esc", "content",
			"?", "help",
			"q", "back",
		)
	}
	if m.appState.CurrentView == state.RouteSetup {
		if m.currentView().TextEntryFocused() {
			return render("esc", "cancel", "enter", "submit")
		}
		return render(
			"↑↓", "navigate",
			"enter", "next",
			"esc", "back",
			"q", "quit",
		)
	}
	if m.currentView().TextEntryFocused() {
		return render("esc", "cancel", "enter", "submit")
	}
	return render(
		"s", "sidebar",
		"tab", "pane",
		"ctrl+r", "refresh",
		"ctrl+s", "save",
		"?", "help",
		"q", "back",
	)
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
	m.sidebarFocused = true
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
		m.shared.Palette = m.palette
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
