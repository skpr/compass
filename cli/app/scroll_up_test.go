package app

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestScrollUp(t *testing.T) {
	model := &Model{
		breakdownScroll: 3,
	}

	model.ScrollUp()
	assert.Equal(t, 2, model.breakdownScroll)

	model.ScrollUp()
	assert.Equal(t, 1, model.breakdownScroll)

	model.ScrollUp()
	assert.Equal(t, 0, model.breakdownScroll)

	model.ScrollUp()
	assert.Equal(t, 0, model.breakdownScroll)
}
