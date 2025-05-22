// Package span for working with span components.
package span

import (
	"fmt"
	"strings"
)

// Block to visualise a span.
const Block = "■"

// Render the execution graph.
func Render(start, span, length int64) string {
	return fmt.Sprintf("│%s%s%s│", strings.Repeat(" ", toPositiveInt(start)), strings.Repeat(Block, toPositiveInt(span)), strings.Repeat(" ", toPositiveInt(length-span-start)))
}

// Function to ensure that we are returning a positive int for our render.
func toPositiveInt(val int64) int {
	if val < 0 {
		val = 0
	}

	return int(val)
}
