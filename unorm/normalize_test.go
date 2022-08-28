package unorm

import (
	"testing"
)

func TestNorm(t *testing.T) {
	tests := [][2]string{
		{"hello world", "hello world"},
		{"", ""},
		{"ö", "o"},
		{"ö", "o"},
		{"àabc", "aabc"},
		{"ÉÉÉÉÉ", "EEEEE"},
	}
	for _, testcase := range tests {
		result := Normalize(testcase[0])
		if result != testcase[1] {
			t.Errorf("Normalize(%q) = %q != %q\n", testcase[0], result, testcase[1])
		}
	}
}
