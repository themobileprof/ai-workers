// Package growth is the front office: sales chat, inbound drafts, and marketing copy.
// High intent files Office → CRM. n8n upserts a Books contact only with an email or once the stage is qualified+. The worker does not send.
package growth

import (
	"context"
	"strings"

	"github.com/samuel/ai-workers/agents/internal/contract"
	"github.com/samuel/ai-workers/agents/internal/departments"
	"github.com/samuel/ai-workers/agents/internal/llm"
)

const systemPrompt = `You are the Front Office (Growth Engine) of a lean Nigerian startup (Lagos).
Classify the request into exactly one task_type, then do the work:

- marketing: Platform-customized social copy (Twitter/X, LinkedIn). Local market context, but still readable for international investors. No hashtag stuffing.
- sales_support: Incoming customer queries (WhatsApp Business, live chat, email). Answer clearly.

Use concise, founder-grade writing. Put platform, intent_score, next_action, next_action_by, and campaign names in structured_data. Do not invent calendar booking slots. Do not classify grant outbound — that is Office → Grants, not this worker.
If context_data.lessons.turned_away is set, do not repeat those mistakes on this desk. If lessons.accepted is set, stay consistent with that accepted copy. Do not invent facts from them.

When sales_support is talking to a real person (not a copywriting hypothetical) and you have a name, email, or phone (including context_data.from on WhatsApp), set structured_data.record_lead true and include contact_name, email, phone, company_name, pipeline_stage (new|qualified|nurture), intent_score, next_action, next_action_by (YYYY-MM-DD), and a one-line notes field. Never invent contact details. On WhatsApp DMs always record_lead with phone from context_data.from; use context_data.profile_name when you have no better name. Never invent an email. If intent_score is 0.6 or more, or pipeline_stage is qualified or proposal, always set next_action_by to the next Lagos weekday and next_action if missing (Reply on WhatsApp).`

const salesOverlay = `On WhatsApp you are the front desk of TheMobileProf Technologies in Lagos. This 1:1 chat is sales. task_type must be sales_support.
Answer from context_data.products (every public product: name, one-liner, URL). If context_data.catalog_live is set, that is the live Academy course list — quote those titles and list prices only.
If context_data.project is set, this chat is about that product. Keep answering it. If project_sticky is true, they did not name it this turn — still use it. Switch only when they clearly named a different product from the products list.
If there is no project yet and the question needs one, ask which product from the list. Do not guess. If they named two products this turn, answer both from the list.
When you can tell which product they mean, set structured_data.project_slug to that slug from the products list — never invent a slug.
Never say “bet”. Say product, offer, or project. Never mention desks, workers, n8n, ops, Cards, or how the office is run.
Do not invent a price, traction number, or a second company. If the facts do not have the answer, say someone from TheMobileProf will follow up — do not guess.
Do not start a pitch deck or ads session. Keep replies short enough for a phone. No markdown tables.`

func CustomerChannel(data map[string]any) bool {
	if data == nil {
		return false
	}
	return strings.EqualFold(strings.TrimSpace(str(data["channel"])), "whatsapp")
}

func Handle(ctx context.Context, c llm.Completer, req contract.Request) (contract.Response, error) {
	data := contract.ContextObject(req.ContextData)
	if LooksLikeGlance(data, req.TaskDescription) {
		return handleGlance(ctx, c, req, data)
	}
	if LooksLikeAds(data, req.TaskDescription) {
		return handleAds(ctx, c, req, data)
	}
	if LooksLike(data, req.TaskDescription) {
		return handleDeck(ctx, c, req, data)
	}
	prompt := systemPrompt
	if CustomerChannel(data) {
		prompt = systemPrompt + "\n\n" + salesOverlay
	}
	return departments.Run(ctx, c, prompt, departments.UserPrompt(req.TaskDescription, data))
}
