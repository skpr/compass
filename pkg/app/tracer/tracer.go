package tracer

import (
	"context"
	"fmt"
	"net/url"

	tea "github.com/charmbracelet/bubbletea"

	applogger "github.com/skpr/compass/pkg/app/logger"
	"github.com/skpr/compass/pkg/app/tracer/extension"
	"github.com/skpr/compass/pkg/app/tracer/http"
)

// Protocol for connecting to Compass for traces.
type Protocol string

const (
	// ProtocolHTTP for connecting to the Compass sidecar over HTTP.
	ProtocolHTTP = "http"
	// ProtocolHTTPS for connecting to the Compass sidecar over HTTPS.
	ProtocolHTTPS = "https"
	// ProtocolExtension for connecting directly to the Compass extension.
	ProtocolExtension = "extension"
)

// Config for connecting to a source of traces.
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

// Start tracing and send traces to the program.
func Start(ctx context.Context, logger *applogger.Logger, p *tea.Program, config Config) error {
	u, err := url.Parse(config.URI)
	if err != nil {
		return fmt.Errorf("failed to parse uri: %w", err)
	}

	switch u.Scheme {
	case ProtocolHTTPS, ProtocolHTTP:
		return http.Start(ctx, logger, p, http.Config{
			URI:                config.URI,
			Token:              config.Token,
			CAFile:             config.CAFile,
			InsecureSkipVerify: config.InsecureSkipVerify,
		})

	case ProtocolExtension:
		return extension.Start(ctx, logger, p, u.Path)

	default:
		return fmt.Errorf("unsupported scheme: %q", u.Scheme)
	}
}
