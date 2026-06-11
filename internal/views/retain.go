package views

import (
	"context"
	"fmt"
	"strings"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/spinner"
	"charm.land/bubbles/v2/textarea"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"

	"hindsight-tui/internal/domain"
	"hindsight-tui/internal/state"
	"hindsight-tui/internal/ui"
)

const (
	retainFocusBank = iota
	retainFocusContent
	retainFocusContext
	retainFocusTags
	retainFocusDocumentID
	retainFocusTimestamp
	retainFocusUpdateMode
	retainFocusMetadata
	retainFocusAsync
	retainFocusViewOps
)

type retainSubmittedMsg struct {
	response *domain.RetainResponse
	err      error
}

type RetainView struct {
	shared       *Shared
	focus        int
	showAdvanced bool
	loading      bool
	status       string
	err          error

	bank       textinput.Model
	content    textarea.Model
	context    textinput.Model
	tags       textinput.Model
	documentID textinput.Model
	timestamp  textinput.Model
	updateMode textinput.Model
	metadata   textarea.Model
	async      bool
	spin       spinner.Model
	response   *domain.RetainResponse
}

func NewRetainView(shared *Shared) *RetainView {
	bank := newTextInput("Bank", "default", activeBank(shared))
	content := newTextArea("Content", "Store one memory item")
	content.SetHeight(5)
	contextInput := newTextInput("Context", "Optional context", "")
	tags := newTextInput("Tags", "comma,separated", "")
	documentID := newTextInput("Document ID", "Optional", "")
	timestamp := newTextInput("Timestamp", "ISO 8601 or unset", "")
	updateMode := newTextInput("Update mode", "empty, replace, append", "")
	metadata := newTextArea("Metadata", "key=value")
	metadata.SetHeight(4)

	view := &RetainView{
		shared:     shared,
		focus:      retainFocusBank,
		bank:       bank,
		content:    content,
		context:    contextInput,
		tags:       tags,
		documentID: documentID,
		timestamp:  timestamp,
		updateMode: updateMode,
		metadata:   metadata,
		spin:       spinner.New(spinner.WithSpinner(spinner.MiniDot)),
	}
	return view
}

func (v *RetainView) Init() tea.Cmd {
	return v.setFocus(v.focus)
}

func (v *RetainView) Title() string {
	return "Retain"
}

func (v *RetainView) ApplyPrefill(bank, content, contextValue string, tags []string) {
	if strings.TrimSpace(bank) != "" {
		v.bank.SetValue(strings.TrimSpace(bank))
	}
	v.content.SetValue(content)
	v.context.SetValue(contextValue)
	v.tags.SetValue(strings.Join(tags, ", "))
	v.err = nil
	v.status = ""
}

func (v *RetainView) Update(msg tea.Msg) (View, tea.Cmd) {
	switch msg := msg.(type) {
	case retainSubmittedMsg:
		v.loading = false
		v.err = msg.err
		if msg.err != nil {
			v.status = ""
			return v, nil
		}
		v.err = nil
		v.response = msg.response
		v.content.SetValue("")
		v.context.SetValue("")
		v.tags.SetValue("")
		v.status = fmt.Sprintf("Stored %d item(s) in %s", msg.response.ItemsCount, msg.response.BankID)
		if msg.response.Async || msg.response.OperationID != nil || len(msg.response.OperationIDs) > 0 {
			hint := "Indexing may complete asynchronously; use Operations or retry recall in a few seconds."
			if len(msg.response.OperationIDs) > 0 {
				hint = "Indexing may complete asynchronously; select View Operations below to inspect them."
			}
			v.status = ui.Lines(
				v.status,
				retainOperationLine(msg.response),
				hint,
			)
		}
		if v.shared != nil && v.shared.State != nil && msg.response.BankID != "" {
			v.shared.State.ActiveBank = msg.response.BankID
		}
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
		if key.Matches(msg, v.shared.KeyMap.Save) {
			if cmd := v.submit(); cmd != nil {
				return v, tea.Batch(cmd, v.spin.Tick)
			}
			return v, nil
		}
		if v.focus == retainFocusAsync && (key.Matches(msg, v.shared.KeyMap.Select) || msg.String() == " ") {
			v.async = !v.async
			return v, nil
		}
		if v.focus == retainFocusViewOps && key.Matches(msg, v.shared.KeyMap.Select) {
			return v, func() tea.Msg { return NavigateMsg{Route: state.RouteOperations} }
		}
	}

	return v, v.updateFocused(msg)
}

