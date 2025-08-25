package tracer

import (
	"context"
	"fmt"
	"net/url"

	tea "github.com/charmbracelet/bubbletea"

	applogger "github.com/skpr/compass/cli/app/logger"
	"github.com/skpr/compass/cli/app/tracer/extension"
	"github.com/skpr/compass/cli/app/tracer/http"
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

// Start tracing and send traces to the program.
func Start(ctx context.Context, logger *applogger.Logger, p *tea.Program, uri string) error {
	u, err := url.Parse(uri)
	if err != nil {
		return fmt.Errorf("failed to parse uri: %w", err)
	}

	switch u.Scheme {
	case ProtocolHTTPS, ProtocolHTTP:
		return http.Start(ctx, logger, p, uri)

	case ProtocolExtension:
		return extension.Start(ctx, logger, p, u.Path)

	default:
		return fmt.Errorf("unsupported scheme: %q", u.Scheme)
	}
}
