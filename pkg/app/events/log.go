// Package events which occur in this application.
package events

import (
	"fmt"
	"time"
)

// Log event which occurred during execution.
type Log struct {
	Time    time.Time
	Type    string
	Message string
}

// Title of the log message.
func (l Log) Title() string {
	return l.Message
}

// Description of the log message.
func (l Log) Description() string {
	return fmt.Sprintf("type=%s time=%s", l.Type, l.Time.Local().Format(time.RFC1123))
}

// FilterValue for searching.
func (l Log) FilterValue() string {
	return l.Message
}
