package sliceutil

// Unique - returns a cleaned up list,
// where every item is unique.
// Does NOT guarantee any ordering, the result can
// be in any order!
func Unique[T comparable](slice []T) []T {
	seen := make(map[T]bool, len(slice))
	uniqueItems := make([]T, 0, len(slice))
	for _, k := range slice {
		if !seen[k] {
			uniqueItems = append(uniqueItems, k)
			seen[k] = true
		}
	}
	return uniqueItems
}

// Filter - returns a slice containing only the elements
// that satisfy the predicate function filterFunc.
func Filter[T any](slice []T, filterFunc func(T) bool) []T {
	filtered := make([]T, 0, len(slice))
	for _, item := range slice {
		if filterFunc(item) {
			filtered = append(filtered, item)
		}
	}
	return filtered
}

// Map - returns a new slice containing the results of applying
// the mapFunc to each element of the input slice.
func Map[T any, M any](slice []T, mapFunc func(T) M) []M {
	result := make([]M, 0, len(slice))
	for _, item := range slice {
		result = append(result, mapFunc(item))
	}
	return result
}
