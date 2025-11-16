package app

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var (
	docStyle = lipgloss.NewStyle()
)

// View for this model.
func (m Model) View() string {
	doc := strings.Builder{}

	doc.WriteString(m.viewMenu() + "\n")

	switch m.PageSelected {
	case PageSearch:
		doc.WriteString(m.searchView() + "\n\n")
	case PageSpans:
		doc.WriteString(m.metadataView() + "\n")
		doc.WriteString(m.spansView() + "\n")
	case PageTotals:
		doc.WriteString(m.metadataView() + "\n")
		doc.WriteString(m.totalsView() + "\n")
	case PageLogs:
		doc.WriteString(m.logsView() + "\n\n")
	}

	doc.WriteString(m.footerView())

	return docStyle.Render(doc.String())
}
