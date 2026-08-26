package theme

// The Skpr palette.
//
// These eleven are the whole palette. Nothing outside this file names a colour
// and nothing here is derived: a repaint happens by changing a value here, and
// the tokens in theme.go keep their meaning.
//
// Three of them are not usable as text on a dark terminal — DarkBlueGrey at
// 1.45:1, DarkGrey at 1.66:1 and DimGrey at 2.82:1 against black — and the
// roles they are given below reflect that. Their own comments say as much: two
// are backgrounds, the third draws delimiters.
const (
	// White for content that needs focus.
	White = "#FFFFFF"
	// Grey for non focused elements.
	Grey = "#808080"
	// Orange which aligns with the Skpr theme.
	Orange = "#EE5622"
	// Blue is the branding blue, and Compass's primary.
	Blue = "#1876d2"
	// Green is the branding green.
	Green = "#5fdd9d"
	// Yellow is the branding yellow.
	Yellow = "#f8df3d"
	// Red is the branding red.
	Red = "#ff6058"
	// DarkBlueGrey is for raised surfaces: a tab you could move to, a card.
	DarkBlueGrey = "#1e2a3a"
	// DarkGrey is the band under the row the cursor is on.
	//
	// Grey rather than the blue grey above it, and a shade lighter. Contrast
	// against the terminal's own ground is luminance and nothing else, so the
	// blue was being paid for in chroma and returning none of it: at the same
	// strength of band a neutral leaves more contrast for the row sitting on
	// it. It was DarkBlueGrey, which at 1.45:1 is a difference you can find
	// once you know it is there rather than one you can see across a full width
	// row — and finding which row is selected is the whole job. This reads at
	// 1.66:1 with every value on the row still above 4.5:1.
	//
	// It also leaves the blue greys to mean one thing. Those are the raised
	// surfaces, which share the chrome's family because they are part of it; a
	// selected row is not a tab, and now does not look like one.
	DarkGrey = "#333333"
	// DimBlue is for low intensity blues.
	DimBlue = "#5f87af"
	// DimGrey is for delimiters and rails.
	DimGrey = "#555555"
)

// PHPPurple is not a Skpr colour and is not meant to be.
//
// It is PHP's own, from the elephant and the logo on php.net. A runtime column
// is naming somebody else's project, and the colour people already associate
// with it identifies it faster than anything from our palette could. It reads
// 5.28:1 on black, so it carries text.
const PHPPurple = "#777BB4"

// The grey scale, sampled along the line between the palette's own White and
// its Grey.
//
// The palette gives two points on that line and two more — DimGrey and
// DarkBlueGrey — which are too dark to carry text at all. Two points is not a
// hierarchy: with only White and Grey to choose from, everything on a row which
// is not the single most important thing lands on the same rung, and five of
// the seven fields in a table row render as the same colour. These are that
// same grey, taken at three more places along it, so a row has depth without
// anything on it being hard to read.
//
// The contrast on each is against black, the worst case for a dark terminal.
const (
	GreyBright = "#d9d9d9" // 14.88:1 — values you came to read
	GreyMid    = "#b9b9b9" // 10.70:1 — values you read second
	GreySoft   = "#9c9c9c" //  7.65:1 — the names of things, separators
)

// The 256 colour fallbacks, each the nearest index in the xterm cube measured
// in Lab rather than in RGB.
//
// The distinction matters more than it sounds. Nearest-by-RGB put the brand
// blue on index 32, ten degrees of hue toward cyan, and an earlier blue on
// index 68 — a periwinkle. A terminal without true colour then rendered the
// brand blue as something else entirely, which is the sort of thing nobody
// notices until they are looking at it. TestFallbacksKeepTheirHue holds these
// to account.
//
// Indices below sixteen are never used: those are defined by the reader's
// terminal theme, so matching one gives a colour which changes under us.
const (
	idxWhite      = "231"
	idxGreyBright = "253"
	idxGreyMid    = "250"
	idxGreySoft   = "247"
	idxGrey       = "244"
	idxOrange     = "202"
	idxBlue       = "25"
	idxGreen      = "78"
	idxYellow     = "220"
	idxRed        = "203"
	// The cube has no dark blue greys at all, so this is the nearest neutral.
	// It is only ever a background, where losing the blue cast costs nothing.
	idxDarkBlueGrey = "235"
	// The band is already a neutral, so its fallback is the cube's own grey
	// ramp and loses nothing at all.
	idxDarkGrey  = "236"
	idxDimBlue   = "67"
	idxDimGrey   = "240"
	idxPHPPurple = "103"
)

// ANSI codes for the sixteen colour fallback, where only the eight hues exist.
const (
	ansiRed     = "1"
	ansiGreen   = "2"
	ansiYellow  = "3"
	ansiBlue    = "4"
	ansiGrey    = "7"
	ansiBrGrey  = "8"
	ansiBrRed   = "9"
	ansiBrGreen = "10"
	ansiBrBlue  = "12"
	ansiBrWhite = "15"
)
