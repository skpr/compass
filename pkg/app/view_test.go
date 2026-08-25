package app

import (
	"fmt"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/skpr/compass/pkg/app/component/datatable"
	"github.com/skpr/compass/pkg/app/events"
	"github.com/skpr/compass/pkg/app/theme"
	"github.com/skpr/compass/pkg/trace"
)

// testModel with some traces and logs in it, at a size.
func testModel(width, height int) *Model {
	m := NewModel("/proc/1/root/usr/lib/php/modules/compass.so", 500)
	m.Init()
	m.updateWindowSize(tea.WindowSizeMsg{Width: width, Height: height})
	m.updateConnection(events.Connection{State: events.ConnectionStateConnected})

	for _, uri := range []string{"/", "/en/recipes", "/frontend/health"} {
		m.updateTrace(testTrace(uri))
	}

	m.updateLog(events.Log{Time: time.Now(), Type: "error", Message: "connection refused"})

	return m
}

func testTrace(uri string) events.Trace {
	return events.Trace{
		IngestionTime: time.Now(),
		Trace: trace.Trace{
			Metadata: trace.Metadata{
				Source: trace.SourceHTTP, Runtime: trace.RuntimePHP,
				ID:        "58bb2c6e56c13ce04c1cb9a87083d735",
				HTTP:      trace.MetadataHTTP{Method: "GET", URI: uri},
				StartTime: 0, EndTime: 402_000_000,
			},
			ResourceUtilisation: trace.ResourceUtilisation{MaxMemory: 18 << 20},
			FunctionCalls: []trace.FunctionCall{
				{Name: `Drupal\Core\DrupalKernel::handle`, StartTime: 0, Elapsed: 400_000_000, Memory: 18 << 20},
				{Name: `Drupal\Core\Render\Renderer::renderRoot`, StartTime: 100_000_000, Elapsed: 150_000_000, Memory: 16 << 20},
			},
			Drupal: &trace.Drupal{
				CacheEvents: []trace.CacheEvent{
					{Origin: trace.CacheOriginObject, Caller: `Drupal\user\Plugin\Block\UserLoginBlock::build`, ObjectType: `Drupal\Core\Session\AccountProxy`, MaxAge: 0, Calls: 3, Contexts: []string{"session"}},
					{Origin: trace.CacheOriginRenderArray, Caller: `Drupal\Core\Render\Renderer::doRender`, MaxAge: -1, Calls: 412, Tags: []string{"node:12"}},
				},
				CacheEventsDropped: 6,
			},
		},
	}
}

// The invariant the whole layout rests on: whatever is on screen, the screen is
// exactly the terminal. A view a line too tall scrolls the terminal; a line too
// short leaves the previous frame's last row behind.
func TestView_IsExactlyTheTerminal(t *testing.T) {
	sizes := []struct{ width, height int }{
		{200, 60}, {120, 40}, {120, 30}, {100, 24}, {80, 24}, {80, 12}, {60, 7},
		{59, 40}, {120, 6}, {40, 20}, {10, 10}, {1, 1},
	}

	pages := []Page{PageSearch, PageLogs, PageFunctions, PageDrupal}

	for _, size := range sizes {
		for _, page := range pages {
			m := testModel(size.width, size.height)

			if page == PageFunctions || page == PageDrupal {
				m.updateKeyEnter()
				m.PageSelected = page
				m.relayout()
			} else {
				m.PageSelected = page
				m.relayout()
			}

			view := m.View()

			assert.Equal(t, size.height, lipgloss.Height(view),
				"height on %s at %dx%d", page, size.width, size.height)

			for i, line := range strings.Split(view, "\n") {
				assert.LessOrEqual(t, ansi.StringWidth(line), size.width,
					"line %d on %s at %dx%d: %q", i, page, size.width, size.height, ansi.Strip(line))
			}
		}
	}
}

func TestView_WithFilterOpen(t *testing.T) {
	m := testModel(120, 30)
	m.startFilter()
	m.updateFilter(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("rec")})

	assert.Equal(t, 30, lipgloss.Height(m.View()))
	assert.Contains(t, ansi.Strip(m.View()), "/rec")
}

func TestView_WithHelpOpen(t *testing.T) {
	m := testModel(120, 30)
	m.showHelp = true

	view := m.View()

	assert.Equal(t, 30, lipgloss.Height(view))
	assert.Contains(t, ansi.Strip(view), "KEYS")
	assert.Contains(t, ansi.Strip(view), "uncacheable")

	// The key column is padded in cells, not bytes, so the hints containing
	// arrows line up with the ones that do not.
	assert.NotContains(t, ansi.Strip(view), "kmove")
}

