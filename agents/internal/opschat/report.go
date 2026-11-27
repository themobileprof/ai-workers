package opschat

import (
	"strings"
)

// LooksLikeReport is expansive field reportage, not a prefix, not “done 1”.
func LooksLikeReport(text string) bool {
	text = strings.TrimSpace(text)
	if text == "" || strings.HasPrefix(text, "/") {
		return false
	}
	runes := []rune(text)
	if len(runes) < 50 {
		return false
	}
	lower := strings.ToLower(text)
	strong := []string{
		"we went", "i went", "hospital", "registered", "flier", "flyer",
		"discount code", "we conducted", "postponed", "anc", "pregnant",
		"hod said", "hod ", "ikorodu", "unilag", "altschool", "alt school",
		"training the trainer", "i have asked", "next week monday", "next opportunity",
	}
	weak := []string{
		"yesterday", "on monday", "on tuesday", "on wednesday", "we spent",
		"i spent", "commission", "speech", "patients", "students",
	}
	n := 0
	for _, c := range strong {
		if strings.Contains(lower, c) {
			n += 2
		}
	}
	for _, c := range weak {
		if strings.Contains(lower, c) {
			n++
		}
	}
	if n == 0 {
		return false
	}
	if len(runes) >= 80 {
		return true
	}
	return n >= 2
}
