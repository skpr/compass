package breakdown

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetExecutionGraph(t *testing.T) {
	assert.Equal(t, "██████████ (300ms)", getExecutionGraph(300, 289))
}
