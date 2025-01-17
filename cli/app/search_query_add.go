package app

import (
	tea "github.com/charmbracelet/bubbletea"
)

// SearchQueryAdd when a alphabetical key is pressed.
func (m *Model) SearchQueryAdd(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	m.searchQuery = m.searchQuery + msg.String()
	m.SearchFilter()
	m.traceSelected = 0
	return m, nil
}
