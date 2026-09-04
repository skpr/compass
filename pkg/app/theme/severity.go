package theme

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/lucasb-eyer/go-colorful"
)

// Severity of a thing on screen, from unremarkable to the reason you opened
// the tool.
//
// The levels line up one to one with the stops of the colour ramp, so a level
// is both a colour and a glyph weight. Every threshold in the interface is
// stated here and nowhere else: before this existed, "when is it slow" had two
// different answers in two different files.
type Severity int

// Levels, coolest first.
const (
	LevelNone Severity = iota
	LevelTrace
	LevelOK
	LevelNotice
	LevelWarn
	LevelHigh
	LevelCritical
)

// Duration thresholds in milliseconds, at the top of each level.
const (
	DurationNoneMs   = 50
	DurationTraceMs  = 100
	DurationOKMs     = 250
	DurationNoticeMs = 500
	DurationWarnMs   = 1000
	DurationHighMs   = 3000
)

// ForDurationMs is the severity of a wall clock duration.
func ForDurationMs(ms int64) Severity {
	switch {
	case ms < DurationNoneMs:
		return LevelNone
	case ms < DurationTraceMs:
		return LevelTrace
	case ms < DurationOKMs:
		return LevelOK
	case ms < DurationNoticeMs:
		return LevelNotice
	case ms < DurationWarnMs:
		return LevelWarn
	case ms < DurationHighMs:
		return LevelHigh
	default:
		return LevelCritical
	}
}

// Request share thresholds, at the top of each level.
const (
	ShareNone   = 0.05
	ShareTrace  = 0.10
	ShareOK     = 0.15
	ShareNotice = 0.25
	ShareWarn   = 0.40
	ShareHigh   = 0.60
)

// ForShare is the severity of the fraction of a request occupied by a
// function call's elapsed duration.
func ForShare(share float64) Severity {
	switch {
	case share < ShareNone:
		return LevelNone
	case share < ShareTrace:
		return LevelTrace
	case share < ShareOK:
		return LevelOK
	case share < ShareNotice:
		return LevelNotice
	case share < ShareWarn:
		return LevelWarn
	case share < ShareHigh:
		return LevelHigh
	default:
		return LevelCritical
	}
}

// MaxAgePermanent is the max age Drupal uses for "cacheable forever".
const MaxAgePermanent int64 = -1

// Max age thresholds in seconds, at the top of each level.
const (
	MaxAgeNoticeSeconds = 300
	MaxAgeWarnSeconds   = 3600
)

// ForMaxAge is the severity of a Drupal cacheability value. A max age of zero
// is what makes a response uncacheable, so it is the top of the scale;
// permanent is the bottom.
func ForMaxAge(seconds int64) Severity {
	switch {
	case seconds == MaxAgePermanent:
		return LevelNone
	case seconds == 0:
		return LevelCritical
	case seconds < MaxAgeNoticeSeconds:
		return LevelWarn
	case seconds < MaxAgeWarnSeconds:
		return LevelNotice
	default:
		return LevelOK
	}
}

// Position of each ramp anchor on a zero to one scale.
//
// There are fewer anchors than severity levels, and deliberately so: the levels
// are how finely the thresholds are drawn, the anchors are how many colours the
// brand offers to draw them with. Everything between two anchors is blended.
var rampStops = [5]float64{0, 0.30, 0.60, 0.80, 1}

// Position of each severity level on that scale.
//
// Weighted toward the hot end, where the distance between a bad value and a
// very bad one is worth more than the distance between two fine ones.
var severityPositions = [7]float64{0, 0.15, 0.30, 0.45, 0.60, 0.80, 1}

// Position of a severity level on the ramp.
func (level Severity) Position() float64 {
	if level < 0 || int(level) >= len(severityPositions) {
		return 0
	}

	return severityPositions[level]
}

// RampSteps is how finely the continuous ramp is quantised.
//
// A blend evaluated per cell produces a different colour in every cell of a bar,
// which reads as noise rather than as a gradient, and defeats the terminal's
// coalescing of identical escape sequences: forty odd colour changes per row,
// thirty rows, and the repaint stutters over a slow link. Sixteen steps is more
// than the eye resolves in a bar this size.
const RampSteps = 16

