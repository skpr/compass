// Package trace implements complete tracing data.
package trace

type Source string

var (
	// SourceHTTP is the source for traces collected from HTTP requests.
	SourceHTTP Source = "http"
	// SourceCLI is the source for traces collected from the command line interface.
	SourceCLI Source = "cli"
)

// Metadata associated with this trace.
type Metadata struct {
	Source    Source       `json:"source"`
	ID        string       `json:"id"`
	HTTP      MetadataHTTP `json:"http,omitempty"`
	CLI       MetadataCLI  `json:"cli,omitempty"`
	StartTime int64        `json:"startTime"`
	EndTime   int64        `json:"endTime"`
}

// ExecutionTime of the trace.
func (m Metadata) ExecutionTime() int64 {
	return m.EndTime - m.StartTime
}

type MetadataHTTP struct {
	Method string `json:"method,omitempty"`
	URI    string `json:"uri,omitempty"`
}

type MetadataCLI struct {
	Command string `json:"command,omitempty"`
}

// Trace data collected for a request.
type Trace struct {
	Metadata            Metadata            `json:"metadata"`
	ResourceUtilisation ResourceUtilisation `json:"resourceUtilisation"`
	FunctionCalls       []FunctionCall      `json:"functionCalls"`
}

type ResourceUtilisation struct {
	MaxMemory int64 `json:"maxMemory"`
}

// FunctionCall provides information about the function call.
type FunctionCall struct {
	Name      string `json:"name"`
	StartTime int64  `json:"startTime"`
	Elapsed   int64  `json:"elapsed"`
	Memory    int64  `json:"memory"`
}
