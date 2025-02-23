// Package types for sharing application types between packages.
package types

import (
	"fmt"
	"time"
)

// Log event which occurred during execution.
type Log struct {
	Time      time.Time
	Component string
	Message   string
}

// Title of the log message.
func (l Log) Title() string {
	return l.Message
}

// Description of the log message.
func (l Log) Description() string {
	return fmt.Sprintf("time=%s, component=%s", l.Time.Local().Format(time.RFC1123), l.Component)
}

// FilterValue for searching.
func (l Log) FilterValue() string {
	return l.Message
}
