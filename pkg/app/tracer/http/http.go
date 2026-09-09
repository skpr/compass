// Package http streams traces from a Compass sidecar over HTTP(S).
package http

import (
	"bufio"
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/skpr/compass/pkg/app/events"
	"github.com/skpr/compass/pkg/trace"
)

// HeaderToken is the header the sidecar authenticates requests with.
const HeaderToken = "X-Skpr-Token"

// maxLineBytes is the largest trace payload supported by the stream client. A
// trace larger than this cannot be carried, so it is dropped and the stream
// continues rather than being torn down. A var so tests can shrink it.
var maxLineBytes = 10 * 1024 * 1024

// Backoff configuration for reconnecting to the sidecar, variables so that
// tests do not have to wait for real world delays.
var (
	backoffInitial = time.Second
	backoffMax     = 30 * time.Second
)

// Sender receives traces and connection updates, implemented by tea.Program.
type Sender interface {
	Send(msg tea.Msg)
}

// Logger for reporting stream failures to the application.
type Logger interface {
	Info(msg string, args ...any)
	Error(msg string, args ...any)
}

// Config for connecting to the sidecar.
type Config struct {
	// URI of the trace stream.
	URI string
	// Token sent to the sidecar for authentication.
	Token string
	// CAFile containing the certificate authority which signed the sidecar certificate.
	CAFile string
	// InsecureSkipVerify disables verification of the sidecar certificate.
	InsecureSkipVerify bool
}

// ErrUnauthorized is returned when the sidecar rejects our credentials.
var ErrUnauthorized = errors.New("sidecar rejected the request, set --token to the value of COMPASS_SIDECAR_TOKEN")

// Start streaming traces from a sidecar and send them to the program.
//
// The sidecar restarts, gets redeployed and is often started after the CLI, so
// the connection is retried with a backoff until the context is cancelled.
func Start(ctx context.Context, logger Logger, p Sender, config Config) error {
	client, err := newClient(config)
	if err != nil {
		return err
	}

	backoff := backoffInitial

	for {
		if err := ctx.Err(); err != nil {
			return err
		}

		p.Send(events.Connection{State: events.ConnectionStateConnecting})

		connected, err := stream(ctx, logger, p, client, config)

		// Credentials will not fix themselves, so surface this immediately.
		if errors.Is(err, ErrUnauthorized) {
			return err
		}

		switch {
		case err == nil, errors.Is(err, context.Canceled):
			// Context cancelled, the application is shutting down.
			if ctxErr := ctx.Err(); ctxErr != nil {
				return ctxErr
			}

			logger.Info("trace stream closed, reconnecting")
		default:
			logger.Error("trace stream failed", "error", err)
		}

		p.Send(events.Connection{State: events.ConnectionStateRetrying, Err: err})

		// A stream which worked before is a healthy sidecar that went away, so
		// start over from the shortest delay rather than the last one we reached.
		if connected {
			backoff = backoffInitial
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(backoff):
		}

		// Back off up to a ceiling so a sidecar which is down does not get hammered.
		if backoff < backoffMax {
			backoff *= 2
			if backoff > backoffMax {
				backoff = backoffMax
			}
		}
	}
}

// stream traces from the sidecar until the connection ends. The first return
// value reports whether the stream was established, so the caller knows the
// difference between a sidecar which went away and one which never answered.
func stream(ctx context.Context, logger Logger, p Sender, client *http.Client, config Config) (bool, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, config.URI, nil)
	if err != nil {
		return false, err
	}

	if config.Token != "" {
		req.Header.Set(HeaderToken, config.Token)
	}

	resp, err := client.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
		// Connected.
	case http.StatusUnauthorized, http.StatusForbidden:
		return false, ErrUnauthorized
	default:
		return false, fmt.Errorf("bad status code: %d", resp.StatusCode)
	}

	p.Send(events.Connection{State: events.ConnectionStateConnected})

	// A bufio.Scanner would return bufio.ErrTooLong on a single line larger than
	// maxLineBytes, ending the whole stream and dropping every trace buffered
	// behind it. Read line by line instead so an oversized trace is skipped on
	// its own and the stream continues.
	reader := bufio.NewReaderSize(resp.Body, 64*1024)

	for {
		select {
		case <-ctx.Done():
			return true, ctx.Err()
		default:
		}

		line, tooLong, err := readLine(reader, maxLineBytes)

		if tooLong {
			// The trace could not fit the transport. Dropping this one keeps the
			// stream alive rather than tearing it down and reconnecting into the
			// same oversized trace. Raising the sidecar function-call limit beyond
			// what the transport carries is what produces this.
			logger.Error("dropped a trace larger than the transport line limit", "limit_bytes", maxLineBytes)
		}

		if err != nil {
			if errors.Is(err, io.EOF) {
				// Clean end of stream; any trailing partial line is incomplete and
				// is discarded.
				return true, nil
			}

			return true, err
		}

		if tooLong {
			continue
		}

		line = bytes.TrimRight(line, "\r\n")
		if len(line) == 0 {
			continue
		}

		var tr trace.Trace

		if err := json.Unmarshal(line, &tr); err != nil {
			logger.Error("failed to parse trace (json)", "error", err)
			continue
		}

		p.Send(events.Trace{
			IngestionTime: time.Now(),
			Trace:         tr,
		})
	}
}

// readLine reads one newline-terminated line from r. When the line would exceed
// max bytes it is drained to its end and discarded, and tooLong is true with a
// nil line, so a single oversized record never grows the client's memory or
// stops the stream. err is io.EOF at the end of the stream, possibly alongside
// a final unterminated line.
func readLine(r *bufio.Reader, max int) (line []byte, tooLong bool, err error) {
	for {
		frag, e := r.ReadSlice('\n')

		if len(frag) > 0 && !tooLong {
			if len(line)+len(frag) > max {
				// Over the limit: stop accumulating and release what we held.
				tooLong = true
				line = nil
			} else {
				// frag aliases the reader's buffer, so copy it out.
				line = append(line, frag...)
			}
		}

		switch {
		case e == nil:
			// Reached the newline: a complete line.
			return line, tooLong, nil
		case errors.Is(e, bufio.ErrBufferFull):
			// More of this line remains beyond the reader's buffer; keep reading.
			continue
		default:
			// io.EOF or a read error, with whatever we accumulated so far.
			return line, tooLong, e
		}
	}
}

// newClient for connecting to the sidecar, configured for TLS if required.
func newClient(config Config) (*http.Client, error) {
	transport := http.DefaultTransport.(*http.Transport).Clone()

	tlsConfig := &tls.Config{
		MinVersion:         tls.VersionTLS12,
		InsecureSkipVerify: config.InsecureSkipVerify,
	}

	if config.CAFile != "" {
		pem, err := os.ReadFile(config.CAFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read ca file: %w", err)
		}

		pool := x509.NewCertPool()

		if !pool.AppendCertsFromPEM(pem) {
			return nil, fmt.Errorf("failed to load certificates from ca file: %s", config.CAFile)
		}

		tlsConfig.RootCAs = pool
	}

	transport.TLSClientConfig = tlsConfig

	return &http.Client{
		Transport: transport,
		// The trace stream is long lived, so we cannot set a client timeout.
	}, nil
}
