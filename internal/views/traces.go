package views

import (
	"context"
	"fmt"
	"strings"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/spinner"
	"charm.land/bubbles/v2/table"
	tea "charm.land/bubbletea/v2"

	"hindsight-tui/internal/domain"
	"hindsight-tui/internal/hindsight"
	"hindsight-tui/internal/ui"
)

type tracesTab int

const (
	tracesAudit tracesTab = iota
	tracesLLM
)

type tracesVersionMsg struct {
	version *domain.VersionInfo
	err     error
}

type tracesLoadedMsg struct {
	gen  int
	tab  tracesTab
	page *domain.Page[map[string]any]
	err  error
}

type TracesView struct {
	shared *Shared
	tab    tracesTab
	pt     pagedTable

	auditInputs []pagedInput
	llmInputs   []pagedInput

	rows   []map[string]any
	notice string
	err    error
}

func NewTracesView(shared *Shared) *TracesView {
	mk := func(name, prompt, placeholder string) pagedInput {
		input := newViewTextInput(placeholder)
		input.Prompt = prompt
		return pagedInput{name: name, input: input}
	}
	auditInputs := []pagedInput{
		mk("action", "Action: ", "action"),
		mk("transport", "Transport: ", "transport"),
		mk("start_date", "Start: ", "YYYY-MM-DD"),
		mk("end_date", "End: ", "YYYY-MM-DD"),
	}
	llmInputs := []pagedInput{
		mk("status", "Status: ", "status"),
		mk("operation", "Operation: ", "operation"),
		mk("scope", "Scope: ", "scope"),
		mk("provider", "Provider: ", "provider"),
		mk("trace_id", "Trace: ", "trace id"),
		mk("document_id", "Document: ", "document id"),
		mk("memory_id", "Memory: ", "memory id"),
		mk("start_date", "Start: ", "YYYY-MM-DD"),
		mk("end_date", "End: ", "YYYY-MM-DD"),
	}
	return &TracesView{
		shared:      shared,
		tab:         tracesAudit,
		auditInputs: auditInputs,
		llmInputs:   llmInputs,
		pt:          newPagedTable(auditInputs, tracesColumns(tracesAudit)),
	}
}

func (v *TracesView) Init() tea.Cmd {
	if v.shared == nil || v.shared.Client == nil {
		return nil
	}
	if v.shared.Version == nil {
		return v.loadVersionCmd()
	}
	return v.loadGateCmd()
}

func (v *TracesView) Update(msg tea.Msg) (View, tea.Cmd) {
	switch msg := msg.(type) {
	case tracesVersionMsg:
		if msg.err != nil {
			v.pt.CancelLoad()
			v.err = msg.err
			return v, nil
		}
		v.shared.Version = msg.version
		if disabled := v.tabDisabledMessage(); disabled != "" {
			v.pt.CancelLoad()
			v.err = nil
			v.notice = disabled
			v.rows = nil
			v.pt.SetPage(nil, 0)
			return v, nil
		}
		return v, v.loadCurrentTabCmd()
	case tracesLoadedMsg:
		if msg.tab != v.tab {
			return v, nil
		}
		if !v.pt.EndLoad(msg.gen) {
			return v, nil
		}
		v.err = msg.err
		if msg.err != nil || msg.page == nil {
			return v, nil
		}
		v.notice = ""
		v.rows = msg.page.Items
		v.pt.SetPage(v.tableRows(), msg.page.Total)
		v.updateSelectedDetail()
		return v, nil
	case spinner.TickMsg:
		return v, v.pt.UpdateFocused(msg)
	case tea.KeyPressMsg:
		switch {
		case key.Matches(msg, v.shared.KeyMap.NextPane):
			return v, v.pt.CycleFocus(1)
		case key.Matches(msg, v.shared.KeyMap.PrevPane):
			return v, v.pt.CycleFocus(-1)
		case key.Matches(msg, v.shared.KeyMap.Refresh):
			return v, v.loadGateCmd()
		}

		if v.pt.TextEntryFocused() {
			if key.Matches(msg, v.shared.KeyMap.Select) {
				v.pt.ResetOffset()
				return v, v.loadGateCmd()
			}
			return v, v.pt.UpdateFocused(msg)
		}

		switch {
		case key.Matches(msg, v.shared.KeyMap.Search):
			return v, v.pt.FocusInput(0)
		case key.Matches(msg, v.shared.KeyMap.Copy):
			v.notice = "Copied JSON to clipboard"
			return v, tea.SetClipboard(v.selectedJSON())
		case msg.String() == "left", msg.String() == "h":
			v.switchTab(tracesAudit)
			return v, tea.Batch(v.pt.FocusTable(), v.loadGateCmd())
		case msg.String() == "right", msg.String() == "l":
			v.switchTab(tracesLLM)
			return v, tea.Batch(v.pt.FocusTable(), v.loadGateCmd())
		case msg.String() == "[":
			if v.pt.PrevPage() {
				return v, v.loadGateCmd()
			}
			return v, nil
		case msg.String() == "]":
			if v.pt.NextPage() {
				return v, v.loadGateCmd()
			}
			return v, nil
		}

		cmd := v.pt.UpdateFocused(msg)
		if v.pt.focus == ptFocusTable {
			v.updateSelectedDetail()
		}
		return v, cmd
	}
	return v, nil
}

