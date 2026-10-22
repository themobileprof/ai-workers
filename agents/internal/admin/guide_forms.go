package admin

import (
	"strconv"
	"strings"

	"github.com/samuel/ai-workers/agents/internal/departments/legal"
	"github.com/samuel/ai-workers/agents/internal/departments/seo"
)

// formRecipe is one page form Help? can fill. Submit posts the same action as the HTML form.
type formRecipe struct {
	ID          string
	Title       string
	Button      string
	Action      string
	Href        string
	OwnerOnly   bool
	Desk        string
	Delivery    bool
	NeedProject bool
	Use         string // for the LLM: when to pick this form
	Fields      []formField
}

type formField struct {
	Name     string
	Label    string
	Prompt   string
	Hint     string
	Kind     string // text, textarea, date, number, select, project, person, desk, secret, check, multi
	Required bool
	Default  string
	PathOnly bool
	Multi    bool
	Options  []string // "value" or "value|label"
}

func formCatalog() []formRecipe {
	stages := []string{"idea|Idea", "testing|Testing", "selling|Selling", "fundable|Fundable", "parked|Parked", "killed|Killed"}
	kinds := []string{"training|Training", "field|Field week", "launch|Launch", "other|Other date"}
	seats := []string{"cofounder|cofounder (public face)", "assistant|assistant"}
	roles := []string{"cofounder", "assistant", "bdm", "owner", "viewer"}
	legalTpl := []string{""}
	for _, s := range legal.Specs() {
		legalTpl = append(legalTpl, s.ID+"|"+s.Label)
	}
	seoActs := []string{}
	for _, s := range seo.Specs() {
		seoActs = append(seoActs, s.ID+"|"+s.Label)
	}
	deskOpts := []string{}
	for _, d := range AllDesks() {
		deskOpts = append(deskOpts, d.Slug+"|"+d.Title)
	}
	return []formRecipe{
		{
			ID: "add_date", Title: "Dates · Save a date", Button: "Save date",
			Action: "/admin/projects/{project_id}/events", Href: "/admin/projects/{project_id}?tab=dates",
			NeedProject: true,
			Use:         "Save a campus training, field week, launch, or other dated work on a project. Same as Projects → Dates → Save date.",
			Fields: []formField{
				{Name: "project_id", Label: "Project", Prompt: "Which project? Name or slug.", Kind: "project", Required: true, PathOnly: true},
				{Name: "title", Label: "What", Prompt: "What is the date for?", Kind: "text", Required: true},
				{Name: "kind", Label: "Kind", Prompt: "Kind: training, field, launch, or other.", Kind: "select", Required: true, Options: kinds},
				{Name: "starts_on", Label: "Starts", Prompt: "Start date as YYYY-MM-DD.", Kind: "date", Required: true},
				{Name: "ends_on", Label: "Ends", Prompt: "End date as YYYY-MM-DD, or skip if it is one day.", Kind: "date"},
				{Name: "venue", Label: "Where", Prompt: "Where is it?", Kind: "text"},
				{Name: "audience", Label: "Who", Prompt: "Who is it for? Skip if none.", Kind: "text"},
				{Name: "outcome", Label: "Success", Prompt: "What does success look like? Skip if you will write it later.", Kind: "text"},
				{Name: "set_week", Label: "This week", Prompt: "Set as this week’s line? yes or no.", Kind: "check", Default: "on"},
			},
		},
		{
			ID: "add_task", Title: "Add a task", Button: "Add task",
			Action: "/admin/delivery/items", Href: "/admin/delivery",
			Delivery: true,
			Use:      "Add one Delivery task on a bet. Same as Delivery → Add a task, or a project Work tab.",
			Fields: []formField{
				{Name: "project_id", Label: "Project", Prompt: "Which project holds this task? Name or slug.", Kind: "project", Required: true},
				{Name: "title", Label: "Task", Prompt: "What is the task in one line?", Kind: "text", Required: true},
				{Name: "assignee_id", Label: "Who", Prompt: "Who does it? A People name, or skip to leave unassigned.", Kind: "person"},
				{Name: "hours", Label: "Hours", Prompt: "Hours. Skip for 1.", Kind: "number", Default: "1"},
				{Name: "due", Label: "Due", Prompt: "Due date as YYYY-MM-DD, or skip.", Kind: "date"},
			},
		},
		{
			ID: "add_project", Title: "Open a project", Button: "Open project",
			Action: "/admin/projects", Href: "/admin/projects/new",
			Use: "Open a new incubated project/bet. Same as Projects → new.",
			Fields: []formField{
				{Name: "name", Label: "Name", Prompt: "Working title.", Kind: "text", Required: true},
				{Name: "one_liner", Label: "One-liner", Prompt: "Who pays, for what?", Kind: "text", Required: true},
				{Name: "slug", Label: "Slug", Prompt: "Slug (optional). Leave blank to use the name.", Kind: "text"},
				{Name: "url", Label: "Public URL", Prompt: "Public URL if people already type one. Skip if none.", Kind: "text"},
				{Name: "stage", Label: "Stage", Prompt: "Stage: idea, testing, selling, fundable, parked, or killed. Skip for Idea.", Kind: "select", Default: "idea", Options: stages},
			},
		},
		{
			ID: "add_person", Title: "Add a person", Button: "Add",
			Action: "/admin/users", Href: "/admin/users/new", OwnerOnly: true,
			Use: "Add a person to the office roster. Owner only. Same as People → Add.",
			Fields: []formField{
				{Name: "name", Label: "Name", Prompt: "Their name.", Kind: "text", Required: true},
				{Name: "phone", Label: "Phone", Prompt: "Phone in 234… form, or skip.", Kind: "text"},
				{Name: "telegram_id", Label: "Telegram id", Prompt: "Their Telegram id from ops bot /start, or skip.", Kind: "text"},
				{Name: "email", Label: "Email", Prompt: "Email, or skip.", Kind: "text"},
				{Name: "role", Label: "Role", Prompt: "Office seat: cofounder, assistant, bdm, or owner. Skip for cofounder.", Kind: "select", Default: "cofounder", Options: roles},
				{Name: "hours_per_week", Label: "Hours / week", Prompt: "Hours per week Delivery may give them. Skip for 6.", Kind: "number", Default: "6"},
				{Name: "password", Label: "Password", Prompt: "A login they will remember — skip if they only work in the field with no desk tick.", Kind: "secret"},
				{Name: "desk", Label: "Desks", Prompt: "Desks to tick (comma list: legal, seo, delivery…). Skip if they have no login.", Kind: "multi", Multi: true, Options: deskOpts},
			},
		},
		{
			ID: "seat_face", Title: "Seat someone on Face", Button: "Seat them",
			Action: "/admin/projects/{project_id}/members", Href: "/admin/projects/{project_id}?tab=face",
			NeedProject: true,
			Use:         "Seat someone already on People onto a project Face tab.",
			Fields: []formField{
				{Name: "project_id", Label: "Project", Prompt: "Which project? Name or slug.", Kind: "project", Required: true, PathOnly: true},
				{Name: "user_id", Label: "Person", Prompt: "Who? A name already on People.", Kind: "person", Required: true},
				{Name: "seat", Label: "Seat", Prompt: "Seat: cofounder (public face) or assistant.", Kind: "select", Required: true, Default: "assistant", Options: seats},
			},
		},
		{
			ID: "ask_placement", Title: "Ask placement", Button: "Ask placement",
			Action: "/admin/projects/{project_id}/place", Href: "/admin/projects/{project_id}?tab=stamp",
			NeedProject: true,
			Use:         "Ask the placement worker to propose a journey/gate. Does not Accept.",
			Fields: []formField{
				{Name: "project_id", Label: "Project", Prompt: "Which project should placement look at?", Kind: "project", Required: true, PathOnly: true},
				{Name: "human_feedback", Label: "Feedback", Prompt: "Any new evidence for the worker? Skip if none.", Kind: "text"},
			},
		},
		{
			ID: "ask_legal", Title: "Ask a legal draft", Button: "Ask draft",
			Action: "/admin/projects/{project_id}/legal", Href: "/admin/projects/{project_id}?tab=legal",
			Desk: "legal", NeedProject: true,
			Use: "Ask Legal for a proposed NDA or contract. Does not Accept or send.",
			Fields: []formField{
				{Name: "project_id", Label: "Project", Prompt: "Which project is this paper for?", Kind: "project", Required: true, PathOnly: true},
				{Name: "action", Label: "Action", Prompt: "draft (from template) or review (pasted text). Skip for draft.", Kind: "select", Default: "draft", Options: []string{"draft|Draft from template", "review|Review pasted text"}},
				{Name: "template", Label: "Template", Prompt: "Template: nda, contractor, ip_assignment, project_face, customer_terms — or skip to infer.", Kind: "select", Options: legalTpl},
				{Name: "counterparty_user_id", Label: "Other party on People", Prompt: "Other party if they are on People. Name, or skip.", Kind: "person"},
				{Name: "counterparty_name", Label: "Other party name", Prompt: "Name if they are not on People. Skip if you already picked someone.", Kind: "text"},
				{Name: "human_feedback", Label: "Notes", Prompt: "Notes, or paste a contract to review.", Kind: "textarea"},
			},
		},
		{
			ID: "ask_seo", Title: "Ask SEO", Button: "Ask SEO",
			Action: "/admin/projects/{project_id}/seo", Href: "/admin/projects/{project_id}?tab=seo",
			Desk: "seo", NeedProject: true,
			Use: "Ask SEO for a proposed page brief, audit, or outline. Does not publish.",
			Fields: []formField{
				{Name: "project_id", Label: "Project", Prompt: "Which project’s page?", Kind: "project", Required: true, PathOnly: true},
				{Name: "action", Label: "Kind", Prompt: "Kind: page, audit, or outline. Skip for page.", Kind: "select", Default: "page", Options: seoActs},
				{Name: "url", Label: "URL", Prompt: "Page URL if you have it. Skip if none.", Kind: "text"},
				{Name: "keyword", Label: "Keyword", Prompt: "Keyword. Skip if none.", Kind: "text"},
				{Name: "human_feedback", Label: "Notes", Prompt: "Notes or a slice of the page HTML. Skip if none.", Kind: "textarea"},
			},
		},
		{
			ID: "ask_delivery", Title: "Ask a week plan", Button: "Ask worker",
			Action: "/admin/delivery/ask", Href: "/admin/delivery?tab=ask",
			Delivery: true,
			Use:      "Ask Delivery to propose a week plan. Writes after a delay unless you Turn away. Accept still works now.",
			Fields: []formField{
				{Name: "project", Label: "Project slug", Prompt: "Pin one project slug, or skip for every bet you may see.", Kind: "text"},
				{Name: "task", Label: "What should the week look like?", Prompt: "What should the week look like? One paragraph.", Kind: "textarea", Required: true},
			},
		},
		{
			ID: "ask_desk", Title: "Ask a worker", Button: "Ask worker",
			Action: "/admin/desks/{desk}/ask", Href: "/admin/desks/{desk}",
			Use: "Ask a handler worker (accounts, growth, legal…) to start a proposed job. Does not Accept.",
			Fields: []formField{
				{Name: "desk", Label: "Worker", Prompt: "Which worker? accounts, growth, community, crm, legal, seo, delivery, internal-ops, product-dev.", Kind: "desk", Required: true, PathOnly: true, Options: deskOpts},
				{Name: "task", Label: "Task", Prompt: "What should this worker draft or flag? One or two sentences.", Kind: "textarea", Required: true},
			},
		},
		{
			ID: "add_crm", Title: "Add a CRM contact", Button: "Save contact",
			Action: "/admin/crm", Href: "/admin/crm",
			Use: "Save a person or company on the CRM book. Does not send mail. Same as Office → CRM.",
			Fields: []formField{
				{Name: "name", Label: "Name", Prompt: "Their name.", Kind: "text", Required: true},
				{Name: "company", Label: "Company", Prompt: "Company, or skip.", Kind: "text"},
				{Name: "email", Label: "Email", Prompt: "Email, or skip. Never invent one.", Kind: "text"},
				{Name: "phone", Label: "Phone", Prompt: "Phone as 234…, or skip. Never invent one.", Kind: "text"},
				{Name: "project_id", Label: "Project", Prompt: "Which bet, or skip for the company.", Kind: "project"},
				{Name: "offer", Label: "Offer", Prompt: "What they want to buy, or already bought. One line, or skip. Never invent.", Kind: "text"},
				{Name: "stage", Label: "Stage", Prompt: "new, qualified, proposal, or nurture. Skip for new.", Kind: "select", Default: "new", Options: []string{"new", "qualified", "proposal", "nurture"}},
				{Name: "next_action", Label: "Next action", Prompt: "The next human step. One line, or skip.", Kind: "text"},
				{Name: "next_action_by", Label: "By", Prompt: "Due date as YYYY-MM-DD, or skip.", Kind: "date"},
				{Name: "notes", Label: "Story", Prompt: "Short rolling story. What they asked, last move.", Kind: "textarea"},
			},
		},
		{
			ID: "log_crm", Title: "Log a CRM touch", Button: "Log touch",
			Action: "/admin/crm/{contact_id}/log", Href: "/admin/crm/{contact_id}",
			Use: "Log a call, WhatsApp, or note on a person. Does not send. Same as Office → CRM → the person → Log.",
			Fields: []formField{
				{Name: "contact_id", Label: "Person", Prompt: "Which CRM contact id? Open the person on Office → CRM if you do not know.", Kind: "text", Required: true, PathOnly: true},
				{Name: "channel", Label: "Channel", Prompt: "call, whatsapp, email, or note.", Kind: "select", Default: "call", Options: []string{"call", "whatsapp", "email", "desk"}},
				{Name: "direction", Label: "Direction", Prompt: "they reached us (in), we reached them (out), or a note.", Kind: "select", Default: "in", Options: []string{"in", "out", "note"}},
				{Name: "body", Label: "What happened", Prompt: "What was said. This is a record, not a message to send.", Kind: "textarea", Required: true},
			},
		},
		{
			ID: "add_grant", Title: "Add a grant row", Button: "Save grant",
			Action: "/admin/grants", Href: "/admin/grants",
			Use: "Save a grant deadline and next action. Does not submit an application. Same as Office → Grants.",
			Fields: []formField{
				{Name: "title", Label: "Name", Prompt: "What is the grant or programme called?", Kind: "text", Required: true},
				{Name: "funder", Label: "Funder", Prompt: "Who funds it? Skip if the name is enough.", Kind: "text"},
				{Name: "deadline", Label: "Deadline", Prompt: "Deadline as YYYY-MM-DD Africa/Lagos, or skip.", Kind: "date"},
				{Name: "project_id", Label: "Project", Prompt: "Which bet, or skip for the company.", Kind: "project"},
				{Name: "fit", Label: "Fit", Prompt: "Why we fit, or what is missing.", Kind: "textarea"},
				{Name: "next_action", Label: "Next action", Prompt: "The next human step. One line.", Kind: "text", Required: true},
			},
		},
		{
			ID: "save_mission", Title: "Save hypothesis and mission", Button: "Save mission",
			Action: "/admin/projects/{project_id}/memory", Href: "/admin/projects/{project_id}?tab=card",
			NeedProject: true,
			Use:         "Save this week’s hypothesis and field mission on a project Card. Not a hire. Same as Projects → Card.",
			Fields: []formField{
				{Name: "project_id", Label: "Project", Prompt: "Which project?", Kind: "project", Required: true, PathOnly: true},
				{Name: "hypothesis", Label: "Hypothesis", Prompt: "What must be true this gate?", Kind: "text"},
				{Name: "mission", Label: "Mission", Prompt: "One field job. Named place, named people.", Kind: "text"},
				{Name: "mission_due", Label: "Due", Prompt: "Due date as YYYY-MM-DD, or skip.", Kind: "date"},
				{Name: "burn_ngn", Label: "Spend this month", Prompt: "Spend this month in NGN from Books. Skip if unknown. Do not compute VAT.", Kind: "text"},
			},
		},
		{
			ID: "save_mandate", Title: "Save a community mandate", Button: "Save mandate",
			Action: "/admin/community", Href: "/admin/community?tab=edit",
			Desk: "community",
			Use:  "Save a Community mandate for an Academy Telegram room.",
			Fields: []formField{
				{Name: "title", Label: "Title", Prompt: "Room title.", Kind: "text", Required: true},
				{Name: "brief", Label: "Mandate", Prompt: "What this room is for, when to speak, when to escalate.", Kind: "textarea", Required: true},
				{Name: "catalog", Label: "Catalog", Prompt: "Courses this room may name. If a course is missing here, the worker must ask you.", Kind: "textarea", Required: true},
				{Name: "slug", Label: "Slug", Prompt: "Slug. Skip to infer.", Kind: "text"},
				{Name: "site_url", Label: "Site", Prompt: "Site URL if this room has one.", Kind: "text"},
				{Name: "telegram_chat_id", Label: "Telegram chat id", Prompt: "Telegram chat id from an Academy Telegram execution. Skip if you will paste it later.", Kind: "text"},
				{Name: "whatsapp_group_id", Label: "WhatsApp group id", Prompt: "WhatsApp group id — usually skip (Cloud API cannot join ordinary groups).", Kind: "text"},
				{Name: "active", Label: "Live", Prompt: "On or paused? Skip for On.", Kind: "select", Default: "true", Options: []string{"true|On", "false|Paused"}},
			},
		},
	}
}

