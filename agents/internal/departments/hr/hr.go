package hr

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

const overlay = `You are the HR clerk for TheMobileProf Technologies (Lagos), the only CAC-registered company in this office.
Projects are bets, not companies. A project "cofounder" is a public face, not a director.
You draft JDs and score applicants against an open JD. You do not hire, fire, or email anyone.
You do not run PAYE, pension, or NHF. Tentacle (later) owns statutory payroll. Accounts pays when a human asks.
Do not invent labour-act section numbers. cannot_hire and cannot_send are always true.
Humans Accept on the company desk. Until then a role or application is a proposal.
Prefer contractor or intern language unless the handler explicitly wants employment.
Nigerian/West African commercial English. Plain sentences.
task_type must be "hr".
structured_data.prompt_version is "v1".`

type Spec struct {
	ID    string
	Label string
	File  string
	Kind  string
	Help  string
}

func Specs() []Spec {
	return []Spec{
		{ID: "intern", Label: "Intern", File: "intern.md", Kind: "intern", Help: "Time-boxed field intern. Not employment, not PAYE."},
		{ID: "contractor", Label: "Contractor", File: "contractor.md", Kind: "contractor", Help: "Independent contractor for TMP. Invoice, not salary."},
		{ID: "assistant", Label: "Project assistant", File: "assistant.md", Kind: "assistant", Help: "Helps the face of a named project. Still TMP."},
		{ID: "employee", Label: "Employee (PAYE later)", File: "employee.md", Kind: "employee", Help: "Employment JD only. Do not run payroll."},
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
	system := overlay + "\n\nAction: " + action
	if spec, ok := Lookup(specID); ok {
		body, err := templateFS.ReadFile("templates/" + spec.File)
		if err != nil {
			return contract.Fail("missing HR template: " + spec.ID), err
		}
		system += "\nTemplate id: " + spec.ID + " (" + spec.Label + ")\n\nUse this skeleton. Fill the role, project, and must-haves from context. Leave blanks as [TO BE COMPLETED].\n\n" + string(body)
	} else if action == "draft_jd" {
		system += "\nNo catalog template matched. Draft a short TMP JD and set template to contractor unless they clearly asked for intern, assistant, or employee."
	} else {
		system += "\nScore or parse the pasted application against open_roles in context. If none match, set role_slug empty and recommendation unclear."
	}
	system += `

structured_data MUST include:
- action: draft_jd | ingest | score | explain
- template: intern | contractor | assistant | employee | none
- kind: intern | contractor | assistant | employee
- role_slug (from open_roles or a new slug; never invent a company)
- role_title
- project_slug (from context if present)
- applicant_name, applicant_email, applicant_phone (from the paste; never invent an email)
- score: integer 0–100 (0 if drafting a JD)
- recommendation: shortlist | reject | unclear
- reasons: string[] (why they fit or fail the JD; required on score/explain)
- jd: full JD text when action is draft_jd
- cv_text: the application text when ingest/score
- cannot_hire: true
- cannot_send: true
- save_role: true only when jd is a full draft
- save_application: true only when there is an applicant to file
output_text: short next human step. Do not paste the full JD or CV.`

	resp, err := departments.Run(ctx, c, system, departments.UserPrompt(req.TaskDescription, data))
	if resp.StructuredData == nil {
		resp.StructuredData = map[string]any{}
	}
	resp.StructuredData["task_type"] = "hr"
	resp.StructuredData["action"] = action
	resp.StructuredData["cannot_hire"] = true
	resp.StructuredData["cannot_send"] = true
	resp.StructuredData["prompt_version"] = "v1"
	if specID != "" {
		if _, ok := resp.StructuredData["template"]; !ok {
			resp.StructuredData["template"] = specID
		}
		if spec, ok := Lookup(specID); ok {
			if _, exists := resp.StructuredData["kind"]; !exists {
				resp.StructuredData["kind"] = spec.Kind
			}
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
		switch a {
		case "draft_jd", "ingest", "score", "explain":
			action = a
		case "draft":
			action = "draft_jd"
		}
	}
	lower := strings.ToLower(task + " " + str(data["template"]))
	if templateID == "" {
		switch {
		case strings.Contains(lower, "intern"):
			templateID = "intern"
		case strings.Contains(lower, "assistant"):
			templateID = "assistant"
		case strings.Contains(lower, "employee") || strings.Contains(lower, "paye") || strings.Contains(lower, "full-time") || strings.Contains(lower, "full time"):
			templateID = "employee"
		case strings.Contains(lower, "contractor") || strings.Contains(lower, "consult"):
			templateID = "contractor"
		}
	}
	if action == "" {
		switch {
		case strings.Contains(lower, "why reject") || strings.Contains(lower, "reason") && strings.Contains(lower, "reject"):
			action = "explain"
		case looksLikeApplication(lower):
			action = "ingest"
		case strings.Contains(lower, "score") || strings.Contains(lower, "shortlist") || strings.Contains(lower, "screen"):
			action = "score"
		case strings.Contains(lower, "jd") || strings.Contains(lower, "job description") || strings.Contains(lower, "job spec") || strings.Contains(lower, "write a role") || strings.Contains(lower, "draft"):
			action = "draft_jd"
		default:
			action = "score"
		}
	}
	return action, templateID
}

func looksLikeApplication(lower string) bool {
	needles := []string{
		"i am applying", "i'm applying", "please find my", "attached cv", "attached resume",
		"curriculum vitae", " resume", "\ncv", " cv ", "application for",
		"i would like to join", "i want to intern", "cover letter",
	}
	for _, n := range needles {
		if strings.Contains(lower, n) {
			return true
		}
	}
	return strings.Contains(lower, "apply") && (strings.Contains(lower, "role") || strings.Contains(lower, "intern") || strings.Contains(lower, "job"))
}

func str(v any) string {
	s, _ := v.(string)
	return s
}
