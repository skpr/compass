// Package main for handling the main application.
package main

import (
	"context"
	"fmt"
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
	URI         string
	OllamaURL   string
	OllamaModel string
}

func main() {
	o := Options{}

	cmd := &cobra.Command{
		Use:     "compass",
		Short:   "A toolkit for pointing developers in the right direction for performance issues.",
		Long:    cmdLong,
		Example: cmdExample,
		RunE: func(cmd *cobra.Command, args []string) error {
			p := tea.NewProgram(app.NewModel(o.URI, o.OllamaURL, o.OllamaModel), tea.WithAltScreen())

			logger, err := applogger.New(p)
			if err != nil {
				return fmt.Errorf("failed to setup logger: %w", err)
			}

			ctx, cancel := context.WithCancel(cmd.Context())

			eg := errgroup.Group{}

			// Start the collector.
			eg.Go(func() error {
				err := tracer.Start(ctx, logger, p, o.URI)
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
	cmd.PersistentFlags().StringVar(&o.OllamaURL, "ollama-url", env.String("COMPASS_OLLAMA_URL", "http://localhost:11434"), "Ollama API URL for AI trace analysis")
	cmd.PersistentFlags().StringVar(&o.OllamaModel, "ollama-model", env.String("COMPASS_OLLAMA_MODEL", "llama3.2"), "Ollama model to use for AI trace analysis")

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
