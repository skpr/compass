package app

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/skpr/compass/pkg/app/events"
	"github.com/skpr/compass/pkg/trace"
)

// pressExport presses x and runs the command it returns, the way the program
// would, handing the result back to the model.
func pressExport(t *testing.T, m *Model) exportedMsg {
	t.Helper()

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("x")})
	require.NotNil(t, cmd)

	msg, ok := cmd().(exportedMsg)
	require.True(t, ok)

	m.Update(msg)

	return msg
}

func readExport(t *testing.T, path string) trace.Trace {
	t.Helper()

	encoded, err := os.ReadFile(path)
	require.NoError(t, err)

	var decoded trace.Trace
	require.NoError(t, json.Unmarshal(encoded, &decoded))

	return decoded
}

func TestExport_SelectedTraceOnSearch(t *testing.T) {
	m := newNavTestModel()
	m.ExportDir = t.TempDir()

	m.updateTrace(newTrace("older"))
	m.updateTrace(newTrace("newer"))
	m.search.MoveDown(1)

	msg := pressExport(t, m)

	require.NoError(t, msg.err)
	assert.Equal(t, m.ExportDir, filepath.Dir(msg.path))
	assert.Equal(t, "older", readExport(t, msg.path).Metadata.ID)

	require.NotNil(t, m.notice)
	assert.False(t, m.notice.err)
	assert.Contains(t, m.notice.text, msg.path)
}

func TestExport_OpenTrace(t *testing.T) {
	m := openedTrace(t)
	m.ExportDir = t.TempDir()

	msg := pressExport(t, m)

	require.NoError(t, msg.err)
	assert.Equal(t, "req-1", readExport(t, msg.path).Metadata.ID)
}

func TestExport_NothingSelected(t *testing.T) {
	m := newNavTestModel()

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("x")})

	assert.Nil(t, cmd)
}

// A failure is reported where the reader is looking, and on Logs where it is
// kept.
func TestExport_Failure(t *testing.T) {
	m := newNavTestModel()

	// A file where the directory should be.
	m.ExportDir = filepath.Join(t.TempDir(), "file")
	require.NoError(t, os.WriteFile(m.ExportDir, nil, 0o644))

	m.updateTrace(newTrace("req-1"))

	msg := pressExport(t, m)

	require.Error(t, msg.err)
	require.NotNil(t, m.notice)
	assert.True(t, m.notice.err)

	newest, ok := m.logs.newest(0)
	require.True(t, ok)
	assert.Equal(t, "error", newest.Type)
}

// The notice answers the key which was pressed, so the next one clears it.
func TestExport_NoticeClearsOnNextKey(t *testing.T) {
	m := newNavTestModel()
	m.notice = &notice{text: "exported somewhere"}

	m.Update(tea.KeyMsg{Type: tea.KeyDown})

	assert.Nil(t, m.notice)
}

// Typing into the filter is text, not a command.
func TestExport_NotWhileFiltering(t *testing.T) {
	m := newNavTestModel()
	m.updateTrace(events.Trace{Trace: trace.Trace{Metadata: trace.Metadata{ID: "req-1"}}})
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("/")})

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("x")})

	if cmd != nil {
		_, exported := cmd().(exportedMsg)
		assert.False(t, exported)
	}
}
