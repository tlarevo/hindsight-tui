package views

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/spinner"
	"charm.land/bubbles/v2/table"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	gloss "charm.land/lipgloss/v2"

	"github.com/tlarevo/hindsight-tui/internal/domain"
	"github.com/tlarevo/hindsight-tui/internal/hindsight"
	"github.com/tlarevo/hindsight-tui/internal/theme"
	"github.com/tlarevo/hindsight-tui/internal/ui"
)

type explorerTab int

const (
	explorerFacts explorerTab = iota
	explorerEntities
	explorerRelationships
	explorerDocuments
	explorerTags
)

type explorerLoadedMsg struct {
	gen         int
	tab         explorerTab
	page        *domain.Page[map[string]any]
	summary     string
	detail      any
	unsupported string
	err         error
}

type ExplorerView struct {
	shared *Shared
	tab    explorerTab
	pt     pagedTable

	rows          []map[string]any
	relationships string
	notice        string
	err           error
}

func NewExplorerView(shared *Shared) *ExplorerView {
	queryInput := newViewTextInput("query")
	queryInput.Prompt = "Query: "
	typeInput := newViewTextInput("type")
	typeInput.Prompt = "Type: "
	tagsInput := newViewTextInput("tag1,tag2")
	tagsInput.Prompt = "Tags: "
	return &ExplorerView{
		shared:        shared,
		tab:           explorerFacts,
		relationships: "No relationship graph loaded.",
		pt: newPagedTable([]pagedInput{
			{name: "query", input: queryInput},
			{name: "type", input: typeInput},
			{name: "tags", input: tagsInput},
		}, explorerColumns(explorerFacts, 40)),
	}
}

func (v *ExplorerView) Init() tea.Cmd {
	return v.loadCurrentTabCmd()
}

