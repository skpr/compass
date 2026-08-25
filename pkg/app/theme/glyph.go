package theme

// The glyphs the interface is drawn with.
//
// Every one of these is a single terminal cell. That is a hard requirement,
// not a preference: the layout budgets width in cells, so a glyph which is two
// cells wide silently pushes everything on its row past the edge. Anything
// whose East Asian Width is Wide or Fullwidth is therefore banned, which is
// what the width test in this package enforces.
//
// Nothing here needs a patched font.
const (
	// Rules and frames.
	RuleLight    = "─"
	RuleHeavy    = "━"
	RuleVertical = "│"

	CornerTopLeft     = "╭"
	CornerTopRight    = "╮"
	CornerBottomLeft  = "╰"
	CornerBottomRight = "╯"
	TeeLeft           = "├"
	TeeRight          = "┤"

	// Bars. The track is a solid block in a dark colour rather than a shaded
	// glyph: the shaded ones dither, which shimmers while a list scrolls and
	// aliases against the fill once the terminal is down to 256 colours.
	BarFull = "█"

	// Markers.
	MarkerPresent   = "●"
	MarkerAbsent    = "○"
	MarkerSeparator = "·"
	MarkerEllipsis  = "…"
	// MarkerItem is the small triangle rather than the large one, which carries
	// emoji presentation and renders two cells wide in several terminals.
	MarkerItem   = "▸"
	MarkerRepeat = "×"

	// SelectionRail marks the row the cursor is on.
	SelectionRail = "▌"
)

// Sparks are the eight heights of a sparkline, shortest first.
var Sparks = [8]string{"▁", "▂", "▃", "▄", "▅", "▆", "▇", "█"}

// Spinner frames for work in progress.
var Spinner = [4]string{"▖", "▘", "▝", "▗"}
