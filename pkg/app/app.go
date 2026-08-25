package app

// Page for selecting a view.
type Page string

const (
	// PageSearch for searching traces.
	PageSearch Page = "Search"
	// PageLogs for view log events.
	PageLogs Page = "Logs"

	// PageFunctions for viewing the function calls of a trace.
	PageFunctions Page = "Functions"
	// PageDrupal for viewing the Drupal specific data collected for a trace.
	PageDrupal Page = "Drupal Cacheable Metadata"
)

// hasDrupal reports whether the open trace has anything Drupal specific to
// show.
//
// Nothing collects it but the Drupal probes, so a Node trace, a PHP CLI run and
// any PHP application which is not Drupal all have none. Offering a page which
// can only say "there is nothing here" is worse than not offering it.
func (m *Model) hasDrupal() bool {
	return m.drupalSummary().Collected
}

// inTrace reports whether a trace is open.
//
// Search and Logs are the whole application. Everything else is a view of one
// trace, reached by opening it from the search page, so the menu and the keys
// behave differently once you are inside one.
func (m *Model) inTrace() bool {
	return m.PageSelected == PageFunctions || m.PageSelected == PageDrupal
}
