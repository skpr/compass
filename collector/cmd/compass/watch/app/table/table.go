// Package table for handling the table component.
package table

import (
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/lipgloss"

	"github.com/skpr/compass/collector/pkg/color"
)

// Render the table.
func Render(columns []table.Column, rows []table.Row, height int) string {
	t := table.New(
		table.WithColumns(columns),
		table.WithRows(rows),
		table.WithHeight(height),
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
