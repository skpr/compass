package breakdown

import (
	"fmt"
	"math"
	"strings"
)

// A graphical representation of the execution time.
func getExecutionGraph(requestStart, functionStartTime, totalExecuteTime, functionExecuteTime int64) string {
	start := getExecutionGraphStart(requestStart, functionStartTime, totalExecuteTime)
	length := getExecutionGraphUsageLength(totalExecuteTime, functionExecuteTime)
	remainder := 50 - start - length

	block := "◼"

	span := strings.Repeat(block, length)

	return fmt.Sprintf("│%s%s%s│", strings.Repeat(" ", start), span, strings.Repeat(" ", remainder))
}

func getExecutionGraphStart(requestStart, functionStartTime, totalExecuteTime int64) int {
	diff := (functionStartTime - requestStart) / 1000

	length := float64(diff) / float64(totalExecuteTime) * 50

	return int(length)
}

func getExecutionGraphUsageLength(totalExecuteTime, functionExecuteTime int64) int {
	length := math.Round(float64(functionExecuteTime) / float64(totalExecuteTime) * 50)

	if length < 1 {
		length = 1
	}

	if length > 50 {
		length = 50
	}

	return int(length)
}
