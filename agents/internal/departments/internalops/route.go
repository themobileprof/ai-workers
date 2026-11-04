package internalops

import (
	"context"
	"strings"

	"github.com/samuel/ai-workers/agents/internal/contract"
	"github.com/samuel/ai-workers/agents/internal/departments"
	"github.com/samuel/ai-workers/agents/internal/llm"
)

const routeOverlay = `You are the office receptionist on ops Telegram for TheMobileProf Technologies (Lagos). One CAC company. Bets are not baby companies.
Staff already identified. You cannot send, Accept, hire, publish, write Drive, or post to Zoho. Chat never Accepts.
A human sent plain text (no /prefix). Name 1–3 desks this should go to. Smallest set. Do not do the work yourself.
Desks and prefixes:
- growth /growth — sales copy, customer replies, Kawasaki pitch-deck session
- delivery /today — this week's tasks, morning-card check-ins (done / still / blocked)
- crm /crm — a person on the conversion book
- legal /legal — NDA/contract draft or claims review
- seo /seo — the bet's own page pack
- files /file — data-room classify
- accounts /accounts — receipt, bill, invoice (Books)
- internal-ops /ops — grants, mixed ops, or /ops legal-read (does not file an NDA)
- product-dev /validate — Card hypothesis/mission/evidence
- community /cm — Academy student rooms (usually the other bot)
If they replied to a message that starts "Today on the desk", Delivery /today is the usual desk for activity updates.
If they are tightening slide copy, Growth /growth.
output_text: 2–5 short lines. Name each desk with its slash prefix. Ask them to reply yes to send, or name one desk. Never empty.
structured_data MUST include:
- task_type: route
- speak: true
- cannot_send: true
- record_lead: false
- route_to: array of department ids from the list above
- route_task: the task those desks should receive (include the quoted morning card or reply_to when relevant)
Do not invent waitlist, MAU, or a second company.`

func looksLikeRoute(data map[string]any) bool {
	a := strings.ToLower(strings.TrimSpace(strVal(data["action"])))
	return a == "route" || a == "reception" || a == "receptionist"
}

func handleRoute(ctx context.Context, c llm.Completer, req contract.Request, data map[string]any) (contract.Response, error) {
	user := departments.UserPrompt(req.TaskDescription, data)
	if reply := strVal(data["reply_to"]); reply != "" {
		user = "They replied to this office message:\n" + reply + "\n\nTheir text:\n" + user
	}
	resp, err := departments.Run(ctx, c, routeOverlay, user)
	if resp.StructuredData == nil {
		resp.StructuredData = map[string]any{}
	}
	resp.StructuredData["task_type"] = "route"
	resp.StructuredData["speak"] = true
	resp.StructuredData["cannot_send"] = true
	resp.StructuredData["record_lead"] = false
	if strings.TrimSpace(resp.OutputText) == "" {
		resp.OutputText = "I can send this to a desk. Reply yes, or start with /today, /growth, /legal."
	}
	if _, ok := resp.StructuredData["route_to"]; !ok {
		resp.StructuredData["route_to"] = []string{}
	}
	if strVal(resp.StructuredData["route_task"]) == "" {
		resp.StructuredData["route_task"] = strings.TrimSpace(req.TaskDescription)
	}
	return resp, err
}

func strVal(v any) string {
	s, _ := v.(string)
	return strings.TrimSpace(s)
}
