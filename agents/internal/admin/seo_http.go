package admin

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/samuel/ai-workers/agents/internal/contract"
	"github.com/samuel/ai-workers/agents/internal/departments/seo"
)

func (s *Server) WithSEO(fn PlaceFunc) {
	if s != nil {
		s.seo = fn
		s.WithWorker("seo", fn)
	}
}

func seoFlash(ok string) string {
	switch ok {
	case "drafted":
		return "Brief saved as proposed. Accept still belongs to a human. The worker cannot publish."
	case "seo_ok":
		return "Brief accepted. Copy from the desk — publishing the live page is not wired."
	case "seo_no":
		return "Brief rejected. Nothing was published."
	default:
		return ""
	}
}

func (s *Server) seoGET(w http.ResponseWriter, r *http.Request) {
	u := s.requireDesk(w, r, "seo")
	if u == nil {
		return
	}
	briefs, err := s.store.ListSeoBriefs(r.Context(), 0)
	projects, _ := s.store.ListProjects(r.Context())
	data := pageData{
		Title:     "SEO",
		Nav:       "seo",
		User:      u,
		SeoBriefs: briefs,
		Projects:  projects,
		SeoKinds:  seo.Specs(),
		CanSEO:    s.seo != nil,
		Flash:     seoFlash(r.URL.Query().Get("ok")),
	}
	if err != nil {
		data.Error = err.Error()
	}
	s.render(w, "seo", data)
}

func (s *Server) seoStamp(status string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !sameOrigin(w, r) {
			return
		}
		u := s.requireDesk(w, r, "seo")
		if u == nil {
			return
		}
		id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		if err := s.store.SetSeoStatus(r.Context(), id, status); err != nil {
			briefs, _ := s.store.ListSeoBriefs(r.Context(), 0)
			projects, _ := s.store.ListProjects(r.Context())
			s.render(w, "seo", pageData{
				Title:     "SEO",
				Nav:       "seo",
				User:      u,
				SeoBriefs: briefs,
				Projects:  projects,
				SeoKinds:  seo.Specs(),
				CanSEO:    s.seo != nil,
				Error:     err.Error(),
			})
			return
		}
		mapped := "accepted"
		if status == "rejected" {
			mapped = "rejected"
		}
		_ = s.store.StampJobsByRef(r.Context(), "seo_brief", id, mapped)
		ok := "seo_ok"
		if status == "rejected" {
			ok = "seo_no"
		}
		http.Redirect(w, r, seoNext(r, "/admin/seo?ok="+ok), http.StatusSeeOther)
	}
}

func seoNext(r *http.Request, fallback string) string {
	_ = r.ParseForm()
	out := fallback
	if next := strings.TrimSpace(r.FormValue("next")); strings.HasPrefix(next, "/admin/") {
		out = next
		if strings.Contains(fallback, "ok=seo_no") {
			out = withQuery(out, "ok", "seo_no")
		} else if strings.Contains(fallback, "ok=seo_ok") {
			out = withQuery(out, "ok", "seo_ok")
		}
	}
	if strings.Contains(out, "/admin/projects/") {
		return withQuery(out, "tab", "seo")
	}
	if strings.Contains(fallback, "ok=seo_no") {
		return withQuery(out, "tab", "rejected")
	}
	if strings.Contains(fallback, "ok=seo_ok") {
		return withQuery(out, "tab", "accepted")
	}
	return out
}

func (s *Server) seoAskPOST(w http.ResponseWriter, r *http.Request) {
	if !sameOrigin(w, r) {
		return
	}
	u := s.requireDesk(w, r, "seo")
	if u == nil {
		return
	}
	if s.seo == nil {
		briefs, _ := s.store.ListSeoBriefs(r.Context(), 0)
		projects, _ := s.store.ListProjects(r.Context())
		s.render(w, "seo", pageData{Title: "SEO", Nav: "seo", User: u, SeoBriefs: briefs, Projects: projects, SeoKinds: seo.Specs(), Error: "SEO worker is not wired"})
		return
	}
	_ = r.ParseForm()
	pid, _ := strconv.ParseInt(r.FormValue("project_id"), 10, 64)
	p, err := s.store.GetProject(r.Context(), pid)
	if err != nil {
		briefs, _ := s.store.ListSeoBriefs(r.Context(), 0)
		projects, _ := s.store.ListProjects(r.Context())
		s.render(w, "seo", pageData{Title: "SEO", Nav: "seo", User: u, SeoBriefs: briefs, Projects: projects, SeoKinds: seo.Specs(), CanSEO: true, Error: "pick a project"})
		return
	}
	saved, err := s.runSeoAsk(r.Context(), p, r.FormValue("action"), r.FormValue("url"), r.FormValue("keyword"), r.FormValue("human_feedback"))
	if err != nil {
		briefs, _ := s.store.ListSeoBriefs(r.Context(), 0)
		projects, _ := s.store.ListProjects(r.Context())
		s.render(w, "seo", pageData{Title: "SEO", Nav: "seo", User: u, SeoBriefs: briefs, Projects: projects, SeoKinds: seo.Specs(), CanSEO: true, Error: err.Error()})
		return
	}
	s.linkJob(r.Context(), "seo", saved.Title, saved.Rationale, "seo_brief", saved.ID, map[string]any{"kind": saved.Kind, "url": saved.URL})
	http.Redirect(w, r, "/admin/seo?ok=drafted", http.StatusSeeOther)
}

