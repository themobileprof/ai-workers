package delivery

import (
	"context"
	"encoding/json"
	"strconv"
	"strings"

	"github.com/samuel/ai-workers/agents/internal/contract"
	"github.com/samuel/ai-workers/agents/internal/departments"
	"github.com/samuel/ai-workers/agents/internal/llm"
)

const reportPrompt = `You are Delivery’s field clerk for TheMobileProf Technologies (Lagos). The human is reporting what happened, in their own words. They will not say “done 1”. Read the paragraph as-is.

You extract and file. You do not ask them to pick a number. You do not Accept, Place, or post Zoho Books.

task_type must be "report".
speak true. output_text is a short recap of what you logged (names, dates, amounts) in Nigerian/West African commercial English. Phone-short. No markdown tables. Do not ask them to confirm with yes unless you proposed work for someone else.

structured_data:
- summary: one sentence of what happened
- log: their narrative, lightly cleaned, still their words
- apply_checkin: true when open items in context_data.bets match what they finished or skipped
- done_ids: work item ids whose titles they completed (match by meaning, not a list number). Empty if none.
- skipped_ids: ids they postponed or could not do
- done_titles / skipped_titles: titles if you cannot see an id
- still_titles: titles still in progress
- contacts: [{name, company, next_action, next_action_by (YYYY-MM-DD), notes, project_slug, record_lead, offer, pipeline_stage}]
  record_lead true when this is a real person or organisation from the field (hospital, HOD, partner, campus, discount-code seller). Do not invent email or phone. Name or company is enough. Reuse context_data.existing_contacts when the same person or company is already on the book.
- events: [{project_slug, title, kind (training|field|launch|other), starts_on (YYYY-MM-DD), venue, audience, outcome}]
  Resolve relative dates from context_data.today (Africa/Lagos). “next Monday” is the next Monday after today. “yesterday” is today minus one day.
- projects: [{slug, card_note}] short this-week notes to append to the Card. Only slugs from context_data.bets.
- spend: [{amount_ngn, what, vendor}] money they spent. Do not set zoho_action. The desk files an Accounts job; chat does not post Books.
- items: new tasks for the reporter only [{project_id, assignee_id (the actor), title, hours, due, kind}]
  save_own_items true when items are the reporter’s next steps from this paragraph.
  save_plan true only when proposing work for someone else — those stay on Job chat for Accept.
cannot_send true. cannot_hire true. cannot_spend true.
If the paragraph is only chatter with nothing to file, task_type silent, speak false, empty output_text.`

func handleReport(ctx context.Context, c llm.Completer, req contract.Request, data map[string]any) (contract.Response, error) {
	low := strings.ToLower(strings.TrimSpace(req.TaskDescription))
	if low == "ok" || low == "lol" || low == "thanks" || low == "👍" {
		return contract.Response{
			Status:         contract.StatusSuccess,
			OutputText:     "",
			StructuredData: map[string]any{"task_type": "silent", "speak": false},
		}, nil
	}
	if wantsRecap(req.TaskDescription) {
		text := recapToday(data)
		return contract.Response{
			Status:     contract.StatusSuccess,
			OutputText: text,
			StructuredData: map[string]any{
				"task_type":     "report",
				"speak":         true,
				"cannot_send":   true,
				"cannot_hire":   true,
				"cannot_spend":  true,
				"apply_checkin": false,
				"save_plan":     false,
			},
		}, nil
	}
	images := departments.TakeImages(data)
	resp, err := departments.Run(ctx, c, reportPrompt, departments.UserPrompt(req.TaskDescription, data), images...)
	if resp.StructuredData == nil {
		resp.StructuredData = map[string]any{}
	}
	if _, ok := resp.StructuredData["task_type"]; !ok {
		resp.StructuredData["task_type"] = "report"
	}
	resp.StructuredData["cannot_send"] = true
	resp.StructuredData["cannot_hire"] = true
	resp.StructuredData["cannot_spend"] = true
	delete(resp.StructuredData, "zoho_action")
	return resp, err
}

func wantsRecap(task string) bool {
	t := strings.ToLower(strings.TrimSpace(task))
	if t == "" {
		return true
	}
	switch t {
	case "check in on the morning card.",
		"what happened on the bets. log it in your own words.",
		"today's tasks and check-ins.":
		return true
	default:
		return false
	}
}

func recapToday(data map[string]any) string {
	actor, _ := data["actor"].(map[string]any)
	name := strVal(actor["name"])
	if name == "" {
		name = "you"
	}
	actorID := int64Val(actor["id"])
	var lines []string
	for _, raw := range anySlice(data["bets"]) {
		bet, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		proj := firstNonEmpty(strVal(bet["name"]), strVal(bet["slug"]))
		for _, itRaw := range anySlice(bet["items"]) {
			it, ok := itRaw.(map[string]any)
			if !ok {
				continue
			}
			st := strings.ToLower(strVal(it["status"]))
			if st != "" && st != "open" {
				continue
			}
			if actorID > 0 && int64Val(it["assignee_id"]) != actorID {
				continue
			}
			title := strVal(it["title"])
			if title == "" {
				continue
			}
			line := proj + ": " + title
			if due := strVal(it["due"]); due != "" {
				line += " (due " + due + ")"
			}
			lines = append(lines, line)
		}
	}
	text := "Today on the desk (" + name + ")."
	if len(lines) == 0 {
		text += " No open tasks assigned to you."
	} else {
		text += "\n- " + strings.Join(lines, "\n- ")
	}
	var coming []string
	for _, raw := range anySlice(data["upcoming_events"]) {
		ev, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		title := strVal(ev["title"])
		if title == "" {
			continue
		}
		when := firstNonEmpty(strVal(ev["when"]), strVal(ev["starts_on"]))
		proj := strVal(ev["project"])
		line := title
		if proj != "" {
			line = proj + ": " + title
		}
		if when != "" {
			line += " (" + when + ")"
		}
		coming = append(coming, line)
	}
	if len(coming) > 0 {
		text += "\nComing up:\n- " + strings.Join(coming, "\n- ")
	}
	text += "\n\nReply in your own words what happened. I log CRM, Dates, Card notes, and spend from the paragraph. You do not need to number tasks."
	return text
}

func anySlice(v any) []any {
	switch t := v.(type) {
	case []any:
		return t
	case []map[string]any:
		out := make([]any, 0, len(t))
		for _, m := range t {
			out = append(out, m)
		}
		return out
	default:
		return nil
	}
}

func firstNonEmpty(a, b string) string {
	if strings.TrimSpace(a) != "" {
		return strings.TrimSpace(a)
	}
	return strings.TrimSpace(b)
}

func int64Val(v any) int64 {
	switch n := v.(type) {
	case int64:
		return n
	case int:
		return int64(n)
	case float64:
		return int64(n)
	case json.Number:
		i, _ := n.Int64()
		return i
	case string:
		i, _ := strconv.ParseInt(strings.TrimSpace(n), 10, 64)
		return i
	default:
		return 0
	}
}
