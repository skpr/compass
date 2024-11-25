package app

import (
	"github.com/skpr/compass/cli/app/components/breakdown"
	"github.com/skpr/compass/cli/app/components/help"
	"github.com/skpr/compass/cli/app/components/info"
	"github.com/skpr/compass/cli/app/components/layout"
)

// View refreshes the display.
func (m Model) View() string {
	return layout.Model{
		Info: info.Model{
			Traces:   m.traces,
			Selected: m.traceSelected,
		},
		Breakdown: breakdown.Model{
			Traces:         m.traces,
			Selected:       m.traceSelected,
			ScrollPosition: m.breakdownScroll,
		},
		Help:     help.Model{},
		Traces:   m.traces,
		Selected: m.traceSelected,
	}.View()
}
