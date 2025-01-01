package app

import (
	"github.com/skpr/compass/cli/app/components/help"
	"github.com/skpr/compass/cli/app/components/info"
	"github.com/skpr/compass/cli/app/components/layout"
	"github.com/skpr/compass/cli/app/components/view/description"
	"github.com/skpr/compass/cli/app/components/view/spans"
	"github.com/skpr/compass/trace"
)

// View refreshes the display.
func (m Model) View() string {
	var trace trace.Trace

	if len(m.traces) > 0 && len(m.traces) >= m.traceSelected {
		trace = m.traces[m.traceSelected]
	}

	l := layout.Model{
		Info: info.Model{
			Trace:    trace,
			Selected: m.traceSelected,
		},
		Help:     help.Model{},
		Selected: m.traceSelected,
		Total:    len(m.traces),
	}

	switch m.viewMode {
	case ViewDescription:
		l.Spans = description.Model{
			Trace: trace,
		}
	default:
		l.Spans = spans.Model{
			Trace:          trace,
			ScrollPosition: m.breakdownScroll,
		}
	}

	return l.View()
}
