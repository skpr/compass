package metadata

import (
	"github.com/charmbracelet/bubbles/table"

	compasstable "github.com/skpr/compass/cli/app/components/table"
)

// View renders the list component.
func (m Model) View() string {
	if len(m.Profiles) == 0 {
		return "No profile data available"
	}

	if len(m.Profiles) < m.Selected {
		return "Incorrect profile selected"
	}

	profile := m.Profiles[m.Selected]

	columns := []table.Column{
		{Title: "URI", Width: 45},
		{Title: "Method", Width: 35},
	}

	rows := []table.Row{
		{
			profile.URI,
			profile.Method,
		},
	}

	return compasstable.Render(columns, rows, 3)
}
