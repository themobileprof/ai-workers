package admin

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/samuel/ai-workers/agents/internal/contract"
	"github.com/samuel/ai-workers/agents/internal/departments/legal"
)

func (s *Server) WithLegal(fn PlaceFunc) {
	if s != nil {
		s.legal = fn
		s.WithWorker("legal", fn)
	}
}

func legalFlash(ok string) string {
	switch ok {
	case "drafted":
		return "Draft saved as proposed. Accept still belongs to a human. The worker cannot stamp approved or send it."
	case "legal_ok":
		return "Draft accepted. Copy from the desk — sending to the other party is not wired."
	case "legal_no":
		return "Draft rejected. Nothing was sent."
	default:
		return ""
	}
}

func (s *Server) legalGET(w http.ResponseWriter, r *http.Request) {
	u := s.requireDesk(w, r, "legal")
	if u == nil {
		return
	}
	drafts, err := s.store.ListLegalDrafts(r.Context(), 0)
	data := pageData{
		Title:          "Legal",
		Nav:            "legal",
		User:           u,
		Drafts:         drafts,
		LegalTemplates: legal.Specs(),
		CanLegal:       s.legal != nil,
		Flash:          legalFlash(r.URL.Query().Get("ok")),
	}
	if err != nil {
		data.Error = err.Error()
	}
	s.render(w, "legal", data)
}

func (s *Server) legalStamp(status string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !sameOrigin(w, r) {
			return
		}
		u := s.requireDesk(w, r, "legal")
		if u == nil {
			return
		}
		id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		if err := s.store.SetLegalStatus(r.Context(), id, status); err != nil {
			drafts, _ := s.store.ListLegalDrafts(r.Context(), 0)
			s.render(w, "legal", pageData{
				Title:          "Legal",
				Nav:            "legal",
				User:           u,
				Drafts:         drafts,
				LegalTemplates: legal.Specs(),
				CanLegal:       s.legal != nil,
				Error:          err.Error(),
			})
			return
		}
		mapped := "accepted"
		if status == "rejected" {
			mapped = "rejected"
		}
		_ = s.store.StampJobsByRef(r.Context(), "legal_draft", id, mapped)
		ok := "legal_ok"
		if status == "rejected" {
			ok = "legal_no"
		}
		http.Redirect(w, r, legalNext(r, "/admin/legal?ok="+ok), http.StatusSeeOther)
	}
}

func legalNext(r *http.Request, fallback string) string {
	_ = r.ParseForm()
	out := fallback
	if next := strings.TrimSpace(r.FormValue("next")); strings.HasPrefix(next, "/admin/") {
		out = next
		if strings.Contains(fallback, "ok=legal_no") {
			out = withQuery(out, "ok", "legal_no")
		} else if strings.Contains(fallback, "ok=legal_ok") {
			out = withQuery(out, "ok", "legal_ok")
		}
	}
	if strings.Contains(out, "/admin/projects/") {
		return withQuery(out, "tab", "legal")
	}
	if strings.Contains(fallback, "ok=legal_no") {
		return withQuery(out, "tab", "rejected")
	}
	if strings.Contains(fallback, "ok=legal_ok") {
		return withQuery(out, "tab", "accepted")
	}
	return out
}

