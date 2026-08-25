// Package span renders a function call as a bar on a request's timeline.
//
// A bar carries three independent facts, in three independent channels: where
// in the request the call happened (position), how long it ran for (length),
// and how much of the request it was itself responsible for (colour). Keeping
// them separate is the point — the previous version coloured by length, which
// meant every frame that merely wrapped the request rendered at maximum
// severity while the function actually burning the time rendered cool.
package span

import (
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"

	"github.com/skpr/compass/pkg/app/theme"
)

// Component renders bars against one request's wall clock.
type Component struct {
	// Duration of the whole request, which the timeline spans.
	Duration time.Duration
	// Blocks is how many cells wide the timeline is.
	Blocks int
}

// New component for rendering spans across a number of cells.
func New(duration time.Duration, blocks int) *Component {
	if blocks < 1 {
		blocks = 1
	}

	return &Component{
		Duration: duration,
		Blocks:   blocks,
	}
}

// Span to be rendered.
type Span struct {
	// Start of the call, relative to the start of the request.
	Start time.Duration
	// Duration the call ran for.
	Duration time.Duration
	// Share of the request the call is itself responsible for, from zero to
	// one. This drives the colour, and it is deliberately not derived from
	// Duration: a call which delegates all of its time to a child is long but
	// not hot.
	Share float64
}

// Bar is a span's parts, kept apart rather than rendered into one string.
//
// A caller which puts a bar in a table needs the parts: composing a selected
// background behind a pre-rendered bar does not work, because the bar's own
// reset sequences close the background along with the colour, and the row loses
// its highlight from the bar onwards.
type Bar struct {
	// Lead is the track before the span starts.
	Lead string
	// Fill is the span itself.
	Fill string
	// Trail is the track after the span ends.
	Trail string
	// Share the fill should be coloured by.
	Share float64
}

// Bar of a span, as its parts.
//
// The geometry is in whole cells rather than in fractions of one, and the
// minimum is enforced after the rounding rather than before it. Enforcing it
// first is not enough: a span shorter than half a cell rounds its start and its
// end to the same cell, and the fill disappears no matter how large the
// fraction it was given. A call which ran at all leaves a mark.
func (c *Component) Bar(s Span) Bar {
	from := c.cell(s.Start)
	to := c.cell(s.Start + s.Duration)

	if to <= from {
		to = from + 1
	}

	// Kept inside the timeline, sliding the whole bar left rather than
	// truncating it, so a call at the very end of a request still shows.
	if to > c.Blocks {
		to = c.Blocks
		from = min(from, c.Blocks-1)
	}

	return Bar{
		Lead:  run(theme.RuleLight, from),
		Fill:  run(theme.BarFull, to-from),
		Trail: run(theme.RuleLight, c.Blocks-to),
		Share: s.Share,
	}
}

// cell of the timeline an offset into the request falls in.
func (c *Component) cell(at time.Duration) int {
	cell := int(FractionDuration(at, c.Duration) * float64(c.Blocks))

	return min(max(cell, 0), max(c.Blocks-1, 0))
}

// run of a glyph, a number of cells long.
//
// The filled and unfilled parts use different glyphs, not just different
// colours. Block over block leaves hue as the only difference between the bar
// and the groove it sits in, which reads as one solid slab — and falls apart
// completely on a terminal with no colour to tell them apart with.
func run(glyph string, cells int) string {
	if cells <= 0 {
		return ""
	}

	return strings.Repeat(glyph, cells)
}

// Render a span as a bar of exactly Blocks cells.
//
// The bar is drawn over a track rather than over blank space, so that the
// extent of the request is visible even where nothing is running, and so a
// short call still reads as a position on a scale rather than as a fleck.
func (c *Component) Render(s Span) string {
	bar := c.Bar(s)

	var (
		fill  = theme.S.Ramp(s.Share)
		track = theme.S.Track
	)

	var b strings.Builder

	// Each run is only styled when it has something in it: styling an empty
	// string still emits the escape sequence and its reset, which is bytes down
	// the wire for nothing and noise in a golden file.
	//nolint:unparam // the signature mirrors the segments callers build.
	paint := func(style lipgloss.Style, run string) {
		if run == "" {
			return
		}

		b.WriteString(style.Render(run))
	}

	paint(track, bar.Lead)
	paint(fill, bar.Fill)
	paint(track, bar.Trail)

	return b.String()
}

// Axis renders the scale the bars are drawn against.
//
// Without it a timeline shows that one call happened before another but not
// where either sits in the request, which is most of what the reader wants
// from it.
func (c *Component) Axis() string {
	if c.Blocks < 8 {
		return strings.Repeat(" ", c.Blocks)
	}

	labels := []struct {
		at   float64
		text string
	}{
		{at: 0, text: "0%"},
		{at: 0.25, text: "25%"},
		{at: 0.5, text: "50%"},
		{at: 0.75, text: "75%"},
		{at: 1, text: "100%"},
	}

	axis := []rune(strings.Repeat(" ", c.Blocks))

	for _, label := range labels {
		at := int(label.at * float64(c.Blocks-1))

		// The last label is right aligned so it does not run off the end.
		if label.at == 1 {
			at = c.Blocks - len([]rune(label.text))
		}

		for i, r := range label.text {
			if at+i < len(axis) {
				axis[at+i] = r
			}
		}
	}

	return string(axis)
}

// FractionDuration calculates what fraction 'part' is of 'total'.
// Returns a float64 (e.g., 0.5 for 50%).
func FractionDuration(part, total time.Duration) float64 {
	if total == 0 {
		return 0.0 // Avoid division by zero
	}

	return float64(part) / float64(total)
}
