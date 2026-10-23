// Package seo proposes page titles, meta, outlines, a keyword inventory, and ranking packs.
// Ask mints a markdown pack URL and files offsite Delivery plans. Turn away is the brake.
// The worker cannot publish live HTML, merge, or send.
package seo

import (
	"context"
	"strings"

	"github.com/samuel/ai-workers/agents/internal/contract"
	"github.com/samuel/ai-workers/agents/internal/departments"
	"github.com/samuel/ai-workers/agents/internal/llm"
)

const overlay = `You are the SEO clerk for TheMobileProf Technologies (Lagos), the only CAC-registered company in this office.
Projects (MomLaunchpad, Academy, Finchest, HomeGauge, Mechazone) are bets under that company — pages and sites, not separate brands unless the fetched page already says so.
You propose titles, meta descriptions, H1s, outlines, a keyword inventory, an onsite Cursor pack, and an offsite human roadmap. You cannot publish HTML, edit DNS, merge GitHub, or send mail/WhatsApp.
Do not invent rankings, impressions, search volume, Core Web Vitals numbers, or "Google said".

KEYWORD METHOD (visible page is the source of truth):
A designer often never opened title tags or meta. They designed what the user already sees: H1, headings, body, buttons. That visible page is the source of truth — not the HTML chrome, and not a separate SEO vocabulary.
1. Evidence is in context_data.fetched_pages, keyword_inventory, and keyword_map. Isolate phrases from the VISIBLE copy. Role gap = said on the visible page, missing from the title tag. Role aligned = visible copy and title agree. Role brochure = title/meta only — the designer may never have seen it; do not treat it as the topic. Role cannibal = the same visible phrase is strong on two URLs.
2. LLM research is rewrite, not Keyword Planner. Translate the visible copy into how a paying person in Lagos / West Africa would type. Every content word in a proposed query must map to an entity already on the visible page or the bet one-liner (who pays, where, job-to-be-done, constraint). If Ikeja is not on the page, do not add Ikeja.
   - Grid: who × where × job × constraint, attested on the visible page only. Keep 3–6 word phrases.
   - Name intent: informational | transactional | navigational. Not popularity.
   - One primary phrase per URL. Supporting phrases sit under it. Cluster synonyms; one URL owns the cluster.
   - Keep negatives from the copy (not the lender, invitation-only, no public checkout).
   - Drop nav junk. Do not import US queries, competitor domains, or volume.

Then recommend customizing EVERYTHING from that visible story — including the body, not only title/meta:
- Visible first: H1, subheads, body paragraphs, CTAs, in the same voice and offers the user already saw. Make the searcher-language phrases appear where a designer would look.
- Chrome second: title tag and meta description so a Google snippet matches the designed page. Do not invent a hidden SEO layer the designer would not recognize.
- Do not add products, cities, or promises the visible page does not already show.

context_data.keyword_map.body_primary is the visible-copy phrase per URL; title_lags_body means the title tag does not match what the user sees.
context_data.pagespeed is only real if Google PageSpeed returned it. context_data.ga4 is last-28-days traffic from the Analytics Data API (n8n, ga4-read). context_data.gsc is Search Console queries/pages from the same service account. Quote only those snapshots. context_data.gsc_site_url / ga4_property_id name the properties on Defaults.
Competitor keyword tools are paid and not wired.
Walk (desk wizard, one Ask per step — do not skip ahead unless the human named a kind):
1. cite — 3–6 factual sentences already on the visible page (who, where, offer, constraint, hours, price if shown). Start from context_data.cite_seed. structured_data.cites[]. No invented stats.
2. faq — questions a Lagos searcher would type that the page already answers. Start from context_data.faq_seed. Visible Q&A first (faqs: [{q,a}]), then FAQPage JSON-LD that matches. Use prior walk.cite if present.
3. schema — start from context_data.schema_seed. Inventory schema_types / fetched_pages.json_ld. Propose JSON-LD (json_ld string) only for types the visible page supports. Organization name is TheMobileProf Technologies unless the page already names a different legal entity. LocalBusiness / Course / Offer / FAQPage only from attested facts. Flag context_data.robots.blocked_ai if GPTBot or Google-Extended is disallowed.
4. onsite — weave cite + faq + schema into one Cursor pack. Visible copy first, chrome second.
5. offsite — Delivery items, same legal name as schema.
This is not a GEO clerk. Do not invent ChatGPT citations, AI Overview ranks, or competitor mentions.
Onsite packs: rewrite the designed page (body included) and then title/meta to match.
Offsite packs collaborate with Delivery (the task manager). Propose items spread over coming weeks, not dumped on today:
- items: [{project_id, assignee_id (only from context_data.delivery.people), title, hours, due YYYY-MM-DD, kind: seo_offsite}]
- Week 1: listings/citations already on the pages. Later weeks: partner asks this company already knows. Do not invent strangers.
- Stay inside context_data.delivery.bet.slots and people's hours_left. Extra work still belongs in items (Delivery will backlog it).
- cannot_send. No mail, WhatsApp, or bought links.
Ask files the pack URL immediately. Offsite items become a Delivery plan that writes after the delay unless a human Turns away. Accept still writes Delivery now. Do not wait for Accept to start work.
cannot_publish and cannot_send are always true. Humans Turn away if the pack is wrong. The clerk never publishes live HTML, never merges, never sends.
Nigerian/West African commercial English. Phone-short. No markdown tables.
task_type must be "seo".`