func lookupRecipe(id string) (formRecipe, bool) {
	for _, r := range formCatalog() {
		if r.ID == id {
			return r, true
		}
	}
	return formRecipe{}, false
}

func allowedCatalog(u User, page string) []formRecipe {
	var out []formRecipe
	for _, r := range formCatalog() {
		if recipeAllowed(r, u, page) == "" {
			out = append(out, r)
		}
	}
	return out
}

func catalogPrompt(recipes []formRecipe) string {
	var b strings.Builder
	b.WriteString("Fillable forms (recipe id must be one of these; never invent an id; never Accept):\n")
	for _, r := range recipes {
		b.WriteString("- id=")
		b.WriteString(r.ID)
		b.WriteString(" title=")
		b.WriteString(r.Title)
		b.WriteString(" use=")
		b.WriteString(r.Use)
		b.WriteString(" fields=")
		for i, f := range r.Fields {
			if i > 0 {
				b.WriteString("; ")
			}
			b.WriteString(f.Name)
			b.WriteString("(")
			b.WriteString(f.Kind)
			if f.Required {
				b.WriteString(", required")
			}
			if f.Default != "" {
				b.WriteString(", default ")
				b.WriteString(f.Default)
			}
			if len(f.Options) > 0 {
				b.WriteString(", ")
				var opts []string
				for _, o := range f.Options {
					if v := optionValue(o); v != "" {
						opts = append(opts, v)
					}
				}
				b.WriteString(strings.Join(opts, "|"))
			}
			b.WriteString(")")
		}
		b.WriteByte('\n')
	}
	return b.String()
}

