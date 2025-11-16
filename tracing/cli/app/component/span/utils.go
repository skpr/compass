package span

func toPositiveInt(in float64) int {
	if in < 0 {
		return 0
	}

	return int(in)
}
