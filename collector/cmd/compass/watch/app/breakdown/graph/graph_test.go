package graph

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetExecutionGraph(t *testing.T) {
	assert.Equal(t, "│       ◼◼◼◼                                       │", Render(1726972907007464, 1726972907927130, 6054, 500), "Render should return the correct execution graph")
}