func (s *Server) projectLegal(w http.ResponseWriter, r *http.Request) {
	if !sameOrigin(w, r) {
		return
	}
	u := s.requireDesk(w, r, "legal")
	if u == nil {
		return
	}
	p, ok := s.loadProject(w, r)
	if !ok {
		return
	}
	if s.legal == nil {
		roster, _ := s.store.ListUsers(r.Context())
		s.render(w, "projects", s.projectView(r.Context(), u, p, roster, "LLM is not configured on this process", ""))
		return
	}
	_ = r.ParseForm()
	action := strings.TrimSpace(r.FormValue("action"))
	template := strings.TrimSpace(r.FormValue("template"))
	feedback := strings.TrimSpace(r.FormValue("human_feedback"))
	extraName := strings.TrimSpace(r.FormValue("counterparty_name"))
	var cp User
	if uid, err := strconv.ParseInt(r.FormValue("counterparty_user_id"), 10, 64); err == nil && uid > 0 {
		if person, err := s.store.GetUser(r.Context(), uid); err == nil {
			cp = person
		}
	}
	company := "TheMobileProf Technologies"
	if settings, err := s.store.Settings(r.Context()); err == nil {
		if n := strings.TrimSpace(settings["company.name"]); n != "" {
			company = n
		}
	}
	req, err := legalRequest(p, company, cp, extraName, template, action, feedback)
	if err != nil {
		roster, _ := s.store.ListUsers(r.Context())
		s.render(w, "projects", s.projectView(r.Context(), u, p, roster, err.Error(), ""))
		return
	}
	resp, err := s.legal(r.Context(), req)
	if err != nil || resp.Status != contract.StatusSuccess {
		msg := resp.OutputText
		if msg == "" && err != nil {
			msg = err.Error()
		}
		roster, _ := s.store.ListUsers(r.Context())
		s.render(w, "projects", s.projectView(r.Context(), u, p, roster, "Legal did not save a draft: "+msg, ""))
		return
	}
	d, err := draftFromStructured(resp.StructuredData, p.ID, cp, extraName)
	if err != nil {
		roster, _ := s.store.ListUsers(r.Context())
		s.render(w, "projects", s.projectView(r.Context(), u, p, roster, "Worker returned an invalid draft (nothing filed): "+err.Error(), ""))
		return
	}
	saved, err := s.store.InsertLegalDraft(r.Context(), d)
	if err != nil {
		roster, _ := s.store.ListUsers(r.Context())
		s.render(w, "projects", s.projectView(r.Context(), u, p, roster, err.Error(), ""))
		return
	}
	s.linkJob(r.Context(), "legal", saved.Title, saved.Rationale, "legal_draft", saved.ID, map[string]any{"template": saved.Template})
	http.Redirect(w, r, withQuery(withQuery("/admin/projects/"+strconv.FormatInt(p.ID, 10), "ok", "drafted"), "tab", "legal"), http.StatusSeeOther)
}

func legalRequest(p Project, company string, cp User, extraName, template, action, feedback string) (contract.Request, error) {
	action = strings.ToLower(strings.TrimSpace(action))
	if action != "review" {
		action = "draft"
	}
	ctx := map[string]any{
		"action":         action,
		"channel":        "desk",
		"cannot_send":    true,
		"human_feedback": strings.TrimSpace(feedback),
		"company":        map[string]any{"name": company},
		"project": map[string]any{
			"id":        p.ID,
			"name":      p.Name,
			"slug":      p.Slug,
			"one_liner": p.OneLiner,
		},
	}
	if spec, ok := legal.Lookup(template); ok {
		ctx["template"] = spec.ID
		template = spec.ID
	} else {
		template = ""
	}
	name := strings.TrimSpace(extraName)
	if name == "" && cp.Name != "" {
		name = cp.Name
	}
	if name != "" {
		ctx["counterparty_name"] = name
	}
	if cp.ID > 0 {
		ctx["counterparty"] = map[string]any{
			"id":    cp.ID,
			"name":  cp.Name,
			"email": cp.Email,
			"role":  cp.Role,
		}
	}
	raw, err := json.Marshal(ctx)
	if err != nil {
		return contract.Request{}, err
	}
	var task string
	if action == "review" {
		task = "Flag predatory or missing clauses. Do not invent citations. Do not send."
		if feedback != "" {
			task += " Pasted text:\n" + feedback
		}
	} else {
		label := template
		if spec, ok := legal.Lookup(template); ok {
			label = spec.Label
		}
		task = "Draft a " + label + " for TheMobileProf Technologies"
		if name != "" {
			task += " and " + name
		}
		task += " on project " + p.Name + ". Fill the catalog skeleton. Leave blanks as [TO BE COMPLETED]. Do not send."
		if feedback != "" {
			task += " Human notes: " + feedback
		}
	}
	return contract.Request{TaskDescription: task, ContextData: raw}, nil
}

