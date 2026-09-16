package http

import (
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
	"unsafe"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/skpr/compass/pkg/app/events"
	"github.com/skpr/compass/pkg/httpstream"
	"github.com/skpr/compass/pkg/trace"
	"github.com/skpr/compass/pkg/tracer/spans"
)

// sender collects the messages which would have been sent to the application.
type sender struct {
	mu   sync.Mutex
	msgs []tea.Msg
}

func (s *sender) Send(msg tea.Msg) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.msgs = append(s.msgs, msg)
}

func (s *sender) traces() []events.Trace {
	s.mu.Lock()
	defer s.mu.Unlock()

	var traces []events.Trace

	for _, msg := range s.msgs {
		if t, ok := msg.(events.Trace); ok {
			traces = append(traces, t)
		}
	}

	return traces
}

func (s *sender) states() []events.ConnectionState {
	s.mu.Lock()
	defer s.mu.Unlock()

	var states []events.ConnectionState

	for _, msg := range s.msgs {
		if c, ok := msg.(events.Connection); ok {
			states = append(states, c.State)
		}
	}

	return states
}

// logger which discards messages.
type logger struct{}

func (logger) Info(_ string, _ ...any)  {}
func (logger) Error(_ string, _ ...any) {}

func writeTrace(t *testing.T, w http.ResponseWriter, id string) {
	t.Helper()

	require.NoError(t, json.NewEncoder(w).Encode(trace.Trace{
		Metadata: trace.Metadata{ID: id},
	}))

	w.(http.Flusher).Flush()
}

func TestStart_Unauthorized(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer server.Close()

	s := &sender{}

	// Authentication failures are terminal, we do not want to retry them.
	err := Start(context.Background(), logger{}, s, Config{URI: server.URL})
	assert.ErrorIs(t, err, ErrUnauthorized)
	assert.Empty(t, s.traces())
}

func TestStart_SendsToken(t *testing.T) {
	tokens := make(chan string, 1)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tokens <- r.Header.Get(HeaderToken)
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer server.Close()

	_ = Start(context.Background(), logger{}, &sender{}, Config{URI: server.URL, Token: "xxxyyyzzz"})

	assert.Equal(t, "xxxyyyzzz", <-tokens)
}

func TestStart_Reconnects(t *testing.T) {
	// Keep the test quick, the connection is retried immediately.
	original := backoffInitial
	backoffInitial = time.Millisecond
	defer func() { backoffInitial = original }()

	var attempts atomic.Int64

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if attempts.Add(1) == 1 {
			// The sidecar goes away mid stream, eg. it was restarted.
			writeTrace(t, w, "first")
			return
		}

		writeTrace(t, w, "second")

		<-r.Context().Done()
	}))
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	s := &sender{}

	done := make(chan error, 1)

	go func() {
		done <- Start(ctx, logger{}, s, Config{URI: server.URL})
	}()

	// Wait for the trace which was sent after the stream was re-established.
	require.Eventually(t, func() bool {
		select {
		case err := <-done:
			t.Fatalf("stream exited before reconnecting: %v", err)
		default:
		}

		return len(s.traces()) == 2
	}, 10*time.Second, 10*time.Millisecond)

	cancel()
	require.ErrorIs(t, <-done, context.Canceled)

	var ids []string
	for _, tr := range s.traces() {
		ids = append(ids, tr.Metadata.ID)
	}

	assert.Equal(t, []string{"first", "second"}, ids)

	// The application is told about every state change.
	assert.Equal(t, []events.ConnectionState{
		events.ConnectionStateConnecting,
		events.ConnectionStateConnected,
		events.ConnectionStateRetrying,
		events.ConnectionStateConnecting,
		events.ConnectionStateConnected,
	}, s.states())
}

