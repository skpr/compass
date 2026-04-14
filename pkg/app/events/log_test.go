package events

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestLog_Title(t *testing.T) {
	l := Log{
		Message: "something went wrong",
	}

	assert.Equal(t, "something went wrong", l.Title())
}

func TestLog_Description(t *testing.T) {
	now := time.Date(2025, 6, 15, 10, 30, 0, 0, time.UTC)
	l := Log{
		Time:    now,
		Type:    "error",
		Message: "something went wrong",
	}

	desc := l.Description()
	assert.Contains(t, desc, "type=error")
	assert.Contains(t, desc, "time=")
}

func TestLog_FilterValue(t *testing.T) {
	l := Log{
		Message: "something went wrong",
	}

	assert.Equal(t, "something went wrong", l.FilterValue())
}
