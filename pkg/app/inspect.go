package app

import (
	"fmt"
	"strings"

	"github.com/skpr/compass/pkg/app/component/text"
	"github.com/skpr/compass/pkg/app/theme"
)

// inspectLabelWidth is the column the field names sit in.
const inspectLabelWidth = 9

// inspectLines for whichever page is showing, or none when it has no panel.
func (m *Model) inspectLines() []string {
	switch m.PageSelected {
	case PageFunctions:
		return m.functionsInspectLines()
	case PageDrupal:
		return m.drupalInspectLines()
	default:
		return nil
	}
}

// inspectRules are the lines which close the panel above and below.
const inspectRules = 2

// inspectHeight the panel needs, including its rules.
//
// It differs by page, because the pages have different amounts to say: a
// cacheability event has a caller, an object and two lists behind it, and a
// function call has a name and two numbers. Padding the shorter one out to
// match would be blank rows rather than symmetry.
func (m *Model) inspectHeight() int {
	lines := m.inspectLines()
	if len(lines) == 0 {
		return 0
	}

	return len(lines) + inspectRules
}

// inspectView renders the panel below a table, showing the row the cursor is on
// in full.
//
// Both tables abbreviate. A column wide enough for a fully qualified PHP name,
// or for a list of cache tags, would take the width the rest of the row needs
// and still cut it in half — so they show as much as fits and this says the
// rest. It follows the cursor, which makes moving down a table the way to read
// through the detail rather than something you do and then look elsewhere.
func (m *Model) inspectView() string {
	lines := m.inspectLines()
	if len(lines) == 0 {
		return ""
	}

	rule := theme.S.RuleIdle.Render(strings.Repeat(theme.RuleLight, max(m.Width, 0)))

	// Closed at both ends. The rule above separates the panel from the table it
	// describes; without one below, its last line runs straight into the key
	// rail and the two read as one block of text.
	//
	// A page always renders the same number of lines whichever row the cursor
	// is on, so the table above does not move as you walk down it.
	framed := append([]string{rule}, lines...)

	return strings.Join(append(framed, rule), "\n")
}

// inspectValue of a single labelled value.
func (m *Model) inspectValue(label, value string) string {
	rendered := theme.S.Header.Render(fmt.Sprintf(" %-*s", inspectLabelWidth, label))

	if value == "" {
		return m.padLine(rendered + theme.S.Faint.Render("none"))
	}

	return m.padLine(text.Fit(rendered+theme.S.Cell.Render(value), m.Width))
}

// inspectList of a labelled list of values.
func (m *Model) inspectList(label string, values []string) string {
	if len(values) == 0 {
		return m.inspectValue(label, "")
	}

	// Two spaces between values rather than one: a cache tag contains a colon
	// and no spaces, so a single space between them reads as one long tag.
	return m.inspectValue(label, strings.Join(values, "  "))
}

// inspectMissing when there is no row under the cursor to describe.
func (m *Model) inspectMissing(what string) []string {
	return []string{m.padLine(" " + theme.S.Faint.Render(what))}
}
