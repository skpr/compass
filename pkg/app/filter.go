package app

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"

	"github.com/skpr/compass/pkg/app/theme"
)

// filterInit prepares the filter input.
func (m *Model) filterInit() {
	input := textinput.New()
	input.Prompt = "/ "
	input.Placeholder = "type to search"
	input.PromptStyle = theme.S.FilterPrompt
	input.TextStyle = theme.S.FilterText
	input.PlaceholderStyle = theme.S.FilterPlaceholder

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
	m.relayout()

	return m, m.filter.Focus()
}

// endFilter takes the cursor out of the filter input, keeping what was typed.
func (m *Model) endFilter() {
	m.filterFocused = false
	m.filter.Blur()

	m.currentTable().Focus()
	m.relayout()
}

// clearFilter removes the filter entirely.
func (m *Model) clearFilter() {
	m.filter.SetValue("")
	m.storeFilter(m.PageSelected, "")
	m.endFilter()
	m.refreshRows()
}

// updateFilter passes a key press to the filter input and rebuilds the rows.
func (m *Model) updateFilter(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	m.filter, cmd = m.filter.Update(msg)
	m.storeFilter(m.PageSelected, m.filter.Value())

	// The rows are rebuilt on every keystroke so the list narrows as you type,
	// which is the whole point of a filter you can see.
	m.refreshRows()

	return m, cmd
}

// matches returns the indices of values containing the query, preserving their
// original order. Matching is case-insensitive and contiguous, so a long URL
// cannot match by scattering query characters across unrelated path segments.
func matches(values []string, query string) []int {
	if strings.TrimSpace(query) == "" {
		indices := make([]int, len(values))
		for i := range indices {
			indices[i] = i
		}

		return indices
	}

	query = strings.ToLower(query)
	indices := make([]int, 0, len(values))
	for index, value := range values {
		if strings.Contains(strings.ToLower(value), query) {
			indices = append(indices, index)
		}
	}

	return indices
}

// filterValue returns the query belonging to a page. Search and Logs share a
// query; the two trace pages keep independent values.
func (m *Model) filterValue(page Page) string {
	if sameFilterScope(page, m.PageSelected) {
		return m.filter.Value()
	}

	switch page {
	case PageSearch, PageLogs:
		return m.listFilterValue
	case PageFunctions:
		return m.functionsFilterValue
	case PageDrupal:
		return m.drupalFilterValue
	default:
		return ""
	}
}

func (m *Model) storeFilter(page Page, value string) {
	switch page {
	case PageSearch, PageLogs:
		m.listFilterValue = value
	case PageFunctions:
		m.functionsFilterValue = value
	case PageDrupal:
		m.drupalFilterValue = value
	}
}

func sameFilterScope(left, right Page) bool {
	if (left == PageSearch || left == PageLogs) && (right == PageSearch || right == PageLogs) {
		return true
	}

	return left == right
}

// selectPage saves the current query and restores the target page's before its
// rows are refreshed.
func (m *Model) selectPage(page Page) {
	m.storeFilter(m.PageSelected, m.filter.Value())
	value := m.filterValue(page)
	m.PageSelected = page
	m.filter.SetValue(value)
	m.filterFocused = false
	m.filter.Blur()
	if m.currentTable() != nil {
		m.refreshRows()
	}
}

const searchWidgetMinWidth = 13

// viewFilter renders the current page's filter as a compact terminal widget.
func (m *Model) viewFilter() string {
	if !m.filtering() || m.Width < searchWidgetMinWidth {
		return ""
	}

	border := theme.S.RuleIdle
	label := theme.S.Header
	labelText := "   SEARCH "
	if m.filterFocused {
		border = theme.S.RuleActive
		label = theme.S.Key
		labelText = " " + theme.MarkerItem + " SEARCH "
	}

	topRules := max(m.Width-2-ansi.StringWidth(labelText)-1, 0)
	top := border.Render(theme.CornerTopLeft+theme.RuleLight) +
		label.Render(labelText) +
		border.Render(strings.Repeat(theme.RuleLight, topRules)+theme.CornerTopRight)

	status := m.filterStatus()
	statusWidth := ansi.StringWidth(status)
	contentWidth := max(m.Width-4, 0)
	queryWidth := contentWidth - statusWidth - 2
	if queryWidth < 8 {
		status = ""
		statusWidth = 0
		queryWidth = contentWidth
	}

	query := ansi.Truncate(m.filter.View(), max(queryWidth, 0), theme.MarkerEllipsis)
	gap := max(contentWidth-ansi.StringWidth(query)-statusWidth, 0)

	middle := border.Render(theme.RuleVertical) +
		theme.S.FilterSurface.Render(" ") +
		query +
		theme.S.FilterSurface.Render(strings.Repeat(" ", gap)) +
		theme.S.FilterMeta.Render(status) +
		theme.S.FilterSurface.Render(" ") +
		border.Render(theme.RuleVertical)

	bottom := border.Render(
		theme.CornerBottomLeft +
			strings.Repeat(theme.RuleLight, max(m.Width-2, 0)) +
			theme.CornerBottomRight,
	)

	return strings.Join([]string{top, middle, bottom}, "\n")
}

func (m *Model) filterStatus() string {
	table := m.currentTable()
	if table == nil {
		return ""
	}

	status := fmt.Sprintf("%d shown", table.Len())
	if hidden := m.hiddenByFilter(); hidden > 0 {
		status += fmt.Sprintf(" %s %d hidden", theme.MarkerSeparator, hidden)
	}

	return status
}

// hiddenByFilter is how many rows the filter is holding back.
func (m *Model) hiddenByFilter() int {
	switch m.PageSelected {
	case PageSearch:
		return m.traces.len() - m.search.Len()
	case PageLogs:
		return m.logs.len() - m.logsTable.Len()
	case PageFunctions:
		return len(m.functionSpans) - m.functions.Len()
	case PageDrupal:
		return len(m.drupalEvents) - m.drupal.Len()
	default:
		return 0
	}
}
