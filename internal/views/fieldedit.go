package views

import (
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
)

// fieldEditor wraps a textinput for inline editing of a single form field.
// Editing is an explicit mode: a field is only edited while active, so global
// navigation keys keep working when the editor is closed.
type fieldEditor struct {
	input  textinput.Model
	active bool
}

// Start opens the editor seeded with value, placing the cursor at the end.
func (e *fieldEditor) Start(value string) tea.Cmd {
	input := textinput.New()
	input.SetWidth(48)
	input.SetValue(value)
	input.CursorEnd()
	e.input = input
	e.active = true
	return e.input.Focus()
}

// Update forwards a message to the underlying textinput.
func (e *fieldEditor) Update(msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd
	e.input, cmd = e.input.Update(msg)
	return cmd
}

// Value returns the current edited text.
func (e *fieldEditor) Value() string {
	return e.input.Value()
}

// Stop closes the editor without committing.
func (e *fieldEditor) Stop() {
	e.input.Blur()
	e.active = false
}

// View renders the editing input.
func (e *fieldEditor) View() string {
	return e.input.View()
}
