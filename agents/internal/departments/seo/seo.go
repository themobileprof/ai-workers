// Package seo proposes page titles, meta, outlines, and a punch list.
// Filing is proposed only; humans Accept. The worker cannot publish or crawl a live host.
package seo

import (
	"context"
	"strings"

	"github.com/samuel/ai-workers/agents/internal/contract"
	"github.com/samuel/ai-workers/agents/internal/departments"
	"github.com/samuel/ai-workers/agents/internal/llm"
)

const overlay = `You are the SEO clerk for TheMobileProf Technologies (Lagos), the only CAC-registered company in this office.
Projects (MomLaunchpad, Academy, Finchest, HomeGauge, Mechazone) are bets under that company — pages and sites, not separate brands unless the pasted page already says so.
You propose titles, meta descriptions, H1s, outlines, and a punch list. You cannot publish HTML, edit DNS, crawl a live host, or read Google Search Console.
Do not invent rankings, impressions, Core Web Vitals numbers, or "Google said".
If they did not paste page text, work only from the URL path and the project one-liner, and say what you assumed.
cannot_publish and cannot_send are always true. Humans Accept on the company desk. Until then the brief is a proposal.
Nigerian/West African commercial English. Phone-short. No markdown tables.
task_type must be "seo".`

type Spec struct {
	ID    string
	Label string
	Help  string
}

func Specs() []Spec {
	return []Spec{
		{ID: "page", Label: "Page brief", Help: "Title, meta description, H1 for one URL. Does not publish."},
		{ID: "audit", Label: "Audit punch list", Help: "Flags from pasted HTML or headers. Does not crawl."},
		{ID: "outline", Label: "Content outline", Help: "Headings for a keyword. Not social copy (that is /growth)."},
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
- action: page | audit | outline
- url (from context or the task; never invent a live ranking)
- keyword
- project_slug (from context if present)
- title (≤60 characters) on page
- meta_description (≤155 characters) on page
- h1 on page
- flags: string[] short issues (audit) or missing bits
- body: outline headings or notes; empty on a thin review
- save_brief: true when title/meta/h1 or an outline is ready to file
- cannot_publish: true
- cannot_send: true
- needs_human: true if they asked to change production HTML, DNS, or Search Console
output_text: short flags and the proposed title/meta. Do not dump a full HTML document.`

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
	}
	lower := strings.ToLower(task + " " + str(data["kind"]) + " " + str(data["template"]))
	switch {
	case strings.Contains(lower, "outline") || strings.Contains(lower, "article brief") || strings.Contains(lower, "what to cover"):
		return "outline"
	case strings.Contains(lower, "audit") || strings.Contains(lower, "crawl") || strings.Contains(lower, "broken") || strings.Contains(lower, "canonical") || strings.Contains(lower, "punch list"):
		return "audit"
	default:
		return "page"
	}
}

func str(v any) string {
	s, _ := v.(string)
	return s
}
