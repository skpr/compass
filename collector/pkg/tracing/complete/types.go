// Package complete implements complete tracing data.
package complete

// FunctionCall provides information about the function call.
type FunctionCall struct {
	Name      string `json:"name"`
	StartTime int64  `json:"startTime"`
	EndTime   int64  `json:"endTime"`
}

// Profile data collected for a request.
type Profile struct {
	RequestID     string         `json:"requestID"`
	ExecutionTime int64          `json:"executionTime"`
	FunctionCalls []FunctionCall `json:"functionCalls"`
}
