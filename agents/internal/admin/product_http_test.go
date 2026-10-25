package admin

import (
	"strings"
	"testing"
)

func TestMatchProjectFromSlugAndName(t *testing.T) {
	projects := []Project{
		{ID: 1, Name: "Mechazone", Slug: "mechazone"},
		{ID: 2, Name: "HomeGauge", Slug: "homegauge"},
		{ID: 3, Name: "TheMobileProf Academy", Slug: "academy"},
	}
	p, ok := matchProject(projects, map[string]any{"project_slug": "mechazone"}, "whatever")
	if !ok || p.ID != 1 {
		t.Fatalf("slug %+v ok=%v", p, ok)
	}
	p, ok = matchProject(projects, nil, "Analyse evidence from three Ikeja HomeGauge desks")
	if !ok || p.Slug != "homegauge" {
		t.Fatalf("name %+v ok=%v", p, ok)
	}
	if _, ok := matchProject(projects, nil, "Generate hypotheses for a WhatsApp books assistant"); ok {
		t.Fatal("generic idea must not pick a bet")
	}
	if _, ok := matchProject(projects, nil, "academic paper"); ok {
		t.Fatal("academic is not academy")
	}
}

func TestMergedCardKeepsBurnAndAppendsEvidence(t *testing.T) {
	p := Project{Hypothesis: "old hyp", Mission: "old mission", MissionDue: "2026-01-01", BurnNGN: "400000", Notes: "This week: three desks"}
	hyp, mission, due, burn, notes, changed := mergedCard(p, map[string]any{
		"card_hypothesis": "A shop will pay TMP this month.",
		"card_evidence":   "Ada at XYZ quoted 22%.",
	})
	if hyp != "A shop will pay TMP this month." || mission != "old mission" || due != "2026-01-01" {
		t.Fatalf("%s %s %s", hyp, mission, due)
	}
	if burn != "400000" {
		t.Fatal("burn")
	}
	if !changed || !strings.Contains(notes, "three desks") || !strings.Contains(notes, "Ada at XYZ") {
		t.Fatalf("notes %q", notes)
	}
}

func TestJobFromWorkerMemoryRef(t *testing.T) {
	j := jobFromWorker("product-dev", "telegram", "hypotheses", map[string]any{
		"save_memory": true, "project_id": int64(7), "title": "Card · hypothesis",
	}, "Shops pay.")
	if j.RefKind != "project_memory" || j.RefID != 7 {
		t.Fatalf("%+v", j)
	}
	place := jobFromWorker("product-dev", "telegram", "place", map[string]any{
		"project_id": int64(7), "suggested_journey": "shop_ledger",
	}, "design partner")
	if place.RefKind != "project" {
		t.Fatalf("placement %+v", place)
	}
}
