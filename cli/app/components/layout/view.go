package layout

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"

	"github.com/skpr/compass/cli/app/styles"
)

// View renders the layout.
func (m Model) View() string {
	var (
		traceInfo   = styles.NewDefaultBoxWithLabel(6).Render(fmt.Sprintf("Traces (%d of %d)", m.Selected+1, len(m.Profiles)), m.TraceInfo.View(), 166)
		requestInfo = styles.NewDefaultBoxWithLabel(6).Render("Request", m.RequestInfo.View(), 166)
		breakdown   = styles.NewDefaultBoxWithLabel(35).Render("Breakdown", m.Breakdown.View(), 166)
		help        = styles.NewDefaultBoxWithLabel(1).Render("Help", m.Help.View(), 166)
	)

	return lipgloss.JoinVertical(lipgloss.Top, traceInfo, requestInfo, breakdown, help)
}
