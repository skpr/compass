package types

type Requests map[string]Request

type Request struct {
	ID        string    `json:"id"`
	Functions Functions `json:"functions"`
}

type Functions map[string]Function

type Function struct {
	Name          string `json:"name"`
	ExecutionTime uint64 `json:"executionTime"`
	Invocations   uint64 `json:"invocations"`
}
