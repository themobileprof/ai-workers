package admin

import (
	"net/http"
	"strconv"
	"strings"
	"time"
)

const crmDeskOrigin = "https://workers.themobileprof.com"

func crmBooksEligible(email, stage string) bool {
	email = strings.ToLower(strings.TrimSpace(email))
	if strings.Contains(email, "@") {
		return true
	}
	switch strings.ToLower(strings.TrimSpace(stage)) {
	case "qualified", "proposal", "won":
		return true
	}
	return false
}

func nextLagosBusinessDay(now time.Time) string {
	if now.IsZero() {
		now = time.Now()
	}
	t := now.In(lagosLocation()).Add(24 * time.Hour)
	for t.Weekday() == time.Saturday || t.Weekday() == time.Sunday {
		t = t.Add(24 * time.Hour)
	}
	return t.Format("2006-01-02")
}

func crmWhatsAppFollowup(channel, stage, nextAction, nextActionBy string, intent float64, now time.Time) (string, string) {
	if strings.ToLower(strings.TrimSpace(channel)) != "whatsapp" {
		return nextAction, nextActionBy
	}
	stage = strings.ToLower(strings.TrimSpace(stage))
	if intent < 0.6 && stage != "qualified" && stage != "proposal" {
		return nextAction, nextActionBy
	}
	if strings.TrimSpace(nextActionBy) == "" {
		nextActionBy = nextLagosBusinessDay(now)
	}
	if strings.TrimSpace(nextAction) == "" {
		nextAction = "Reply on WhatsApp"
	}
	return nextAction, nextActionBy
}

func crmLookupQuery(raw string) string {
	s := strings.TrimSpace(raw)
	s = strings.TrimPrefix(s, "/crm")
	s = strings.TrimSpace(s)
	lower := strings.ToLower(s)
	for _, p := range []string{"follow-up ", "follow up ", "followup "} {
		if strings.HasPrefix(lower, p) {
			s = strings.TrimSpace(s[len(p):])
			break
		}
	}
	s = strings.Trim(s, ".?! ")
	return s
}

func likeContains(q string) string {
	q = strings.ReplaceAll(q, `\`, `\\`)
	q = strings.ReplaceAll(q, `%`, `\%`)
	q = strings.ReplaceAll(q, `_`, `\_`)
	return "%" + q + "%"
}

func crmDueLine(c CRMContact, today time.Time) string {
	label := strings.TrimSpace(c.Name)
	if c.Company != "" && !strings.EqualFold(c.Company, c.Name) {
		if label == "" {
			label = c.Company
		} else {
			label += " · " + c.Company
		}
	}
	if label == "" {
		label = "Contact " + strconv.FormatInt(c.ID, 10)
	}
	action := strings.TrimSpace(c.NextAction)
	if action == "" {
		action = "next step not written"
	}
	when := "today"
	if due, err := time.ParseInLocation("2006-01-02", c.NextActionBy, lagosLocation()); err == nil {
		day := today.In(lagosLocation())
		start := time.Date(day.Year(), day.Month(), day.Day(), 0, 0, 0, 0, lagosLocation())
		if due.Before(start) {
			when = "overdue " + due.Format("2 Jan")
		}
	}
	href := crmDeskOrigin + "/admin/crm/" + strconv.FormatInt(c.ID, 10)
	return "• " + label + " — " + action + " (" + when + ")\n  " + href
}

func crmDueText(contacts []CRMContact, today time.Time) string {
	if len(contacts) == 0 {
		return ""
	}
	const capN = 12
	var b strings.Builder
	b.WriteString("CRM due today (Lagos)\n")
	n := len(contacts)
	if n > capN {
		n = capN
	}
	for i := 0; i < n; i++ {
		if i > 0 {
			b.WriteByte('\n')
		}
		b.WriteString(crmDueLine(contacts[i], today))
	}
	if len(contacts) > capN {
		b.WriteString("\n…and ")
		b.WriteString(strconv.Itoa(len(contacts) - capN))
		b.WriteString(" more on Office → CRM")
	}
	b.WriteString("\n\nThis does not message the customer. Open the person, Log after you act, or /crm follow up Apex.")
	text := b.String()
	if len(text) > 3900 {
		text = text[:3890] + "…"
	}
	return text
}

func crmDueDigest(contacts []CRMContact, users []User, today time.Time) map[string]any {
	day := today.In(lagosLocation()).Format("2006-01-02")
	out := map[string]any{
		"ok":     true,
		"today":  day,
		"count":  len(contacts),
		"people": []map[string]any{},
	}
	if len(contacts) == 0 {
		return out
	}
	text := crmDueText(contacts, today)
	var people []map[string]any
	for _, w := range mailWatchers(users, nil, "") {
		people = append(people, map[string]any{
			"telegram_id": w["telegram_id"],
			"name":        w["name"],
			"role":        w["role"],
			"text":        text,
			"speak":       true,
		})
	}
	out["people"] = people
	return out
}

func (s *Server) internalCRMDueGET(w http.ResponseWriter, r *http.Request) {
	if !s.internalOK(r) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	if s.store == nil {
		writeJSON(w, crmDueDigest(nil, nil, time.Now()))
		return
	}
	contacts, err := s.store.ListDueCRMContacts(r.Context(), time.Now())
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	users, err := s.store.ListUsers(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	writeJSON(w, crmDueDigest(contacts, users, time.Now()))
}
