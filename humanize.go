package main

import "fmt"

// Humanize converts a byte size to a human readable number, for example: 2048
// -> "2 kB"
func Humanize(v int) string {
	prfx := []string{"", "K", "M", "G", "T", "P", "E"}
	for i, s := range prfx {
		n := v >> 10
		if v <= 0 || n == 0 || i == len(prfx)-1 {
			return fmt.Sprintf("%d %sB", v, s)
		}
		v = n
	}
	return ""
}
