package spans

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetPositionStartAndEnd(t *testing.T) {
	begin, end := getPositionStartAndEnd(0, 5, 10)
	assert.Equal(t, 0, begin)
	assert.Equal(t, 5, end)

	begin, end = getPositionStartAndEnd(1, 5, 10)
	assert.Equal(t, 1, begin)
	assert.Equal(t, 6, end)

	begin, end = getPositionStartAndEnd(7, 5, 10)
	assert.Equal(t, 5, begin)
	assert.Equal(t, 10, end)
}
