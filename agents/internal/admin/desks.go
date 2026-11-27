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
		{Slug: "growth", Title: "Growth", Department: "growth", Prefix: "/growth", Help: "On customer WhatsApp it answers questions about every public product from the project Cards. It also drafts social copy, a pitch-deck session, and Google Search ads you Accept then Place. It cannot send or spend from chat. Inbound info@ is CRM.", BookPath: "/admin/growth", BookLabel: "Growth book"},
		{Slug: "community", Title: "Community", Department: "community", Prefix: "/cm", Help: "Voice for one Academy Telegram room at a time. Live catalog, join welcome, DM billing email, /invite landing, daily ₦500 airtime, and a morning check-in only after students talked. Campus trainings are project Dates, not rooms. Step in when the worker sets escalate_to_founder.", BookPath: "/admin/community", BookLabel: "Mandates"},
		{Slug: "crm", Title: "CRM", Department: "crm", Prefix: "/crm", Help: "Conversion book: a person, an offer, and a timeline of email, WhatsApp, and calls. Due done takes a dated row off the 08:30 ping. It cannot send.", BookPath: "/admin/crm", BookLabel: "CRM book"},
		{Slug: "legal", Title: "Legal", Department: "legal", Prefix: "/legal", Help: "Proposes NDAs and contracts. Watch flags labour/CAC/NDPR gotchas on ops Telegram after another worker answers; caution and stop also file a job here. Copy the text yourself — it does not email anyone.", BookPath: "/admin/legal", BookLabel: "Legal book"},
		{Slug: "delivery", Title: "Delivery", Department: "delivery", Prefix: "/today", Help: "Tasks across bets. Stage caps how many; extras sit on Backlog. Plans write after a delay unless you Turn away. Fathom notes use the same path. The 08:30 ping includes CRM due for owner/BDM. Chat never Accepts.", BookPath: "/admin/delivery", BookLabel: "Delivery"},
		{Slug: "internal-ops", Title: "Ops", Department: "internal-ops", Prefix: "/ops", Help: "Operations room. Unprefixed ops Telegram picks one desk and does the work in that chat. Accounts via /ops still classifies Books. Grant answers write a deadline row. They do not submit an application."},
		{Slug: "product-dev", Title: "Product", Department: "product-dev", Prefix: "/validate", Help: "Placement and validation. /validate proposes Card hypothesis, mission, and evidence; Accept writes the Card. Accept on Decide writes the stage. Chat never stamps.", BookPath: "/admin/projects", BookLabel: "Projects"},
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
	return nav == "settings" || nav == "files" || nav == "grants" || nav == "crm"
}

// Does is the flowing list of what this clerk actually does, shown on the worker page.
func (d Desk) Does() string {
	return workerDoes(d.Slug)
}

func workerDoes(slug string) string {
	return workerDoesText[slug]
}

