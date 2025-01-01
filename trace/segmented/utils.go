package segmented

import (
	"math"
)

func getSegmentStart(requestStart, functionStartTime, totalExecuteTime int64, fullLength float64) int {
	return int(float64((functionStartTime-requestStart)/1000) / float64(totalExecuteTime) * fullLength)
}

func getSegmentLength(totalExecuteTime, functionExecuteTime int64, fullLength float64) int {
	length := math.Round(float64(functionExecuteTime) / float64(totalExecuteTime) * fullLength)

	if length < 1 {
		length = 1
	}

	if length > fullLength {
		length = fullLength
	}

	return int(length)
}
