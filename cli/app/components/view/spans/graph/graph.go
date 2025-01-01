// Package graph for rendering the execution graph.
package graph

import (
	"fmt"
	"strings"
)

// Block to visualise a span.
const Block = "◼"

// Render the execution graph.
func Render(start, length int) string {
	return fmt.Sprintf("%s%s", strings.Repeat(" ", start), strings.Repeat(Block, length))
}
