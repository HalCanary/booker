package humanize

import "testing"

func TestHumanize(t *testing.T) {
	for v, s := range map[int]string{
		0:                    "0 B",
		1:                    "1 B",
		1023:                 "1023 B",
		1024:                 "1 KB",
		1025:                 "1 KB",
		2047:                 "1 KB",
		2048:                 "2 KB",
		2049:                 "2 KB",
		1073741823:           "1023 MB",
		1073741824:           "1 GB",
		1073741825:           "1 GB",
		-1:                   "-1 B",
		9223372036854775807:  "7 EB",
		-9223372036854775808: "-9223372036854775808 B",
	} {
		u := Humanize(v)
		if u != s {
			t.Errorf("(%d) %q != %q\n", v, u, s)
		}
	}
}
