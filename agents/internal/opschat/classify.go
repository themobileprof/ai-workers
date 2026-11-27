package opschat

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/samuel/ai-workers/agents/internal/departments"
	"github.com/samuel/ai-workers/agents/internal/llm"
)

const classifyOverlay = `You route one office Telegram message for TheMobileProf Technologies (Lagos). One CAC company.
Pick exactly one department. Do the routing only — do not do the desk's work. Do not list options. Do not ask which desk. Do not say "or".
Departments:
- delivery — field reportage and this week's work: hospital visits, trainings, what happened, what is next. Not “done 1”.
- growth — sales copy, customer replies, Kawasaki pitch-deck slides, Google Search ads pack
- legal — NDA/contract draft or claims review
- files — data-room classify when the message is mainly a file with no field story
- accounts — receipt, bill, invoice (Books) when the message is mainly money lines
- internal-ops — grants, mixed ops
- crm — closing or following a named person already on the book (Marialena done), not a new field visit
- product-dev — Card hypothesis/mission/evidence
- community — Academy student rooms (usually the other bot)
- hr — hiring copy (rare)
If they replied to "Today on the desk", delivery.
If they narrate a visit, a training, a partner deal, fliers, or what happened yesterday, delivery — even if they also mention spend.
If they are tightening slide copy or Search ads, growth.
structured_data MUST include department: one id from the list above.
output_text: unused, keep short.
status: success.`

// Classify returns exactly one department id.
func Classify(ctx context.Context, c llm.Completer, task, replyTo string) (string, error) {
	if c == nil {
		return "internal-ops", nil
	}
	user := task
	if strings.TrimSpace(replyTo) != "" {
		user = "They replied to:\n" + replyTo + "\n\nTheir text:\n" + task
	}
	resp, err := departments.Run(ctx, c, classifyOverlay, user)
	if err != nil {
		return "", err
	}
	dept := strings.TrimSpace(strings.ToLower(strVal(resp.StructuredData["department"])))
	if !KnownDepartment(dept) {
		dept = strings.TrimSpace(strings.ToLower(strVal(resp.StructuredData["route_to"])))
	}
	if !KnownDepartment(dept) {
		return "internal-ops", nil
	}
	return dept, nil
}

func strVal(v any) string {
	switch t := v.(type) {
	case string:
		return strings.TrimSpace(t)
	case []any:
		if len(t) == 0 {
			return ""
		}
		s, _ := t[0].(string)
		return strings.TrimSpace(s)
	default:
		b, _ := json.Marshal(v)
		return strings.Trim(string(b), `"`)
	}
}
