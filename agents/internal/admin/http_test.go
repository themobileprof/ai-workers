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

func TestGuideDockOnSignedInLayout(t *testing.T) {
	s, err := New(nil, "")
	if err != nil {
		t.Fatal(err)
	}
	rec := httptest.NewRecorder()
	s.render(rec, "home", pageData{Title: "Board", Nav: "home", User: &User{Name: "Ada", Role: "owner"}})
	body := rec.Body.String()
	if !strings.Contains(body, `class="guide-fab"`) || !strings.Contains(body, "Help?") || !strings.Contains(body, "/admin/guide/ask") {
		t.Fatal("signed-in layout needs Help?")
	}
	if !strings.Contains(body, "Fill it or ask") || !strings.Contains(body, "submitDraft") {
		t.Fatal("Help? must walk page forms")
	}
	if !strings.Contains(body, `data-room="home"`) || !strings.Contains(body, `class="now"`) {
		t.Fatal("board must be a distinct room with a now strip")
	}
	if !strings.Contains(body, "Check each card") || !strings.Contains(body, `class="recipe"`) {
		t.Fatal("board must list numbered steps")
	}
	deliv := httptest.NewRecorder()
	s.render(deliv, "delivery", pageData{Title: "Delivery", Nav: "delivery", User: &User{Name: "Ada", Role: "owner"}})
	if !strings.Contains(deliv.Body.String(), "Close today’s list") || !strings.Contains(deliv.Body.String(), `class="recipe"`) {
		t.Fatal("delivery must list numbered steps")
	}
	if !strings.Contains(body, `href="/admin/guide"`) {
		t.Fatal("office Guide link")
	}
	if !strings.Contains(body, `href="/admin/files"`) {
		t.Fatal("office Files link")
	}
	if !strings.Contains(body, `href="/admin/grants"`) {
		t.Fatal("office Grants link")
	}
	if !strings.Contains(body, `href="/admin/crm"`) {
		t.Fatal("office CRM link")
	}
	crmPage := httptest.NewRecorder()
	s.render(crmPage, "crm", pageData{
		Title: "CRM", Nav: "crm", User: &User{Name: "Ada", Role: "owner"},
		Contacts: []CRMContact{{ID: 1, Name: "Chinedu", Company: "Apex", Stage: "new"}},
	})
	crmBody := crmPage.Body.String()
	if !strings.Contains(crmBody, "People and the story so far") || !strings.Contains(crmBody, "Chinedu") || !strings.Contains(crmBody, `href="/admin/crm/1"`) {
		t.Fatal("crm book")
	}
	person := httptest.NewRecorder()
	s.render(person, "crm", pageData{
		Title: "Chinedu", Nav: "crm", User: &User{Name: "Ada", Role: "owner"},
		Contact: &CRMContact{ID: 1, Name: "Chinedu", Company: "Apex", Stage: "qualified", Offer: "WhatsApp books"},
		CRMEvents: []CRMEvent{{Channel: "call", Direction: "in", Summary: "Asked for a demo"}},
	})
	if !strings.Contains(person.Body.String(), "Asked for a demo") || !strings.Contains(person.Body.String(), "Log a call") {
		t.Fatal("crm person timeline")
	}
	login := httptest.NewRecorder()
	s.render(login, "login", pageData{Title: "Sign in"})
	if strings.Contains(login.Body.String(), `class="guide-fab"`) {
		t.Fatal("login must not show Help?")
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
	if first := allDocs()[0]; first.Slug != "you" || !first.Player() || !first.Live() {
		t.Fatalf("first page %s kind=%s", first.Slug, first.Kind)
	}
	contract, ok := lookupDoc("contract")
	if !ok || contract.Player() || !contract.Worker() {
		t.Fatal("contract is a worker page")
	}
	you, ok := lookupDoc("you")
	if !ok || !you.Player() || !strings.Contains(string(you.Body), "Academy Telegram") || strings.Contains(string(you.Body), "task_description") {
		t.Fatal("you sheet")
	}
	own, ok := lookupDoc("owner")
	if !ok || !strings.Contains(string(own.Body), "People") || strings.Contains(string(own.Body), "POST /departments") {
		t.Fatal("owner sheet")
	}
	face, ok := lookupDoc("project-face")
	if !ok || !strings.Contains(string(face.Body), "not a director") || !strings.Contains(string(face.Body), "/idea") {
		t.Fatal("project-face sheet")
	}
	field, ok := lookupDoc("field")
	if !ok || !strings.Contains(string(field.Body), "names") {
		t.Fatal("field sheet")
	}
	handler, ok := lookupDoc("handler")
	if !ok || !strings.Contains(string(handler.Body), "Chat does") {
		t.Fatal("handler sheet")
	}
	desk, ok := lookupDoc("desk")
	if !ok || !desk.Live() || !strings.Contains(string(desk.Body), "People") {
		t.Fatal("desk playbook")
	}
	proj, ok := lookupDoc("projects")
	if !ok || !proj.Live() || !strings.Contains(string(proj.Body), "this-week") || !strings.Contains(string(proj.Body), "Dates") {
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
	deliv, ok := lookupDoc("delivery")
	if !ok || !deliv.Live() || !strings.Contains(string(deliv.Body), "hours/week") || !strings.Contains(string(deliv.Body), "Fathom") {
		t.Fatal("delivery playbook")
	}
	seoDoc, ok := lookupDoc("seo")
	if !ok || !seoDoc.Partial() || !strings.Contains(string(seoDoc.Body), "Search Console") {
		t.Fatal("seo playbook")
	}
	cm, ok := lookupDoc("community")
	if !ok || !strings.Contains(string(cm.Body), "lms.themobileprof.com") {
		t.Fatal("community LMS specimen")
	}
	crmDoc, ok := lookupDoc("crm")
	if !ok || !strings.Contains(string(crmDoc.Body), "Office → CRM") {
		t.Fatal("crm playbook")
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
