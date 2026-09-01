package extension

import (
	"context"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/skpr/compass/pkg/app/events"
	applogger "github.com/skpr/compass/pkg/app/logger"
	"github.com/skpr/compass/pkg/trace"
	"github.com/skpr/compass/pkg/tracer/functioncalls"
	"github.com/skpr/compass/pkg/tracer/php"
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
func (s *Sink) ProcessTrace(ctx context.Context, tr trace.Trace) error {
	s.p.Send(events.Trace{
		IngestionTime: time.Now(),
		Trace:         tr,
	})

	return nil
}

// Start tracing from a file extension and send traces to the program.
func Start(ctx context.Context, logger *applogger.Logger, p *tea.Program, path string) error {
	return php.Run(ctx, NewSink(p), path, functioncalls.DefaultMax)
}
