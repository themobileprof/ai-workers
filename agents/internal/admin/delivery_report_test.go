package admin

import "testing"

func TestMatchOpenWorkByTitle(t *testing.T) {
	items := []WorkItem{
		{ID: 7, Title: "Ikorodu General Hospital visit", Status: "open", AssigneeID: 1},
		{ID: 8, Title: "Academy lesson editor", Status: "open", AssigneeID: 1},
		{ID: 9, Title: "Someone else's grant", Status: "open", AssigneeID: 2},
	}
	actor := User{ID: 1, Role: "handler"}
	got := matchOpenWork(items, actor, []string{"hospital visit", "ikorodu general"})
	if len(got) != 1 || got[0] != 7 {
		t.Fatalf("%v", got)
	}
	if id := matchOneWork(items, actor, "grant"); id != 0 {
		t.Fatalf("must not close another person's task: %d", id)
	}
	owner := User{ID: 1, Role: "owner"}
	if id := matchOneWork(items, owner, "grant"); id != 9 {
		t.Fatalf("owner may match: %d", id)
	}
}

func TestSpendJobFromReportDoesNotPostBooks(t *testing.T) {
	job, ok := spendJobFromReport(User{ID: 1, Name: "Samuel"}, map[string]any{
		"amount_ngn": 20000, "what": "fuel", "vendor": "MRS",
	})
	if !ok {
		t.Fatal("expected job")
	}
	if job.Desk != "accounts" || job.Status != "proposed" {
		t.Fatalf("%+v", job)
	}
	if _, has := job.Payload["zoho_action"]; has {
		t.Fatal("mixed report must not set zoho_action")
	}
	if job.Payload["from_report"] != true {
		t.Fatalf("%+v", job.Payload)
	}
	if _, ok := spendJobFromReport(User{}, map[string]any{"amount_ngn": 0, "what": "fuel"}); ok {
		t.Fatal("zero amount")
	}
}

func TestFormatAmount(t *testing.T) {
	if formatAmount(20000) != "20000" {
		t.Fatal(formatAmount(20000))
	}
	if formatAmount(15.5) != "15.5" {
		t.Fatal(formatAmount(15.5))
	}
}