func recipeAllowed(r formRecipe, u User, page string) string {
	if r.OwnerOnly && !isOwner(&u) {
		return "Only the owner adds people on People. Ask the owner."
	}
	if r.Delivery && !canOffice(&u) && !Handles(u, "delivery") {
		return "Delivery is not ticked on your People card. Ask the owner."
	}
	if r.Desk != "" && !Handles(u, r.Desk) {
		return "Tick " + r.Desk + " on your People card (Desks tab), sign out and in, then try again."
	}
	if r.ID == "ask_desk" {
		desk := strings.TrimSpace(parseDeskSlug(page))
		if desk != "" && !Handles(u, desk) && !(desk == "delivery" && canOffice(&u)) {
			return "That worker is not ticked on your People card."
		}
	}
	return ""
}

func optionValue(opt string) string {
	v, _, _ := strings.Cut(opt, "|")
	return strings.TrimSpace(v)
}

func optionLabel(opt string) string {
	v, label, ok := strings.Cut(opt, "|")
	if ok {
		return strings.TrimSpace(label)
	}
	return strings.TrimSpace(v)
}

func validOption(f formField, val string) bool {
	if len(f.Options) == 0 {
		return true
	}
	val = strings.TrimSpace(val)
	if val == "" {
		return !f.Required
	}
	if f.Multi {
		for _, part := range strings.Split(val, ",") {
			part = strings.TrimSpace(part)
			if part == "" {
				continue
			}
			if !validOption(formField{Options: f.Options}, part) {
				return false
			}
		}
		return true
	}
	for _, opt := range f.Options {
		if optionValue(opt) == val || strings.EqualFold(optionLabel(opt), val) {
			return true
		}
	}
	return false
}

