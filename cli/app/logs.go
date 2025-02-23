package app

import (
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/lipgloss"

	"github.com/skpr/compass/cli/app/color"
	"github.com/skpr/compass/cli/app/component/table"
)

func (m *Model) logsInit() {
	logs := list.New([]list.Item{}, table.GetItemDeletegate(), 20, 40)
	logs.Title = "Log events while capturing traces"
	logs.DisableQuitKeybindings()

	styles := list.DefaultStyles()

	styles.Title = lipgloss.NewStyle().
		Foreground(lipgloss.AdaptiveColor{Light: color.Blue, Dark: color.Blue})

	logs.Styles = styles

	m.logs = logs
}

func (m *Model) logsView() string {
	return m.logs.View()
}
