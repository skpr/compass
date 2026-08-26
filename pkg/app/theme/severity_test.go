package theme

import (
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/lucasb-eyer/go-colorful"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestForDurationMs(t *testing.T) {
	tests := []struct {
		ms       int64
		expected Severity
	}{
		{ms: 0, expected: LevelNone},
		{ms: 49, expected: LevelNone},
		{ms: 50, expected: LevelTrace},
		{ms: 99, expected: LevelTrace},
		{ms: 100, expected: LevelOK},
		{ms: 249, expected: LevelOK},
		{ms: 250, expected: LevelNotice},
		{ms: 499, expected: LevelNotice},
		{ms: 500, expected: LevelWarn},
		{ms: 999, expected: LevelWarn},
		{ms: 1000, expected: LevelHigh},
		{ms: 2999, expected: LevelHigh},
		{ms: 3000, expected: LevelCritical},
		{ms: 60_000, expected: LevelCritical},
	}

	for _, tt := range tests {
		assert.Equal(t, tt.expected, ForDurationMs(tt.ms), "%dms", tt.ms)
	}
}

func TestForShare(t *testing.T) {
	assert.Equal(t, LevelNone, ForShare(0))
	assert.Equal(t, LevelNone, ForShare(0.049))
	assert.Equal(t, LevelTrace, ForShare(0.05))
	assert.Equal(t, LevelOK, ForShare(0.10))
	assert.Equal(t, LevelNotice, ForShare(0.15))
	assert.Equal(t, LevelWarn, ForShare(0.25))
	assert.Equal(t, LevelHigh, ForShare(0.40))
	assert.Equal(t, LevelCritical, ForShare(0.60))
	assert.Equal(t, LevelCritical, ForShare(1))
}

// Zero is the value which made the response uncacheable, so it has to be the
// top of the scale, and permanent has to be the bottom.
func TestForMaxAge(t *testing.T) {
	assert.Equal(t, LevelCritical, ForMaxAge(0))
	assert.Equal(t, LevelNone, ForMaxAge(MaxAgePermanent))
	assert.Equal(t, LevelWarn, ForMaxAge(60))
	assert.Equal(t, LevelNotice, ForMaxAge(900))
	assert.Equal(t, LevelOK, ForMaxAge(86400))
}

func TestRampAt_EndsAreTheEndStops(t *testing.T) {
	theme := Default()

	assert.Equal(t, theme.Ramp[0].TrueColor, RampAt(0).TrueColor)
	assert.Equal(t, theme.Ramp[len(theme.Ramp)-1].TrueColor, RampAt(1).TrueColor)
}

func TestRampAt_ClampsOutOfRange(t *testing.T) {
	assert.Equal(t, RampAt(0).TrueColor, RampAt(-5).TrueColor)
	assert.Equal(t, RampAt(1).TrueColor, RampAt(5).TrueColor)
}

// The ramp has to actually get warmer.
//
// Warmth is a rotation of hue, not a rise in any one channel: coral is a
// lighter red than gold, so the blue channel climbs at the hot end even though
// the colour is unambiguously hotter. Measuring hue in HCL is what matches
// what the eye is doing, and the scale is built so it never wraps past red.
func TestRampAt_HueRotatesTowardRed(t *testing.T) {
	previous := 360.0

	for step := 0; step <= RampSteps; step++ {
		colour, err := colorful.Hex(RampAt(float64(step) / float64(RampSteps)).TrueColor)
		require.NoError(t, err)

		hue, _, _ := colour.Hcl()

		assert.LessOrEqual(t, hue, previous+1, "hue rose at step %d (%s)", step, colour.Hex())

		previous = hue
	}

	// And it has actually travelled: blue to red, not a wobble in the greens.
	coolest, err := colorful.Hex(RampAt(0).TrueColor)
	require.NoError(t, err)

	hottest, err := colorful.Hex(RampAt(1).TrueColor)
	require.NoError(t, err)

	coolHue, _, _ := coolest.Hcl()
	hotHue, _, _ := hottest.Hcl()

	assert.Greater(t, coolHue-hotHue, 180.0, "the ramp should cross most of the colour wheel")
}

