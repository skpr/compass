package app

import (
	"github.com/skpr/compass/cli/app/components/help"
	"github.com/skpr/compass/cli/app/components/info"
	"github.com/skpr/compass/cli/app/components/layout"
	"github.com/skpr/compass/cli/app/components/report/count"
	"github.com/skpr/compass/cli/app/components/report/spans"
	"github.com/skpr/compass/cli/app/components/search"
	"github.com/skpr/compass/trace"
)

// View refreshes the display.
func (m *Model) View() string {
	var trace trace.Trace

	if len(m.traces.Filtered) > 0 && len(m.traces.Filtered) >= m.traceSelected {
		trace = m.traces.Filtered[m.traceSelected]
	}

	l := layout.Model{
		Search: search.Model{
			Query: m.searchQuery,
		},
		Info: info.Model{
			Trace:    trace,
			Selected: m.traceSelected,
		},
		Help:     help.Model{},
		Selected: m.traceSelected,
		Total:    len(m.traces.Filtered),
	}

	switch m.viewMode {
	case ViewCount:
		l.Spans = count.Model{
			Trace:          trace,
			ScrollPosition: m.breakdownScroll,
		}
	default:
		l.Spans = spans.Model{
			Trace:          trace,
			ScrollPosition: m.breakdownScroll,
		}
	}

	return l.View()
}
