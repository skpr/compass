package list

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetPositionStartAndEnd(t *testing.T) {
	begin, end := getPositionStartAndEnd(0, 5, 10)
	assert.Equal(t, 0, begin)
	assert.Equal(t, 5, end)

	begin, end = getPositionStartAndEnd(1, 5, 10)
	assert.Equal(t, 0, begin)
	assert.Equal(t, 5, end)

	begin, end = getPositionStartAndEnd(7, 5, 10)
	assert.Equal(t, 3, begin)
	assert.Equal(t, 8, end)

	begin, end = getPositionStartAndEnd(10, 5, 10)
	assert.Equal(t, 5, begin)
	assert.Equal(t, 10, end)
}
