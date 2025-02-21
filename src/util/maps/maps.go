package maps

import "github.com/dlclark/regexp2"

// ForEveryMatchingRx funs `cb` for every regular expression (keys in `rxMap`) matching `match`.
//
// `cb` argument is a `rxMap` value.
func ForEveryMatchingRx[T any](rxMap map[string]T, match string, cb func(mapVal T)) {
	for rx, mapVal := range rxMap {
		if ok, _ := regexp2.MustCompile(rx, regexp2.RE2).MatchString(match); ok {
			cb(mapVal)
		}
	}
}
