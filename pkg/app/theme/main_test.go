package theme

import (
	"os"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

// TestMain pins the colour profile.
//
// lipgloss resolves colour through a renderer which sniffs the terminal, and
// under go test there is no terminal: left alone it degrades to no colour at
// all, so an assertion about colour would silently pass against an empty
// string. Pinning it also means a rendered screen is the same bytes on a
// developer's machine and in CI.
func TestMain(m *testing.M) {
	lipgloss.SetColorProfile(termenv.TrueColor)
	lipgloss.SetHasDarkBackground(true)

	os.Exit(m.Run())
}
