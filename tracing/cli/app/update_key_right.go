package app

import (
	tea "github.com/charmbracelet/bubbletea"
)

func (m *Model) updateKeyRight() (tea.Model, tea.Cmd) {
	switch m.PageSelected {
	case PageSearch:
		m.PageSelected = PageSpans
	case PageSpans:
		m.PageSelected = PageTotals
	case PageTotals:
		m.PageSelected = PageLogs
	}

	return m, nil
}
