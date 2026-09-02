// Package datatable renders a scrollable, selectable table of rows.
//
// It exists because the widget it replaced could not hold colour. That one
// truncated a cell by counting the bytes of the rendered string, so an escape
// sequence counted toward the width and a cell cut in the middle of one lost
// its reset and bled colour across the screen; and it applied the selected
// background by wrapping the whole row, so the first coloured cell in a row
// ended the highlight. Both are structural, and both are why cells here carry
// plain text and a style rather than rendered output.
//
// It also only renders the rows which are on screen, which is what makes a
// list of several thousand spans cost the same as a list of ten.
package datatable

import (
	tea "github.com/charmbracelet/bubbletea"
)

// headerHeight is the column titles and the rule beneath them.
const headerHeight = 2

// MinWidth below which a table cannot say anything useful.
const MinWidth = 24

// Model of a table.
type Model struct {
	columns []Column

	// rows is circular storage. rowStart is logical row zero and rowLen is the
	// number of populated slots, allowing bounded front insertion without
	// shifting every retained row.
	rows     []Row
	rowStart int
	rowLen   int

	// cursor is the row the reader is on, and offset is the first row on
	// screen. They are only ever changed together, through clamp, which is why
	// they cannot drift apart.
	cursor int
	offset int

	width  int
	height int

	focused bool
	empty   string
	keys    KeyMap
	rail    bool

	// Recomputed when the size or the columns change, never per frame.
	widths []int
}

// Option for a table.
type Option func(*Model)

// WithColumns sets the columns.
func WithColumns(columns ...Column) Option {
	return func(m *Model) {
		m.columns = columns
	}
}

// WithEmptyMessage sets what is shown when there are no rows.
func WithEmptyMessage(message string) Option {
	return func(m *Model) {
		m.empty = message
	}
}

// WithSelectionRail draws a marker in a reserved column at each end of the
// selected row.
//
// The background wash already says which row the cursor is on, but it is the
// first thing to go when the terminal is down to sixteen colours and has no
// backgrounds at all. The rail is a glyph, so it survives that.
//
// At each end rather than only at the left, because a row is as wide as the
// terminal: the marker beside the first column is no help to an eye which is
// already out at the last one.
func WithSelectionRail() Option {
	return func(m *Model) {
		m.rail = true
	}
}

// WithKeyMap sets the key bindings.
func WithKeyMap(keys KeyMap) Option {
	return func(m *Model) {
		m.keys = keys
	}
}

// New table.
func New(opts ...Option) *Model {
	m := &Model{
		keys:    DefaultKeyMap(),
		focused: true,
	}

	for _, opt := range opts {
		opt(m)
	}

	m.resolve()

	return m
}

// SetColumns of the table.
func (m *Model) SetColumns(columns []Column) {
	m.columns = columns
	m.resolve()
}

// Columns of the table.
func (m *Model) Columns() []Column {
	return m.columns
}

// SetRows of the table.
//
// The cursor is kept where it is rather than reset, so a list which is being
// appended to while it is read does not jump under the reader.
func (m *Model) SetRows(rows []Row) {
	m.rows = rows
	m.rowStart = 0
	m.rowLen = len(rows)
	m.clamp()
}

// PrependRowBounded adds a logical first row in O(1), retaining at most limit
// rows. The top cursor follows the live edge; a cursor below it and its
// viewport move with the selected logical row while that row remains retained.
func (m *Model) PrependRowBounded(row Row, limit int) {
	if limit <= 0 {
		m.SetRows(nil)
		return
	}

	m.ensureRowCapacity(limit)

	hadRows := m.rowLen > 0
	m.rowStart = (m.rowStart - 1 + len(m.rows)) % len(m.rows)
	m.rows[m.rowStart] = row
	if m.rowLen < limit {
		m.rowLen++
	}

	if hadRows && m.cursor > 0 {
		m.cursor++
		m.offset++
	}
	m.clamp()
}

// SetRow replaces one logical row without rebuilding the table.
func (m *Model) SetRow(index int, row Row) bool {
	if index < 0 || index >= m.rowLen {
		return false
	}

	m.rows[m.rowIndex(index)] = row
	return true
}

// TrimRows discards logical rows from the end.
func (m *Model) TrimRows(limit int) {
	limit = max(limit, 0)
	for m.rowLen > limit {
		m.rowLen--
		index := m.rowIndex(m.rowLen)
		m.rows[index] = nil
	}
	m.clamp()
}

func (m *Model) ensureRowCapacity(limit int) {
	if len(m.rows) == limit {
		return
	}

	keep := min(m.rowLen, limit)
	rows := make([]Row, limit)
	for i := range keep {
		rows[i] = m.rowAt(i)
	}

	m.rows = rows
	m.rowStart = 0
	m.rowLen = keep
}

func (m *Model) rowIndex(index int) int {
	return (m.rowStart + index) % len(m.rows)
}

