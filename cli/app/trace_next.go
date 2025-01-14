package app

import tea "github.com/charmbracelet/bubbletea"

// TraceNext when the right arrow key is pressed.
func (m *Model) TraceNext() (tea.Model, tea.Cmd) {
	if m.traceSelected < len(m.traces.Filtered)-1 {
		m.traceSelected++
		m.breakdownScroll = 0
	}

	return m, nil
}
