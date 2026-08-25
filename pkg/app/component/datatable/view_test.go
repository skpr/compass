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

// Every line is exactly the width the table was given, and there are exactly
// as many of them as its height. A component which breaks either pushes the
// rest of the screen around as its data changes.
func TestView_IsExactlyItsSize(t *testing.T) {
	sizes := []struct{ width, height int }{
		{120, 20}, {80, 10}, {60, 5}, {40, 4}, {24, 3}, {200, 40},
	}

	for _, rows := range []int{0, 1, 3, 100} {
		for _, size := range sizes {
			m := testTable(rows)
			m.SetSize(size.width, size.height)

			view := m.View()

			assert.Equal(t, size.height, lipgloss.Height(view),
				"height rows=%d size=%dx%d", rows, size.width, size.height)

			for i, line := range strings.Split(view, "\n") {
				assert.Equal(t, size.width, ansi.StringWidth(line),
					"line %d, rows=%d size=%dx%d", i, rows, size.width, size.height)
			}
		}
	}
}

// A long value is cut to fit rather than pushing its row past the edge, and it
// is cut by what it prints rather than by how many bytes it takes to say it.
func TestView_TruncatesWideCells(t *testing.T) {
	m := New(WithColumns(Column{Title: "name", Flex: 1, MinWidth: 10}))
	m.SetSize(30, 5)
	m.SetRows([]Row{{Styled(strings.Repeat("x", 200), theme.S.Severity(theme.LevelCritical))}})

	view := m.View()

	for _, line := range strings.Split(view, "\n") {
		assert.Equal(t, 30, ansi.StringWidth(line))
	}

	assert.Contains(t, ansi.Strip(view), theme.MarkerEllipsis)
}

// The bug this component exists to fix. The widget it replaces applied the
// selected background by wrapping the whole row, so the first coloured cell in
// it closed the highlight with its own reset and every cell after rendered
// unhighlighted. Composing the background onto each segment keeps it unbroken.
func TestView_SelectedRowKeepsItsBackgroundPastAColouredCell(t *testing.T) {
	m := New(WithColumns(
		Column{Title: "caller", Flex: 1, MinWidth: 10},
		Column{Title: "max age", Width: 10},
		Column{Title: "calls", Width: 6},
	))
	m.SetSize(60, 5)
	m.SetRows([]Row{{
		Text("Blocker::render"),
		Styled("0s", theme.S.Severity(theme.LevelCritical)),
		Text("412"),
	}})

	row := strings.Split(m.View(), "\n")[2]

	background := backgroundOf(t, theme.S.Selected.Render("x"))

	// The last thing on the line which sets a background must still be the
	// selected one: if a cell's reset had ended it, the tail would be bare.
	assert.True(t, strings.HasSuffix(stripTrailingReset(row), background+strings.Repeat(" ", trailingBlanks(row))+"\x1b[0m") ||
		strings.Contains(row, background),
		"selected background missing from %q", row)

	assert.Greater(t, strings.Count(row, background), 1,
		"the background should be reopened after each coloured cell, in %q", row)
}

func TestView_UnselectedRowHasNoBackground(t *testing.T) {
	m := testTable(3)
	m.Blur()

	for _, line := range strings.Split(m.View(), "\n")[2:] {
		assert.NotContains(t, line, backgroundOf(t, theme.S.Selected.Render("x")))
	}
}

func TestView_RightAlignment(t *testing.T) {
	m := New(WithColumns(
		Column{Title: "name", Flex: 1, MinWidth: 10},
		Column{Title: "n", Width: 8, Align: AlignRight},
	))
	m.SetSize(40, 4)
	m.SetRows([]Row{{Text("a"), Text("42")}})

	row := ansi.Strip(strings.Split(m.View(), "\n")[2])

	assert.True(t, strings.HasSuffix(row, "      42"), "not right aligned: %q", row)
}

func TestView_EmptyMessage(t *testing.T) {
	m := testTable(0)

	assert.Contains(t, ansi.Strip(m.View()), "nothing here")
}

func TestView_HeaderIsUpperCased(t *testing.T) {
	m := testTable(1)

	assert.Contains(t, ansi.Strip(m.View()), "NAME")
}

// Below its minimum the table renders blank rather than emitting garbage or a
// negative width.
func TestView_TooNarrow(t *testing.T) {
	m := testTable(10)
	m.SetSize(MinWidth-1, 8)

	view := m.View()

	assert.Equal(t, 8, lipgloss.Height(view))
	assert.Empty(t, strings.TrimSpace(ansi.Strip(view)))
}

// backgroundOf extracts the escape sequence which sets a background.
func backgroundOf(t *testing.T, rendered string) string {
	t.Helper()

	start := strings.Index(rendered, "\x1b[48;2;")
	require.GreaterOrEqual(t, start, 0, "no background in %q", rendered)

	end := strings.Index(rendered[start:], "m")
	require.Greater(t, end, 0)

	return rendered[start : start+end+1]
}

func stripTrailingReset(line string) string {
	return strings.TrimSuffix(line, "\x1b[0m")
}

func trailingBlanks(line string) int {
	stripped := ansi.Strip(line)

	return len(stripped) - len(strings.TrimRight(stripped, " "))
}
