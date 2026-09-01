package main

import (
	"testing"

	"github.com/muesli/termenv"
	"github.com/stretchr/testify/assert"
)

// The bug this function exists for.
//
// `docker compose exec compass compass` is the documented way to run Compass
// against the demo stack, and docker hands the container TERM=xterm with no
// COLORTERM, whatever it was launched from. termenv reads that as sixteen
// colours, at which depth Chrome and Accent resolve to the terminal's own
// index 4 — a purple under Dracula — and the masthead, the active tab and the
// key hints all render purple.
func TestGenericSixteenColourTermIsTakenAs256(t *testing.T) {
	assert.Equal(t, termenv.ANSI256, colorProfile("", "xterm", termenv.ANSI))
}

// The kernel console has sixteen colours and no more, so its answer stands.
func TestLinuxConsoleKeepsItsSixteenColours(t *testing.T) {
	assert.Equal(t, termenv.ANSI, colorProfile("", "linux", termenv.ANSI))
}

// Every other answer is passed through: a terminal which says truecolor is
// believed, and no colour at all stays no colour, which is what a pipe and
// NO_COLOR both arrive as.
func TestEveryOtherDetectedProfileIsLeftAlone(t *testing.T) {
	for _, profile := range []termenv.Profile{termenv.TrueColor, termenv.ANSI256, termenv.Ascii} {
		assert.Equal(t, profile, colorProfile("", "xterm-256color", profile))
	}
}

func TestOverrideWins(t *testing.T) {
	cases := map[string]termenv.Profile{
		"truecolor": termenv.TrueColor,
		"24bit":     termenv.TrueColor,
		"TrueColor": termenv.TrueColor,
		" 256 ":     termenv.ANSI256,
		"16":        termenv.ANSI,
		"none":      termenv.Ascii,
	}

	for override, want := range cases {
		t.Run(override, func(t *testing.T) {
			assert.Equal(t, want, colorProfile(override, "xterm-kitty", termenv.TrueColor))
		})
	}
}

// Anything unrecognised leaves detection to do its job, so a typo degrades to
// the behaviour of not having set it rather than to no colour at all.
func TestUnrecognisedOverrideFallsBackToDetection(t *testing.T) {
	for _, override := range []string{"", "auto", "yes", "banana"} {
		assert.Equal(t, termenv.TrueColor, colorProfile(override, "xterm", termenv.TrueColor))
	}
}
