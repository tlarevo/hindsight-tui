package views

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	gloss "charm.land/lipgloss/v2"

	"hindsight-tui/internal/domain"
	apperrors "hindsight-tui/internal/errors"
	"hindsight-tui/internal/hindsight"
	"hindsight-tui/internal/state"
	"hindsight-tui/internal/ui"
)

type simpleListItem struct {
	title string
	desc  string
	value string
}

func renderFriendlyError(err error) string {
	if err == nil {
		return ""
	}
	friendly := apperrors.Friendly(err)
	parts := []string{friendly.Title, friendly.Detail}
	for _, fix := range friendly.Fixes {
		parts = append(parts, "- "+fix)
	}
	return strings.TrimSpace(strings.Join(parts, "\n"))
}

func valueOrDefault(value *string, fallback string) string {
	if value == nil || strings.TrimSpace(*value) == "" {
		return fallback
	}
	return *value
}

func routeCmd(route state.Route) tea.Cmd {
	return func() tea.Msg { return NavigateMsg{Route: route} }
}

func listItemsFromBanks(banks []domain.BankSummary) []simpleListItem {
	items := make([]simpleListItem, 0, len(banks))
	for _, bank := range banks {
		desc := valueOrDefault(bank.Name, bank.BankID)
		if mission := valueOrDefault(bank.Mission, ""); mission != "" {
			desc = ui.TruncateRunes(mission, 56)
		}
		items = append(items, simpleListItem{title: bank.BankID, desc: desc, value: bank.BankID})
	}
	return items
}

func sortedEnabledFeatures(version *domain.VersionInfo) []string {
	if version == nil {
		return nil
	}
	features := map[string]bool{
		"audit_log":           version.Features.AuditLog,
		"bank_config_api":     version.Features.BankConfigAPI,
		"document_export_api": version.Features.DocumentExportAPI,
		"document_import_api": version.Features.DocumentImportAPI,
		"file_upload_api":     version.Features.FileUploadAPI,
		"llm_trace":           version.Features.LLMTrace,
		"mcp":                 version.Features.MCP,
		"observations":        version.Features.Observations,
		"store_document_text": version.Features.StoreDocumentText,
		"worker":              version.Features.Worker,
	}
	out := make([]string, 0, len(features))
	for name, enabled := range features {
		if enabled {
			out = append(out, name)
		}
	}
	sort.Strings(out)
	return out
}

func operationSummary(row map[string]any) string {
	id := stringifyAny(row["id"])
	if id == "" {
		id = stringifyAny(row["operation_id"])
	}
	status := stringifyAny(row["status"])
	typ := stringifyAny(row["type"])
	if typ == "" {
		typ = stringifyAny(row["task_type"])
	}
	created := stringifyAny(row["created_at"])
	parts := make([]string, 0, 4)
	if id != "" {
		parts = append(parts, id)
	}
	if status != "" {
		parts = append(parts, status)
	}
	if typ != "" {
		parts = append(parts, typ)
	}
	if created != "" {
		parts = append(parts, created)
	}
	if len(parts) == 0 {
		return ui.PrettyJSON(row)
	}
	return strings.Join(parts, " | ")
}

func stringifyAny(value any) string {
	switch typed := value.(type) {
	case nil:
		return ""
	case string:
		return typed
	case fmt.Stringer:
		return typed.String()
	case bool:
		if typed {
			return "true"
		}
		return "false"
	default:
		return fmt.Sprintf("%v", typed)
	}
}

func activeBankFromListShared(current string, fallback string, banks []domain.BankSummary) string {
	if bankExists(banks, current) {
		return current
	}
	if bankExists(banks, fallback) {
		return fallback
	}
	if len(banks) > 0 {
		return banks[0].BankID
	}
	if fallback != "" {
		return fallback
	}
	return "default"
}

func bankExists(banks []domain.BankSummary, bankID string) bool {
	for _, bank := range banks {
		if bank.BankID == bankID {
			return true
		}
	}
	return false
}