func TestView_TooSmall(t *testing.T) {
	m := testModel(20, 5)

	assert.Contains(t, ansi.Strip(m.View()), "too small")
}

// Everything has to stay readable with the colour taken away, which is the
// acceptance test for encoding severity in the gutter and the bar as well as
// in the hue.
func TestView_ReadableWithoutColour(t *testing.T) {
	m := testModel(120, 30)

	plain := ansi.Strip(m.View())

	assert.Contains(t, plain, "SEARCH")
	assert.Contains(t, plain, "GET /en/recipes")
	assert.Contains(t, plain, "402ms")
}

func TestSearch_OpensTheTraceUnderTheCursor(t *testing.T) {
	m := testModel(120, 30)

	m.search.SetCursor(1)
	m.updateKeyEnter()

	require.NotNil(t, m.Current)
	assert.Equal(t, "/en/recipes", m.Current.Metadata.HTTP.URI)
	assert.Equal(t, PageFunctions, m.PageSelected)
}

// The filter reorders the list, so the row under the cursor is not at its own
// index in the unfiltered traces. Opening the wrong trace would be a silent,
// maddening bug.
func TestSearch_OpensTheRightTraceWhileFiltered(t *testing.T) {
	m := testModel(120, 30)

	m.startFilter()
	m.updateFilter(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("health")})

	require.Equal(t, 1, m.search.Len(), "the filter should have narrowed to one trace")

	m.endFilter()
	m.search.SetCursor(0)
	m.updateKeyEnter()

	require.NotNil(t, m.Current)
	assert.Equal(t, "/frontend/health", m.Current.Metadata.HTTP.URI)
}

func TestFilter_NarrowsAndRestores(t *testing.T) {
	m := testModel(120, 30)

	require.Equal(t, 3, m.search.Len())

	m.startFilter()
	m.updateFilter(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("recipes")})
	assert.Less(t, m.search.Len(), 3)

	m.clearFilter()
	assert.Equal(t, 3, m.search.Len())
	assert.False(t, m.filtering())
}

func TestFilter_IsNotOfferedOnTheTracePages(t *testing.T) {
	m := testModel(120, 30)
	m.updateKeyEnter()

	assert.False(t, m.filterable())
}

// The request id is the thing which ties a trace to whatever else logged during
// it, so it has to be on the list, not only inside the trace.
func TestSearch_ShowsTheRequestID(t *testing.T) {
	m := testModel(120, 30)

	row, ok := m.search.SelectedRow()
	require.True(t, ok)

	assert.Contains(t, cellTexts(row), "58bb2c6e")
}

// A trace which arrived without an X-Request-ID header has no id, and the
// extension says so with a sentinel. Printing that sentinel on every row would
// be worse than printing nothing.
func TestSearch_UnknownRequestIDIsNotShown(t *testing.T) {
	m := testModel(120, 30)

	anonymous := testTrace("/anonymous")
	anonymous.Metadata.ID = trace.IDUnknown

	m.updateTrace(anonymous)
	m.search.SetCursor(0)

	row, ok := m.search.SelectedRow()
	require.True(t, ok)

	texts := cellTexts(row)
	assert.NotContains(t, texts, trace.IDUnknown)
	assert.Contains(t, texts, "·", "an absent id should read as absent, not as a word")
}

// The list shortens the id to fit; the open trace is where the whole value is,
// so that there is somewhere to copy it from. That is what the wide fields are
// for — a request id with its last twenty characters elided is no use to
// anybody.
func TestDetail_ShowsTheFullRequestID(t *testing.T) {
	m := testModel(120, 30)
	m.updateKeyEnter()

	assert.Contains(t, ansi.Strip(m.viewDetail()), "58bb2c6e56c13ce04c1cb9a87083d735")
}

func TestDetail_OmitsAnUnknownRequestID(t *testing.T) {
	m := testModel(120, 30)

	anonymous := testTrace("/anonymous")
	anonymous.Metadata.ID = trace.IDUnknown

	m.updateTrace(anonymous)
	m.search.SetCursor(0)
	m.updateKeyEnter()

	assert.NotContains(t, ansi.Strip(m.viewDetail()), trace.IDUnknown)
}

// cellTexts of a row, with the styling taken off.
func cellTexts(row datatable.Row) []string {
	texts := make([]string, 0, len(row))

	for _, cell := range row {
		texts = append(texts, cell.String())
	}

	return texts
}

