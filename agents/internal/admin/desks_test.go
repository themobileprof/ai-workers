package admin

import (
	"strings"
	"testing"
)

func TestAllDesksUnique(t *testing.T) {
	seen := map[string]bool{}
	for _, d := range AllDesks() {
		if d.Slug == "" || d.Department == "" || seen[d.Slug] {
			t.Fatalf("desk %+v", d)
		}
		seen[d.Slug] = true
		if d.Path() != "/admin/desks/"+d.Slug {
			t.Fatalf("path %s", d.Path())
		}
	}
	if _, ok := LookupDesk("delivery"); !ok {
		t.Fatal("delivery")
	}
	if _, ok := LookupDesk("hr"); ok {
		t.Fatal("hr desk removed")
	}
	if _, ok := LookupDesk("seo"); !ok {
		t.Fatal("seo")
	}
	cm, ok := LookupDesk("community")
	if !ok || cm.BookPath != "/admin/community" {
		t.Fatal("community mandates book")
	}
	if _, ok := DeskByDepartment("internal-ops"); !ok {
		t.Fatal("ops")
	}
	if _, ok := LookupDesk("missing"); ok {
		t.Fatal("unknown desk")
	}
	if _, ok := DeskByDepartment("nope"); ok {
		t.Fatal("unknown dept")
	}
}

func TestHandles(t *testing.T) {
	owner := User{Role: "owner"}
	if Handles(owner, "legal") || Handles(owner, "delivery") {
		t.Fatal("owner does not implicitly handle desks")
	}
	assigned := User{Role: "owner", Desks: []string{"delivery"}}
	if !Handles(assigned, "delivery") || Handles(assigned, "legal") {
		t.Fatal("assignment only")
	}
	handler := User{Role: "bdm", Desks: []string{"delivery", "growth"}}
	if !Handles(handler, "delivery") || Handles(handler, "legal") {
		t.Fatal("assigned desks only")
	}
	if Handles(handler, "") {
		t.Fatal("empty")
	}
}

func TestIsOwnerAndVisibleDesks(t *testing.T) {
	if isOwner(nil) || canOffice(nil) {
		t.Fatal("nil")
	}
	owner := &User{Role: "owner", Desks: []string{"legal"}}
	if !isOwner(owner) || !canOffice(owner) {
		t.Fatal("owner")
	}
	bdm := &User{Role: "bdm"}
	if isOwner(bdm) || !canOffice(bdm) {
		t.Fatal("bdm")
	}
	got := visibleDesks(owner)
	slugs := map[string]bool{}
	for _, d := range got {
		slugs[d.Slug] = true
	}
	if !slugs["legal"] || !slugs["delivery"] {
		t.Fatalf("%+v", got)
	}
	if visibleDesks(nil) != nil {
		t.Fatal("nil desks")
	}
}

func TestAfterLoginPath(t *testing.T) {
	owner := User{Role: "owner", Desks: []string{"legal"}}
	if afterLoginPath(owner) != "/admin/" {
		t.Fatal("office lands on Board")
	}
	one := User{Role: "cofounder", Desks: []string{"legal"}}
	if afterLoginPath(one) != "/admin/desks/legal" {
		t.Fatalf("got %s", afterLoginPath(one))
	}
	many := User{Role: "assistant", Desks: []string{"growth", "legal"}}
	if afterLoginPath(many) != "/admin/desks" {
		t.Fatal("picker")
	}
	none := User{Role: "viewer"}
	if afterLoginPath(none) != "/admin/desks" {
		t.Fatal("empty handler must not loop on Board")
	}
}

func TestDeskNavOn(t *testing.T) {
	desks := []Desk{{Slug: "legal"}}
	if deskNavOn("home", desks) || !deskNavOn("desk-legal", desks) || !deskNavOn("legal", desks) {
		t.Fatal("desk nav")
	}
	if officeMenuOn("home") || !officeMenuOn("settings") || !officeMenuOn("flows") || !officeMenuOn("guide") || !officeMenuOn("files") {
		t.Fatal("office menu")
	}
}

