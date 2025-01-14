package app

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSearchQueryDelete(t *testing.T) {
	model := &Model{
		searchQuery: "abc",
	}

	model.SearchQueryDelete()
	assert.Equal(t, "ab", model.searchQuery)

	model.SearchQueryDelete()
	assert.Equal(t, "a", model.searchQuery)

	model.SearchQueryDelete()
	assert.Equal(t, "", model.searchQuery)

	// Make sure there are no panics.
	model.SearchQueryDelete()
	assert.Equal(t, "", model.searchQuery)
}
