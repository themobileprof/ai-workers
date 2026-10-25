package productdev

import (
	"context"
	"embed"
	"strings"

	"github.com/samuel/ai-workers/agents/internal/contract"
	"github.com/samuel/ai-workers/agents/internal/departments"
	"github.com/samuel/ai-workers/agents/internal/journeys"
	"github.com/samuel/ai-workers/agents/internal/llm"
)

//go:embed prompts/*.md
var validationPrompts embed.FS

const validationOverlay = `You are the product validation manager for a lean startup (often Lagos / West Africa).
The intern does field work. You only propose. Application/n8n code persists and decides.
Behaviour beats opinions. Evidence beats assumptions. Never fabricate interviews or market facts.
Never let "they liked the idea" stand in for usage or payment.
You never write the committed journey or gate stamp. Humans Accept / Amend / Reject on the desk.
Hypothesis, this week's mission, and evidence summaries propose Card fields. Accept on Product writes the Card. Chat does not.
If context_data.projects or context_data.project names a bet, set project_slug to that slug. Do not invent a bet.
task_type must be "product_validation".
structured_data.action must echo the action you performed.
structured_data.cannot_write_stamp is always true.
structured_data.prompt_version is "v1".`

var validationFiles = map[string]string{
	"generate_hypotheses":       "hypothesis_generation.md",
	"generate_plan":             "validation_plan.md",
	"generate_mission":          "mission_generation.md",
	"generate_interview_guide":  "interview_guide.md",
	"analyse_evidence":          "evidence_analysis.md",
	"update_hypothesis":         "hypothesis_update.md",
	"recommend_next_experiment": "next_experiment.md",
	"generate_decision_report":  "decision_report.md",
	"place_on_journey":          "place_on_journey.md",
}

func validationAction(data map[string]any, task string) (string, bool) {
	if raw, ok := data["action"].(string); ok {
		action := strings.ToLower(strings.TrimSpace(raw))
		if _, known := validationFiles[action]; known {
			return action, true
		}
	}
	lower := strings.ToLower(task)
	switch {
	case strings.Contains(lower, "hypothesis") || strings.Contains(lower, "hypotheses"):
		return "generate_hypotheses", true
	case strings.Contains(lower, "interview guide"):
		return "generate_interview_guide", true
	case strings.Contains(lower, "analyse evidence") || strings.Contains(lower, "analyze evidence"):
		return "analyse_evidence", true
	case strings.Contains(lower, "decision") || strings.Contains(lower, "go/pivot") || strings.Contains(lower, "kill"):
		return "generate_decision_report", true
	case strings.Contains(lower, "next experiment") || strings.Contains(lower, "next mission"):
		return "recommend_next_experiment", true
	case strings.Contains(lower, "place") && strings.Contains(lower, "journey"):
		return "place_on_journey", true
	case strings.Contains(lower, "which gate") || strings.Contains(lower, "current gate"):
		return "place_on_journey", true
	case strings.Contains(lower, "validation plan") || strings.Contains(lower, "phases"):
		return "generate_plan", true
	case strings.Contains(lower, "mission"):
		return "generate_mission", true
	}
	if _, ok := data["product_description"]; ok {
		return "generate_hypotheses", true
	}
	if _, ok := data["assumptions"]; ok {
		return "generate_hypotheses", true
	}
	if _, ok := data["project"]; ok {
		return "place_on_journey", true
	}
	return "", false
}

func runValidation(ctx context.Context, c llm.Completer, req contract.Request, data map[string]any, action string) (contract.Response, error) {
	file := validationFiles[action]
	body, err := validationPrompts.ReadFile("prompts/" + file)
	if err != nil {
		return contract.Fail("missing validation prompt: " + action), err
	}
	system := validationOverlay + "\n\nAction: " + action + "\n\n" + string(body)
	if action == "place_on_journey" {
		system += "\n\n" + journeys.PromptBlock()
	}
	resp, err := departments.Run(ctx, c, system, departments.UserPrompt(req.TaskDescription, data))
	if resp.StructuredData == nil {
		resp.StructuredData = map[string]any{}
	}
	resp.StructuredData["task_type"] = "product_validation"
	resp.StructuredData["action"] = action
	resp.StructuredData["cannot_write_stamp"] = true
	resp.StructuredData["prompt_version"] = "v1"
	stampCardProposal(resp.StructuredData, action)
	if slug := projectSlug(data); slug != "" {
		if str(resp.StructuredData["project_slug"]) == "" {
			resp.StructuredData["project_slug"] = slug
		}
	}
	return resp, err
}

