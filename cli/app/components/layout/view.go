package layout

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"

	"github.com/skpr/compass/cli/app/styles"
)

// View renders the layout.
func (m Model) View() string {
	var (
		search    = styles.NewDefaultBoxWithLabel(1).Render("Search", m.Search.View(), 166)
		info      = styles.NewDefaultBoxWithLabel(6).Render(fmt.Sprintf("Information (%d of %d)", m.Selected+1, m.Total), m.Info.View(), 166)
		breakdown = styles.NewDefaultBoxWithLabel(35).Render("Breakdown", m.Spans.View(), 166)
		help      = styles.NewDefaultBoxWithLabel(1).Render("Help", m.Help.View(), 166)
	)

	return lipgloss.JoinVertical(lipgloss.Top, search, info, breakdown, help)
}
