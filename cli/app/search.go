package app

import (
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/lipgloss"
	"github.com/skpr/compass/cli/app/color"
)

func (m *Model) searchInit() {
	delegate := list.NewDefaultDelegate()
	delegate.Styles.SelectedTitle = lipgloss.NewStyle().
		Border(lipgloss.NormalBorder(), false, false, false, true).
		BorderForeground(lipgloss.AdaptiveColor{Light: color.Blue, Dark: color.Blue}).
		Foreground(lipgloss.AdaptiveColor{Light: color.Blue, Dark: color.Blue}).
		Padding(0, 0, 0, 1)
	delegate.Styles.SelectedDesc = delegate.Styles.SelectedTitle.
		Foreground(lipgloss.AdaptiveColor{Light: color.Blue, Dark: color.Blue})

	search := list.New([]list.Item{}, delegate, 20, 40)
	search.Title = "Traces collected by Compass"
	search.DisableQuitKeybindings()

	styles := list.DefaultStyles()

	styles.Title = lipgloss.NewStyle().
		Foreground(lipgloss.AdaptiveColor{Light: color.Blue, Dark: color.Blue})

	search.Styles = styles

	m.search = search
}

func (m *Model) searchView() string {
	return m.search.View()
}