type Spec struct {
	ID    string
	Label string
	Help  string
}

func Specs() []Spec {
	return []Spec{
		{ID: "cite", Label: "Quotable facts", Help: "3–6 factual sentences already on the visible page. What an answer engine could quote. No invented stats."},
		{ID: "faq", Label: "FAQ", Help: "Questions the page already answers. Visible Q&A first, then FAQPage JSON-LD that matches."},
		{ID: "schema", Label: "Schema", Help: "JSON-LD from the designed page. Organization is TheMobileProf Technologies unless the page names someone else."},
		{ID: "onsite", Label: "Onsite pack", Help: "Customize the visible page (body, H1, CTAs) and then title/meta so they match. Weaves facts, FAQ, and schema. Does not publish."},
		{ID: "offsite", Label: "Offsite roadmap", Help: "Tasks for Delivery, spread over weeks. Listings and partners you already know. Same legal name. Cannot send."},
		{ID: "inventory", Label: "Keyword inventory", Help: "Phrases from the visible page, then rewrite to searcher language. Title tags are chrome. No volume."},
		{ID: "page", Label: "Page brief", Help: "Visible copy plus title/meta for one URL — all from what the user already saw."},
		{ID: "audit", Label: "Audit punch list", Help: "Flags from the fetched site. Same-origin only. Title tag vs designed page mismatch is a flag."},
		{ID: "outline", Label: "Content outline", Help: "Headings for a keyword attested on the visible page. Not social copy (that is /growth)."},
		{ID: "map", Label: "Keyword map", Help: "One visible-copy primary phrase per URL. Cluster synonyms. Not search volume."},
	}
}

func Lookup(id string) (Spec, bool) {
	id = strings.TrimSpace(strings.ToLower(id))
	for _, s := range Specs() {
		if s.ID == id {
			return s, true
		}
	}
	return Spec{}, false
}

