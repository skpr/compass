package datatable

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/skpr/compass/pkg/app/theme"
)

// wrapped is what a wrapped cell says, one string per line.
func wrapped(cell Cell, width, limit int) []string {
	lines := cell.wrap(width, limit)

	out := make([]string, 0, len(lines))
	for _, line := range lines {
		out = append(out, Cell{Segments: line}.String())
	}

	return out
}

// testWrappingTable is a log shaped table: a couple of narrow fixed columns
// and one wide column of prose.
func testWrappingTable(rows []Row) *Model {
	m := New(
		WithSelectionRail(),
		WithColumns(
			Column{Title: "time", Width: 12},
			Column{Title: "level", Width: 5},
			Column{Title: "message", Flex: 1, MinWidth: 20, Wrap: true},
		),
		WithEmptyMessage("nothing here"),
	)

	m.SetSize(60, 14)
	m.SetRows(rows)

	return m
}

// logRows of a message repeated, so the heights are known: at width 60 the
// message column is 37 wide.
func logRows(messages ...string) []Row {
	rows := make([]Row, 0, len(messages))

	for _, message := range messages {
		rows = append(rows, Row{Text("10:00:00.000"), Text("ERROR"), Text(message)})
	}

	return rows
}

func TestWrap_BreaksAtSpaces(t *testing.T) {
	lines := wrapped(Text("the quick brown fox jumps over it"), 10, maxRowLines)

	assert.Equal(t, []string{"the quick", "brown fox", "jumps over", "it"}, lines)

	for _, line := range lines {
		assert.LessOrEqual(t, ansi.StringWidth(line), 10)
	}
}

// A file path or a stack frame is one long word, and it is most of what this
// has to show. It gets cut at the width rather than left to overflow.
func TestWrap_BreaksInsideAWordWhichHasNoSpaces(t *testing.T) {
	lines := wrapped(Text(strings.Repeat("ab", 12)), 8, maxRowLines)

	assert.Equal(t, []string{"abababab", "abababab", "abababab"}, lines)
}

// A word which will not fit in what is left of a line starts the next one
// rather than being broken across the two.
func TestWrap_MovesAWholeWordToTheNextLine(t *testing.T) {
	lines := wrapped(Text("aaa bbbbbbbb ccc"), 8, maxRowLines)

	assert.Equal(t, []string{"aaa", "bbbbbbbb", "ccc"}, lines)
}

func TestWrap_HonoursNewlinesInTheMessage(t *testing.T) {
	lines := wrapped(Text("first\nsecond"), 40, maxRowLines)

	assert.Equal(t, []string{"first", "second"}, lines)
}

// A message longer than the limit is cut, and says that it was: a reader has
// to be able to tell the end of a message from the end of the room for it.
func TestWrap_MarksAMessageItCouldNotFinish(t *testing.T) {
	lines := wrapped(Text(strings.Repeat("word ", 200)), 20, 3)

	require.Len(t, lines, 3)
	assert.True(t, strings.HasSuffix(lines[2], theme.MarkerEllipsis), "not marked: %q", lines[2])
}

func TestWrap_ShortCellStaysOnOneLine(t *testing.T) {
	assert.Equal(t, []string{"short"}, wrapped(Text("short"), 20, maxRowLines))
	assert.Equal(t, []string{""}, wrapped(Text(""), 20, maxRowLines))
}

// The flow runs across the segments, so a break lands where the text runs out
// of room and each piece keeps the style it came with.
func TestWrap_KeepsTheStyleOfEachSegment(t *testing.T) {
	critical := theme.S.Severity(theme.LevelCritical)

	lines := Join(Plain("aaaa bbbb "), Seg("cccc dddd", critical)).wrap(10, maxRowLines)

	require.Len(t, lines, 2)
	assert.Equal(t, "aaaa bbbb", Cell{Segments: lines[0]}.String())
	assert.Equal(t, "cccc dddd", Cell{Segments: lines[1]}.String())

	require.Len(t, lines[1], 1)
	require.NotNil(t, lines[1][0].Style)
	assert.Equal(t, critical.GetForeground(), lines[1][0].Style.GetForeground())
}

// The rule the whole component is held to, with rows which are no longer one
// line tall: every line is exactly the width and there are exactly as many of
// them as the height.
func TestView_WrappingIsStillExactlyItsSize(t *testing.T) {
	rows := logRows(
		"failed to attach probe: uprobe/php_execute_ex: no such file or directory",
		"short",
		strings.Repeat("x", 400),
		"a message with several words in it which needs a second line",
	)

	// Every one of these is wide enough for the columns at their floors, which
	// is the narrowest a table of this shape can be drawn at all.
	for _, size := range []struct{ width, height int }{
		{120, 20}, {80, 10}, {60, 14}, {60, 5}, {80, 4}, {44, 6}, {200, 40},
	} {
		m := testWrappingTable(rows)
		m.SetSize(size.width, size.height)

		for _, cursor := range []int{0, 2, 3} {
			m.SetCursor(cursor)

			view := m.View()

			assert.Equal(t, size.height, lipgloss.Height(view), "height at %dx%d", size.width, size.height)

			for i, line := range strings.Split(view, "\n") {
				assert.Equal(t, size.width, ansi.StringWidth(line), "line %d at %dx%d", i, size.width, size.height)
			}
		}
	}
}

