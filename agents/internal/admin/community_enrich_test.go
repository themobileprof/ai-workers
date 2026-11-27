package admin

import "testing"

func TestLastAskAfterTurn(t *testing.T) {
	if got := lastAskAfterTurn("workshop_help", "step"); got != "workshop" {
		t.Fatalf("still gathering: %s", got)
	}
	if got := lastAskAfterTurn("workshop_help", ""); got != "answered" {
		t.Fatalf("answered workshop: %s", got)
	}
	if got := lastAskAfterTurn("lesson_help", "lesson"); got != "lesson" {
		t.Fatalf("still gathering lesson: %s", got)
	}
	if got := lastAskAfterTurn("faq", ""); got != "answered" {
		t.Fatalf("faq: %s", got)
	}
	if got := lastAskAfterTurn("progress", ""); got != "answered" {
		t.Fatalf("progress: %s", got)
	}
	if got := lastAskAfterTurn("silent", ""); got != "" {
		t.Fatalf("silent should keep prior last_ask: %q", got)
	}
}
