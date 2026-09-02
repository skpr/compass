// Package layout divides the terminal between the regions of the interface.
//
// Every height on screen is decided here. Before this existed the arithmetic
// was spread across three files as magic numbers with no minimums, so a short
// terminal produced negative heights and the components were left to cope.
package layout

// Fixed heights of the chrome, in lines.
//
// These are a contract rather than an observation: a component whose rendered
// height is not exactly this is a bug, and the tests hold both sides to it.
const (
	// BannerHeight is the masthead: three rows of letterforms and a line of air
	// beneath them. The letterforms are drawn two pixel rows to a character
	// row, so three is what a six row bitmap costs.
	BannerHeight = 4
	// MenuHeight is the tab labels and the rule beneath them.
	MenuHeight = 2
	// FooterHeight is the key rail.
	FooterHeight = 1
	// FilterHeight is the rounded search field: top frame, input and bottom frame.
	FilterHeight = 3
)

// MinContentHeight is a header, a rule and a single row.
const MinContentHeight = 3

// The smallest terminal worth drawing a real interface into. Below either of
// these the view says so rather than rendering something unusable.
//
// The height is derived rather than chosen, so it stays honest if a region is
// ever added: it is the chrome which cannot be dropped, plus the smallest
// content region worth showing. Between this and a comfortable terminal the
// optional strips are what give way.
const (
	MinWidth  = 60
	MinHeight = BannerHeight + MenuHeight + FooterHeight + MinContentHeight
)

// Regions of the screen, in lines.
type Regions struct {
	Banner  int
	Menu    int
	Detail  int
	Filter  int
	Inspect int
	Content int
	Footer  int

	Width  int
	Height int

	// TooSmall is set when the terminal cannot hold the chrome and a usable
	// content region together.
	TooSmall bool
}

// Options describing what a screen needs room for.
type Options struct {
	// DetailHeight is the block describing the open trace, which is however
	// many rows its fields need at the current width. Zero when no trace is
	// open.
	DetailHeight int
	// Filter is the input, shown while a list is being narrowed.
	Filter bool
	// InspectHeight is the panel below a table showing the selected row in
	// full, which is however many lines that page has to say. Zero when the
	// page has no panel.
	InspectHeight int
}

// Compute the regions for a terminal size.
//
// Pure and total: it never returns a negative, and every region always sums to
// the height it was given.
func Compute(width, height int, opts Options) Regions {
	regions := Regions{Width: width, Height: height}

	if width < MinWidth || height < MinHeight {
		regions.TooSmall = true

		return regions
	}

	regions.Banner = BannerHeight
	regions.Menu = MenuHeight
	regions.Footer = FooterHeight

	regions.Detail = max(opts.DetailHeight, 0)

	// The filter is never dropped: it is only on screen because the reader
	// asked for it, and taking it away mid-keystroke would be worse than
	// anything else giving way.
	if opts.Filter {
		regions.Filter = FilterHeight
	}

	regions.Inspect = max(opts.InspectHeight, 0)

	regions.Content = height - regions.chrome()

	// The rows are what the reader came for, so the strips above them give way
	// first: on a short terminal a trace's function list is worth more than the
	// summary of a trace you are already looking at.
	// The rows are what the reader came for, so the panels above and below them
	// give way first. Inspect goes before Detail: it describes one row, and a
	// screen with no room for rows has nothing to describe.
	if regions.Content < MinContentHeight && regions.Inspect > 0 {
		regions.Inspect = 0
		regions.Content = height - regions.chrome()
	}

	if regions.Content < MinContentHeight && regions.Detail > 0 {
		regions.Detail = 0
		regions.Content = height - regions.chrome()
	}

	if regions.Content < MinContentHeight {
		return Regions{Width: width, Height: height, TooSmall: true}
	}

	return regions
}

// chrome is every line which is not the content region.
func (r Regions) chrome() int {
	return r.Banner + r.Menu + r.Detail + r.Filter + r.Inspect + r.Footer
}

// Total height of every region, which is the terminal height whenever the
// interface is drawn at all.
func (r Regions) Total() int {
	return r.chrome() + r.Content
}
