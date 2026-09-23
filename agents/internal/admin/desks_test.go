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
	if _, ok := LookupDesk("hr"); !ok {
		t.Fatal("hr")
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
	if Handles(owner, "legal") || Handles(owner, "hr") {
		t.Fatal("owner does not implicitly handle desks")
	}
	assigned := User{Role: "owner", Desks: []string{"hr"}}
	if !Handles(assigned, "hr") || Handles(assigned, "legal") {
		t.Fatal("assignment only")
	}
	handler := User{Role: "bdm", Desks: []string{"hr", "growth"}}
	if !Handles(handler, "hr") || Handles(handler, "legal") {
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
	owner := &User{Role: "owner", Desks: []string{"hr"}}
	if !isOwner(owner) || !canOffice(owner) {
		t.Fatal("owner")
	}
	bdm := &User{Role: "bdm"}
	if isOwner(bdm) || !canOffice(bdm) {
		t.Fatal("bdm")
	}
	got := visibleDesks(owner)
	if len(got) != 1 || got[0].Slug != "hr" {
		t.Fatalf("%+v", got)
	}
	if visibleDesks(nil) != nil {
		t.Fatal("nil desks")
	}
}

func TestAfterLoginPath(t *testing.T) {
	owner := User{Role: "owner", Desks: []string{"hr"}}
	if afterLoginPath(owner) != "/admin/" {
		t.Fatal("office lands on Board")
	}
	one := User{Role: "cofounder", Desks: []string{"hr"}}
	if afterLoginPath(one) != "/admin/desks/hr" {
		t.Fatalf("got %s", afterLoginPath(one))
	}
	many := User{Role: "assistant", Desks: []string{"hr", "legal"}}
	if afterLoginPath(many) != "/admin/desks" {
		t.Fatal("picker")
	}
	none := User{Role: "viewer"}
	if afterLoginPath(none) != "/admin/desks" {
		t.Fatal("empty handler must not loop on Board")
	}
}

func TestDeskNavOn(t *testing.T) {
	desks := []Desk{{Slug: "hr"}}
	if deskNavOn("home", desks) || !deskNavOn("desk-hr", desks) || !deskNavOn("hr", desks) {
		t.Fatal("desk nav")
	}
	if officeMenuOn("home") || !officeMenuOn("settings") || !officeMenuOn("flows") {
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
	if shouldFileInboundJob("hr", map[string]any{"channel": "email"}, map[string]any{}) {
		t.Fatal("email hr is filed via applications POST")
	}
	if !shouldFileInboundJob("community", map[string]any{"channel": "whatsapp"}, map[string]any{"escalate_to_founder": true}) {
		t.Fatal("escalate")
	}
	if shouldFileInboundJob("growth", map[string]any{"channel": "whatsapp"}, map[string]any{"task_type": "sales"}) {
		t.Fatal("growth chatter")
	}
}

func TestWithLoginCap(t *testing.T) {
	got := withLoginCap([]string{"whatsapp_accounts"}, []string{"hr"})
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
	u := User{Desks: []string{"hr"}}
	if !assignedDesk(u, "hr") || assignedDesk(u, "legal") {
		t.Fatal("assigned")
	}
	if assignedDesk((*User)(nil), "hr") {
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
		Title:    "HR",
		Nav:      "desk-hr",
		User:     &User{ID: 2, Name: "Ada", Role: "cofounder"},
		NavDesks: []Desk{{Slug: "hr", Title: "HR"}},
		Desk:     &Desk{Slug: "hr", Title: "HR", Prefix: "/hr"},
	}
	buf.Reset()
	if err := srv.pages["desks"].ExecuteTemplate(&buf, "layout.html", handler); err != nil {
		t.Fatal(err)
	}
	got = buf.String()
	if strings.Contains(got, `data-nav="home"`) || strings.Contains(got, `href="/admin/projects"`) {
		t.Fatal("handler should not see office nav")
	}
	if !strings.Contains(got, "/admin/desks/hr") || !strings.Contains(got, "Workers") {
		t.Fatal("assigned desk")
	}
}
