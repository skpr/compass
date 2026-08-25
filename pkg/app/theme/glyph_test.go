package theme

import (
	"testing"

	"github.com/mattn/go-runewidth"
	"github.com/stretchr/testify/assert"
)

// Every glyph the interface draws with has to be exactly one cell.
//
// The layout budgets width in cells, so a two cell glyph pushes its whole row
// past the edge of the screen. This is not hypothetical: the span component
// used to draw its rail with U+FF5C, a fullwidth bar, and it was budgeted as
// one cell. That glyph would fail this test.
func TestGlyphsAreOneCell(t *testing.T) {
	glyphs := map[string]string{
		"RuleLight":         RuleLight,
		"RuleHeavy":         RuleHeavy,
		"RuleVertical":      RuleVertical,
		"CornerTopLeft":     CornerTopLeft,
		"CornerTopRight":    CornerTopRight,
		"CornerBottomLeft":  CornerBottomLeft,
		"CornerBottomRight": CornerBottomRight,
		"TeeLeft":           TeeLeft,
		"TeeRight":          TeeRight,
		"BarFull":           BarFull,
		"MarkerPresent":     MarkerPresent,
		"MarkerAbsent":      MarkerAbsent,
		"MarkerSeparator":   MarkerSeparator,
		"MarkerEllipsis":    MarkerEllipsis,
		"MarkerItem":        MarkerItem,
		"MarkerRepeat":      MarkerRepeat,
		"SelectionRail":     SelectionRail,
		"SelectionRailEnd":  SelectionRailEnd,
	}

	for name, glyph := range glyphs {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, 1, runewidth.StringWidth(glyph))
		})
	}
}

func TestSparksAndSpinnerAreOneCell(t *testing.T) {
	for i, spark := range Sparks {
		assert.Equal(t, 1, runewidth.StringWidth(spark), "spark %d", i)
	}

	for i, frame := range Spinner {
		assert.Equal(t, 1, runewidth.StringWidth(frame), "frame %d", i)
	}
}

// Proves the guard above has teeth. U+FF5C is the glyph the span component
// used to draw its rail with, and it is two cells everywhere.
func TestFullwidthGlyphWouldFailTheGuard(t *testing.T) {
	assert.Equal(t, 2, runewidth.StringWidth("｜"))
}
