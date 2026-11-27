package app

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/skpr/compass/pkg/app/events"
	"github.com/skpr/compass/pkg/trace"
	"github.com/skpr/compass/pkg/trace/export"
)

// exportedMsg reports where an exported trace was written.
type exportedMsg struct {
	path string
	err  error
}

// notice is a one line outcome shown in the footer until the next key.
type notice struct {
	text string
	err  bool
}

// exportTrace writes the trace being looked at to a file: the open one, or
// the one under the cursor on Search.
//
// The file is the hand-off to something which is not this screen — a ticket,
// a diff against tomorrow's run, or an agent like Claude Code which is
// working on the slow request and can read a file but cannot read a terminal.
func (m *Model) exportTrace() (tea.Model, tea.Cmd) {
	var t trace.Trace

	switch {
	case m.inTrace() && m.Current != nil:
		t = m.Current.Trace
	case m.PageSelected == PageSearch:
		selected, ok := m.selectedTrace()
		if !ok {
			return m, nil
		}

		t = selected.Trace
	default:
		return m, nil
	}

	dir := m.ExportDir

	// Written off the update loop, which a slow disk would otherwise stall.
	return m, func() tea.Msg {
		path, err := export.Write(dir, t)

		return exportedMsg{path: path, err: err}
	}
}

func (m *Model) updateExported(msg exportedMsg) (tea.Model, tea.Cmd) {
	log := events.Log{Time: time.Now(), Type: "info", Message: "exported trace to " + msg.path}

	m.notice = &notice{text: "exported " + msg.path}

	if msg.err != nil {
		log.Type = "error"
		log.Message = msg.err.Error()
		m.notice = &notice{text: msg.err.Error(), err: true}
	}

	// Kept on Logs as well, because the footer forgets it at the next key.
	return m.updateLog(log)
}
