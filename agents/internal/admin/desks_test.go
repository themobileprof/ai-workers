package admin

import "testing"

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
	if _, ok := DeskByDepartment("internal-ops"); !ok {
		t.Fatal("ops")
	}
}

func TestHandles(t *testing.T) {
	owner := User{Role: "owner"}
	if !Handles(owner, "legal") || !Handles(owner, "hr") {
		t.Fatal("owner handles all")
	}
	handler := User{Role: "bdm", Desks: []string{"hr", "growth"}}
	if !Handles(handler, "hr") || Handles(handler, "legal") {
		t.Fatal("assigned desks only")
	}
	if Handles(handler, "") {
		t.Fatal("empty")
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
