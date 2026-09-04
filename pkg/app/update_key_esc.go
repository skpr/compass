package app

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

const traceCloseEscapeWait = 50 * time.Millisecond

type deferredTraceCloseMsg struct {
	sequence uint64
}

// deferTraceClose gives the remainder of an ESC-prefixed terminal navigation
// sequence time to arrive before treating its first byte as a standalone Esc.
func (m *Model) deferTraceClose() (tea.Model, tea.Cmd) {
	m.traceCloseSequence++
	m.traceClosePending = true
	sequence := m.traceCloseSequence

	return m, tea.Tick(traceCloseEscapeWait, func(time.Time) tea.Msg {
		return deferredTraceCloseMsg{sequence: sequence}
	})
}

// cancelTraceCloseForContinuation consumes the suffix left behind when Bubble
// Tea receives an ANSI sequence in a separate read from its leading ESC byte.
func (m *Model) cancelTraceCloseForContinuation(msg tea.KeyMsg) bool {
	if !m.traceClosePending || msg.Type != tea.KeyRunes || len(msg.Runes) == 0 {
		return false
	}

	if msg.Runes[0] != '[' && msg.Runes[0] != 'O' {
		return false
	}

	m.traceClosePending = false
	m.traceCloseSequence++

	return true
}

// updateDeferredTraceClose closes the trace only if no ANSI continuation has
// invalidated the timer which was created for this Escape key.
func (m *Model) updateDeferredTraceClose(msg deferredTraceCloseMsg) (tea.Model, tea.Cmd) {
	if !m.traceClosePending || msg.sequence != m.traceCloseSequence || !m.inTrace() {
		return m, nil
	}

	return m.updateKeyEsc()
}

// updateKeyEsc closes whatever is open: the help overlay first, then the trace.
func (m *Model) updateKeyEsc() (tea.Model, tea.Cmd) {
	m.traceClosePending = false
	m.traceCloseSequence++

	if m.showHelp {
		m.showHelp = false

		return m, nil
	}

	m.selectPage(PageSearch)
	m.relayout()

	return m, nil
}
