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

// wrap a cell to a width, flowing it onto as many lines as it needs, up to a
// limit. The last line of a cell which has more to say than the limit allows
// is marked, so a reader can tell "that is all of it" from "there is more".
//
// The flow runs across the segments rather than within each one, so a cell
// made of several styles breaks where the text runs out of room rather than
// where one style ends.
func (c Cell) wrap(width, limit int) [][]Segment {
	if width <= 0 || limit <= 0 {
		return nil
	}

	// The ordinary case, and the one every unwrapped column takes: it fits on
	// the line it is already on.
	if c.width() <= width && !strings.Contains(c.String(), "\n") {
		return [][]Segment{c.Segments}
	}

	lines := wrapSegments(c.Segments, width)

	if len(lines) <= limit {
		return lines
	}

	lines = lines[:limit]
	lines[limit-1] = mark(lines[limit-1], width)

	return lines
}

// wrapSegments flows segments onto lines of a width.
//
// Breaking happens at spaces where there is one to break at and mid word where
// there is not, which is the only option for the values this sees most: a file
// path or a stack frame is one long word.
func wrapSegments(segments []Segment, width int) [][]Segment {
	var (
		lines   [][]Segment
		current []Segment
		used    int
	)

	newline := func() {
		lines = append(lines, trimTrailing(current))
		current = nil
		used = 0
	}

	for _, segment := range segments {
		for part, text := range strings.Split(segment.Text, "\n") {
			// A message which already says where its lines end is honoured:
			// that break is the author's rather than the width's.
			if part > 0 {
				newline()
			}

			for text != "" {
				if used >= width {
					newline()
				}

				if used == 0 {
					// The space a line was broken at does not then start the
					// line it was broken onto.
					text = strings.TrimLeft(text, " ")
					if text == "" {
						break
					}
				}

				head, rest := breakLine(text, width-used, used == 0)
				if head == "" {
					// The next word will not fit in what is left of this line
					// but will fit on one of its own.
					newline()

					continue
				}

				current = append(current, Segment{Text: head, Style: segment.Style})
				used += ansi.StringWidth(head)
				text = rest
			}
		}
	}

	if len(current) > 0 || len(lines) == 0 {
		lines = append(lines, trimTrailing(current))
	}

	return lines
}

// trimTrailing takes the spaces off the end of a line. They are the spaces a
// break landed on, and a line which ends in them is a line which is wider
// than what it says.
func trimTrailing(segments []Segment) []Segment {
	for len(segments) > 0 {
		last := len(segments) - 1

		trimmed := strings.TrimRight(segments[last].Text, " ")
		if trimmed != "" {
			segments[last].Text = trimmed

			break
		}

		segments = segments[:last]
	}

	return segments
}

// breakLine splits text into the part which fits in the room left on a line
// and the part which does not.
//
// An empty head means nothing worth keeping fits in what is left, and the
// caller should start a new line. That only happens part way along one.
func breakLine(text string, room int, atLineStart bool) (string, string) {
	if room <= 0 {
		return "", text
	}

	if ansi.StringWidth(text) <= room {
		return text, ""
	}

	head := ansi.Truncate(text, room, "")
	rest := text[len(head):]

	// The cut landed on a space, which means it landed between two words.
	if strings.HasPrefix(rest, " ") {
		if trimmed := strings.TrimRight(head, " "); trimmed != "" {
			return trimmed, strings.TrimLeft(rest, " ")
		}
	}

	// It landed inside a word, so the break moves back to the last space
	// before it, if this line has one.
	if index := strings.LastIndex(head, " "); index > 0 {
		return strings.TrimRight(head[:index], " "), strings.TrimLeft(text[index:], " ")
	}

	// A word longer than the whole line has to be cut somewhere, but not until
	// it has a line to itself.
	if !atLineStart {
		return "", text
	}

	return head, rest
}

// mark a line as having more after it.
func mark(segments []Segment, width int) []Segment {
	return Cell{Segments: append(segments, Segment{Text: theme.MarkerEllipsis})}.fit(width)
}
