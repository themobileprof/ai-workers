package growth

import (
	"net/url"
	"strings"
	"unicode"
)

const adsFunnelCap = 5

type AdsFunnelStep struct {
	Name  string `json:"name"`
	Event string `json:"event,omitempty"`
	Path  string `json:"path,omitempty"`
}

func AdsCellContent(c AdsCell) string {
	return adsUTMToken(strings.TrimSpace(c.Geo)+" "+strings.TrimSpace(c.Theme), 80)
}

// AdsTrackURL stamps conservative UTM on a Place landing URL so GA4 can join
// the bet slug and cell. Existing utm_* query keys are left alone.
func AdsTrackURL(base, channel, slug, content string) string {
	raw := strings.TrimSpace(base)
	if raw == "" {
		return ""
	}
	u, err := url.Parse(raw)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return raw
	}
	q := u.Query()
	source := "google"
	if strings.EqualFold(strings.TrimSpace(channel), "meta") {
		source = "facebook"
	}
	if q.Get("utm_source") == "" {
		q.Set("utm_source", source)
	}
	if q.Get("utm_medium") == "" {
		q.Set("utm_medium", "cpc")
	}
	if q.Get("utm_campaign") == "" {
		if tok := adsUTMToken(slug, 80); tok != "" {
			q.Set("utm_campaign", tok)
		}
	}
	if q.Get("utm_content") == "" {
		if tok := adsUTMToken(content, 80); tok != "" {
			q.Set("utm_content", tok)
		}
	}
	u.RawQuery = q.Encode()
	return u.String()
}

func adsUTMToken(s string, n int) string {
	s = strings.TrimSpace(strings.ToLower(s))
	if s == "" || n <= 0 {
		return ""
	}
	var b strings.Builder
	dash := false
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsNumber(r) {
			b.WriteRune(r)
			dash = false
			continue
		}
		if !dash && b.Len() > 0 {
			b.WriteByte('-')
			dash = true
		}
	}
	out := strings.Trim(b.String(), "-")
	if utf8Count := utf8RuneLen(out); utf8Count > n {
		out = string([]rune(out)[:n])
		out = strings.Trim(out, "-")
	}
	return out
}

func utf8RuneLen(s string) int {
	return len([]rune(s))
}

func adsEventName(s string) string {
	s = strings.TrimSpace(s)
	var b strings.Builder
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsNumber(r) || r == '_' {
			b.WriteRune(r)
		}
	}
	return ClipRunes(b.String(), 40)
}

func DefaultAdsFunnelSteps() []AdsFunnelStep {
	return []AdsFunnelStep{
		{Name: "Session", Event: "session_start"},
		{Name: "Landing", Event: "page_view"},
	}
}

// ParseAdsFunnelSteps reads Defaults ads.funnel_steps or the Card funnel line.
// Each step is "Label:event_name" or "Label:/path", comma or newline separated.
func ParseAdsFunnelSteps(raw string) []AdsFunnelStep {
	out := parseAdsFunnelLine(raw)
	if len(out) < 2 {
		return DefaultAdsFunnelSteps()
	}
	return out
}

func AdsFunnelLineOK(raw string) bool {
	return len(parseAdsFunnelLine(raw)) >= 2
}

func parseAdsFunnelLine(raw string) []AdsFunnelStep {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	parts := strings.FieldsFunc(raw, func(r rune) bool { return r == '\n' || r == ';' })
	if len(parts) == 1 && strings.Contains(raw, ",") {
		parts = strings.Split(raw, ",")
	}
	var out []AdsFunnelStep
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		name, rest, ok := strings.Cut(p, ":")
		name, rest = strings.TrimSpace(name), strings.TrimSpace(rest)
		if !ok || name == "" || rest == "" {
			continue
		}
		step := AdsFunnelStep{Name: ClipRunes(name, 40)}
		if strings.HasPrefix(rest, "/") {
			step.Path = ClipRunes(rest, 120)
		} else {
			step.Event = adsEventName(rest)
			if step.Event == "" {
				continue
			}
		}
		out = append(out, step)
		if len(out) == adsFunnelCap {
			break
		}
	}
	return out
}

func AdsPaidReportBody() map[string]any {
	return map[string]any{
		"dateRanges": []map[string]string{{"startDate": "7daysAgo", "endDate": "yesterday"}},
		"dimensions": []map[string]string{
			{"name": "sessionSourceMedium"},
			{"name": "sessionCampaignName"},
			{"name": "sessionManualAdContent"},
		},
		"metrics": []map[string]string{
			{"name": "sessions"},
			{"name": "engagedSessions"},
			{"name": "bounceRate"},
		},
		"dimensionFilter": map[string]any{
			"orGroup": map[string]any{
				"expressions": []map[string]any{
					{"filter": map[string]any{"fieldName": "sessionMedium", "stringFilter": map[string]any{"matchType": "EXACT", "value": "cpc"}}},
					{"filter": map[string]any{"fieldName": "sessionDefaultChannelGroup", "stringFilter": map[string]any{"matchType": "EXACT", "value": "Paid Search"}}},
					{"filter": map[string]any{"fieldName": "sessionDefaultChannelGroup", "stringFilter": map[string]any{"matchType": "EXACT", "value": "Paid Social"}}},
				},
			},
		},
		"limit":    30,
		"orderBys": []map[string]any{{"desc": true, "metric": map[string]string{"metricName": "sessions"}}},
	}
}

func AdsFunnelReportBody(steps []AdsFunnelStep) map[string]any {
	if len(steps) < 2 {
		steps = DefaultAdsFunnelSteps()
	}
	var funnelSteps []map[string]any
	for i, s := range steps {
		item := map[string]any{
			"name":             s.Name,
			"filterExpression": adsFunnelFilter(s),
		}
		if i > 0 {
			item["isDirectlyFollowedBy"] = false
		}
		funnelSteps = append(funnelSteps, item)
	}
	return map[string]any{
		"dateRanges": []map[string]string{{"startDate": "7daysAgo", "endDate": "yesterday"}},
		"funnel": map[string]any{
			"isOpenFunnel": false,
			"steps":        funnelSteps,
		},
		"funnelBreakdown": map[string]any{
			"breakdownDimension": map[string]string{"name": "sessionCampaignName"},
			"limit":              15,
		},
	}
}

func adsFunnelFilter(s AdsFunnelStep) map[string]any {
	if s.Path != "" {
		return map[string]any{
			"funnelEventFilter": map[string]any{
				"eventName": "page_view",
				"funnelParameterFilterExpression": map[string]any{
					"funnelParameterFilter": map[string]any{
						"eventParameterName": "page_path",
						"stringFilter":       map[string]any{"matchType": "CONTAINS", "value": s.Path},
					},
				},
			},
		}
	}
	event := strings.TrimSpace(s.Event)
	if event == "" {
		event = "page_view"
	}
	return map[string]any{
		"funnelEventFilter": map[string]any{"eventName": event},
	}
}
