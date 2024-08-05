package main

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/skpr/compass/collector/internal/tracing"
)

func TestReduce(t *testing.T) {
	profile := reduce(tracing.Profile{
		Functions: map[string]tracing.FunctionSummary{
			"Foo": {
				TotalExecutionTime: 99,
			},
			"Bar": {
				TotalExecutionTime: 101,
			},
		},
	}, 100)

	want := tracing.Profile{
		Functions: map[string]tracing.FunctionSummary{
			"Bar": {
				TotalExecutionTime: 101,
			},
		},
	}

	assert.Equal(t, want, profile)
}
