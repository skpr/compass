package app

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/skpr/compass/cli/app/events"
)

func (m *Model) updateKeyEnter() (tea.Model, tea.Cmd) {
	if m.PageSelected != PageSearch {
		return m, nil
	}

	trace, ok := m.search.SelectedItem().(events.Trace)
	if ok {
		m.Current = &trace
		m.PageSelected = PageSpans
	}

	m.metadataSetRows()
	m.spansSetRows()
	m.totalsSetRows()

	return m, nil
}
