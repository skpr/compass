package app

import (
	tea "github.com/charmbracelet/bubbletea"
)

func (m *Model) updateKeyEnter() (tea.Model, tea.Cmd) {
	if m.PageSelected != PageSearch {
		return m, nil
	}

	// Opening a trace is the only way into the pages which view one, so there
	// is nothing to open when the search list has no selection.
	selected, ok := m.selectedTrace()
	if !ok {
		return m, nil
	}

	m.Current = &selected

	// A trace starts with clean page-specific filters. The Search/Logs query is
	// saved and restored when the trace is closed.
	m.functionsFilterValue = ""
	m.drupalFilterValue = ""
	m.functionSpans = nil
	m.functionVisible = nil
	m.drupalEvents = nil
	m.drupalVisible = nil
	m.selectPage(PageFunctions)

	// The rows are built before the layout rather than after it. The panel
	// below the table describes the row under the cursor, so its height is a
	// property of the rows: laying out first sizes it for a page which has no
	// rows yet, and the panel then renders taller than the region it was given
	// and scrolls the masthead off the top of the terminal.
	m.drupalSetRows()

	m.functions.GotoTop()
	m.drupal.GotoTop()

	// The trace pages carry a strip the top level does not, so the regions
	// change along with the page.
	m.relayout()

	return m, nil
}
