package datatable

import "sort"

// gap between columns, in cells.
const gap = 2

// resolveWidths distributes a total width across columns.
//
// Fixed columns are served first, the flexible ones share what is left in
// proportion to their weight, and nothing goes below its floor. When even the
// floors will not fit, whole columns are dropped in descending priority order
// until they do.
//
// A dropped column is returned with a width of zero.
func resolveWidths(columns []Column, total int) []int {
	widths := make([]int, len(columns))

	if len(columns) == 0 || total <= 0 {
		return widths
	}

	keep := dropToFit(columns, total)

	// Gaps only sit between the columns which survived.
	available := total - (len(keep)-1)*gap

	var (
		fixed  int
		weight int
	)

	for _, i := range keep {
		if columns[i].fixed() {
			fixed += columns[i].Width
			continue
		}

		weight += columns[i].Flex
	}

	remaining := available - fixed

	for _, i := range keep {
		if columns[i].fixed() {
			widths[i] = columns[i].Width
			continue
		}

		share := remaining * columns[i].Flex / max(weight, 1)
		widths[i] = max(share, columns[i].floor())
	}

	// Sharing rounds down and the floors round up, so the total drifts either
	// way. The widest flexible column absorbs the difference, being the one
	// where a cell or two is least noticeable.
	if widest := widestFlexible(columns, widths, keep); widest >= 0 {
		widths[widest] += available - sum(widths)
		widths[widest] = max(widths[widest], columns[widest].floor())
	}

	return widths
}

// dropToFit returns the indices of the columns which fit, dropping the lowest
// priority ones until the floors and gaps do.
func dropToFit(columns []Column, total int) []int {
	keep := make([]int, 0, len(columns))
	for i := range columns {
		keep = append(keep, i)
	}

	// Highest priority number is given up first. Ties break on the rightmost
	// column, so a table sheds from its edge inwards.
	order := make([]int, len(keep))
	copy(order, keep)

	sort.SliceStable(order, func(a, b int) bool {
		if columns[order[a]].Priority != columns[order[b]].Priority {
			return columns[order[a]].Priority > columns[order[b]].Priority
		}

		return order[a] > order[b]
	})

	for _, candidate := range order {
		if floorsFit(columns, keep, total) {
			break
		}

		if columns[candidate].Priority <= 0 || len(keep) == 1 {
			continue
		}

		keep = remove(keep, candidate)
	}

	return keep
}

// floorsFit reports whether the kept columns fit at their floors.
func floorsFit(columns []Column, keep []int, total int) bool {
	needed := (len(keep) - 1) * gap

	for _, i := range keep {
		needed += columns[i].floor()
	}

	return needed <= total
}

// widestFlexible column among those kept, or -1 when none is flexible.
func widestFlexible(columns []Column, widths []int, keep []int) int {
	widest := -1

	for _, i := range keep {
		if columns[i].fixed() {
			continue
		}

		if widest < 0 || widths[i] > widths[widest] {
			widest = i
		}
	}

	return widest
}

func remove(values []int, value int) []int {
	out := values[:0]

	for _, v := range values {
		if v != value {
			out = append(out, v)
		}
	}

	return out
}

func sum(values []int) int {
	var total int

	for _, v := range values {
		total += v
	}

	return total
}