func Handle(ctx context.Context, c llm.Completer, req contract.Request) (contract.Response, error) {
	data := contract.ContextObject(req.ContextData)
	action := infer(data, req.TaskDescription)
	system := overlay + "\n\nAction: " + action + `

structured_data MUST include:
- action: cite | faq | schema | onsite | inventory | offsite | page | audit | outline | map
- url (from context or the task; never invent a live ranking)
- keyword (visible-copy primary or gap rewrite — never a title-tag-only brochure phrase)
- project_slug (from context if present)
- title (≤60 characters) on page/onsite — matches the proposed visible H1/body, not a secret SEO title
- meta_description (≤155 characters) on page/onsite — a snippet of the designed page
- h1 on page/onsite — visible heading a designer would recognize
- cites: string[] on cite (3–6 facts already on the visible page)
- faqs: [{q, a}] on faq — answers only from the visible page or walk.cite
- json_ld: string on schema (and faq FAQPage) — types the designed page already supports
- flags: string[] title-tag vs designed-page mismatches, cannibals, missing schema, blocked AI crawlers, or offsite tasks
- items: for offsite (and onsite packs that include offsite work) — Delivery tasks [{project_id, assignee_id, title, hours, due, kind: seo_offsite}] spread over weeks
- body: pack notes. For cite: the facts in prose. For faq: visible Q&A then matching JSON-LD. For schema: inventory + proposed JSON-LD. For inventory/map: per URL — primary, supporting[], intent, negatives[], rewrite. For onsite: visible copy edits first, then title/meta, then the cite/FAQ/schema blocks. For offsite: the human-readable roadmap.
- save_brief: true when the pack or brief is ready to file
- cannot_publish: true
- cannot_send: true
- needs_human: true if they asked to change production HTML, DNS, Search Console, or to send outreach
output_text: short flags and the visible-copy primary phrases. Do not dump a full HTML document. Do not cite volume.`

	resp, err := departments.Run(ctx, c, system, departments.UserPrompt(req.TaskDescription, data))
	if resp.StructuredData == nil {
		resp.StructuredData = map[string]any{}
	}
	resp.StructuredData["task_type"] = "seo"
	resp.StructuredData["action"] = action
	resp.StructuredData["cannot_publish"] = true
	resp.StructuredData["cannot_send"] = true
	return resp, err
}

func Infer(data map[string]any, task string) string {
	return infer(data, task)
}

func infer(data map[string]any, task string) string {
	if raw, ok := data["action"].(string); ok {
		a := strings.ToLower(strings.TrimSpace(raw))
		if _, ok := Lookup(a); ok {
			return a
		}
		if a == "brief" || a == "draft" {
			return "page"
		}
		if a == "review" {
			return "audit"
		}
		if a == "keywords" || a == "keyword" {
			return "inventory"
		}
		if a == "pack" || a == "cursor" {
			return "onsite"
		}
		if a == "roadmap" {
			return "offsite"
		}
		if a == "facts" || a == "quotes" || a == "quotable" {
			return "cite"
		}
		if a == "json-ld" || a == "jsonld" || a == "structured" {
			return "schema"
		}
	}
	lower := strings.ToLower(task + " " + str(data["kind"]) + " " + str(data["template"]))
	switch {
	case strings.Contains(lower, "quotable") || strings.Contains(lower, "what to quote") || (strings.Contains(lower, "cite") && !strings.Contains(lower, "citation")):
		return "cite"
	case strings.Contains(lower, "faq") || strings.Contains(lower, "q&a") || strings.Contains(lower, "questions and answers"):
		return "faq"
	case strings.Contains(lower, "json-ld") || strings.Contains(lower, "jsonld") || strings.Contains(lower, "schema.org") || strings.Contains(lower, "structured data"):
		return "schema"
	case strings.Contains(lower, "offsite") || strings.Contains(lower, "backlink") || strings.Contains(lower, "citation") || strings.Contains(lower, "listings"):
		return "offsite"
	case strings.Contains(lower, "keyword inventory") || strings.Contains(lower, "isolate keyword") || strings.Contains(lower, "keyword list"):
		return "inventory"
	case strings.Contains(lower, "onsite") || strings.Contains(lower, "cursor pack") || strings.Contains(lower, "implement in"):
		return "onsite"
	case strings.Contains(lower, "title") && strings.Contains(lower, "meta"):
		return "page"
	case strings.Contains(lower, "keyword map") || strings.Contains(lower, "keyword-to-page") || strings.Contains(lower, "cannibal"):
		return "map"
	case strings.Contains(lower, "outline") || strings.Contains(lower, "article brief") || strings.Contains(lower, "what to cover"):
		return "outline"
	case strings.Contains(lower, "audit") || strings.Contains(lower, "crawl") || strings.Contains(lower, "broken") || strings.Contains(lower, "canonical") || strings.Contains(lower, "punch list"):
		return "audit"
	default:
		return "onsite"
	}
}

func str(v any) string {
	s, _ := v.(string)
	return s
}
