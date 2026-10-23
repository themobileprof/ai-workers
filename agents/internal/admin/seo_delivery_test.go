package admin

import (
	"context"
	"strings"
	"testing"
)

func TestParseSeoPlanItemsSpreadsWeeks(t *testing.T) {
	got := parseSeoPlanItems([]any{
		map[string]any{"title": "Update Google Business hours", "hours": 1},
		map[string]any{"title": "Ask the Ikeja desk for a citation", "hours": 2},
	}, 9)
	if len(got) != 2 {
		t.Fatalf("%d", len(got))
	}
	if got[0].ProjectID != 9 || got[0].Kind != "seo_offsite" || got[0].Due == "" {
		t.Fatalf("%+v", got[0])
	}
	if got[1].Due == "" || got[1].Due == got[0].Due {
		t.Fatalf("dues should spread %+v", got)
	}
}

func TestWorkItemsFromPlanMaps(t *testing.T) {
	items := workItemsFromPlan([]map[string]any{
		{"project_id": 3, "title": "List the Academy on a partner page", "hours": 1.5, "kind": "seo_offsite", "due": "2026-11-01"},
	})
	if len(items) != 1 || items[0].ProjectID != 3 || items[0].Kind != "seo_offsite" {
		t.Fatalf("%+v", items)
	}
}

func TestSeoPackListsDeliveryTasks(t *testing.T) {
	body := seoPackMarkdown(SeoBrief{
		ID: 4, Kind: "offsite", Title: "Academy listings", Status: "proposed",
		PlanItems: []seoPlanItem{{Title: "Confirm NAP on Google Business", Due: "2026-11-01", Hours: 1, Kind: "seo_offsite"}},
	}, Project{Name: "Academy", Slug: "academy"}, nil)
	for _, want := range []string{"Delivery", "Confirm NAP on Google Business", "2026-11-01", "unless a human Turns away"} {
		if !strings.Contains(body, want) {
			t.Fatalf("missing %q in %s", want, body)
		}
	}
}

func TestFileSeoOffsitePlanSkipsEmptyAndRejected(t *testing.T) {
	s := &Server{store: &Store{}}
	if s.fileSeoOffsitePlan(context.Background(), User{}, SeoBrief{ID: 1, Status: "rejected", PlanItems: []seoPlanItem{{Title: "x", ProjectID: 1}}}) {
		t.Fatal("rejected brief filed a plan")
	}
	if s.fileSeoOffsitePlan(context.Background(), User{}, SeoBrief{ID: 2, Status: "proposed"}) {
		t.Fatal("empty items filed a plan")
	}
}
