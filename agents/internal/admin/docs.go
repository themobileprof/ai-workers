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
	Kind     string // "player" handout, or empty/"worker" for desk specimens
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
func (d DocMeta) Player() bool  { return d.Kind == "player" }
func (d DocMeta) Worker() bool  { return !d.Player() }
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
		Slug:     "you",
		Title:    "How you fit",
		Kicker:   "Start here",
		Kind:     "player",
		Path:     "Pick the sheet that matches your People card",
		Wire:     "live",
		Summary:  "One company. You are a seat, not a department. Print this map, then the page for your role.",
		Prefixes: []string{"Owner", "BDM", "Project face", "Field", "Handler"},
	},
	{
		Slug:     "owner",
		Title:    "Owner",
		Kicker:   "Office",
		Kind:     "player",
		Path:     "Board · People · Projects",
		Wire:     "live",
		Summary:  "You run the Lagos office. Workers propose. You click Accept. You add people and seat them on bets.",
		Prefixes: []string{"People", "Ops Telegram", "/approve is email only"},
	},
	{
		Slug:     "bdm",
		Title:    "Business desk",
		Kicker:   "Office",
		Kind:     "player",
		Path:     "Board · Projects",
		Wire:     "live",
		Summary:  "Same Board as the owner. You Accept on bets. You do not add people — ask the owner.",
		Prefixes: []string{"Board", "Ask placement", "this-week note"},
	},
	{
		Slug:     "project-face",
		Title:    "Project face",
		Kicker:   "On one bet",
		Kind:     "player",
		Path:     "Seat on a project card · cofounder",
		Wire:     "live",
		Summary:  "You are the public face of one bet. You are not a director of TheMobileProf. You do not see the whole office.",
		Prefixes: []string{"Your project only", "No Board"},
	},
	{
		Slug:     "field",
		Title:    "Field week",
		Kicker:   "On one bet",
		Kind:     "player",
		Path:     "Seat on a project card · assistant",
		Wire:     "live",
		Summary:  "You go to the place and come back with names. The card’s this-week line is your mission. No invented company.",
		Prefixes: []string{"Named people", "This-week note"},
	},
	{
		Slug:     "handler",
		Title:    "Running a worker",
		Kicker:   "Desk",
		Kind:     "player",
		Path:     "Workers menu · Ask → Accept",
		Wire:     "live",
		Summary:  "You were ticked on a People card. Ask, chat, Accept. Chat never Accepts. JSON specimens live on the worker pages.",
		Prefixes: []string{"Ask", "Accept", "Turn away"},
	},
	{
		Slug:     "contract",
		Title:    "How to call a worker",
		Kicker:   "Envelope",
		Path:     "POST /departments/{name}",
		Wire:     "live",
		Summary:  "Every department takes the same JSON. n8n is the only caller. Header X-Internal-Token.",
		Prefixes: nil,
	},
	{
		Slug:     "desk",
		Title:    "Company desk",
		Kicker:   "Humans",
		Path:     "/admin/",
		Wire:     "live",
		Summary:  "Board for owner/BDM. Handler desks appear when that person signs in. Chat does not Accept.",
		Prefixes: []string{"People", "Workers menu", "Ask → Accept"},
	},
	{
		Slug:     "projects",
		Title:    "Projects",
		Kicker:   "Incubation",
		Path:     "/admin/projects",
		Wire:     "live",
		Summary:  "Bets under one company. Run walks the same base kit for every bet. Field humans bring named next steps. Dates hold a campus training until you deliver. The worker proposes a gate; you Accept on Decide.",
		Prefixes: []string{"Run", "Ask placement", "/validate", "Dates", "this-week note"},
	},
	{
		Slug:     "growth",
		Title:    "Growth",
		Kicker:   "Front office",
		Path:     "POST /departments/growth",
		Wire:     "live",
		Summary:  "Sales chat and marketing copy. Every customer WhatsApp DM files Office → CRM. Books upserts only with an email or once qualified+. Inbound info@ is CRM, not an auto-reply.",
		Prefixes: []string{"/growth", "1:1 WhatsApp (default)"},
	},
	{
		Slug:     "community",
		Title:    "Community",
		Kicker:   "Room voice",
		Path:     "POST /departments/community",
		Wire:     "live",
		Summary:  "Mandated Academy Telegram rooms, Monday intro, /cm, LMS catalog. WhatsApp Cloud API cannot join ordinary groups.",
		Prefixes: []string{"/cm", "/community", "Academy Telegram group"},
	},
	{
		Slug:     "crm",
		Title:    "CRM",
		Kicker:   "Pipeline",
		Path:     "POST /departments/crm · /admin/crm",
		Wire:     "live",
		Summary:  "The conversion book. WhatsApp, info@, calls, and /crm file a timeline on the person. Offer and stage are the pitch. /crm follow up proposes the next step. The 08:30 Telegram card lists due dates to owner/BDM. Won stays for a later upsell. It does not send.",
		Prefixes: []string{"/crm", "info@ inbound", "Office → CRM"},
	},
	{
		Slug:     "accounts",
		Title:    "Accounts",
		Kicker:   "Books classifier",
		Path:     "POST /departments/accounts",
		Wire:     "live",
		Summary:  "Extract vendor, amount, email, and Zoho document type from text or a receipt photo (Gemini OCR). n8n posts to Books and Paystack. Go never computes VAT.",
		Prefixes: []string{"/accounts"},
	},
	{
		Slug:     "legal",
		Title:    "Legal",
		Kicker:   "Parent clerk",
		Path:     "POST /departments/legal",
		Wire:     "live",
		Summary:  "Telegram watch flags labour/CAC/NDPR/CBN gotchas without a prefix. Caution and stop also file a Legal job. /legal and desk Ask draft file proposed NDAs. Humans Accept. The worker cannot send.",
		Prefixes: []string{"/legal", "Telegram watch", "desk Ask draft"},
	},
	{
		Slug:     "files",
		Title:    "Files",
		Kicker:   "Data room",
		Path:     "POST /departments/files",
		Wire:     "live",
		Summary:  "Ops Telegram /file or a dropped PDF. A catalog match files itself; n8n writes Drive. Unsure rows stay proposed for Accept. The worker cannot invent a folder.",
		Prefixes: []string{"/file", "dropped PDF", "desk Files"},
	},
	{
		Slug:     "seo",
		Title:    "SEO",
		Kicker:   "Parent clerk",
		Path:     "POST /departments/seo",
		Wire:     "live",
		Summary:  "Fetches the bet’s own URL. Walk is quotable facts, FAQ, schema, onsite pack, then offsite. Ask issues a markdown pack for Cursor. Offsite work is a Delivery week plan that writes after the delay unless you Turn away. GA4 and Search Console last-28-days arrive from n8n. Not a GEO clerk.",
		Prefixes: []string{"/seo", "desk Ask"},
	},
	{
		Slug:     "delivery",
		Title:    "Delivery",
		Kicker:   "Project PM",
		Path:     "POST /departments/delivery",
		Wire:     "live",
		Summary:  "Tasks across bets. Stage sets how many; extras sit on the backlog. Pace is hours/week on People. Plans write after a delay unless you Turn away. Fathom notes use the same path. Academy is one bet.",
		Prefixes: []string{"/today", "/pm", "/idea", "Fathom webhook"},
	},
	{
		Slug:     "internal-ops",
		Title:    "Internal ops",
		Kicker:   "Operations room",
		Path:     "POST /departments/internal-ops",
		Wire:     "partial",
		Summary:  "/ops reaches the worker. Accounts via /ops writes Books. /ops legal stays chat-only; filing is /legal. Grant hunting writes a deadline row on Grants. It does not submit an application.",
		Prefixes: []string{"/ops"},
	},
	{
		Slug:     "product-dev",
		Title:    "Product-dev",
		Kicker:   "Engineering lab",
		Path:     "POST /departments/product-dev",
		Wire:     "live",
		Summary:  "/validate proposes Card hypothesis, mission, and evidence. Accept on Product writes the Card. Placement still Accepts on Decide. Chat never stamps GO / PIVOT / KILL. WhatsApp intern nag is later.",
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
