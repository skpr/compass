// Package main for handling the main application.
package main

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/jwalton/gchalk"
	"github.com/spf13/cobra"

	"github.com/skpr/compass/collector/cmd/compass/watch"
	"github.com/skpr/compass/collector/pkg/color"
)

const cmdExample = `
  # Watch and analyze new profiles.
  compass watch`

var cmd = &cobra.Command{
	Use:     "compass",
	Short:   "A toolkit for pointing developers in the right direction for performance issues.",
	Example: cmdExample,
	Long: `   _____ ____  __  __ _____         _____ _____
  / ____/ __ \|  \/  |  __ \ /\    / ____/ ____|
 | |   | |  | | \  / | |__) /  \  | (___| (___
 | |   | |  | | |\/| |  ___/ /\ \  \___ \\___ \
 | |___| |__| | |  | | |  / ____ \ ____) |___) |
  \_____\____/|_|  |_|_| /_/    \_\_____/_____/

A toolkit for pointing developers in the right direction for performance issues.`,
}

func main() {
	cobra.AddTemplateFunc("StyleHeading", styleHeading)
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

	cmd.AddCommand(watch.NewCommand())

	if err := cmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

// Helper function for styling headings in the usage template.
func styleHeading(data string) string {
	return gchalk.WithHex(color.Orange).Bold(data)
}
