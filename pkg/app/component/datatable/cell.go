package datatable

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"

	"github.com/skpr/compass/pkg/app/theme"
)

// Segment of a cell which carries its own style.
//
// The text is plain. That is the single most important rule in this package:
// a cell never holds pre-rendered escape sequences, because the component has
// to be able to truncate it to a column width and compose a selected
// background behind it. The widget this replaced took pre-rendered strings and
// truncated them by counting bytes, which cut values inside their escape
// sequences and bled colour across the rest of the screen.
type Segment struct {
	// Text of the segment, without escape sequences.
	Text string
	// Style to render it in. When unset the column's default is used.
	Style *lipgloss.Style
}

// Cell of a row.
type Cell struct {
	// Segments the cell is made of, in order. A cell of one segment is the
	// ordinary case; several is for the few values which are genuinely
	// multi-coloured, like a bar drawn over a track.
	Segments []Segment
}

// Text cell in the column's default style.
func Text(text string) Cell {
	return Cell{Segments: []Segment{{Text: text}}}
}

// Styled cell.
func Styled(text string, style lipgloss.Style) Cell {
	return Cell{Segments: []Segment{{Text: text, Style: &style}}}
}

// Join a cell from segments.
func Join(segments ...Segment) Cell {
	return Cell{Segments: segments}
}

// Seg is a styled segment.
func Seg(text string, style lipgloss.Style) Segment {
	return Segment{Text: text, Style: &style}
}

// Plain is an unstyled segment.
func Plain(text string) Segment {
	return Segment{Text: text}
}

// Row of cells.
type Row []Cell

// String of a cell, with the styling taken off. This is what tests assert on,
// and what filtering matches against.
func (c Cell) String() string {
	var b strings.Builder

	for _, segment := range c.Segments {
		b.WriteString(segment.Text)
	}

	return b.String()
}

// width of a cell in terminal cells.
func (c Cell) width() int {
	var total int

	for _, segment := range c.Segments {
		total += ansi.StringWidth(segment.Text)
	}

	return total
}

// fit a cell to a width, truncating from the right and marking that it was cut.
//
// Truncation walks the segments rather than the concatenated string, so a cell
// which is cut keeps the styling of everything up to the cut.
func (c Cell) fit(width int) []Segment {
	if width <= 0 {
		return nil
	}

	if c.width() <= width {
		return c.Segments
	}

	// One cell of the budget goes to the marker which says there is more.
	budget := width - 1

	out := make([]Segment, 0, len(c.Segments)+1)

	var used int

	for _, segment := range c.Segments {
		segmentWidth := ansi.StringWidth(segment.Text)

		if used+segmentWidth <= budget {
			out = append(out, segment)
			used += segmentWidth

			continue
		}

		if room := budget - used; room > 0 {
			out = append(out, Segment{
				Text:  ansi.Truncate(segment.Text, room, ""),
				Style: segment.Style,
			})
		}

		break
	}

	return append(out, Segment{Text: theme.MarkerEllipsis})
}

// Render a cell on its own, outside a table.
//
// For the strips above a table, which show the same values in the same styles
// but are not rows. Going through the cell keeps one definition of how a value
// looks rather than two which drift apart.
func (c Cell) Render() string {
	var b strings.Builder

	for _, segment := range c.Segments {
		if segment.Style == nil {
			b.WriteString(segment.Text)

			continue
		}

		b.WriteString(segment.Style.Render(segment.Text))
	}

	return b.String()
}
