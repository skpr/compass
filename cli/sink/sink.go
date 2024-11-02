// Package sink implements a simple sink that stores profiles in the CLI application.
package sink

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/skpr/compass/profile/aggregated"
	"github.com/skpr/compass/profile/complete"
)

// New client for handling profiles to stdout.
func New(p *tea.Program) *Client {
	return &Client{
		p: p,
	}
}

// Client for handling profiles to stdout.
type Client struct {
	p *tea.Program
}

// Initialize the plugin.
func (c *Client) Initialize() error {
	return nil
}

// ProcessProfile from the collector.
func (c *Client) ProcessProfile(completeProfile complete.Profile) error {
	profile := aggregated.FromCompleteProfile(completeProfile)
	c.p.Send(profile)
	return nil
}
