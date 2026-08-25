package logo

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"

	"github.com/skpr/compass/pkg/app/theme"
)

// Diagonal is the hatching the wordmark stands in.
const Diagonal = "╱"

// Flank widths, in cells.
const (
	// leftFlank is fixed, so the wordmark sits in the same column whatever the
	// terminal is doing.
	leftFlank = 6
	// flankGap separates the hatching from the letterforms.
	flankGap = 2
	// minRightFlank keeps some hatching on the right even when the screen is
	// narrow enough that the wordmark has eaten most of it.
	minRightFlank = 4
)

// Options for rendering the wordmark.
type Options struct {
	// Width of the region to fill.
	Width int
	// Word to render.
	Word string
	// From and To are the ends of the gradient run across the letterforms.
	From lipgloss.CompleteColor
	To   lipgloss.CompleteColor
	// Field is the colour of the hatching.
	Field lipgloss.CompleteColor
}

// Render the wordmark, as Rows lines of exactly Width cells.
//
// A gradient runs across the letterforms and diagonal hatching fills the space
// either side, stepping in on the right so the block reads as a shape rather
// than a rectangle. Below the width the letterforms need, the hatching goes
// first and then the mark falls back to plain text: a wordmark cut in half is
// worse than no wordmark.
func Render(o Options) []string {
	if o.Width <= 0 {
		return nil
	}

	word := Word(o.Word)
	width := Width(o.Word)

	if width+leftFlank+flankGap*2+minRightFlank > o.Width {
		return renderPlain(o)
	}

	colours := theme.Gradient(o.From, o.To, width)
	field := lipgloss.NewStyle().Foreground(o.Field)

	rows := make([]string, Rows)

	for row := range Rows {
		var b strings.Builder

		b.WriteString(field.Render(strings.Repeat(Diagonal, leftFlank)))
		b.WriteString(strings.Repeat(" ", flankGap))
		b.WriteString(gradientRow(word[row], colours))
		b.WriteString(strings.Repeat(" ", flankGap))

		// Stepped in by one cell per row, which is what stops the flank
		// reading as a wall beside the mark.
		right := o.Width - leftFlank - flankGap*2 - width - row
		if right > 0 {
			b.WriteString(field.Render(strings.Repeat(Diagonal, right)))
		}

		rows[row] = pad(b.String(), o.Width)
	}

	return rows
}

// gradientRow renders one row of letterforms, each cell taking its colour from
// where it falls across the word.
//
// Blanks are left unstyled: they carry no ink, and styling them would double
// the escape sequences in a block which is redrawn on every frame.
func gradientRow(row string, colours []lipgloss.CompleteColor) string {
	var b strings.Builder

	for i, r := range []rune(row) {
		if r == ' ' {
			b.WriteRune(r)

			continue
		}

		colour := colours[min(i, len(colours)-1)]

		b.WriteString(lipgloss.NewStyle().Foreground(colour).Render(string(r)))
	}

	return b.String()
}

// renderPlain falls back to the word in text, centred, for a screen too narrow
// to stand the letterforms in.
func renderPlain(o Options) []string {
	rows := make([]string, Rows)

	spaced := strings.Join(strings.Split(strings.ToUpper(o.Word), ""), " ")
	if ansi.StringWidth(spaced) > o.Width {
		spaced = ansi.Truncate(strings.ToUpper(o.Word), o.Width, "")
	}

	colours := theme.Gradient(o.From, o.To, ansi.StringWidth(spaced))

	left := (o.Width - ansi.StringWidth(spaced)) / 2

	rows[Rows/2] = pad(strings.Repeat(" ", left)+gradientRow(spaced, colours), o.Width)

	for row := range Rows {
		if rows[row] == "" {
			rows[row] = strings.Repeat(" ", o.Width)
		}
	}

	return rows
}

// pad a line out to a width.
func pad(line string, width int) string {
	if fill := width - ansi.StringWidth(line); fill > 0 {
		return line + strings.Repeat(" ", fill)
	}

	return line
}
