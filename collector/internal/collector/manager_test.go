package collector

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/skpr/compass/collector/pkg/tracing"
)

func TestReduce(t *testing.T) {
	profile := reduceFunctions(map[string]tracing.FunctionSummary{
		"Foo": {
			TotalExecutionTime: 99,
		},
		"Bar": {
			TotalExecutionTime: 101,
		},
	}, 100)

	want := map[string]tracing.FunctionSummary{
		"Bar": {
			TotalExecutionTime: 101,
		},
	}

	assert.Equal(t, want, profile)
}
