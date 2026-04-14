package app

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUpdateKeyRight_FromSearch(t *testing.T) {
	m := &Model{PageSelected: PageSearch}
	m.updateKeyRight()
	assert.Equal(t, PageSpans, m.PageSelected)
}

func TestUpdateKeyRight_FromSpans(t *testing.T) {
	m := &Model{PageSelected: PageSpans}
	m.updateKeyRight()
	assert.Equal(t, PageTotals, m.PageSelected)
}

func TestUpdateKeyRight_FromTotals(t *testing.T) {
	m := &Model{PageSelected: PageTotals}
	m.updateKeyRight()
	assert.Equal(t, PageLogs, m.PageSelected)
}

func TestUpdateKeyRight_FromLogs(t *testing.T) {
	m := &Model{PageSelected: PageLogs}
	m.updateKeyRight()
	// Should stay on Logs (no wrap).
	assert.Equal(t, PageLogs, m.PageSelected)
}

func TestUpdateKeyLeft_FromLogs(t *testing.T) {
	m := &Model{PageSelected: PageLogs}
	m.updateKeyLeft()
	assert.Equal(t, PageTotals, m.PageSelected)
}

func TestUpdateKeyLeft_FromTotals(t *testing.T) {
	m := &Model{PageSelected: PageTotals}
	m.updateKeyLeft()
	assert.Equal(t, PageSpans, m.PageSelected)
}

func TestUpdateKeyLeft_FromSpans(t *testing.T) {
	m := &Model{PageSelected: PageSpans}
	m.updateKeyLeft()
	assert.Equal(t, PageSearch, m.PageSelected)
}

func TestUpdateKeyLeft_FromSearch(t *testing.T) {
	m := &Model{PageSelected: PageSearch}
	m.updateKeyLeft()
	// Should stay on Search (no wrap).
	assert.Equal(t, PageSearch, m.PageSelected)
}

func TestUpdateKeyEnter_NotOnSearch(t *testing.T) {
	m := &Model{PageSelected: PageSpans}
	m.updateKeyEnter()
	// Should not change page when not on Search.
	assert.Equal(t, PageSpans, m.PageSelected)
}
