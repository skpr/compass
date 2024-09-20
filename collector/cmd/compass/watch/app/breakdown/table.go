package breakdown

import (
	"fmt"
	"sort"

	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/lipgloss"

	"github.com/skpr/compass/collector/pkg/color"
)

// Prints the table component.
func (m Model) renderTable() string {
	if len(m.Profiles) == 0 {
		return "No profile data available"
	}

	if len(m.Profiles) < m.Selected {
		return "Incorrect profile selected"
	}

	profile := m.Profiles[m.Selected]

	if len(profile.Functions) == 0 {
		return "No functions available for profile"
	}

	sort.Slice(profile.Functions, func(i, j int) bool {
		return profile.Functions[i].TotalExecutionTime > profile.Functions[j].TotalExecutionTime
	})

	var rows []table.Row

	start, end := getPositionStartAndEnd(m.ScrollPosition, m.VisibleRows, len(profile.Functions))

	for i, f := range profile.Functions {
		if i < start || i >= end {
			continue
		}

		rows = append(rows, []string{
			f.Name,
			getExecutionGraph(f.TotalExecutionTime, profile.ExecutionTime),
			fmt.Sprintf("%d", f.Invocations),
		})
	}

	columns := []table.Column{
		{Title: "Function", Width: 109},
		{Title: "Execution Time", Width: 30},
		{Title: "Invocations", Width: 15},
	}

	t := table.New(
		table.WithColumns(columns),
		table.WithRows(rows),
		table.WithHeight(m.Height-3),
	)

	s := table.DefaultStyles()

	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderBottom(true).
		Foreground(lipgloss.Color(color.White)).
		Bold(true)

	s.Selected = s.Selected.
		Foreground(lipgloss.Color(color.White)).
		Bold(false)

	s.Cell = s.Cell.
		Foreground(lipgloss.Color(color.White)).
		Bold(false)

	t.SetStyles(s)

	return t.View()
}
