package views

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/spinner"
	"charm.land/bubbles/v2/table"
	"charm.land/bubbles/v2/textinput"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"

	"hindsight-tui/internal/domain"
	"hindsight-tui/internal/ui"
)

const (
	recallFocusQuery = iota
	recallFocusBank
	recallFocusBudget
	recallFocusMaxTokens
	recallFocusWorld
	recallFocusExperience
	recallFocusObservation
	recallFocusTags
	recallFocusTagsMatch
	recallFocusIncludeEntities
	recallFocusIncludeChunks
	recallFocusIncludeSourceFacts
	recallFocusTrace
	recallFocusResults
	recallFocusDetail
)

type recallSubmittedMsg struct {
	response *domain.RecallResponse
	err      error
}

type RecallView struct {
	shared       *Shared
	focus        int
	showAdvanced bool
	loading      bool
	status       string
	err          error
	lastRequest  *domain.RecallRequest

	query           textinput.Model
	bank            textinput.Model
	budget          textinput.Model
	maxTokens       textinput.Model
	tags            textinput.Model
	tagsMatch       textinput.Model
	includeEntities bool
	includeChunks   bool
	includeSources  bool
	trace           bool
	world           bool
	experience      bool
	observation     bool
	results         table.Model
	detail          viewport.Model
	response        *domain.RecallResponse
	spin            spinner.Model
}

func NewRecallView(shared *Shared) *RecallView {
	query := newTextInput("Query", "What do you want to recall?", "")
	bank := newTextInput("Bank", "default", activeBank(shared))
	budget := newTextInput("Budget", "low, mid, high", "mid")
	maxTokens := newTextInput("Max tokens", "4096", "4096")
	tags := newTextInput("Tags", "comma,separated", "")
	tagsMatch := newTextInput("Tags match", "any, all, any_strict, all_strict", "any")

	results := table.New(
		table.WithColumns([]table.Column{{Title: "#", Width: 4}, {Title: "Type", Width: 14}, {Title: "Text", Width: 50}}),
		table.WithRows(nil),
		table.WithFocused(false),
		table.WithHeight(8),
	)
	results.Blur()

	detail := viewport.New()
	detail.SetContent("Run a recall to inspect a result.")

	return &RecallView{
		shared:          shared,
		query:           query,
		bank:            bank,
		budget:          budget,
		maxTokens:       maxTokens,
		tags:            tags,
		tagsMatch:       tagsMatch,
		includeEntities: true,
		world:           true,
		experience:      true,
		results:         results,
		detail:          detail,
		spin:            spinner.New(spinner.WithSpinner(spinner.MiniDot)),
	}
}

func (v *RecallView) Init() tea.Cmd {
	return v.setFocus(v.focus)
}

func (v *RecallView) Title() string {
	return "Recall"
}

