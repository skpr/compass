package list

// Returns the start and end of the list that should be displayed.
func getPositionStartAndEnd(position, visible, length int) (int, int) {
	if length <= visible {
		return 0, length
	}

	if position < visible {
		return 0, visible
	}

	if position+1 > length {
		return length - visible, length
	}

	return position - visible + 1, position + 1
}
