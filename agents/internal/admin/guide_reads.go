package admin

// guideReads returns numbered “Read” pointers for a worker slug or book page.
func guideReads(kind string) []DeskLink {
	var out []DeskLink
	if meta, ok := lookupDoc(kind); ok {
		out = append(out, DeskLink{
			Href:  meta.Href(),
			Label: meta.Title + " sheet",
			Why:   shortDocWhy(kind, meta.Summary),
		})
	}
	switch kind {
	case "accounts":
		out = append(out,
			DeskLink{Href: "/admin/settings?tab=zoho", Label: "Zoho ids", Why: "Chart of accounts numbers n8n uses on every Books write."},
		)
	case "community":
		out = append(out,
			DeskLink{Href: "/admin/community", Label: "Mandates", Why: "One form per Academy Telegram room. Not campus trainings."},
		)
	case "legal":
		out = append(out,
			DeskLink{Href: "/admin/legal", Label: "Legal book", Why: "Accept lives here. Copy the text; it does not email anyone."},
		)
	case "seo":
		out = append(out,
			DeskLink{Href: "/admin/seo", Label: "SEO book", Why: "Accept lives here. You paste onto the live page."},
		)
	case "delivery":
		out = append(out,
			DeskLink{Href: "/admin/delivery", Label: "Delivery", Why: "Today’s list. Done or Skip. Ask plan, then Accept on Job chat."},
			DeskLink{Href: "/admin/docs/project-face", Label: "Project face sheet", Why: "Seat on Face before someone gets work on this bet."},
			DeskLink{Href: "/admin/guide", Label: "FAQ · meeting notes", Why: "Fathom files a proposed plan. Accept here. Chat never writes the board."},
		)
	case "product-dev":
		out = append(out,
			DeskLink{Href: "/admin/projects", Label: "Projects", Why: "Run walks the base kit. Accept on Decide writes the stage."},
		)
	case "files":
		out = append(out,
			DeskLink{Href: "/admin/settings?tab=drive", Label: "Shared Drive id", Why: "Paste the id on this Defaults tab before Accept writes Drive."},
			DeskLink{Href: "/admin/docs/desk", Label: "Company desk sheet", Why: "Files is an Office book for owner/BDM, not a handler desk."},
		)
	case "hr":
		out = append(out,
			DeskLink{Href: "/admin/users", Label: "People", Why: "Roster and login. Applicants are not that."},
			DeskLink{Href: "/admin/guide", Label: "FAQ", Why: "Recruiting is closed. Inbound CVs go to Growth, not a hire."},
		)
	case "handler", "desks":
		out = append(out,
			DeskLink{Href: "/admin/docs", Label: "Docs index", Why: "How you fit, then the sheet that matches your People card."},
		)
	}
	if kind != "hr" && kind != "files" {
		out = append(out, DeskLink{
			Href:  "/admin/docs/handler",
			Label: "Running a worker",
			Why:   "Ask, wait, then Accept. Chat and Help? never Accept.",
		})
	}
	out = append(out, DeskLink{
		Href:  "/admin/docs/you",
		Label: "How you fit",
		Why:   "Which desks you see after login.",
	})
	return dedupeLinks(out)
}

func shortDocWhy(kind, summary string) string {
	switch kind {
	case "accounts":
		return "What the classifier may name. It never computes VAT."
	case "growth":
		return "Replies and copy when you ask. It cannot send WhatsApp or email itself."
	case "community":
		return "Academy Telegram rooms. WhatsApp Cloud API cannot join ordinary groups."
	case "crm":
		return "Qualify or update a person. info@ files a thread summary. n8n writes Books contacts, not Zoho CRM."
	case "legal":
		return "Specimens for NDAs. The worker cannot send the document."
	case "seo":
		return "Titles, meta, outlines. The worker cannot publish."
	case "delivery":
		return "Stage caps how many tasks. Fathom notes still need Accept."
	case "internal-ops":
		return "/ops reaches the worker. /ops legal stays chat-only; filing is /legal."
	case "product-dev":
		return "/validate proposes a gate. You Accept on Decide."
	case "files":
		return "Ops Telegram /file proposes a path. n8n writes Drive after Accept."
	case "handler":
		return "Ask, wait, then Accept. Chat never Accepts."
	default:
		if summary != "" {
			return summary
		}
		return "Specimens and what this worker must not do."
	}
}

func prependLink(out []DeskLink, first DeskLink) []DeskLink {
	return dedupeLinks(append([]DeskLink{first}, out...))
}

func dedupeLinks(in []DeskLink) []DeskLink {
	seen := map[string]bool{}
	var out []DeskLink
	for _, l := range in {
		if l.Href == "" || seen[l.Href] {
			continue
		}
		seen[l.Href] = true
		out = append(out, l)
	}
	return out
}
