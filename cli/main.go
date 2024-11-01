// Package main for handling the main application.
package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"regexp"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/jwalton/gchalk"
	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"

	"github.com/skpr/compass/cli/app"
	"github.com/skpr/compass/cli/app/color"
	"github.com/skpr/compass/cli/sink"
	"github.com/skpr/compass/collector"
	"github.com/skpr/compass/collector/extension/discovery"
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
	ProcessName   string
	ProcessPoll   time.Duration
	ExtensionPath string
}

func main() {
	o := Options{}

	cmd := &cobra.Command{
		Use:     "compass-sidecar",
		Short:   "A toolkit for pointing developers in the right direction for performance issues.",
		Long:    cmdLong,
		Example: cmdExample,
		RunE: func(_ *cobra.Command, _ []string) error {
			p := tea.NewProgram(app.NewModel(), tea.WithAltScreen())

			ctx, cancel := context.WithCancel(context.Background())

			eg := errgroup.Group{}

			// Start the collector.
			eg.Go(func() error {
				logger := slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
					Level: slog.LevelError,
				}))

				path, err := discovery.GetPathFromProcess(logger, o.ProcessName, o.ExtensionPath, o.ProcessPoll)
				if err != nil {
					return err
				}

				return collector.Run(ctx, logger, sink.New(p), collector.RunOptions{
					ExecutablePath: path,
				})
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

	// Extension discovery flags.
	cmd.PersistentFlags().StringVar(&o.ProcessName, "process-name", "php-fpm", "Name of the process which will be used for discovery")
	cmd.PersistentFlags().DurationVar(&o.ProcessPoll, "process-poll", time.Second*5, "How frequently to poll for current list of processes")
	cmd.PersistentFlags().StringVar(&o.ExtensionPath, "extension-path", "/usr/lib/php/modules/compass.so", "Path to the Compass extension")

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
