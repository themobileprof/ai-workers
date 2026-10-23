package admin

import (
	"strings"
	"testing"
)

func TestSeoBriefFromStructured(t *testing.T) {
	if _, err := seoBriefFromStructured(map[string]any{"action": "page"}, 1); err == nil {
		t.Fatal("empty")
	}
	d, err := seoBriefFromStructured(map[string]any{
		"action":           "page",
		"url":              "https://homegauge.themobileprof.com/rates",
		"title":            "Ikeja mortgage rates",
		"meta_description": "Today’s owner-occupier rates from named Ikeja desks.",
		"h1":               "Mortgage rates in Ikeja",
		"flags":            []any{"no H1 on the paste"},
	}, 7)
	if err != nil || d.ProjectID != 7 || d.Kind != "page" || d.Meta == "" {
		t.Fatalf("%v %+v", err, d)
	}
	audit, err := seoBriefFromStructured(map[string]any{
		"action": "audit",
		"flags":  []string{"missing canonical", "title over 60"},
	}, 7)
	if err != nil || audit.Kind != "audit" || len(audit.Flags) != 2 {
		t.Fatalf("%v %+v", err, audit)
	}
	off, err := seoBriefFromStructured(map[string]any{
		"action": "offsite",
		"items": []any{
			map[string]any{"title": "Ask the existing partner for a citation", "hours": 1},
		},
	}, 7)
	if err != nil || off.Kind != "offsite" || len(off.PlanItems) != 1 || off.PlanItems[0].Kind != "seo_offsite" {
		t.Fatalf("%v %+v", err, off)
	}
	cite, err := seoBriefFromStructured(map[string]any{
		"action": "cite",
		"cites":  []any{"HomeGauge lists Ikeja owner-occupier mortgage rates."},
	}, 7)
	if err != nil || cite.Kind != "cite" || !strings.Contains(cite.Body, "Ikeja owner-occupier") {
		t.Fatalf("%v %+v", err, cite)
	}
}