// The point of the whole change: a message too long for the column is all
// there, on as many lines as it takes, rather than cut off at the edge.
func TestView_LongMessageIsShownInFull(t *testing.T) {
	message := "failed to attach probe: uprobe/php_execute_ex: no such file or directory"

	m := testWrappingTable(logRows(message))

	view := ansi.Strip(m.View())

	assert.NotContains(t, view, theme.MarkerEllipsis)

	var got []string

	for _, line := range strings.Split(view, "\n")[2:] {
		line = strings.TrimPrefix(line, theme.SelectionRail)
		line = strings.TrimSuffix(strings.TrimRight(line, " "), theme.SelectionRailEnd)

		got = append(got, line)
	}

	// The row, read back off the screen with the wrapping taken out.
	assert.Equal(t, "10:00:00.000 ERROR "+message, strings.Join(strings.Fields(strings.Join(got, " ")), " "))
}

// The columns beside a wrapped message are blank on its continuation lines:
// the timestamp belongs to the row, and repeating it would read as more rows
// than there are.
func TestView_ContinuationLinesRepeatNothing(t *testing.T) {
	m := testWrappingTable(logRows("a message with quite enough words in it to need a second line"))

	lines := strings.Split(ansi.Strip(m.View()), "\n")

	assert.Contains(t, lines[2], "10:00:00.000")
	assert.NotContains(t, lines[3], "10:00:00.000")
	assert.NotContains(t, lines[3], "ERROR")
	assert.NotEmpty(t, strings.TrimSpace(lines[3]))
}

// The rail runs down the side of the whole selected row rather than marking
// only its first line, so a wrapped row still reads as one row.
func TestView_SelectedWrappedRowIsRailedOnEveryLine(t *testing.T) {
	m := testWrappingTable(logRows(
		"short",
		"a message with quite enough words in it to need a second line",
	))
	m.SetCursor(1)

	lines := strings.Split(ansi.Strip(m.View()), "\n")

	// The header, the rule, then the one line of the first row.
	for _, line := range lines[3:5] {
		assert.True(t, strings.HasPrefix(line, theme.SelectionRail), "no rail at the left of %q", line)
		assert.True(t, strings.HasSuffix(line, theme.SelectionRailEnd), "no rail at the right of %q", line)
	}
}

// No row is allowed to take the whole pane, however much it has to say.
func TestView_OneRowCannotFillThePane(t *testing.T) {
	m := testWrappingTable(logRows(strings.Repeat("word ", 500), "the row after it"))

	assert.Equal(t, maxRowLines, m.rowHeight(0))
	assert.Contains(t, ansi.Strip(m.View()), "the row after it")
}

// The window is measured in lines, so a taller row means fewer rows on screen.
func TestModel_WrappedRowsTakeTheRoomOfSeveralRows(t *testing.T) {
	m := testWrappingTable(logRows(
		"a message with quite enough words in it to need a second line",
		"short",
	))

	assert.Equal(t, 2, m.rowHeight(0))
	assert.Equal(t, 1, m.rowHeight(1))

	m.SetSize(60, headerHeight+2)

	// Two lines of room, and the first row takes both of them.
	from, to := m.window()
	assert.Equal(t, 0, from)
	assert.Equal(t, 1, to)
}

// Moving onto a wrapped row scrolls far enough to show all of it, not just
// its first line.
func TestModel_ScrollsUntilTheWholeCursorRowIsVisible(t *testing.T) {
	rows := logRows("one", "two", "three", strings.Repeat("word ", 40))

	m := testWrappingTable(rows)
	m.SetSize(60, headerHeight+5)

	m.GotoBottom()

	from, to := m.window()
	require.Equal(t, 4, to)

	var lines int
	for i := from; i < to; i++ {
		lines += m.rowHeight(i)
	}

	assert.LessOrEqual(t, lines, m.visibleHeight(), "the window overflows the pane")
	assert.Equal(t, m.rowHeight(3), lines-(3-from), "the cursor row is not on screen in full")
}

// The last row of a list has to be reachable even when it is taller than the
// room left for it, so the window may end part way through a row.
func TestModel_WindowMayEndPartWayThroughARow(t *testing.T) {
	m := testWrappingTable(logRows(strings.Repeat("word ", 100)))
	m.SetSize(60, headerHeight+2)

	from, to := m.window()

	assert.Equal(t, 0, from)
	assert.Equal(t, 1, to)
	assert.Equal(t, headerHeight+2, lipgloss.Height(m.View()))
}

// A table with no wrapping column keeps the arithmetic it had: every row is
// one line, and nothing has to be measured to know it.
func TestModel_UnwrappedTablesStayOneLinePerRow(t *testing.T) {
	m := testTable(50)

	assert.False(t, m.wraps)
	assert.Equal(t, 1, m.rowHeight(0))

	from, to := m.window()
	assert.Equal(t, m.visibleHeight(), to-from)
}