func (v *RecallView) Update(msg tea.Msg) (View, tea.Cmd) {
	switch msg := msg.(type) {
	case recallSubmittedMsg:
		v.loading = false
		v.err = msg.err
		if msg.err != nil {
			v.status = ""
			return v, nil
		}
		v.err = nil
		v.response = msg.response
		v.status = fmt.Sprintf("Loaded %d result(s) from %s", len(msg.response.Results), strings.TrimSpace(v.bank.Value()))
		v.syncResults()
		return v, nil
	case spinner.TickMsg:
		if v.loading {
			var cmd tea.Cmd
			v.spin, cmd = v.spin.Update(msg)
			return v, cmd
		}
		return v, nil
	case tea.KeyMsg:
		if key.Matches(msg, v.shared.KeyMap.Advanced) && !v.TextEntryFocused() {
			v.showAdvanced = !v.showAdvanced
			if !v.focusVisible(v.focus) {
				return v, v.setFocus(v.focusOrder()[0])
			}
			return v, nil
		}
		if key.Matches(msg, v.shared.KeyMap.NextPane) {
			return v, v.moveFocus(1)
		}
		if key.Matches(msg, v.shared.KeyMap.PrevPane) {
			return v, v.moveFocus(-1)
		}
		if key.Matches(msg, v.shared.KeyMap.Refresh) && v.lastRequest != nil && !v.loading {
			v.loading = true
			v.err = nil
			return v, tea.Batch(v.runRecall(strings.TrimSpace(v.bank.Value()), *v.lastRequest), v.spin.Tick)
		}
		if key.Matches(msg, v.shared.KeyMap.Save) {
			if cmd := v.submit(); cmd != nil {
				return v, tea.Batch(cmd, v.spin.Tick)
			}
			return v, nil
		}
		if v.focus == recallFocusResults && key.Matches(msg, v.shared.KeyMap.Copy) {
			selected := v.selectedResult()
			if selected == nil {
				return v, nil
			}
			v.status = "Sent memory text to terminal clipboard"
			return v, tea.SetClipboard(selected.Text)
		}
		if v.focus == recallFocusResults && key.Matches(msg, v.shared.KeyMap.Reflect) {
			selected := v.selectedResult()
			if selected == nil {
				return v, nil
			}
			prefill := fmt.Sprintf("Use this recalled memory as context: %s\n\nQuestion: ", selected.Text)
			bank := strings.TrimSpace(v.bank.Value())
			return v, func() tea.Msg {
				return OpenReflectMsg{Bank: bank, Query: prefill}
			}
		}
		if v.isCheckboxFocus() && (key.Matches(msg, v.shared.KeyMap.Select) || msg.String() == " ") {
			v.toggleFocusedCheckbox()
			return v, nil
		}
	}

	cmds := []tea.Cmd{v.updateFocused(msg)}
	if v.focus == recallFocusResults {
		before := v.results.Cursor()
		var cmd tea.Cmd
		v.results, cmd = v.results.Update(msg)
		cmds = append(cmds, cmd)
		if before != v.results.Cursor() {
			v.syncDetail()
		}
	}
	if v.focus == recallFocusDetail {
		var cmd tea.Cmd
		v.detail, cmd = v.detail.Update(msg)
		cmds = append(cmds, cmd)
	}
	return v, tea.Batch(cmds...)
}

func (v *RecallView) View(width, height int) string {
	left := []string{
		renderFocusedInput("Query", v.query.View(), v.focus == recallFocusQuery),
		renderFocusedInput("Bank", v.bank.View(), v.focus == recallFocusBank),
		renderFocusedInput("Budget", v.budget.View(), v.focus == recallFocusBudget),
		renderFocusedInput("Max tokens", v.maxTokens.View(), v.focus == recallFocusMaxTokens),
		renderFocusedInput("Types", ui.Lines(
			renderFocusedInput("world", boolField(v.world), v.focus == recallFocusWorld),
			renderFocusedInput("experience", boolField(v.experience), v.focus == recallFocusExperience),
			renderFocusedInput("observation", boolField(v.observation), v.focus == recallFocusObservation),
		), false),
	}
	if v.showAdvanced {
		left = append(left,
			renderFocusedInput("Tags", v.tags.View(), v.focus == recallFocusTags),
			renderFocusedInput("Tags match", v.tagsMatch.View(), v.focus == recallFocusTagsMatch),
			renderFocusedInput("Include entities", boolField(v.includeEntities), v.focus == recallFocusIncludeEntities),
			renderFocusedInput("Include chunks", boolField(v.includeChunks), v.focus == recallFocusIncludeChunks),
			renderFocusedInput("Include source facts", boolField(v.includeSources), v.focus == recallFocusIncludeSourceFacts),
			renderFocusedInput("Trace", boolField(v.trace), v.focus == recallFocusTrace),
		)
	} else {
		left = append(left, "Advanced fields hidden. Press a to edit tags, include options, and trace.")
	}
	if v.loading {
		left = append(left, "", v.spin.View()+" Recalling memories…")
	}
	if v.status != "" {
		left = append(left, "", v.status)
	}
	if v.err != nil {
		left = append(left, "", renderFriendlyError(v.err))
	}

	tableWidth := width/2 - 6
	if tableWidth < 30 {
		tableWidth = 30
	}
	v.results.SetWidth(tableWidth)
	v.results.SetHeight(max(6, height/3))
	v.detail.SetWidth(tableWidth)
	v.detail.SetHeight(max(8, height/3))

	right := ui.Lines(
		renderFocusedInput("Results", v.results.View(), v.focus == recallFocusResults),
		"",
		renderFocusedInput("Detail", v.detail.View(), v.focus == recallFocusDetail),
	)
	return ui.TwoColumn(ui.Panel("Recall", strings.Join(left, "\n\n"), width/2), ui.Panel("Matches", right, width/2), width)
}

