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
	"hindsight-tui/internal/ui"
)

type operationsLoadedMsg struct {
	gen  int
	page *domain.Page[map[string]any]
	err  error
}

type OperationsView struct {
	shared *Shared
	pt     pagedTable
	rows   []map[string]any
	notice string
	err    error
}

func NewOperationsView(shared *Shared) *OperationsView {
	statusInput := newViewTextInput("status")
	statusInput.Prompt = "Status: "
	typeInput := newViewTextInput("type")
	typeInput.Prompt = "Type: "
	return &OperationsView{
		shared: shared,
		pt: newPagedTable([]pagedInput{
			{name: "status", input: statusInput},
			{name: "type", input: typeInput},
		}, operationsColumns()),
	}
}

func (v *OperationsView) Init() tea.Cmd {
	return v.loadCmd()
}

func (v *OperationsView) Update(msg tea.Msg) (View, tea.Cmd) {
	switch msg := msg.(type) {
	case operationsLoadedMsg:
		if !v.pt.EndLoad(msg.gen) {
			return v, nil
		}
		v.err = msg.err
		v.notice = ""
		if msg.err == nil && msg.page != nil {
			v.rows = msg.page.Items
			v.pt.SetPage(v.tableRows(), msg.page.Total)
			v.updateSelectedDetail()
		}
		return v, nil
	case spinner.TickMsg:
		return v, v.pt.UpdateFocused(msg)
	case tea.KeyPressMsg:
		switch {
		case key.Matches(msg, v.shared.KeyMap.NextPane):
			return v, v.pt.CycleFocus(1)
		case key.Matches(msg, v.shared.KeyMap.PrevPane):
			return v, v.pt.CycleFocus(-1)
		case key.Matches(msg, v.shared.KeyMap.Search):
			return v, v.pt.FocusInput(0)
		case key.Matches(msg, v.shared.KeyMap.Copy) && !v.pt.TextEntryFocused():
			v.notice = "Copied JSON to clipboard"
			return v, tea.SetClipboard(v.selectedJSON())
		case key.Matches(msg, v.shared.KeyMap.Refresh):
			return v, v.loadCmd()
		case key.Matches(msg, v.shared.KeyMap.Select) && v.pt.TextEntryFocused():
			v.pt.ResetOffset()
			return v, v.loadCmd()
		case msg.String() == "[":
			if v.pt.PrevPage() {
				return v, v.loadCmd()
			}
			return v, nil
		case msg.String() == "]":
			if v.pt.NextPage() {
				return v, v.loadCmd()
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

func (v *OperationsView) View(width, height int) string {
	if width <= 0 || height <= 0 {
		return ""
	}

	status := fmt.Sprintf("Bank: %s | %s", currentViewBank(v.shared), v.pt.StatusLine())
	if v.pt.loading {
		status = fmt.Sprintf("Bank: %s | %s", currentViewBank(v.shared), v.pt.LoadingView())
	}
	if v.notice != "" {
		status += " | " + v.notice
	}
	filters := strings.Join([]string{
		focusedInputView("status", v.pt.inputs[0].input, v.pt.focus == 0),
		focusedInputView("type", v.pt.inputs[1].input, v.pt.focus == 1),
	}, "  ")

	contentWidth := max(20, width-2)
	leftWidth := max(20, contentWidth/2)
	rightWidth := contentWidth - leftWidth
	bodyHeight := max(4, height-8)
	v.pt.table.SetWidth(leftWidth - 2)
	v.pt.table.SetHeight(bodyHeight)
	v.pt.detail.SetWidth(max(10, rightWidth-2))
	v.pt.detail.SetHeight(bodyHeight)

	leftBody := v.pt.table.View()
	if v.pt.rowCount() == 0 && v.err == nil && !v.pt.loading {
		leftBody = "No operations."
	}
	if v.err != nil {
		leftBody = renderFriendlyError(v.err)
	}

	content := ui.TwoColumn(
		ui.Panel("Operations", leftBody, leftWidth),
		ui.Panel("Detail", v.pt.detail.View(), rightWidth),
		contentWidth,
	)
	footer := "[ ] page • / status filter • tab switch pane • enter apply • c copy • ctrl+r refresh"
	return ui.Lines(renderTabs(width, 0, []string{"Operations"}), filters, status, content, footer)
}

func (v *OperationsView) Title() string {
	return "Operations"
}

func (v *OperationsView) TextEntryFocused() bool {
	return v.pt.TextEntryFocused()
}

func (v *OperationsView) loadCmd() tea.Cmd {
	client := v.shared.Client
	bank := currentViewBank(v.shared)
	timeout := sharedTimeout(v.shared)
	status := strings.TrimSpace(v.pt.Value("status"))
	opType := strings.TrimSpace(v.pt.Value("type"))
	limit := v.pt.limit
	offset := v.pt.offset
	gen, tick := v.pt.BeginLoad()

	load := func() tea.Msg {
		if client == nil {
			return operationsLoadedMsg{gen: gen, err: fmt.Errorf("hindsight client is unavailable")}
		}
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()
		page, err := client.ListOperations(ctx, bank, status, opType, limit, offset)
		return operationsLoadedMsg{gen: gen, page: page, err: err}
	}
	return tea.Batch(load, tick)
}

func operationsColumns() []table.Column {
	return []table.Column{
		{Title: "ID", Width: 24},
		{Title: "Status", Width: 12},
		{Title: "Task Type", Width: 16},
		{Title: "Created", Width: 24},
	}
}

func (v *OperationsView) tableRows() []table.Row {
	rows := make([]table.Row, 0, len(v.rows))
	for _, row := range v.rows {
		rows = append(rows, table.Row{
			firstNonEmpty(row, "id"),
			firstNonEmpty(row, "status"),
			firstNonEmpty(row, "type", "task_type"),
			firstNonEmpty(row, "created_at", "started_at", "updated_at"),
		})
	}
	return rows
}

func (v *OperationsView) selectedJSON() string {
	selected := v.pt.table.Cursor()
	if selected < 0 || selected >= len(v.rows) {
		return "{}"
	}
	return ui.PrettyJSON(v.rows[selected])
}

func (v *OperationsView) updateSelectedDetail() {
	v.pt.SetDetail(v.selectedJSON())
}
