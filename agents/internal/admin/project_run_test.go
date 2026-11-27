package admin

import (
	"strings"
	"testing"
)

func TestProjectRunMarksNowAndMust(t *testing.T) {
	p := Project{ID: 7, Name: "Apex Clerk", OneLiner: "Who pays, for what.", Stage: "idea"}
	steps := projectRun(runInput{Project: p})
	if n := runMustLeft(steps); n != 5 {
		t.Fatalf("empty kit must left %d", n)
	}
	now := 0
	for _, s := range steps {
		if s.ID == "card" && (!s.Now || s.Stamp != "now") {
			t.Fatalf("card should be now, got %+v", s)
		}
		if s.Kind == "Must" && s.Done {
			t.Fatalf("must %s done too early", s.ID)
		}
		if s.Kind == "As needed" && s.Stamp != "later" {
			t.Fatalf("optional %s stamp %s", s.ID, s.Stamp)
		}
		if s.Now {
			now++
		}
	}
	if now != 1 {
		t.Fatalf("now count %d", now)
	}

	p.Notes = "Three desks in Ikeja."
	p.Members = []ProjectMember{{UserID: 2, Name: "Ada", Seat: "cofounder"}}
	p.Journey = "shop_ledger"
	p.Gate = "design_partner"
	steps = projectRun(runInput{
		Project: p,
		Events:  []ProjectEvent{{ID: 1, Title: "Field week"}},
		Items:   []WorkItem{{ID: 1, Title: "Visit the bay"}},
	})
	if n := runMustLeft(steps); n != 0 {
		t.Fatalf("full kit must left %d", n)
	}
	foundGate := false
	for _, s := range steps {
		if s.ID == "gate" {
			foundGate = true
			if !s.Done {
				t.Fatalf("gate should be done when notes, face, and work exist")
			}
		}
		if s.ID == "legal" && (!s.Now || s.Stamp != "now") {
			t.Fatalf("first optional should be now, legal %+v", s)
		}
	}
	if !foundGate {
		t.Fatal("placed project missing this-gate step")
	}
}

func TestProjectNextDo(t *testing.T) {
	empty := Project{ID: 7, Name: "Apex"}
	if empty.NextDo() != "Write this week’s line on the Card." || !strings.Contains(empty.NextHref(), "tab=card") {
		t.Fatalf("%q %s", empty.NextDo(), empty.NextHref())
	}
	p := Project{
		ID: 7, OneLiner: "Who pays", Notes: "Go to the bay",
		Members:  []ProjectMember{{Name: "Ada"}},
		Proposal: &Proposal{Journey: "shop_ledger", Gate: "design_partner"},
	}
	if !strings.Contains(p.NextDo(), "Decide") || !strings.Contains(p.NextHref(), "tab=stamp") {
		t.Fatalf("%q %s", p.NextDo(), p.NextHref())
	}
	parked := Project{ID: 7, Stage: "parked"}
	if !parked.Shelved() || !strings.Contains(parked.NextDo(), "Parked") {
		t.Fatalf("%q", parked.NextDo())
	}
}

func TestProjectRunLabel(t *testing.T) {
	empty := Project{Name: "Apex"}
	if empty.RunLabel() == "" {
		t.Fatal("empty kit should list gaps")
	}
	full := Project{
		OneLiner: "Who pays", Notes: "Go to the bay",
		Members: []ProjectMember{{Name: "Ada"}},
		Journey: "shop_ledger", Gate: "design_partner",
		NextEvent: &ProjectEvent{Title: "Field"},
	}
	if full.RunLabel() != "" {
		t.Fatalf("full kit label %q", full.RunLabel())
	}
}

func TestProjectRunAdsStep(t *testing.T) {
	p := Project{ID: 7, Name: "HomeGauge", OneLiner: "Who pays, for what.", URL: "https://homegauge.themobileprof.com"}
	steps := projectRun(runInput{Project: p, CanGrowth: true})
	var ads *RunStep
	for i := range steps {
		if steps[i].ID == "ads" {
			ads = &steps[i]
		}
	}
	if ads == nil || ads.Done || !strings.Contains(ads.Href, "tab=ads") {
		t.Fatalf("%+v", ads)
	}
	done := projectRun(runInput{
		Project: p, CanGrowth: true,
		Decks: []GrowthDeck{{Kind: "ads", Status: "proposed"}, {Kind: "session", Status: "proposed"}},
	})
	for _, s := range done {
		if s.ID == "ads" && !s.Done {
			t.Fatalf("ads pack should mark ads done: %+v", s)
		}
		if s.ID == "deck" && !s.Done {
			t.Fatalf("pitch pack should mark deck done: %+v", s)
		}
	}
}
