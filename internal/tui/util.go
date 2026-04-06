package tui

func maxInt(left, right int) int {
	if left > right {
		return left
	}

	return right
}

func minInt(left, right int) int {
	if left < right {
		return left
	}

	return right
}
