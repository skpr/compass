package httpstream

import (
	"bufio"
	"bytes"
	"compress/gzip"
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

// A trace is mostly the same function names repeated across the request, so a
// client which accepts gzip is served the stream compressed.
func TestServe_CompressesForAClientWhichAcceptsIt(t *testing.T) {
	big := trace.Trace{Metadata: trace.Metadata{ID: "compressed"}}
	for i := 0; i < 2000; i++ {
		big.Spans = append(big.Spans, trace.Span{Name: "Drupal\\Core\\Entity\\Sql\\SqlContentEntityStorage::loadMultiple", Calls: 1})
	}

	traces := make(chan trace.Trace, 1)
	traces <- big
	close(traces)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		Serve(w, r, traces, time.Second, testLogger())
	}))
	defer srv.Close()

	req, err := http.NewRequest(http.MethodGet, srv.URL, nil)
	require.NoError(t, err)

	// Set by hand: the transport adds and undoes this on our behalf otherwise,
	// which would hide what was on the wire.
	req.Header.Set("Accept-Encoding", "gzip")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, "gzip", resp.Header.Get("Content-Encoding"))
	assert.Equal(t, "application/x-ndjson", resp.Header.Get("Content-Type"))
	assert.Equal(t, "Accept-Encoding", resp.Header.Get("Vary"))

	compressed, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	gz, err := gzip.NewReader(bytes.NewReader(compressed))
	require.NoError(t, err)

	decompressed, err := io.ReadAll(gz)
	require.NoError(t, err)

	var got trace.Trace
	require.NoError(t, json.Unmarshal(bytes.TrimSpace(decompressed), &got))

	assert.Equal(t, "compressed", got.Metadata.ID)
	assert.Len(t, got.Spans, 2000)

	assert.Less(t, len(compressed), len(decompressed)/4,
		"compressed %d bytes against %d uncompressed", len(compressed), len(decompressed))
}

// A record has to reach the client as it is written, rather than sitting in the
// compressor waiting for the one after it: a trace which arrives a minute later
// is not a trace which arrived.
func TestServe_CompressedRecordsArriveOneAtATime(t *testing.T) {
	traces := make(chan trace.Trace)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		Serve(w, r, traces, time.Second, testLogger())
	}))
	defer srv.Close()

	req, err := http.NewRequest(http.MethodGet, srv.URL, nil)
	require.NoError(t, err)
	req.Header.Set("Accept-Encoding", "gzip")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, "gzip", resp.Header.Get("Content-Encoding"))

	// The stream header is flushed when the sidecar answers, so this returns
	// before any trace has been written.
	gz, err := gzip.NewReader(resp.Body)
	require.NoError(t, err)

	scanner := bufio.NewScanner(gz)

	for _, id := range []string{"a", "b"} {
		traces <- trace.Trace{Metadata: trace.Metadata{ID: id}}

		require.True(t, scanner.Scan(), "record %q did not arrive on its own", id)

		var got trace.Trace
		require.NoError(t, json.Unmarshal(scanner.Bytes(), &got))
		assert.Equal(t, id, got.Metadata.ID)
	}

	close(traces)
}

// A client which says nothing about encodings, or refuses gzip, is served what
// it was served before.
func TestServe_UncompressedByDefault(t *testing.T) {
	// The default transport asks for gzip on behalf of a request which names no
	// encoding, and then undoes it before the response is returned, which would
	// let a compressed answer pass for an uncompressed one here.
	client := &http.Client{Transport: &http.Transport{DisableCompression: true}}

	for name, accept := range map[string]string{
		"no header":     "",
		"refused":       "gzip;q=0",
		"another codec": "br",
		"refused via *": "*;q=0",
	} {
		t.Run(name, func(t *testing.T) {
			traces := make(chan trace.Trace, 1)
			traces <- trace.Trace{Metadata: trace.Metadata{ID: "plain"}}
			close(traces)

			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				Serve(w, r, traces, time.Second, testLogger())
			}))
			defer srv.Close()

			req, err := http.NewRequest(http.MethodGet, srv.URL, nil)
			require.NoError(t, err)

			if accept != "" {
				req.Header.Set("Accept-Encoding", accept)
			}

			resp, err := client.Do(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Empty(t, resp.Header.Get("Content-Encoding"))

			body, err := io.ReadAll(resp.Body)
			require.NoError(t, err)

			var got trace.Trace
			require.NoError(t, json.Unmarshal(bytes.TrimSpace(body), &got))
			assert.Equal(t, "plain", got.Metadata.ID)
		})
	}
}

func TestAcceptsGzip(t *testing.T) {
	for header, want := range map[string]bool{
		"":                      false,
		"gzip":                  true,
		"GZIP":                  true,
		" gzip , deflate":       true,
		"gzip;q=0.5":            true,
		"gzip;q=0":              false,
		"identity;q=1, gzip":    true,
		"deflate, br":           false,
		"*":                     true,
		"*;q=0":                 false,
		"*;q=0, gzip":           true,
		"gzip;q=0, *":           false,
		"deflate;q=1, gzip;q=0": false,
	} {
		assert.Equal(t, want, acceptsGzip(header), "Accept-Encoding: %q", header)
	}
}
