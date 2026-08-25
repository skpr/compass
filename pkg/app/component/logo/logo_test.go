package logo

import (
	"os"
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/mattn/go-runewidth"
	"github.com/muesli/termenv"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/skpr/compass/pkg/app/theme"
)

func TestMain(m *testing.M) {
	lipgloss.SetColorProfile(termenv.TrueColor)
	lipgloss.SetHasDarkBackground(true)

	os.Exit(m.Run())
}

func options(width int) Options {
	t := theme.Default()

	return Options{
		Width: width,
		Word:  "COMPASS",
		From:  t.Chrome,
		To:    t.RuntimeNode,
		Field: t.LogoField,
	}
}

// The glyphs the mark is drawn with are all one cell, like everything else the
// interface draws with. A two cell glyph here would push the whole masthead
// past the edge of the screen.
func TestGlyphsAreOneCell(t *testing.T) {
	for _, glyph := range []string{Diagonal, glyphBoth, glyphTop, glyphBottom} {
		assert.Equal(t, 1, runewidth.StringWidth(glyph), "%q", glyph)
	}
}

func TestWord(t *testing.T) {
	rows := Word("COMPASS")

	require.Len(t, rows, Rows)

	for i, row := range rows {
		assert.Equal(t, Width("COMPASS"), runewidth.StringWidth(row), "row %d", i)
	}

	// The shapes, so a change to a letterform is a change to this test rather
	// than something noticed months later in a screenshot.
	assert.Equal(t, "▄▀▀▀▄ ▄▀▀▀▄ █▄ ▄█ █▀▀▀▄ ▄▀▀▀▄ ▄▀▀▀▀ ▄▀▀▀▀", rows[0])
	assert.Equal(t, "█     █   █ █ ▀ █ █▄▄▄▀ █▄▄▄█  ▀▀▀▄  ▀▀▀▄", rows[1])
	assert.Equal(t, "▀▄▄▄▀ ▀▄▄▄▀ █   █ █     █   █ ▀▄▄▄▀ ▀▄▄▄▀", rows[2])
}

// Every letter of the wordmark has a form. A missing one renders as a blank of
// the right size rather than closing the gap, so this would otherwise show up
// as a wordmark with a hole in it.
func TestWord_EveryLetterOfTheWordmarkIsDrawn(t *testing.T) {
	for _, r := range "COMPASS" {
		_, ok := letterforms[r]
		assert.True(t, ok, "no letterform for %q", string(r))
	}
}

func TestWord_UnknownLetterKeepsItsSpace(t *testing.T) {
	rows := Word("CZ")

	for _, row := range rows {
		assert.Equal(t, Width("CZ"), runewidth.StringWidth(row))
	}
}

// Every row is exactly the width it was asked for, whatever it holds.
func TestRender_IsExactlyItsWidth(t *testing.T) {
	for _, width := range []int{200, 120, 80, 60, 55, 40, 20, 8, 1} {
		rows := Render(options(width))

		require.Len(t, rows, Rows, "width=%d", width)

		for i, row := range rows {
			assert.Equal(t, width, ansi.StringWidth(row), "row %d at width %d", i, width)
		}
	}
}

func TestRender_ZeroWidth(t *testing.T) {
	assert.Nil(t, Render(options(0)))
}

// The mark stands in hatching, and the hatching steps in on the right so the
// block reads as a shape rather than as a rectangle.
func TestRender_FieldStepsIn(t *testing.T) {
	rows := Render(options(120))

	var widths []int

	for _, row := range rows {
		widths = append(widths, strings.Count(ansi.Strip(row), Diagonal))
	}

	assert.Greater(t, widths[0], widths[1])
	assert.Greater(t, widths[1], widths[2])
}

// A screen too narrow to stand the letterforms in gets the word in text. Half a
// wordmark is worse than no wordmark.
func TestRender_FallsBackBelowTheWidthItNeeds(t *testing.T) {
	rows := Render(options(40))

	plain := ansi.Strip(strings.Join(rows, "\n"))

	assert.Contains(t, plain, "C O M P A S S")
	assert.NotContains(t, plain, Word("COMPASS")[0])
}

func TestRender_FallbackShrinksToTheWord(t *testing.T) {
	rows := Render(options(9))

	assert.Contains(t, ansi.Strip(strings.Join(rows, "\n")), "COMPASS")
}

// The gradient has to actually run: the first letter and the last are different
// colours, or it is a flat fill with extra steps.
func TestRender_GradientRunsAcrossTheMark(t *testing.T) {
	row := Render(options(120))[0]

	colours := foregrounds(row)
	require.Greater(t, len(colours), 4)

	// The first is the hatching, so the mark's own run starts after it.
	assert.NotEqual(t, colours[1], colours[len(colours)-1],
		"the gradient does not run across the mark")

	// And it runs, rather than stepping between two: most cells differ.
	distinct := make(map[string]struct{}, len(colours))
	for _, colour := range colours {
		distinct[colour] = struct{}{}
	}

	assert.Greater(t, len(distinct), len(colours)/2)
}

// Blanks inside the letterforms are left unstyled. Styling them would double
// the escape sequences in a block redrawn on every frame, for no ink.
func TestRender_DoesNotStyleBlanks(t *testing.T) {
	row := Render(options(120))[1]

	// A styled blank looks like this: a colour, one space, a reset.
	assert.NotContains(t, row, "m \x1b[0m", "a blank was given its own colour")
}

// foregrounds returns each colour-setting escape in a rendered row, in order.
// Resets are left out: they say where a colour stopped, not what it was.
func foregrounds(row string) []string {
	var found []string

	for i := 0; i < len(row); i++ {
		if row[i] != 0x1b {
			continue
		}

		end := i
		for end < len(row) && row[end] != 'm' {
			end++
		}

		if end < len(row) {
			if sequence := row[i : end+1]; strings.Contains(sequence, "38;2;") {
				found = append(found, sequence)
			}
		}

		i = end
	}

	return found
}
