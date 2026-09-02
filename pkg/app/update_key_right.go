package app

import (
	tea "github.com/charmbracelet/bubbletea"
)

func (m *Model) updateKeyRight() (tea.Model, tea.Cmd) {
	switch m.PageSelected {
	case PageSearch:
		m.selectPage(PageLogs)
	case PageFunctions:
		// Only where there is a page to move to. The tab is not on screen for a
		// trace with no Drupal data, and a key which moves you somewhere you
		// cannot see is worse than one which does nothing.
		if m.hasDrupal() {
			m.selectPage(PageDrupal)
		}
	}

	m.relayout()

	return m, nil
}
