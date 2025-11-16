package http

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/skpr/compass/tracing/cli/app/events"
	applogger "github.com/skpr/compass/tracing/cli/app/logger"
	"github.com/skpr/compass/tracing/trace"
)

// Start tracing from a http URI endpoint and send traces to the program.
func Start(ctx context.Context, logger *applogger.Logger, p *tea.Program, uri string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, uri, nil)
	if err != nil {
		return err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bad status code: %d", resp.StatusCode)
	}

	scanner := bufio.NewScanner(resp.Body)

	// Start with a 64KB initial buffer, allow up to, say, 10MB per line.
	const maxLine = 10 * 1024 * 1024
	buf := make([]byte, 64*1024)
	scanner.Buffer(buf, maxLine)

	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		line := scanner.Bytes()

		var trace trace.Trace

		if err := json.Unmarshal(line, &trace); err != nil {
			logger.Error("failed to parse trace (json)", "error", err)
			continue
		}

		p.Send(events.Trace{
			IngestionTime: time.Now(),
			Trace:         trace,
		})
	}

	return scanner.Err()
}
