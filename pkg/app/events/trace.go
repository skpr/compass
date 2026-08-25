package events

import (
	"fmt"
	"time"

	"github.com/skpr/compass/pkg/trace"
)

// Trace for review.
//
// This carries no styling of its own. How a trace looks belongs with the page
// which shows it, where the column widths are known; keeping it here meant two
// copies of the byte formatter and a set of fixed width badges which only made
// sense on one screen.
type Trace struct {
	IngestionTime time.Time
	trace.Trace
}

// FilterValue for search.
func (t Trace) FilterValue() string {
	var identity string

	switch t.Metadata.Source {
	case trace.SourceHTTP:
		identity = fmt.Sprintf("%s %s", t.Metadata.HTTP.Method, t.Metadata.HTTP.URI)
	case trace.SourceCLI:
		identity = t.Metadata.CLI.Command
	default:
		identity = t.Metadata.ID
	}

	return fmt.Sprintf("%s %s %s", t.Metadata.Runtime, t.Metadata.Source, identity)
}
