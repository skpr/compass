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
			},
		},
	}
}

func TestUpdateTrace_NewestFirst(t *testing.T) {
	m := NewModel("/tmp/compass.so", 10)
	m.Init()

	m.updateTrace(newTrace("first"))
	m.updateTrace(newTrace("second"))

	require.Equal(t, []string{"second", "first"}, traceIDs(m))
}

func TestUpdateTrace_EvictsOldest(t *testing.T) {
	m := NewModel("/tmp/compass.so", 3)
	m.Init()

	for i := 0; i < 10; i++ {
		m.updateTrace(newTrace(fmt.Sprintf("trace-%d", i)))
	}

	// Retention is capped, and the most recent traces are the ones we kept.
	assert.Equal(t, []string{"trace-9", "trace-8", "trace-7"}, traceIDs(m))
}

// traceIDs of the retained traces, newest first.
func traceIDs(m *Model) []string {
	var ids []string

	for _, t := range m.traces {
		ids = append(ids, t.Metadata.ID)
	}

	return ids
}

// The list renders from the retained traces, so the two must not drift: an
// evicted trace has to leave the screen along with the slice.
func TestUpdateTrace_RowsFollowTheRetainedTraces(t *testing.T) {
	m := NewModel("/tmp/compass.so", 3)
	m.Init()

	for i := range 10 {
		m.updateTrace(newTrace(fmt.Sprintf("trace-%d", i)))
	}

	assert.Equal(t, len(m.traces), m.search.Len())
}

func TestUpdateTrace_DefaultRetention(t *testing.T) {
	m := NewModel("/tmp/compass.so", 0)
	assert.Equal(t, DefaultMaxTraces, m.MaxTraces)
}
