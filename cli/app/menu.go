package app

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/skpr/compass/cli/app/color"
	"github.com/skpr/compass/cli/app/types"
)

var (
	highlight = lipgloss.AdaptiveColor{Light: color.Blue, Dark: color.Blue}

	activeTabBorder = lipgloss.Border{
		Top:         "─",
		Bottom:      " ",
		Left:        "│",
		Right:       "│",
		TopLeft:     "╭",
		TopRight:    "╮",
		BottomLeft:  "┘",
		BottomRight: "└",
	}

	tabBorder = lipgloss.Border{
		Top:         "─",
		Bottom:      "─",
		Left:        "│",
		Right:       "│",
		TopLeft:     "╭",
		TopRight:    "╮",
		BottomLeft:  "┴",
		BottomRight: "┴",
	}

	tab = lipgloss.NewStyle().
		Border(tabBorder, true).
		BorderForeground(highlight).
		Padding(0, 1)

	activeTab = tab.Border(activeTabBorder, true)

	tabGap = tab.
		BorderTop(false).
		BorderLeft(false).
		BorderRight(false)
)

func (m *Model) viewMenu() string {
	var (
		labelSearch  = fmt.Sprintf("%s (%d)", types.PageSearch, len(m.search.Items()))
		labelsSpans  = string(types.PageSpans)
		labelsTotals = string(types.PageTotals)
		labelLogs    = fmt.Sprintf("%s (%d)", types.PageLogs, len(m.logs.Items()))
	)

	row := lipgloss.JoinHorizontal(
		lipgloss.Top,
		renderTab(labelSearch, m.PageSelected == types.PageSearch),
		renderTab(labelsSpans, m.PageSelected == types.PageSpans),
		renderTab(labelsTotals, m.PageSelected == types.PageTotals),
		renderTab(labelLogs, m.PageSelected == types.PageLogs),
	)

	gap := tabGap.Render(strings.Repeat(" ", max(0, m.Width-lipgloss.Width(row)-2)))

	return lipgloss.JoinHorizontal(lipgloss.Bottom, row, gap)
}

func renderTab(label string, active bool) string {
	if active {
		return activeTab.Render(label)
	}

	return tab.Render(label)
}
