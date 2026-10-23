package admin

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

func (d SeoBrief) PackDownloadPath() string {
	if d.ID <= 0 {
		return ""
	}
	return "/admin/seo/" + strconv.FormatInt(d.ID, 10) + "/pack.md"
}

func (d SeoBrief) PackPath() string {
	if d.PackToken == "" || d.Status == "rejected" {
		return ""
	}
	return "/site/seo/" + d.PackToken
}

func (d SeoBrief) PackFilename() string {
	slug := strings.ToLower(strings.TrimSpace(d.ProjectName))
	slug = strings.Map(func(r rune) rune {
		if r >= 'a' && r <= 'z' || r >= '0' && r <= '9' {
			return r
		}
		if r == ' ' || r == '-' {
			return '-'
		}
		return -1
	}, slug)
	slug = strings.Trim(slug, "-")
	if slug == "" {
		slug = "seo"
	}
	return fmt.Sprintf("%s-seo-%d.md", slug, d.ID)
}

func seoPackMarkdown(d SeoBrief, p Project, siblings []SeoBrief) string {
	var b strings.Builder
	b.WriteString("# SEO pack " + strconv.FormatInt(d.ID, 10) + "\n\n")
	name := strings.TrimSpace(p.Name)
	if name == "" {
		name = d.ProjectName
	}
	b.WriteString("Bet: **" + name + "**")
	if p.Slug != "" {
		b.WriteString(" (`" + p.Slug + "`)")
	}
	b.WriteString("\n\n")
	if d.URL != "" {
		b.WriteString("Start URL: " + d.URL + "\n\n")
	}
	if p.GitRepo != "" {
		b.WriteString("Repo to open in Cursor: `" + p.GitRepo + "` (clone it yourself — this office does not open a GitHub PR).\n\n")
	} else {
		b.WriteString("Open the site workspace in Cursor yourself. This pack does not push to GitHub.\n\n")
	}
	if p.OneLiner != "" {
		b.WriteString("One-liner: " + p.OneLiner + "\n\n")
	}
	b.WriteString("Kind: " + d.KindLabel() + " · status: " + d.Status + " · updated: " + d.UpdatedAt.UTC().Format(time.RFC3339) + "\n\n")
	b.WriteString("This pack does not publish, merge, or send. Do not invent search volume, rankings, impressions, or Core Web Vitals.\n\n")
	b.WriteString("The **visible page** is the source of truth (H1, headings, body, buttons). A designer may never have seen the HTML title or meta. Prefer **gap** (on the designed page, missing from the title tag). Do not trust **brochure** title-only phrases. Customize the visible copy *and* the chrome from what the user already saw.\n\n")

	b.WriteString("## Keyword inventory (from the designed page)\n\n")
	b.WriteString("Count is on-site visible-copy occurrences. Role: gap (on the page, not in the title tag) · aligned (page and title agree) · cannibal (same visible phrase on two URLs) · brochure (title/meta only — the designer may never have seen it).\n\n")
	if len(d.Inventory) == 0 {
		b.WriteString("No inventory stored. Fetch may have failed, or the pages had no usable phrases.\n\n")
	} else {
		for _, k := range d.Inventory {
			b.WriteString("- `" + k.Phrase + "` — " + k.Role)
			if k.BodyCount > 0 {
				b.WriteString(fmt.Sprintf(" · body count %d", k.BodyCount))
			} else if k.Count > 0 {
				b.WriteString(fmt.Sprintf(" · count %d", k.Count))
			}
			pages := k.BodyPages
			if len(pages) == 0 {
				pages = k.Pages
			}
			if len(pages) > 0 {
				b.WriteString(" · " + strings.Join(pages, ", "))
			}
			b.WriteString("\n")
		}
		b.WriteString("\n")
	}

	writeSeoWalkSections(&b, d, siblings)

	if d.Kind == "offsite" {
		b.WriteString("## Offsite work (via Delivery)\n\n")
		b.WriteString("The clerk cannot send mail, WhatsApp, or buy links. Offsite work is handed to **Delivery** as soon as this pack files, spread over weeks. Stage caps still apply; extras sit on Backlog. The plan writes after the delay unless a human Turns away on Job chat. Accept writes it now. Use the same legal name as the schema step (" + seoCompanyName + " unless the page names someone else).\n\n")
		writeSeoPackBody(&b, d)
		writeSeoPlanItems(&b, d)
	} else if d.Kind != "cite" && d.Kind != "faq" && d.Kind != "schema" {
		b.WriteString("## Onsite changes (implement in Cursor)\n\n")
		b.WriteString("Start from the page a designer already shipped. Propose copy they would recognize: same offer, place, and voice — clearer, in searcher language.\n\n")
		b.WriteString("1. **Visible first:** H1, subheads, body paragraphs, CTAs.\n")
		b.WriteString("2. **Chrome second:** title tag and meta description so a Google snippet matches that designed page.\n")
		b.WriteString("3. **Quotable facts, FAQ, schema:** implement those sections so an answer engine can cite the designed page. Do not invent a hidden GEO layer.\n")
		b.WriteString("Do not add offers, cities, or products the visible page does not already show. Do not leave SEO only in tags the designer never opens.\n\n")
		if d.Title != "" {
			b.WriteString("### Proposed title\n\n" + d.Title + "\n\n")
		}
		if d.Meta != "" {
			b.WriteString("### Proposed meta description\n\n" + d.Meta + "\n\n")
		}
		if d.H1 != "" {
			b.WriteString("### Proposed H1\n\n" + d.H1 + "\n\n")
		}
		if d.Keyword != "" {
			b.WriteString("Primary phrase for this brief: `" + d.Keyword + "`\n\n")
		}
		writeSeoPackBody(&b, d)
		if d.Kind != "inventory" {
			b.WriteString("## Offsite work (via Delivery)\n\n")
			b.WriteString("Do this after onsite ships. Ask **Offsite roadmap** if this pack has no task list. That Ask files a Delivery plan, spread over weeks, which writes after the delay unless Turn away.\n")
			b.WriteString("No cold outreach. No customer WhatsApp. No invented directories.\n\n")
			writeSeoPlanItems(&b, d)
		}
	} else {
		writeSeoPackBody(&b, d)
	}

	b.WriteString("## Cursor prompt\n\n")
	switch d.Kind {
	case "cite":
		b.WriteString("Put the **Quotable facts** where a designer would see them (body, not only meta). Do not invent a seventh fact.\n")
	case "faq":
		b.WriteString("Add the **Visible FAQ** as on-page Q&A, then FAQPage JSON-LD that matches that block exactly. Do not add questions the page does not already answer.\n")
	case "schema":
		b.WriteString("Add the **Proposed JSON-LD** to the matching templates. Organization is " + seoCompanyName + " unless the visible page names a different legal entity. Do not invent NAP, ratings, or offers.\n")
	default:
		b.WriteString("Implement the **Onsite changes** in this site’s codebase. Match existing templates and components. ")
		b.WriteString("Change the visible copy the designer already sees (body, H1, CTAs), then title and meta so they match. ")
		b.WriteString("Keep the quotable facts, FAQ, and JSON-LD in sync with that copy. ")
		b.WriteString("Do not invent pages, search volume, or Core Web Vitals. Do not commit unless a human asks. ")
		b.WriteString("Leave offsite tasks for a person — do not send mail or messages.\n")
	}
	return b.String()
}

