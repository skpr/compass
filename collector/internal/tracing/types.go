package tracing

// Function being called by PHP.
type Function struct {
	Name          string `json:"name"`
	ExecutionTime uint64 `json:"executionTime"`
}

// Profile data collected for a request.
type Profile struct {
	RequestID     string             `json:"requestID"`
	ExecutionTime uint64             `json:"executionTime"`
	Namespace     map[string]Summary `json:"namespace"`
	Function      map[string]Summary `json:"function"`
}

// Summary provides summarised function data.
type Summary struct {
	Invocations        uint64 `json:"invocations"`
	TotalExecutionTime uint64 `json:"totalExecutionTime"`
}
