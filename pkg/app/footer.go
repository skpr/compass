package app

import (
	"strings"

	"github.com/charmbracelet/x/ansi"

	"github.com/skpr/compass/pkg/app/component/text"
	"github.com/skpr/compass/pkg/app/theme"
)

// hint is one key and what it does.
type hint struct {
	key  string
	desc string
}

// viewFooter renders the key rail.
//
// It replaces a grey band across the bottom of the screen which held a path and
// a logo. A band of mid grey is the loudest thing on a dark terminal and it was
// spent on the two least useful facts on the page; the keys are what a reader
// actually needs, and the path is kept beside them at a weight which does not
// compete.
func (m *Model) viewFooter() string {
	keys := m.hints()

	rendered := make([]string, 0, len(keys))

	for _, h := range keys {
		rendered = append(rendered, theme.S.Key.Render(h.key)+" "+theme.S.KeyDesc.Render(h.desc))
	}

	left := " " + strings.Join(rendered, theme.S.KeyDesc.Render("   "))

	// The probe path takes whatever the keys leave, cut through the middle: its
	// two ends are what identify it and the mount point in between is filler.
	room := m.Width - ansi.StringWidth(left) - 2
	if room < 12 {
		return m.padLine(left)
	}

	path := theme.S.Faint.Render(text.FitMiddle(m.ProbePath, room))

	gap := m.Width - ansi.StringWidth(left) - ansi.StringWidth(path) - 1
	if gap < 1 {
		gap = 1
	}

	return left + strings.Repeat(" ", gap) + path + " "
}

// hints for wherever the interface currently is.
func (m *Model) hints() []hint {
	if m.showHelp {
		return []hint{{key: "?", desc: "close"}, {key: "q", desc: "quit"}}
	}

	if m.inTrace() {
		return []hint{
			{key: "↑↓", desc: "move"},
			{key: "←→", desc: "page"},
			{key: "esc", desc: "back"},
			{key: "?", desc: "keys"},
			{key: "q", desc: "quit"},
		}
	}

	if m.filterFocused {
		return []hint{
			{key: "enter", desc: "keep"},
			{key: "esc", desc: "clear"},
		}
	}

	hints := []hint{{key: "↑↓", desc: "move"}}

	if m.PageSelected == PageSearch {
		hints = append(hints, hint{key: "enter", desc: "open"})
	}

	hints = append(hints, hint{key: "/", desc: "filter"})

	return append(hints,
		hint{key: "←→", desc: "tabs"},
		hint{key: "?", desc: "keys"},
		hint{key: "q", desc: "quit"},
	)
}
