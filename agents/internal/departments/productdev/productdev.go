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

Be specific and technical. structured_data should include findings[], severity, and any redacted_secret_refs.`

func Handle(ctx context.Context, c llm.Completer, req contract.Request) (contract.Response, error) {
	return departments.Run(ctx, c, systemPrompt, departments.UserPrompt(req.TaskDescription, contract.ContextObject(req.ContextData)))
}
