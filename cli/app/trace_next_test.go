package app

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/skpr/compass/trace"
)

func TestTraceNext(t *testing.T) {
	model := &Model{
		traces: Traces{
			Filtered: []trace.Trace{
				{
					Metadata: trace.Metadata{
						RequestID: "111222333",
					},
				},
				{
					Metadata: trace.Metadata{
						RequestID: "xxxyyyzzz",
					},
				},
				{
					Metadata: trace.Metadata{
						RequestID: "xxxyyyzzz",
					},
				},
			},
		},
	}

	model.TraceNext()
	assert.Equal(t, 1, model.traceSelected)

	model.TraceNext()
	assert.Equal(t, 2, model.traceSelected)

	model.TraceNext()
	assert.Equal(t, 2, model.traceSelected)
}
