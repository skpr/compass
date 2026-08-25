package datatable

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"

	"github.com/skpr/compass/pkg/app/theme"
)

// View renders the table at exactly its height.
//
// Always exactly: a component which sometimes renders a line short pushes
// everything below it up the screen as the data changes, and one which renders
// a line long scrolls the terminal.
func (m *Model) View() string {
	if m.width < MinWidth || m.height < headerHeight+1 {
		return m.blank(m.height)
	}

	lines := make([]string, 0, m.height)
	lines = append(lines, m.header(), m.rule())

	if len(m.rows) == 0 {
		lines = append(lines, m.emptyLine())
	} else {
		from, to := m.window()

		for i := from; i < to; i++ {
			lines = append(lines, m.renderRow(m.rows[i], m.focused && i == m.cursor))
		}
	}

	// Padded rather than left short, so the region keeps its height whether it
	// holds three rows or three hundred.
	for len(lines) < m.height {
		lines = append(lines, m.pad(""))
	}

	return strings.Join(lines[:m.height], "\n")
}

// header of column titles.
func (m *Model) header() string {
	var b strings.Builder

	b.WriteString(strings.Repeat(" ", m.railWidth()))

	m.eachColumn(func(index, width int, first bool) {
		if !first {
			b.WriteString(strings.Repeat(" ", gap))
		}

		b.WriteString(theme.S.Header.Render(align(
			ansi.Truncate(strings.ToUpper(m.columns[index].Title), width, ""),
			width,
			m.columns[index].Align,
		)))
	})

	return m.pad(b.String())
}

// rule beneath the header.
func (m *Model) rule() string {
	return theme.S.RuleIdle.Render(strings.Repeat(theme.RuleLight, m.width))
}

// renderRow of cells.
//
// The selected background is composed onto each segment's own style rather
// than wrapped around the finished line. Wrapping is what the previous widget
// did, and it meant the first coloured cell in a row ended the highlight: the
// cell's reset closed the background along with its own colour.
func (m *Model) renderRow(row Row, selected bool) string {
	var b strings.Builder

	background := theme.S.Theme().SurfaceSelected

	if m.rail {
		if selected {
			b.WriteString(theme.S.SelectRail.Background(background).Render(theme.SelectionRail))
		} else {
			b.WriteString(" ")
		}
	}

	m.eachColumn(func(index, width int, first bool) {
		if !first {
			b.WriteString(m.spacer(selected, gap))
		}

		var cell Cell
		if index < len(row) {
			cell = row[index]
		}

		segments := cell.fit(width)

		var used int
		for _, segment := range segments {
			used += ansi.StringWidth(segment.Text)
		}

		padding := max(width-used, 0)

		// A column of right aligned numerals reads as a chart; the same column
		// left aligned reads as a mess.
		if m.alignmentOf(index) == AlignRight {
			b.WriteString(m.spacer(selected, padding))
		}

		for _, segment := range segments {
			style := m.styleFor(index, segment)

			// One row per frame carries this, not every cell of every row, so
			// composing the background here is not on the hot path.
			if selected {
				style = style.Background(background)
			}

			b.WriteString(style.Render(segment.Text))
		}

		if m.alignmentOf(index) != AlignRight {
			b.WriteString(m.spacer(selected, padding))
		}
	})

	line := m.padSelected(b.String(), m.width-m.railEndWidth(), selected)

	if m.rail {
		if selected {
			line += m.spacer(selected, 1) + theme.S.SelectRail.Background(background).Render(theme.SelectionRailEnd)
		} else {
			line += "  "
		}
	}

	return line
}

// alignmentOf a column.
func (m *Model) alignmentOf(column int) Align {
	if column < len(m.columns) {
		return m.columns[column].Align
	}

	return AlignLeft
}

// styleFor a segment, falling back to the column's default and then to plain.
func (m *Model) styleFor(column int, segment Segment) lipgloss.Style {
	if segment.Style != nil {
		return *segment.Style
	}

	if column < len(m.columns) && m.columns[column].Style != nil {
		return *m.columns[column].Style
	}

	return theme.S.Cell
}

// eachColumn calls fn for every column which survived the width resolution.
func (m *Model) eachColumn(fn func(index, width int, first bool)) {
	first := true

	for index, width := range m.widths {
		if width <= 0 {
			continue
		}

		fn(index, width, first)

		first = false
	}
}

// spacer of blanks, carrying the selected background across the gaps so the
// highlight is a continuous band rather than a row of stripes.
func (m *Model) spacer(selected bool, width int) string {
	if width <= 0 {
		return ""
	}

	blanks := strings.Repeat(" ", width)

	if !selected {
		return blanks
	}

	return theme.S.Selected.Render(blanks)
}

// padSelected fills a rendered row out to a width, carrying the selected
// background across the padding.
func (m *Model) padSelected(line string, width int, selected bool) string {
	if padding := width - ansi.StringWidth(line); padding > 0 {
		return line + m.spacer(selected, padding)
	}

	return line
}

// pad a line out to the full width.
func (m *Model) pad(line string) string {
	if padding := m.width - ansi.StringWidth(line); padding > 0 {
		return line + strings.Repeat(" ", padding)
	}

	return line
}

// emptyLine explaining that there is nothing to show.
func (m *Model) emptyLine() string {
	if m.empty == "" {
		return m.pad("")
	}

	return m.pad(theme.S.Empty.Render(ansi.Truncate(m.empty, m.width, theme.MarkerEllipsis)))
}

// blank region of a height, for when there is not enough room to draw.
func (m *Model) blank(height int) string {
	if height <= 0 {
		return ""
	}

	return strings.TrimSuffix(strings.Repeat(m.pad("")+"\n", height), "\n")
}

// align text within a width.
func align(text string, width int, alignment Align) string {
	padding := width - ansi.StringWidth(text)
	if padding <= 0 {
		return text
	}

	if alignment == AlignRight {
		return strings.Repeat(" ", padding) + text
	}

	return text + strings.Repeat(" ", padding)
}
