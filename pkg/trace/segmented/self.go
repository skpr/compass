package segmented

import (
	"sort"
	"time"

	"github.com/skpr/compass/pkg/trace"
)

// SelfTime of each function call, keyed by its index in the given slice.
//
// Self time is how long a call ran for minus how long its direct children ran
// for: the time the function actually spent doing its own work. It is the
// number worth colouring a timeline by. Inclusive time is not — every frame
// which merely wraps the request has almost all of it, so ranking by inclusive
// time puts the kernel at the top and the hotspot in the middle.
//
// The nesting is recovered by containment rather than being recorded: a call
// whose interval sits inside another's was called by it. Sorting by start, and
// by descending length where two calls start together, walks the stack in the
// order the request did.
//
// # Accuracy
//
// The probe only fires for calls above the extension's threshold, so the tree
// is incomplete: a child cheaper than the threshold is missing, and its time is
// counted against its parent. Self time is therefore an upper bound rather than
// a measurement, and the error on any one frame is bounded by the threshold
// times the number of calls hidden beneath it. It is still far closer to the
// truth than inclusive time, which is wrong by design.
func SelfTime(calls []trace.FunctionCall) []time.Duration {
	self := make([]time.Duration, len(calls))

	for i, call := range calls {
		self[i] = call.Elapsed
	}

	order := make([]int, len(calls))
	for i := range order {
		order[i] = i
	}

	sort.SliceStable(order, func(a, b int) bool {
		left, right := calls[order[a]], calls[order[b]]

		if left.Offset != right.Offset {
			return left.Offset < right.Offset
		}

		// A parent starting at the same instant as its child has to come first,
		// and it is the longer of the two.
		return left.Elapsed > right.Elapsed
	})

	// The stack holds the calls still open, outermost first.
	var stack []int

	for _, index := range order {
		call := calls[index]
		end := call.Offset + call.Elapsed

		for len(stack) > 0 {
			open := calls[stack[len(stack)-1]]

			if open.Offset+open.Elapsed > call.Offset {
				break
			}

			stack = stack[:len(stack)-1]
		}

		if len(stack) > 0 {
			parent := stack[len(stack)-1]

			// Only the part of the child inside the parent counts against it.
			// Threshold filtering and aggregation can produce intervals which
			// overhang, and charging a parent for time after it returned would
			// take its self time negative.
			overlap := min(end, calls[parent].Offset+calls[parent].Elapsed) - call.Offset
			if overlap > 0 {
				self[parent] -= overlap
			}
		}

		stack = append(stack, index)
	}

	for i := range self {
		if self[i] < 0 {
			self[i] = 0
		}
	}

	return self
}
