package admin

import (
	"context"
	"strings"
	"testing"

	"github.com/samuel/ai-workers/agents/internal/acadchat"
)

func TestAcademyOnboardTextIsShort(t *testing.T) {
	s := &Server{}
	got := s.academyOnboardText(context.Background(), acadchat.Inbound{
		Name:      "Ada",
		JoinNames: []string{"Bola"},
	})
	if !strings.Contains(got, "Welcome Bola") || !strings.Contains(got, "Which course") {
		t.Fatal(got)
	}
	if strings.Contains(got, "\n") || strings.Contains(strings.ToLower(got), "billing") || strings.Contains(strings.ToLower(got), "workshop") || strings.Contains(got, "Upcoming") {
		t.Fatalf("join should be one short line: %s", got)
	}
}
