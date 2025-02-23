// Package table for formatting.
package table

import (
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/lipgloss"

	"github.com/skpr/compass/cli/app/color"
)

// GetItemDeletegate for formatting.
func GetItemDeletegate() list.ItemDelegate {
	delegate := list.NewDefaultDelegate()

	delegate.Styles.SelectedTitle = lipgloss.NewStyle().
		Border(lipgloss.NormalBorder(), false, false, false, true).
		BorderForeground(lipgloss.AdaptiveColor{Light: color.Blue, Dark: color.Blue}).
		Foreground(lipgloss.AdaptiveColor{Light: color.Blue, Dark: color.Blue}).
		Padding(0, 0, 0, 1)

	delegate.Styles.SelectedDesc = delegate.Styles.SelectedTitle.
		Foreground(lipgloss.AdaptiveColor{Light: color.Blue, Dark: color.Blue})

	return delegate
}
