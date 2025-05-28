// Package span for rendering a trace function call as a span.
package span

import (
	"fmt"
	"strings"
	"time"

	"github.com/jwalton/gchalk"
)

// Block to visualise a span.
const Block = "■"

var (
	// BlockRed is used to identify key spans.
	red = gchalk.WithHex("#ff6058")
	// BlockOrange is used to identify important spans.
	orange = gchalk.WithHex("#ee5622")
	// BlockYellow is used to identify informational spans.
	yellow = gchalk.WithHex("#f8df3d")
)

// Component for rendering spans.
type Component struct {
	// TotalDuration of the request in MS.
	Duration time.Duration
	// TotalBlocks of the component (number of characters).
	Blocks float64
}

// New component for rendering spans.
func New(duration time.Duration, blocks float64) *Component {
	return &Component{
		Duration: duration,
		Blocks:   blocks,
	}
}

// Span which will be rendered.
type Span struct {
	// When the span starts.
	Start time.Duration
	// Duration of the span in MS.
	Duration time.Duration
}

// Render a span.
func (c *Component) Render(span Span) string {
	var (
		pre  = toPositiveInt(FractionDuration(span.Start, c.Duration) * c.Blocks)
		fill = toPositiveInt(FractionDuration(span.Duration, c.Duration) * c.Blocks)
		post = toPositiveInt(FractionDuration(c.Duration-(span.Start+span.Duration), c.Duration) * c.Blocks)
	)

	fill = tidyFill(pre+fill+post, int(c.Blocks), fill)

	// return fmt.Sprintf("｜%s%s%s｜ %dms", strings.Repeat(" ", pre), strings.Repeat(getBlockWithColor(FractionDuration(span.Duration, c.Duration)), fill), strings.Repeat(" ", post), span.Duration.Milliseconds())
	return fmt.Sprintf("｜%s%s%s｜ %dms", strings.Repeat(" ", pre), colorForFill(FractionDuration(span.Duration, c.Duration), strings.Repeat(Block, fill)), strings.Repeat(" ", post), span.Duration.Milliseconds())
}

// Returns a block which is colorised based on the percentage of the spans.
func colorForFill(val float64, fill string) string {
	if val > 0.75 {
		return red.Paint(fill)
	}

	if val > 0.5 {
		return orange.Paint(fill)
	}

	if val > 0.15 {
		return yellow.Paint(fill)
	}

	return fill
}

// Tidy up the fill of the span eg. Make sure we don't have any white space
// at the end of the span component.
func tidyFill(total, want, fill int) int {
	if total > want {
		return fill - (total - want)
	}

	if want > total {
		return fill + (want - total)
	}

	return fill
}

// FractionDuration calculates what fraction 'part' is of 'total'.
// Returns a float64 (e.g., 0.5 for 50%).
func FractionDuration(part, total time.Duration) float64 {
	if total == 0 {
		return 0.0 // Avoid division by zero
	}

	return float64(part) / float64(total)
}
