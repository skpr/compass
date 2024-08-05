package tracing

// Function being called by PHP.
type Function struct {
	Name          string `json:"name"`
	ExecutionTime uint64 `json:"executionTime"`
}

// Profile data collected for a request.
type Profile struct {
	RequestID     string                     `json:"requestID"`
	ExecutionTime uint64                     `json:"executionTime"`
	Functions     map[string]FunctionSummary `json:"functions"`
}

// FunctionSummary provides summaries function data.
type FunctionSummary struct {
	TotalExecutionTime uint64 `json:"totalExecutionTime"`
	Invocations        uint64 `json:"invocations"`
}