func (v *RecallView) submit() tea.Cmd {
	bank := strings.TrimSpace(v.bank.Value())
	if err := ui.ValidateBankID(bank); err != nil {
		v.err = err
		v.status = ""
		return nil
	}
	query := strings.TrimSpace(v.query.Value())
	if query == "" {
		v.err = fmt.Errorf("query is required")
		v.status = ""
		return nil
	}
	maxTokens, err := parsePositiveInt(v.maxTokens.Value(), "max tokens")
	if err != nil {
		v.err = err
		v.status = ""
		return nil
	}
	budget := strings.TrimSpace(v.budget.Value())
	if budget != "low" && budget != "mid" && budget != "high" {
		v.err = fmt.Errorf("budget must be low, mid, or high")
		v.status = ""
		return nil
	}
	tagsMatch := strings.TrimSpace(v.tagsMatch.Value())
	if tagsMatch != "" && tagsMatch != "any" && tagsMatch != "all" && tagsMatch != "any_strict" && tagsMatch != "all_strict" {
		v.err = fmt.Errorf("tags match must be any, all, any_strict, or all_strict")
		v.status = ""
		return nil
	}

	types := make([]string, 0, 3)
	if v.world {
		types = append(types, "world")
	}
	if v.experience {
		types = append(types, "experience")
	}
	if v.observation {
		types = append(types, "observation")
	}
	include := map[string]any{}
	if v.includeEntities {
		include["entities"] = map[string]any{"max_tokens": 500}
	}
	if v.includeChunks {
		include["chunks"] = map[string]any{"enabled": true}
	}
	if v.includeSources {
		include["source_facts"] = map[string]any{"enabled": true}
	}
	if len(include) == 0 {
		include = nil
	}

	request := domain.RecallRequest{
		Query:     query,
		Types:     types,
		Budget:    budget,
		MaxTokens: maxTokens,
		Trace:     v.trace,
		Include:   include,
		Tags:      ui.ParseTags(v.tags.Value()),
		TagsMatch: tagsMatch,
	}
	v.err = nil
	v.status = ""
	v.loading = true
	v.lastRequest = &request
	if v.shared != nil && v.shared.State != nil {
		v.shared.State.ActiveBank = bank
	}
	return v.runRecall(bank, request)
}

func (v *RecallView) runRecall(bank string, request domain.RecallRequest) tea.Cmd {
	copy := request
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), sharedTimeout(v.shared))
		defer cancel()
		response, err := v.shared.Client.Recall(ctx, bank, copy)
		return recallSubmittedMsg{response: response, err: err}
	}
}

func (v *RecallView) moveFocus(delta int) tea.Cmd {
	return v.setFocus(moveFocusIn(v.focusOrder(), v.focus, delta))
}

func (v *RecallView) focusOrder() []int {
	order := []int{
		recallFocusQuery,
		recallFocusBank,
		recallFocusBudget,
		recallFocusMaxTokens,
		recallFocusWorld,
		recallFocusExperience,
		recallFocusObservation,
	}
	if v.showAdvanced {
		order = append(order,
			recallFocusTags,
			recallFocusTagsMatch,
			recallFocusIncludeEntities,
			recallFocusIncludeChunks,
			recallFocusIncludeSourceFacts,
			recallFocusTrace,
		)
	}
	return append(order, recallFocusResults, recallFocusDetail)
}

func (v *RecallView) focusVisible(focus int) bool {
	for _, candidate := range v.focusOrder() {
		if candidate == focus {
			return true
		}
	}
	return false
}

func (v *RecallView) setFocus(next int) tea.Cmd {
	v.focus = next
	v.query.Blur()
	v.bank.Blur()
	v.budget.Blur()
	v.maxTokens.Blur()
	v.tags.Blur()
	v.tagsMatch.Blur()
	v.results.Blur()

	switch next {
	case recallFocusQuery:
		return v.query.Focus()
	case recallFocusBank:
		return v.bank.Focus()
	case recallFocusBudget:
		return v.budget.Focus()
	case recallFocusMaxTokens:
		return v.maxTokens.Focus()
	case recallFocusTags:
		return v.tags.Focus()
	case recallFocusTagsMatch:
		return v.tagsMatch.Focus()
	case recallFocusResults:
		v.results.Focus()
		return nil
	default:
		return nil
	}
}

