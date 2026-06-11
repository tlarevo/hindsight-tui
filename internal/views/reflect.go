package views

import (
	"context"
	"fmt"
	"strings"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/spinner"
	"charm.land/bubbles/v2/textarea"
	"charm.land/bubbles/v2/textinput"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"

	"hindsight-tui/internal/domain"
	"hindsight-tui/internal/ui"
)

const (
	reflectFocusQuestion = iota
	reflectFocusContext
	reflectFocusBank
	reflectFocusBudget
	reflectFocusMaxTokens
	reflectFocusWorld
	reflectFocusExperience
	reflectFocusObservation
	reflectFocusTags
	reflectFocusTagsMatch
	reflectFocusIncludeFacts
	reflectFocusIncludeToolCalls
	reflectFocusExcludeMentalModels
	reflectFocusResponse
	reflectFocusEvidence
	reflectFocusRetainAction
	reflectFocusCopyAction
	reflectFocusExportPath
)

type reflectSubmittedMsg struct {
	response *domain.ReflectResponse
	err      error
}

type ReflectView struct {
	shared       *Shared
	focus        int
	showAdvanced bool
	loading      bool
	status       string
	err          error
	exporting    bool
	lastRequest  *domain.ReflectRequest

	question            textarea.Model
	context             textarea.Model
	bank                textinput.Model
	budget              textinput.Model
	maxTokens           textinput.Model
	tags                textinput.Model
	tagsMatch           textinput.Model
	exportPath          textinput.Model
	factWorld           bool
	factExperience      bool
	factObservation     bool
	includeFacts        bool
	includeToolCalls    bool
	excludeMentalModels bool
	responseViewport    viewport.Model
	evidenceViewport    viewport.Model
	response            *domain.ReflectResponse
	spin                spinner.Model
}

func NewReflectView(shared *Shared) *ReflectView {
	question := newTextArea("Question", "Ask Hindsight a question")
	question.SetHeight(4)
	contextInput := newTextArea("Context", "Optional context")
	contextInput.SetHeight(4)
	bank := newTextInput("Bank", "default", activeBank(shared))
	budget := newTextInput("Budget", "low, mid, high", "low")
	maxTokens := newTextInput("Max tokens", "4096", "4096")
	tags := newTextInput("Tags", "comma,separated", "")
	tagsMatch := newTextInput("Tags match", "any, all, any_strict, all_strict", "any")
	exportPath := newTextInput("Export path", "~/reflection.md", "")

	responseViewport := viewport.New()
	responseViewport.SetContent("Run a reflection to render a response.")
	evidenceViewport := viewport.New()
	evidenceViewport.SetContent("Evidence appears here.")

	return &ReflectView{
		shared:           shared,
		question:         question,
		context:          contextInput,
		bank:             bank,
		budget:           budget,
		maxTokens:        maxTokens,
		tags:             tags,
		tagsMatch:        tagsMatch,
		exportPath:       exportPath,
		factWorld:        true,
		factExperience:   true,
		factObservation:  true,
		includeFacts:     true,
		responseViewport: responseViewport,
		evidenceViewport: evidenceViewport,
		spin:             spinner.New(spinner.WithSpinner(spinner.MiniDot)),
	}
}

func (v *ReflectView) Init() tea.Cmd {
	return v.setFocus(v.focus)
}

func (v *ReflectView) Title() string {
	return "Reflect"
}

func (v *ReflectView) ApplyPrefill(bank, query string) {
	if strings.TrimSpace(bank) != "" {
		v.bank.SetValue(strings.TrimSpace(bank))
	}
	v.question.SetValue(query)
	v.err = nil
	v.status = ""
}

