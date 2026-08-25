package app

import (
	"os"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

// TestMain pins the colour profile.
//
// lipgloss sniffs the terminal to decide what it may emit, and under go test
// there is none: left alone it degrades to no colour, so an assertion about
// colour would pass against an empty string. Pinning it also makes a rendered
// screen the same bytes on a developer's machine and in CI.
func TestMain(m *testing.M) {
	lipgloss.SetColorProfile(termenv.TrueColor)
	lipgloss.SetHasDarkBackground(true)

	os.Exit(m.Run())
}