func (m *Model) rowAt(index int) Row {
	return m.rows[m.rowIndex(index)]
}

// Rows in display order.
func (m *Model) Rows() []Row {
	rows := make([]Row, m.rowLen)
	for i := range rows {
		rows[i] = m.rowAt(i)
	}

	return rows
}

// Len is how many rows the table holds.
func (m *Model) Len() int {
	return m.rowLen
}

// Cursor position.
func (m *Model) Cursor() int {
	return m.cursor
}

// SetCursor to a row.
func (m *Model) SetCursor(row int) {
	m.cursor = row
	m.clamp()
}

// Offset is the first row on screen.
func (m *Model) Offset() int {
	return m.offset
}

// SelectedRow, and whether there was one.
func (m *Model) SelectedRow() (Row, bool) {
	if m.cursor < 0 || m.cursor >= m.rowLen {
		return nil, false
	}

	return m.rowAt(m.cursor), true
}

// MoveUp the cursor.
func (m *Model) MoveUp(n int) {
	m.cursor -= n
	m.clamp()
}

// MoveDown the cursor.
func (m *Model) MoveDown(n int) {
	m.cursor += n
	m.clamp()
}

// GotoTop of the rows.
func (m *Model) GotoTop() {
	m.cursor = 0
	m.clamp()
}

// GotoBottom of the rows.
func (m *Model) GotoBottom() {
	m.cursor = m.rowLen - 1
	m.clamp()
}

// SetSize of the table.
func (m *Model) SetSize(width, height int) {
	m.width = width
	m.height = height

	m.resolve()
	m.clamp()
}

// Width of the table.
func (m *Model) Width() int {
	return m.width
}

// Height of the table.
func (m *Model) Height() int {
	return m.height
}

// Focus the table, so that it takes key presses and shows its cursor.
func (m *Model) Focus() {
	m.focused = true
}

// Blur the table.
func (m *Model) Blur() {
	m.focused = false
}

// Focused reports whether the table has focus.
func (m *Model) Focused() bool {
	return m.focused
}

// VisibleRows currently on screen. Exposed so that tests can assert on what is
// shown rather than on escape sequences.
func (m *Model) VisibleRows() []Row {
	from, to := m.window()
	rows := make([]Row, 0, to-from)
	for i := from; i < to; i++ {
		rows = append(rows, m.rowAt(i))
	}

	return rows
}

// Update the table.
func (m *Model) Update(msg tea.Msg) (*Model, tea.Cmd) {
	if !m.focused {
		return m, nil
	}

	// Named rather than shadowing msg: a short declaration which reuses a name
	// already in this scope assigns to it instead of declaring a new one, so
	// the type assertion would be thrown away.
	pressed, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}

	switch {
	case keyMatches(pressed, m.keys.Up):
		m.MoveUp(1)
	case keyMatches(pressed, m.keys.Down):
		m.MoveDown(1)
	case keyMatches(pressed, m.keys.PageUp):
		m.MoveUp(m.visibleHeight())
	case keyMatches(pressed, m.keys.PageDown):
		m.MoveDown(m.visibleHeight())
	case keyMatches(pressed, m.keys.Top):
		m.GotoTop()
	case keyMatches(pressed, m.keys.Bottom):
		m.GotoBottom()
	}

	return m, nil
}

// visibleHeight is how many rows fit on screen.
func (m *Model) visibleHeight() int {
	return max(m.height-headerHeight, 1)
}

// window of rows currently on screen, as a half open range.
func (m *Model) window() (int, int) {
	from := min(m.offset, m.rowLen)
	to := min(from+m.visibleHeight(), m.rowLen)

	return from, to
}

// clamp keeps the cursor inside the rows and the window around the cursor.
//
// Every move goes through here. That is the whole reason the cursor and the
// scroll position cannot disagree: there is one place where either changes.
func (m *Model) clamp() {
	if m.rowLen == 0 {
		m.cursor, m.offset = 0, 0

		return
	}

	visible := m.visibleHeight()

	m.cursor = min(max(m.cursor, 0), m.rowLen-1)

	// The window follows the cursor when it walks off either edge...
	m.offset = min(m.offset, m.cursor)
	m.offset = max(m.offset, m.cursor-visible+1)

	// ...and never scrolls past the end into blank space.
	m.offset = min(m.offset, max(m.rowLen-visible, 0))
	m.offset = max(m.offset, 0)
}

// railWidth reserved at the left of every row.
func (m *Model) railWidth() int {
	if !m.rail {
		return 0
	}

	return 1
}

// railEndWidth reserved at the right of every row: the marker and a space
// keeping it off the last column, which is usually a right aligned number
// sitting hard against the edge.
func (m *Model) railEndWidth() int {
	if !m.rail {
		return 0
	}

	return 2
}

// resolve the column widths for the current size.
func (m *Model) resolve() {
	m.widths = resolveWidths(m.columns, m.width-m.railWidth()-m.railEndWidth())
}
