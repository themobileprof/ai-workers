package legal

import (
	"context"
	"embed"
	"strings"

	"github.com/samuel/ai-workers/agents/internal/contract"
	"github.com/samuel/ai-workers/agents/internal/departments"
	"github.com/samuel/ai-workers/agents/internal/llm"
)

//go:embed templates/*.md
var templateFS embed.FS

const overlay = `You are the Legal clerk for TheMobileProf Technologies (Lagos), the only CAC-registered company in this office.
Projects (MomLaunchpad, Academy, Finchest, HomeGauge, Mechazone) are bets under that company, not separate companies.
A project "cofounder" is the public face of a bet. They are not automatically a director of TheMobileProf Technologies.
You draft and flag. You are not a lawyer and you do not file at CAC, FIRS, or NITDA.
Do not invent case citations, section numbers, or "the Companies Act says".
Do not email or stamp a document. structured_data.cannot_send is always true.
Humans Accept on the company desk. Until then the draft is a proposal.
Nigerian/West African commercial English. Plain sentences.
task_type must be "legal".
structured_data.prompt_version is "v1".`

type Spec struct {
	ID    string
	Label string
	File  string
	Help  string
}

func Specs() []Spec {
	return []Spec{
		{ID: "nda", Label: "NDA", File: "nda.md", Help: "Confidentiality both ways. No licence to project IP."},
		{ID: "contractor", Label: "Contractor", File: "contractor.md", Help: "Independent contractor for TMP. Not employment, not PAYE."},
		{ID: "ip_assignment", Label: "IP assignment", File: "ip_assignment.md", Help: "Work product assigns to TheMobileProf Technologies."},
		{ID: "project_face", Label: "Project face", File: "project_face.md", Help: "Public face of a named project. Customers and marks stay with TMP."},
		{ID: "customer_terms", Label: "Customer / pilot terms", File: "customer_terms.md", Help: "Pilot invoiced as TMP. Paystack. Not a second company."},
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
	action, specID := infer(data, req.TaskDescription)
	if action == "watch" {
		return handleWatch(ctx, c, req, data)
	}
	system := overlay + "\n\nAction: " + action
	if spec, ok := Lookup(specID); ok {
		body, err := templateFS.ReadFile("templates/" + spec.File)
		if err != nil {
			return contract.Fail("missing legal template: " + spec.ID), err
		}
		system += "\nTemplate id: " + spec.ID + " (" + spec.Label + ")\n\nUse this skeleton. Fill names, project, and dates from context. Leave blanks as [TO BE COMPLETED] rather than inventing.\n\n" + string(body)
	} else if action == "draft" {
		system += "\nNo catalog template matched. Draft a short TMP-owned note and set template to none. Still cannot send."
	} else {
		system += "\nReview the pasted text. List predatory or missing clauses. Set template to none unless a catalog id clearly fits."
	}
	system += `

structured_data MUST include:
- action: review | draft
- template: nda | contractor | ip_assignment | project_face | customer_terms | none
- counterparty_name (from context; never invent an email)
- project_slug (from context if present)
- flags: string[] (short risks or missing clauses)
- needs_human_lawyer: boolean (true if employment, equity, regulated finance, or you are unsure)
- save_draft: boolean (true only when body is a full draft, not a review)
- cannot_send: true
- body: the draft text when save_draft is true; omit or empty on review
output_text: short flags and next human step. Do not paste the full contract into output_text.`

	resp, err := departments.Run(ctx, c, system, departments.UserPrompt(req.TaskDescription, data))
	if resp.StructuredData == nil {
		resp.StructuredData = map[string]any{}
	}
	resp.StructuredData["task_type"] = "legal"
	resp.StructuredData["action"] = action
	resp.StructuredData["cannot_send"] = true
	resp.StructuredData["prompt_version"] = "v1"
	if specID != "" {
		if _, ok := resp.StructuredData["template"]; !ok {
			resp.StructuredData["template"] = specID
		}
	}
	return resp, err
}

func infer(data map[string]any, task string) (action, templateID string) {
	if raw, ok := data["template"].(string); ok {
		if spec, ok := Lookup(raw); ok {
			templateID = spec.ID
		}
	}
	if raw, ok := data["action"].(string); ok {
		a := strings.ToLower(strings.TrimSpace(raw))
		if a == "draft" || a == "review" || a == "watch" {
			action = a
		}
	}
	lower := strings.ToLower(task + " " + str(data["template"]))
	if templateID == "" {
		switch {
		case strings.Contains(lower, "nda") || strings.Contains(lower, "non-disclosure") || strings.Contains(lower, "confidential"):
			templateID = "nda"
		case strings.Contains(lower, "contractor") || strings.Contains(lower, "consult"):
			templateID = "contractor"
		case strings.Contains(lower, "ip assignment") || strings.Contains(lower, "intellectual property") || strings.Contains(lower, "work product"):
			templateID = "ip_assignment"
		case strings.Contains(lower, "face of") || strings.Contains(lower, "project face") || strings.Contains(lower, "cofounder agreement"):
			templateID = "project_face"
		case strings.Contains(lower, "pilot") || strings.Contains(lower, "customer terms") || strings.Contains(lower, "sla"):
			templateID = "customer_terms"
		}
	}
	if action == "watch" {
		return action, ""
	}
	if action == "" {
		switch {
		case strings.Contains(lower, "flag") || strings.Contains(lower, "review") || strings.Contains(lower, "predatory"):
			action = "review"
		case templateID != "" || strings.Contains(lower, "draft") || strings.Contains(lower, "write") || strings.Contains(lower, "prepare"):
			action = "draft"
		default:
			action = "review"
		}
	}
	return action, templateID
}

func str(v any) string {
	s, _ := v.(string)
	return s
}
