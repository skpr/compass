package app

import (
	"fmt"
	"testing"

	"github.com/skpr/compass/trace"
	"github.com/stretchr/testify/assert"
)

func TestScrollDown(t *testing.T) {
	var functions []trace.FunctionCall

	for i := 1; i <= 35; i++ {
		functions = append(functions, trace.FunctionCall{
			Name: fmt.Sprintf("function_%d", i+1),
		})
	}

	model := &Model{
		traceSelected: 0,
		traces: Traces{
			Filtered: []trace.Trace{
				{
					Metadata: trace.Metadata{
						RequestID: "xxxxxxxxx",
					},
					FunctionCalls: functions,
				},
			},
		},
	}

	model.ScrollDown()
	assert.Equal(t, 1, model.breakdownScroll)

	model.ScrollDown()
	assert.Equal(t, 2, model.breakdownScroll)

	model.ScrollDown()
	assert.Equal(t, 3, model.breakdownScroll)

	model.ScrollDown()
	assert.Equal(t, 4, model.breakdownScroll)

	// Hit the bottom of the scroll.
	model.ScrollDown()
	assert.Equal(t, 4, model.breakdownScroll)
}