func (v *TracesView) View(width, height int) string {
	if width <= 0 || height <= 0 {
		return ""
	}
	p := v.shared.Palette

	status := p.StatusLabel("Bank", currentViewBank(v.shared), "neutral") + p.Muted.Render(" │ ") + p.Muted.Render(v.pt.StatusLine())
	if v.pt.loading {
		status = p.StatusLabel("Bank", currentViewBank(v.shared), "neutral") + p.Muted.Render(" │ ") + p.Spinner.Render(v.pt.LoadingView())
	}
	if v.notice != "" {
		status += p.Muted.Render(" │ ") + p.Warning.Render(v.notice)
	}

	contentWidth := max(20, width-2)
	leftWidth := max(20, contentWidth/2)
	rightWidth := contentWidth - leftWidth
	bodyHeight := max(4, height-10)
	v.pt.table.SetWidth(leftWidth - 4)
	v.pt.table.SetHeight(bodyHeight)
	v.pt.table.SetColumns(tracesColumns(v.tab))
	v.pt.detail.SetWidth(max(10, rightWidth-4))
	v.pt.detail.SetHeight(bodyHeight)

	leftBody := v.pt.table.View()
	if v.pt.rowCount() == 0 && v.err == nil && v.notice == "" && !v.pt.loading {
		leftBody = p.Muted.Render("No rows.")
	}
	if v.err != nil {
		leftBody = renderFriendlyError(v.err)
	}
	if v.notice != "" {
		leftBody = p.Warning.Render(v.notice)
	}

	content := ui.TwoColumn(
		p.Panel(traceTabLabel(v.tab), leftBody, leftWidth),
		p.Panel("Detail", v.pt.detail.View(), rightWidth),
		contentWidth,
	)
	footer := p.Footer.Render("left/right tab • [ ] page • / filter • tab pane • c copy • enter apply • ctrl+r refresh")
	return ui.Lines(renderTabs(p, width, int(v.tab), []string{"Audit Logs", "LLM Requests"}), v.renderFilters(width), status, content, footer)
}

func (v *TracesView) Title() string {
	return "Traces"
}

func (v *TracesView) TextEntryFocused() bool {
	return v.pt.TextEntryFocused()
}

func (v *TracesView) renderFilters(width int) string {
	p := v.shared.Palette
	parts := make([]string, 0, len(v.pt.inputs))
	for i := range v.pt.inputs {
		parts = append(parts, focusedInputView(p, v.pt.inputs[i].name, v.pt.inputs[i].input, v.pt.focus == i))
	}
	return strings.Join(parts, "  ")
}

func (v *TracesView) switchTab(tab tracesTab) {
	v.tab = tab
	v.pt.ResetOffset()
	if tab == tracesAudit {
		v.pt.SetInputs(v.auditInputs)
	} else {
		v.pt.SetInputs(v.llmInputs)
	}
}

func (v *TracesView) loadGateCmd() tea.Cmd {
	if v.shared == nil || v.shared.Client == nil {
		v.pt.CancelLoad()
		v.err = fmt.Errorf("hindsight client is unavailable")
		return nil
	}
	if v.shared.Version == nil {
		return v.loadVersionCmd()
	}
	if disabled := v.tabDisabledMessage(); disabled != "" {
		v.pt.CancelLoad()
		v.err = nil
		v.notice = disabled
		v.rows = nil
		v.pt.SetPage(nil, 0)
		return nil
	}
	return v.loadCurrentTabCmd()
}

func (v *TracesView) loadVersionCmd() tea.Cmd {
	client := v.shared.Client
	timeout := sharedTimeout(v.shared)
	v.notice = ""
	v.err = nil
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()
		version, err := client.Version(ctx)
		return tracesVersionMsg{version: version, err: err}
	}
}

