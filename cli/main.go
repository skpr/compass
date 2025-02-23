// Package main for handling the main application.
package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"regexp"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/christgf/env"
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
	ExtensionPath string
}

func main() {
	o := Options{}

	cmd := &cobra.Command{
		Use:     "compass",
		Short:   "A toolkit for pointing developers in the right direction for performance issues.",
		Long:    cmdLong,
		Example: cmdExample,
		RunE: func(_ *cobra.Command, _ []string) error {
			logger := slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
				Level: slog.LevelError,
			}))

			path, err := discovery.GetPathFromProcess(logger, o.ProcessName, o.ExtensionPath)
			if err != nil {
				return err
			}

			p := tea.NewProgram(app.NewModel(path), tea.WithAltScreen())

			ctx, cancel := context.WithCancel(context.Background())

			eg := errgroup.Group{}

			// Start the collector.
			eg.Go(func() error {
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

	cmd.PersistentFlags().StringVar(&o.ProcessName, "process-name", env.String("COMPASS_PROCESS_NAME", "php-fpm"), "Name of the process which will be used for discovery")
	cmd.PersistentFlags().StringVar(&o.ExtensionPath, "extension-path", env.String("COMPASS_EXTENSION_PATH", "/usr/lib/php/modules/compass.so"), "Path to the Compass extension")

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