func (s *Server) projectSEO(w http.ResponseWriter, r *http.Request) {
	if !sameOrigin(w, r) {
		return
	}
	u := s.requireDesk(w, r, "seo")
	if u == nil {
		return
	}
	p, ok := s.loadProject(w, r)
	if !ok {
		return
	}
	if s.seo == nil {
		roster, _ := s.store.ListUsers(r.Context())
		s.render(w, "projects", s.projectView(r.Context(), u, p, roster, "LLM is not configured on this process", ""))
		return
	}
	_ = r.ParseForm()
	saved, err := s.runSeoAsk(r.Context(), p, r.FormValue("action"), r.FormValue("url"), r.FormValue("keyword"), r.FormValue("human_feedback"))
	if err != nil {
		roster, _ := s.store.ListUsers(r.Context())
		s.render(w, "projects", s.projectView(r.Context(), u, p, roster, "SEO did not save a brief: "+err.Error(), ""))
		return
	}
	s.linkJob(r.Context(), "seo", saved.Title, saved.Rationale, "seo_brief", saved.ID, map[string]any{"kind": saved.Kind, "url": saved.URL})
	http.Redirect(w, r, withQuery(withQuery("/admin/projects/"+strconv.FormatInt(p.ID, 10), "ok", "drafted"), "tab", "seo"), http.StatusSeeOther)
}

func (s *Server) runSeoAsk(ctx context.Context, p Project, action, pageURL, keyword, feedback string) (SeoBrief, error) {
	req, err := seoRequest(p, action, pageURL, keyword, feedback)
	if err != nil {
		return SeoBrief{}, err
	}
	resp, err := s.seo(ctx, req)
	if err != nil || resp.Status != contract.StatusSuccess {
		msg := resp.OutputText
		if msg == "" && err != nil {
			msg = err.Error()
		}
		if msg == "" {
			msg = "worker failed"
		}
		return SeoBrief{}, errors.New(msg)
	}
	d, err := seoBriefFromStructured(resp.StructuredData, p.ID)
	if err != nil {
		return SeoBrief{}, err
	}
	if d.URL == "" {
		d.URL = strings.TrimSpace(pageURL)
	}
	if d.Keyword == "" {
		d.Keyword = strings.TrimSpace(keyword)
	}
	return s.store.InsertSeoBrief(ctx, d)
}

func seoRequest(p Project, action, pageURL, keyword, feedback string) (contract.Request, error) {
	action = strings.ToLower(strings.TrimSpace(action))
	if _, ok := seo.Lookup(action); !ok {
		action = "page"
	}
	pageURL = strings.TrimSpace(pageURL)
	if pageURL == "" {
		pageURL = strings.TrimSpace(p.URL)
	}
	ctx := map[string]any{
		"action":         action,
		"channel":        "desk",
		"cannot_publish": true,
		"cannot_send":    true,
		"url":            pageURL,
		"keyword":        strings.TrimSpace(keyword),
		"human_feedback": strings.TrimSpace(feedback),
		"project": map[string]any{
			"id":        p.ID,
			"name":      p.Name,
			"slug":      p.Slug,
			"one_liner": p.OneLiner,
			"url":       p.URL,
		},
	}
	raw, err := json.Marshal(ctx)
	if err != nil {
		return contract.Request{}, err
	}
	task := "Propose SEO for " + p.Name
	if pageURL != "" {
		task += " at " + pageURL
	}
	task += ". Do not publish. Do not invent rankings."
	if feedback != "" {
		task += " Human notes:\n" + feedback
	}
	return contract.Request{TaskDescription: task, ContextData: raw}, nil
}

