package datatable

import (
	"fmt"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testTable(rows int) *Model {
	m := New(
		WithColumns(
			Column{Title: "name", Flex: 1, MinWidth: 10},
			Column{Title: "value", Width: 8, Align: AlignRight},
		),
		WithEmptyMessage("nothing here"),
	)

	m.SetSize(60, 12)
	m.SetRows(makeRows(rows))

	return m
}

func makeRows(n int) []Row {
	rows := make([]Row, 0, n)

	for i := range n {
		rows = append(rows, Row{Text(fmt.Sprintf("row-%d", i)), Text(fmt.Sprintf("%d", i))})
	}

	return rows
}

func TestModel_CursorStartsAtTheTop(t *testing.T) {
	m := testTable(20)

	assert.Zero(t, m.Cursor())
	assert.Zero(t, m.Offset())
}

func TestModel_MoveDownScrollsOnlyOnceTheCursorLeavesTheWindow(t *testing.T) {
	m := testTable(50)

	visible := m.visibleHeight()

	for range visible - 1 {
		m.MoveDown(1)
	}

	assert.Equal(t, visible-1, m.Cursor())
	assert.Zero(t, m.Offset(), "the window should not move until the cursor leaves it")

	m.MoveDown(1)
	assert.Equal(t, 1, m.Offset())
}

func TestModel_CursorNeverLeavesTheRows(t *testing.T) {
	m := testTable(5)

	m.MoveDown(100)
	assert.Equal(t, 4, m.Cursor())

	m.MoveUp(100)
	assert.Zero(t, m.Cursor())
}

// The window must never scroll past the end into blank space.
func TestModel_OffsetNeverShowsBlankTail(t *testing.T) {
	m := testTable(50)

	m.GotoBottom()

	assert.Equal(t, 50-m.visibleHeight(), m.Offset())
	assert.Len(t, m.VisibleRows(), m.visibleHeight())
}

// A list which is being appended to while it is read must not throw the reader
// off the end when it shrinks.
func TestModel_ShrinkingRowsClampsTheCursor(t *testing.T) {
	m := testTable(50)
	m.GotoBottom()

	m.SetRows(makeRows(3))

	assert.Equal(t, 2, m.Cursor())
	assert.Zero(t, m.Offset())
}

func TestModel_EmptyRows(t *testing.T) {
	m := testTable(0)

	assert.Zero(t, m.Cursor())
	assert.Zero(t, m.Offset())
	assert.Empty(t, m.VisibleRows())

	_, ok := m.SelectedRow()
	assert.False(t, ok)

	// Moving around an empty table is a no-op rather than a panic.
	m.MoveDown(5)
	m.MoveUp(5)
	m.GotoBottom()
	assert.Zero(t, m.Cursor())
}

func TestModel_SelectedRow(t *testing.T) {
	m := testTable(10)
	m.SetCursor(4)

	row, ok := m.SelectedRow()
	require.True(t, ok)
	assert.Equal(t, "row-4", row[0].String())
}

// Resizing to a shorter window has to bring the cursor back into view.
func TestModel_ResizeKeepsTheCursorVisible(t *testing.T) {
	m := testTable(50)
	m.SetCursor(40)

	m.SetSize(60, 6)

	from, to := m.window()
	assert.GreaterOrEqual(t, m.Cursor(), from)
	assert.Less(t, m.Cursor(), to)
}

func TestModel_WindowOfOne(t *testing.T) {
	m := testTable(20)
	m.SetSize(60, headerHeight+1)

	require.Equal(t, 1, m.visibleHeight())

	m.MoveDown(5)
	assert.Equal(t, 5, m.Cursor())
	assert.Equal(t, 5, m.Offset())
	assert.Len(t, m.VisibleRows(), 1)
}

func TestModel_Keys(t *testing.T) {
	m := testTable(50)

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	assert.Equal(t, 1, m.Cursor())

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyUp})
	assert.Zero(t, m.Cursor())

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("G")})
	assert.Equal(t, 49, m.Cursor())

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("g")})
	assert.Zero(t, m.Cursor())
}

func TestModel_BlurredIgnoresKeys(t *testing.T) {
	m := testTable(50)
	m.Blur()

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})

	assert.Zero(t, m.Cursor())
	assert.False(t, m.Focused())
}
