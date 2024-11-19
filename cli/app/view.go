package app

import (
	"github.com/skpr/compass/cli/app/components/breakdown"
	"github.com/skpr/compass/cli/app/components/help"
	"github.com/skpr/compass/cli/app/components/info"
	"github.com/skpr/compass/cli/app/components/layout"
	"github.com/skpr/compass/cli/app/components/metadata"
)

// View refreshes the display.
func (m Model) View() string {
	return layout.Model{
		TraceInfo: info.Model{
			Profiles: m.profiles,
			Selected: m.profileSelected,
		},
		RequestInfo: metadata.Model{
			Profiles: m.profiles,
			Selected: m.profileSelected,
		},
		Breakdown: breakdown.Model{
			Profiles:       m.profiles,
			Selected:       m.profileSelected,
			ScrollPosition: m.breakdownScroll,
		},
		Help:     help.Model{},
		Profiles: m.profiles,
		Selected: m.profileSelected,
	}.View()
}
