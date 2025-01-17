package app

import (
	tea "github.com/charmbracelet/bubbletea"
)

// SearchQueryDelete is triggered when the delete key is pressed.
func (m *Model) SearchQueryDelete() (tea.Model, tea.Cmd) {
	if len(m.searchQuery) > 0 {
		r := []rune(m.searchQuery)
		m.searchQuery = string(r[:len(r)-1])
	}

	m.SearchFilter()

	return m, nil
}
