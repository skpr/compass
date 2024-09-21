package breakdown

import (
	"fmt"
	"strings"
)

// A graphical representation of the execution time.
func getExecutionGraph(totalExecutionTime, executionTime float64) string {
	length := int(executionTime / totalExecutionTime * 20)

	if length == 0 {
		length = 1
	}

	if length > 20 {
		length = 20
	}

	return fmt.Sprintf("%s (%vms)", strings.Repeat("█", length), int(executionTime))
}
