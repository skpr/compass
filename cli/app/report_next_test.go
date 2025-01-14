package app

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestReportNext(t *testing.T) {
	model := &Model{}
	model.ReportNext()
	assert.Equal(t, model.viewMode, ViewCount)
}
