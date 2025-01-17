package app

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
)

func TestSearchQueryAdd(t *testing.T) {
	model := &Model{}

	model.SearchQueryAdd(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	model.SearchQueryAdd(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'b'}})
	model.SearchQueryAdd(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}})

	assert.Equal(t, "abc", model.searchQuery)
}
