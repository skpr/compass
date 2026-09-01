package http

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/skpr/compass/pkg/app/events"
	"github.com/skpr/compass/pkg/trace"
	"github.com/skpr/compass/pkg/tracer/functioncalls"
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

func TestDefaultFunctionCallLimitFitsTransport(t *testing.T) {
	const maxInt64 = int64(^uint64(0) >> 1)

	tr := trace.Trace{
		Metadata: trace.Metadata{
			ID: "ffffffffffffffffffffffffffffffff",
			HTTP: trace.MetadataHTTP{
				Method: strings.Repeat("M", 100),
				URI:    strings.Repeat("u", 2_000),
			},
		},
		FunctionCalls: make([]trace.FunctionCall, functioncalls.DefaultMax),
		Drupal: &trace.Drupal{
			CacheEvents: make([]trace.CacheEvent, 250),
		},
	}

	for i := range tr.FunctionCalls {
		tr.FunctionCalls[i] = trace.FunctionCall{
			Name:    strings.Repeat("f", 100),
			Offset:  time.Duration(maxInt64),
			Elapsed: time.Duration(maxInt64),
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
