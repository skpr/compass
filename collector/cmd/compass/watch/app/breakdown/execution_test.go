package breakdown

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetExecutionGraph(t *testing.T) {
	assert.Equal(t, "███████████████████ (289ms)", getExecutionGraph(300, 289))
	assert.Equal(t, "█ (1ms)", getExecutionGraph(300, 1))
}