func (v *ExplorerView) Update(msg tea.Msg) (View, tea.Cmd) {
	switch msg := msg.(type) {
	case explorerLoadedMsg:
		if msg.tab != v.tab {
			return v, nil
		}
		if !v.pt.EndLoad(msg.gen) {
			return v, nil
		}
		v.err = msg.err
		v.notice = ""
		if msg.err != nil {
			return v, nil
		}
		if msg.unsupported != "" {
			v.rows = nil
			v.relationships = msg.unsupported
			v.notice = msg.unsupported
			v.pt.SetPage(nil, 0)
			v.pt.SetDetail("{}")
			return v, nil
		}
		if msg.page != nil {
			v.rows = msg.page.Items
			v.pt.SetPage(explorerRows(v.tab, v.rows), msg.page.Total)
			v.updateSelectedDetail()
			return v, nil
		}
		v.rows = nil
		v.relationships = msg.summary
		v.pt.SetDetail(ui.PrettyJSON(msg.detail))
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
			return v, v.loadCurrentTabCmd()
		}

		if v.pt.TextEntryFocused() {
			if key.Matches(msg, v.shared.KeyMap.Select) {
				v.pt.ResetOffset()
				return v, v.loadCurrentTabCmd()
			}
			return v, v.pt.UpdateFocused(msg)
		}

		switch {
		case key.Matches(msg, v.shared.KeyMap.Search):
			return v, v.pt.FocusInput(0)
		case key.Matches(msg, v.shared.KeyMap.Copy) && v.usesTable():
			v.notice = "Copied JSON to clipboard"
			return v, tea.SetClipboard(v.selectedJSON())
		case msg.String() == "left":
			v.prevTab()
			return v, tea.Batch(v.pt.FocusTable(), v.loadCurrentTabCmd())
		case msg.String() == "right":
			v.nextTab()
			return v, tea.Batch(v.pt.FocusTable(), v.loadCurrentTabCmd())
		case msg.String() == "[":
			if v.pt.PrevPage() {
				return v, v.loadCurrentTabCmd()
			}
			return v, nil
		case msg.String() == "]":
			if v.pt.NextPage() {
				return v, v.loadCurrentTabCmd()
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
func (v *ExplorerView) View(width, height int) string {
	if width <= 0 || height <= 0 {
		return ""
	}
	p := v.shared.Palette

	tabs := renderTabs(p, width, int(v.tab), []string{"Facts", "Entities", "Relationships", "Documents", "Tags"})
	filters := v.renderFilters(width)
	status := p.StatusLabel("Bank", currentViewBank(v.shared), "neutral") + p.Muted.Render(" │ ") +
		p.StatusLabel("Tab", explorerTabLabel(v.tab), "neutral") + p.Muted.Render(" │ ") +
		p.Muted.Render(v.pt.StatusLine())
	if v.pt.loading {
		status = p.StatusLabel("Bank", currentViewBank(v.shared), "neutral") + p.Muted.Render(" │ ") +
			p.StatusLabel("Tab", explorerTabLabel(v.tab), "neutral") + p.Muted.Render(" │ ") +
			p.Spinner.Render(v.pt.LoadingView())
	}
	if v.notice != "" {
		status += p.Muted.Render(" │ ") + p.Warning.Render(v.notice)
	}

	contentWidth := max(20, width-2)
	leftWidth := max(20, contentWidth/2)
	rightWidth := contentWidth - leftWidth
	bodyHeight := max(4, height-8)
	v.pt.detail.SetWidth(max(10, rightWidth-4))
	v.pt.detail.SetHeight(bodyHeight)

	var content string
	if v.usesTable() {
		v.pt.table.SetWidth(leftWidth - 4)
		v.pt.table.SetHeight(bodyHeight)
		v.pt.table.SetColumns(explorerColumns(v.tab, leftWidth-4))
		leftBody := v.pt.table.View()
		if v.pt.rowCount() == 0 && v.notice == "" && v.err == nil && !v.pt.loading {
			leftBody = p.Muted.Render("No rows.")
		}
		if v.notice != "" {
			leftBody = p.Warning.Render(v.notice)
		}
		if v.err != nil {
			leftBody = renderFriendlyError(v.err)
		}
		content = ui.TwoColumn(
			p.Panel(explorerTabLabel(v.tab), leftBody, leftWidth),
			p.Panel("Detail", v.pt.detail.View(), rightWidth),
			contentWidth,
		)
	} else {
		body := v.relationships
		if v.err != nil {
			body = renderFriendlyError(v.err)
		} else if v.notice != "" {
			body = p.Warning.Render(v.notice)
		}
		content = ui.TwoColumn(
			p.Panel("Relationships", body, leftWidth),
			p.Panel("Graph JSON", v.pt.detail.View(), rightWidth),
			contentWidth,
		)
	}

	footer := p.Footer.Render("left/right tab • [ ] page • / filter • tab pane • c copy • enter apply • ctrl+r refresh")
	return ui.Lines(tabs, filters, status, content, footer)
}

func (v *ExplorerView) Title() string {
	return "Explorer"
}

func (v *ExplorerView) TextEntryFocused() bool {
	return v.pt.TextEntryFocused()
}

func (v *ExplorerView) renderFilters(width int) string {
	p := v.shared.Palette
	parts := []string{focusedInputView(p, "/ search", v.pt.inputs[0].input, v.pt.focus == 0)}
	switch v.tab {
	case explorerFacts, explorerRelationships:
		parts = append(parts, focusedInputView(p, "memory type", v.pt.inputs[1].input, v.pt.focus == 1))
	}
	if v.tab == explorerDocuments || v.tab == explorerRelationships {
		parts = append(parts, focusedInputView(p, "tags", v.pt.inputs[2].input, v.pt.focus == 2))
	}
	line := strings.Join(parts, "  ")
	return gloss.NewStyle().Width(width).Render(line)
}

func (v *ExplorerView) loadCurrentTabCmd() tea.Cmd {
	client := v.shared.Client
	bank := currentViewBank(v.shared)
	tab := v.tab
	timeout := sharedTimeout(v.shared)
	query := strings.TrimSpace(v.pt.Value("query"))
	memoryType := strings.TrimSpace(v.pt.Value("type"))
	tags := ui.ParseTags(v.pt.Value("tags"))
	limit := v.pt.limit
	offset := v.pt.offset
	gen, tick := v.pt.BeginLoad()

	load := func() tea.Msg {
		if client == nil {
			return explorerLoadedMsg{gen: gen, tab: tab, err: fmt.Errorf("hindsight client is unavailable")}
		}
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()
		switch tab {
		case explorerFacts:
			page, err := client.ListMemories(ctx, bank, query, memoryType, limit, offset)
			if hindsight.IsUnsupported(err) {
				return explorerLoadedMsg{gen: gen, tab: tab, unsupported: unsupportedListingMessage("facts")}
			}
			return explorerLoadedMsg{gen: gen, tab: tab, page: page, err: err}
		case explorerEntities:
			page, err := client.ListEntities(ctx, bank, query, limit, offset)
			if hindsight.IsUnsupported(err) {
				return explorerLoadedMsg{gen: gen, tab: tab, unsupported: unsupportedListingMessage("entities")}
			}
			return explorerLoadedMsg{gen: gen, tab: tab, page: page, err: err}
		case explorerRelationships:
			memoryGraph, err := client.GetMemoryGraph(ctx, bank, memoryType, query, tags, limit)
			if hindsight.IsUnsupported(err) {
				return explorerLoadedMsg{gen: gen, tab: tab, unsupported: unsupportedListingMessage("relationships")}
			}
			if err != nil {
				return explorerLoadedMsg{gen: gen, tab: tab, err: err}
			}
			entityGraph, err := client.GetEntityGraph(ctx, bank, limit)
			if hindsight.IsUnsupported(err) {
				return explorerLoadedMsg{gen: gen, tab: tab, unsupported: unsupportedListingMessage("relationships")}
			}
			if err != nil {
				return explorerLoadedMsg{gen: gen, tab: tab, err: err}
			}
			detail := map[string]any{
				"memory_graph": decodeJSONMap(memoryGraph),
				"entity_graph": decodeJSONMap(entityGraph),
			}
			memoryNodes, memoryEdges := graphCounts(memoryGraph)
			entityNodes, entityEdges := graphCounts(entityGraph)
			summary := fmt.Sprintf("Memory graph: %d nodes, %d edges\nEntity graph: %d nodes, %d edges", memoryNodes, memoryEdges, entityNodes, entityEdges)
			return explorerLoadedMsg{gen: gen, tab: tab, summary: summary, detail: detail}
		case explorerDocuments:
			page, err := client.ListDocuments(ctx, bank, query, tags, limit, offset)
			if hindsight.IsUnsupported(err) {
				return explorerLoadedMsg{gen: gen, tab: tab, unsupported: unsupportedListingMessage("documents")}
			}
			return explorerLoadedMsg{gen: gen, tab: tab, page: page, err: err}
		case explorerTags:
			page, err := client.ListTags(ctx, bank, query, limit, offset)
			if hindsight.IsUnsupported(err) {
				return explorerLoadedMsg{gen: gen, tab: tab, unsupported: unsupportedListingMessage("tags")}
			}
			return explorerLoadedMsg{gen: gen, tab: tab, page: page, err: err}
		default:
			return explorerLoadedMsg{gen: gen, tab: tab, err: fmt.Errorf("unknown explorer tab")}
		}
	}
	return tea.Batch(load, tick)
}

func (v *ExplorerView) usesTable() bool {
	return v.tab != explorerRelationships
}

func (v *ExplorerView) prevTab() {
	v.tab = explorerTab((int(v.tab) + 4) % 5)
	v.pt.ResetOffset()
}

func (v *ExplorerView) nextTab() {
	v.tab = explorerTab((int(v.tab) + 1) % 5)
	v.pt.ResetOffset()
}

func (v *ExplorerView) selectedJSON() string {
	selected := v.pt.table.Cursor()
	if selected < 0 || selected >= len(v.rows) {
		return "{}"
	}
	return ui.PrettyJSON(v.rows[selected])
}

func (v *ExplorerView) updateSelectedDetail() {
	if !v.usesTable() {
		return
	}
	v.pt.SetDetail(v.selectedJSON())
}

func explorerTabLabel(tab explorerTab) string {
	switch tab {
	case explorerFacts:
		return "Facts"
	case explorerEntities:
		return "Entities"
	case explorerRelationships:
		return "Relationships"
	case explorerDocuments:
		return "Documents"
	case explorerTags:
		return "Tags"
	default:
		return "Explorer"
	}
}

func explorerColumns(tab explorerTab, width int) []table.Column {
	quarter := max(8, width/4)
	half := max(16, width/2)
	switch tab {
	case explorerFacts:
		return []table.Column{{Title: "ID", Width: quarter}, {Title: "Type", Width: 12}, {Title: "Tags", Width: quarter}, {Title: "Text", Width: half}}
	case explorerEntities:
		return []table.Column{{Title: "Entity", Width: half}, {Title: "Type", Width: 16}, {Title: "Count", Width: 8}, {Title: "Memory IDs", Width: quarter}}
	case explorerDocuments:
		return []table.Column{{Title: "ID", Width: quarter}, {Title: "Title", Width: half}, {Title: "Tags", Width: quarter}, {Title: "Updated", Width: 20}}
	case explorerTags:
		return []table.Column{{Title: "Tag", Width: half}, {Title: "Count", Width: 8}, {Title: "Source", Width: 16}, {Title: "Sample", Width: quarter}}
	default:
		return []table.Column{{Title: "Value", Width: width}}
	}
}

func explorerRows(tab explorerTab, rows []map[string]any) []table.Row {
	out := make([]table.Row, 0, len(rows))
	for _, row := range rows {
		switch tab {
		case explorerFacts:
			out = append(out, table.Row{
				mapString(row["id"]),
				firstNonEmpty(row, "type", "memory_type"),
				joinValue(row["tags"]),
				firstNonEmpty(row, "text", "content"),
			})
		case explorerEntities:
			out = append(out, table.Row{
				firstNonEmpty(row, "entity", "name", "id"),
				firstNonEmpty(row, "type", "kind", "category"),
				firstNonEmpty(row, "count", "mentions", "memory_count"),
				joinValue(firstExisting(row, "memory_ids", "memories")),
			})
		case explorerDocuments:
			out = append(out, table.Row{
				firstNonEmpty(row, "document_id", "id"),
				firstNonEmpty(row, "title", "name", "filename"),
				joinValue(row["tags"]),
				firstNonEmpty(row, "updated_at", "created_at"),
			})
		case explorerTags:
			out = append(out, table.Row{
				firstNonEmpty(row, "tag", "name", "value"),
				firstNonEmpty(row, "count", "uses", "memory_count"),
				firstNonEmpty(row, "source"),
				firstNonEmpty(row, "sample", "example"),
			})
		}
	}
	return out
}

func newViewTextInput(placeholder string) textinput.Model {
	input := textinput.New()
	input.Placeholder = placeholder
	input.SetWidth(24)
	return input
}

func newViewTable() table.Model {
	model := table.New(table.WithFocused(true), table.WithHeight(8))
	model.Focus()
	return model
}

func focusedInputView(p theme.Palette, label string, input textinput.Model, focused bool) string {
	if focused {
		style := gloss.NewStyle().Padding(0, 1).Border(gloss.RoundedBorder()).BorderForeground(p.FocusedBorderColor)
		return style.Render(p.FocusedLabel.Render(label) + " " + input.View())
	}
	return p.Muted.Render(label) + " " + input.View()
}

func renderTabs(p theme.Palette, width int, active int, labels []string) string {
	parts := make([]string, 0, len(labels))
	for i, label := range labels {
		if i == active {
			parts = append(parts, p.TabActive.Render(label))
			continue
		}
		parts = append(parts, p.TabInactive.Render(label))
	}
	return gloss.NewStyle().Width(width).Render(strings.Join(parts, " "))
}

func currentViewBank(shared *Shared) string {
	if shared == nil {
		return "default"
	}
	if shared.State != nil && strings.TrimSpace(shared.State.ActiveBank) != "" {
		return strings.TrimSpace(shared.State.ActiveBank)
	}
	if shared.Config != nil && strings.TrimSpace(shared.Config.DefaultBank) != "" {
		return strings.TrimSpace(shared.Config.DefaultBank)
	}
	return "default"
}

func unsupportedListingMessage(resource string) string {
	return fmt.Sprintf("This Hindsight API version does not expose %s listing", resource)
}

func firstExisting(row map[string]any, keys ...string) any {
	for _, key := range keys {
		if value, ok := row[key]; ok && value != nil {
			return value
		}
	}
	return nil
}

func firstNonEmpty(row map[string]any, keys ...string) string {
	for _, key := range keys {
		if value, ok := row[key]; ok {
			text := mapString(value)
			if strings.TrimSpace(text) != "" {
				return text
			}
		}
	}
	return ""
}

func mapString(value any) string {
	switch typed := value.(type) {
	case nil:
		return ""
	case string:
		return typed
	case fmt.Stringer:
		return typed.String()
	case json.RawMessage:
		return string(typed)
	case []string:
		return strings.Join(typed, ", ")
	case []any:
		items := make([]string, 0, len(typed))
		for _, item := range typed {
			items = append(items, mapString(item))
		}
		return strings.Join(items, ", ")
	default:
		return fmt.Sprint(value)
	}
}

func joinValue(value any) string {
	return mapString(value)
}

func decodeJSONMap(raw json.RawMessage) map[string]any {
	if len(raw) == 0 {
		return map[string]any{}
	}
	var decoded map[string]any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		return map[string]any{"raw": string(raw), "decode_error": err.Error()}
	}
	return decoded
}

func graphCounts(raw json.RawMessage) (nodes int, edges int) {
	decoded := decodeJSONMap(raw)
	nodes = len(anySlice(decoded["nodes"]))
	edges = len(anySlice(decoded["edges"]))
	return nodes, edges
}

func anySlice(value any) []any {
	slice, ok := value.([]any)
	if !ok {
		return nil
	}
	return slice
}