func (v *RecallView) updateFocused(msg tea.Msg) tea.Cmd {
	switch v.focus {
	case recallFocusQuery:
		var cmd tea.Cmd
		v.query, cmd = v.query.Update(msg)
		return cmd
	case recallFocusBank:
		var cmd tea.Cmd
		v.bank, cmd = v.bank.Update(msg)
		return cmd
	case recallFocusBudget:
		var cmd tea.Cmd
		v.budget, cmd = v.budget.Update(msg)
		return cmd
	case recallFocusMaxTokens:
		var cmd tea.Cmd
		v.maxTokens, cmd = v.maxTokens.Update(msg)
		return cmd
	case recallFocusTags:
		var cmd tea.Cmd
		v.tags, cmd = v.tags.Update(msg)
		return cmd
	case recallFocusTagsMatch:
		var cmd tea.Cmd
		v.tagsMatch, cmd = v.tagsMatch.Update(msg)
		return cmd
	default:
		return nil
	}
}

func (v *RecallView) TextEntryFocused() bool {
	switch v.focus {
	case recallFocusQuery, recallFocusBank, recallFocusBudget, recallFocusMaxTokens, recallFocusTags, recallFocusTagsMatch:
		return true
	default:
		return false
	}
}

func (v *RecallView) isCheckboxFocus() bool {
	switch v.focus {
	case recallFocusWorld, recallFocusExperience, recallFocusObservation,
		recallFocusIncludeEntities, recallFocusIncludeChunks, recallFocusIncludeSourceFacts, recallFocusTrace:
		return true
	default:
		return false
	}
}

func (v *RecallView) toggleFocusedCheckbox() {
	switch v.focus {
	case recallFocusWorld:
		v.world = !v.world
	case recallFocusExperience:
		v.experience = !v.experience
	case recallFocusObservation:
		v.observation = !v.observation
	case recallFocusIncludeEntities:
		v.includeEntities = !v.includeEntities
	case recallFocusIncludeChunks:
		v.includeChunks = !v.includeChunks
	case recallFocusIncludeSourceFacts:
		v.includeSources = !v.includeSources
	case recallFocusTrace:
		v.trace = !v.trace
	}
}

func (v *RecallView) syncResults() {
	rows := make([]table.Row, 0)
	if v.response != nil {
		rows = make([]table.Row, 0, len(v.response.Results))
		for i, result := range v.response.Results {
			rows = append(rows, table.Row{
				fmt.Sprintf("%d", i+1),
				recallType(result.Type),
				ui.TruncateRunes(result.Text, 48),
			})
		}
	}
	v.results.SetRows(rows)
	if len(rows) == 0 {
		v.detail.SetContent("No recall results.")
		return
	}
	v.results.SetCursor(0)
	v.syncDetail()
}

func (v *RecallView) syncDetail() {
	selected := v.selectedResult()
	if selected == nil {
		v.detail.SetContent("No recall results.")
		return
	}
	v.detail.SetContent(formatRecallResult(*selected))
}

func (v *RecallView) selectedResult() *domain.RecallResult {
	if v.response == nil || len(v.response.Results) == 0 {
		return nil
	}
	index := v.results.Cursor()
	if index < 0 || index >= len(v.response.Results) {
		return nil
	}
	return &v.response.Results[index]
}

func recallType(value *string) string {
	if value == nil || *value == "" {
		return "unknown"
	}
	return *value
}

func formatRecallResult(result domain.RecallResult) string {
	parts := []string{
		"Text:\n" + result.Text,
		"Type: " + recallType(result.Type),
		"Entities: " + strings.Join(result.Entities, ", "),
		"Tags: " + strings.Join(result.Tags, ", "),
		"Context: " + derefOrEmpty(result.Context),
		"Document ID: " + derefOrEmpty(result.DocumentID),
		"Occurred start: " + derefOrEmpty(result.OccurredStart),
		"Occurred end: " + derefOrEmpty(result.OccurredEnd),
		"Mentioned at: " + derefOrEmpty(result.MentionedAt),
		"Chunk ID: " + derefOrEmpty(result.ChunkID),
		"Source fact IDs: " + strings.Join(result.SourceFactIDs, ", "),
		"Metadata:\n" + ui.PrettyJSON(result.Metadata),
	}
	return strings.Join(parts, "\n\n")
}

func parsePositiveInt(raw, label string) (int, error) {
	value, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil || value <= 0 {
		return 0, fmt.Errorf("%s must be a positive integer", label)
	}
	return value, nil
}

func derefOrEmpty(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