func coerceOption(f formField, raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	if f.Kind == "check" {
		low := strings.ToLower(raw)
		if low == "yes" || low == "y" || low == "on" || low == "true" || low == "1" {
			return "on"
		}
		if low == "no" || low == "n" || low == "off" || low == "false" || low == "0" || low == "skip" {
			return ""
		}
		return raw
	}
	if len(f.Options) == 0 {
		return raw
	}
	if f.Multi {
		var parts []string
		for _, p := range strings.Split(raw, ",") {
			p = strings.TrimSpace(p)
			if p == "" {
				continue
			}
			got := coerceOption(formField{Options: f.Options}, p)
			if got != "" {
				parts = append(parts, got)
			}
		}
		return strings.Join(parts, ",")
	}
	low := strings.ToLower(raw)
	for _, opt := range f.Options {
		v := optionValue(opt)
		if strings.EqualFold(v, raw) || strings.ToLower(optionLabel(opt)) == low {
			return v
		}
		if strings.Contains(strings.ToLower(optionLabel(opt)), low) || strings.Contains(low, strings.ToLower(v)) {
			return v
		}
	}
	return raw
}

func seedDefaults(r formRecipe, values map[string]string) {
	if values == nil {
		return
	}
	for _, f := range r.Fields {
		if strings.TrimSpace(values[f.Name]) == "" && f.Default != "" {
			values[f.Name] = f.Default
		}
	}
}

