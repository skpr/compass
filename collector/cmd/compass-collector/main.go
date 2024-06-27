package main

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/jwalton/gchalk"
	"github.com/spf13/cobra"

	"github.com/skpr/compass/collector/cmd/compass-collector/run"
)

const cmdExample = `
    # Run the collector.
    compass-collector run --container=php-fpm
`

var cmd = &cobra.Command{
	Use:     "compass-collector",
	Short:   "Collector for the Compass project.",
	Example: cmdExample,
	Long: `   __________  __  _______  ___   __________
	/ ____/ __ \/  |/  / __ \/   | / ___/ ___/
   / /   / / / / /|_/ / /_/ / /| | \__ \\__ \
  / /___/ /_/ / /  / / ____/ ___ |___/ /__/ /
  \____/\____/_/  /_/_/   /_/  |_/____/____/

A tool for pointing developers in the right direction for performance issues.`,
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

	cmd.AddCommand(run.NewCommand())

	if err := cmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func styleHeading(data string) string {
	return gchalk.WithHex("#ee5622").Bold(data)
}
