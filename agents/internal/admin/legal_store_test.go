package admin

import "testing"

func TestDraftFromStructured(t *testing.T) {
	_, err := draftFromStructured(map[string]any{"template": "nda", "action": "draft"}, 1, User{}, "Ada")
	if err == nil {
		t.Fatal("draft needs a body")
	}
	d, err := draftFromStructured(map[string]any{
		"template":           "contractor",
		"action":             "draft",
		"body":               "TheMobileProf Technologies and Ada.",
		"flags":              []any{"independent contractor"},
		"needs_human_lawyer": false,
		"counterparty_name":  "Ada",
	}, 9, User{ID: 4, Name: "Ada"}, "")
	if err != nil {
		t.Fatal(err)
	}
	if d.ProjectID != 9 || d.Template != "contractor" || d.CounterpartyUserID != 4 || d.Title != "Contractor" {
		t.Fatalf("%+v", d)
	}
	rev, err := draftFromStructured(map[string]any{
		"template": "none",
		"action":   "review",
		"flags":    []any{"vendor owns all data"},
	}, 2, User{}, "")
	if err != nil {
		t.Fatal(err)
	}
	if rev.Body != "" || rev.Title != "Review" || len(rev.Flags) != 1 {
		t.Fatalf("%+v", rev)
	}
	if _, err := draftFromStructured(map[string]any{"template": "made_up", "action": "draft", "body": "x"}, 1, User{}, ""); err == nil {
		t.Fatal("unknown template")
	}
}

func TestLegalFlash(t *testing.T) {
	if legalFlash("drafted") == "" || legalFlash("legal_ok") == "" {
		t.Fatal("flash")
	}
	if legalFlash("nope") != "" {
		t.Fatal("unknown")
	}
}
