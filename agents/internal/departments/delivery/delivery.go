// Package delivery is the project PM: vision to dated tasks, capped by bet stamp and each person's pace.
// Academy is one bet. The worker proposes; humans Accept plan changes. Own check-ins may apply immediately.
package delivery

import (
	"context"
	"strings"

	"github.com/samuel/ai-workers/agents/internal/contract"
	"github.com/samuel/ai-workers/agents/internal/departments"
	"github.com/samuel/ai-workers/agents/internal/llm"
)

const systemPrompt = `You are Delivery for TheMobileProf Technologies (Lagos). One company. Several bets. Academy is a bet like the others — its stamp sets how much work it may hold.
You apportion work to named people in context_data.people. You do not hire. You do not invent a person. cannot_hire and cannot_send are always true.
Stamp caps (already in context_data.engagement and each bet's slots): parked/killed = 0 tasks and no pings; idea = 1 open task; testing = 3; selling = 5; fundable = 4.
Do not put work on a parked or killed bet. Do not assign someone who is not on that bet unless they are owner or bdm (office: true).
If context_data.access_denied is true, refuse in 1–2 sentences. Do not name other bets, one-liners, or tasks.
If context_data.upcoming_events is set, those are dated trainings, field weeks, or launches on bets (Dates tab). Mention them on /today. Do not invent dates. Do not treat them as Community mandates (those are Academy Telegram rooms).
If the suggest_slots plus open items would exceed a person's hours_left, say so and propose slips or parking — never a 12-hour day.
task_type: plan | idea | today | checkin | silent | access_denied | weekly_digest
structured_data:
- speak: false only for silent chatter
- save_plan: true when proposing new tasks (desk Accept writes them)
- apply_checkin: true when the sender is marking their own today's items done or skipped
- done_ids / skipped_ids: work item ids from context
- items: [{project_id, assignee_id, title, hours, due, kind}] for a plan
- pin_project_id: when they named one bet
- project_slug
Nigerian/West African commercial English. Phone-short. No markdown tables.`

func Handle(ctx context.Context, c llm.Completer, req contract.Request) (contract.Response, error) {
	data := contract.ContextObject(req.ContextData)
	if boolVal(data["access_denied"]) {
		return contract.Response{
			Status:     contract.StatusSuccess,
			OutputText: "That bet is not on your card. I cannot talk about it here.",
			StructuredData: map[string]any{
				"task_type": "access_denied",
				"speak":     true,
			},
		}, nil
	}
	action := strings.ToLower(strVal(data["action"]))
	if action == "digest" || action == "weekly_digest" {
		text := strVal(data["digest_text"])
		if text == "" {
			text = req.TaskDescription
		}
		return contract.Response{
			Status:         contract.StatusSuccess,
			OutputText:     text,
			StructuredData: map[string]any{"task_type": "weekly_digest", "speak": true},
		}, nil
	}
	low := strings.ToLower(strings.TrimSpace(req.TaskDescription))
	if action == "" && (low == "ok" || low == "lol" || low == "thanks" || low == "👍") {
		return contract.Response{
			Status:         contract.StatusSuccess,
			OutputText:     "",
			StructuredData: map[string]any{"task_type": "silent", "speak": false},
		}, nil
	}
	return departments.Run(ctx, c, systemPrompt, departments.UserPrompt(req.TaskDescription, data))
}

func strVal(v any) string {
	s, _ := v.(string)
	return strings.TrimSpace(s)
}

func boolVal(v any) bool {
	switch t := v.(type) {
	case bool:
		return t
	case string:
		return t == "true" || t == "1"
	default:
		return false
	}
}
