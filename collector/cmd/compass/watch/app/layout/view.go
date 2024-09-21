package layout

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"

	"github.com/skpr/compass/collector/pkg/styles"
)

// View renders the layout.
func (m Model) View() string {
	var (
		info      = styles.NewDefaultBoxWithLabel(6).Render(fmt.Sprintf("Info (%d of %d)", m.Selected+1, len(m.Profiles)), m.Info.View(), 164)
		breakdown = styles.NewDefaultBoxWithLabel(35).Render("Breakdown", m.Breakdown.View(), 164)
		help      = styles.NewDefaultBoxWithLabel(1).Render("Help", m.Help.View(), 164)
	)

	return lipgloss.JoinVertical(lipgloss.Top, info, breakdown, help)
}
