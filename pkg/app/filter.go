package app

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/sahilm/fuzzy"

	"github.com/skpr/compass/pkg/app/theme"
)

// filterInit prepares the filter input.
func (m *Model) filterInit() {
	input := textinput.New()
	input.Prompt = "/"
	input.Placeholder = "filter"
	input.PromptStyle = theme.S.Key
	input.TextStyle = theme.S.Primary
	input.PlaceholderStyle = theme.S.Faint

	m.filter = input
}

// filtering reports whether a filter is being typed or is in force.
func (m *Model) filtering() bool {
	return m.filterFocused || m.filter.Value() != ""
}

// startFilter puts the cursor in the filter input.
func (m *Model) startFilter() (tea.Model, tea.Cmd) {
	m.filterFocused = true

	m.currentTable().Blur()

	return m, m.filter.Focus()
}

// endFilter takes the cursor out of the filter input, keeping what was typed.
func (m *Model) endFilter() {
	m.filterFocused = false
	m.filter.Blur()

	m.currentTable().Focus()
}

// clearFilter removes the filter entirely.
func (m *Model) clearFilter() {
	m.filter.SetValue("")
	m.endFilter()
	m.refreshRows()
}

// updateFilter passes a key press to the filter input and rebuilds the rows.
func (m *Model) updateFilter(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	m.filter, cmd = m.filter.Update(msg)

	// The rows are rebuilt on every keystroke so the list narrows as you type,
	// which is the whole point of a filter you can see.
	m.refreshRows()

	return m, cmd
}

// matches returns the indices of the values which match the filter, in the
// order they should be shown.
//
// Fuzzy rather than substring: a trace is found by typing the parts of a path
// you remember, not the contiguous run you would have to look up.
func matches(values []string, query string) []int {
	if strings.TrimSpace(query) == "" {
		indices := make([]int, len(values))
		for i := range indices {
			indices[i] = i
		}

		return indices
	}

	found := fuzzy.Find(query, values)

	indices := make([]int, 0, len(found))
	for _, match := range found {
		indices = append(indices, match.Index)
	}

	return indices
}

// viewFilter renders the input, or nothing when no filter is in play.
func (m *Model) viewFilter() string {
	if !m.filtering() {
		return ""
	}

	line := " " + m.filter.View()

	if hidden := m.hiddenByFilter(); hidden > 0 {
		line += theme.S.Faint.Render(fmt.Sprintf("   %d hidden", hidden))
	}

	return m.padLine(line)
}

// hiddenByFilter is how many rows the filter is holding back.
func (m *Model) hiddenByFilter() int {
	switch m.PageSelected {
	case PageSearch:
		return len(m.traces) - m.search.Len()
	case PageLogs:
		return len(m.logs) - m.logsTable.Len()
	default:
		return 0
	}
}
