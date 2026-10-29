// Package growth is the front office: sales chat, inbound drafts, and marketing copy.
// High intent files Office → CRM. n8n upserts a Books contact only with an email or once the stage is qualified+. The worker does not send.
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
- sales_support: Incoming customer queries (WhatsApp Business, live chat, email). Answer clearly.

Use concise, founder-grade writing. Put platform, intent_score, next_action, next_action_by, and campaign names in structured_data. Do not invent calendar booking slots. Do not classify grant outbound — that is Office → Grants, not this worker.

When sales_support is talking to a real person (not a copywriting hypothetical) and you have a name, email, or phone (including context_data.from on WhatsApp), set structured_data.record_lead true and include contact_name, email, phone, company_name, pipeline_stage (new|qualified|nurture), intent_score, next_action, next_action_by (YYYY-MM-DD), and a one-line notes field. Never invent contact details. On WhatsApp DMs always record_lead with phone from context_data.from; use context_data.profile_name when you have no better name. Never invent an email. If intent_score is 0.6 or more, or pipeline_stage is qualified or proposal, always set next_action_by to the next Lagos weekday and next_action if missing (Reply on WhatsApp).`

func Handle(ctx context.Context, c llm.Completer, req contract.Request) (contract.Response, error) {
	data := contract.ContextObject(req.ContextData)
	if LooksLike(data, req.TaskDescription) {
		return handleDeck(ctx, c, req, data)
	}
	return departments.Run(ctx, c, systemPrompt, departments.UserPrompt(req.TaskDescription, data))
}
