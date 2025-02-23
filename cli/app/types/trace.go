package types

import (
	"fmt"
	"time"

	"github.com/skpr/compass/cli/app/utils"
	"github.com/skpr/compass/trace"
)

// Trace for review.
type Trace struct {
	IngestionTime time.Time
	trace.Trace
}

// Title of the trace.
func (t Trace) Title() string {
	return fmt.Sprintf("%dms %s %s", utils.NanosecondsToMilliseconds(t.Metadata.ExecutionTime()), t.Metadata.Method, t.Metadata.URI)
}

// Description of the trace.
func (t Trace) Description() string {
	return fmt.Sprintf("request_id=%s, function_calls=%d, ingestion_time=%s", t.Metadata.RequestID, len(t.FunctionCalls), t.IngestionTime.Local().Format(time.RFC1123))
}

// FilterValue for search.
func (t Trace) FilterValue() string {
	return t.Title()
}