func (v *TracesView) loadCurrentTabCmd() tea.Cmd {
	client := v.shared.Client
	bank := currentViewBank(v.shared)
	tab := v.tab
	timeout := sharedTimeout(v.shared)
	filters := v.filters()
	limit := v.pt.limit
	offset := v.pt.offset
	gen, tick := v.pt.BeginLoad()

	load := func() tea.Msg {
		if client == nil {
			return tracesLoadedMsg{gen: gen, tab: tab, err: fmt.Errorf("hindsight client is unavailable")}
		}
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()
		switch tab {
		case tracesAudit:
			page, err := client.ListAuditLogs(ctx, bank, filters, limit, offset)
			return tracesLoadedMsg{gen: gen, tab: tab, page: page, err: err}
		case tracesLLM:
			page, err := client.ListLLMRequests(ctx, bank, filters, limit, offset)
			return tracesLoadedMsg{gen: gen, tab: tab, page: page, err: err}
		default:
			return tracesLoadedMsg{gen: gen, tab: tab, err: fmt.Errorf("unknown traces tab")}
		}
	}
	return tea.Batch(load, tick)
}

func (v *TracesView) filters() hindsight.TraceFilters {
	if v.tab == tracesAudit {
		return hindsight.TraceFilters{
			Action:    strings.TrimSpace(v.pt.Value("action")),
			Transport: strings.TrimSpace(v.pt.Value("transport")),
			StartDate: strings.TrimSpace(v.pt.Value("start_date")),
			EndDate:   strings.TrimSpace(v.pt.Value("end_date")),
		}
	}
	return hindsight.TraceFilters{
		Status:     strings.TrimSpace(v.pt.Value("status")),
		Operation:  strings.TrimSpace(v.pt.Value("operation")),
		Scope:      strings.TrimSpace(v.pt.Value("scope")),
		Provider:   strings.TrimSpace(v.pt.Value("provider")),
		TraceID:    strings.TrimSpace(v.pt.Value("trace_id")),
		DocumentID: strings.TrimSpace(v.pt.Value("document_id")),
		MemoryID:   strings.TrimSpace(v.pt.Value("memory_id")),
		StartDate:  strings.TrimSpace(v.pt.Value("start_date")),
		EndDate:    strings.TrimSpace(v.pt.Value("end_date")),
	}
}

func (v *TracesView) tabDisabledMessage() string {
	if v.shared == nil || v.shared.Version == nil {
		return ""
	}
	switch v.tab {
	case tracesAudit:
		if !v.shared.Version.Features.AuditLog {
			return "Audit logging is disabled on this Hindsight server."
		}
	case tracesLLM:
		if !v.shared.Version.Features.LLMTrace {
			return "LLM tracing is disabled on this Hindsight server."
		}
	}
	return ""
}

func tracesColumns(tab tracesTab) []table.Column {
	if tab == tracesAudit {
		return []table.Column{{Title: "ID", Width: 20}, {Title: "Action", Width: 16}, {Title: "Transport", Width: 14}, {Title: "Created", Width: 24}}
	}
	return []table.Column{{Title: "ID", Width: 20}, {Title: "Status", Width: 12}, {Title: "Operation", Width: 16}, {Title: "Created", Width: 24}}
}

func (v *TracesView) tableRows() []table.Row {
	rows := make([]table.Row, 0, len(v.rows))
	for _, row := range v.rows {
		if v.tab == tracesAudit {
			rows = append(rows, table.Row{firstNonEmpty(row, "id"), firstNonEmpty(row, "action"), firstNonEmpty(row, "transport"), firstNonEmpty(row, "created_at", "updated_at")})
			continue
		}
		rows = append(rows, table.Row{firstNonEmpty(row, "id"), firstNonEmpty(row, "status"), firstNonEmpty(row, "operation", "scope"), firstNonEmpty(row, "created_at", "updated_at")})
	}
	return rows
}

func (v *TracesView) selectedJSON() string {
	selected := v.pt.table.Cursor()
	if selected < 0 || selected >= len(v.rows) {
		return "{}"
	}
	return ui.PrettyJSON(v.rows[selected])
}

func (v *TracesView) updateSelectedDetail() {
	v.pt.SetDetail(v.selectedJSON())
}

func traceTabLabel(tab tracesTab) string {
	if tab == tracesLLM {
		return "LLM Requests"
	}
	return "Audit Logs"
}
