package app

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/x/ansi"

	"github.com/skpr/compass/pkg/app/component/text"
	"github.com/skpr/compass/pkg/app/theme"
)

// gapBetweenTabs, in cells. The tabs carry their own padding, so this is the
// space between two chips rather than between two words.
const gapBetweenTabs = 1

// viewMenu renders the tabs and the rule beneath them.
//
// Two lines rather than the three a bordered tab row costs, and the rule does
// two jobs at once: it marks which tab is active by thickening under it, and it
// closes the header. A box around two words was the worst ratio of ink to
// meaning on the screen.
func (m *Model) viewMenu() string {
	// No context beside the trace tabs: which trace is open is the first thing
	// the block below them says, with the name of the field beside it, and
	// saying it twice in two different shapes is worse than saying it once.
	if m.inTrace() {
		tabs := []tab{{label: string(PageFunctions), active: m.PageSelected == PageFunctions}}

		if m.hasDrupal() {
			tabs = append(tabs, tab{label: string(PageDrupal), active: m.PageSelected == PageDrupal})
		}

		return m.renderTabs(tabs, "")
	}

	return m.renderTabs([]tab{
		{label: string(PageSearch), count: len(m.traces), showCount: true, active: m.PageSelected == PageSearch},
		{label: string(PageLogs), count: len(m.logs), showCount: true, active: m.PageSelected == PageLogs},
	}, "")
}

// tab on the menu.
type tab struct {
	label     string
	count     int
	showCount bool
	active    bool
}

// text of a tab, as it appears on screen.
func (t tab) text() string {
	label := strings.ToUpper(t.label)

	if !t.showCount {
		return label
	}

	return fmt.Sprintf("%s %d", label, t.count)
}

// renderTabs and the rule which underlines the active one.
//
// The row carries three things: the tabs, whatever the page wants to say about
// what it is showing, and the state of the connection. The last two live here
// rather than on the band above because both carry a colour, and a colour on
// the brand blue says nothing.
func (m *Model) renderTabs(tabs []tab, context string) string {
	var labels strings.Builder

	// No leading space: the chips carry their own padding, so starting at the
	// edge puts the first tab's text in the same column as the wordmark above
	// it, and lets the active tab sit flush under the band it shares a colour
	// with.
	for i, t := range tabs {
		if i > 0 {
			labels.WriteString(strings.Repeat(" ", gapBetweenTabs))
		}

		style := theme.S.TabIdle
		if t.active {
			style = theme.S.TabActive
		}

		labels.WriteString(style.Render(t.text()))
	}

	// A single rule the whole way across. The tabs carry their own background
	// now, so a heavy run under the active one would be saying a second time
	// what its colour already says; the rule is left to close the header.
	rule := theme.S.RuleIdle.Render(strings.Repeat(theme.RuleLight, max(m.Width, 0)))

	return m.fillTabRow(labels.String(), context, m.connectionSummary()) + "\n" + rule
}

// fillTabRow places the context after the tabs and the status against the right
// edge, giving the context up first when they will not all fit.
func (m *Model) fillTabRow(tabs, context, status string) string {
	const gap = 3

	room := m.Width - ansi.StringWidth(tabs) - ansi.StringWidth(status) - gap*2

	if context != "" && room > 8 {
		tabs += strings.Repeat(" ", gap) + theme.S.Dim.Render(text.Fit(context, room))
	}

	pad := m.Width - ansi.StringWidth(tabs) - ansi.StringWidth(status)
	if pad < 1 {
		return m.padLine(tabs)
	}

	return tabs + strings.Repeat(" ", pad-1) + status + " "
}

// connectionSummary is the dot and the state.
//
// A dot which has been the same colour for twenty minutes is easy to stop
// seeing, so the word stays beside it.
func (m *Model) connectionSummary() string {
	state := string(m.connection.State)
	if state == "" {
		return ""
	}

	style, ok := theme.S.State[state]
	if !ok {
		style = theme.S.Dim
	}

	return style.Render(theme.MarkerPresent + " " + state)
}

// padLine out to the full width.
func (m *Model) padLine(line string) string {
	if padding := m.Width - ansi.StringWidth(line); padding > 0 {
		return line + strings.Repeat(" ", padding)
	}

	return line
}
