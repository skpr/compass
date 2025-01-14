package app

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/skpr/compass/trace"
)

func TestTracePrevious(t *testing.T) {
	model := &Model{
		traceSelected: 2,
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

	model.TracePrevious()
	assert.Equal(t, 1, model.traceSelected)

	model.TracePrevious()
	assert.Equal(t, 0, model.traceSelected)

	model.TracePrevious()
	assert.Equal(t, 0, model.traceSelected)
}