// The functions page reads as the request ran: what called what, and where the
// time went in sequence. A ranking would take the causal structure out of it.
func TestFunctions_OrderedByExecution(t *testing.T) {
	m := testModel(120, 40)

	m.Current = &events.Trace{Trace: trace.Trace{
		Metadata: trace.Metadata{
			Source: trace.SourceHTTP, StartTime: 0, EndTime: 1_000_000_000,
		},
		FunctionCalls: []trace.FunctionCall{
			// Deliberately out of order, and with the hottest last, so that
			// neither insertion order nor self time could produce this result
			// by accident.
			{Name: "third", StartTime: 600_000_000, Elapsed: 100_000_000},
			{Name: "first", StartTime: 100_000_000, Elapsed: 50_000_000},
			{Name: "second", StartTime: 300_000_000, Elapsed: 500_000_000},
		},
	}}

	m.functionsSetRows()

	assert.Equal(t, []string{"first", "second", "third"}, functionNames(m))
}

// A caller and the call it makes can start in the same instant. The caller has
// to come first, or the page reads as though the child invoked the parent.
func TestFunctions_CallerBeforeCallee(t *testing.T) {
	m := testModel(120, 40)

	m.Current = &events.Trace{Trace: trace.Trace{
		Metadata: trace.Metadata{
			Source: trace.SourceHTTP, StartTime: 0, EndTime: 1_000_000_000,
		},
		FunctionCalls: []trace.FunctionCall{
			{Name: "child", StartTime: 0, Elapsed: 200_000_000},
			{Name: "parent", StartTime: 0, Elapsed: 900_000_000},
		},
	}}

	m.functionsSetRows()

	assert.Equal(t, []string{"parent", "child"}, functionNames(m))
}

// Ordering by time is not the same as ranking by it: the hotspot still has to
// be findable, which is what the self column is for.
func TestFunctions_HotspotIsFindableOutOfOrder(t *testing.T) {
	m := testModel(120, 40)

	m.Current = &events.Trace{Trace: trace.Trace{
		Metadata: trace.Metadata{
			Source: trace.SourceHTTP, StartTime: 0, EndTime: 1_000_000_000,
		},
		FunctionCalls: []trace.FunctionCall{
			// A frame which wraps the whole request but does none of the work,
			// and the function actually burning the time underneath it.
			{Name: "wrapper", StartTime: 0, Elapsed: 1_000_000_000},
			{Name: "hotspot", StartTime: 100_000_000, Elapsed: 800_000_000},
		},
	}}

	m.functionsSetRows()

	require.Equal(t, []string{"wrapper", "hotspot"}, functionNames(m))

	rows := m.functions.Rows()

	// The wrapper is the longer call and comes first, but the hotspot is the
	// one the self column points at. The wrapper keeps the 200ms it spent
	// outside the child, which is the point: self time is what it did itself,
	// not nothing.
	assert.Equal(t, "20.0%", rows[0][selfColumn].String(), "the wrapper's own time")
	assert.Equal(t, "80.0%", rows[1][selfColumn].String(), "the hotspot's own time")

	// And they are not the same colour, so it is visible as well as readable.
	assert.NotEqual(t,
		rows[0][selfColumn].Segments[0].Style.Render("x"),
		rows[1][selfColumn].Segments[0].Style.Render("x"),
	)
}

// Column positions on the functions page, for the tests which read a cell.
const (
	functionColumn = 0
	selfColumn     = 1
)

// functionNames of the rows on the functions page, in the order shown.
func functionNames(m *Model) []string {
	names := make([]string, 0, m.functions.Len())

	for _, row := range m.functions.Rows() {
		names = append(names, row[functionColumn].String())
	}

	return names
}

// A legend which quietly stops halfway is worse than a short one: the reader
// cannot tell "that is all of it" from "your terminal is too short".
func TestView_HelpMarksWhereItWasCut(t *testing.T) {
	short := testModel(90, 22)
	short.showHelp = true

	assert.Contains(t, ansi.Strip(short.View()), "more, on a taller terminal")

	tall := testModel(90, 34)
	tall.showHelp = true

	view := ansi.Strip(tall.View())
	assert.NotContains(t, view, "more, on a taller terminal")
	assert.Contains(t, view, "SELF", "the whole legend should fit a terminal this tall")
}

