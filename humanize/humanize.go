// Copyright 2022 Hal Canary
// Use of this program is governed by the file LICENSE.
package humanize

import "fmt"

// Humanize converts a byte size to a human readable number, for example: 2048
// -> "2 kB"
func Humanize(v int) string {
	prfx := []string{"", "K", "M", "G", "T", "P", "E"}
	for i, s := range prfx {
		if v <= 9999 || i == len(prfx)-1 {
			return fmt.Sprintf("%d %sB", v, s)
		}
		v = v >> 10
	}
	return ""
}
