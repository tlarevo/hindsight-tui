package views

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"github.com/tlarevo/hindsight-tui/internal/domain"
	apperrors "github.com/tlarevo/hindsight-tui/internal/errors"
	"github.com/tlarevo/hindsight-tui/internal/hindsight"
	"github.com/tlarevo/hindsight-tui/internal/theme"
	"github.com/tlarevo/hindsight-tui/internal/ui"
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

func renderMenu(p theme.Palette, title string, items []simpleListItem, selected int, focused bool) string {
	lines := make([]string, 0, len(items))
	for i, item := range items {
		marker := "  "
		if i == selected {
			if focused {
				marker = p.Primary.Render("▸ ")
			} else {
				marker = p.Accent.Render("▸ ")
			}
		}
		titleText := item.title
		if i == selected && focused {
			titleText = p.Primary.Render(item.title)
		}
		line := marker + titleText
		if item.desc != "" {
			line += p.Muted.Render(" — ") + p.Muted.Render(item.desc)
		}
		lines = append(lines, line)
	}
	if len(lines) == 0 {
		lines = append(lines, p.Muted.Render("No items"))
	}
	return p.Panel(title, strings.Join(lines, "\n"), 0)
}

func renderField(p theme.Palette, label string, value string, focused bool) string {
	if focused {
		return fmt.Sprintf("%s%s: %s", p.FocusedLabel.Render("▸ "), p.FocusedLabel.Render(label), value)
	}
	return fmt.Sprintf("%s: %s", p.FormLabel.Render(label), p.Muted.Render(value))
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
		shared:      shared,
		health:      shared.Health,
		version:     shared.Version,
		activeBank:  shared.State.ActiveBank,
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
	p := v.shared.Palette
	contentWidth := max(20, width-2)

	// Status section with semantic coloring
	statusLines := []string{
		p.StatusLabel("Backend", string(v.shared.Config.Backend), "neutral"),
		p.StatusLabel("API URL", v.shared.Config.APIURL, ""),
		p.StatusLabel("Active bank", blankFallback(v.activeBank, v.shared.Config.DefaultBank), "neutral"),
		p.StatusLabel("Bank count", fmt.Sprintf("%d", len(v.banks)), ""),
	}
	if v.health != nil {
		healthKind := "good"
		if !v.health.OK {
			healthKind = "bad"
		}
		statusLines = append(statusLines, p.StatusLabelWithIcon("Health", map[bool]string{true: "running", false: "degraded"}[v.health.OK], healthKind))
		if strings.TrimSpace(v.health.Detail) != "" {
			statusLines = append(statusLines, p.StatusLabel("Health detail", v.health.Detail, healthKind))
		}
	}
	if v.version != nil {
		statusLines = append(statusLines, p.StatusLabel("API version", blankFallback(v.version.APIVersion, "unavailable"), "good"))
		features := sortedEnabledFeatures(v.version)
		if len(features) == 0 {
			statusLines = append(statusLines, p.Muted.Render("Feature flags: none enabled"))
		} else {
			statusLines = append(statusLines, p.StatusLabel("Feature flags", strings.Join(features, ", "), "neutral"))
		}
	} else {
		statusLines = append(statusLines, p.StatusLabel("API version", "unavailable", "bad"), p.Muted.Render("Feature flags: unavailable"))
	}

	statsBody := p.Muted.Render("unavailable")
	if v.statsAvailable {
		statsBody = ui.PrettyJSON(v.stats)
	} else if v.statsErr != nil {
		statsBody = p.Error.Render(apperrors.Friendly(v.statsErr).Title)
	}
	operationsBody := p.Muted.Render("unavailable")
	if v.operationsAvailable {
		lines := make([]string, 0, len(v.operations))
		for _, row := range v.operations {
			lines = append(lines, operationSummary(row))
		}
		if len(lines) == 0 {
			operationsBody = p.Muted.Render("No recent operations")
		} else {
			operationsBody = strings.Join(lines, "\n")
		}
	} else if v.operationsErr != nil {
		operationsBody = p.Error.Render(apperrors.Friendly(v.operationsErr).Title)
	}

	panels := ui.Lines(
		p.Panel("Status", strings.Join(statusLines, "\n"), contentWidth),
		p.Panel("Current bank stats", statsBody, contentWidth),
		p.Panel("Recent operations", operationsBody, contentWidth),
	)
	if v.loading {
		return ui.Lines(p.Spinner.Render("⟳")+" "+p.Primary.Render("Loading dashboard..."), panels)
	}
	if v.err != nil {
		return ui.Lines(renderFriendlyError(v.err), panels)
	}
	return panels
}
