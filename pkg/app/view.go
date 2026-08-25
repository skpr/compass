package app

import (
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/skpr/compass/pkg/app/component/datatable"
	"github.com/skpr/compass/pkg/app/component/text"
	"github.com/skpr/compass/pkg/app/layout"
	"github.com/skpr/compass/pkg/app/theme"
)

// View for this model.
//
// The regions are computed rather than assumed, and every one of them renders
// exactly the height it was given, so the composed screen is always exactly the
// terminal and never scrolls it.
func (m Model) View() string {
	if m.regions.TooSmall {
		width, height := max(m.Width, 1), max(m.Height, 1)

		// Fitted, not just centred: the message is wider than the terminals
		// which get to see it, and Place does not truncate.
		return lipgloss.Place(
			width, height,
			lipgloss.Center, lipgloss.Center,
			theme.S.Empty.Render(text.Fit("terminal too small", width)),
		)
	}

	sections := []string{m.viewBanner(), m.viewMenu()}

	// The help takes everything the strips and the page would have had between
	// them, so it is measured against the screen rather than the content region.
	if m.showHelp {
		sections = append(sections, m.viewHelp(), m.viewFooter())

		return strings.Join(sections, "\n")
	}

	if m.regions.Detail > 0 {
		sections = append(sections, m.viewDetail())
	}

	if m.regions.Filter > 0 {
		sections = append(sections, m.viewFilter())
	}

	sections = append(sections, m.pageView())

	if m.regions.Inspect > 0 {
		sections = append(sections, m.inspectView())
	}

	sections = append(sections, m.viewFooter())

	return strings.Join(sections, "\n")
}

// pageView of whichever page is selected.
func (m Model) pageView() string {
	switch m.PageSelected {
	case PageSearch:
		return m.searchView()
	case PageLogs:
		return m.logsView()
	case PageFunctions:
		return m.functionsView()
	case PageDrupal:
		return m.drupalView()
	default:
		return ""
	}
}

// refreshRows rebuilds whichever list the filter applies to.
//
// The filter is shared between the two lists rather than being one per page,
// so only the page on screen needs rebuilding.
func (m *Model) refreshRows() {
	switch m.PageSelected {
	case PageSearch:
		m.searchSetRows()
	case PageLogs:
		m.logsSetRows()
	}
}

// contentHeight available to the page.
func (m Model) contentHeight() int {
	return max(m.regions.Content, 1)
}

// overlayHeight available to something drawn in place of the whole interior:
// everything between the menu and the footer, strips included.
func (m Model) overlayHeight() int {
	return max(m.Height-m.regions.Banner-m.regions.Menu-m.regions.Footer, 1)
}

// layoutOptions for wherever the interface currently is.
func (m *Model) layoutOptions() layout.Options {
	options := layout.Options{
		Filter:        m.filtering(),
		InspectHeight: m.inspectHeight(),
	}

	if m.inTrace() {
		options.DetailHeight = m.detailHeight(m.Width)
	}

	return options
}

// relayout recomputes the regions and pushes the sizes into the tables.
//
// It runs on a page change as well as on a resize, because which strips are on
// screen depends on which page is showing.
func (m *Model) relayout() {
	m.regions = layout.Compute(m.Width, m.Height, m.layoutOptions())

	height := m.contentHeight()

	// Guarded rather than assumed: a resize or a page change can arrive before
	// Init has built the tables, and a layout pass is not worth a crash.
	for _, table := range []*datatable.Model{m.search, m.logsTable, m.functions, m.drupal} {
		if table == nil {
			continue
		}

		table.SetSize(m.Width, height)
	}

	if m.functions != nil {
		m.functionsSetColumns()
	}
}
