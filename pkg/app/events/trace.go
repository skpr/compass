package events

import (
	"fmt"
	"time"

	skprtime "github.com/skpr/compass/pkg/app/time"
	"github.com/skpr/compass/pkg/trace"
)

// Trace for review.
type Trace struct {
	IngestionTime time.Time
	trace.Trace
}

// Title of the trace.
func (t Trace) Title() string {
	switch t.Metadata.Source {
	case trace.SourceHTTP:
		return fmt.Sprintf("%dms (%s) %s %s", skprtime.NanosecondsToMilliseconds(t.Metadata.ExecutionTime()), formatBytes(t.ResourceUtilisation.MaxMemory), t.Metadata.HTTP.Method, t.Metadata.HTTP.URI)
	case trace.SourceCLI:
		return fmt.Sprintf("%dms (%s) %s", skprtime.NanosecondsToMilliseconds(t.Metadata.ExecutionTime()), formatBytes(t.ResourceUtilisation.MaxMemory), t.Metadata.CLI.Command)
	default:
		return fmt.Sprintf("%dms (%s) UNKNOWN", skprtime.NanosecondsToMilliseconds(t.Metadata.ExecutionTime()), formatBytes(t.ResourceUtilisation.MaxMemory))
	}
}

// Description of the trace.
func (t Trace) Description() string {
	return fmt.Sprintf("id=%s, source=%s, function_calls=%d, ingestion_time=%s", t.Metadata.ID, t.Metadata.Source, len(t.FunctionCalls), t.IngestionTime.Local().Format(time.RFC1123))
}

// FilterValue for search.
func (t Trace) FilterValue() string {
	return t.Title()
}

// Returns the bytes in a human readable format.
func formatBytes(b int64) string {
	const unit = 1024

	if b < unit {
		return fmt.Sprintf("%d B", b)
	}

	div, exp := int64(unit), 0

	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}

	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}
