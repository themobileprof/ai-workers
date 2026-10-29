package admin

import (
	"strings"
	"testing"

	"github.com/samuel/ai-workers/agents/internal/departments/growth"
)

func TestNextDeckWalkAction(t *testing.T) {
	if nextDeckWalkAction(nil) != "session" {
		t.Fatal("empty starts at session")
	}
	if nextDeckWalkAction([]GrowthDeck{{Kind: "session", Status: "rejected"}}) != "session" {
		t.Fatal("rejected session should rerun")
	}
	if nextDeckWalkAction([]GrowthDeck{{Kind: "session", Status: "proposed"}}) != "" {
		t.Fatal("walk filled")
	}
}

func TestResolveDeckActionWalk(t *testing.T) {
	if resolveDeckAction("walk", nil) != "session" {
		t.Fatal("walk")
	}
	if resolveDeckAction("", []GrowthDeck{{Kind: "session", Status: "proposed"}}) != "session" {
		t.Fatal("session stays live")
	}
	if resolveDeckAction("plan", nil) != "plan" {
		t.Fatal("named kind")
	}
}

func TestDeckWalkNowStamp(t *testing.T) {
	p := Project{Name: "HomeGauge", OneLiner: "Salaried buyers get mortgage help."}
	steps := deckWalk(nil, p)
	if len(steps) != 1 || !steps[0].Now || steps[0].ID != "session" {
		t.Fatalf("%+v", steps)
	}
	done := deckWalk([]GrowthDeck{{Kind: "session", Status: "proposed"}}, p)
	if !done[0].Done || done[0].Now {
		t.Fatalf("%+v", done)
	}
	empty := deckWalk(nil, Project{Name: "HomeGauge"})
	if empty[0].Stamp != "later" || empty[0].Now {
		t.Fatalf("no card facts should wait, %+v", empty[0])
	}
}

func TestGrowthPackIncludesCard(t *testing.T) {
	slides := seedSessionSlides(Project{Name: "HomeGauge", OneLiner: "Salaried buyers get mortgage help."})
	body := growthPackMarkdown(GrowthDeck{ID: 3, Kind: "session", Title: "HomeGauge deck", Status: "proposed", Slides: slides, Body: growth.SlidesMarkdown(slides)},
		Project{Name: "HomeGauge", Slug: "homegauge", OneLiner: "Salaried buyers get mortgage help.", Hypothesis: "A desk will quote today."},
		nil)
	for _, want := range []string{"TheMobileProf Technologies", "Salaried buyers", "1. Title", "does not send", "product", "Guy Kawasaki"} {
		if !strings.Contains(body, want) {
			t.Fatalf("missing %q in %s", want, body)
		}
	}
}

func TestCardBeatsKawasaki(t *testing.T) {
	got := cardBeats(Project{
		Name: "HomeGauge", OneLiner: "Salaried buyers get mortgage help.",
		Hypothesis: "A desk will quote today.", Stage: "testing",
		Members: []ProjectMember{{Name: "Ada", Seat: "cofounder"}},
	})
	for _, want := range []string{
		"1. Title — HomeGauge · TheMobileProf Technologies",
		"2. Problem / Opportunity — A desk will quote today.",
		"3. Value Proposition — Salaried buyers get mortgage help.",
		"4. Underlying Magic — [TO BE COMPLETED]",
		"7. Competitive Analysis — [TO BE COMPLETED]",
		"8. Management Team — Ada (cofounder)",
		"9. Financials and Key Metrics — [TO BE COMPLETED]",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("missing %q in %s", want, got)
		}
	}
	if strings.Contains(got, "MAU") {
		t.Fatal("must not invent MAU")
	}
}

func TestGrowthDeckFromStructured(t *testing.T) {
	if _, err := growthDeckFromStructured(map[string]any{"action": "deck"}, 1, nil); err == nil {
		t.Fatal("empty")
	}
	d, err := growthDeckFromStructured(map[string]any{
		"action":   "audience",
		"audience": "Ada (cofounder)",
		"title":    "HomeGauge room",
	}, 7, nil)
	if err != nil || d.Kind != "audience" || d.Body != "Ada (cofounder)" {
		t.Fatalf("%v %+v", err, d)
	}
	got, err := growthDeckFromStructured(map[string]any{
		"action":        "session",
		"current_slide": 2,
		"slides": []any{
			map[string]any{"n": 2, "copy": "A desk will quote today.", "status": "draft"},
		},
	}, 7, seedSessionSlides(Project{Name: "HomeGauge", OneLiner: "Salaried buyers get mortgage help."}))
	if err != nil || got.Kind != "session" || got.CurrentSlide != 2 || got.Slides[1].Copy != "A desk will quote today." {
		t.Fatalf("%v %+v", err, got)
	}
}