func (v *ReflectView) Update(msg tea.Msg) (View, tea.Cmd) {
	switch msg := msg.(type) {
	case reflectSubmittedMsg:
		v.loading = false
		v.err = msg.err
		if msg.err != nil {
			v.status = ""
			return v, nil
		}
		v.err = nil
		v.response = msg.response
		v.status = "Reflection ready."
		v.responseViewport.SetContent(msg.response.Text)
		v.evidenceViewport.SetContent(renderReflectEvidence(msg.response))
		return v, nil
	case spinner.TickMsg:
		if v.loading {
			var cmd tea.Cmd
			v.spin, cmd = v.spin.Update(msg)
			return v, cmd
		}
		return v, nil
	case ExportedFileMsg:
		if msg.Err != nil {
			v.err = msg.Err
			v.status = ""
			return v, nil
		}
		v.err = nil
		v.exporting = false
		v.exportPath.SetValue("")
		v.status = "Exported reflection to " + msg.Path
		return v, v.setFocus(reflectFocusResponse)
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
			return v, tea.Batch(v.runReflect(strings.TrimSpace(v.bank.Value()), *v.lastRequest), v.spin.Tick)
		}
		if key.Matches(msg, v.shared.KeyMap.Export) && v.response != nil && !v.TextEntryFocused() {
			v.exporting = true
			return v, v.setFocus(reflectFocusExportPath)
		}
		if v.exporting && v.focus == reflectFocusExportPath && (key.Matches(msg, v.shared.KeyMap.Select) || key.Matches(msg, v.shared.KeyMap.Save)) {
			return v, v.exportReflection()
		}
		if key.Matches(msg, v.shared.KeyMap.Save) && !v.exporting {
			if cmd := v.submit(); cmd != nil {
				return v, tea.Batch(cmd, v.spin.Tick)
			}
			return v, nil
		}
		if v.isCheckboxFocus() && (key.Matches(msg, v.shared.KeyMap.Select) || msg.String() == " ") {
			v.toggleFocusedCheckbox()
			return v, nil
		}
		if v.response != nil && key.Matches(msg, v.shared.KeyMap.Copy) && !v.TextEntryFocused() {
			v.status = "Sent reflection to terminal clipboard"
			return v, tea.SetClipboard(v.response.Text)
		}
		if v.response != nil && v.focus == reflectFocusRetainAction && key.Matches(msg, v.shared.KeyMap.Select) {
			bank := strings.TrimSpace(v.bank.Value())
			content := v.response.Text
			return v, func() tea.Msg {
				return OpenRetainMsg{Bank: bank, Content: content, Context: "reflection", Tags: []string{"reflection"}}
			}
		}
		if v.response != nil && v.focus == reflectFocusCopyAction && key.Matches(msg, v.shared.KeyMap.Select) {
			v.status = "Sent reflection to terminal clipboard"
			return v, tea.SetClipboard(v.response.Text)
		}
	}

	cmds := []tea.Cmd{v.updateFocused(msg)}
	if v.focus == reflectFocusResponse {
		var cmd tea.Cmd
		v.responseViewport, cmd = v.responseViewport.Update(msg)
		cmds = append(cmds, cmd)
	}
	if v.focus == reflectFocusEvidence {
		var cmd tea.Cmd
		v.evidenceViewport, cmd = v.evidenceViewport.Update(msg)
		cmds = append(cmds, cmd)
	}
	return v, tea.Batch(cmds...)
}

func (v *ReflectView) View(width, height int) string {
	left := []string{
		renderFocusedInput("Question", v.question.View(), v.focus == reflectFocusQuestion),
		renderFocusedInput("Context", v.context.View(), v.focus == reflectFocusContext),
		renderFocusedInput("Bank", v.bank.View(), v.focus == reflectFocusBank),
		renderFocusedInput("Budget", v.budget.View(), v.focus == reflectFocusBudget),
		renderFocusedInput("Max tokens", v.maxTokens.View(), v.focus == reflectFocusMaxTokens),
		renderFocusedInput("Fact types", ui.Lines(
			renderFocusedInput("world", boolField(v.factWorld), v.focus == reflectFocusWorld),
			renderFocusedInput("experience", boolField(v.factExperience), v.focus == reflectFocusExperience),
			renderFocusedInput("observation", boolField(v.factObservation), v.focus == reflectFocusObservation),
		), false),
	}
	if v.showAdvanced {
		left = append(left,
			renderFocusedInput("Tags", v.tags.View(), v.focus == reflectFocusTags),
			renderFocusedInput("Tags match", v.tagsMatch.View(), v.focus == reflectFocusTagsMatch),
			renderFocusedInput("Include facts", boolField(v.includeFacts), v.focus == reflectFocusIncludeFacts),
			renderFocusedInput("Include tool calls", boolField(v.includeToolCalls), v.focus == reflectFocusIncludeToolCalls),
			renderFocusedInput("Exclude mental models", boolField(v.excludeMentalModels), v.focus == reflectFocusExcludeMentalModels),
		)
	} else {
		left = append(left, "Advanced fields hidden. Press a to edit tags and include flags.")
	}
	if v.loading {
		left = append(left, "", v.spin.View()+" Reflecting…")
	}
	if v.status != "" {
		left = append(left, "", v.status)
	}
	if v.err != nil {
		left = append(left, "", renderFriendlyError(v.err))
	}
	if v.exporting {
		left = append(left, "", renderFocusedInput("Export path", v.exportPath.View(), v.focus == reflectFocusExportPath))
	}

	rightWidth := width/2 - 6
	if rightWidth < 30 {
		rightWidth = 30
	}
	v.responseViewport.SetWidth(rightWidth)
	v.responseViewport.SetHeight(max(8, height/3))
	v.evidenceViewport.SetWidth(rightWidth)
	v.evidenceViewport.SetHeight(max(8, height/3))

	actions := []string{}
	if v.response != nil {
		actions = append(actions,
			renderFocusedInput("Retain this reflection", "enter", v.focus == reflectFocusRetainAction),
			renderFocusedInput("Copy Reflection", "c or enter", v.focus == reflectFocusCopyAction),
		)
	}
	right := ui.Lines(
		renderFocusedInput("Response", v.responseViewport.View(), v.focus == reflectFocusResponse),
		"",
		renderFocusedInput("Evidence", v.evidenceViewport.View(), v.focus == reflectFocusEvidence),
	)
	if len(actions) > 0 {
		right = ui.Lines(right, "", strings.Join(actions, "\n\n"))
	}
	return ui.TwoColumn(ui.Panel("Reflect", strings.Join(left, "\n\n"), width/2), ui.Panel("Response", right, width/2), width)
}

