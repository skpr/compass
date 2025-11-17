package app

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/skpr/compass/tracing/cli/app/events"
)

// Init initializes the model.
func (m *Model) Init() tea.Cmd {
	m.PageSelected = PageSearch

	m.Traces = make(map[string]events.Trace)

	m.searchInit()
	m.logsInit()
	m.metadataInit()
	m.spansInit()
	m.totalsInit()

	return nil
}
