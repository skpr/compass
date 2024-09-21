package breakdown

import (
	"fmt"
	"strings"
)

// A graphical representation of the execution time.
func getExecutionGraph(totalExecutionTime, executionTime int64) string {
	length := float64(executionTime) / float64(totalExecutionTime) * 20

	if length < 1 {
		length = 1
	}

	if length > 20 {
		length = 20
	}

	return fmt.Sprintf("%s (%vms)", strings.Repeat("█", int(length)), int(executionTime))
}
