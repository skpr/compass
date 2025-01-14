package app

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestReportPrevious(t *testing.T) {
	model := &Model{}
	model.ReportPrevious()
	assert.Equal(t, model.viewMode, ViewSpans)
}
