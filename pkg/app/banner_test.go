package app

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/skpr/compass/pkg/app/component/logo"
	"github.com/skpr/compass/pkg/app/layout"
	"github.com/skpr/compass/pkg/app/theme"
)

// The wordmark is drawn rather than written, so there is no "COMPASS" in the
// output to look for — the letterforms are what is there.
func TestViewBanner_IsTheWordmark(t *testing.T) {
	m := testModel(120, 30)

	plain := ansi.Strip(m.viewBanner())

	for _, row := range logo.Word(Wordmark) {
		assert.Contains(t, plain, row)
	}
}

// The band runs the whole way across and stands its full height, or it is a
// chip rather than a masthead.
func TestViewBanner_FillsItsRegion(t *testing.T) {
	for _, width := range []int{200, 120, 80, 60, 20, 8, 1} {
		m := testModel(120, 30)
		m.Width = width

		banner := m.viewBanner()

		assert.Equal(t, layout.BannerHeight, lipgloss.Height(banner), "width=%d", width)

		for i, line := range strings.Split(banner, "\n") {
			assert.Equal(t, width, ansi.StringWidth(line), "line %d at width %d", i, width)
		}
	}
}

// The mark stands in a field of hatching, which is what fills the band without
// filling it in.
func TestViewBanner_HasAField(t *testing.T) {
	m := testModel(120, 30)

	assert.Contains(t, ansi.Strip(m.viewBanner()), logo.Diagonal)
}

// A line of air under the mark, so the tabs read as a separate thing rather
// than as the bottom of it.
func TestViewBanner_EndsInAir(t *testing.T) {
	m := testModel(120, 30)

	lines := strings.Split(ansi.Strip(m.viewBanner()), "\n")

	require.Len(t, lines, layout.BannerHeight)
	assert.Empty(t, strings.TrimSpace(lines[len(lines)-1]))
}

// Too narrow to stand the letterforms in, and the mark falls back to text
// rather than being cut in half.
func TestViewBanner_FallsBackOnANarrowScreen(t *testing.T) {
	m := testModel(120, 30)
	m.Width = 30

	plain := ansi.Strip(m.viewBanner())

	assert.Contains(t, plain, "C O M P A S S")
	assert.NotContains(t, plain, logo.Word(Wordmark)[0])
}

// Every colour the interface uses for state is invisible on the brand blue, so
// the band carries nothing which depends on one.
func TestViewBanner_CarriesNothingColoured(t *testing.T) {
	m := testModel(120, 30)
	m.updateKeyEnter()

	plain := ansi.Strip(m.viewBanner())

	assert.NotContains(t, plain, "connected")
	assert.NotContains(t, plain, "GET")
}

// Which is why the connection state moved to the row below, where the ground is
// dark and its colour still means something.
//
// Which trace is open is not there: the block beneath says so, with the name of
// the field beside it, and saying it twice in two shapes is worse than once.
func TestViewMenu_CarriesTheState(t *testing.T) {
	m := testModel(120, 30)

	assert.Contains(t, ansi.Strip(m.viewMenu()), "connected")

	m.updateKeyEnter()

	menu := ansi.Strip(m.viewMenu())
	assert.Contains(t, menu, "connected")
	assert.NotContains(t, menu, "GET /")
}

// A narrow terminal keeps the state: whether anything is arriving is worth a
// corner of the screen at any size.
func TestViewMenu_NarrowKeepsTheState(t *testing.T) {
	m := testModel(120, 30)
	m.updateKeyEnter()
	m.Width = 62
	m.relayout()

	assert.Contains(t, ansi.Strip(m.viewMenu()), "connected")

	for _, line := range strings.Split(m.viewMenu(), "\n") {
		assert.LessOrEqual(t, ansi.StringWidth(line), 62)
	}
}

// The tabs are filled chips, so which one is active is a shape rather than a
// weight — legible before anything is read.
func TestViewMenu_ActiveTabIsFilledWithTheChrome(t *testing.T) {
	m := testModel(120, 30)

	row := strings.Split(m.viewMenu(), "\n")[0]

	require.NotEmpty(t, theme.S.Theme().Chrome.TrueColor)

	assert.Contains(t, row, backgroundSequence(t, theme.S.TabActive.Render("x")),
		"the active tab is not filled with the chrome")
	assert.Contains(t, row, backgroundSequence(t, theme.S.TabIdle.Render("x")),
		"the inactive tab is not filled with a surface")
}

// The chips carry their own padding, so the first tab's text sits one cell in,
// matching everything else on the screen.
func TestViewMenu_ActiveTabIsInset(t *testing.T) {
	m := testModel(120, 30)

	tabs := ansi.Strip(strings.Split(m.viewMenu(), "\n")[0])

	assert.Equal(t, 1, strings.Index(tabs, strings.ToUpper(string(PageSearch))))
}

// backgroundSequence extracts the escape which sets a style's background.
func backgroundSequence(t *testing.T, rendered string) string {
	t.Helper()

	start := strings.Index(rendered, "\x1b[")
	require.GreaterOrEqual(t, start, 0)

	end := strings.Index(rendered[start:], "m")
	require.Greater(t, end, 0)

	sequence := rendered[start : start+end+1]
	require.Contains(t, sequence, "48;2;", "no background in %q", rendered)

	return sequence
}

// A trace with no Drupal data has no Drupal page. Offering a tab which can only
// say "there is nothing here" is worse than not offering it.
func TestViewMenu_DrupalTabOnlyWhenThereIsDrupalData(t *testing.T) {
	m := testModel(120, 30)
	m.updateKeyEnter()

	require.True(t, m.hasDrupal())
	assert.Contains(t, ansi.Strip(m.viewMenu()), strings.ToUpper(string(PageDrupal)))

	// A Node trace, a PHP CLI run, a PHP application which is not Drupal: none
	// of them collect any of it.
	m.Current.Drupal = nil

	menu := ansi.Strip(m.viewMenu())
	assert.NotContains(t, menu, strings.ToUpper(string(PageDrupal)))
	assert.Contains(t, menu, strings.ToUpper(string(PageFunctions)))
}
