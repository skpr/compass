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
	return fmt.Sprintf("│%s%s%s│", strings.Repeat(" ", int(start)), strings.Repeat(Block, int(span)), strings.Repeat(" ", int(length-span-start)))
}
