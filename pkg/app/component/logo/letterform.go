// Package logo renders the Compass wordmark as block letterforms.
//
// Each letter is a six by five bitmap, drawn two pixel rows to a character row
// with the half block glyphs: a cell with both rows set is █, the top row ▀,
// the bottom row ▄. Six rows of pixels therefore become three rows of text,
// which is as tall as a wordmark can be before it costs more of the screen than
// it is worth.
package logo

import "strings"

// Rows a rendered letterform occupies.
const Rows = 3

// Columns a letterform occupies, not counting the space after it.
const Columns = 5

// letterform as a six row bitmap. A set pixel is anything but a space.
type letterform [6]string

var letterforms = map[rune]letterform{
	'C': {
		".XXX.",
		"X...X",
		"X....",
		"X....",
		"X...X",
		".XXX.",
	},
	'O': {
		".XXX.",
		"X...X",
		"X...X",
		"X...X",
		"X...X",
		".XXX.",
	},
	'M': {
		"X...X",
		"XX.XX",
		"X.X.X",
		"X...X",
		"X...X",
		"X...X",
	},
	'P': {
		"XXXX.",
		"X...X",
		"X...X",
		"XXXX.",
		"X....",
		"X....",
	},
	'A': {
		".XXX.",
		"X...X",
		"X...X",
		"XXXXX",
		"X...X",
		"X...X",
	},
	'S': {
		".XXXX",
		"X....",
		".XXX.",
		"....X",
		"X...X",
		".XXX.",
	},
}

// Half block glyphs, by which of the two pixel rows in a cell are set.
const (
	glyphBoth   = "█"
	glyphTop    = "▀"
	glyphBottom = "▄"
	glyphNone   = " "
)

// render a letterform into its three rows of text.
func (l letterform) render() []string {
	rows := make([]string, Rows)

	for row := range Rows {
		var b strings.Builder

		top, bottom := l[row*2], l[row*2+1]

		for column := range Columns {
			b.WriteString(cell(set(top, column), set(bottom, column)))
		}

		rows[row] = b.String()
	}

	return rows
}

// set reports whether a pixel in a bitmap row is on.
func set(row string, column int) bool {
	return column < len(row) && row[column] != '.'
}

// cell for a pair of vertically stacked pixels.
func cell(top, bottom bool) string {
	switch {
	case top && bottom:
		return glyphBoth
	case top:
		return glyphTop
	case bottom:
		return glyphBottom
	default:
		return glyphNone
	}
}

// Word renders text as block letterforms, one space between each.
//
// A letter with no form is rendered as a blank of the same size rather than
// skipped, so the wordmark cannot silently lose a letter.
func Word(text string) []string {
	rows := make([]string, Rows)

	for i, r := range strings.ToUpper(text) {
		form, ok := letterforms[r]

		rendered := make([]string, Rows)
		if ok {
			rendered = form.render()
		} else {
			for row := range Rows {
				rendered[row] = strings.Repeat(" ", Columns)
			}
		}

		for row := range Rows {
			if i > 0 {
				rows[row] += " "
			}

			rows[row] += rendered[row]
		}
	}

	return rows
}

// Width of a rendered word.
func Width(text string) int {
	letters := len([]rune(text))
	if letters == 0 {
		return 0
	}

	return letters*Columns + letters - 1
}
