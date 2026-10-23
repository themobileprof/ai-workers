package admin

import (
	"strings"
	"testing"
)

func TestNextSeoWalkAction(t *testing.T) {
	if nextSeoWalkAction(nil) != "cite" {
		t.Fatal("empty starts at cite")
	}
	got := nextSeoWalkAction([]SeoBrief{
		{Kind: "cite", Status: "proposed"},
		{Kind: "faq", Status: "rejected"},
	})
	if got != "faq" {
		t.Fatalf("rejected faq should rerun, got %s", got)
	}
	if nextSeoWalkAction([]SeoBrief{
		{Kind: "cite", Status: "proposed"},
		{Kind: "faq", Status: "proposed"},
		{Kind: "schema", Status: "proposed"},
		{Kind: "onsite", Status: "proposed"},
		{Kind: "offsite", Status: "proposed"},
	}) != "" {
		t.Fatal("walk filled")
	}
}

func TestResolveSeoActionWalk(t *testing.T) {
	if resolveSeoAction("walk", nil) != "cite" {
		t.Fatal("walk")
	}
	if resolveSeoAction("", []SeoBrief{{Kind: "cite", Status: "proposed"}}) != "faq" {
		t.Fatal("next after cite")
	}
	if resolveSeoAction("audit", nil) != "audit" {
		t.Fatal("named kind")
	}
}

func TestSeoWalkNowStamp(t *testing.T) {
	steps := seoWalk(nil, "https://homegauge.themobileprof.com")
	if len(steps) != 5 || !steps[0].Now || steps[0].ID != "cite" || steps[1].Stamp != "later" {
		t.Fatalf("%+v", steps)
	}
	done := seoWalk([]SeoBrief{{Kind: "cite", Status: "proposed"}}, "https://homegauge.themobileprof.com")
	if !done[0].Done || !done[1].Now || done[1].ID != "faq" {
		t.Fatalf("%+v", done)
	}
}

func TestSeoPackIncludesWalkSiblings(t *testing.T) {
	body := seoPackMarkdown(SeoBrief{ID: 9, Kind: "onsite", Title: "Rates pack", Status: "proposed"},
		Project{Name: "HomeGauge"},
		[]SeoBrief{{Kind: "cite", Status: "proposed", Body: "HomeGauge lists Ikeja owner-occupier rates."}})
	for _, want := range []string{"Quotable facts", "Ikeja owner-occupier"} {
		if !strings.Contains(body, want) {
			t.Fatalf("missing %q in %s", want, body)
		}
	}
}
