// Package styles for handling the styles applied to the CLI.
package styles

import (
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/skpr/compass/collector/pkg/color"
)

// BoxWithLabel is a style for a box with a label.
type BoxWithLabel struct {
	BoxStyle   lipgloss.Style
	LabelStyle lipgloss.Style
}

// NewDefaultBoxWithLabel returns a new BoxWithLabel with default styles.
func NewDefaultBoxWithLabel(height int) BoxWithLabel {
	return BoxWithLabel{
		BoxStyle: lipgloss.NewStyle().
			Border(lipgloss.ThickBorder()).
			BorderForeground(lipgloss.Color(color.Orange)).
			Padding(1, 0).
			Height(height),

		// You could, of course, also set background and foreground colors here
		// as well.
		LabelStyle: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color(color.White)).
			PaddingTop(0).
			PaddingBottom(0).
			PaddingLeft(1).
			PaddingRight(1),
	}
}

// Render renders a box with a label.
func (b BoxWithLabel) Render(label, content string, width int) string {
	var (
		border          = b.BoxStyle.GetBorderStyle()
		topBorderStyler = lipgloss.NewStyle().Foreground(b.BoxStyle.GetBorderTopForeground()).Render
		topLeft         = topBorderStyler(border.TopLeft)
		topRight        = topBorderStyler(border.TopRight)
		renderedLabel   = b.LabelStyle.Render(label)
	)

	// Render top row with the label
	borderWidth := b.BoxStyle.GetHorizontalBorderSize()
	cellsShort := max(0, width+borderWidth-lipgloss.Width(topLeft+topRight+renderedLabel))
	gap := strings.Repeat(border.Top, cellsShort)
	top := topLeft + renderedLabel + topBorderStyler(gap) + topRight

	// Render the rest of the box
	bottom := b.BoxStyle.Copy().
		BorderTop(false).
		Width(width).
		Padding(1, 2).
		Render(content)

	// Stack the pieces
	return top + "\n" + bottom
}