func nextMissing(r formRecipe, values map[string]string) *formField {
	for i := range r.Fields {
		f := &r.Fields[i]
		if !f.Required {
			continue
		}
		if strings.TrimSpace(values[f.Name]) == "" {
			return f
		}
	}
	return nil
}

func safeFormAction(path string) string {
	path = strings.TrimSpace(path)
	if !strings.HasPrefix(path, "/admin/") || strings.Contains(path, "..") || strings.Contains(path, "{") {
		return ""
	}
	low := strings.ToLower(path)
	for _, bad := range []string{"/accept", "/reject", "/approve", "/delete", "/done", "/skip", "/delivered"} {
		if strings.Contains(low, bad) {
			return ""
		}
	}
	return path
}

func resolveAction(r formRecipe, values map[string]string) string {
	path := r.Action
	href := r.Href
	repl := func(s string) string {
		s = strings.ReplaceAll(s, "{project_id}", strings.TrimSpace(values["project_id"]))
		s = strings.ReplaceAll(s, "{desk}", strings.TrimSpace(values["desk"]))
		cid := strings.TrimSpace(values["contact_id"])
		if !digitsOnly(cid) {
			cid = ""
		}
		s = strings.ReplaceAll(s, "{contact_id}", cid)
		return s
	}
	path = repl(path)
	_ = href
	return safeFormAction(path)
}

