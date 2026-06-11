package views

import (
	"fmt"

	"charm.land/bubbles/v2/spinner"
	"charm.land/bubbles/v2/table"
	"charm.land/bubbles/v2/textinput"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
)

const (
	ptFocusTable  = -1
	ptFocusDetail = -2
)

// pagedInput pairs a named filter with its text input so a pagedTable can
// expose values by name (replacing the per-view input-by-name switches).
type pagedInput struct {
	name  string
	input textinput.Model
}

// pagedTable bundles the filter inputs, results table, scrollable detail
// viewport, loading spinner, focus ring, and offset/limit pagination shared by
// the Explorer, Traces, and Operations listing views.
type pagedTable struct {
	inputs  []pagedInput
	table   table.Model
	detail  viewport.Model
	spin    spinner.Model
	focus   int
	limit   int
	offset  int
	total   int
	gen     int
	loading bool
}

func newPagedTable(inputs []pagedInput, columns []table.Column) pagedTable {
	tbl := newViewTable()
	if len(columns) > 0 {
		tbl.SetColumns(columns)
	}
	pt := pagedTable{
		inputs: inputs,
		table:  tbl,
		detail: viewport.New(viewport.WithWidth(40), viewport.WithHeight(8)),
		spin:   spinner.New(spinner.WithSpinner(spinner.MiniDot)),
		limit:  25,
		focus:  ptFocusTable,
	}
	if len(inputs) > 0 {
		pt.focus = 0
	}
	pt.applyFocus()
	return pt
}

// TextEntryFocused reports whether a filter input currently has focus, so the
// owning view can refuse to treat keystrokes as global navigation.
func (t *pagedTable) TextEntryFocused() bool {
	return t.focus >= 0
}

func (t *pagedTable) applyFocus() tea.Cmd {
	var cmd tea.Cmd
	for i := range t.inputs {
		if t.focus == i {
			cmd = t.inputs[i].input.Focus()
		} else {
			t.inputs[i].input.Blur()
		}
	}
	if t.focus == ptFocusTable {
		t.table.Focus()
	} else {
		t.table.Blur()
	}
	return cmd
}

// CycleFocus rotates focus through the inputs, then the table, then the detail
// pane, wrapping around.
func (t *pagedTable) CycleFocus(delta int) tea.Cmd {
	order := make([]int, 0, len(t.inputs)+2)
	for i := range t.inputs {
		order = append(order, i)
	}
	order = append(order, ptFocusTable, ptFocusDetail)

	cur := 0
	for i, f := range order {
		if f == t.focus {
			cur = i
			break
		}
	}
	n := len(order)
	t.focus = order[((cur+delta)%n+n)%n]
	return t.applyFocus()
}

// FocusInput moves focus directly to the input at idx (used by the search key).
func (t *pagedTable) FocusInput(idx int) tea.Cmd {
	if idx < 0 || idx >= len(t.inputs) {
		return nil
	}
	t.focus = idx
	return t.applyFocus()
}

// FocusTable moves focus to the results table (used after a tab switch).
func (t *pagedTable) FocusTable() tea.Cmd {
	t.focus = ptFocusTable
	return t.applyFocus()
}

// BeginLoad bumps the generation counter, marks the table loading, and returns
// the new generation together with the spinner's tick command.
func (t *pagedTable) BeginLoad() (int, tea.Cmd) {
	t.gen++
	t.loading = true
	return t.gen, t.spin.Tick
}

// EndLoad reports whether the response for gen is still current. A stale gen
// (superseded by a newer load) returns false so the caller drops the response.
func (t *pagedTable) EndLoad(gen int) bool {
	if gen != t.gen {
		return false
	}
	t.loading = false
	return true
}

// CancelLoad invalidates any in-flight load and stops the spinner. Used when a
// view abandons the current request (e.g. switching to a disabled tab).
func (t *pagedTable) CancelLoad() {
	t.gen++
	t.loading = false
}

// SetInputs swaps the active filter inputs. Callers keep ownership of the slice
// so input values persist across swaps (used by Traces' per-tab filter sets).
func (t *pagedTable) SetInputs(inputs []pagedInput) {
	t.inputs = inputs
}

// SetPage installs the rows and total for the freshly loaded page.
func (t *pagedTable) SetPage(rows []table.Row, total int) {
	t.table.SetRows(rows)
	t.table.SetCursor(0)
	t.total = total
}

// ResetOffset returns to the first page (used when filters change).
func (t *pagedTable) ResetOffset() {
	t.offset = 0
}

func (t *pagedTable) rowCount() int {
	return len(t.table.Rows())
}

// CanNext reports whether a further page exists. When the server reports a
// total it is authoritative; otherwise a full page implies there may be more.
func (t *pagedTable) CanNext() bool {
	if t.total > 0 {
		return t.offset+t.limit < t.total
	}
	return t.rowCount() == t.limit
}

// NextPage advances one page when possible, reporting whether the offset moved.
func (t *pagedTable) NextPage() bool {
	if !t.CanNext() {
		return false
	}
	t.offset += t.limit
	return true
}

// PrevPage steps back one page when possible, reporting whether the offset moved.
func (t *pagedTable) PrevPage() bool {
	if t.offset <= 0 {
		return false
	}
	t.offset = max(0, t.offset-t.limit)
	return true
}

// StatusLine summarizes the visible range and flags the last page.
func (t *pagedTable) StatusLine() string {
	rows := t.rowCount()
	var line string
	if t.total > 0 {
		first := t.offset + 1
		if rows == 0 {
			first = t.offset
		}
		line = fmt.Sprintf("rows %d–%d of %d", first, t.offset+rows, t.total)
	} else {
		line = fmt.Sprintf("offset %d", t.offset)
	}
	if !t.CanNext() {
		line += " · last page"
	}
	return line
}

// LoadingView renders the animated spinner with a loading label.
func (t *pagedTable) LoadingView() string {
	return t.spin.View() + " Loading…"
}

// UpdateFocused routes a message to the focused input, table, or detail pane,
// and advances the spinner while a load is in flight.
func (t *pagedTable) UpdateFocused(msg tea.Msg) tea.Cmd {
	if _, ok := msg.(spinner.TickMsg); ok {
		if !t.loading {
			return nil
		}
		var cmd tea.Cmd
		t.spin, cmd = t.spin.Update(msg)
		return cmd
	}

	switch {
	case t.focus == ptFocusTable:
		var cmd tea.Cmd
		t.table, cmd = t.table.Update(msg)
		return cmd
	case t.focus == ptFocusDetail:
		var cmd tea.Cmd
		t.detail, cmd = t.detail.Update(msg)
		return cmd
	default:
		var cmd tea.Cmd
		t.inputs[t.focus].input, cmd = t.inputs[t.focus].input.Update(msg)
		return cmd
	}
}

// Value returns the current text of the named input, or "" when absent.
func (t *pagedTable) Value(name string) string {
	for i := range t.inputs {
		if t.inputs[i].name == name {
			return t.inputs[i].input.Value()
		}
	}
	return ""
}

// SetDetail replaces the detail viewport content.
func (t *pagedTable) SetDetail(content string) {
	t.detail.SetContent(content)
}
