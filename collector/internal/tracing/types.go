package tracing

// Function being called by PHP.
type Function struct {
	Name          string  `json:"name"`
	ExecutionTime float64 `json:"executionTime"`
}

// Profile data collected for a request.
type Profile struct {
	RequestID     string                     `json:"requestID"`
	ExecutionTime float64                    `json:"executionTime"`
	Functions     map[string]FunctionSummary `json:"functions"`
}

// FunctionSummary provides summaries function data.
type FunctionSummary struct {
	TotalExecutionTime float64 `json:"totalExecutionTime"`
	Invocations        int32   `json:"invocations"`
}
