// Package hr drafts JDs and scores applicants. Rows are proposed. Humans Shortlist. No hire email, no PAYE.
package hr

import (
	"context"
	"embed"
	"strings"

	"github.com/samuel/ai-workers/agents/internal/contract"
	"github.com/samuel/ai-workers/agents/internal/llm"
)

//go:embed templates/*.md
var templateFS embed.FS

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
	return contract.Response{
		Status: contract.StatusSuccess,
		OutputText: "Recruiting is off. To give someone work, the owner adds them on People, sets hours per week, and seats them on a project Face tab. Delivery then assigns tasks. This desk does not hire or file applicants.",
		StructuredData: map[string]any{
			"task_type":    "hr_closed",
			"cannot_hire":  true,
			"cannot_send":  true,
			"speak":        true,
			"save_role":    false,
			"save_application": false,
		},
	}, nil
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
