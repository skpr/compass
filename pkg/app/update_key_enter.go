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

	m.PageSelected = PageFunctions

	// The trace pages carry a strip the top level does not, so the regions
	// change along with the page.
	m.relayout()

	m.functions.GotoTop()
	m.drupal.GotoTop()

	m.functionsSetRows()
	m.drupalSetRows()

	return m, nil
}
