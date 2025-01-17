package count

import (
	"github.com/skpr/compass/trace"
)

// Trace data collected for a request.
type Trace struct {
	// Metadata associated with this trace.
	Metadata trace.Metadata `json:"metadata"`
	// TotalFunctionCalls that occurred during this trace.
	TotalFunctionCalls int `json:"totalFunctionCalls"`
	// Functions that are included in trace.
	Functions []Function `json:"functions"`
}

// Function provides information about a function during a trace.
type Function struct {
	// Name of the function.
	Name string `json:"name"`
	// Calls that occurred for this function in the trace.
	Calls int `json:"calls"`
	// Percentage of the request which this function was called.
	Percentage int `json:"percentage"`
}
