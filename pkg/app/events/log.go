// Package events which occur in this application.
package events

import (
	"time"
)

// Log event which occurred during execution.
type Log struct {
	Time    time.Time
	Type    string
	Message string
}

// FilterValue for searching.
func (l Log) FilterValue() string {
	return l.Message
}
