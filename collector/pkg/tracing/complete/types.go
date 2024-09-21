// Package complete implements complete tracing data.
package complete

import "time"

// Profile data collected for a request.
type Profile struct {
	RequestID     string         `json:"requestID"`
	IngestedTime  time.Time      `json:"ingestedTime"`
	ExecutionTime int64          `json:"executionTime"`
	FunctionCalls []FunctionCall `json:"functionCalls"`
}

// FunctionCall provides information about the function call.
type FunctionCall struct {
	Name      string `json:"name"`
	StartTime int64  `json:"startTime"`
	EndTime   int64  `json:"endTime"`
}
