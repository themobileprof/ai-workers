package admin

import (
	"strings"
	"unicode"
)

func normalizePhone(s string) string {
	var b strings.Builder
	for _, r := range s {
		if unicode.IsDigit(r) {
			b.WriteRune(r)
		}
	}
	d := b.String()
	if strings.HasPrefix(d, "00") {
		d = d[2:]
	}
	if len(d) == 11 && strings.HasPrefix(d, "0") {
		d = "234" + d[1:]
	}
	if len(d) == 10 && (d[0] == '7' || d[0] == '8' || d[0] == '9') {
		d = "234" + d
	}
	return d
}
