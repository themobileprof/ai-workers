package admin

import (
	"strings"
	"testing"

	"github.com/samuel/ai-workers/agents/internal/departments/legal"
	"github.com/samuel/ai-workers/agents/internal/journeys"
)

func TestNormalizeSlug(t *testing.T) {
	got := normalizeSlug("  Lagos Mechanic Books!! ")
	if got != "lagos-mechanic-books" {
		t.Fatalf("slug %q", got)
	}
	if normalizeSlug("___") != "" {
		t.Fatal("empty after strip")
	}
}

func TestCleanProject(t *testing.T) {
	_, err := cleanProject("", "", "x", "idea", "", "", "", "")
	if err == nil {
		t.Fatal("name required")
	}
	_, err = cleanProject("Apex", "", "", "idea", "", "", "", "")
	if err == nil {
		t.Fatal("one-liner required")
	}
	p, err := cleanProject("Apex Clerk", "", "Workshops pay for a WhatsApp clerk.", "", "", "momlaunchpad.com", "", "")
	if err != nil {
		t.Fatal(err)
	}
	if p.Slug != "apex-clerk" || p.Stage != "idea" || p.URL != "https://momlaunchpad.com" {
		t.Fatalf("%+v", p)
	}
	if _, err := cleanProject("Apex", "apex", "line", "ipo", "", "", "", ""); err == nil {
		t.Fatal("unknown stage")
	}
	placed, err := cleanProject("MomLaunchpad", "momlaunchpad", "line", "idea", "", "", "consumer_subscription", "paid_conversion")
	if err != nil {
		t.Fatal(err)
	}
	if placed.Stage != "selling" {
		t.Fatalf("gate should set stage, got %s", placed.Stage)
	}
	if _, err := cleanProject("X", "x", "line", "idea", "", "", "consumer_subscription", "first_funded"); err == nil {
		t.Fatal("gate from another journey")
	}
}

func TestDefaultProjects(t *testing.T) {
	seen := map[string]bool{}
	want := map[string]string{
		"momlaunchpad": "selling",
		"academy":      "selling",
		"finchest":     "testing",
		"homegauge":    "testing",
		"mechazone":    "testing",
	}
	got := defaultProjects()
	if len(got) != 5 {
		t.Fatalf("len %d", len(got))
	}
	for _, p := range got {
		if seen[p.Slug] {
			t.Fatalf("dup slug %s", p.Slug)
		}
		seen[p.Slug] = true
		if want[p.Slug] != p.Stage {
			t.Fatalf("%s stage %s", p.Slug, p.Stage)
		}
		if _, err := cleanProject(p.Name, p.Slug, p.OneLiner, p.Stage, p.Notes, p.URL, p.Journey, p.Gate); err != nil {
			t.Fatalf("%s: %v", p.Slug, err)
		}
		if p.Journey == "" || p.Gate == "" {
			t.Fatalf("%s missing placement", p.Slug)
		}
	}
	for slug := range want {
		if !seen[slug] {
			t.Fatalf("missing %s", slug)
		}
	}
}

func TestProposalFromStructured(t *testing.T) {
	_, err := proposalFromStructured(map[string]any{"suggested_journey": "shop_ledger", "suggested_gate": "design_partner"})
	if err == nil {
		t.Fatal("mission required")
	}
	p, err := proposalFromStructured(map[string]any{
		"suggested_journey": "shop_ledger",
		"suggested_gate":    "design_partner",
		"mission":           "Get one shop on the installer.",
		"confidence":        70.0,
	})
	if err != nil {
		t.Fatal(err)
	}
	if p.Confidence != 70 || p.Journey != "shop_ledger" {
		t.Fatalf("%+v", p)
	}
	if _, err := proposalFromStructured(map[string]any{
		"suggested_journey": "shop_ledger",
		"suggested_gate":    "paid_conversion",
		"mission":           "nope",
	}); err == nil {
		t.Fatal("cross-journey gate")
	}
}

func TestProjectStagesAndSeats(t *testing.T) {
	if !validStage("fundable") || validStage("series-a") {
		t.Fatal("stages")
	}
	if !validSeat("cofounder") || validSeat("director") {
		t.Fatal("seats")
	}
	if lab, ok := stageByID("testing"); !ok || lab.Label != "Testing" {
		t.Fatal("testing label")
	}
	roles := deskRoles()
	if roles[0] != "owner" || roles[1] != "bdm" {
		t.Fatalf("roles %v", roles)
	}
}

