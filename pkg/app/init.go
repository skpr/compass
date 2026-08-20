package app

import (
	tea "github.com/charmbracelet/bubbletea"
)

// Init initializes the model.
func (m *Model) Init() tea.Cmd {
	m.PageSelected = PageSearch

	if m.MaxTraces <= 0 {
		m.MaxTraces = DefaultMaxTraces
	}

	m.searchInit()
	m.logsInit()
	m.metadataInit()
	m.spansInit()
	m.totalsInit()

	return nil
}
