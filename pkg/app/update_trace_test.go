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
	m := NewModel("/tmp/compass.so", 10, 10)
	m.Init()

	m.updateTrace(newTrace("first"))
	m.updateTrace(newTrace("second"))

	require.Equal(t, []string{"second", "first"}, traceIDs(m))
}

func TestUpdateTrace_EvictsOldest(t *testing.T) {
	m := NewModel("/tmp/compass.so", 3, 10)
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
	m := NewModel("/tmp/compass.so", 10, 10)
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
	m := NewModel("/tmp/compass.so", 3, 10)
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
	m := NewModel("/tmp/compass.so", 0, 0)
	assert.Equal(t, DefaultMaxTraces, m.MaxTraces)
	assert.Equal(t, DefaultMaxLogs, m.MaxLogs)
	assert.Equal(t, DefaultMaxTraces, m.traces.limit())
	assert.Equal(t, DefaultMaxLogs, m.logs.limit())
}
