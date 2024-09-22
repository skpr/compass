package breakdown

import (
	"fmt"
	"strings"
)

// A graphical representation of the execution time.
func getExecutionGraph(requestStart, functionStartTime, totalExecuteTime, functionExecuteTime int64) string {
	start := getExecutionGraphStart(requestStart, functionStartTime, totalExecuteTime)
	length := getExecutionGraphUsageLength(totalExecuteTime, functionExecuteTime)
	return fmt.Sprintf("%s%s", strings.Repeat(" ", start), strings.Repeat("█", length))
}

func getExecutionGraphStart(requestStart, functionStartTime, totalExecuteTime int64) int {
	diff := (functionStartTime - requestStart) / 1000

	length := float64(diff) / float64(totalExecuteTime) * 50

	return int(length)
}

func getExecutionGraphUsageLength(totalExecuteTime, functionExecuteTime int64) int {
	length := float64(functionExecuteTime) / float64(totalExecuteTime) * 50

	if length < 1 {
		length = 1
	}

	if length > 50 {
		length = 50
	}

	return int(length)
}
