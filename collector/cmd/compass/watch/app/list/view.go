package list

import (
	"fmt"

	"github.com/skpr/compass/collector/pkg/styles"
)

// Label for the list component.
const Label = "Collected Profiles"

// View renders the list component.
func (m Model) View() string {
	labelSelected := 0

	if len(m.Profiles) != 0 {
		labelSelected = m.Selected + 1
	}

	label := fmt.Sprintf("%s (%d/%d)", Label, labelSelected, len(m.Profiles))

	return styles.NewDefaultBoxWithLabel(m.Height).Render(label, m.getList(), m.Width)
}
