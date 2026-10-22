package admin

import (
	"net/http"
	"strings"
)

// staffMay is whether this People card may run that department on the ops Telegram bot.
func staffMay(u User, dept string) bool {
	dept = strings.TrimSpace(strings.ToLower(dept))
	if dept == "" || !u.Active {
		return false
	}
	if canOffice(&u) {
		return true
	}
	if dept == "files" {
		return u.Role == "cofounder" || Handles(u, "files")
	}
	if dept == "delivery" && (u.Role == "cofounder" || u.Role == "assistant") {
		return true
	}
	if d, ok := DeskByDepartment(dept); ok {
		return Handles(u, d.Slug)
	}
	return false
}

func staffCanApprove(u User) bool {
	return canOffice(&u)
}

func staffPrefixes(u User) []string {
	seen := map[string]bool{}
	var out []string
	add := func(p string) {
		p = strings.TrimSpace(p)
		if p == "" || seen[p] {
			return
		}
		seen[p] = true
		out = append(out, p)
	}
	if canOffice(&u) {
		for _, d := range AllDesks() {
			add(d.Prefix)
		}
		add("/file")
		add("/approve")
		add("/stop")
		return out
	}
	for _, slug := range u.Desks {
		if d, ok := LookupDesk(slug); ok {
			add(d.Prefix)
		}
		if slug == "files" {
			add("/file")
		}
	}
	if u.Role == "cofounder" {
		add("/today")
		add("/file")
	}
	if u.Role == "assistant" {
		add("/today")
	}
	return out
}

func staffDefaultDept(u User) string {
	if canOffice(&u) || Handles(u, "internal-ops") {
		return "internal-ops"
	}
	if staffMay(u, "delivery") {
		return "delivery"
	}
	desks := visibleDesks(&u)
	if len(desks) > 0 {
		return desks[0].Department
	}
	return ""
}

func (s *Server) internalStaffGET(w http.ResponseWriter, r *http.Request) {
	if !s.internalOK(r) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	tid := firstNonEmpty(r.URL.Query().Get("telegram_id"), r.URL.Query().Get("from"))
	u, ok, err := s.store.LookupUserByTelegram(r.Context(), tid)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	if !ok {
		writeJSON(w, map[string]any{"ok": true, "known": false, "telegram_id": normalizeTelegramID(tid)})
		return
	}
	projects, _ := s.store.ListProjects(r.Context())
	var bets []map[string]any
	for _, p := range projects {
		if !canSeeBet(u, p.Members) {
			continue
		}
		bets = append(bets, map[string]any{"id": p.ID, "slug": p.Slug, "name": p.Name, "stage": p.Stage})
	}
	writeJSON(w, map[string]any{
		"ok":           true,
		"known":        true,
		"telegram_id":  u.TelegramID,
		"can_approve":  staffCanApprove(u),
		"prefixes":     staffPrefixes(u),
		"default_dept": staffDefaultDept(u),
		"user": map[string]any{
			"id":     u.ID,
			"name":   u.Name,
			"role":   u.Role,
			"desks":  u.Desks,
			"office": canOffice(&u),
		},
		"projects": bets,
	})
}
