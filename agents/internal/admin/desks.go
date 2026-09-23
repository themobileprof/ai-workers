package admin

// Desk is a handler dashboard for one department worker. Catalog lives in git.
type Desk struct {
	Slug       string
	Title      string
	Department string
	Prefix     string
	Help       string
	BookPath   string
	BookLabel  string
}

func AllDesks() []Desk {
	return []Desk{
		{Slug: "accounts", Title: "Accounts", Department: "accounts", Prefix: "/accounts", Help: "Books classifier. Handler confirms what Zoho already posted or a lookup the worker flagged.", BookPath: "/admin/settings", BookLabel: "Defaults"},
		{Slug: "growth", Title: "Growth", Department: "growth", Prefix: "/growth", Help: "Front office copy and replies. Handler complements drafts that must not send themselves."},
		{Slug: "community", Title: "Community", Department: "community", Prefix: "/cm", Help: "Room voice with a mandate per WhatsApp group. Handler steps in when the worker sets escalate_to_founder.", BookPath: "/admin/community", BookLabel: "Mandates"},
		{Slug: "crm", Title: "CRM", Department: "crm", Prefix: "/crm", Help: "Pipeline notes. Handler confirms a contact the worker asked a human to review."},
		{Slug: "legal", Title: "Legal", Department: "legal", Prefix: "/legal", Help: "NDA and contract proposals. Accept still lives on the Legal book.", BookPath: "/admin/legal", BookLabel: "Legal book"},
		{Slug: "hr", Title: "HR", Department: "hr", Prefix: "/hr", Help: "JDs and applicants. Open / Shortlist on the HR book. Chat does not hire.", BookPath: "/admin/hr", BookLabel: "HR book"},
		{Slug: "internal-ops", Title: "Ops", Department: "internal-ops", Prefix: "/ops", Help: "Operations room. Accounts via /ops still writes Books; grants stay chat until you stamp a job."},
		{Slug: "product-dev", Title: "Product", Department: "product-dev", Prefix: "/validate", Help: "Placement and validation. Journey stamps stay on the project card.", BookPath: "/admin/projects", BookLabel: "Projects"},
	}
}

func LookupDesk(slug string) (Desk, bool) {
	for _, d := range AllDesks() {
		if d.Slug == slug {
			return d, true
		}
	}
	return Desk{}, false
}

func DeskByDepartment(dept string) (Desk, bool) {
	for _, d := range AllDesks() {
		if d.Department == dept {
			return d, true
		}
	}
	return Desk{}, false
}

func Handles(u User, slug string) bool {
	if slug == "" {
		return false
	}
	for _, d := range u.Desks {
		if d == slug {
			return true
		}
	}
	return false
}

func visibleDesks(u *User) []Desk {
	if u == nil {
		return nil
	}
	var out []Desk
	for _, d := range AllDesks() {
		if Handles(*u, d.Slug) {
			out = append(out, d)
		}
	}
	return out
}

func isOwner(u *User) bool {
	return u != nil && u.Role == "owner"
}

func canOffice(u *User) bool {
	if u == nil {
		return false
	}
	return u.Role == "owner" || u.Role == "bdm"
}

func deskNavOn(nav string, desks []Desk) bool {
	if nav == "desks" {
		return true
	}
	for _, d := range desks {
		if nav == "desk-"+d.Slug || nav == d.Slug {
			return true
		}
	}
	return false
}

func officeMenuOn(nav string) bool {
	return nav == "settings" || nav == "docs" || nav == "flows"
}

func afterLoginPath(u User) string {
	if canOffice(&u) {
		return "/admin/"
	}
	desks := visibleDesks(&u)
	if len(desks) == 1 {
		return desks[0].Path()
	}
	return "/admin/desks"
}
