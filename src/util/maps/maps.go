package maps

import "regexp"

// ForEveryMatchingRx funs `cb` for every regular expression (keys in `rxMap`) matching `match`.
//
// `cb` argument is a `rxMap` value.
func ForEveryMatchingRx[T any](rxMap map[string]T, match string, cb func(mapVal T)) {
	for rx, mapVal := range rxMap {
		if regexp.MustCompile(rx).MatchString(match) {
			cb(mapVal)
		}
	}
}
