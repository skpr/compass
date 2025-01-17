package graph

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetExecutionGraph(t *testing.T) {
	assert.Equal(t, "       ◼◼◼◼", Render(7, 4), "Render should return the correct execution graph")
}