func TestShouldFileInboundJob(t *testing.T) {
	if shouldFileInboundJob("legal", map[string]any{"channel": "desk", "action": "draft"}, map[string]any{}) {
		t.Fatal("desk channel stays on the HTTP insert")
	}
	if shouldFileInboundJob("legal", map[string]any{"channel": "telegram", "action": "watch"}, map[string]any{}) {
		t.Fatal("watch")
	}
	if !shouldFileInboundJob("legal", map[string]any{"channel": "telegram"}, map[string]any{}) {
		t.Fatal("telegram legal")
	}
	if !shouldFileInboundJob("seo", map[string]any{"channel": "whatsapp"}, map[string]any{"save_brief": true}) {
		t.Fatal("seo inbound")
	}
	if shouldFileInboundJob("delivery", map[string]any{"channel": "whatsapp"}, map[string]any{"task_type": "today"}) {
		t.Fatal("delivery ping without save_plan")
	}
	if !shouldFileInboundJob("delivery", map[string]any{"channel": "whatsapp"}, map[string]any{"save_plan": true}) {
		t.Fatal("delivery plan")
	}
	if !shouldFileInboundJob("community", map[string]any{"channel": "whatsapp"}, map[string]any{"escalate_to_founder": true}) {
		t.Fatal("escalate")
	}
	if shouldFileInboundJob("growth", map[string]any{"channel": "whatsapp"}, map[string]any{"task_type": "sales"}) {
		t.Fatal("growth chatter")
	}
}

func TestWithLoginCap(t *testing.T) {
	got := withLoginCap([]string{"whatsapp_accounts"}, []string{"legal"})
	if len(got) != 2 {
		t.Fatalf("%v", got)
	}
	keep := withLoginCap([]string{"web_admin"}, []string{"legal"})
	if len(keep) != 1 {
		t.Fatalf("%v", keep)
	}
}

func TestDeskFlash(t *testing.T) {
	if deskFlash("asked") == "" || deskFlash("job_ok") == "" {
		t.Fatal("flash")
	}
}

func TestAssignedDesk(t *testing.T) {
	u := User{Desks: []string{"delivery"}}
	if !assignedDesk(u, "delivery") || assignedDesk(u, "legal") {
		t.Fatal("assigned")
	}
	if assignedDesk((*User)(nil), "delivery") {
		t.Fatal("nil")
	}
}

func TestLayoutOfficeVsHandlerNav(t *testing.T) {
	srv, err := New(nil, "")
	if err != nil {
		t.Fatal(err)
	}
	office := pageData{Title: "Board", Nav: "home", User: &User{ID: 1, Name: "Sam", Role: "owner"}, CanOffice: true, CanPeople: true}
	var buf strings.Builder
	if err := srv.pages["home"].ExecuteTemplate(&buf, "layout.html", office); err != nil {
		t.Fatal(err)
	}
	got := buf.String()
	if !strings.Contains(got, `data-nav="home"`) || !strings.Contains(got, "/admin/users") {
		t.Fatal("office nav")
	}
	if !strings.Contains(got, "Office") || !strings.Contains(got, `href="/admin/settings"`) {
		t.Fatal("office dropdown")
	}
	if strings.Contains(got, "/admin/desks/hr") || strings.Contains(got, "/admin/desks/legal") {
		t.Fatal("unassigned worker on office nav")
	}
	handler := pageData{
		Title:    "Legal",
		Nav:      "desk-legal",
		User:     &User{ID: 2, Name: "Ada", Role: "cofounder"},
		NavDesks: []Desk{{Slug: "legal", Title: "Legal"}},
		Desk:     &Desk{Slug: "legal", Title: "Legal", Prefix: "/legal"},
	}
	buf.Reset()
	if err := srv.pages["desks"].ExecuteTemplate(&buf, "layout.html", handler); err != nil {
		t.Fatal(err)
	}
	got = buf.String()
	if strings.Contains(got, `data-nav="home"`) || strings.Contains(got, `href="/admin/projects"`) {
		t.Fatal("handler should not see office nav")
	}
	if !strings.Contains(got, "/admin/desks/legal") || !strings.Contains(got, "Workers") {
		t.Fatal("assigned desk")
	}
}
