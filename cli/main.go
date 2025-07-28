// Package main for handling the main application.
package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/christgf/env"
	"github.com/jwalton/gchalk"
	"github.com/skpr/compass/trace"
	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"

	"github.com/skpr/compass/cli/app"
	"github.com/skpr/compass/cli/app/color"
	"github.com/skpr/compass/cli/app/events"
	applogger "github.com/skpr/compass/cli/app/logger"
)

const cmdExample = `
  # Watch and analyze new profiles.
  compass`

const cmdLong = `   _____ ____  __  __ _____         _____ _____
  / ____/ __ \|  \/  |  __ \ /\    / ____/ ____|
 | |   | |  | | \  / | |__) /  \  | (___| (___
 | |   | |  | | |\/| |  ___/ /\ \  \___ \\___ \
 | |___| |__| | |  | | |  / ____ \ ____) |___) |
  \_____\____/|_|  |_|_| /_/    \_\_____/_____/

A toolkit for pointing developers in the right direction for performance issues.`

// Options for the CLI.
type Options struct {
	URL string
}

func main() {
	o := Options{}

	cmd := &cobra.Command{
		Use:     "compass",
		Short:   "A toolkit for pointing developers in the right direction for performance issues.",
		Long:    cmdLong,
		Example: cmdExample,
		RunE: func(cmd *cobra.Command, _ []string) error {
			p := tea.NewProgram(app.NewModel(o.URL), tea.WithAltScreen())

			logger, err := applogger.New(p)
			if err != nil {
				return fmt.Errorf("failed to setup logger: %w", err)
			}

			ctx, cancel := context.WithCancel(cmd.Context())

			eg := errgroup.Group{}

			// Start the collector.
			eg.Go(func() error {
				req, err := http.NewRequestWithContext(ctx, http.MethodGet, o.URL, nil)
				if err != nil {
					logger.Error("failed to create request", "error", err)
					return err
				}

				resp, err := http.DefaultClient.Do(req)
				if err != nil {
					logger.Error("request failed", "error", err)
					return err
				}
				defer resp.Body.Close()

				if resp.StatusCode != http.StatusOK {
					logger.Error("bad status code", "code", resp.StatusCode)
					return fmt.Errorf("bad status code: %d", resp.StatusCode)
				}

				scanner := bufio.NewScanner(resp.Body)

				for scanner.Scan() {
					select {
					case <-ctx.Done():
						return ctx.Err()
					default:
					}

					line := scanner.Bytes()

					var trace trace.Trace

					if err := json.Unmarshal(line, &trace); err != nil {
						fmt.Fprintf(os.Stderr, "Failed to parse JSON: %v\n", err)
						continue
					}

					p.Send(events.Trace{
						IngestionTime: time.Now(),
						Trace:         trace,
					})
				}

				return scanner.Err()
			})

			// Start the application.
			eg.Go(func() error {
				_, err := p.Run()
				if err != nil {
					return fmt.Errorf("failed to run program: %w", err)
				}

				cancel()

				return nil
			})

			return eg.Wait()
		},
	}

	cmd.PersistentFlags().StringVar(&o.URL, "url", env.String("COMPASS_SIDECAR_URL", "http://localhost:28624/v1/traces"), "URL of the Compass sidecar service")

	cobra.AddTemplateFunc("StyleHeading", func(data string) string {
		return gchalk.WithHex(color.Orange).Bold(data)
	})

	usageTemplate := cmd.UsageTemplate()
	usageTemplate = strings.NewReplacer(
		`Usage:`, `{{StyleHeading "Usage:"}}`,
		`Aliases:`, `{{StyleHeading "Aliases:"}}`,
		`Examples:`, `{{StyleHeading "Examples:"}}`,
		`Available Commands:`, `{{StyleHeading "Available Commands:"}}`,
		`Global Flags:`, `{{StyleHeading "Global Flags:"}}`,
	).Replace(usageTemplate)

	re := regexp.MustCompile(`(?m)^Flags:\s*$`)
	usageTemplate = re.ReplaceAllLiteralString(usageTemplate, `{{StyleHeading "Flags:"}}`)
	cmd.SetUsageTemplate(usageTemplate)

	err := cmd.Execute()
	if err != nil {
		panic(err)
	}
}
