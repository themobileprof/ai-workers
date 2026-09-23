package admin

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSameOrigin(t *testing.T) {
	ok := httptest.NewRequest(http.MethodPost, "https://workers.themobileprof.com/admin/login", nil)
	ok.Host = "workers.themobileprof.com"
	ok.Header.Set("Origin", "https://workers.themobileprof.com")
	if rec := httptest.NewRecorder(); !sameOrigin(rec, ok) {
		t.Fatal("expected same origin")
	}

	bad := httptest.NewRequest(http.MethodPost, "https://workers.themobileprof.com/admin/login", nil)
	bad.Host = "workers.themobileprof.com"
	bad.Header.Set("Origin", "https://evil.example")
	rec := httptest.NewRecorder()
	if sameOrigin(rec, bad) {
		t.Fatal("cross origin must fail")
	}
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status %d", rec.Code)
	}

	ref := httptest.NewRequest(http.MethodPost, "https://workers.themobileprof.com/admin/logout", nil)
	ref.Host = "workers.themobileprof.com"
	ref.Header.Set("Referer", "https://workers.themobileprof.com/admin/")
	if rec := httptest.NewRecorder(); !sameOrigin(rec, ref) {
		t.Fatal("referer same host should pass")
	}
}

func TestPublicHome(t *testing.T) {
	mux := http.NewServeMux()
	RegisterPublic(mux)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "before you can pay for the rest of the company") {
		t.Fatalf("landing %d", rec.Code)
	}
	logo := httptest.NewRecorder()
	mux.ServeHTTP(logo, httptest.NewRequest(http.MethodGet, "/site/themobileprof_cloud.png", nil))
	if logo.Code != http.StatusOK || logo.Body.Len() < 1000 {
		t.Fatalf("logo %d len=%d", logo.Code, logo.Body.Len())
	}
}

func TestTemplatesParse(t *testing.T) {
	if _, err := New(nil, ""); err != nil {
		t.Fatal(err)
	}
}

func TestPlaybookCatalog(t *testing.T) {
	if err := mustPlaybookFiles(); err != nil {
		t.Fatal(err)
	}
	page, ok := lookupDoc("growth")
	if !ok || page.Path != "POST /departments/growth" || page.Prev == nil || page.Next == nil {
		t.Fatalf("growth page %+v ok=%v", page.DocMeta, ok)
	}
	if _, ok := lookupDoc("missing"); ok {
		t.Fatal("unknown slug")
	}
	if first := allDocs()[0]; first.Slug != "contract" || !first.Live() {
		t.Fatalf("first page %s", first.Slug)
	}
	desk, ok := lookupDoc("desk")
	if !ok || !desk.Live() || !strings.Contains(string(desk.Body), "People") {
		t.Fatal("desk playbook")
	}
	proj, ok := lookupDoc("projects")
	if !ok || !proj.Live() || !strings.Contains(string(proj.Body), "this-week") {
		t.Fatal("projects playbook")
	}
	prod, ok := lookupDoc("product-dev")
	if !ok || !prod.Partial() || !strings.Contains(string(prod.Body), "Not wired") {
		t.Fatal("product-dev must stay a partial with later-boxes")
	}
	leg, ok := lookupDoc("legal")
	if !ok || !leg.Partial() || !strings.Contains(string(leg.Body), "Not wired") {
		t.Fatal("legal must stay a partial with a later-box for send")
	}
	h, ok := lookupDoc("hr")
	if !ok || !h.Partial() || !strings.Contains(string(h.Body), "Not wired") {
		t.Fatal("hr must stay a partial with later-boxes")
	}
	cm, ok := lookupDoc("community")
	if !ok || !strings.Contains(string(cm.Body), "lms.themobileprof.com") {
		t.Fatal("community LMS specimen")
	}
	later := DocMeta{Slug: "x", Wire: "later"}
	if !later.Later() || later.Href() != "/admin/docs/x" || later.WireLabel() != "Not wired" {
		t.Fatalf("later %+v %s %s", later, later.Href(), later.WireLabel())
	}
	if !allDocs()[0].Live() || allDocs()[0].WireLabel() != "On the wire" {
		t.Fatal("live label")
	}
	if !leg.Partial() || leg.WireLabel() != "Prefix live · follow-up missing" {
		t.Fatal("partial label")
	}
}

func TestHasStoredCap(t *testing.T) {
	u := User{Role: "owner", Caps: []string{"whatsapp_accounts"}}
	if !hasStoredCap(u, "whatsapp_accounts") || hasStoredCap(u, "web_admin") {
		t.Fatal("stored caps only")
	}
	if !HasCap(u, "web_admin") {
		t.Fatal("owner has all caps for auth")
	}
}

func TestRemovalBlocked(t *testing.T) {
	owner := User{ID: 1, Role: "owner"}
	bookkeeper := User{ID: 2, Role: "accounts"}
	if err := removalBlocked(&owner, owner, 1); err == nil {
		t.Fatal("must not remove self")
	}
	if err := removalBlocked(&owner, owner, 2); err == nil {
		t.Fatal("must not remove self even with another owner")
	}
	if err := removalBlocked(&bookkeeper, owner, 1); err == nil {
		t.Fatal("must not remove last owner")
	}
	if err := removalBlocked(&owner, bookkeeper, 1); err != nil {
		t.Fatal(err)
	}
	if err := lastOwnerLocked(owner, "viewer", true, 1); err == nil {
		t.Fatal("must not demote last owner")
	}
	if err := lastOwnerLocked(owner, "owner", false, 1); err == nil {
		t.Fatal("must not deactivate last owner")
	}
	if err := lastOwnerLocked(owner, "viewer", true, 2); err != nil {
		t.Fatal(err)
	}
}