var workerDoesText = map[string]string{
	"accounts": "Accounts names the Zoho Books document from a receipt, bill, or invoice you send on ops Telegram with /accounts. n8n posts it; this clerk does not compute VAT or withhold tax — Books already owns that. You confirm the audit trail on the same chat (expense, bill, invoice id, or a Paystack link). Hypotheticals must say “do not record”. Chart-of-accounts ids live on Defaults → Zoho, not in the message. Customer WhatsApp cannot book this.",

	"growth": "Growth is the front office. On customer WhatsApp it answers questions about every public product from the project Cards (Academy, HomeGauge, Mechazone, and the others) and files the person on CRM when there is a phone — it never books receipts or runs office prefixes there. Once they name a product, later messages stay on that product until they name another. Customers hear product, offer, or project — never bet. Anyone can DM; there is no number allowlist. On ops Telegram and this desk it drafts LinkedIn or X copy you post yourself. Ask a pitch deck on a project and it walks Guy Kawasaki’s ten slides from the Card; you paste into Slides and /file the PDF. Ask Search ads and it walks Google Search text from the Card: final URL, geo and keyword tests (not age or gender), headlines, descriptions, then Pexels search phrases. The clerk uses the same Growth LLM already on the VM (LLM keys in `.env`). You Accept, then either follow Book it yourself in the Google Ads or Meta UI, or Place on Google Search or Meta from the Growth book. Do not do both for the same channel. n8n holds Google Ads and Facebook Ads OAuth. Google sunset developer tokens on 9 September 2026; API access is the Cloud project that owns the Google Ads credential. Place reads remaining funds and stays conservative: platform-minimum daily, never more than remaining/90, exact match only, Google Search only (no partners). One experiment per geo/keyword cell. RSA rotates headlines. Place runs all Lagos hours for a week so weekday vs weekend and morning/evening can show. A 07:00 Lagos job then pauses a cell that spent with 0 clicks, and turns off dayparts that spent with 0 clicks while another daypart got clicks. Conversions wait two weeks, and only if tracking already fired. Place stamps UTM so GA4 last 7 days of paid sessions and the funnel land on this book. After the morning GA4 pull, a proposed glance job on this book quotes those numbers — it does not Place. Go drops geo and keywords that are not on the Card. Funnel events come from the project Card (TRAFFIC.md in the site repo), else Defaults → Ads. Account ids live on Defaults → Ads; a daily ceiling there is optional. Go never holds those tokens. Campaigns start paused unless you set ENABLED. The clerk cannot send mail or WhatsApp, cannot spend from chat, and cannot invent traction, cities, or CPC. Inbound info@ is CRM, not this worker.",

	"community": "Community is the voice of one Academy Telegram room at a time. It welcomes people who join, runs the weekly intro, answers progress and lesson questions from the live LMS catalog, helps with workshop steps, and stays quiet on chatter. Students DM this bot the email on their LMS account for billing — never in the group. /invite is their landing page. At 16:00 Lagos it sends ₦500 airtime even when nobody earned it, and names yesterday’s winner when there is one. At 09:30 it checks in only when a student talked, asked a lesson question, or confirmed an invite yesterday — so an empty room is not two bot posts a day. Campus trainings belong on project Dates, not here. It does not do books, legal, or HR. Step in when it escalates to a founder.",

	"crm": "CRM is the conversion book: a person, an offer, a stage, and a timeline of email, WhatsApp, and calls. Customer WhatsApp, info@, /crm, and a logged call file a touch on that person. /crm follow up proposes the next human step from the story. Due done takes them off the 08:30 ping without changing stage. Won stays for a later upsell; lost closes; nurture waits. It never invents an email or phone and never sends. n8n upserts a Zoho Books contact when there is an email, or once the stage is qualified, proposal, or won. There is no Zoho CRM app.",

	"legal": "Legal drafts NDAs and contracts from a template on a bet. Ask draft on the project, or /legal on ops Telegram. After another desk answers on ops Telegram, it watches for labour, CAC, NDPR, payments, tax, or contract gotchas — a caution or stop also files a job here. You Accept, then copy the text yourself. It cannot email anyone, cannot mark approved, and must not invent case citations. Customer WhatsApp /legal is refused. /approve is still only the pending info@ draft.",

	"delivery": "Delivery is the week board across every bet. You add a task, Ask a plan, or accept Fathom notes after a recording. The project’s stage caps how many tasks may be open; extras sit on Backlog. Plans write after a delay unless you Turn away. The 08:30 Lagos ping lists today’s work and CRM due for owner and BDM. Reply in ops Telegram with /today, or to that card in plain text. Seat people on Face; hours/week on People is pace. Chat never Accepts. Customer WhatsApp cannot run Delivery.",

	"internal-ops": "Ops is the operations room. /ops plus the task. If it is a receipt, bill, or invoice, it uses the same Books path as Accounts. Legal via /ops is chat-only — file an NDA on Legal. Grant answers write a deadline and next action on Grants; they do not submit an application. Unprefixed ops Telegram already picks one desk and does the work, so you often do not need this prefix. Customer WhatsApp cannot run /ops.",

	"product-dev": "Product proposes Card work on a bet: hypotheses, this week’s mission, interview guides, evidence, and journey placement. /validate on ops Telegram, or Ask on this desk. Accept on Product writes the Card. Accept on the project Decide tab writes the stage. Chat never stamps GO, PIVOT, or KILL. Customer WhatsApp is not this worker.",

	"files": "Files is the data-room clerk. On ops Telegram you send /file or drop a PDF. A catalog match files itself and n8n writes Google Drive. Odd paths wait here for Accept. It cannot upload, share, or invent a folder. Office papers use the slug themobileprof, not a bet. Academy Telegram and customer WhatsApp do not file. Receipt photos without /file still go to Accounts.",
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