func resolveHref(r formRecipe, values map[string]string) string {
	href := r.Href
	href = strings.ReplaceAll(href, "{project_id}", strings.TrimSpace(values["project_id"]))
	href = strings.ReplaceAll(href, "{desk}", strings.TrimSpace(values["desk"]))
	if strings.Contains(href, "{") {
		if strings.HasPrefix(r.Href, "/admin/") && !strings.Contains(r.Href, "{") {
			return r.Href
		}
		return "/admin/"
	}
	return href
}

func parseProjectID(page string) int64 {
	page = strings.TrimSpace(page)
	const p = "/admin/projects/"
	i := strings.Index(page, p)
	if i < 0 {
		return 0
	}
	rest := page[i+len(p):]
	id := strings.Builder{}
	for _, c := range rest {
		if c < '0' || c > '9' {
			break
		}
		id.WriteRune(c)
	}
	n, _ := strconv.ParseInt(id.String(), 10, 64)
	return n
}

func parseDeskSlug(page string) string {
	const p = "/admin/desks/"
	i := strings.Index(page, p)
	if i < 0 {
		return ""
	}
	rest := page[i+len(p):]
	rest = strings.Trim(rest, "/")
	if j := strings.IndexAny(rest, "/?#"); j >= 0 {
		rest = rest[:j]
	}
	if _, ok := LookupDesk(rest); ok {
		return rest
	}
	return ""
}