func (s *Server) EnrichSEO(ctx context.Context, req contract.Request) contract.Request {
	if s == nil || s.store == nil {
		return req
	}
	data := contract.ContextObject(req.ContextData)
	slug := firstNonEmpty(strVal(data["project_slug"]), strVal(data["project"]))
	if nested, ok := data["project"].(map[string]any); ok && slug == "" {
		slug = strVal(nested["slug"])
	}
	if slug == "" {
		return req
	}
	p, err := s.store.GetProjectBySlug(ctx, slug)
	if err != nil {
		return req
	}
	data["project"] = map[string]any{
		"id": p.ID, "name": p.Name, "slug": p.Slug, "one_liner": p.OneLiner, "url": p.URL,
	}
	if strVal(data["url"]) == "" && p.URL != "" {
		data["url"] = p.URL
	}
	raw, _ := json.Marshal(data)
	req.ContextData = raw
	return req
}

func (s *Server) applySeoResponse(ctx context.Context, req contract.Request, resp contract.Response) {
	if s == nil || s.store == nil || resp.StructuredData == nil {
		return
	}
	if !boolVal(resp.StructuredData["save_brief"]) {
		return
	}
	data := contract.ContextObject(req.ContextData)
	p, err := s.resolveSeoProject(ctx, data, resp.StructuredData)
	if err != nil {
		return
	}
	d, err := seoBriefFromStructured(resp.StructuredData, p.ID)
	if err != nil {
		return
	}
	saved, err := s.store.InsertSeoBrief(ctx, d)
	if err != nil {
		return
	}
	resp.StructuredData["seo_brief_id"] = saved.ID
}

func (s *Server) resolveSeoProject(ctx context.Context, body, sd map[string]any) (Project, error) {
	if id := int64Val(firstNonEmptyAny(body["project_id"], sd["project_id"])); id > 0 {
		return s.store.GetProject(ctx, id)
	}
	slug := firstNonEmpty(strVal(body["project_slug"]), strVal(sd["project_slug"]))
	if slug == "" {
		if nested, ok := sd["project"].(map[string]any); ok {
			slug = strVal(nested["slug"])
			if id := int64Val(nested["id"]); id > 0 {
				return s.store.GetProject(ctx, id)
			}
		}
		if nested, ok := body["project"].(map[string]any); ok {
			slug = strVal(nested["slug"])
			if id := int64Val(nested["id"]); id > 0 {
				return s.store.GetProject(ctx, id)
			}
		}
	}
	if slug == "" {
		return Project{}, errors.New("project is required")
	}
	return s.store.GetProjectBySlug(ctx, slug)
}

func (s *Server) internalSeoBriefsGET(w http.ResponseWriter, r *http.Request) {
	if !s.internalOK(r) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	var pid int64
	if raw := strings.TrimSpace(r.URL.Query().Get("project_id")); raw != "" {
		pid, _ = strconv.ParseInt(raw, 10, 64)
	}
	list, err := s.store.ListSeoBriefs(r.Context(), pid)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	out := make([]map[string]any, 0, len(list))
	for _, d := range list {
		out = append(out, seoBriefJSON(d))
	}
	writeJSON(w, map[string]any{"briefs": out})
}

func (s *Server) internalSeoBriefsPOST(w http.ResponseWriter, r *http.Request) {
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
	p, err := s.resolveSeoProject(r.Context(), body, sd)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	d, err := seoBriefFromStructured(sd, p.ID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	saved, err := s.store.InsertSeoBrief(r.Context(), d)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	s.linkJob(r.Context(), "seo", saved.Title, saved.Rationale, "seo_brief", saved.ID, map[string]any{"kind": saved.Kind})
	writeJSON(w, map[string]any{"ok": true, "committed": false, "status": saved.Status, "brief": seoBriefJSON(saved)})
}

func seoBriefJSON(d SeoBrief) map[string]any {
	return map[string]any{
		"id":           d.ID,
		"project_id":   d.ProjectID,
		"project_name": d.ProjectName,
		"kind":         d.Kind,
		"url":          d.URL,
		"keyword":      d.Keyword,
		"title":        d.Title,
		"meta":         d.Meta,
		"h1":           d.H1,
		"body":         d.Body,
		"flags":        d.Flags,
		"status":       d.Status,
		"rationale":    d.Rationale,
		"created_at":   d.CreatedAt,
		"updated_at":   d.UpdatedAt,
	}
}
