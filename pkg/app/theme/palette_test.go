package theme

import (
	"fmt"
	"math"
	"strconv"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/lucasb-eyer/go-colorful"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ground the interface is drawn against. Terminals vary, but black is the worst
// case for a dark theme and the one worth measuring.
const ground = "#000000"

// Every colour the interface draws with comes from the palette.
//
// Four do not: the three greys sampled along the palette's own White to Grey
// line, and PHP's own purple, which is deliberately somebody else's brand.
// This test is what keeps that list where it is.
func TestEveryTokenComesFromThePalette(t *testing.T) {
	palette := map[string]struct{}{
		White: {}, Grey: {}, Orange: {}, Blue: {}, Green: {},
		Yellow: {}, Red: {}, DarkBlueGrey: {}, DimBlue: {}, DimGrey: {},
		GreyBright: {}, GreyMid: {}, GreySoft: {}, PHPPurple: {},
	}

	th := Default()

	tokens := map[string]string{
		"TextStrong":      th.TextStrong.TrueColor,
		"TextPrimary":     th.TextPrimary.TrueColor,
		"TextDim":         th.TextDim.TrueColor,
		"TextFaint":       th.TextFaint.TrueColor,
		"TextInverse":     th.TextInverse.TrueColor,
		"SurfaceSelected": th.SurfaceSelected.TrueColor,
		"SurfaceRaised":   th.SurfaceRaised.TrueColor,
		"Border":          th.Border.TrueColor,
		"BorderStrong":    th.BorderStrong.TrueColor,
		"Track":           th.Track.TrueColor,
		"Chrome":          th.Chrome.TrueColor,
		"ChromeFar":       th.ChromeFar.TrueColor,
		"ChromeText":      th.ChromeText.TrueColor,
		"LogoField":       th.LogoField.TrueColor,
		"Accent":          th.Accent.TrueColor,
		"AccentBright":    th.AccentBright.TrueColor,
		"Brand":           th.Brand.TrueColor,
		"RuntimePHP":      th.RuntimePHP.TrueColor,
		"RuntimeNode":     th.RuntimeNode.TrueColor,
		"SourceHTTP":      th.SourceHTTP.TrueColor,
		"SourceCLI":       th.SourceCLI.TrueColor,
		"StateConnected":  th.StateConnected.TrueColor,
		"StateConnecting": th.StateConnecting.TrueColor,
		"StateRetrying":   th.StateRetrying.TrueColor,
		"StateIdle":       th.StateIdle.TrueColor,
	}

	for name, colour := range tokens {
		assert.Contains(t, palette, colour, "%s is not a palette colour", name)
	}

	for i, stop := range th.Ramp {
		assert.Contains(t, palette, stop.TrueColor, "ramp stop %d is not a palette colour", i)
	}
}

// The derived greys are on the line between the palette's own White and Grey,
// so they are the palette's grey at more places along it rather than three new
// colours which happen to look greyish.
func TestDerivedGreysAreOnThePaletteLine(t *testing.T) {
	white, grey := mustColour(t, White), mustColour(t, Grey)

	for _, hex := range []string{GreyBright, GreyMid, GreySoft} {
		colour := mustColour(t, hex)

		nearest := 1.0

		for step := 0; step <= 100; step++ {
			if d := colour.DistanceLab(white.BlendLab(grey, float64(step)/100)); d < nearest {
				nearest = d
			}
		}

		assert.Less(t, nearest, 0.02, "%s is off the White to Grey line", hex)
	}
}

// Two points is not a hierarchy. The scale has to be ordered and separated
// enough to read as one, and every rung has to be legible: most of what a row
// is made of is secondary, so a "secondary" grey below the line means a table
// you have to work at.
func TestGreyLadderIsOrderedAndLegible(t *testing.T) {
	black := mustColour(t, ground)

	ladder := []struct{ name, hex string }{
		{"White", White},
		{"GreyBright", GreyBright},
		{"GreyMid", GreyMid},
		{"GreySoft", GreySoft},
		{"Grey", Grey},
	}

	for i, rung := range ladder {
		assert.GreaterOrEqual(t, contrast(mustColour(t, rung.hex), black), 4.5,
			"%s is not legible", rung.name)

		if i == 0 {
			continue
		}

		above := contrast(mustColour(t, ladder[i-1].hex), black)
		below := contrast(mustColour(t, rung.hex), black)

		assert.Greater(t, above, below, "%s should be brighter than %s", ladder[i-1].name, rung.name)
		assert.Greater(t, above/below, 1.2, "%s and %s are too close to tell apart", ladder[i-1].name, rung.name)
	}
}

// Anything the reader has to read has to be readable.
func TestForegroundsAreLegible(t *testing.T) {
	th := Default()

	black := mustColour(t, ground)

	for name, colour := range map[string]string{
		"TextStrong":      th.TextStrong.TrueColor,
		"TextPrimary":     th.TextPrimary.TrueColor,
		"TextDim":         th.TextDim.TrueColor,
		"TextFaint":       th.TextFaint.TrueColor,
		"AccentBright":    th.AccentBright.TrueColor,
		"Brand":           th.Brand.TrueColor,
		"RuntimePHP":      th.RuntimePHP.TrueColor,
		"RuntimeNode":     th.RuntimeNode.TrueColor,
		"SourceHTTP":      th.SourceHTTP.TrueColor,
		"SourceCLI":       th.SourceCLI.TrueColor,
		"StateConnected":  th.StateConnected.TrueColor,
		"StateConnecting": th.StateConnecting.TrueColor,
		"StateRetrying":   th.StateRetrying.TrueColor,
	} {
		assert.GreaterOrEqual(t, contrast(mustColour(t, colour), black), 4.5,
			"%s is not legible on the ground", name)
	}
}

// The ramp colours text as well as bars — a duration and a self share are both
// rendered in it — so every stop has to be readable, not just the hot end.
func TestWholeRampIsLegible(t *testing.T) {
	black := mustColour(t, ground)

	for step := 0; step <= RampSteps; step++ {
		colour := mustColour(t, RampAt(float64(step)/float64(RampSteps)).TrueColor)

		assert.GreaterOrEqual(t, contrast(colour, black), 4.5, "ramp at step %d", step)
	}
}

// Warmth is a rotation of hue, not a rise in any one channel.
func TestRampWarmsMonotonically(t *testing.T) {
	previous := 360.0

	for step := 0; step <= RampSteps; step++ {
		colour := mustColour(t, RampAt(float64(step)/float64(RampSteps)).TrueColor)

		hue, _, _ := colour.Hcl()

		assert.LessOrEqual(t, hue, previous+1, "hue rose at step %d (%s)", step, colour.Hex())

		previous = hue
	}
}

// The bar and the rail it sits in have to be distinguishable by colour as well
// as by glyph.
func TestRampCoolEndIsDistinctFromTheTrack(t *testing.T) {
	assert.NotEqual(t, Default().Track.TrueColor, RampAt(0).TrueColor)
}

// What sits on the chrome has to be readable there, not on the ground.
func TestChromeIsLegible(t *testing.T) {
	th := Default()

	assert.Equal(t, Blue, th.Chrome.TrueColor, "the chrome should be Compass's primary blue")

	assert.GreaterOrEqual(t,
		contrast(mustColour(t, th.ChromeText.TrueColor), mustColour(t, th.Chrome.TrueColor)),
		4.5, "the tab label is not legible on the chrome")
}

// The hatching is white, which makes it the loudest thing in the masthead. That
// is deliberate, so it is written down: the mark stands in the field rather
// than the field sitting behind the mark.
func TestLogoFieldIsWhite(t *testing.T) {
	assert.Equal(t, White, Default().LogoField.TrueColor)
}

// The blue the masthead is actually made of is the chrome's and nothing else's.
//
// Its far end is not: DimBlue is also the ramp's coolest stop. That overlap is
// harmless in a way the alternative was not — the cool end of the ramp means
// "nothing to see here", so a wordmark fading toward it fades toward silence.
// Fading toward the hot end would have the masthead borrowing the colour of the
// worst thing on the page, which is what an orange chrome did.
func TestChromeIsNotOnTheRamp(t *testing.T) {
	th := Default()

	for i, stop := range th.Ramp {
		assert.NotEqual(t, th.Chrome.TrueColor, stop.TrueColor, "ramp stop %d is the chrome colour", i)
	}

	// And where it does overlap, it overlaps the cool end.
	assert.Equal(t, th.Ramp[0].TrueColor, th.ChromeFar.TrueColor)
}

// No 256 colour fallback may land on a hue the true colour does not have.
//
// This is not hypothetical: an earlier fallback for a blue was xterm 68, which
// is #5f87d7 — a periwinkle. On any terminal without true colour the brand blue
// rendered violet.
func TestFallbacksKeepTheirHue(t *testing.T) {
	th := Default()

	for name, colour := range map[string]lipgloss.CompleteColor{
		"Blue":         th.Chrome,
		"AccentBright": th.AccentBright,
		"RuntimePHP":   th.RuntimePHP,
		"Green":        th.RuntimeNode,
		"Yellow":       th.SourceCLI,
		"Orange":       th.Brand,
		"Red":          th.StateRetrying,
		"TextPrimary":  th.TextPrimary,
		"TextDim":      th.TextDim,
	} {
		trueColour := mustColour(t, colour.TrueColor)
		fallback := mustColour(t, xterm(t, colour.ANSI256))

		trueHue, trueChroma, _ := trueColour.Hcl()
		fallbackHue, fallbackChroma, _ := fallback.Hcl()

		// Greys have no meaningful hue, so only coloured tokens are checked.
		if trueChroma < 0.1 || fallbackChroma < 0.1 {
			continue
		}

		// Ten degrees. The cube is coarse, so some drift is unavoidable — the
		// worst of these is under seven — but ten is well inside the range
		// where a colour still reads as itself, and the two failures this
		// replaces were at ten and twenty-five.
		assert.LessOrEqual(t, math.Abs(trueHue-fallbackHue), 10.0,
			"%s falls back to a different hue: %s becomes %s", name, colour.TrueColor, fallback.Hex())
	}
}

// xterm resolves a 256 colour index to its hex.
func xterm(t *testing.T, index string) string {
	t.Helper()

	levels := []int{0, 95, 135, 175, 215, 255}

	i, err := strconv.Atoi(index)
	require.NoError(t, err)
	require.GreaterOrEqual(t, i, 16, "indices below sixteen are the terminal's own")

	if i >= 232 {
		v := 8 + (i-232)*10

		return rgbHex(v, v, v)
	}

	j := i - 16

	return rgbHex(levels[j/36], levels[(j/6)%6], levels[j%6])
}

// rgbHex of three channels.
func rgbHex(r, g, b int) string {
	return fmt.Sprintf("#%02x%02x%02x", r, g, b)
}

func mustColour(t *testing.T, hex string) colorful.Color {
	t.Helper()

	c, err := colorful.Hex(hex)
	require.NoError(t, err, "bad hex %q", hex)

	return c
}

// contrast ratio between two colours, as WCAG defines it.
func contrast(a, b colorful.Color) float64 {
	la, lb := relativeLuminance(a), relativeLuminance(b)
	if la < lb {
		la, lb = lb, la
	}

	return (la + 0.05) / (lb + 0.05)
}

func relativeLuminance(c colorful.Color) float64 {
	channel := func(v float64) float64 {
		if v <= 0.03928 {
			return v / 12.92
		}

		return math.Pow((v+0.055)/1.055, 2.4)
	}

	return 0.2126*channel(c.R) + 0.7152*channel(c.G) + 0.0722*channel(c.B)
}