// The marker fires for one reason only. A mark which means two different things
// stops meaning either of them.
func TestSearch_MarkerIsOnlyForUncacheable(t *testing.T) {
	m := testModel(120, 30)

	cacheable := testTrace("/cacheable")
	cacheable.Drupal = &trace.Drupal{
		// Truncated, but perfectly cacheable: no marker.
		CacheEvents:        []trace.CacheEvent{{Caller: "A::a", MaxAge: -1, Calls: 1}},
		CacheEventsDropped: 99,
	}

	m.updateTrace(cacheable)
	m.search.SetCursor(0)

	row, ok := m.search.SelectedRow()
	require.True(t, ok)
	assert.Equal(t, " ", row[attentionColumn].String())

	// And the trace which cannot be cached does carry it.
	m.search.SetCursor(1)

	row, ok = m.search.SelectedRow()
	require.True(t, ok)
	assert.Equal(t, AttentionMarker, row[attentionColumn].String())
}

// attentionColumn is the marker's position on a search row.
const attentionColumn = 0

// The table counts the tags; the panel below it says what they are. Without it
// the values were collected, aggregated and then never shown anywhere.
func TestDrupal_InspectShowsTheSelectedRowsTagsAndContexts(t *testing.T) {
	m := testModel(120, 34)

	rich := testTrace("/rich")
	rich.Drupal = &trace.Drupal{CacheEvents: []trace.CacheEvent{
		{
			Origin: trace.CacheOriginObject, Caller: "Sql::execute", MaxAge: 0, Calls: 6,
			ObjectType: `Drupal\views\ViewExecutable`,
			Tags:       []string{"node_list", "node:12", "taxonomy_term_list"},
			Contexts:   []string{"user", "url.query_args:page"},
		},
		{
			Origin: trace.CacheOriginRenderArray, Caller: "Other::build", MaxAge: 3600, Calls: 2,
			Tags: []string{"config:system.site"},
		},
	}}

	m.updateTrace(rich)
	m.search.SetCursor(0)
	m.updateKeyEnter()
	m.PageSelected = PageDrupal
	m.relayout()
	m.drupalSetRows()

	panel := ansi.Strip(m.inspectView())

	assert.Contains(t, panel, "node_list")
	assert.Contains(t, panel, "taxonomy_term_list")
	assert.Contains(t, panel, "url.query_args:page")

	// The table shows the class on its own; the namespace is what says which
	// module it came from, so the panel carries the whole name.
	assert.Contains(t, panel, `Drupal\views\ViewExecutable`)

	// And it follows the cursor rather than showing the first row forever.
	m.drupal.MoveDown(1)

	panel = ansi.Strip(m.inspectView())

	assert.Contains(t, panel, "config:system.site")
	assert.NotContains(t, panel, "node_list")
	assert.Contains(t, panel, "none", "an empty list should say so rather than render blank")

	// A render array has no object at all, which is a fact rather than a gap.
	assert.Contains(t, panel, "object")
}

// The panel is a fixed height, so the table below does not move as the cursor
// walks over rows with different numbers of tags.
func TestDrupal_InspectIsAFixedHeight(t *testing.T) {
	m := testModel(120, 34)
	m.updateKeyEnter()
	m.PageSelected = PageDrupal
	m.relayout()

	for _, cursor := range []int{0, 1} {
		m.drupal.SetCursor(cursor)

		panel := m.inspectView()

		assert.Equal(t, m.inspectHeight(), lipgloss.Height(panel), "cursor=%d", cursor)

		for _, line := range strings.Split(panel, "\n") {
			assert.LessOrEqual(t, ansi.StringWidth(line), 120)
		}
	}
}

// A tag list too long for the screen is cut, not wrapped: wrapping would make
// the panel taller than the layout budgeted for it.
func TestDrupal_InspectCutsALongTagList(t *testing.T) {
	m := testModel(120, 34)

	long := testTrace("/long")

	tags := make([]string, 0, 40)
	for i := range 40 {
		tags = append(tags, fmt.Sprintf("node:%d", i))
	}

	long.Drupal = &trace.Drupal{CacheEvents: []trace.CacheEvent{
		{Origin: trace.CacheOriginObject, Caller: "Many::tags", MaxAge: 0, Calls: 1, Tags: tags},
	}}

	m.updateTrace(long)
	m.search.SetCursor(0)
	m.updateKeyEnter()
	m.PageSelected = PageDrupal
	m.relayout()
	m.drupalSetRows()

	panel := m.inspectView()

	assert.Equal(t, m.inspectHeight(), lipgloss.Height(panel))
	assert.Contains(t, ansi.Strip(panel), "…")
}

