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
task_type must be "product_validation".
structured_data.action must echo the action you performed.
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
	resp.StructuredData["prompt_version"] = "v1"
	return resp, err
}
