// Package trace implements complete tracing data.
package trace

import "time"

// Trace data collected for a request.
type Trace struct {
	Metadata      Metadata       `json:"metadata"`
	FunctionCalls []FunctionCall `json:"functionCalls"`
}

// Metadata associated with this trace.
type Metadata struct {
	RequestID      string        `json:"requestID"`
	URI            string        `json:"uri"`
	Method         string        `json:"method"`
	StartTime      time.Time     `json:"startTime"`
	MonotonicStart time.Duration `json:"monotonicStart"`
	MonotonicEnd   time.Duration `json:"monotonicEnd"`
}

// ExecutionTime of the trace.
func (m Metadata) ExecutionTime() time.Duration {
	return m.MonotonicEnd - m.MonotonicStart
}

// FunctionCall provides information about the function call.
type FunctionCall struct {
	Name           string        `json:"name"`
	MonotonicStart time.Duration `json:"monotonicStart"`
	MonotonicEnd   time.Duration `json:"monotonicEnd"`
	Elapsed        time.Duration `json:"elapsed"`
}