// One section, every field named. The point of the block is that you do not
// have to know what a position means to read it.
func TestDetail_ShowsEveryField(t *testing.T) {
	m := testModel(120, 34)
	m.updateKeyEnter()

	block := ansi.Strip(m.viewDetail())

	for _, label := range []string{
		"URI", "ID", "RUNTIME", "SOURCE", "METHOD",
		"DURATION", "MAX MEMORY", "CALLS", "INGESTED",
		"CACHEABLE", "CACHE TAGS", "CONTEXTS", "DROPPED",
	} {
		assert.Contains(t, block, label, "the block is missing %s", label)
	}

	assert.Contains(t, block, "/frontend/health")
	assert.Contains(t, block, "402ms")
	assert.Contains(t, block, "uncacheable")
}

// The identifying fields lead, so the row cap takes the least useful away
// first rather than the most.
func TestDetail_IdentifyingFieldsComeFirst(t *testing.T) {
	for _, width := range []int{200, 120, 80, 62} {
		m := testModel(width, 40)
		m.updateKeyEnter()

		lines := strings.Split(ansi.Strip(m.viewDetail()), "\n")

		require.GreaterOrEqual(t, len(lines), 2)
		assert.Contains(t, lines[0], "URI", "width=%d", width)
		assert.Contains(t, lines[1], "ID", "width=%d", width)
	}
}

// A value nobody wants abbreviated gets a row of its own, whatever the width.
func TestDetail_LongValuesAreNotTruncated(t *testing.T) {
	for _, width := range []int{200, 120, 80} {
		m := testModel(width, 40)
		m.updateKeyEnter()

		assert.Contains(t, ansi.Strip(m.viewDetail()), "58bb2c6e56c13ce04c1cb9a87083d735", "width=%d", width)
	}
}

// The block is as tall as its fields need, and the layout is told rather than
// assuming: a wider terminal fits them in fewer rows.
func TestDetail_HeightFollowsTheWidth(t *testing.T) {
	m := testModel(200, 40)
	m.updateKeyEnter()

	wide := m.detailHeight(200)
	narrow := m.detailHeight(80)

	assert.Less(t, wide, narrow, "a wider block should need fewer rows")
	assert.Equal(t, wide, lipgloss.Height(m.viewDetail()))

	for _, width := range []int{200, 120, 80, 62} {
		m.Width = width
		m.relayout()

		assert.Equal(t, m.detailHeight(width), lipgloss.Height(m.viewDetail()), "width=%d", width)
		assert.LessOrEqual(t, m.detailHeight(width), detailMaxRows+detailRule, "width=%d", width)
	}
}

// The block is closed off from the table underneath it. Without the rule its
// last field sits directly on the column headings and the two read as one list
// of labels.
func TestDetail_IsClosedOffFromTheTable(t *testing.T) {
	for _, page := range []Page{PageFunctions, PageDrupal} {
		m := testModel(120, 40)
		m.updateKeyEnter()
		m.PageSelected = page

		lines := strings.Split(ansi.Strip(m.viewDetail()), "\n")

		require.NotEmpty(t, lines)
		assert.Equal(t, strings.Repeat(theme.RuleLight, m.Width), lines[len(lines)-1], "page=%v", page)
	}
}

// And the screen agrees: the row above the column headings is that rule, not a
// field.
func TestView_DetailDoesNotRunIntoTheHeadings(t *testing.T) {
	// A heading only that page's table has, so the search does not land on the
	// tab of the same name.
	for page, heading := range map[Page]string{PageFunctions: "ELAPSED", PageDrupal: "MAX AGE"} {
		m := testModel(120, 40)
		m.updateKeyEnter()
		m.PageSelected = page
		m.relayout()

		lines := strings.Split(ansi.Strip(m.View()), "\n")

		headings := -1

		for i, line := range lines {
			if strings.Contains(line, heading) {
				headings = i
				break
			}
		}

		require.Positive(t, headings, "no column headings on screen for %v", page)
		assert.Equal(t, strings.Repeat(theme.RuleLight, m.Width), lines[headings-1], "page=%v", page)
	}
}

// A counter reading zero is a field spent saying nothing went wrong.
func TestDetail_OmitsDroppedWhenThereAreNone(t *testing.T) {
	m := testModel(120, 34)

	clean := testTrace("/clean")
	clean.Drupal = &trace.Drupal{CacheEvents: []trace.CacheEvent{
		{Caller: "A::a", MaxAge: -1, Calls: 1},
	}}

	m.updateTrace(clean)
	m.search.SetCursor(0)
	m.updateKeyEnter()

	assert.NotContains(t, ansi.Strip(m.viewDetail()), "DROPPED")
}

