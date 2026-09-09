package http

import (
	"bufio"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/skpr/compass/pkg/trace"
)

func TestReadLine(t *testing.T) {
	const max = 16

	t.Run("normal line", func(t *testing.T) {
		r := bufio.NewReaderSize(strings.NewReader("hello\nworld\n"), 8)

		line, tooLong, err := readLine(r, max)
		require.NoError(t, err)
		assert.False(t, tooLong)
		assert.Equal(t, "hello\n", string(line))
	})

	t.Run("oversized line is drained and skipped", func(t *testing.T) {
		// A line well over max, followed by a normal one.
		r := bufio.NewReaderSize(strings.NewReader(strings.Repeat("x", 100)+"\nok\n"), 8)

		line, tooLong, err := readLine(r, max)
		require.NoError(t, err)
		assert.True(t, tooLong, "a line longer than max must be reported as too long")
		assert.Nil(t, line, "an oversized line must not be retained")

		// The following line is unaffected.
		line, tooLong, err = readLine(r, max)
		require.NoError(t, err)
		assert.False(t, tooLong)
		assert.Equal(t, "ok\n", string(line))
	})

	t.Run("line spanning several buffer refills", func(t *testing.T) {
		payload := strings.Repeat("a", 30) // > buffer (8), < max (fits after trim? 30 > 16)
		r := bufio.NewReaderSize(strings.NewReader(payload+"\n"), 8)

		_, tooLong, err := readLine(r, max)
		require.NoError(t, err)
		assert.True(t, tooLong, "30 bytes exceeds max 16 even though it spans refills")
	})

	t.Run("trailing partial line at EOF", func(t *testing.T) {
		r := bufio.NewReaderSize(strings.NewReader("partial"), 8)

		line, tooLong, err := readLine(r, max)
		require.ErrorIs(t, err, io.EOF)
		assert.False(t, tooLong)
		assert.Equal(t, "partial", string(line))
	})
}

// TestStream_SkipsOversizedTraceAndContinues is the end-to-end property: a
// trace too large for the transport is dropped, but the traces around it still
// reach the application and the stream is not torn down.
func TestStream_SkipsOversizedTraceAndContinues(t *testing.T) {
	original := maxLineBytes
	maxLineBytes = 1024
	defer func() { maxLineBytes = original }()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeTrace(t, w, "first")

		// An oversized trace: a URI larger than the shrunk maxLineBytes.
		big := trace.Trace{Metadata: trace.Metadata{ID: "big", HTTP: trace.MetadataHTTP{URI: strings.Repeat("u", 4096)}}}
		require.NoError(t, json.NewEncoder(w).Encode(big))
		w.(http.Flusher).Flush()

		writeTrace(t, w, "second")

		<-r.Context().Done()
	}))
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	s := &sender{}
	done := make(chan error, 1)
	go func() { done <- Start(ctx, logger{}, s, Config{URI: server.URL}) }()

	require.Eventually(t, func() bool {
		var ids []string
		for _, tr := range s.traces() {
			ids = append(ids, tr.Metadata.ID)
		}
		return len(ids) == 2 && ids[0] == "first" && ids[1] == "second"
	}, 10*time.Second, 10*time.Millisecond, "both normal traces should arrive with the oversized one skipped")

	cancel()
	<-done
}
