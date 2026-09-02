package app

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/skpr/compass/pkg/app/events"
)

func (m *Model) updateLog(event events.Log) (tea.Model, tea.Cmd) {
	m.ensureLogHistory()
	newRun := m.appendLog(event)

	if strings.TrimSpace(m.filterValue(PageLogs)) != "" || m.logsTable == nil {
		if m.logsTable != nil {
			m.logsSetRows()
		}

		return m, nil
	}

	newest, _ := m.logEntries.newest(0)
	if newRun {
		m.logsTable.PrependRowBounded(m.logRow(newest), m.MaxLogs)
	} else {
		m.logsTable.SetRow(0, m.logRow(newest))
	}

	// Evicting one raw event can remove the oldest run or only decrement it.
	// Both are edge operations on the collapsed table.
	m.logsTable.TrimRows(m.logEntries.len())
	if oldest, ok := m.logEntries.oldest(0); ok {
		m.logsTable.SetRow(m.logEntries.len()-1, m.logRow(oldest))
	}

	return m, nil
}

// appendLog updates raw retention and its run-length encoded view in O(1).
// It reports whether the new event started a run.
func (m *Model) appendLog(event events.Log) bool {
	_, evicted := m.logs.append(event)
	if evicted {
		oldest, _ := m.logEntries.oldest(0)
		oldest.Repeats--
		if oldest.Repeats == 0 {
			m.logEntries.removeOldest()
		} else {
			m.logEntries.setOldest(0, oldest)
		}
	}

	if newest, ok := m.logEntries.newest(0); ok && sameLogRun(newest.Log, event) {
		newest.Repeats++
		if event.Time.After(newest.Time) {
			newest.Time = event.Time
		}
		m.logEntries.setNewest(0, newest)

		return false
	}

	m.logEntries.append(logEntry{Log: event, Repeats: 1})

	return true
}

func (m *Model) ensureLogHistory() {
	limit := m.MaxLogs
	if limit <= 0 {
		limit = DefaultMaxLogs
		m.MaxLogs = limit
	}

	// Limits are normally fixed at construction. If an embedding changes one,
	// preserve the bounded raw history and rebuild its run index once.
	if m.logs.limit() != limit {
		m.logs.setLimit(limit)
		m.rebuildLogEntries()
	}
	m.logEntries.setLimit(limit)
}

func (m *Model) rebuildLogEntries() {
	m.logEntries = newHistory[logEntry](m.MaxLogs)

	for i := 0; i < m.logs.len(); i++ {
		event, _ := m.logs.oldest(i)
		if newest, ok := m.logEntries.newest(0); ok && sameLogRun(newest.Log, event) {
			newest.Repeats++
			if event.Time.After(newest.Time) {
				newest.Time = event.Time
			}
			m.logEntries.setNewest(0, newest)
			continue
		}

		m.logEntries.append(logEntry{Log: event, Repeats: 1})
	}
}

func sameLogRun(a, b events.Log) bool {
	return a.Message == b.Message && a.Type == b.Type
}
