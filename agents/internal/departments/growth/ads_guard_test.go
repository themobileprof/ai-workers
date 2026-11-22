package growth

import "testing"

func TestGuardAdsDropsCitiesNotOnCard(t *testing.T) {
	blob := CardBlob(map[string]any{
		"one_liner": "Salaried Nigerian homebuyers get mortgage help.",
		"url":       "https://homegauge.themobileprof.com",
		"slug":      "homegauge",
	})
	p, notes := GuardAds(AdsPack{
		Cells:     []AdsCell{{Geo: "Ikeja", Theme: "mortgage help"}, {Geo: "", Theme: "mortgage help"}},
		Keywords:  []string{"mortgage help", "Ikeja mortgage"},
		Headlines: []AdsLine{{Text: "Mortgage help"}, {Text: "Join the waitlist"}},
		Pexels:    []PexelsQuery{{Query: "house keys"}, {Query: "happy woman laptop"}},
	}, blob)
	if len(p.Cells) != 1 || p.Cells[0].Geo != "" {
		t.Fatalf("cells %+v notes %v", p.Cells, notes)
	}
	if len(p.Keywords) != 1 || p.Keywords[0] != "mortgage help" {
		t.Fatalf("keywords %+v", p.Keywords)
	}
	if len(p.Headlines) != 1 || p.Headlines[0].Text != "Mortgage help" {
		t.Fatalf("headlines %+v notes %v", p.Headlines, notes)
	}
	if len(p.Pexels) != 0 {
		t.Fatalf("pexels %+v", p.Pexels)
	}
	if len(notes) == 0 {
		t.Fatal("expected drop notes")
	}
}

func TestCardAllowsEmptyBlobRejects(t *testing.T) {
	if CardAllows("", "Lagos") {
		t.Fatal("empty card must not allow a city")
	}
	if !CardAllows("lagos mortgage help", "Lagos") {
		t.Fatal("lagos on card")
	}
}

func TestGuardAdsEmptyCardLeavesPack(t *testing.T) {
	p, notes := GuardAds(AdsPack{Keywords: []string{"Ikeja"}}, "")
	if len(p.Keywords) != 1 || notes != nil {
		t.Fatalf("%+v %v", p.Keywords, notes)
	}
}
