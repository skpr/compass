package app

import (
	tea "github.com/charmbracelet/bubbletea"
)

func (m *Model) updateKeySummary() (tea.Model, tea.Cmd) {
	// Only available on the Spans page with a trace selected.
	if m.PageSelected != PageSpans || m.Current == nil {
		return m, nil
	}

	// Toggle off if already showing.
	if m.showSummary {
		m.showSummary = false
		return m, nil
	}

	// Show the overlay.
	m.showSummary = true

	traceID := m.Current.Metadata.ID

	// Check the cache first.
	if cached, ok := m.summaryCache[traceID]; ok {
		m.summaryText = cached
		m.summaryLoading = false
		m.summaryError = ""
		return m, nil
	}

	// Not cached, start loading from Ollama.
	m.summaryLoading = true
	m.summaryText = ""
	m.summaryError = ""

	return m, fetchSummary(m.OllamaURL, m.OllamaModel, traceID, m.Current.Trace)
}
