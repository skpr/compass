package types

type Function struct {
	Name          string `json:"name"`
	ExecutionTime uint64 `json:"executionTime"`
}

type Profile struct {
	RequestID string                     `json:"requestID"`
	Functions map[string]FunctionSummary `json:"functions"`
}

type FunctionSummary struct {
	TotalExecutionTime uint64 `json:"totalExecutionTime"`
	Invocations        uint64 `json:"invocations"`
}
