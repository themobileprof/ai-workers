package admin

import (
	"net/http"
	"strconv"
	"strings"
	"time"
)

const crmDeskOrigin = "https://workers.themobileprof.com"

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
