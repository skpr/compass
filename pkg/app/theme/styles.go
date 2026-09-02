package theme

import (
	"github.com/charmbracelet/lipgloss"
)

// Styles are every style the interface renders with, built once.
//
// A lipgloss.Style is a value over a backing map, so calling NewStyle or any
// mutator allocates. Doing that inside a render path means allocating on every
// frame, for every row, of a list that repaints as traces stream in. The rule
// this type exists to enforce: no NewStyle and no mutator below View.
//
// The one honest exception is a width, which is not known until the terminal
// size is. Components which need one prepare their styles when they are
// resized, not when they are drawn.
type Styles struct {
	// Text. There is no unbolded Strong: white is close enough to the brand
	// grey that the two are told apart by weight rather than by colour, so the
	// top of the ladder always carries both.
	Primary lipgloss.Style
	Dim     lipgloss.Style
	Faint   lipgloss.Style
	Bold    lipgloss.Style

	// Chrome.
	Wordmark   lipgloss.Style
	TabActive  lipgloss.Style
	TabIdle    lipgloss.Style
	RuleActive lipgloss.Style
	RuleIdle   lipgloss.Style

	// Rows.
	Header      lipgloss.Style
	Cell        lipgloss.Style
	CellDim     lipgloss.Style
	CellFaint   lipgloss.Style
	Selected    lipgloss.Style
	SelectRail  lipgloss.Style
	Track       lipgloss.Style
	Empty       lipgloss.Style
	SurfaceCard lipgloss.Style

	// Interactive filter field.
	FilterSurface     lipgloss.Style
	FilterPrompt      lipgloss.Style
	FilterText        lipgloss.Style
	FilterPlaceholder lipgloss.Style
	FilterMeta        lipgloss.Style

	// Categorical.
	RuntimePHP  lipgloss.Style
	RuntimeNode lipgloss.Style
	SourceHTTP  lipgloss.Style
	SourceCLI   lipgloss.Style

	// Connection state, keyed by the state's own string so the footer does not
	// need a switch.
	State map[string]lipgloss.Style

	// Key hints in the footer rail.
	Key     lipgloss.Style
	KeyDesc lipgloss.Style

	theme    Theme
	severity [7]lipgloss.Style
	ramp     [RampSteps + 1]lipgloss.Style
}

// NewStyles for a theme.
func NewStyles(t Theme) Styles {
	base := lipgloss.NewStyle()

	s := Styles{
		theme: t,

		Primary: base.Foreground(t.TextPrimary),
		Dim:     base.Foreground(t.TextDim),
		Faint:   base.Foreground(t.TextFaint),
		Bold:    base.Foreground(t.TextStrong).Bold(true),

		// White on the brand blue is 4.60:1, which carries bold text and is the
		// only thing on the palette which carries anything there.
		Wordmark: base.Foreground(t.ChromeText).Background(t.Chrome).Bold(true),
		// Filled, so which tab is active is a shape rather than a weight. The
		// active one takes the brand blue, which ties it to the band above it;
		// the rest take a raised surface, which reads as a tab you could move
		// to rather than as body text that happens to be dim.
		TabActive:  base.Foreground(t.ChromeText).Background(t.Chrome).Bold(true).Padding(0, 1),
		TabIdle:    base.Foreground(t.TextDim).Background(t.SurfaceRaised).Padding(0, 1),
		RuleActive: base.Foreground(t.Accent),
		RuleIdle:   base.Foreground(t.Border),

		Header:      base.Foreground(t.TextDim).Bold(true),
		Cell:        base.Foreground(t.TextPrimary),
		CellDim:     base.Foreground(t.TextDim),
		CellFaint:   base.Foreground(t.TextFaint),
		Selected:    base.Background(t.SurfaceSelected),
		SelectRail:  base.Foreground(t.AccentBright),
		Track:       base.Foreground(t.Track),
		Empty:       base.Foreground(t.TextFaint),
		SurfaceCard: base.Background(t.SurfaceRaised).Foreground(t.TextPrimary),

		FilterSurface:     base.Background(t.SurfaceRaised).Foreground(t.TextPrimary),
		FilterPrompt:      base.Background(t.SurfaceRaised).Foreground(t.AccentBright).Bold(true),
		FilterText:        base.Background(t.SurfaceRaised).Foreground(t.TextPrimary),
		FilterPlaceholder: base.Background(t.SurfaceRaised).Foreground(t.TextFaint),
		FilterMeta:        base.Background(t.SurfaceRaised).Foreground(t.TextFaint),

		RuntimePHP:  base.Foreground(t.RuntimePHP),
		RuntimeNode: base.Foreground(t.RuntimeNode),
		SourceHTTP:  base.Foreground(t.SourceHTTP),
		SourceCLI:   base.Foreground(t.SourceCLI),

		State: map[string]lipgloss.Style{
			"connected":  base.Foreground(t.StateConnected),
			"connecting": base.Foreground(t.StateConnecting),
			"retrying":   base.Foreground(t.StateRetrying),
		},

		Key:     base.Foreground(t.AccentBright),
		KeyDesc: base.Foreground(t.TextFaint),
	}

	for level := range s.severity {
		s.severity[level] = base.Foreground(t.Color(Severity(level)))
	}

	for step := range s.ramp {
		s.ramp[step] = base.Foreground(ramp[step])
	}

	return s
}

// Severity style for a level.
func (s Styles) Severity(level Severity) lipgloss.Style {
	if level < 0 || int(level) >= len(s.severity) {
		return s.severity[0]
	}

	return s.severity[level]
}

// Ramp style for a position on the zero to one severity scale.
func (s Styles) Ramp(position float64) lipgloss.Style {
	step := int(clampUnit(position)*float64(RampSteps) + 0.5)

	return s.ramp[step]
}

// Theme these styles were built from, for the few places which need a colour
// rather than a style.
func (s Styles) Theme() Theme {
	return s.theme
}

// S is the interface's styles. Built once, at startup.
var S = NewStyles(Default())
