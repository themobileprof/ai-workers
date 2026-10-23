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

// DeskLink is a specific Docs or book pointer shown on a worker or book page.
type DeskLink struct {
	Href  string
	Label string
	Why   string
}

func AllDesks() []Desk {
	return []Desk{
		{Slug: "accounts", Title: "Accounts", Department: "accounts", Prefix: "/accounts", Help: "This worker names the Zoho document. It does not compute VAT or withhold tax. Confirm what Books already posted, then Accept.", BookPath: "/admin/settings?tab=zoho", BookLabel: "Zoho ids"},
		{Slug: "growth", Title: "Growth", Department: "growth", Prefix: "/growth", Help: "Drafts replies and front-office copy when you ask. It cannot send WhatsApp or email itself. Inbound info@ is filed by CRM, not auto-replied."},
		{Slug: "community", Title: "Community", Department: "community", Prefix: "/cm", Help: "Voice for one Academy Telegram room at a time. Campus trainings are project Dates, not rooms. Step in when the worker sets escalate_to_founder.", BookPath: "/admin/community", BookLabel: "Mandates"},
		{Slug: "crm", Title: "CRM", Department: "crm", Prefix: "/crm", Help: "Conversion book: a person, an offer, and a timeline of email, WhatsApp, and calls. It cannot send.", BookPath: "/admin/crm", BookLabel: "CRM book"},
		{Slug: "legal", Title: "Legal", Department: "legal", Prefix: "/legal", Help: "Proposes NDAs and contracts. Accept still lives on the Legal book. Copy the text yourself — it does not email anyone.", BookPath: "/admin/legal", BookLabel: "Legal book"},
		{Slug: "seo", Title: "SEO", Department: "seo", Prefix: "/seo", Help: "Fetches the bet’s own URL. The visible page is the source of truth. Isolates keywords, customizes body and chrome together, writes an onsite pack for Cursor. Ask issues the pack URL. Offsite work is a Delivery week plan that writes after the delay unless you Turn away. It cannot publish, send, or spy on competitors.", BookPath: "/admin/seo", BookLabel: "SEO book"},
		{Slug: "delivery", Title: "Delivery", Department: "delivery", Prefix: "/today", Help: "Tasks across bets. Stage caps how many; extras sit on Backlog. Plans write after a delay unless you Turn away. Fathom notes use the same path. Chat never Accepts.", BookPath: "/admin/delivery", BookLabel: "Delivery"},
		{Slug: "internal-ops", Title: "Ops", Department: "internal-ops", Prefix: "/ops", Help: "Operations room. Accounts via /ops still writes Books. Grant answers write a deadline row. They do not submit an application."},
		{Slug: "product-dev", Title: "Product", Department: "product-dev", Prefix: "/validate", Help: "Placement and validation. Accept on the project Decide tab writes the stage. Chat never stamps.", BookPath: "/admin/projects", BookLabel: "Projects"},
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

// GuideLinks are the Docs and books this desk should point at — not a generic Sheet button.
func (d Desk) GuideLinks() []DeskLink {
	out := guideReads(d.Slug)
	if d.BookPath != "" {
		out = prependLink(out, DeskLink{Href: d.BookPath, Label: d.BookLabel, Why: "The page where this work is accepted."})
	}
	return out
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
	if canOffice(u) && !Handles(*u, "delivery") {
		if d, ok := LookupDesk("delivery"); ok {
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
	return nav == "settings" || nav == "docs" || nav == "flows" || nav == "guide" || nav == "files" || nav == "grants" || nav == "crm"
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
