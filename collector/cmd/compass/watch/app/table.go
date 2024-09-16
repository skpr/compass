package app

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/lipgloss"

	"github.com/skpr/compass/collector/pkg/color"
)

// Helper function for printing the table component.
func (m Model) profileTable() string {
	if len(m.profiles) == 0 {
		return "No profile data available"
	}

	if len(m.profiles) < m.selectedProfile {
		return "Incorrect profile selected"
	}

	profile := m.profiles[m.selectedProfile]

	if len(profile.Functions) == 0 {
		return "No functions available for profile"
	}

	sort.Slice(profile.Functions, func(i, j int) bool {
		return profile.Functions[i].TotalExecutionTime > profile.Functions[j].TotalExecutionTime
	})

	var rows []table.Row

	start, end := getTableDataStarAndEnd(m.profileScroll, TableRows, len(profile.Functions))

	for i, f := range profile.Functions {
		if i < start || i >= end {
			continue
		}

		rows = append(rows, []string{
			f.Name,
			fmt.Sprintf("%s (%vms)", strings.Repeat("█", int(f.TotalExecutionTime/profile.ExecutionTime*20)), f.TotalExecutionTime),
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
		table.WithHeight(TableHeight),
	)

	s := table.DefaultStyles()

	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240")).
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

// Helper function which returns the start and end of the list that should be displayed.
func getTableDataStarAndEnd(position, visible, length int) (int, int) {
	// If the length is less than the visible amount, show all.
	if visible > length {
		return 0, length
	}

	// If the position plus the visible amount is greater than the length, show the last visible amount.
	if position+visible > length {
		return length - visible, length
	}

	return position, position + visible
}