func (v *RetainView) View(width, height int) string {
	body := []string{
		renderFocusedInput("Bank", v.bank.View(), v.focus == retainFocusBank),
		renderFocusedInput("Content", v.content.View(), v.focus == retainFocusContent),
		renderFocusedInput("Context", v.context.View(), v.focus == retainFocusContext),
		renderFocusedInput("Tags", v.tags.View(), v.focus == retainFocusTags),
	}
	if v.showAdvanced {
		body = append(body,
			renderFocusedInput("Document ID", v.documentID.View(), v.focus == retainFocusDocumentID),
			renderFocusedInput("Timestamp", v.timestamp.View(), v.focus == retainFocusTimestamp),
			renderFocusedInput("Update mode", v.updateMode.View(), v.focus == retainFocusUpdateMode),
			renderFocusedInput("Metadata", v.metadata.View(), v.focus == retainFocusMetadata),
			renderFocusedInput("Async", boolField(v.async), v.focus == retainFocusAsync),
		)
	} else {
		body = append(body, "Advanced fields hidden. Press a to edit metadata, timestamp, update mode, and async.")
	}
	if v.loading {
		body = append(body, "", v.spin.View()+" Retaining memory…")
	}
	if v.status != "" {
		body = append(body, "", v.status)
	}
	if v.err != nil {
		body = append(body, "", renderFriendlyError(v.err))
	}
	if v.response != nil && len(v.response.OperationIDs) > 0 {
		body = append(body, "", renderFocusedInput("View Operations", "enter", v.focus == retainFocusViewOps))
	}
	return ui.Panel("Retain", strings.Join(body, "\n\n"), width)
}

func (v *RetainView) moveFocus(delta int) tea.Cmd {
	return v.setFocus(moveFocusIn(v.focusOrder(), v.focus, delta))
}

func (v *RetainView) focusOrder() []int {
	order := []int{
		retainFocusBank,
		retainFocusContent,
		retainFocusContext,
		retainFocusTags,
	}
	if v.showAdvanced {
		order = append(order,
			retainFocusDocumentID,
			retainFocusTimestamp,
			retainFocusUpdateMode,
			retainFocusMetadata,
			retainFocusAsync,
		)
	}
	if v.response != nil && len(v.response.OperationIDs) > 0 {
		order = append(order, retainFocusViewOps)
	}
	return order
}

func (v *RetainView) focusVisible(focus int) bool {
	for _, candidate := range v.focusOrder() {
		if candidate == focus {
			return true
		}
	}
	return false
}

func (v *RetainView) setFocus(next int) tea.Cmd {
	v.focus = next
	v.bank.Blur()
	v.content.Blur()
	v.context.Blur()
	v.tags.Blur()
	v.documentID.Blur()
	v.timestamp.Blur()
	v.updateMode.Blur()
	v.metadata.Blur()

	switch next {
	case retainFocusBank:
		return v.bank.Focus()
	case retainFocusContent:
		return v.content.Focus()
	case retainFocusContext:
		return v.context.Focus()
	case retainFocusTags:
		return v.tags.Focus()
	case retainFocusDocumentID:
		return v.documentID.Focus()
	case retainFocusTimestamp:
		return v.timestamp.Focus()
	case retainFocusUpdateMode:
		return v.updateMode.Focus()
	case retainFocusMetadata:
		return v.metadata.Focus()
	default:
		return nil
	}
}

