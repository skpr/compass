package app

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"

	"github.com/skpr/compass/collector/pkg/styles"
)

// View refreshes the display.
func (m Model) View() string {
	labelSelected := 0

	if len(m.profiles) != 0 {
		labelSelected = m.selectedProfile + 1
	}

	renderedList := styles.NewDefaultBoxWithLabel(6).Render(fmt.Sprintf("Collected Profiles (%d/%d)", labelSelected, len(m.profiles)), m.profileList(), Width)
	summary := styles.NewDefaultBoxWithLabel(31).Render("Profile Summary", m.profileTable(), Width)
	tooltipView := styles.NewDefaultBoxWithLabel(1).Render("How To Use", howMessage, Width)

	return lipgloss.JoinVertical(lipgloss.Top, renderedList, summary, tooltipView)
}
