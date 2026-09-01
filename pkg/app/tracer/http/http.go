// Package http streams traces from a Compass sidecar over HTTP(S).
package http

import (
	"bufio"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/skpr/compass/pkg/app/events"
	"github.com/skpr/compass/pkg/trace"
)

// HeaderToken is the header the sidecar authenticates requests with.
const HeaderToken = "X-Skpr-Token"

// maxLineBytes is the largest trace payload supported by the stream client.
const maxLineBytes = 10 * 1024 * 1024

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

	scanner := bufio.NewScanner(resp.Body)

	// Start with a 64KB initial buffer and grow only for larger traces.
	buf := make([]byte, 64*1024)
	scanner.Buffer(buf, maxLineBytes)

	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return true, ctx.Err()
		default:
		}

		line := scanner.Bytes()

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

	return true, scanner.Err()
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
