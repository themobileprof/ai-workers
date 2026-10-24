package admin

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/samuel/ai-workers/agents/internal/llm"
)

func (s *Server) WithGuide(c llm.Completer) {
	if s != nil {
		s.guide = c
	}
}

func (s *Server) guideGET(w http.ResponseWriter, r *http.Request) {
	u := s.requireAdmin(w, r)
	if u == nil {
		return
	}
	if !canOffice(u) {
		http.Redirect(w, r, "/admin/docs", http.StatusSeeOther)
		return
	}
	asks, err := s.store.ListGuideAsks(r.Context(), 200)
	data := pageData{Title: "Asked", Nav: "guide", User: u, GuideAsks: asks, Docs: allDocs()}
	if err != nil {
		data.Error = err.Error()
	}
	s.render(w, "guide", data)
}

func (s *Server) guideAskedMarkdown(w http.ResponseWriter, r *http.Request) {
	u := s.requireAdmin(w, r)
	if u == nil {
		return
	}
	if !canOffice(u) {
		http.Error(w, "owner or bdm only", http.StatusForbidden)
		return
	}
	asks, err := s.store.ListGuideAsks(r.Context(), 200)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	w.Header().Set("Content-Type", "text/markdown; charset=utf-8")
	w.Header().Set("Content-Disposition", "inline; filename=\"asked.md\"")
	_, _ = w.Write([]byte(ExportGuideAsksMarkdown(asks)))
}

func (s *Server) guideAskPOST(w http.ResponseWriter, r *http.Request) {
	if !sameOrigin(w, r) {
		return
	}
	u := s.requireAdmin(w, r)
	if u == nil {
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 32<<10)
	var body struct {
		Question string        `json:"question"`
		Page     string        `json:"page"`
		History  []llm.Message `json:"history"`
		Draft    *GuideDraft   `json:"draft"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}
	q := strings.TrimSpace(body.Question)
	if q == "" {
		http.Error(w, "question is required", http.StatusBadRequest)
		return
	}
	if len(q) > 500 {
		q = q[:500]
	}
	page := strings.TrimSpace(body.Page)
	if page != "" && !strings.HasPrefix(page, "/admin") {
		page = ""
	}
	in := guideInput{
		User:     *u,
		Question: q,
		Page:     page,
		History:  body.History,
		Draft:    sanitizeIncomingDraft(body.Draft),
	}
	rep := s.answerGuide(r.Context(), in)
	if s.store != nil {
		_ = s.store.InsertGuideAsk(r.Context(), u.ID, redactGuideQuestion(q, in), rep.Answer, page, rep.Unsure)
	}
	writeJSON(w, rep)
}
