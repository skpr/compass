package breakdown

import (
	"fmt"
	"strings"
)

// A graphical representation of the execution time.
func getExecutionGraph(totalExecutionTime, executionTime float64) string {
	return fmt.Sprintf("%s (%vms)", strings.Repeat("█", int(totalExecutionTime/executionTime*10)), totalExecutionTime)
}