func (v *RetainView) updateFocused(msg tea.Msg) tea.Cmd {
	switch v.focus {
	case retainFocusBank:
		var cmd tea.Cmd
		v.bank, cmd = v.bank.Update(msg)
		return cmd
	case retainFocusContent:
		var cmd tea.Cmd
		v.content, cmd = v.content.Update(msg)
		return cmd
	case retainFocusContext:
		var cmd tea.Cmd
		v.context, cmd = v.context.Update(msg)
		return cmd
	case retainFocusTags:
		var cmd tea.Cmd
		v.tags, cmd = v.tags.Update(msg)
		return cmd
	case retainFocusDocumentID:
		var cmd tea.Cmd
		v.documentID, cmd = v.documentID.Update(msg)
		return cmd
	case retainFocusTimestamp:
		var cmd tea.Cmd
		v.timestamp, cmd = v.timestamp.Update(msg)
		return cmd
	case retainFocusUpdateMode:
		var cmd tea.Cmd
		v.updateMode, cmd = v.updateMode.Update(msg)
		return cmd
	case retainFocusMetadata:
		var cmd tea.Cmd
		v.metadata, cmd = v.metadata.Update(msg)
		return cmd
	default:
		return nil
	}
}

func (v *RetainView) TextEntryFocused() bool {
	return v.focus != retainFocusAsync && v.focus != retainFocusViewOps
}

func (v *RetainView) submit() tea.Cmd {
	bank := strings.TrimSpace(v.bank.Value())
	if err := ui.ValidateBankID(bank); err != nil {
		v.err = err
		v.status = ""
		return nil
	}
	content := v.content.Value()
	if strings.TrimSpace(content) == "" {
		v.err = fmt.Errorf("content is required")
		v.status = ""
		return nil
	}
	metadata, err := ui.ParseMetadataLines(v.metadata.Value())
	if err != nil {
		v.err = err
		v.status = ""
		return nil
	}
	updateMode := strings.TrimSpace(v.updateMode.Value())
	if updateMode != "" && updateMode != "replace" && updateMode != "append" {
		v.err = fmt.Errorf("update mode must be empty, replace, or append")
		v.status = ""
		return nil
	}

	v.err = nil
	v.status = ""
	v.loading = true
	v.response = nil
	if v.shared != nil && v.shared.State != nil {
		v.shared.State.ActiveBank = bank
	}

	request := domain.RetainRequest{
		Items: []domain.MemoryItem{{
			Content:    content,
			Context:    optionalString(v.context.Value()),
			Metadata:   metadata,
			DocumentID: optionalString(v.documentID.Value()),
			Tags:       ui.ParseTags(v.tags.Value()),
			Timestamp:  optionalString(v.timestamp.Value()),
			UpdateMode: optionalString(updateMode),
		}},
		Async: v.async,
	}

	client := v.shared.Client
	timeout := sharedTimeout(v.shared)
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()
		response, err := client.Retain(ctx, bank, request)
		return retainSubmittedMsg{response: response, err: err}
	}
}

func newTextInput(prompt, placeholder, value string) textinput.Model {
	input := textinput.New()
	input.Prompt = ""
	input.Placeholder = placeholder
	input.SetValue(value)
	return input
}

func newTextArea(prompt, placeholder string) textarea.Model {
	input := textarea.New()
	input.Prompt = ""
	input.Placeholder = placeholder
	input.ShowLineNumbers = false
	input.SetWidth(40)
	return input
}

func activeBank(shared *Shared) string {
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

func optionalString(value string) *string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	copy := value
	return &copy
}

func renderFocusedInput(label, body string, focused bool) string {
	prefix := "  "
	if focused {
		prefix = "> "
	}
	return ui.Lines(prefix+label, body)
}

func boolField(value bool) string {
	if value {
		return "[x]"
	}
	return "[ ]"
}

func retainOperationLine(response *domain.RetainResponse) string {
	ids := append([]string(nil), response.OperationIDs...)
	if response.OperationID != nil && *response.OperationID != "" && len(ids) == 0 {
		ids = append(ids, *response.OperationID)
	}
	if len(ids) == 0 {
		return ""
	}
	return "Operation IDs: " + strings.Join(ids, ", ")
}
