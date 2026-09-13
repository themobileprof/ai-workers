package admin

import (
	"embed"
	"fmt"
	"html/template"
	"strings"
)

//go:embed playbook/*.html
var playbookFS embed.FS

type DocMeta struct {
	Slug     string
	Title    string
	Kicker   string
	Path     string
	Wire     string
	Summary  string
	Prefixes []string
}

type DocPage struct {
	DocMeta
	Body template.HTML
	Prev *DocMeta
	Next *DocMeta
}

func (d DocMeta) Live() bool    { return d.Wire == "live" }
func (d DocMeta) Partial() bool { return d.Wire == "partial" }
func (d DocMeta) Later() bool   { return d.Wire == "later" }
func (d DocMeta) Href() string  { return "/admin/docs/" + d.Slug }
func (d DocMeta) WireLabel() string {
	switch d.Wire {
	case "live":
		return "On the wire"
	case "partial":
		return "Prefix live · follow-up missing"
	default:
		return "Not wired"
	}
}

var docCatalog = []DocMeta{
	{
		Slug:     "contract",
		Title:    "How to call a worker",
		Kicker:   "Envelope",
		Path:     "POST /departments/{name}",
		Wire:     "live",
		Summary:  "Every department takes the same JSON. n8n is the only public caller.",
		Prefixes: nil,
	},
	{
		Slug:     "growth",
		Title:    "Growth",
		Kicker:   "Front office",
		Path:     "POST /departments/growth",
		Wire:     "live",
		Summary:  "Sales chat, inbound email drafts, and marketing copy. High intent upserts a Books contact.",
		Prefixes: []string{"/growth", "1:1 WhatsApp (default)", "info@ inbound"},
	},
	{
		Slug:     "community",
		Title:    "Community",
		Kicker:   "Room voice",
		Path:     "POST /departments/community",
		Wire:     "live",
		Summary:  "WhatsApp/Telegram groups, /cm, and the access-denied reply when a number is not on the books desk.",
		Prefixes: []string{"/cm", "/community", "group chat (default)"},
	},
	{
		Slug:     "crm",
		Title:    "CRM",
		Kicker:   "Pipeline",
		Path:     "POST /departments/crm",
		Wire:     "live",
		Summary:  "Qualify or update a person. n8n writes Zoho Books contacts, not Zoho CRM.",
		Prefixes: []string{"/crm"},
	},
	{
		Slug:     "accounts",
		Title:    "Accounts",
		Kicker:   "Books classifier",
		Path:     "POST /departments/accounts",
		Wire:     "live",
		Summary:  "Extract vendor, amount, and Zoho document type. n8n posts to Books. Go never computes VAT.",
		Prefixes: []string{"/accounts"},
	},
	{
		Slug:     "internal-ops",
		Title:    "Internal ops",
		Kicker:   "Operations room",
		Path:     "POST /departments/internal-ops",
		Wire:     "partial",
		Summary:  "/ops reaches the worker. Accounts via /ops writes Books. Legal and grant hunting still have no follow-up.",
		Prefixes: []string{"/ops"},
	},
	{
		Slug:     "product-dev",
		Title:    "Product-dev",
		Kicker:   "Engineering lab",
		Path:     "POST /departments/product-dev",
		Wire:     "partial",
		Summary:  "/validate reaches the worker. n8n does not yet keep project state, intern missions, or GitHub QA.",
		Prefixes: []string{"/validate"},
	},
}

func allDocs() []DocMeta {
	out := make([]DocMeta, len(docCatalog))
	copy(out, docCatalog)
	return out
}

func lookupDoc(slug string) (DocPage, bool) {
	slug = strings.TrimSpace(slug)
	for i, meta := range docCatalog {
		if meta.Slug != slug {
			continue
		}
		raw, err := playbookFS.ReadFile("playbook/" + slug + ".html")
		if err != nil {
			return DocPage{}, false
		}
		page := DocPage{DocMeta: meta, Body: template.HTML(raw)}
		if i > 0 {
			prev := docCatalog[i-1]
			page.Prev = &prev
		}
		if i+1 < len(docCatalog) {
			next := docCatalog[i+1]
			page.Next = &next
		}
		return page, true
	}
	return DocPage{}, false
}

func mustPlaybookFiles() error {
	for _, d := range docCatalog {
		name := "playbook/" + d.Slug + ".html"
		if _, err := playbookFS.ReadFile(name); err != nil {
			return fmt.Errorf("missing %s: %w", name, err)
		}
	}
	return nil
}
