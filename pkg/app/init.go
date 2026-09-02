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
	if m.MaxLogs <= 0 {
		m.MaxLogs = DefaultMaxLogs
	}

	m.traces.setLimit(m.MaxTraces)
	m.logs.setLimit(m.MaxLogs)
	m.logEntries.setLimit(m.MaxLogs)

	m.searchInit()
	m.logsInit()
	m.functionsInit()
	m.drupalInit()
	m.filterInit()

	m.relayout()

	return nil
}