func TestTemplatesIncludeProjects(t *testing.T) {
	srv, err := New(nil, "")
	if err != nil {
		t.Fatal(err)
	}
	if srv.pages["projects"] == nil || srv.pages["legal"] == nil || srv.pages["hr"] == nil || srv.pages["desks"] == nil {
		t.Fatal("projects template missing")
	}
	user := &User{ID: 1, Name: "Sam", Role: "owner"}
	p := Project{ID: 7, Name: "Apex Clerk", Slug: "apex-clerk", OneLiner: "Workshops pay for a clerk.", Stage: "testing", Notes: "Five interviews"}
	p.Members = []ProjectMember{{UserID: 2, Name: "Ada", Seat: "cofounder", Role: "cofounder"}}
	p.Proposal = &Proposal{Journey: "shop_ledger", Gate: "design_partner", Mission: "One shop.", Confidence: 60}
	cases := []struct {
		name string
		page string
		data pageData
		want string
	}{
		{"home", "home", pageData{Title: "Board", Nav: "home", User: user, Projects: []Project{p}, UserCount: 2}, "Apex Clerk"},
		{"list", "projects", pageData{Title: "Projects", Nav: "projects", User: user, Projects: []Project{p}, Stages: projectStages()}, "proposal"},
		{"new", "projects", pageData{Title: "Add", Nav: "projects", User: user, Adding: true, Stages: projectStages(), Journeys: journeys.All()}, "consumer_subscription"},
		{"edit", "projects", pageData{Title: "Amend", Nav: "projects", User: user, Project: &p, Roster: []User{*user}, Stages: projectStages(), Seats: projectSeats(), Journeys: journeys.All(), CanPlace: true}, "Ada"},
		{"proposal", "projects", pageData{Title: "Amend", Nav: "projects", User: user, Project: &p, Roster: []User{*user}, Stages: projectStages(), Seats: projectSeats(), Journeys: journeys.All()}, "has not moved the stamp"},
		{"ask-draft", "projects", pageData{Title: "Amend", Nav: "projects", User: user, Project: &p, Roster: []User{*user}, Stages: projectStages(), Seats: projectSeats(), Journeys: journeys.All(), CanLegal: true, LegalTemplates: legal.Specs()}, "Ask draft"},
		{"legal", "legal", pageData{Title: "Legal", Nav: "legal", User: user, Drafts: []LegalDraft{{ID: 3, ProjectID: 7, ProjectName: "Apex Clerk", Title: "NDA", Template: "nda", Status: "proposed", Body: "TheMobileProf Technologies", CounterpartyName: "Ada"}}, CanLegal: true}, "Apex Clerk"},
		{"hr", "hr", pageData{Title: "HR", Nav: "hr", User: user, HrRoles: []HrRole{{ID: 4, Title: "Field intern", Kind: "intern", Template: "intern", Status: "proposed", JD: "TheMobileProf Technologies intern"}}, CanHR: true}, "Field intern"},
		{"desk", "desks", pageData{Title: "Legal", Nav: "desk-legal", User: user, Desk: &Desk{Slug: "legal", Title: "Legal", Prefix: "/legal", Help: "NDA"}, CanAsk: true, Jobs: []DeskJob{{ID: 9, Title: "NDA Ada", Status: "proposed", Source: "desk"}}}, "NDA Ada"},
		{"users-new", "users", pageData{Title: "Add", Nav: "users", User: user, Adding: true, Roles: deskRoles()}, "bdm"},
	}
	for _, tc := range cases {
		var buf strings.Builder
		if err := srv.pages[tc.page].ExecuteTemplate(&buf, "layout.html", tc.data); err != nil {
			t.Fatalf("%s: %v", tc.name, err)
		}
		if !strings.Contains(buf.String(), tc.want) {
			t.Fatalf("%s missing %q", tc.name, tc.want)
		}
		if strings.Contains(buf.String(), "/admin/integrations") {
			t.Fatalf("%s still links to the removed Tools page", tc.name)
		}
	}
}

func TestBoardHasThreeTabs(t *testing.T) {
	srv, err := New(nil, "")
	if err != nil {
		t.Fatal(err)
	}
	user := &User{ID: 1, Name: "Sam", Role: "owner"}
	p := Project{ID: 7, Name: "Apex Clerk", Stage: "testing"}
	pages := []struct {
		page string
		data pageData
		min  int
	}{
		{"home", pageData{Title: "Board", Nav: "home", User: user, Projects: []Project{p}, UserCount: 2}, 3},
		{"hr", pageData{Title: "HR", Nav: "hr", User: user, CanHR: true}, 3},
		{"legal", pageData{Title: "Legal", Nav: "legal", User: user}, 3},
		{"desks", pageData{Title: "Legal", Nav: "desk-legal", User: user, Desk: &Desk{Slug: "legal", Title: "Legal", Prefix: "/legal"}, CanAsk: true}, 3},
		{"projects", pageData{Title: "Amend", Nav: "projects", User: user, Project: &p, Stages: projectStages(), Seats: projectSeats(), Journeys: journeys.All()}, 3},
		{"settings", pageData{Title: "Defaults", Nav: "settings", User: user, SettingGroups: settingGroupsFrom(map[string]string{})}, 3},
		{"users", pageData{Title: "Add", Nav: "users", User: user, Adding: true, Roles: deskRoles()}, 3},
	}
	for _, tc := range pages {
		var buf strings.Builder
		if err := srv.pages[tc.page].ExecuteTemplate(&buf, "layout.html", tc.data); err != nil {
			t.Fatalf("%s: %v", tc.page, err)
		}
		got := buf.String()
		if !strings.Contains(got, `data-tabs`) {
			t.Fatalf("%s missing tabset", tc.page)
		}
		if n := strings.Count(got, `role="tab"`); n < tc.min {
			t.Fatalf("%s tabs %d, want >= %d", tc.page, n, tc.min)
		}
	}
}

func TestWithQueryAndStatusCount(t *testing.T) {
	got := withQuery("/admin/hr?ok=role_open", "tab", "roles")
	if !strings.Contains(got, "tab=roles") || !strings.Contains(got, "ok=role_open") {
		t.Fatalf("%s", got)
	}
	if statusCount([]LegalDraft{{Status: "proposed"}, {Status: "approved"}}, "proposed") != 1 {
		t.Fatal("legal count")
	}
	if statusCount([]Integration{{Status: "live"}, {Status: "host"}}, "live") != 1 {
		t.Fatal("tools count")
	}
}
