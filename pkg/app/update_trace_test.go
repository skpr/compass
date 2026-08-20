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

	for _, item := range m.search.Items() {
		ids = append(ids, item.(events.Trace).Metadata.ID)
	}

	return ids
}

func TestUpdateTrace_DefaultRetention(t *testing.T) {
	m := NewModel("/tmp/compass.so", 0)
	assert.Equal(t, DefaultMaxTraces, m.MaxTraces)
}
