package app

// history is a fixed-capacity ring in arrival order. New values are appended
// in O(1); once full, append overwrites the oldest value. Presentation code
// reads it newest-first without changing that storage order.
type history[T any] struct {
	values []T
	start  int
	length int
}

func newHistory[T any](limit int) history[T] {
	var h history[T]
	h.setLimit(limit)

	return h
}

// setLimit changes the capacity while retaining the newest values which fit.
func (h *history[T]) setLimit(limit int) {
	if limit < 1 {
		limit = 1
	}
	if len(h.values) == limit {
		return
	}

	keep := min(h.length, limit)
	values := make([]T, limit)

	// Copy oldest to newest so the resized ring starts unwrapped. Resizing is a
	// configuration operation; ordinary insertion never walks retained data.
	for i := range keep {
		value, _ := h.newest(keep - 1 - i)
		values[i] = value
	}

	h.values = values
	h.start = 0
	h.length = keep
}

// append adds a value and reports the oldest value when the ring was full.
func (h *history[T]) append(value T) (T, bool) {
	if len(h.values) == 0 {
		h.setLimit(1)
	}

	if h.length < len(h.values) {
		index := (h.start + h.length) % len(h.values)
		h.values[index] = value
		h.length++

		var zero T
		return zero, false
	}

	evicted := h.values[h.start]
	h.values[h.start] = value
	h.start = (h.start + 1) % len(h.values)

	return evicted, true
}

func (h *history[T]) len() int {
	return h.length
}

func (h *history[T]) limit() int {
	return len(h.values)
}

// newest returns the value at a newest-first logical index.
func (h *history[T]) newest(index int) (T, bool) {
	if index < 0 || index >= h.length {
		var zero T
		return zero, false
	}

	physical := (h.start + h.length - 1 - index) % len(h.values)
	return h.values[physical], true
}

// oldest returns the value at an oldest-first logical index.
func (h *history[T]) oldest(index int) (T, bool) {
	if index < 0 || index >= h.length {
		var zero T
		return zero, false
	}

	return h.values[(h.start+index)%len(h.values)], true
}

func (h *history[T]) setNewest(index int, value T) bool {
	if index < 0 || index >= h.length {
		return false
	}

	physical := (h.start + h.length - 1 - index) % len(h.values)
	h.values[physical] = value

	return true
}

func (h *history[T]) setOldest(index int, value T) bool {
	if index < 0 || index >= h.length {
		return false
	}

	h.values[(h.start+index)%len(h.values)] = value

	return true
}

func (h *history[T]) removeOldest() (T, bool) {
	if h.length == 0 {
		var zero T
		return zero, false
	}

	value := h.values[h.start]
	var zero T
	h.values[h.start] = zero
	h.start = (h.start + 1) % len(h.values)
	h.length--

	if h.length == 0 {
		h.start = 0
	}

	return value, true
}
