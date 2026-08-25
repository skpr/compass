package app

import (
	"os"
	"testing"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

// origin the test requests start from. Their timestamps are instants, and the
// offsets the tests are written in are nanoseconds against this.
var origin = time.Unix(1700000000, 0)

// at an offset into a test request.
func at(offset time.Duration) time.Time {
	return origin.Add(offset)
}

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
