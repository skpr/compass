package extension

import (
	"context"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/skpr/compass/tracing/cli/app/events"
	applogger "github.com/skpr/compass/tracing/cli/app/logger"
	"github.com/skpr/compass/tracing/collector"
	"github.com/skpr/compass/tracing/trace"
)

// New for sending traces to the program.
type Sink struct {
	p *tea.Program
}

// NewSink for sending traces to the program.
func NewSink(p *tea.Program) *Sink {
	return &Sink{p}
}

// Initialize the plugin.
func (s *Sink) Initialize() error {
	return nil
}

// ProcessTrace which has been collected.
func (s *Sink) ProcessTrace(ctx context.Context, trace trace.Trace) error {
	s.p.Send(events.Trace{
		IngestionTime: time.Now(),
		Trace:         trace,
	})

	return nil
}

// Start tracing from a file extension and send traces to the program.
func Start(ctx context.Context, logger *applogger.Logger, p *tea.Program, path string) error {
	return collector.Run(ctx, logger, NewSink(p), collector.RunOptions{
		ExecutablePath: path,
	})
}
