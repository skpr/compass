package app

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/skpr/compass/pkg/app/component/datatable"
	"github.com/skpr/compass/pkg/app/events"
)

// Update triggers on messages and updates the model.
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	// Handle key presses.
	case tea.KeyMsg:
		// Bubble Tea can emit the ESC byte of an arrow/wheel sequence as a
		// standalone key when the terminal splits the sequence across reads.
		if m.cancelTraceCloseForContinuation(msg) {
			return m, nil
		}

		// While the filter has the cursor almost every key is text, so it is
		// asked first and only the keys which leave it are taken away.
		if m.filterFocused {
			switch msg.String() {
			case tea.KeyCtrlC.String():
				return m, tea.Quit
			case tea.KeyEsc.String():
				m.clearFilter()

				return m, nil
			case tea.KeyEnter.String(), tea.KeyDown.String(), tea.KeyUp.String():
				m.endFilter()

				return m, nil
			}

			return m.updateFilter(msg)
		}

		switch msg.String() {
		case tea.KeyCtrlC.String(), "q":
			return m, tea.Quit

		case "/":
			if m.filterable() {
				return m.startFilter()
			}

		case "?":
			m.showHelp = !m.showHelp

			return m, nil

		// For navigating the main menu.
		case tea.KeyRight.String():
			return m.updateKeyRight()
		case tea.KeyLeft.String():
			return m.updateKeyLeft()

		case tea.KeyEnter.String():
			return m.updateKeyEnter()

		// Ours while something is open. Otherwise it belongs to the page.
		case tea.KeyEsc.String():
			if m.filtering() {
				m.clearFilter()

				return m, nil
			}

			if m.showHelp {
				return m.updateKeyEsc()
			}
			if m.inTrace() {
				return m.deferTraceClose()
			}
		}

	case deferredTraceCloseMsg:
		return m.updateDeferredTraceClose(msg)

	case tea.WindowSizeMsg:
		return m.updateWindowSize(msg)

	case events.Trace:
		return m.updateTrace(msg)

	case events.Log:
		return m.updateLog(msg)

	case events.Connection:
		return m.updateConnection(msg)
	}

	// Everything the chrome did not claim goes to whichever page is showing,
	// unless the help is over it.
	if m.showHelp {
		return m, nil
	}

	if table := m.currentTable(); table != nil {
		_, cmd := table.Update(msg)

		return m, cmd
	}

	return m, nil
}

// filterable reports whether the page on screen has rows worth narrowing.
func (m *Model) filterable() bool {
	return m.currentTable() != nil
}

// currentTable of the selected page.
func (m *Model) currentTable() *datatable.Model {
	switch m.PageSelected {
	case PageSearch:
		return m.search
	case PageLogs:
		return m.logsTable
	case PageFunctions:
		return m.functions
	case PageDrupal:
		return m.drupal
	default:
		return nil
	}
}
