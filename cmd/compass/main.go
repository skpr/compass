// Package main for handling the main application.
package main

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/christgf/env"
	"github.com/jwalton/gchalk"
	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"

	"github.com/skpr/compass/pkg/app"
	"github.com/skpr/compass/pkg/app/color"
	applogger "github.com/skpr/compass/pkg/app/logger"
	"github.com/skpr/compass/pkg/app/tracer"
)

const cmdExample = `
  # Watch and analyze new profiles.
  compass

  # Connect to a sidecar which requires a token.
  compass --uri https://sidecar:28624/v1/traces --token xxxyyyzzz`

const cmdLong = `   _____ ____  __  __ _____         _____ _____
  / ____/ __ \|  \/  |  __ \ /\    / ____/ ____|
 | |   | |  | | \  / | |__) /  \  | (___| (___
 | |   | |  | | |\/| |  ___/ /\ \  \___ \\___ \
 | |___| |__| | |  | | |  / ____ \ ____) |___) |
  \_____\____/|_|  |_|_| /_/    \_\_____/_____/

A toolkit for pointing developers in the right direction for performance issues.`

// Options for the CLI.
type Options struct {
	URI                string
	Token              string
	CAFile             string
	InsecureSkipVerify bool
	MaxTraces          int
}

func main() {
	o := Options{}

	cmd := &cobra.Command{
		Use:     "compass",
		Short:   "A toolkit for pointing developers in the right direction for performance issues.",
		Long:    cmdLong,
		Example: cmdExample,
		// Usage is not helpful for a runtime failure.
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			p := tea.NewProgram(app.NewModel(o.URI, o.MaxTraces), tea.WithAltScreen())

			logger, err := applogger.New(p)
			if err != nil {
				return fmt.Errorf("failed to setup logger: %w", err)
			}

			ctx, cancel := context.WithCancel(cmd.Context())

			eg := errgroup.Group{}

			// Start the collector.
			eg.Go(func() error {
				err := tracer.Start(ctx, logger, p, tracer.Config{
					URI:                o.URI,
					Token:              o.Token,
					CAFile:             o.CAFile,
					InsecureSkipVerify: o.InsecureSkipVerify,
				})
				if err != nil {
					logger.Error(err.Error())
				}

				return err
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

	cmd.PersistentFlags().StringVar(&o.URI, "uri", env.String("COMPASS_URI", "http://localhost:28624/v1/traces"), "URI to connect to for tracing")
	cmd.PersistentFlags().StringVar(&o.Token, "token", env.String("COMPASS_TOKEN", ""), "Token sent to the sidecar for authentication")
	cmd.PersistentFlags().StringVar(&o.CAFile, "ca-file", env.String("COMPASS_CA_FILE", ""), "Path to the certificate authority which signed the sidecar certificate")
	cmd.PersistentFlags().BoolVar(&o.InsecureSkipVerify, "insecure-skip-verify", env.Bool("COMPASS_INSECURE_SKIP_VERIFY", false), "Skip verification of the sidecar certificate")
	cmd.PersistentFlags().IntVar(&o.MaxTraces, "max-traces", env.Int("COMPASS_MAX_TRACES", app.DefaultMaxTraces), "Maximum number of traces to retain, oldest are discarded first")

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

	// Cobra prints the error, so exit quietly rather than panicking with a
	// stack trace over the top of it.
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
