package httpstream

import (
	"bufio"
	"encoding/json"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/skpr/compass/pkg/trace"
)

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

// TestServe_StreamsUntilChannelCloses is the happy path: every trace put on the
// channel reaches the client, and closing the channel ends the response.
func TestServe_StreamsUntilChannelCloses(t *testing.T) {
	traces := make(chan trace.Trace, 3)
	traces <- trace.Trace{Metadata: trace.Metadata{ID: "a"}}
	traces <- trace.Trace{Metadata: trace.Metadata{ID: "b"}}
	close(traces)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		Serve(w, r, traces, time.Second, testLogger())
	}))
	defer srv.Close()

	resp, err := http.Get(srv.URL)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, "application/x-ndjson", resp.Header.Get("Content-Type"))

	var ids []string
	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		var tr trace.Trace
		require.NoError(t, json.Unmarshal(scanner.Bytes(), &tr))
		ids = append(ids, tr.Metadata.ID)
	}

	assert.Equal(t, []string{"a", "b"}, ids)
}

// TestServe_ReturnsWhenClientDisconnects covers the request-context path: once
// the client's connection goes away, Serve returns rather than blocking on the
// channel forever.
func TestServe_ReturnsWhenClientDisconnects(t *testing.T) {
	traces := make(chan trace.Trace) // never sent to
	done := make(chan struct{})

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	srv := &http.Server{
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			Serve(w, r, traces, time.Second, testLogger())
			close(done)
		}),
		ReadHeaderTimeout: time.Second,
	}
	go func() { _ = srv.Serve(listener) }()
	defer srv.Close()

	conn, err := net.Dial("tcp", listener.Addr().String())
	require.NoError(t, err)

	_, err = conn.Write([]byte("GET /v1/traces HTTP/1.1\r\nHost: localhost\r\n\r\n"))
	require.NoError(t, err)

	// Give the server a moment to read the request and start the handler, then
	// drop the connection. The server observes the close and cancels
	// r.Context(), which Serve selects on.
	time.Sleep(100 * time.Millisecond)
	require.NoError(t, conn.Close())

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("Serve did not return after the client disconnected")
	}
}

// TestServe_DisconnectsNonReadingClient is the property the write deadline
// exists for: a client which establishes the stream and then stops reading must
// not pin the serving goroutine forever. With a short per-write deadline, Serve
// returns once the socket buffer fills and a write times out.
func TestServe_DisconnectsNonReadingClient(t *testing.T) {
	// A generously sized trace so the kernel/socket buffers fill in a bounded
	// number of writes rather than needing millions of tiny records.
	big := trace.Trace{Metadata: trace.Metadata{ID: "x"}}
	for i := 0; i < 5000; i++ {
		big.Spans = append(big.Spans, trace.Span{Name: "some/function/name/that/is/reasonably/long", Calls: 1})
	}

	traces := make(chan trace.Trace)
	done := make(chan struct{})

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	srv := &http.Server{
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			Serve(w, r, traces, 100*time.Millisecond, testLogger())
			close(done)
		}),
		ReadHeaderTimeout: time.Second,
	}
	go func() { _ = srv.Serve(listener) }()
	defer srv.Close()

	// Raw connection: send the request, read nothing from the body.
	conn, err := net.Dial("tcp", listener.Addr().String())
	require.NoError(t, err)
	defer conn.Close()

	_, err = conn.Write([]byte("GET /v1/traces HTTP/1.1\r\nHost: localhost\r\n\r\n"))
	require.NoError(t, err)

	// Feed traces until Serve returns. Because the client never reads, writes
	// will block and then fail the deadline; the send stops once done closes.
	go func() {
		for {
			select {
			case <-done:
				return
			case traces <- big:
			case <-time.After(50 * time.Millisecond):
			}
		}
	}()

	select {
	case <-done:
	case <-time.After(10 * time.Second):
		t.Fatal("Serve did not disconnect a non-reading client within the write deadline")
	}
}
