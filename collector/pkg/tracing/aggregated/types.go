package aggregated

// Profile being sent to stdout.
type Profile struct {
	RequestID     string              `json:"requestID"`
	ExecutionTime int64               `json:"executionTime"`
	Functions     map[string]Function `json:"functions"`
}

// Function being called by PHP.
type Function struct {
	Name          string `json:"name"`
	ExecutionTime int64  `json:"executionTime"`
	Invocations   int32  `json:"invocations"`
}
