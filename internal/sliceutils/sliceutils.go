package sliceutils

import "slices"

func ContainsOne[T comparable](slice []T, items ...T) (T, bool) {
	for _, s := range slice {
		if slices.Contains(items, s) {
			return s, true
		}

	}
	var d T
	return d, false
}
