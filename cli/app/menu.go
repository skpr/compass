package app

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/skpr/compass/cli/app/color"
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
		labelSearch  = fmt.Sprintf("%s (%d)", PageSearch, len(m.search.Items()))
		labelsSpans  = string(PageSpans)
		labelsTotals = string(PageTotals)
		labelLogs    = fmt.Sprintf("%s (%d)", PageLogs, len(m.logs.Items()))
	)

	row := lipgloss.JoinHorizontal(
		lipgloss.Top,
		renderTab(labelSearch, m.PageSelected == PageSearch),
		renderTab(labelsSpans, m.PageSelected == PageSpans),
		renderTab(labelsTotals, m.PageSelected == PageTotals),
		renderTab(labelLogs, m.PageSelected == PageLogs),
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
