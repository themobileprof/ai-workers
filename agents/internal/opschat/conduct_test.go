package opschat

import (
	"context"
	"strings"
	"testing"
)

func TestMatchBetPrefersNamedSlug(t *testing.T) {
	bets := []BetRef{
		{ID: 1, Slug: "homegauge", Name: "HomeGauge"},
		{ID: 2, Slug: "academy", Name: "Academy"},
	}
	if got := MatchBet("Search ads for HomeGauge", bets, "academy"); got != "homegauge" {
		t.Fatalf("got %q", got)
	}
	if got := MatchBet("tighten the copy", bets, "academy"); got != "academy" {
		t.Fatalf("sticky %q", got)
	}
}

func TestDeskOnlyPlace(t *testing.T) {
	bets := []BetRef{{ID: 7, Slug: "homegauge", Name: "HomeGauge"}}
	c, ok := DeskOnly("Place the HomeGauge ads on Google", bets, "")
	if !ok || c.Intent != IntentDesk || c.DeskAction != "place" {
		t.Fatalf("%+v ok=%v", c, ok)
	}
	if !strings.Contains(c.DeskPath, "/admin/growth") || !strings.Contains(c.DeskPath, "project_id=7") {
		t.Fatalf("path %s", c.DeskPath)
	}
	text := FormatDeskReply(c)
	if !strings.Contains(text, "never") || !strings.Contains(text, c.DeskPath) {
		t.Fatalf("%s", text)
	}
}

func TestDeskOnlyMatureAndDecide(t *testing.T) {
	bets := []BetRef{{ID: 3, Slug: "mechazone", Name: "Mechazone"}}
	c, ok := DeskOnly("Mark Mechazone ads spend mature", bets, "")
	if !ok || c.DeskAction != "mature" || !strings.Contains(c.DeskPath, "tab=card") {
		t.Fatalf("%+v", c)
	}
	c, ok = DeskOnly("Change the stage on Decide for Mechazone", bets, "")
	if !ok || c.DeskAction != "decide" || !strings.Contains(c.DeskPath, "tab=decide") {
		t.Fatalf("%+v", c)
	}
}

func TestFormatPlanAndGap(t *testing.T) {
	plan := FormatPlanPreamble(Conduct{
		Department: "growth",
		Plan: []PlanStep{
			{Department: "growth", Why: "draft the Search pack"},
			{Department: "legal", Why: "claims review"},
		},
	})
	if !strings.Contains(plan, "1. growth") || !strings.Contains(plan, "Running only the first") {
		t.Fatal(plan)
	}
	gap := FormatGapReply(Conduct{
		GapMissing:  "outbound scraper for landlords",
		GapDesk:     "growth",
		GapSpecimen: "A later worker that proposes a list and never sends.",
	})
	if !strings.Contains(gap, "not a live worker") || !strings.Contains(gap, "Specimen") {
		t.Fatal(gap)
	}
}

func TestConductDeskWithoutLLM(t *testing.T) {
	bets := []BetRef{{ID: 1, Slug: "academy", Name: "Academy"}}
	got, err := ConductRoute(context.Background(), nil, "Open Defaults and amend the ads ceiling", "", "academy", bets)
	if err != nil {
		t.Fatal(err)
	}
	if got.Intent != IntentDesk || got.DeskAction != "defaults" {
		t.Fatalf("%+v", got)
	}
}

func TestClassifyStillReturnsDepartment(t *testing.T) {
	c := stubLLM{text: `{"status":"success","output_text":"ok","structured_data":{"intent":"work","department":"legal","project_slug":""}}`}
	got, err := Classify(context.Background(), c, "NDA for the facilitator", "")
	if err != nil || got != "legal" {
		t.Fatalf("%q %v", got, err)
	}
}

func TestConductPlanRunsFirstOnly(t *testing.T) {
	c := stubLLM{text: `{"status":"success","output_text":"ok","structured_data":{"intent":"plan","department":"growth","project_slug":"homegauge","plan":[{"department":"growth","why":"ads pack"},{"department":"product-dev","why":"segments"}]}}`}
	bets := []BetRef{{ID: 2, Slug: "homegauge", Name: "HomeGauge"}}
	got, err := ConductRoute(context.Background(), c, "HomeGauge ads then mark winners", "", "", bets)
	if err != nil {
		t.Fatal(err)
	}
	if got.Intent != IntentPlan || got.Department != "growth" || len(got.Plan) != 2 {
		t.Fatalf("%+v", got)
	}
}

func TestConductGap(t *testing.T) {
	c := stubLLM{text: `{"status":"success","output_text":"ok","structured_data":{"intent":"gap","gap_missing":"TikTok auto-poster","gap_desk":"growth","gap_specimen":"Propose copy only; human posts."}}`}
	got, err := ConductRoute(context.Background(), c, "auto post our TikToks every morning", "", "", nil)
	if err != nil {
		t.Fatal(err)
	}
	if got.Intent != IntentGap || got.GapMissing == "" || got.Department != "" {
		t.Fatalf("%+v", got)
	}
}
