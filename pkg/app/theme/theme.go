// Package theme is the colour vocabulary of the interface.
//
// Dark terminals only: there is no light variant and no adaptive pair. The
// tokens are named for what a colour means rather than what it looks like, so
// that "what does this hue tell me" has an answer for every coloured thing on
// screen. Anything which cannot answer it is decoration and does not belong.
package theme

import (
	"github.com/charmbracelet/lipgloss"
)

// Theme is the set of colours the interface draws with.
type Theme struct {
	// Text, in descending prominence. Three greys carry most of the visual
	// hierarchy, so that the design still works with colour turned off.
	TextStrong  lipgloss.CompleteColor
	TextPrimary lipgloss.CompleteColor
	TextDim     lipgloss.CompleteColor
	TextFaint   lipgloss.CompleteColor
	// TextInverse is for text on a light severity fill, where the usual white
	// would be unreadable.
	TextInverse lipgloss.CompleteColor

	// Surfaces. These resolve to no background on a sixteen colour terminal,
	// which the rest of the design is built to survive.
	SurfaceSelected lipgloss.CompleteColor
	SurfaceRaised   lipgloss.CompleteColor

	// Track is the unfilled part of a bar. It is a foreground rather than a
	// surface — the rail is drawn, not painted behind — so unlike the surfaces
	// it needs a real fallback at every depth. Left empty it inherits the
	// terminal's default foreground, which is white, and the groove turns into
	// a solid bright slab.
	Track lipgloss.CompleteColor

	Border       lipgloss.CompleteColor
	BorderStrong lipgloss.CompleteColor

	// Chrome is the band across the top and the tab standing in it.
	//
	// Chrome is the band across the top and the tab standing in it.
	//
	// Compass's primary blue, and the only job that colour has: nothing else in
	// the interface draws with it, so blue means "you are here" and nothing
	// else. ChromeText is the white which sits on it.
	//
	// ChromeFar is the far end of the wordmark's gradient — the palette's dim
	// blue, fifteen degrees of hue away, so the mark reads as one colour with
	// depth rather than as two blended.
	Chrome     lipgloss.CompleteColor
	ChromeFar  lipgloss.CompleteColor
	ChromeText lipgloss.CompleteColor

	// LogoField is the diagonal hatching the wordmark stands in.
	//
	// White, so the hatching reads as drawn rather than as texture. It is the
	// loudest thing in the masthead as a result — louder than the mark standing
	// in it — which is a look rather than an accident.
	LogoField lipgloss.CompleteColor

	// Accent marks something you can act on: a key, a focus.
	Accent       lipgloss.CompleteColor
	AccentBright lipgloss.CompleteColor
	// Brand is the orange, kept for the one place which needs to be unmistakably
	// Skpr and is not chrome.
	Brand lipgloss.CompleteColor

	// Ramp is the severity scale, coolest first.
	//
	// Five anchors rather than one per severity level: the levels are how
	// finely thresholds are drawn, the anchors are how many colours the brand
	// actually offers to draw them with. Positions between them are blended.
	Ramp [5]lipgloss.CompleteColor

	// Categorical colours label a thing rather than measure it, so they are
	// never blended and never appear on the ramp.
	RuntimePHP  lipgloss.CompleteColor
	RuntimeNode lipgloss.CompleteColor
	SourceHTTP  lipgloss.CompleteColor
	SourceCLI   lipgloss.CompleteColor

	StateConnected  lipgloss.CompleteColor
	StateConnecting lipgloss.CompleteColor
	StateRetrying   lipgloss.CompleteColor
	StateIdle       lipgloss.CompleteColor
}

// complete colour from its true colour, 256 and 16 representations.
func complete(trueColor, ansi256, ansi string) lipgloss.CompleteColor {
	return lipgloss.CompleteColor{TrueColor: trueColor, ANSI256: ansi256, ANSI: ansi}
}

// Default theme.
func Default() Theme {
	return Theme{
		TextStrong:  complete(White, idxWhite, ansiBrWhite),
		TextPrimary: complete(GreyBright, idxGreyBright, ansiBrWhite),
		TextDim:     complete(GreyMid, idxGreyMid, ansiGrey),
		TextFaint:   complete(GreySoft, idxGreySoft, ansiGrey),
		TextInverse: complete(DarkBlueGrey, idxDarkBlueGrey, ansiBlack),

		// Empty ANSI means no colour is emitted at sixteen, which for a
		// background is exactly right: the terminal's own is used instead.
		SurfaceSelected: complete(DarkBlueGrey, idxDarkBlueGrey, ""),
		SurfaceRaised:   complete(DarkBlueGrey, idxDarkBlueGrey, ""),

		// The palette's own delimiters and rails.
		Border:       complete(DimGrey, idxDimGrey, ansiBrGrey),
		BorderStrong: complete(DimGrey, idxDimGrey, ansiBrGrey),
		Track:        complete(DimGrey, idxDimGrey, ansiBrGrey),

		Chrome:     complete(Blue, idxBlue, ansiBlue),
		ChromeFar:  complete(DimBlue, idxDimBlue, ansiBrBlue),
		ChromeText: complete(White, idxWhite, ansiBrWhite),
		LogoField:  complete(White, idxWhite, ansiBrWhite),

		Accent:       complete(Blue, idxBlue, ansiBlue),
		AccentBright: complete(DimBlue, idxDimBlue, ansiBrBlue),
		Brand:        complete(Orange, idxOrange, ansiRed),

		// Cool to hot, every stop from the palette. The cool end is the
		// palette's dim blue rather than a grey, so a bar at the bottom of the
		// scale is still distinguishable from the rail it sits in.
		Ramp: [5]lipgloss.CompleteColor{
			complete(DimBlue, idxDimBlue, ansiBlue),
			complete(Green, idxGreen, ansiBrGreen),
			complete(Yellow, idxYellow, ansiYellow),
			complete(Orange, idxOrange, ansiYellow),
			complete(Red, idxRed, ansiBrRed),
		},

		// PHP's own colour rather than one of ours: it is naming somebody
		// else's project, and the colour people already associate with it says
		// so faster than anything from our palette could.
		RuntimePHP:  complete(PHPPurple, idxPHPPurple, ansiBrBlue),
		RuntimeNode: complete(Green, idxGreen, ansiBrGreen),
		// HTTP is grey on purpose. It is almost every row, so giving it a hue
		// spends the reader's attention on the case which tells them nothing.
		SourceHTTP: complete(Grey, idxGrey, ansiBrGrey),
		SourceCLI:  complete(Yellow, idxYellow, ansiYellow),

		StateConnected:  complete(Green, idxGreen, ansiGreen),
		StateConnecting: complete(Yellow, idxYellow, ansiYellow),
		StateRetrying:   complete(Red, idxRed, ansiBrRed),
		StateIdle:       complete(Grey, idxGrey, ansiBrGrey),
	}
}
