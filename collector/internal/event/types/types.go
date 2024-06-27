package types

type Trace struct {
	ID                 string     `json:"requestID"`
	TotalExecutionTime uint64     `json:"totalExecutionTime"`
	Functions          []Function `json:"functions"`
}

type Function struct {
	Name          string `json:"name"`
	ExecutionTime uint64 `json:"executionTime"`
}
