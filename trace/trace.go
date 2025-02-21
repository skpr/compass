// Package trace implements complete tracing data.
package trace

// Metadata associated with this trace.
type Metadata struct {
	RequestID     string `json:"requestID"`
	URI           string `json:"uri"`
	Method        string `json:"method"`
	StartTime     int64  `json:"startTime"`
	EndTime       int64  `json:"endTime"`
	ExecutionTime int64  `json:"executionTime"`
}

// Trace data collected for a request.
type Trace struct {
	Metadata      Metadata       `json:"metadata"`
	FunctionCalls []FunctionCall `json:"functionCalls"`
}

// FunctionCall provides information about the function call.
type FunctionCall struct {
	Name      string `json:"name"`
	StartTime int64  `json:"startTime"`
	Elapsed   int64  `json:"elapsed"`
}
