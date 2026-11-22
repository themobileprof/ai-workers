package growth

import (
	"context"
	"fmt"
	"strings"

	"github.com/samuel/ai-workers/agents/internal/contract"
	"github.com/samuel/ai-workers/agents/internal/llm"
)

// LooksLikeGlance is a read-only paid-traffic question, not a new RSA pack.
func LooksLikeGlance(data map[string]any, task string) bool {
	if strings.EqualFold(strings.TrimSpace(str(data["channel"])), "whatsapp") {
		return false
	}
	lower := strings.ToLower(task)
	for _, k := range []string{
		"ga4", "paid session", "paid traffic", "how are the ads", "how did the ads",
		"ads doing", "evaluate the ads", "did anyone click", "ads funnel",
	} {
		if strings.Contains(lower, k) {
			return true
		}
	}
	if strings.Contains(lower, "funnel") && (strings.Contains(lower, "ad") || strings.Contains(lower, "ga4") || strings.Contains(lower, "paid")) {
		return true
	}
	return false
}

func handleGlance(_ context.Context, _ llm.Completer, req contract.Request, data map[string]any) (contract.Response, error) {
	body := FormatAdsGlance(asStringMap(data["ads_ga4"]))
	if body == "" {
		body = "No paid GA4 snapshot on the desk yet. Place stamps utm_campaign to the bet slug. This clerk cannot spend."
	}
	looks := looksFrom(data)
	return contract.Response{
		Status:     contract.StatusSuccess,
		OutputText: body,
		StructuredData: map[string]any{
			"task_type":    "ads_glance",
			"action":       "ads_glance",
			"cannot_send":  true,
			"cannot_spend": true,
			"save_deck":    false,
			"record_lead":  false,
			"looks":        looks,
			"project_slug": nestedSlug(data),
		},
	}, nil
}

func FormatAdsGlance(ga4 map[string]any) string {
	if len(ga4) == 0 {
		return ""
	}
	if found, ok := ga4["found"].(bool); ok && !found {
		return ""
	}
	var b strings.Builder
	b.WriteString("Paid traffic glance (last 7 days). This is not Place. Chat cannot spend.\n")
	if n := str(ga4["note"]); n != "" {
		b.WriteString(n + "\n")
	}
	paid := anyList(ga4["paid"])
	if len(paid) == 0 {
		b.WriteString("No paid sessions tagged with a bet slug yet.\n")
		return b.String()
	}
	b.WriteString("\nSessions:\n")
	for i, row := range paid {
		if i == 8 {
			break
		}
		m, _ := row.(map[string]any)
		if m == nil {
			continue
		}
		camp := str(m["campaign"])
		if camp == "" {
			camp = "(no campaign)"
		}
		cell := str(m["content"])
		line := "- " + camp
		if cell != "" {
			line += " / " + cell
		}
		line += fmt.Sprintf(" · %.0f sessions", floatOf(m["sessions"]))
		if e := floatOf(m["engaged"]); e > 0 {
			line += fmt.Sprintf(", %.0f engaged", e)
		}
		b.WriteString(line + "\n")
	}
	if steps := anyList(ga4["funnel"]); len(steps) > 0 {
		b.WriteString("\nFunnel (events that already fire):\n")
		for _, row := range steps {
			m, _ := row.(map[string]any)
			if m == nil {
				continue
			}
			name := str(m["name"])
			if name == "" {
				name = "step"
			}
			b.WriteString(fmt.Sprintf("- %s · %.0f users\n", name, floatOf(m["users"])))
		}
	}
	b.WriteString("\nA third funnel step stays 0 until that event exists on the site (TRAFFIC.md / Card). Do not invent generate_lead.")
	return b.String()
}

func asStringMap(v any) map[string]any {
	m, _ := v.(map[string]any)
	return m
}

func anyList(v any) []any {
	list, _ := v.([]any)
	return list
}

func floatOf(v any) float64 {
	switch t := v.(type) {
	case float64:
		return t
	case int:
		return float64(t)
	case int64:
		return float64(t)
	default:
		return 0
	}
}
