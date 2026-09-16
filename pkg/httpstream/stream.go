// Package httpstream serves a channel of traces to an HTTP client as
// newline-delimited JSON, bounding each write with a deadline so a client that
// stops reading cannot pin the serving goroutine and its connection forever.
package httpstream

import (
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/skpr/compass/pkg/trace"
)

// DefaultWriteTimeout bounds a single write to a streaming client. The whole
// stream is long lived, so the server sets no WriteTimeout; instead each
// individual Encode/Flush is given this deadline. A client which is connected
// but not draining its socket then fails a write once its send buffer fills,
// rather than blocking the serving goroutine indefinitely.
const DefaultWriteTimeout = 30 * time.Second

// CompressionLevel used for a client which accepts gzip.
//
// A trace is mostly the same function names repeated across the slices of the
// request they were called in, so it compresses by about five times at this
// level. The levels above it are not worth what they cost here: on a 1.6 MB
// trace, level 1 took 5ms for 19% of the original, level 3 7ms for 15% and the
// default level 14ms for 13%. This runs in the sidecar, next to the
// application being traced, so the cheapest level which captures most of the
// win is the right trade.
const CompressionLevel = gzip.BestSpeed

// Serve streams traces to the client until the request context is cancelled,
// the source channel is closed, or a write exceeds writeTimeout. It writes the
// NDJSON response headers itself and flushes after every record.
//
// The stream is compressed when the client accepts gzip. Each record is
// flushed through the compressor as it is written, so compression does not
// delay a trace behind the one after it.
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
	w.Header().Set("Vary", "Accept-Encoding")

	var (
		out io.Writer = w
		gz  *gzip.Writer
	)

	if acceptsGzip(r.Header.Get("Accept-Encoding")) {
		w.Header().Set("Content-Encoding", "gzip")

		// The only error here is an invalid level, which is a constant above.
		gz, _ = gzip.NewWriterLevel(w, CompressionLevel)
		defer gz.Close()

		out = gz
	}

	w.WriteHeader(http.StatusOK)

	rc := http.NewResponseController(w)
	enc := json.NewEncoder(out)
	ctx := r.Context()

	// flush what has been written all the way to the socket. The compressor
	// holds a record until it is told to end a block, so it is flushed first.
	flush := func() error {
		if gz != nil {
			if err := gz.Flush(); err != nil {
				return err
			}
		}

		if err := rc.Flush(); err != nil && !errors.Is(err, http.ErrNotSupported) {
			return err
		}

		return nil
	}

	// Bound this single write. A writer which does not support deadlines (e.g. a
	// test recorder) reports ErrNotSupported; there is nothing to arm in that
	// case, so carry on without the protection rather than dropping the stream.
	arm := func() error {
		if err := rc.SetWriteDeadline(time.Now().Add(writeTimeout)); err != nil && !errors.Is(err, http.ErrNotSupported) {
			return err
		}

		return nil
	}

	// Send the response headers, and the gzip header with them, before waiting
	// for a trace. A client which has to read the compressed stream's header
	// before it can start decoding would otherwise sit through the wait for the
	// first request, unable to tell a connected sidecar from a silent one.
	if err := arm(); err != nil {
		logger.Error("Failed to set write deadline", append([]any{"error", err}, logArgs...)...)
		return
	}

	if err := flush(); err != nil {
		logger.Error("Failed to flush to client", append([]any{"error", err}, logArgs...)...)
		return
	}

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

			if err := arm(); err != nil {
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

			if err := flush(); err != nil {
				if errors.Is(ctx.Err(), context.Canceled) {
					return
				}

				logger.Error("Failed to flush to client", append([]any{"error", err}, logArgs...)...)
				return
			}
		}
	}
}

// acceptsGzip reports whether an Accept-Encoding header asks for gzip.
//
// A client which lists gzip with a quality of zero is refusing it rather than
// asking for it, and a wildcard accepts whatever we would like to send.
func acceptsGzip(header string) bool {
	// What a wildcard said, for a header which carries one and does not name
	// gzip itself. An explicit entry wins over it either way.
	var wildcard bool

	for _, part := range strings.Split(header, ",") {
		fields := strings.Split(part, ";")
		coding := strings.ToLower(strings.TrimSpace(fields[0]))

		if coding != "gzip" && coding != "*" {
			continue
		}

		// A coding listed with a quality of zero is refused rather than asked
		// for. Anything else, including an unparseable quality, is an ask.
		accepted := true

		for _, param := range fields[1:] {
			key, value, found := strings.Cut(param, "=")
			if !found || !strings.EqualFold(strings.TrimSpace(key), "q") {
				continue
			}

			if quality, err := strconv.ParseFloat(strings.TrimSpace(value), 64); err == nil && quality == 0 {
				accepted = false
			}
		}

		if coding == "gzip" {
			return accepted
		}

		wildcard = accepted
	}

	return wildcard
}
