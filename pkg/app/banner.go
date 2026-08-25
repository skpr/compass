package app

import (
	"strings"

	"github.com/skpr/compass/pkg/app/component/logo"
	"github.com/skpr/compass/pkg/app/layout"
	"github.com/skpr/compass/pkg/app/theme"
)

// Wordmark shown across the top.
const Wordmark = "COMPASS"

// viewBanner renders the masthead.
//
// The wordmark is drawn as block letterforms with a gradient running across
// them, standing in a field of diagonal hatching. The letterforms are what make
// it a masthead: at terminal resolution there is no type size to reach for, so
// the only way to make something bigger is to draw it out of more cells.
//
// The gradient runs within the blue: from Compass's primary to the palette's
// dim blue and no further. Fifteen degrees of hue, so the mark reads as one
// colour with depth rather than as two blended — a gradient between two hues
// has to pass through everything in between, and the middle of a wordmark is
// the wrong place to find a colour belonging to neither end.
//
// It sits on the terminal's own ground rather than on a filled band. A gradient
// needs somewhere dark to run — on a solid blue the only colours with enough
// contrast are all within a shade of white, which is no gradient at all.
func (m *Model) viewBanner() string {
	width := max(m.Width, 1)

	rows := logo.Render(logo.Options{
		Width: width,
		Word:  Wordmark,
		From:  theme.S.Theme().Chrome,
		To:    theme.S.Theme().ChromeFar,
		Field: theme.S.Theme().LogoField,
	})

	// A blank line under the mark, so the tabs read as a separate thing rather
	// than as the bottom of it.
	for len(rows) < layout.BannerHeight {
		rows = append(rows, strings.Repeat(" ", width))
	}

	return strings.Join(rows[:layout.BannerHeight], "\n")
}
