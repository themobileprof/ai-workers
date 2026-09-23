// Package productdev is the engineering lab: validation, QA, and journey placement.
// place_on_journey proposes; the desk Accepts. Hypotheses are still one-shot.
package productdev

import (
	"context"

	"github.com/samuel/ai-workers/agents/internal/contract"
	"github.com/samuel/ai-workers/agents/internal/departments"
	"github.com/samuel/ai-workers/agents/internal/llm"
)

const systemPrompt = `You are the Engineering Lab of a lean Nigerian startup.
Classify the request into exactly one task_type, then do the work:

- product_ops_qa: Code diffs, GitHub webhook payloads, or system logs. Find functional bugs and exposed secrets (API keys, tokens, private keys). Never repeat a full secret; redact to last 4 characters and name the file/path.
- customer_success: Cohort usage telemetry. Identify churn risk and write a hyper-personalized retention trigger. Put risk_score 0-1, cohort, and channel (email/whatsapp) in structured_data.
- product_validation: Idea/product validation loop (hypotheses, intern missions, evidence analysis, GO/PIVOT/KILL, journey placement). Prefer this when context_data has action, product_description, assumptions, hypotheses, evidence, or project.

Be specific and technical. structured_data should include findings[], severity, and any redacted_secret_refs for QA tasks.`

func Handle(ctx context.Context, c llm.Completer, req contract.Request) (contract.Response, error) {
	data := contract.ContextObject(req.ContextData)
	if action, ok := validationAction(data, req.TaskDescription); ok {
		return runValidation(ctx, c, req, data, action)
	}
	return departments.Run(ctx, c, systemPrompt, departments.UserPrompt(req.TaskDescription, data))
}
