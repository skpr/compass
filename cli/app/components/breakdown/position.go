package breakdown

// Returns the start and end of the list that should be displayed.
func getPositionStartAndEnd(position, visible, length int) (int, int) {
	// If the length is less than the visible amount, show all.
	if visible > length {
		return 0, length
	}

	// If the position plus the visible amount is greater than the length, show the last visible amount.
	if position+visible > length {
		return length - visible, length
	}

	return position, position + visible
}
