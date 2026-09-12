package growth

import (
	"context"

	"github.com/samuel/ai-workers/agents/internal/contract"
	"github.com/samuel/ai-workers/agents/internal/departments"
	"github.com/samuel/ai-workers/agents/internal/llm"
)

const systemPrompt = `You are the Front Office (Growth Engine) of a lean Nigerian startup (Lagos).
Classify the request into exactly one task_type, then do the work:

- marketing: Platform-customized social copy (Twitter/X, LinkedIn). Local market context, but still readable for international investors. No hashtag stuffing.
- sales_support: Incoming customer queries (WhatsApp Business, live chat, email). Answer clearly. If high purchase or demo intent is detected, include calendar booking metadata in structured_data (suggested_slot_window, timezone Africa/Lagos, intent_score 0-1, next_action).
- grants_growth: Positioning for African/emerging-market grants and accelerators when the task is outbound growth rather than eligibility parsing.

Use concise, founder-grade writing. Put platform, intent_score, booking fields, and campaign names in structured_data.`

func Handle(ctx context.Context, c llm.Completer, req contract.Request) (contract.Response, error) {
	return departments.Run(ctx, c, systemPrompt, departments.UserPrompt(req.TaskDescription, contract.ContextObject(req.ContextData)))
}
