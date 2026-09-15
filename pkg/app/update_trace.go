package app

import (
	"strings"
	"unsafe"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/skpr/compass/pkg/app/events"
	"github.com/skpr/compass/pkg/trace"
)

func (m *Model) updateTrace(event events.Trace) (tea.Model, tea.Cmd) {
	m.ensureTraceHistory()

	event.Bytes = traceBytes(event.Trace)

	m.tracesBytes += event.Bytes

	if evicted, ok := m.traces.append(event); ok {
		m.tracesBytes -= evicted.Bytes
	}

	trimmed := m.enforceTraceBytes()

	// An unfiltered arrival affects exactly one visible row. The datatable's
	// bounded front insertion preserves the selected logical row and avoids a
	// full values/filter/rows rebuild on the common path.
	//
	// Evicting for weight takes rows off the far end as well, which front
	// insertion cannot express, so that case rebuilds.
	if trimmed == 0 && strings.TrimSpace(m.filterValue(PageSearch)) == "" && m.search != nil {
		m.visible = nil
		m.search.PrependRowBounded(m.traceRow(event), m.MaxTraces)
	} else if m.search != nil {
		m.searchSetRows()
	}

	return m, nil
}

// enforceTraceBytes discards the oldest traces until the retained ones are
// within the byte budget, and reports how many it discarded.
//
// The newest trace is never discarded: a single request which is larger than
// the whole budget is exactly the one somebody is looking at Compass to see,
// and showing nothing would be a worse answer than going over.
func (m *Model) enforceTraceBytes() int {
	limit := m.MaxBytes
	if limit <= 0 {
		limit = DefaultMaxBytes
		m.MaxBytes = limit
	}

	var trimmed int

	for m.tracesBytes > limit && m.traces.len() > 1 {
		evicted, ok := m.traces.removeOldest()
		if !ok {
			break
		}

		m.tracesBytes -= evicted.Bytes
		trimmed++
	}

	return trimmed
}

// Sizes of the values a retained trace is made of, for the accounting above.
// The header sizes are what a slice or string costs before its contents.
const (
	traceSize      = int(unsafe.Sizeof(events.Trace{}))
	spanSize       = int(unsafe.Sizeof(trace.Span{}))
	cacheEventSize = int(unsafe.Sizeof(trace.CacheEvent{}))
	stringSize     = int(unsafe.Sizeof(""))
)

// traceBytes is roughly what retaining a trace costs.
//
// Roughly is the point: this bounds memory rather than accounting for it, so
// it counts the parts which vary by orders of magnitude between traces — the
// spans, the cache events and the strings hanging off them — and does not try
// to model what the allocator did with them.
func traceBytes(t trace.Trace) int {
	bytes := traceSize +
		len(t.Metadata.ID) +
		len(t.Metadata.HTTP.Method) +
		len(t.Metadata.HTTP.URI) +
		len(t.Metadata.CLI.Command)

	for _, span := range t.Spans {
		bytes += spanSize + len(span.Name)
	}

	if t.Drupal != nil {
		for _, event := range t.Drupal.CacheEvents {
			bytes += cacheEventSize + len(event.Caller) + len(event.ObjectType)

			for _, tag := range event.Tags {
				bytes += stringSize + len(tag)
			}

			for _, context := range event.Contexts {
				bytes += stringSize + len(context)
			}
		}
	}

	return bytes
}

func (m *Model) ensureTraceHistory() {
	limit := m.MaxTraces
	if limit <= 0 {
		limit = DefaultMaxTraces
		m.MaxTraces = limit
	}

	if m.traces.limit() == limit {
		return
	}

	// Resizing can drop traces, so what is retained has to be weighed again.
	m.traces.setLimit(limit)
	m.recountTraceBytes()
}

// recountTraceBytes from the retained traces. Only for the paths which change
// what is retained without going through an eviction.
func (m *Model) recountTraceBytes() {
	m.tracesBytes = 0

	for i := range m.traces.len() {
		event, ok := m.traces.oldest(i)
		if !ok {
			break
		}

		m.tracesBytes += event.Bytes
	}
}