// Quantisation is what keeps a bar from emitting a different escape sequence in
// every cell, so it has to actually quantise.
func TestRampAt_IsQuantised(t *testing.T) {
	// Two positions inside the same sixteenth resolve to the same colour.
	assert.Equal(t, RampAt(0.501).TrueColor, RampAt(0.505).TrueColor)

	distinct := make(map[string]struct{})

	for i := 0; i <= 1000; i++ {
		distinct[RampAt(float64(i)/1000).TrueColor] = struct{}{}
	}

	assert.LessOrEqual(t, len(distinct), RampSteps+1)
}

// Below true colour there is no index for a blended value, so the ramp falls
// back to the named stops rather than to a rounded approximation of them.
func TestRampAt_FallbacksSnapToNamedStops(t *testing.T) {
	theme := Default()

	named := make(map[string]struct{}, len(theme.Ramp))
	for _, stop := range theme.Ramp {
		named[stop.ANSI256] = struct{}{}
	}

	for step := 0; step <= RampSteps; step++ {
		colour := RampAt(float64(step) / float64(RampSteps))

		assert.Contains(t, named, colour.ANSI256, "step %d", step)
	}
}

// The styles are built from the ramp table at package initialisation, and Go
// initialises variables before it runs init functions. Building the table in an
// init left every ramp style holding a zero value colour, which renders as no
// colour at all — a timeline drawn entirely in the terminal's default
// foreground, with nothing to say it had failed. This asserts the ordering.
func TestRampStylesAreInitialisedBeforeTheStylesWhichReadThem(t *testing.T) {
	for step := 0; step <= RampSteps; step++ {
		position := float64(step) / float64(RampSteps)

		assert.NotEmpty(t, RampAt(position).TrueColor, "step %d", step)
		assert.Contains(t, S.Ramp(position).Render("x"), "\x1b[", "step %d renders without colour", step)
	}
}

// Every colour used to draw with needs a fallback at every depth.
//
// Track was grouped with the surfaces and given an empty sixteen colour
// fallback, on the reasoning that a background should simply vanish there. But
// it is a foreground — the groove of a bar is drawn, not painted behind — and
// an empty fallback means the terminal's default foreground, which is white. A
// dim groove turned into a solid white slab on exactly the terminals least able
// to cope with one.
func TestForegroundTokensHaveASixteenColourFallback(t *testing.T) {
	th := Default()

	foregrounds := map[string]lipgloss.CompleteColor{
		"TextStrong":      th.TextStrong,
		"TextPrimary":     th.TextPrimary,
		"TextDim":         th.TextDim,
		"TextFaint":       th.TextFaint,
		"Track":           th.Track,
		"Border":          th.Border,
		"BorderStrong":    th.BorderStrong,
		"Accent":          th.Accent,
		"AccentBright":    th.AccentBright,
		"Brand":           th.Brand,
		"RuntimePHP":      th.RuntimePHP,
		"RuntimeNode":     th.RuntimeNode,
		"SourceHTTP":      th.SourceHTTP,
		"SourceCLI":       th.SourceCLI,
		"StateConnected":  th.StateConnected,
		"StateConnecting": th.StateConnecting,
		"StateRetrying":   th.StateRetrying,
		"StateIdle":       th.StateIdle,
	}

	for name, colour := range foregrounds {
		assert.NotEmpty(t, colour.TrueColor, "%s true colour", name)
		assert.NotEmpty(t, colour.ANSI256, "%s 256 colour", name)
		assert.NotEmpty(t, colour.ANSI, "%s 16 colour", name)
	}

	for i, stop := range th.Ramp {
		assert.NotEmpty(t, stop.ANSI, "ramp stop %d has no 16 colour fallback", i)
	}
}

// The surfaces are the exception, and deliberately so: a background which
// cannot be represented is better absent than approximated.
func TestSurfacesFallBackToNoBackground(t *testing.T) {
	th := Default()

	assert.Empty(t, th.SurfaceSelected.ANSI)
	assert.Empty(t, th.SurfaceRaised.ANSI)
}
