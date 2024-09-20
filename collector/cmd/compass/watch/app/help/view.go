package help

import (
	"github.com/skpr/compass/collector/pkg/styles"
)

const (
	// Label for the help component.
	Label = "How To Use"
	// Message for how to operate the application.
	Message = "Left and Right = Select Profile\tUp and Down = Scroll Profile Breakdown\tCtrl + C = Exit Window"
)

// View renders the help component.
func (m Model) View() string {
	return styles.NewDefaultBoxWithLabel(m.Height).Render(Label, Message, m.Width)
}
