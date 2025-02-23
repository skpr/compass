package types

// Page for selecting a view.
type Page string

const (
	// PageSearch for searching traces.
	PageSearch Page = "Search"
	// PageSpans for viewing spans of a trace.
	PageSpans Page = "Spans"
	// PageTotals for viewing totals of a trace.
	PageTotals Page = "Totals"
	// PageLogs for view log events.
	PageLogs Page = "Logs"
)
