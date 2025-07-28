// Package sink implements a simple sink that stores traces in the CLI application.
package sink

import (
	"context"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/skpr/compass/cli/app/types"
	"github.com/skpr/compass/trace"
)

// New client for handling traces to stdout.
func New(p *tea.Program) *Client {
	return &Client{
		p: p,
	}
}

// Client for handling traces to stdout.
type Client struct {
	p *tea.Program
}

// Initialize the plugin.
func (c *Client) Initialize() error {
	return nil
}

// ProcessTrace from the collector.
func (c *Client) ProcessTrace(_ context.Context, t trace.Trace) error {
	trace := types.Trace{
		IngestionTime: time.Now(),
		Trace:         t,
	}

	c.p.Send(trace)
	return nil
}
