package admin

import (
	"testing"
)

func TestNormalizePhone(t *testing.T) {
	cases := map[string]string{
		"+234 803 395 4301": "2348033954301",
		"08033954301":       "2348033954301",
		"2348033954301":     "2348033954301",
		"8033954301":        "2348033954301",
	}
	for in, want := range cases {
		if got := normalizePhone(in); got != want {
			t.Fatalf("%q → %q want %q", in, got, want)
		}
	}
}