// ramp holds the quantised scale, built once. Blending per render would be the
// single hottest allocation in the interface.
//
// Built by a function call rather than in an init, so that Go orders it before
// the styles which read it. Package level variables are initialised before init
// runs, so an init here would leave every ramp style built from a zero value
// colour, which renders as no colour at all.
var ramp = buildRamp()

// buildRamp quantises the continuous scale into its steps.
func buildRamp() [RampSteps + 1]lipgloss.CompleteColor {
	var (
		t      = Default()
		steps  [RampSteps + 1]lipgloss.CompleteColor
		lastAt = float64(RampSteps)
	)

	for i := range steps {
		steps[i] = blendRamp(t, float64(i)/lastAt)
	}

	return steps
}

// blendRamp interpolates the true colour between the two stops a position
// falls between.
//
// Only the true colour is blended. The 256 and 16 colour representations snap
// to the nearer stop instead, because an interpolated value has no index of its
// own and rounding one to the nearest available shade is what washes a ramp out
// into mud.
func blendRamp(t Theme, position float64) lipgloss.CompleteColor {
	position = clampUnit(position)

	upper := 1
	for upper < len(rampStops)-1 && rampStops[upper] < position {
		upper++
	}

	lower := upper - 1

	var local float64
	if span := rampStops[upper] - rampStops[lower]; span > 0 {
		local = (position - rampStops[lower]) / span
	}

	// A position which lands exactly on a stop keeps that stop verbatim, rather
	// than round-tripping through the blend and coming back in a different case.
	// The named stops are what the palette declares; the blend fills between.
	switch local {
	case 0:
		return t.Ramp[lower]
	case 1:
		return t.Ramp[upper]
	}

	nearer := lower
	if local >= 0.5 {
		nearer = upper
	}

	from, errFrom := colorful.Hex(t.Ramp[lower].TrueColor)
	to, errTo := colorful.Hex(t.Ramp[upper].TrueColor)

	if errFrom != nil || errTo != nil {
		return t.Ramp[nearer]
	}

	return lipgloss.CompleteColor{
		TrueColor: from.BlendHcl(to, local).Clamped().Hex(),
		ANSI256:   t.Ramp[nearer].ANSI256,
		ANSI:      t.Ramp[nearer].ANSI,
	}
}

// RampAt is the colour for a position on the zero to one severity scale.
func RampAt(position float64) lipgloss.CompleteColor {
	step := int(clampUnit(position)*float64(RampSteps) + 0.5)

	return ramp[step]
}

// Color of a severity level.
func (t Theme) Color(level Severity) lipgloss.CompleteColor {
	return RampAt(level.Position())
}

// clampUnit holds a value inside zero to one.
func clampUnit(v float64) float64 {
	switch {
	case v < 0:
		return 0
	case v > 1:
		return 1
	default:
		return v
	}
}

// Gradient of n colours running between two.
//
// lipgloss v1 has no blending of its own, so this is the one place a ramp is
// built outside the severity scale. Only the true colour is interpolated; the
// fallbacks snap to whichever end is nearer, because an interpolated value has
// no index of its own to fall back to.
func Gradient(from, to lipgloss.CompleteColor, n int) []lipgloss.CompleteColor {
	if n <= 0 {
		return nil
	}

	if n == 1 {
		return []lipgloss.CompleteColor{from}
	}

	start, errStart := colorful.Hex(from.TrueColor)
	end, errEnd := colorful.Hex(to.TrueColor)

	colours := make([]lipgloss.CompleteColor, n)

	for i := range colours {
		position := float64(i) / float64(n-1)

		nearer := from
		if position >= 0.5 {
			nearer = to
		}

		if errStart != nil || errEnd != nil {
			colours[i] = nearer

			continue
		}

		colours[i] = lipgloss.CompleteColor{
			TrueColor: start.BlendHcl(end, position).Clamped().Hex(),
			ANSI256:   nearer.ANSI256,
			ANSI:      nearer.ANSI,
		}
	}

	return colours
}