func blankFallback(value string, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func moveIndex(current int, delta int, length int) int {
	if length <= 0 {
		return 0
	}
	next := current + delta
	if next < 0 {
		return 0
	}
	if next >= length {
		return length - 1
	}
	return next
}

func renderMenu(title string, items []simpleListItem, selected int, focused bool) string {
	lines := make([]string, 0, len(items))
	for i, item := range items {
		marker := "  "
		if i == selected {
			if focused {
				marker = "> "
			} else {
				marker = "* "
			}
		}
		line := marker + item.title
		if item.desc != "" {
			line += " — " + item.desc
		}
		lines = append(lines, line)
	}
	if len(lines) == 0 {
		lines = append(lines, "No items")
	}
	return ui.Panel(title, strings.Join(lines, "\n"), 0)
}

func renderField(label string, value string, focused bool) string {
	marker := "  "
	if focused {
		marker = "> "
	}
	return fmt.Sprintf("%s%s: %s", marker, label, value)
}

type dashboardLoadMsg struct {
	Health              *domain.HealthStatus
	Version             *domain.VersionInfo
	Banks               []domain.BankSummary
	Stats               map[string]any
	StatsAvailable      bool
	Operations          []map[string]any
	OperationsAvailable bool
	StatsErr            error
	OperationsErr       error
	ActiveBank          string
	Err                 error
}

type DashboardView struct {
	shared              *Shared
	width               int
	height              int
	quickActions        []simpleListItem
	quickActionIndex    int
	loading             bool
	err                 error
	health              *domain.HealthStatus
	version             *domain.VersionInfo
	banks               []domain.BankSummary
	stats               map[string]any
	statsAvailable      bool
	operations          []map[string]any
	operationsAvailable bool
	statsErr            error
	operationsErr       error
	activeBank          string
}

func NewDashboardView(shared *Shared) *DashboardView {
	return &DashboardView{
		shared: shared,
		quickActions: []simpleListItem{
			{title: "Create/select bank", desc: "Open Banks", value: fmt.Sprintf("%d", state.RouteBanks)},
			{title: "Retain memory", desc: "Open Retain", value: fmt.Sprintf("%d", state.RouteRetain)},
			{title: "Recall memory", desc: "Open Recall", value: fmt.Sprintf("%d", state.RouteRecall)},
			{title: "Reflect", desc: "Open Reflect", value: fmt.Sprintf("%d", state.RouteReflect)},
			{title: "Explorer", desc: "Open Explorer", value: fmt.Sprintf("%d", state.RouteExplorer)},
			{title: "Traces", desc: "Open Traces", value: fmt.Sprintf("%d", state.RouteTraces)},
			{title: "Config", desc: "Open Config", value: fmt.Sprintf("%d", state.RouteConfig)},
			{title: "Help", desc: "Open Help", value: fmt.Sprintf("%d", state.RouteHelp)},
		},
		health:     shared.Health,
		version:    shared.Version,
		activeBank: shared.State.ActiveBank,
	}
}

func (v *DashboardView) Title() string { return "Dashboard" }

func (v *DashboardView) TextEntryFocused() bool {
	return false
}

func (v *DashboardView) Init() tea.Cmd {
	v.loading = true
	return loadDashboardCmd(v.shared)
}

func loadDashboardCmd(shared *Shared) tea.Cmd {
	client := shared.Client
	preferred := shared.State.ActiveBank
	defaultBank := shared.Config.DefaultBank
	timeout := sharedTimeout(shared)
	return func() tea.Msg {
		msg := dashboardLoadMsg{}
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()

		health, err := client.Health(ctx)
		if err != nil {
			msg.Err = err
			return msg
		}
		msg.Health = health
		if health != nil && !health.OK && strings.Contains(health.Detail, hindsight.EmbedInstallHint()) {
			msg.ActiveBank = blankFallback(preferred, defaultBank)
			return msg
		}

		version, err := client.Version(ctx)
		if err != nil {
			msg.Err = err
			return msg
		}
		msg.Version = version

		banks, err := client.ListBanks(ctx)
		if err != nil {
			msg.Err = err
			return msg
		}
		msg.Banks = banks
		msg.ActiveBank = activeBankFromListShared(preferred, defaultBank, banks)
		if strings.TrimSpace(msg.ActiveBank) == "" {
			return msg
		}

		stats, err := client.BankStats(ctx, msg.ActiveBank)
		if err == nil {
			msg.Stats = stats
			msg.StatsAvailable = true
		} else {
			msg.StatsErr = err
		}
		operations, err := client.ListOperations(ctx, msg.ActiveBank, "", "", 5, 0)
		if err == nil && operations != nil {
			msg.Operations = operations.Items
			msg.OperationsAvailable = true
		} else if err != nil {
			msg.OperationsErr = err
		}
		return msg
	}
}

func (v *DashboardView) Update(msg tea.Msg) (View, tea.Cmd) {
	switch typed := msg.(type) {
	case dashboardLoadMsg:
		v.loading = false
		v.err = typed.Err
		if typed.Health != nil {
			v.health = typed.Health
			v.shared.Health = typed.Health
		}
		if typed.Version != nil {
			v.version = typed.Version
			v.shared.Version = typed.Version
		}
		if typed.Banks != nil {
			v.banks = typed.Banks
		}
		if typed.ActiveBank != "" {
			v.activeBank = typed.ActiveBank
			v.shared.State.ActiveBank = typed.ActiveBank
		}
		v.stats = typed.Stats
		v.statsAvailable = typed.StatsAvailable
		v.statsErr = typed.StatsErr
		v.operations = typed.Operations
		v.operationsAvailable = typed.OperationsAvailable
		v.operationsErr = typed.OperationsErr
		return v, nil
	case tea.KeyPressMsg:
		if key.Matches(typed, v.shared.KeyMap.Refresh) {
			v.loading = true
			v.err = nil
			return v, loadDashboardCmd(v.shared)
		}
		if key.Matches(typed, v.shared.KeyMap.Down) {
			v.quickActionIndex = moveIndex(v.quickActionIndex, 1, len(v.quickActions))
			return v, nil
		}
		if key.Matches(typed, v.shared.KeyMap.Up) {
			v.quickActionIndex = moveIndex(v.quickActionIndex, -1, len(v.quickActions))
			return v, nil
		}
		if key.Matches(typed, v.shared.KeyMap.Select) {
			raw := v.quickActions[v.quickActionIndex].value
			switch raw {
			case fmt.Sprintf("%d", state.RouteBanks):
				return v, routeCmd(state.RouteBanks)
			case fmt.Sprintf("%d", state.RouteRetain):
				return v, routeCmd(state.RouteRetain)
			case fmt.Sprintf("%d", state.RouteRecall):
				return v, routeCmd(state.RouteRecall)
			case fmt.Sprintf("%d", state.RouteReflect):
				return v, routeCmd(state.RouteReflect)
			case fmt.Sprintf("%d", state.RouteExplorer):
				return v, routeCmd(state.RouteExplorer)
			case fmt.Sprintf("%d", state.RouteTraces):
				return v, routeCmd(state.RouteTraces)
			case fmt.Sprintf("%d", state.RouteConfig):
				return v, routeCmd(state.RouteConfig)
			case fmt.Sprintf("%d", state.RouteHelp):
				return v, routeCmd(state.RouteHelp)
			}
		}
		return v, nil
	default:
		return v, nil
	}
}

func (v *DashboardView) View(width, height int) string {
	v.width = width
	v.height = height
	if width <= 0 || height <= 0 {
		return ""
	}
	statusLines := []string{
		fmt.Sprintf("Backend: %s", v.shared.Config.Backend),
		fmt.Sprintf("API URL: %s", v.shared.Config.APIURL),
		fmt.Sprintf("Active bank: %s", blankFallback(v.activeBank, v.shared.Config.DefaultBank)),
		fmt.Sprintf("Bank count: %d", len(v.banks)),
	}
	if v.health != nil {
		status := "degraded"
		if v.health.OK {
			status = "running"
		}
		statusLines = append(statusLines, fmt.Sprintf("Health: %s", status))
		if strings.TrimSpace(v.health.Detail) != "" {
			statusLines = append(statusLines, fmt.Sprintf("Health detail: %s", v.health.Detail))
		}
	}
	if v.version != nil {
		statusLines = append(statusLines, fmt.Sprintf("API version: %s", blankFallback(v.version.APIVersion, "unavailable")))
		features := sortedEnabledFeatures(v.version)
		if len(features) == 0 {
			statusLines = append(statusLines, "Feature flags: none enabled")
		} else {
			statusLines = append(statusLines, "Feature flags: "+strings.Join(features, ", "))
		}
	} else {
		statusLines = append(statusLines, "API version: unavailable", "Feature flags: unavailable")
	}

	statsBody := "unavailable"
	if v.statsAvailable {
		statsBody = ui.PrettyJSON(v.stats)
	} else if v.statsErr != nil {
		statsBody = apperrors.Friendly(v.statsErr).Title
	}
	operationsBody := "unavailable"
	if v.operationsAvailable {
		lines := make([]string, 0, len(v.operations))
		for _, row := range v.operations {
			lines = append(lines, operationSummary(row))
		}
		if len(lines) == 0 {
			operationsBody = "No recent operations"
		} else {
			operationsBody = strings.Join(lines, "\n")
		}
	} else if v.operationsErr != nil {
		operationsBody = apperrors.Friendly(v.operationsErr).Title
	}

	hints := []string{
		"uvx hindsight-embed@latest configure",
		"pipx install hindsight-embed",
		"hindsight-embed configure",
		"pip install hindsight-api",
		"hindsight-api",
	}
	if v.health != nil && !v.health.OK && strings.Contains(v.health.Detail, hindsight.EmbedInstallHint()) {
		hints = append([]string{hindsight.EmbedInstallHint(), "Open Config to switch backend or update API URL."}, hints...)
	}

	left := ui.Lines(
		ui.Panel("Status", strings.Join(statusLines, "\n"), 0),
		ui.Panel("Current bank stats", statsBody, 0),
		ui.Panel("Recent operations", operationsBody, 0),
	)
	right := ui.Lines(
		renderMenu("Quick actions", v.quickActions, v.quickActionIndex, true),
		ui.Panel("Bootstrap hints", strings.Join(hints, "\n"), 0),
	)
	banner := gloss.NewStyle().Bold(true).Render("Dashboard")
	if v.loading {
		banner += "\nLoading dashboard..."
	}
	if v.err != nil {
		banner += "\n" + renderFriendlyError(v.err)
	}
	return ui.Lines(banner, ui.TwoColumn(left, right, max(width-2, 20)))
}
