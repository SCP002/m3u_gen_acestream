package maps

import "regexp"

// ForEveryMatchingRx funs `cb` for every regular expression (keys in `rxMap`) matching `match`.
//
// `cb` argument is a `rxMap` value.
func ForEveryMatchingRx(rxMap map[string]string, match string, cb func(mapVal string)) {
	for rxStr, val := range rxMap {
		rx := regexp.MustCompile(rxStr)
		if rx.MatchString(match) {
			cb(val)
		}
	}
}
