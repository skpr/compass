package app

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/skpr/compass/pkg/app/events"
	"github.com/skpr/compass/pkg/trace"
)

func TestUpdateKeyRight_FromSearch(t *testing.T) {
	m := &Model{PageSelected: PageSearch}
	m.updateKeyRight()
	assert.Equal(t, PageLogs, m.PageSelected)
}

func TestUpdateKeyRight_FromLogs(t *testing.T) {
	m := &Model{PageSelected: PageLogs}
	m.updateKeyRight()
	// Should stay on Logs (no wrap).
	assert.Equal(t, PageLogs, m.PageSelected)
}

func TestUpdateKeyLeft_FromLogs(t *testing.T) {
	m := &Model{PageSelected: PageLogs}
	m.updateKeyLeft()
	assert.Equal(t, PageSearch, m.PageSelected)
}

func TestUpdateKeyLeft_FromSearch(t *testing.T) {
	m := &Model{PageSelected: PageSearch}
	m.updateKeyLeft()
	// Should stay on Search (no wrap).
	assert.Equal(t, PageSearch, m.PageSelected)
}

// The trace pages are their own level, so moving between them must not escape
// back out to Search or Logs.
func TestUpdateKeyRight_FromFunctions(t *testing.T) {
	m := openedTrace(t)
	m.updateKeyRight()
	assert.Equal(t, PageDrupal, m.PageSelected)
}

func TestUpdateKeyRight_FromDrupal(t *testing.T) {
	m := openedTrace(t)
	m.PageSelected = PageDrupal
	m.updateKeyRight()
	assert.Equal(t, PageDrupal, m.PageSelected)
}

func TestUpdateKeyLeft_FromDrupal(t *testing.T) {
	m := openedTrace(t)
	m.PageSelected = PageDrupal
	m.updateKeyLeft()
	assert.Equal(t, PageFunctions, m.PageSelected)
}

func TestUpdateKeyLeft_FromFunctions(t *testing.T) {
	m := openedTrace(t)
	m.updateKeyLeft()
	assert.Equal(t, PageFunctions, m.PageSelected)
}

// A trace with nothing Drupal in it has no Drupal page, so the key which would
// move to it does nothing rather than moving somewhere invisible.
func TestUpdateKeyRight_NoDrupalPageToMoveTo(t *testing.T) {
	m := openedTrace(t)
	m.Current.Drupal = nil

	m.updateKeyRight()

	assert.Equal(t, PageFunctions, m.PageSelected)
	assert.False(t, m.hasDrupal())
}

// openedTrace with Drupal data, on the Functions page.
func openedTrace(t *testing.T) *Model {
	t.Helper()

	m := newNavTestModel()

	m.updateTrace(events.Trace{Trace: trace.Trace{
		Metadata: trace.Metadata{
			ID: "req-1", Source: trace.SourceHTTP,
			HTTP:      trace.MetadataHTTP{Method: "GET", URI: "/node/1"},
			StartTime: at(0), EndTime: at(1_000_000),
		},
		Drupal: &trace.Drupal{CacheEvents: []trace.CacheEvent{
			{Caller: "A::a", MaxAge: 0, Calls: 1},
		}},
	}})

	m.updateKeyEnter()

	return m
}

func TestUpdateKeyEnter_NotOnSearch(t *testing.T) {
	m := &Model{PageSelected: PageLogs}
	m.updateKeyEnter()
	assert.Equal(t, PageLogs, m.PageSelected)
	assert.Nil(t, m.Current)
}

// There is no way into the trace pages other than opening a trace, so an empty
// search page must not be able to get there.
func TestUpdateKeyEnter_NothingSelected(t *testing.T) {
	m := newNavTestModel()

	m.updateKeyEnter()

	assert.Equal(t, PageSearch, m.PageSelected)
	assert.Nil(t, m.Current)
}

func TestUpdateKeyEnter_OpensTheTrace(t *testing.T) {
	m := newNavTestModel()

	m.updateTrace(events.Trace{
		Trace: trace.Trace{
			Metadata: trace.Metadata{
				ID:        "req-1",
				Source:    trace.SourceHTTP,
				HTTP:      trace.MetadataHTTP{Method: "GET", URI: "/node/1"},
				StartTime: at(1_000_000),
				EndTime:   at(2_000_000),
			},
		},
	})

	m.updateKeyEnter()

	require.NotNil(t, m.Current)
	assert.Equal(t, "req-1", m.Current.Metadata.ID)
	assert.Equal(t, PageFunctions, m.PageSelected)
}

func TestUpdateKeyEsc_ReturnsToSearch(t *testing.T) {
	m := &Model{PageSelected: PageDrupal}
	m.updateKeyEsc()
	assert.Equal(t, PageSearch, m.PageSelected)
}

func TestInTrace(t *testing.T) {
	assert.False(t, (&Model{PageSelected: PageSearch}).inTrace())
	assert.False(t, (&Model{PageSelected: PageLogs}).inTrace())
	assert.True(t, (&Model{PageSelected: PageFunctions}).inTrace())
	assert.True(t, (&Model{PageSelected: PageDrupal}).inTrace())
}

// newNavTestModel with the tables initialised, which the enter handler needs in
// order to ask the search list what is selected.
func newNavTestModel() *Model {
	m := &Model{Width: 120, Height: 40}
	m.Init()

	return m
}

// Arrow and wheel input is an ESC-prefixed ANSI sequence. Bubble Tea can split
// it into a standalone Esc and a rune suffix when the terminal read fragments,
// which must not close the trace.
func TestUpdate_FragmentedArrowDoesNotCloseTrace(t *testing.T) {
	m := openedTrace(t)

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	require.NotNil(t, cmd)
	assert.Equal(t, PageFunctions, m.PageSelected)
	assert.True(t, m.traceClosePending)
	sequence := m.traceCloseSequence

	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("[B")})
	assert.Equal(t, PageFunctions, m.PageSelected)
	assert.False(t, m.traceClosePending)

	// The already-scheduled close must also be harmless after cancellation.
	m.Update(deferredTraceCloseMsg{sequence: sequence})
	assert.Equal(t, PageFunctions, m.PageSelected)
	require.NotNil(t, m.Current)
}

func TestUpdate_StandaloneEscStillReturnsToSearch(t *testing.T) {
	m := openedTrace(t)

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	require.NotNil(t, cmd)
	assert.Equal(t, PageFunctions, m.PageSelected)

	m.Update(cmd())
	assert.Equal(t, PageSearch, m.PageSelected)
	assert.False(t, m.traceClosePending)
}
