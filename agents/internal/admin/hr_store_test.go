package admin

import "testing"

func TestRoleFromStructured(t *testing.T) {
	_, err := roleFromStructured(map[string]any{"template": "intern"}, 0)
	if err == nil {
		t.Fatal("jd required")
	}
	r, err := roleFromStructured(map[string]any{
		"template":   "intern",
		"role_title": "Field intern",
		"role_slug":  "field-intern",
		"jd":         "TheMobileProf Technologies intern for Mechazone.",
	}, 7)
	if err != nil {
		t.Fatal(err)
	}
	if r.ProjectID != 7 || r.Kind != "intern" || r.Title != "Field intern" {
		t.Fatalf("%+v", r)
	}
}

func TestApplicationFromStructured(t *testing.T) {
	_, err := applicationFromStructured(map[string]any{"recommendation": "reject"}, 1, "email")
	if err == nil {
		t.Fatal("applicant required")
	}
	a, err := applicationFromStructured(map[string]any{
		"applicant_name":  "Chinedu",
		"applicant_email": "chinedu@example.com",
		"cv_text":         "Lagos. WhatsApp intern.",
		"score":           70.0,
		"recommendation":  "shortlist",
		"reasons":         []any{"WhatsApp sample in the thread"},
	}, 3, "email")
	if err != nil {
		t.Fatal(err)
	}
	if a.RoleID != 3 || a.Score != 70 || a.Recommendation != "shortlist" || a.Source != "email" {
		t.Fatalf("%+v", a)
	}
}

func TestHrFlash(t *testing.T) {
	if hrFlash("app_filed") == "" || hrFlash("role_open") == "" {
		t.Fatal("flash")
	}
}