// A trace carrying the maximum spans, each as large as the probes can make
// one, still has to fit the line the stream is read from.
func TestDefaultSpanLimitFitsTransport(t *testing.T) {
	const maxInt64 = int64(^uint64(0) >> 1)

	tr := trace.Trace{
		Metadata: trace.Metadata{
			ID: "ffffffffffffffffffffffffffffffff",
			HTTP: trace.MetadataHTTP{
				Method: strings.Repeat("M", 100),
				URI:    strings.Repeat("u", 2_000),
			},
		},
		Spans: make([]trace.Span, spans.DefaultMax),
		Calls: maxInt64,
		Drupal: &trace.Drupal{
			CacheEvents: make([]trace.CacheEvent, 250),
		},
	}

	for i := range tr.Spans {
		tr.Spans[i] = trace.Span{
			Name:    strings.Repeat("f", 100),
			Offset:  time.Duration(maxInt64),
			Elapsed: time.Duration(maxInt64),
			Total:   time.Duration(maxInt64),
			Calls:   maxInt64,
			Memory:  maxInt64,
		}
	}

	for i := range tr.Drupal.CacheEvents {
		tr.Drupal.CacheEvents[i] = trace.CacheEvent{
			Caller:     strings.Repeat("c", 255),
			ObjectType: strings.Repeat("o", 255),
			Tags:       []string{strings.Repeat("t", 1_024)},
			Contexts:   []string{strings.Repeat("x", 512)},
			MaxAge:     maxInt64,
			Offset:     time.Duration(maxInt64),
			Calls:      maxInt64,
		}
	}

	encoded, err := json.Marshal(tr)
	require.NoError(t, err)
	assert.Less(t, len(encoded)+1, maxLineBytes, "JSON line is %d bytes", len(encoded)+1)
}

// The client asks for gzip, and reads a compressed stream a record at a time.
func TestStart_ReadsACompressedStream(t *testing.T) {
	accepts := make(chan string, 1)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		accepts <- r.Header.Get("Accept-Encoding")

		w.Header().Set("Content-Type", "application/x-ndjson")
		w.Header().Set("Content-Encoding", "gzip")
		w.WriteHeader(http.StatusOK)

		gz := gzip.NewWriter(w)
		defer gz.Close()

		for _, id := range []string{"first", "second"} {
			require.NoError(t, json.NewEncoder(gz).Encode(trace.Trace{
				Metadata: trace.Metadata{ID: id},
			}))
			require.NoError(t, gz.Flush())
			w.(http.Flusher).Flush()
		}

		<-r.Context().Done()
	}))
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	s := &sender{}

	done := make(chan error, 1)
	go func() { done <- Start(ctx, logger{}, s, Config{URI: server.URL}) }()

	require.Eventually(t, func() bool {
		return len(s.traces()) == 2
	}, 10*time.Second, 10*time.Millisecond)

	cancel()
	require.ErrorIs(t, <-done, context.Canceled)

	assert.Equal(t, "gzip", <-accepts)

	var ids []string
	for _, tr := range s.traces() {
		ids = append(ids, tr.Metadata.ID)
	}

	assert.Equal(t, []string{"first", "second"}, ids)
}

