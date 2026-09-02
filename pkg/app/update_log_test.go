package app

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/skpr/compass/pkg/app/events"
	"github.com/skpr/compass/pkg/app/theme"
)

func testLog(second int, level, message string) events.Log {
	return events.Log{
		Time:    time.Unix(int64(second), 0),
		Type:    level,
		Message: message,
	}
}

func TestUpdateLog_BoundsHistoryAndDisplaysNewestFirst(t *testing.T) {
	m := NewModel("", 10, 3)
	m.Init()

	for i := range 5 {
		m.updateLog(testLog(i, "info", fmt.Sprintf("message-%d", i)))
	}

	assert.Equal(t, 3, m.logs.len())
	assert.Equal(t, 3, m.logsTable.Len())
	assert.Equal(t, []string{"message-4", "message-3", "message-2"}, logMessages(m))

	oldest, ok := m.logs.oldest(0)
	require.True(t, ok)
	assert.Equal(t, "message-2", oldest.Message)
}

func TestUpdateLog_RepeatsCollapseAcrossOldestEviction(t *testing.T) {
	m := NewModel("", 10, 3)
	m.Init()

	m.updateLog(testLog(0, "error", "A"))
	m.updateLog(testLog(1, "error", "A"))
	m.updateLog(testLog(2, "warn", "B"))

	require.Equal(t, []string{"B", "A"}, logMessages(m))
	require.Equal(t, theme.MarkerRepeat+"2", m.logsTable.Rows()[1][2].String())

	// The first A is evicted while its run still has one retained occurrence.
	m.updateLog(testLog(3, "warn", "B"))
	require.Equal(t, []string{"B", "A"}, logMessages(m))
	assert.Equal(t, theme.MarkerRepeat+"2", m.logsTable.Rows()[0][2].String())
	assert.Empty(t, m.logsTable.Rows()[1][2].String())

	// Evicting the final A removes its row; the B run still counts only the
	// three raw events which remain inside the retention boundary.
	m.updateLog(testLog(4, "warn", "B"))
	require.Equal(t, []string{"B"}, logMessages(m))
	assert.Equal(t, theme.MarkerRepeat+"3", m.logsTable.Rows()[0][2].String())
	assert.Equal(t, 3, m.logs.len())
}

func TestUpdateLog_FilteringUsesCollapsedRetainedRuns(t *testing.T) {
	m := NewModel("", 10, 4)
	m.Init()

	m.updateLog(testLog(0, "info", "database ready"))
	m.updateLog(testLog(1, "error", "connection refused"))
	m.updateLog(testLog(2, "error", "connection refused"))
	m.filter.SetValue("connection")
	m.logsSetRows()

	require.Equal(t, []string{"connection refused"}, logMessages(m))
	assert.Equal(t, theme.MarkerRepeat+"2", m.logsTable.Rows()[0][2].String())

	// Filtered arrivals take the full filter-match path, but remain bounded and
	// preserve collapse counts.
	m.updateLog(testLog(3, "warn", "unrelated warning"))
	m.updateLog(testLog(4, "error", "connection refused"))

	require.Equal(t, []string{"connection refused", "connection refused"}, logMessages(m),
		"a different intervening event starts a new run")
	assert.Equal(t, 4, m.logs.len())
}

func logMessages(m *Model) []string {
	rows := m.logsTable.Rows()
	messages := make([]string, 0, len(rows))
	for _, row := range rows {
		messages = append(messages, row[3].String())
	}

	return messages
}
