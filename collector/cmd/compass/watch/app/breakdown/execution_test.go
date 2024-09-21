package breakdown

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetExecutionGraph(t *testing.T) {
	assert.Equal(t, "███████████████████ (289ms)", getExecutionGraph(300.123, 289.123123))
	assert.Equal(t, "█ (1ms)", getExecutionGraph(300.123, 1))
}
