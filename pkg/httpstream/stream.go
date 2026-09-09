// Package httpstream serves a channel of traces to an HTTP client as
// newline-delimited JSON, bounding each write with a deadline so a client that
// stops reading cannot pin the serving goroutine and its connection forever.
package httpstream

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/skpr/compass/pkg/trace"
)

// DefaultWriteTimeout bounds a single write to a streaming client. The whole
// stream is long lived, so the server sets no WriteTimeout; instead each
// individual Encode/Flush is given this deadline. A client which is connected
// but not draining its socket then fails a write once its send buffer fills,
// rather than blocking the serving goroutine indefinitely.
const DefaultWriteTimeout = 30 * time.Second

// Serve streams traces to the client until the request context is cancelled,
// the source channel is closed, or a write exceeds writeTimeout. It writes the
// NDJSON response headers itself and flushes after every record.
//
// writeTimeout <= 0 uses DefaultWriteTimeout.
func Serve(w http.ResponseWriter, r *http.Request, traces <-chan trace.Trace, writeTimeout time.Duration, logger *slog.Logger, logArgs ...any) {
	if writeTimeout <= 0 {
		writeTimeout = DefaultWriteTimeout
	}

	// The stream is newline-delimited JSON: one trace object per line. It is not
	// SSE, so it is labelled as NDJSON rather than text/event-stream.
	w.Header().Set("Content-Type", "application/x-ndjson")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)

	rc := http.NewResponseController(w)
	enc := json.NewEncoder(w)
	ctx := r.Context()

	for {
		select {
		case <-ctx.Done():
			logger.Info("Client disconnected", logArgs...)
			return
		case msg, ok := <-traces:
			if !ok {
				logger.Info("Subscriber channel closed", logArgs...)
				return
			}

			// Bound this single write. A writer which does not support deadlines
			// (e.g. a test recorder) reports ErrNotSupported; there is nothing to
			// arm in that case, so carry on without the protection rather than
			// dropping the stream.
			if err := rc.SetWriteDeadline(time.Now().Add(writeTimeout)); err != nil && !errors.Is(err, http.ErrNotSupported) {
				logger.Error("Failed to set write deadline", append([]any{"error", err}, logArgs...)...)
				return
			}

			if err := enc.Encode(msg); err != nil {
				// A cancelled client is a normal disconnect, not an error. A write
				// which exceeded the deadline surfaces here too: the client was not
				// reading, so the connection is torn down and the goroutine returns.
				if errors.Is(ctx.Err(), context.Canceled) {
					logger.Info("Client write failed due to context cancellation", logArgs...)
					return
				}

				logger.Error("Failed to write to client", append([]any{"error", err}, logArgs...)...)
				return
			}

			if err := rc.Flush(); err != nil && !errors.Is(err, http.ErrNotSupported) {
				if errors.Is(ctx.Err(), context.Canceled) {
					return
				}

				logger.Error("Failed to flush to client", append([]any{"error", err}, logArgs...)...)
				return
			}
		}
	}
}
