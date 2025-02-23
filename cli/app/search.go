package app

import (
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/lipgloss"

	"github.com/skpr/compass/cli/app/color"
	"github.com/skpr/compass/cli/app/component/table"
)

func (m *Model) searchInit() {
	search := list.New([]list.Item{}, table.GetItemDeletegate(), 20, 40)
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
