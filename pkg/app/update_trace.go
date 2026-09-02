package app

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/skpr/compass/pkg/app/events"
)

func (m *Model) updateTrace(event events.Trace) (tea.Model, tea.Cmd) {
	m.ensureTraceHistory()
	m.traces.append(event)

	// An unfiltered arrival affects exactly one visible row. The datatable's
	// bounded front insertion preserves the selected logical row and avoids a
	// full values/filter/rows rebuild on the common path.
	if strings.TrimSpace(m.filterValue(PageSearch)) == "" && m.search != nil {
		m.visible = nil
		m.search.PrependRowBounded(m.traceRow(event), m.MaxTraces)
	} else if m.search != nil {
		m.searchSetRows()
	}

	return m, nil
}

func (m *Model) ensureTraceHistory() {
	limit := m.MaxTraces
	if limit <= 0 {
		limit = DefaultMaxTraces
		m.MaxTraces = limit
	}

	m.traces.setLimit(limit)
}
