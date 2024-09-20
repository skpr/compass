package app

import (
	"github.com/charmbracelet/lipgloss"

	"github.com/skpr/compass/collector/cmd/compass/watch/app/breakdown"
	"github.com/skpr/compass/collector/cmd/compass/watch/app/help"
	"github.com/skpr/compass/collector/cmd/compass/watch/app/list"
)

const (
	// Width for all components.
	Width = 164
	// ListHeight is the height applied to the list component.
	ListHeight = 6
	// VisibleProfiles is the number of profiles visible in the list.
	VisibleProfiles = 4
	// BreakdownHeight is the height applied to the breakdown component.
	BreakdownHeight = 30 // This accounts for the header and border.
	// BreakdownRows is the number of rows visible in the breakdown
	BreakdownRows = 26
	// HelpHeight is the height applied to the help component.
	HelpHeight = 1
)

// View refreshes the display.
func (m Model) View() string {
	renderedList := list.Model{
		Profiles:        m.profiles,
		Selected:        m.profileSelected,
		VisibleProfiles: VisibleProfiles,
		Height:          ListHeight,
		Width:           Width,
	}.View()

	renderedBreakdown := breakdown.Model{
		Profiles:       m.profiles,
		Selected:       m.profileSelected,
		ScrollPosition: m.breakdownScroll,
		VisibleRows:    BreakdownRows,
		Height:         BreakdownHeight,
		Width:          Width,
	}.View()

	renderedHelp := help.Model{
		Height: HelpHeight,
		Width:  Width,
	}.View()

	return lipgloss.JoinVertical(lipgloss.Top, renderedList, renderedBreakdown, renderedHelp)
}