// The functions table abbreviates a namespace to initials and then truncates
// whatever is left, so the panel below it carries the whole name — and the
// numbers behind the two columns which are a percentage and a picture.
func TestFunctions_InspectShowsTheSelectedRowInFull(t *testing.T) {
	m := testModel(120, 34)
	m.updateKeyEnter()

	panel := ansi.Strip(m.inspectView())

	assert.Contains(t, panel, `Drupal\Core\DrupalKernel::handle`)
	assert.Contains(t, panel, "function")
	assert.Contains(t, panel, "self")
	assert.Contains(t, panel, "window")

	// And it follows the cursor.
	m.functions.MoveDown(1)

	panel = ansi.Strip(m.inspectView())

	assert.Contains(t, panel, `Drupal\Core\Render\Renderer::renderRoot`)
	assert.NotContains(t, panel, "DrupalKernel")
}

// The self column is a percentage and the timeline is a picture. Neither says
// how long anything actually took, which is what the panel is for.
func TestFunctions_InspectGivesTheNumbersBehindTheColumns(t *testing.T) {
	m := testModel(120, 34)
	m.updateKeyEnter()
	m.functions.SetCursor(1)

	panel := ansi.Strip(m.inspectView())

	// renderRoot ran from 100ms to 250ms of a 402ms request.
	assert.Contains(t, panel, "150ms")
	assert.Contains(t, panel, "100ms in")
	assert.Contains(t, panel, "402ms")
}

// Each page's panel is as tall as that page has to say, and the layout is told
// rather than assuming. What has to hold is that a page's own panel never
// changes height as the cursor moves down it, or the table above would shuffle
// with every keypress.
func TestInspect_HeightIsConstantWithinAPage(t *testing.T) {
	m := testModel(120, 34)
	m.updateKeyEnter()

	for _, page := range []Page{PageFunctions, PageDrupal} {
		m.PageSelected = page
		m.relayout()

		height := m.inspectHeight()
		require.Positive(t, height, "%s should have a panel", page)
		assert.Equal(t, height, lipgloss.Height(m.inspectView()), "%s", page)

		for cursor := range 2 {
			m.currentTable().SetCursor(cursor)

			assert.Equal(t, height, lipgloss.Height(m.inspectView()), "%s at cursor %d", page, cursor)
		}
	}

	// The cacheability page has more to say, so its panel is taller.
	m.PageSelected = PageFunctions
	functions := m.inspectHeight()

	m.PageSelected = PageDrupal
	assert.Greater(t, m.inspectHeight(), functions)
}

// A page with no panel gets no region rather than an empty one.
func TestInspect_NoPanelOnTheTopLevel(t *testing.T) {
	m := testModel(120, 34)

	assert.Zero(t, m.inspectHeight())
	assert.Empty(t, m.inspectView())
}

// The panel is closed at both ends. Without a rule below it, its last line runs
// straight into the key rail and the two read as one block of text.
func TestInspect_IsClosedAtBothEnds(t *testing.T) {
	m := testModel(120, 34)
	m.updateKeyEnter()

	for _, page := range []Page{PageFunctions, PageDrupal} {
		m.PageSelected = page
		m.relayout()

		lines := strings.Split(ansi.Strip(m.inspectView()), "\n")
		require.GreaterOrEqual(t, len(lines), 3, "%s", page)

		rule := strings.Repeat("─", 120)

		assert.Equal(t, rule, lines[0], "%s has no rule above it", page)
		assert.Equal(t, rule, lines[len(lines)-1], "%s has no rule below it", page)

		// And the height budgeted for it counts both.
		assert.Equal(t, len(lines), m.inspectHeight(), "%s", page)
	}
}

// The line above the key rail has to be the panel's, not the last thing the
// panel had to say.
func TestView_PanelDoesNotRunIntoTheKeyRail(t *testing.T) {
	m := testModel(120, 34)
	m.updateKeyEnter()

	lines := strings.Split(ansi.Strip(m.View()), "\n")

	footer := lines[len(lines)-1]
	require.Contains(t, footer, "esc")

	assert.Equal(t, strings.Repeat("─", 120), lines[len(lines)-2],
		"the key rail should sit under a rule, not under the panel's last value")
}
