package app

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/skpr/compass/cli/app/types"
)

func (m *Model) updateKeyEnter() (tea.Model, tea.Cmd) {
	if m.PageSelected != types.PageSearch {
		return m, nil
	}

	trace, ok := m.search.SelectedItem().(types.Trace)
	if ok {
		m.Current = &trace
		m.PageSelected = types.PageSpans
	}

	m.metadataSetRows()
	m.spansSetRows()
	m.totalsSetRows()

	return m, nil
}
