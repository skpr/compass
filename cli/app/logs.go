package app

import (
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/lipgloss"
	"github.com/skpr/compass/cli/app/color"
)

func (m *Model) logsInit() {
	delegate := list.NewDefaultDelegate()
	delegate.Styles.SelectedTitle = lipgloss.NewStyle().
		Border(lipgloss.NormalBorder(), false, false, false, true).
		BorderForeground(lipgloss.AdaptiveColor{Light: color.Blue, Dark: color.Blue}).
		Foreground(lipgloss.AdaptiveColor{Light: color.Blue, Dark: color.Blue}).
		Padding(0, 0, 0, 1)
	delegate.Styles.SelectedDesc = delegate.Styles.SelectedTitle.
		Foreground(lipgloss.AdaptiveColor{Light: color.Blue, Dark: color.Blue})

	logs := list.New([]list.Item{}, delegate, 20, 40)
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
