package span

import (
	"os"
	"testing"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/muesli/termenv"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/skpr/compass/pkg/app/theme"
)

func TestMain(m *testing.M) {
	lipgloss.SetColorProfile(termenv.TrueColor)
	lipgloss.SetHasDarkBackground(true)

	os.Exit(m.Run())
}

func TestFractionDuration(t *testing.T) {
	assert.InDelta(t, 0.5, FractionDuration(50*time.Millisecond, 100*time.Millisecond), 0.001)
	assert.InDelta(t, 1.0, FractionDuration(100*time.Millisecond, 100*time.Millisecond), 0.001)
	assert.InDelta(t, 0.0, FractionDuration(0, 100*time.Millisecond), 0.001)
	// A trace with no measurable duration must not divide by zero.
	assert.InDelta(t, 0.0, FractionDuration(50*time.Millisecond, 0), 0.001)
}

// Every bar has to be exactly the width the layout budgeted for it, whatever
// it contains. This is the invariant the whole timeline column rests on.
func TestComponent_Render_IsAlwaysExactlyBlocksWide(t *testing.T) {
	const blocks = 50

	c := New(100*time.Millisecond, blocks)

	tests := []struct {
		name string
		span Span
	}{
		{name: "zero", span: Span{}},
		{name: "whole request", span: Span{Duration: 100 * time.Millisecond}},
		{name: "in the middle", span: Span{Start: 40 * time.Millisecond, Duration: 20 * time.Millisecond}},
		{name: "at the very end", span: Span{Start: 99 * time.Millisecond, Duration: time.Millisecond}},
		{name: "sub cell", span: Span{Start: 10 * time.Millisecond, Duration: time.Microsecond}},
		{name: "starts past the end", span: Span{Start: 200 * time.Millisecond, Duration: time.Millisecond}},
		{name: "longer than the request", span: Span{Duration: 500 * time.Millisecond}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, blocks, ansi.StringWidth(c.Render(tt.span)))
		})
	}
}

func TestComponent_Render_ZeroWidthTimeline(t *testing.T) {
	c := New(100*time.Millisecond, 0)

	// New clamps to at least one cell rather than producing an empty bar.
	assert.Equal(t, 1, ansi.StringWidth(c.Render(Span{Duration: 50 * time.Millisecond})))
}

func TestComponent_Render_ZeroDurationTrace(t *testing.T) {
	c := New(0, 20)

	assert.Equal(t, 20, ansi.StringWidth(c.Render(Span{Duration: 5 * time.Millisecond})))
}

// A call too short to fill a cell still has to appear, or everything at the
// resolution limit silently disappears from the timeline.
func TestComponent_Render_ShortSpanStillDraws(t *testing.T) {
	c := New(time.Second, 20)

	rendered := ansi.Strip(c.Render(Span{Start: 500 * time.Millisecond, Duration: time.Microsecond}))

	require.Equal(t, 20, len([]rune(rendered)))
}

// Position is a channel of its own: two calls of the same length at different
// points in the request must not render identically.
func TestComponent_Render_PositionIsVisible(t *testing.T) {
	c := New(100*time.Millisecond, 40)

	early := c.Render(Span{Start: 0, Duration: 10 * time.Millisecond})
	late := c.Render(Span{Start: 80 * time.Millisecond, Duration: 10 * time.Millisecond})

	assert.NotEqual(t, early, late)
}

// Colour follows elapsed duration as a share of the whole request, just like
// the percentage and gutter weight on the Functions page.
func TestComponent_Render_ColourFollowsDuration(t *testing.T) {
	c := New(100*time.Millisecond, 40)

	short := c.Render(Span{Start: 0, Duration: 10 * time.Millisecond})
	long := c.Render(Span{Start: 0, Duration: 90 * time.Millisecond})

	assert.NotEqual(t, colourOf(t, short), colourOf(t, long))

	// Position is a separate fact: equal durations at opposite ends of the
	// request use the same colour.
	late := c.Render(Span{Start: 80 * time.Millisecond, Duration: 10 * time.Millisecond})
	assert.Equal(t, colourOf(t, short), colourOf(t, late))
}

func TestComponent_Axis_IsExactlyBlocksWide(t *testing.T) {
	for _, blocks := range []int{1, 4, 8, 20, 44, 120} {
		assert.Equal(t, blocks, ansi.StringWidth(New(time.Second, blocks).Axis()), "blocks=%d", blocks)
	}
}

func TestComponent_Axis_Labelled(t *testing.T) {
	axis := New(time.Second, 44).Axis()

	assert.Contains(t, axis, "0%")
	assert.Contains(t, axis, "50%")
	assert.Contains(t, axis, "100%")
}

// Too narrow to label, so it renders blank rather than a scale that lies.
func TestComponent_Axis_TooNarrowToLabel(t *testing.T) {
	assert.Equal(t, "    ", New(time.Second, 4).Axis())
}

