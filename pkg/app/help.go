package app

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"

	"github.com/skpr/compass/pkg/app/theme"
)

// viewHelp renders the key map and the legend over the page.
//
// The gutter, the flags and the colour ramp are all terse by design, and terse
// only works when there is somewhere to look them up. This is that place, which
// is what licenses the rest of the interface to be as compact as it is.
func (m *Model) viewHelp() string {
	sections := []struct {
		title string
		rows  [][2]string
	}{
		{
			title: "Keys",
			rows: [][2]string{
				{"↑ ↓ / j k", "move"},
				{"g / G", "top / bottom"},
				{"pgup / pgdn", "page"},
				{"← →", "switch tab"},
				{"/", "filter the current page"},
				{"enter", "open the selected trace"},
				{"esc", "close the trace"},
				{"?", "this help"},
				{"q / ctrl+c", "quit"},
			},
		},
		{
			title: "Reading a row",
			rows: [][2]string{
				{theme.SelectionRail, "the row the cursor is on"},
				{AttentionMarker, "uncacheable: something set a max age of zero"},
				{TruncatedMarker, "after calls: additional function calls were dropped"},
				{PartialTimingMarker, "derived timing uses retained function calls only"},
			},
		},
		{
			title: "Self",
			rows: [][2]string{
				{"", "the share of a request a function spent on its own work,"},
				{"", "not waiting on what it called. Reads high: calls under the"},
				{"", "extension's threshold are not recorded."},
			},
		},
	}

	var lines []string

	for i, section := range sections {
		if i > 0 {
			lines = append(lines, "")
		}

		lines = append(lines, theme.S.Header.Render(strings.ToUpper(section.title)))

		for _, row := range section.rows {
			lines = append(lines,
				theme.S.Key.Render(pad(row[0], 12))+theme.S.Dim.Render(row[1]),
			)
		}
	}

	height := m.overlayHeight()

	// Cut to fit rather than overflow: Place centres what it is given but will
	// not shrink it, so a panel taller than the room available pushes the
	// footer off the bottom of the screen.
	//
	// The cut is marked. A legend which quietly stops halfway is worse than a
	// short one, because the reader has no way to tell the difference between
	// "that is all of it" and "your terminal is too short for the rest".
	if room := height - 2; room > 0 && len(lines) > room {
		lines = lines[:room]

		if room > 0 {
			lines[room-1] = theme.S.Faint.Render(theme.MarkerEllipsis + " more, on a taller terminal")
		}
	}

	panel := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(theme.S.Theme().BorderStrong).
		Background(theme.S.Theme().SurfaceRaised).
		Padding(0, 2).
		Render(strings.Join(lines, "\n"))

	if lipgloss.Height(panel) > height || lipgloss.Width(panel) > m.Width {
		panel = strings.Join(strings.Split(panel, "\n")[:min(height, lipgloss.Height(panel))], "\n")
	}

	return lipgloss.Place(m.Width, height, lipgloss.Center, lipgloss.Center, panel)
}

// pad a string out to a width in cells.
//
// Cells, not bytes. Every key hint in this overlay contains an arrow, and an
// arrow is three bytes and one column, so padding by length left the key column
// ragged and ran "↑ ↓ / j k" straight into its description.
func pad(s string, width int) string {
	if fill := width - ansi.StringWidth(s); fill > 0 {
		return s + strings.Repeat(" ", fill)
	}

	return s
}
