package segmented

import (
	"fmt"

	"github.com/skpr/compass/trace"
)

// Trace data collected for a request.
type Trace struct {
	// Metadata associated with this trace.
	Metadata trace.Metadata `json:"metadata"`
	// Total number of segments in this trace.
	Segments int `json:"segments"`
	// TotalFunctionCalls that occurred during this trace.
	TotalFunctionCalls int `json:"totalFunctionCalls"`
	// Spans that are included in trace.
	Spans []Span `json:"spans"`
}

// Span provides information about a function call during a .
type Span struct {
	// Name of the function.
	Name string `json:"name"`
	// The original start time of the function called in the span.
	StartTime int64 `json:"startTime"`
	// Which segment this function started.
	Start int `json:"start"`
	// How many segments this function call spans.
	Length int `json:"length"`
	// TotalFunctionCalls that were called during this span.
	TotalFunctionCalls int `json:"calls"`
}

// GetName of the span and include the amount when more than one call.
func (s Span) GetName() string {
	if s.TotalFunctionCalls > 1 {
		return fmt.Sprintf("%s (%d)", s.Name, s.TotalFunctionCalls)
	}

	return s.Name
}
