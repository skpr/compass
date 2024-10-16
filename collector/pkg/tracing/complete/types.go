// Package complete implements complete tracing data.
package complete

// Profile data collected for a request.
type Profile struct {
	RequestID     string         `json:"requestID"`
	StartTime     int64          `json:"startTime"`
	ExecutionTime int64          `json:"executionTime"`
	FunctionCalls []FunctionCall `json:"functionCalls"`
}

// FunctionCall provides information about the function call.
type FunctionCall struct {
	Name      string `json:"name"`
	StartTime int64  `json:"startTime"`
	EndTime   int64  `json:"endTime"`
}