func (v *ReflectView) submit() tea.Cmd {
	bank := strings.TrimSpace(v.bank.Value())
	if err := ui.ValidateBankID(bank); err != nil {
		v.err = err
		v.status = ""
		return nil
	}
	question := strings.TrimSpace(v.question.Value())
	if question == "" {
		v.err = fmt.Errorf("question is required")
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
	factTypes := make([]string, 0, 3)
	if v.factWorld {
		factTypes = append(factTypes, "world")
	}
	if v.factExperience {
		factTypes = append(factTypes, "experience")
	}
	if v.factObservation {
		factTypes = append(factTypes, "observation")
	}
	include := map[string]any{}
	if v.includeFacts {
		include["facts"] = true
	}
	if v.includeToolCalls {
		include["tool_calls"] = true
	}
	if len(include) == 0 {
		include = nil
	}
	query := question
	if ctxValue := strings.TrimSpace(v.context.Value()); ctxValue != "" {
		query = fmt.Sprintf("Question:\n%s\n\nContext:\n%s", question, v.context.Value())
	}

	request := domain.ReflectRequest{
		Query:               query,
		Budget:              budget,
		MaxTokens:           maxTokens,
		Include:             include,
		Tags:                ui.ParseTags(v.tags.Value()),
		TagsMatch:           tagsMatch,
		FactTypes:           factTypes,
		ExcludeMentalModels: v.excludeMentalModels,
	}
	v.err = nil
	v.status = ""
	v.loading = true
	v.lastRequest = &request
	if v.shared != nil && v.shared.State != nil {
		v.shared.State.ActiveBank = bank
	}
	return v.runReflect(bank, request)
}

func (v *ReflectView) runReflect(bank string, request domain.ReflectRequest) tea.Cmd {
	copy := request
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), sharedTimeout(v.shared))
		defer cancel()
		response, err := v.shared.Client.Reflect(ctx, bank, copy)
		return reflectSubmittedMsg{response: response, err: err}
	}
}

func (v *ReflectView) exportReflection() tea.Cmd {
	if v.response == nil {
		return nil
	}
	path := strings.TrimSpace(v.exportPath.Value())
	if path == "" {
		v.err = fmt.Errorf("path is required")
		v.status = ""
		return nil
	}
	v.err = nil
	v.status = ""
	content := []byte(v.response.Text)
	return func() tea.Msg {
		err := ui.WritePrivateText(path, content)
		return ExportedFileMsg{Path: path, Err: err}
	}
}

func (v *ReflectView) moveFocus(delta int) tea.Cmd {
	return v.setFocus(moveFocusIn(v.focusOrder(), v.focus, delta))
}

func (v *ReflectView) focusOrder() []int {
	order := []int{
		reflectFocusQuestion,
		reflectFocusContext,
		reflectFocusBank,
		reflectFocusBudget,
		reflectFocusMaxTokens,
		reflectFocusWorld,
		reflectFocusExperience,
		reflectFocusObservation,
	}
	if v.showAdvanced {
		order = append(order,
			reflectFocusTags,
			reflectFocusTagsMatch,
			reflectFocusIncludeFacts,
			reflectFocusIncludeToolCalls,
			reflectFocusExcludeMentalModels,
		)
	}
	order = append(order, reflectFocusResponse, reflectFocusEvidence)
	if v.response != nil {
		order = append(order, reflectFocusRetainAction, reflectFocusCopyAction)
	}
	if v.exporting {
		order = append(order, reflectFocusExportPath)
	}
	return order
}

