package main

import (
	"strings"

	"github.com/muesli/termenv"
)

// EnvColor states the colour depth outright, for when detection is wrong.
const EnvColor = "COMPASS_COLOR"

// Colour depths COMPASS_COLOR accepts. Anything else, "auto" included, leaves
// detection to do its job.
var colorNames = map[string]termenv.Profile{
	"truecolor": termenv.TrueColor,
	"24bit":     termenv.TrueColor,
	"256":       termenv.ANSI256,
	"16":        termenv.ANSI,
	"none":      termenv.Ascii,
}

// colorProfile is the depth the interface renders at.
//
// Sixteen colours are not the palette's. At that depth every token resolves to
// an index in the reader's own terminal theme, so the brand blue is whatever
// they have named blue: Dracula names it #bd93f9, a purple, and the masthead,
// the active tab and every key hint come out purple with it. The palette's own
// blues are only reachable from 256 upward.
//
// termenv answers sixteen for a bare TERM=xterm, which is right about xterm's
// terminfo and wrong about the terminal in front of it. `docker compose exec`
// hands the container TERM=xterm whatever it was launched from, dropping the
// colour it forwards nothing about, and so does plenty of ssh — which is how
// the documented way to run Compass ends up being the way that renders purple.
// So a sixteen colour answer is taken as 256, where the fallbacks in palette.go
// apply and blue is blue.
//
// TERM=linux is the one place sixteen is the truth: the kernel console really
// has no more, and emitting 256 colour sequences at it would print them.
func colorProfile(override, term string, detected termenv.Profile) termenv.Profile {
	if profile, ok := colorNames[strings.ToLower(strings.TrimSpace(override))]; ok {
		return profile
	}

	if detected == termenv.ANSI && term != "linux" {
		return termenv.ANSI256
	}

	return detected
}