func (s *Server) internalLegalDraftsGET(w http.ResponseWriter, r *http.Request) {
	if !s.internalOK(r) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	var pid int64
	if raw := strings.TrimSpace(r.URL.Query().Get("project_id")); raw != "" {
		pid, _ = strconv.ParseInt(raw, 10, 64)
	}
	drafts, err := s.store.ListLegalDrafts(r.Context(), pid)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	out := make([]map[string]any, 0, len(drafts))
	for _, d := range drafts {
		out = append(out, legalDraftJSON(d))
	}
	writeJSON(w, map[string]any{"drafts": out})
}

func (s *Server) internalLegalDraftsPOST(w http.ResponseWriter, r *http.Request) {
	if !s.internalOK(r) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	var body map[string]any
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}
	sd := body
	if nested, ok := body["structured_data"].(map[string]any); ok {
		sd = nested
	}
	p, err := s.resolveLegalProject(r, body, sd)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	var cp User
	if uid := int64Val(firstNonEmptyAny(body["counterparty_user_id"], sd["counterparty_user_id"])); uid > 0 {
		if person, err := s.store.GetUser(r.Context(), uid); err == nil {
			cp = person
		}
	}
	extraName := firstNonEmpty(strVal(body["counterparty_name"]), strVal(sd["counterparty_name"]))
	d, err := draftFromStructured(sd, p.ID, cp, extraName)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	saved, err := s.store.InsertLegalDraft(r.Context(), d)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	s.linkJob(r.Context(), "legal", saved.Title, saved.Rationale, "legal_draft", saved.ID, map[string]any{"template": saved.Template})
	writeJSON(w, map[string]any{"ok": true, "committed": false, "status": saved.Status, "draft": legalDraftJSON(saved)})
}

func (s *Server) resolveLegalProject(r *http.Request, body, sd map[string]any) (Project, error) {
	if id := int64Val(firstNonEmptyAny(body["project_id"], sd["project_id"])); id > 0 {
		return s.store.GetProject(r.Context(), id)
	}
	slug := firstNonEmpty(strVal(body["project_slug"]), strVal(sd["project_slug"]))
	if slug == "" {
		if nested, ok := sd["project"].(map[string]any); ok {
			slug = strVal(nested["slug"])
			if id := int64Val(nested["id"]); id > 0 {
				return s.store.GetProject(r.Context(), id)
			}
		}
	}
	if slug == "" {
		return Project{}, errors.New("project is required")
	}
	return s.store.GetProjectBySlug(r.Context(), slug)
}

func legalDraftJSON(d LegalDraft) map[string]any {
	return map[string]any{
		"id":                   d.ID,
		"project_id":           d.ProjectID,
		"project_name":         d.ProjectName,
		"counterparty_user_id": d.CounterpartyUserID,
		"counterparty_name":    d.CounterpartyName,
		"template":             d.Template,
		"title":                d.Title,
		"body":                 d.Body,
		"flags":                d.Flags,
		"needs_human_lawyer":   d.NeedsLawyer,
		"status":               d.Status,
		"rationale":            d.Rationale,
		"created_at":           d.CreatedAt,
		"updated_at":           d.UpdatedAt,
	}
}

func int64Val(v any) int64 {
	switch t := v.(type) {
	case int64:
		return t
	case int:
		return int64(t)
	case float64:
		return int64(t)
	case json.Number:
		n, _ := t.Int64()
		return n
	case string:
		n, _ := strconv.ParseInt(strings.TrimSpace(t), 10, 64)
		return n
	default:
		return 0
	}
}

func firstNonEmptyAny(vals ...any) any {
	for _, v := range vals {
		if v != nil {
			return v
		}
	}
	return nil
}