func stampCardProposal(sd map[string]any, action string) {
	switch action {
	case "generate_hypotheses", "update_hypothesis":
		if h := flattenHypotheses(sd); h != "" {
			sd["card_hypothesis"] = h
			sd["save_memory"] = true
			sd["needs_human"] = true
			sd["title"] = "Card · hypothesis"
		}
	case "generate_mission", "recommend_next_experiment":
		m, due := flattenMission(sd)
		if m != "" {
			sd["card_mission"] = m
			if due != "" {
				sd["card_mission_due"] = due
			}
			sd["save_memory"] = true
			sd["needs_human"] = true
			sd["title"] = "Card · mission"
		}
	case "analyse_evidence":
		ev := strings.TrimSpace(str(sd["summary"]))
		if ev == "" {
			ev = strings.TrimSpace(str(sd["card_evidence"]))
		}
		if ev != "" {
			sd["card_evidence"] = clip(ev, 400)
			sd["save_memory"] = true
			sd["needs_human"] = true
			sd["title"] = "Card · evidence"
		}
		if h := strings.TrimSpace(str(sd["updated_hypothesis"])); h != "" {
			sd["card_hypothesis"] = clip(h, 800)
			sd["save_memory"] = true
			sd["needs_human"] = true
		}
	case "generate_decision_report":
		sd["needs_human"] = true
		sd["save_memory"] = false
		dec := strings.ToUpper(strings.TrimSpace(str(sd["decision"])))
		if dec != "" {
			sd["title"] = "Decide · " + dec
		} else {
			sd["title"] = "Decide · recommendation"
		}
	case "generate_plan", "generate_interview_guide":
		sd["needs_human"] = true
		if str(sd["title"]) == "" {
			sd["title"] = "Product · " + strings.ReplaceAll(action, "_", " ")
		}
	}
}

func flattenHypotheses(sd map[string]any) string {
	if h := strings.TrimSpace(str(sd["card_hypothesis"])); h != "" {
		return clip(h, 800)
	}
	raw := sd["hypotheses"]
	arr, ok := raw.([]any)
	if !ok {
		return clip(str(sd["hypothesis"]), 800)
	}
	var parts []string
	for _, x := range arr {
		m, ok := x.(map[string]any)
		if !ok {
			continue
		}
		stmt := strings.TrimSpace(str(m["statement"]))
		if stmt == "" {
			stmt = strings.TrimSpace(str(m["title"]))
		}
		if stmt == "" {
			continue
		}
		parts = append(parts, stmt)
		if len(parts) >= 3 {
			break
		}
	}
	return clip(strings.Join(parts, "\n"), 800)
}

func flattenMission(sd map[string]any) (string, string) {
	due := isoDate(str(sd["card_mission_due"]))
	if due == "" {
		due = isoDate(str(sd["mission_due"]))
	}
	if m := strings.TrimSpace(str(sd["card_mission"])); m != "" {
		return clip(m, 400), due
	}
	switch m := sd["mission"].(type) {
	case string:
		return clip(strings.TrimSpace(m), 400), due
	case map[string]any:
		title := strings.TrimSpace(str(m["title"]))
		obj := strings.TrimSpace(str(m["objective"]))
		line := title
		if obj != "" && obj != title {
			if line != "" {
				line = line + " — " + obj
			} else {
				line = obj
			}
		}
		if due == "" {
			due = isoDate(str(m["due"]))
		}
		return clip(line, 400), due
	default:
		return "", due
	}
}

func projectSlug(data map[string]any) string {
	if data == nil {
		return ""
	}
	if s := strings.TrimSpace(str(data["project_slug"])); s != "" {
		return s
	}
	switch p := data["project"].(type) {
	case string:
		return strings.TrimSpace(p)
	case map[string]any:
		return strings.TrimSpace(str(p["slug"]))
	default:
		return ""
	}
}

func isoDate(s string) string {
	s = strings.TrimSpace(s)
	if len(s) >= 10 {
		s = s[:10]
	}
	if len(s) != 10 || s[4] != '-' || s[7] != '-' {
		return ""
	}
	for _, i := range []int{0, 1, 2, 3, 5, 6, 8, 9} {
		if s[i] < '0' || s[i] > '9' {
			return ""
		}
	}
	return s
}

func clip(s string, n int) string {
	s = strings.TrimSpace(s)
	if n <= 0 || len(s) <= n {
		return s
	}
	return strings.TrimSpace(s[:n-1]) + "…"
}

func str(v any) string {
	s, _ := v.(string)
	return s
}