func writeSeoWalkSections(b *strings.Builder, d SeoBrief, siblings []SeoBrief) {
	cite := walkSource(d, siblings, "cite")
	faq := walkSource(d, siblings, "faq")
	schema := walkSource(d, siblings, "schema")
	if cite.Body != "" && d.Kind != "cite" {
		b.WriteString("## Quotable facts (from the walk)\n\n")
		b.WriteString(cite.Body + "\n\n")
	}
	if faq.Body != "" && d.Kind != "faq" {
		b.WriteString("## FAQ (from the walk)\n\n")
		b.WriteString(faq.Body + "\n\n")
	}
	if schema.Body != "" && d.Kind != "schema" {
		b.WriteString("## Schema (from the walk)\n\n")
		b.WriteString(schema.Body + "\n\n")
	}
}

func walkSource(d SeoBrief, siblings []SeoBrief, kind string) SeoBrief {
	if d.Kind == kind {
		return d
	}
	return latestLiveSeo(siblings, kind)
}

func writeSeoPlanItems(b *strings.Builder, d SeoBrief) {
	if len(d.PlanItems) == 0 {
		return
	}
	b.WriteString("Proposed Delivery tasks:\n\n")
	for _, it := range d.PlanItems {
		b.WriteString("- " + it.Title)
		if it.Due != "" {
			b.WriteString(" · due " + it.Due)
		}
		if it.Hours > 0 {
			b.WriteString(fmt.Sprintf(" · %.0fh", it.Hours))
		}
		b.WriteString("\n")
	}
	b.WriteString("\n")
}

func writeSeoPackBody(b *strings.Builder, d SeoBrief) {
	if d.Rationale != "" {
		b.WriteString(d.Rationale + "\n\n")
	}
	if len(d.Flags) > 0 {
		b.WriteString("Punch list:\n\n")
		for _, f := range d.Flags {
			b.WriteString("- " + f + "\n")
		}
		b.WriteString("\n")
	}
	if d.Body != "" {
		b.WriteString(d.Body + "\n\n")
	}
}

func writeMarkdownFile(w http.ResponseWriter, name, body string, download bool) {
	w.Header().Set("Content-Type", "text/markdown; charset=utf-8")
	w.Header().Set("X-Robots-Tag", "noindex")
	w.Header().Set("Cache-Control", "no-store")
	disp := "inline"
	if download {
		disp = "attachment"
	}
	w.Header().Set("Content-Disposition", disp+`; filename="`+name+`"`)
	_, _ = w.Write([]byte(body))
}

func (s *Server) seoPackDownload(w http.ResponseWriter, r *http.Request) {
	u := s.requireDesk(w, r, "seo")
	if u == nil {
		return
	}
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	d, err := s.store.GetSeoBrief(r.Context(), id)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	p, _ := s.store.GetProject(r.Context(), d.ProjectID)
	sibs, _ := s.store.ListSeoBriefs(r.Context(), d.ProjectID)
	writeMarkdownFile(w, d.PackFilename(), seoPackMarkdown(d, p, sibs), true)
}

func (s *Server) seoPackPublic(w http.ResponseWriter, r *http.Request) {
	token := strings.TrimSpace(r.PathValue("token"))
	if token == "" {
		http.NotFound(w, r)
		return
	}
	d, err := s.store.GetSeoBriefByPackToken(r.Context(), token)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	p, _ := s.store.GetProject(r.Context(), d.ProjectID)
	sibs, _ := s.store.ListSeoBriefs(r.Context(), d.ProjectID)
	writeMarkdownFile(w, d.PackFilename(), seoPackMarkdown(d, p, sibs), false)
}
