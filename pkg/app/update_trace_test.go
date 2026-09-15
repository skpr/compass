package app

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/skpr/compass/pkg/app/events"
	"github.com/skpr/compass/pkg/trace"
)

func newTrace(id string) events.Trace {
	return events.Trace{
		Trace: trace.Trace{
			Metadata: trace.Metadata{
				ID:     id,
				Source: trace.SourceHTTP,
				HTTP:   trace.MetadataHTTP{Method: "GET", URI: "/" + id},
			},
		},
	}
}

func TestUpdateTrace_NewestFirst(t *testing.T) {
	m := NewModel("/tmp/compass.so", Options{MaxTraces: 10, MaxLogs: 10})
	m.Init()

	m.updateTrace(newTrace("first"))
	m.updateTrace(newTrace("second"))

	require.Equal(t, []string{"second", "first"}, traceIDs(m))
}

func TestUpdateTrace_EvictsOldest(t *testing.T) {
	m := NewModel("/tmp/compass.so", Options{MaxTraces: 3, MaxLogs: 10})
	m.Init()

	for i := 0; i < 10; i++ {
		m.updateTrace(newTrace(fmt.Sprintf("trace-%d", i)))
	}

	assert.Equal(t, []string{"trace-9", "trace-8", "trace-7"}, traceIDs(m))
	assert.Equal(t, 3, m.traces.len())
	assert.Equal(t, 3, m.search.Len())
}

// traceIDs of the retained traces, newest first.
func traceIDs(m *Model) []string {
	ids := make([]string, 0, m.traces.len())

	for i := 0; i < m.traces.len(); i++ {
		event, _ := m.traces.newest(i)
		ids = append(ids, event.Metadata.ID)
	}

	return ids
}

// A live arrival should not replace the row being read merely because every
// newer row moved down by one.
func TestUpdateTrace_PreservesSelectedLogicalRow(t *testing.T) {
	m := NewModel("/tmp/compass.so", Options{MaxTraces: 10, MaxLogs: 10})
	m.Init()

	for _, id := range []string{"first", "second", "third"} {
		m.updateTrace(newTrace(id))
	}
	m.search.SetCursor(1)

	before, ok := m.selectedTrace()
	require.True(t, ok)
	require.Equal(t, "second", before.Metadata.ID)

	m.updateTrace(newTrace("fourth"))

	after, ok := m.selectedTrace()
	require.True(t, ok)
	assert.Equal(t, "second", after.Metadata.ID)
	assert.Equal(t, 2, m.search.Cursor())
}

func TestUpdateTrace_FilteringAndEvictionUseRetainedHistory(t *testing.T) {
	m := NewModel("/tmp/compass.so", Options{MaxTraces: 3, MaxLogs: 10})
	m.Init()

	for _, id := range []string{"keep-old", "other", "keep-new", "latest"} {
		m.updateTrace(newTrace(id))
	}

	m.filter.SetValue("keep")
	m.searchSetRows()

	require.Equal(t, 1, m.search.Len(), "the evicted keep-old trace must not match")
	m.search.SetCursor(0)
	selected, ok := m.selectedTrace()
	require.True(t, ok)
	assert.Equal(t, "keep-new", selected.Metadata.ID)
}

func TestUpdateTrace_DefaultRetention(t *testing.T) {
	m := NewModel("/tmp/compass.so", Options{MaxTraces: 0, MaxLogs: 0})
	assert.Equal(t, DefaultMaxTraces, m.MaxTraces)
	assert.Equal(t, DefaultMaxLogs, m.MaxLogs)
	assert.Equal(t, DefaultMaxTraces, m.traces.limit())
	assert.Equal(t, DefaultMaxLogs, m.logs.limit())
}

// traceOfSpans is a trace whose weight is dominated by its spans.
func traceOfSpans(id string, spans int) events.Trace {
	event := newTrace(id)

	event.Spans = make([]trace.Span, 0, spans)
	for i := range spans {
		event.Spans = append(event.Spans, trace.Span{
			Name:  fmt.Sprintf("Drupal\\Core\\Entity\\Sql\\SqlContentEntityStorage%d::loadMultiple", i),
			Calls: 1,
		})
	}

	event.Calls = int64(spans)

	return event
}

// A count of traces does not bound memory: a request making a million calls
// produces a trace orders of magnitude larger than a request making ten, so
// the history is bounded by what it weighs as well.
func TestUpdateTrace_EvictsForWeight(t *testing.T) {
	m := NewModel("/tmp/compass.so", Options{MaxTraces: 100, MaxLogs: 10})
	m.Init()

	// Room for about two of these.
	m.MaxBytes = 2 * traceBytes(traceOfSpans("sized", 500).Trace)

	for i := range 10 {
		m.updateTrace(traceOfSpans(fmt.Sprintf("trace-%d", i), 500))
	}

	assert.LessOrEqual(t, m.tracesBytes, m.MaxBytes)
	assert.Less(t, m.traces.len(), 10, "the byte budget evicted nothing")
	assert.Positive(t, m.traces.len())

	// The newest are the ones kept, and the rows follow them.
	newest, ok := m.traces.newest(0)
	require.True(t, ok)
	assert.Equal(t, "trace-9", newest.Metadata.ID)
	assert.Equal(t, m.traces.len(), m.search.Len())
}

// A single request larger than the whole budget is exactly the one somebody
// opened Compass to look at.
func TestUpdateTrace_KeepsTheNewestHoweverLarge(t *testing.T) {
	m := NewModel("/tmp/compass.so", Options{MaxTraces: 100, MaxLogs: 10})
	m.Init()
	m.MaxBytes = 1

	m.updateTrace(traceOfSpans("small", 1))
	m.updateTrace(traceOfSpans("enormous", 10_000))

	require.Equal(t, 1, m.traces.len())

	kept, ok := m.traces.newest(0)
	require.True(t, ok)
	assert.Equal(t, "enormous", kept.Metadata.ID)
	assert.Equal(t, 1, m.search.Len())
}

// The weight of what is retained has to track what arrives and leaves, or the
// budget drifts away from the traces it is meant to bound.
func TestUpdateTrace_WeightTracksTheRetainedTraces(t *testing.T) {
	m := NewModel("/tmp/compass.so", Options{MaxTraces: 3, MaxLogs: 10})
	m.Init()

	for i := range 10 {
		m.updateTrace(traceOfSpans(fmt.Sprintf("trace-%d", i), 20))
	}

	var weight int
	for i := range m.traces.len() {
		event, ok := m.traces.oldest(i)
		require.True(t, ok)
		weight += event.Bytes
	}

	assert.Equal(t, weight, m.tracesBytes)
}
