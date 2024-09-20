package breakdown

import "github.com/skpr/compass/collector/pkg/styles"

// Label applied to the breakdown component.
const Label = "Profile Breakdown"

// View renders the breakdown component.
func (m Model) View() string {
	return styles.NewDefaultBoxWithLabel(m.Height).Render(Label, m.renderTable(), m.Width)
}
