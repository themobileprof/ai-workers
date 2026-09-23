package admin

import (
	"net/http"
	"strconv"
	"strings"
)

func communityFlash(ok string) string {
	switch ok {
	case "saved":
		return "Mandate saved. Add the WhatsApp group id from an n8n execution if the room is not matched yet."
	case "deleted":
		return "Mandate removed. The worker has no brief for that room."
	default:
		return ""
	}
}

func (s *Server) communityGET(w http.ResponseWriter, r *http.Request) {
	u := s.requireDesk(w, r, "community")
	if u == nil {
		return
	}
	list, err := s.store.ListCommunityMandates(r.Context())
	data := pageData{
		Title:    "Community",
		Nav:      "community",
		User:     u,
		Mandates: list,
		Flash:    communityFlash(r.URL.Query().Get("ok")),
	}
	if err != nil {
		data.Error = err.Error()
	}
	if id, err := strconv.ParseInt(r.URL.Query().Get("id"), 10, 64); err == nil && id > 0 {
		if m, err := s.store.GetCommunityMandate(r.Context(), id); err == nil {
			data.Mandate = &m
		}
	}
	s.render(w, "community", data)
}

func (s *Server) communityPOST(w http.ResponseWriter, r *http.Request) {
	if !sameOrigin(w, r) {
		return
	}
	u := s.requireDesk(w, r, "community")
	if u == nil {
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad form", http.StatusBadRequest)
		return
	}
	m := CommunityMandate{
		Slug:            strings.TrimSpace(r.FormValue("slug")),
		Title:           strings.TrimSpace(r.FormValue("title")),
		SiteURL:         strings.TrimSpace(r.FormValue("site_url")),
		Brief:           strings.TrimSpace(r.FormValue("brief")),
		Catalog:         strings.TrimSpace(r.FormValue("catalog")),
		WhatsAppGroupID: strings.TrimSpace(r.FormValue("whatsapp_group_id")),
		TelegramChatID:  strings.TrimSpace(r.FormValue("telegram_chat_id")),
		Active:          r.FormValue("active") != "false",
	}
	if id, err := strconv.ParseInt(r.FormValue("id"), 10, 64); err == nil {
		m.ID = id
	}
	if _, err := s.store.UpsertCommunityMandate(r.Context(), m); err != nil {
		list, _ := s.store.ListCommunityMandates(r.Context())
		s.render(w, "community", pageData{
			Title:    "Community",
			Nav:      "community",
			User:     u,
			Mandates: list,
			Mandate:  &m,
			Error:    err.Error(),
		})
		return
	}
	http.Redirect(w, r, "/admin/community?ok=saved", http.StatusSeeOther)
}

func (s *Server) communityDelete(w http.ResponseWriter, r *http.Request) {
	if !sameOrigin(w, r) {
		return
	}
	u := s.requireDesk(w, r, "community")
	if u == nil {
		return
	}
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	if err := s.store.DeleteCommunityMandate(r.Context(), id); err != nil {
		list, _ := s.store.ListCommunityMandates(r.Context())
		s.render(w, "community", pageData{Title: "Community", Nav: "community", User: u, Mandates: list, Error: err.Error()})
		return
	}
	http.Redirect(w, r, "/admin/community?ok=deleted", http.StatusSeeOther)
}

func (s *Server) internalCommunityMandate(w http.ResponseWriter, r *http.Request) {
	if !s.internalOK(r) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	groupID := strings.TrimSpace(r.URL.Query().Get("group_id"))
	telegramID := strings.TrimSpace(r.URL.Query().Get("telegram_chat_id"))
	m, ok, err := s.store.LookupCommunityMandate(r.Context(), groupID, telegramID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	if !ok {
		writeJSON(w, map[string]any{"found": false})
		return
	}
	writeJSON(w, map[string]any{"found": true, "mandate": m.JSON()})
}

func (s *Server) internalCommunityMandates(w http.ResponseWriter, r *http.Request) {
	if !s.internalOK(r) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	list, err := s.store.ListCommunityMandates(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	out := make([]map[string]any, 0, len(list))
	for _, m := range list {
		if m.Active {
			out = append(out, m.JSON())
		}
	}
	writeJSON(w, map[string]any{"mandates": out})
}