func (v *ReflectView) focusVisible(focus int) bool {
	for _, candidate := range v.focusOrder() {
		if candidate == focus {
			return true
		}
	}
	return false
}

func (v *ReflectView) setFocus(next int) tea.Cmd {
	v.focus = next
	v.question.Blur()
	v.context.Blur()
	v.bank.Blur()
	v.budget.Blur()
	v.maxTokens.Blur()
	v.tags.Blur()
	v.tagsMatch.Blur()
	v.exportPath.Blur()

	switch next {
	case reflectFocusQuestion:
		return v.question.Focus()
	case reflectFocusContext:
		return v.context.Focus()
	case reflectFocusBank:
		return v.bank.Focus()
	case reflectFocusBudget:
		return v.budget.Focus()
	case reflectFocusMaxTokens:
		return v.maxTokens.Focus()
	case reflectFocusTags:
		return v.tags.Focus()
	case reflectFocusTagsMatch:
		return v.tagsMatch.Focus()
	case reflectFocusExportPath:
		return v.exportPath.Focus()
	default:
		return nil
	}
}

func (v *ReflectView) TextEntryFocused() bool {
	switch v.focus {
	case reflectFocusQuestion, reflectFocusContext, reflectFocusBank, reflectFocusBudget, reflectFocusMaxTokens, reflectFocusTags, reflectFocusTagsMatch, reflectFocusExportPath:
		return true
	default:
		return false
	}
}

func (v *ReflectView) updateFocused(msg tea.Msg) tea.Cmd {
	switch v.focus {
	case reflectFocusQuestion:
		var cmd tea.Cmd
		v.question, cmd = v.question.Update(msg)
		return cmd
	case reflectFocusContext:
		var cmd tea.Cmd
		v.context, cmd = v.context.Update(msg)
		return cmd
	case reflectFocusBank:
		var cmd tea.Cmd
		v.bank, cmd = v.bank.Update(msg)
		return cmd
	case reflectFocusBudget:
		var cmd tea.Cmd
		v.budget, cmd = v.budget.Update(msg)
		return cmd
	case reflectFocusMaxTokens:
		var cmd tea.Cmd
		v.maxTokens, cmd = v.maxTokens.Update(msg)
		return cmd
	case reflectFocusTags:
		var cmd tea.Cmd
		v.tags, cmd = v.tags.Update(msg)
		return cmd
	case reflectFocusTagsMatch:
		var cmd tea.Cmd
		v.tagsMatch, cmd = v.tagsMatch.Update(msg)
		return cmd
	case reflectFocusExportPath:
		var cmd tea.Cmd
		v.exportPath, cmd = v.exportPath.Update(msg)
		return cmd
	default:
		return nil
	}
}

func (v *ReflectView) isCheckboxFocus() bool {
	switch v.focus {
	case reflectFocusWorld, reflectFocusExperience, reflectFocusObservation,
		reflectFocusIncludeFacts, reflectFocusIncludeToolCalls, reflectFocusExcludeMentalModels:
		return true
	default:
		return false
	}
}

func (v *ReflectView) toggleFocusedCheckbox() {
	switch v.focus {
	case reflectFocusWorld:
		v.factWorld = !v.factWorld
	case reflectFocusExperience:
		v.factExperience = !v.factExperience
	case reflectFocusObservation:
		v.factObservation = !v.factObservation
	case reflectFocusIncludeFacts:
		v.includeFacts = !v.includeFacts
	case reflectFocusIncludeToolCalls:
		v.includeToolCalls = !v.includeToolCalls
	case reflectFocusExcludeMentalModels:
		v.excludeMentalModels = !v.excludeMentalModels
	}
}

func renderReflectEvidence(response *domain.ReflectResponse) string {
	payload := map[string]any{}
	if len(response.BasedOn) > 0 {
		payload["based_on"] = response.BasedOn
	}
	if response.Usage != nil {
		payload["usage"] = response.Usage
	}
	if len(response.Trace) > 0 {
		payload["trace"] = response.Trace
	}
	if len(response.StructuredOutput) > 0 {
		payload["structured_output"] = response.StructuredOutput
	}
	if len(payload) == 0 {
		return "{}"
	}
	return ui.PrettyJSON(payload)
}