// The same function named by two traces is one string once they are retained:
// the decoder makes a new one for every span, which is about half of what a
// trace weighs.
func TestStart_SharesNamesAcrossTraces(t *testing.T) {
	const name = "Drupal\\Core\\Entity\\Sql\\SqlContentEntityStorage::loadMultiple"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		for _, id := range []string{"first", "second"} {
			require.NoError(t, json.NewEncoder(w).Encode(trace.Trace{
				Metadata: trace.Metadata{ID: id},
				Spans: []trace.Span{
					{Name: name, Calls: 1},
					{Name: name, Calls: 2},
				},
				Drupal: &trace.Drupal{
					CacheEvents: []trace.CacheEvent{
						{Caller: name, Tags: []string{"node:1"}, Contexts: []string{"user.permissions"}},
						{Caller: name, Tags: []string{"node:1"}, Contexts: []string{"user.permissions"}},
					},
				},
			}))
			w.(http.Flusher).Flush()
		}

		<-r.Context().Done()
	}))
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	s := &sender{}

	done := make(chan error, 1)
	go func() { done <- Start(ctx, logger{}, s, Config{URI: server.URL}) }()

	require.Eventually(t, func() bool {
		return len(s.traces()) == 2
	}, 10*time.Second, 10*time.Millisecond)

	cancel()
	require.ErrorIs(t, <-done, context.Canceled)

	traces := s.traces()

	// Same characters is what the decoder already gave us; the same string is
	// the point, so the retained traces are compared by what they point at.
	first := unsafe.StringData(traces[0].Spans[0].Name)

	for _, tr := range traces {
		for _, span := range tr.Spans {
			assert.Same(t, first, unsafe.StringData(span.Name), "span name was not shared")
		}

		for _, event := range tr.Drupal.CacheEvents {
			assert.Same(t, first, unsafe.StringData(event.Caller), "caller was not shared")
			assert.Same(t, unsafe.StringData(traces[0].Drupal.CacheEvents[0].Tags[0]), unsafe.StringData(event.Tags[0]), "tag was not shared")
			assert.Same(t, unsafe.StringData(traces[0].Drupal.CacheEvents[0].Contexts[0]), unsafe.StringData(event.Contexts[0]), "context was not shared")
		}
	}
}

// The names come from the application rather than from us, so a program which
// generates them cannot grow the map for as long as the CLI runs.
func TestInterner_Bounded(t *testing.T) {
	i := newInterner()

	for n := range MaxInternedNames + 100 {
		i.intern(fmt.Sprintf("function-%d", n))
	}

	assert.Len(t, i.strings, MaxInternedNames)

	// Past the bound a name is still carried, it is just not shared.
	assert.Equal(t, "function-0", i.intern("function-0"))
	assert.Equal(t, "beyond-the-bound", i.intern("beyond-the-bound"))
}

// A trace with no Drupal data is the common case: PHP CLI, Node, and any PHP
// application which is not Drupal.
func TestInterner_TraceWithoutDrupal(t *testing.T) {
	i := newInterner()

	tr := trace.Trace{Spans: []trace.Span{{Name: "one"}, {Name: "one"}}}
	i.trace(&tr)

	assert.Same(t, unsafe.StringData(tr.Spans[0].Name), unsafe.StringData(tr.Spans[1].Name))
}

// The client and the sidecar's stream, together: the traces are compressed on
// the wire and arrive sharing the strings they name.
func TestStart_AgainstTheSidecarStream(t *testing.T) {
	const name = "Drupal\\Core\\Entity\\Sql\\SqlContentEntityStorage::loadMultiple"

	traces := make(chan trace.Trace, 2)

	for _, id := range []string{"first", "second"} {
		tr := trace.Trace{Metadata: trace.Metadata{ID: id}}
		for range 500 {
			tr.Spans = append(tr.Spans, trace.Span{Name: name, Calls: 1})
		}

		traces <- tr
	}

	encodings := make(chan string, 1)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		httpstream.Serve(w, r, traces, time.Second, slog.New(slog.NewTextHandler(io.Discard, nil)))
		encodings <- w.Header().Get("Content-Encoding")
	}))
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	s := &sender{}

	done := make(chan error, 1)
	go func() { done <- Start(ctx, logger{}, s, Config{URI: server.URL}) }()

	require.Eventually(t, func() bool {
		return len(s.traces()) == 2
	}, 10*time.Second, 10*time.Millisecond)

	cancel()
	require.ErrorIs(t, <-done, context.Canceled)

	assert.Equal(t, "gzip", <-encodings, "the stream was not compressed")

	shared := unsafe.StringData(s.traces()[0].Spans[0].Name)

	for _, tr := range s.traces() {
		require.Len(t, tr.Spans, 500)

		for _, span := range tr.Spans {
			require.Same(t, shared, unsafe.StringData(span.Name))
		}
	}
}
