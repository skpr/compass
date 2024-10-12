package graph

import (
	"fmt"
	"math"
	"strings"
)

// Length of the execution graph.
const Length = 50

// Render the execution graph.
func Render(requestStart, functionStartTime, totalExecuteTime, functionExecuteTime int64) string {
	start := getExecutionGraphStart(requestStart, functionStartTime, totalExecuteTime)
	length := getExecutionGraphUsageLength(totalExecuteTime, functionExecuteTime)
	remainder := Length - start - length

	block := "◼"

	span := strings.Repeat(block, length)

	return fmt.Sprintf("│%s%s%s│", strings.Repeat(" ", start), span, strings.Repeat(" ", remainder))
}

// Returns when the graph should start.
func getExecutionGraphStart(requestStart, functionStartTime, totalExecuteTime int64) int {
	diff := (functionStartTime - requestStart) / 1000

	length := float64(diff) / float64(totalExecuteTime) * Length

	return int(length)
}

// Returns the length of the graph.
func getExecutionGraphUsageLength(totalExecuteTime, functionExecuteTime int64) int {
	length := math.Round(float64(functionExecuteTime) / float64(totalExecuteTime) * 50)

	if length < 1 {
		length = 1
	}

	if length > Length {
		length = Length
	}

	return int(length)
}