// colourOf returns the colour the fill of a bar is drawn in.
//
// A bar which does not start at the left edge opens with the track colour, so
// picking the first escape sequence would compare the groove rather than the
// bar sitting in it.
func colourOf(t *testing.T, rendered string) string {
	t.Helper()

	track := escapes(t, theme.S.Track.Render("x"))
	require.NotEmpty(t, track)

	for _, sequence := range escapes(t, rendered) {
		if sequence == track[0] || sequence == "\x1b[0m" {
			continue
		}

		return sequence
	}

	t.Fatalf("no fill colour in %q", rendered)

	return ""
}

// escapes returns every escape sequence in a string, in order.
func escapes(t *testing.T, s string) []string {
	t.Helper()

	var found []string

	for i := 0; i < len(s); i++ {
		if s[i] != 0x1b {
			continue
		}

		end := i
		for end < len(s) && s[end] != 'm' {
			end++
		}

		require.Less(t, end, len(s), "unterminated escape sequence in %q", s)

		found = append(found, s[i:end+1])
		i = end
	}

	return found
}

// The filled and unfilled parts of a bar are told apart by their glyph, not
// only by their colour.
//
// Drawn as blocks over blocks the two differ by hue alone, which reads as one
// solid slab even in full colour and carries nothing at all without it. A bar
// sitting on a rail reads as a bar at any colour depth.
func TestComponent_Bar_FillAndTrackUseDifferentGlyphs(t *testing.T) {
	c := New(100*time.Millisecond, 40)

	bar := c.Bar(Span{Start: 30 * time.Millisecond, Duration: 20 * time.Millisecond})

	require.NotEmpty(t, bar.Lead)
	require.NotEmpty(t, bar.Fill)
	require.NotEmpty(t, bar.Trail)

	assert.NotContains(t, bar.Lead, theme.BarFull, "the rail must not be drawn with the fill glyph")
	assert.NotContains(t, bar.Trail, theme.BarFull)
	assert.NotContains(t, bar.Fill, theme.RuleLight)

	// And with no styling at all, the bar is still visible in the rail.
	plain := bar.Lead + bar.Fill + bar.Trail
	assert.Contains(t, plain, theme.RuleLight+theme.BarFull)
}

// A call which ran at all leaves a mark, wherever in the request it happened.
//
// This is the case the previous geometry got wrong. The minimum was enforced in
// eighths of a cell and the rounding to whole cells came afterwards, so a span
// shorter than half a cell rounded its start and its end into the same cell and
// rendered as unbroken rail — a row which looked like nothing had run.
func TestComponent_Bar_ShortSpanAlwaysHasFill(t *testing.T) {
	const (
		blocks  = 44
		request = 7442 * time.Millisecond
	)

	c := New(request, blocks)

	// Across the whole request, because the failure depended on where the span
	// fell relative to a cell boundary rather than on how short it was.
	for offset := time.Duration(0); offset < request; offset += 13 * time.Millisecond {
		for _, duration := range []time.Duration{
			time.Nanosecond, time.Microsecond, time.Millisecond, 50 * time.Millisecond,
		} {
			bar := c.Bar(Span{Start: offset, Duration: duration})

			assert.NotEmpty(t, bar.Fill,
				"no fill for a %s span at %s", duration, offset)
			assert.Equal(t, blocks, len([]rune(bar.Lead+bar.Fill+bar.Trail)),
				"wrong width for a %s span at %s", duration, offset)
		}
	}
}

// A call at the very end of the request slides inside the timeline rather than
// being truncated away.
func TestComponent_Bar_AtTheEndOfTheRequest(t *testing.T) {
	c := New(time.Second, 20)

	bar := c.Bar(Span{Start: 999 * time.Millisecond, Duration: time.Millisecond})

	assert.NotEmpty(t, bar.Fill)
	assert.Empty(t, bar.Trail, "a span at the end should sit against the right edge")
	assert.Equal(t, 20, len([]rune(bar.Lead+bar.Fill+bar.Trail)))
}

// A span with no duration at all is still a thing which happened.
func TestComponent_Bar_ZeroDuration(t *testing.T) {
	c := New(time.Second, 20)

	bar := c.Bar(Span{Start: 500 * time.Millisecond})

	assert.NotEmpty(t, bar.Fill)
}

// Position survives the minimum: two short calls at different points in the
// request must not collapse onto the same cell.
func TestComponent_Bar_ShortSpansKeepTheirPosition(t *testing.T) {
	c := New(time.Second, 40)

	early := c.Bar(Span{Start: 100 * time.Millisecond, Duration: time.Millisecond})
	late := c.Bar(Span{Start: 900 * time.Millisecond, Duration: time.Millisecond})

	assert.NotEqual(t, len(early.Lead), len(late.Lead))
}
