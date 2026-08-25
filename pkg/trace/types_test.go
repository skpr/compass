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

func TestMetadata_Identified(t *testing.T) {
	assert.True(t, Metadata{ID: "58bb2c6e56c13ce04c1cb9a87083d735"}.Identified())
	assert.True(t, Metadata{ID: "61642"}.Identified(), "a CLI trace is identified by its pid")

	// The extension sends this when the request had no X-Request-ID header, so
	// it identifies nothing and two traces carrying it are not the same request.
	assert.False(t, Metadata{ID: IDUnknown}.Identified())
	assert.False(t, Metadata{}.Identified())
}
