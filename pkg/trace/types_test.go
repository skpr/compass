package trace

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMetadata_ExecutionTime(t *testing.T) {
	m := Metadata{
		StartTime: 100,
		EndTime:   200,
	}
	assert.Equal(t, int64(100), m.ExecutionTime())
}

func TestMetadata_ExecutionTime_Zero(t *testing.T) {
	m := Metadata{
		StartTime: 100,
		EndTime:   100,
	}
	assert.Equal(t, int64(0), m.ExecutionTime())
}

func TestMetadata_ExecutionTime_Negative(t *testing.T) {
	m := Metadata{
		StartTime: 200,
		EndTime:   100,
	}
	assert.Equal(t, int64(-100), m.ExecutionTime())
}
