package app

import (
	"github.com/skpr/compass/collector/cmd/compass/watch/app/breakdown"
	"github.com/skpr/compass/collector/cmd/compass/watch/app/help"
	"github.com/skpr/compass/collector/cmd/compass/watch/app/info"
	"github.com/skpr/compass/collector/cmd/compass/watch/app/layout"
)

// View refreshes the display.
func (m Model) View() string {
	return layout.Model{
		Info: info.Model{
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
