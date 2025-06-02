// Package sink implements a simple sink that stores traces in the CLI application.
package sink

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/skpr/compass/cli/app/events"
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
func (c *Client) ProcessTrace(t trace.Trace) error {
	trace := events.Trace{
		IngestionTime: time.Now(),
		Trace:         t,
	}

	c.p.Send(trace)
	return nil
}
