package admin

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/samuel/ai-workers/agents/internal/contract"
	"github.com/samuel/ai-workers/agents/internal/departments/legal"
	"github.com/samuel/ai-workers/agents/internal/journeys"
)

func stageLabel(id string) string {
	if s, ok := stageByID(id); ok {
		return s.Label
	}
	return id
}

func gateLabel(journeyID, gateID string) string {
	if g, ok := journeys.LookupGate(journeyID, gateID); ok {
		return g.Label
	}
	return gateID
}

func splitPlacement(s string) (string, string) {
	s = strings.TrimSpace(s)
	j, g, ok := strings.Cut(s, "/")
	if !ok {
		return "", ""
	}
	return strings.TrimSpace(j), strings.TrimSpace(g)
}

func (s *Server) home(w http.ResponseWriter, r *http.Request) {
	u := s.requireAdmin(w, r)
	if u == nil {
		return
	}
	users, _ := s.store.ListUsers(r.Context())
	allow, _ := s.store.WhatsAppAccounts(r.Context())
	settings, _ := s.store.Settings(r.Context())
	projects, _ := s.store.ListProjects(r.Context())
	s.render(w, "home", pageData{
		Title:        "Board",
		Nav:          "home",
		User:         u,
		Users:        users,
		Allowlist:    allow,
		UserCount:    len(users),
		SettingCount: len(settings),
		Projects:     projects,
	})
}

func (s *Server) projectsGET(w http.ResponseWriter, r *http.Request) {
	u := s.requireAdmin(w, r)
	if u == nil {
		return
	}
	projects, err := s.store.ListProjects(r.Context())
	data := pageData{Title: "Projects", Nav: "projects", User: u, Projects: projects, Stages: projectStages(), Journeys: journeys.All()}
	if err != nil {
		data.Error = err.Error()
	}
	s.render(w, "projects", data)
}

func (s *Server) projectNewGET(w http.ResponseWriter, r *http.Request) {
	u := s.requireAdmin(w, r)
	if u == nil {
		return
	}
	s.render(w, "projects", pageData{
		Title:    "Add project",
		Nav:      "projects",
		User:     u,
		Adding:   true,
		Stages:   projectStages(),
		Seats:    projectSeats(),
		Journeys: journeys.All(),
	})
}

func (s *Server) projectEditGET(w http.ResponseWriter, r *http.Request) {
	u := s.requireAdmin(w, r)
	if u == nil {
		return
	}
	p, ok := s.loadProject(w, r)
	if !ok {
		return
	}
	roster, _ := s.store.ListUsers(r.Context())
	okq := r.URL.Query().Get("ok")
	flash := placementFlash(okq)
	if flash == "" {
		flash = legalFlash(okq)
	}
	s.render(w, "projects", s.projectView(r.Context(), u, p, roster, "", flash))
}

func (s *Server) projectsPOST(w http.ResponseWriter, r *http.Request) {
	if !sameOrigin(w, r) {
		return
	}
	u := s.requireAdmin(w, r)
	if u == nil {
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad form", http.StatusBadRequest)
		return
	}
	j, g := splitPlacement(r.FormValue("placement"))
	p, err := s.store.CreateProject(r.Context(), r.FormValue("name"), r.FormValue("slug"), r.FormValue("one_liner"), r.FormValue("stage"), r.FormValue("notes"), r.FormValue("url"), j, g)
	if err != nil {
		s.render(w, "projects", pageData{
			Title:    "Add project",
			Nav:      "projects",
			User:     u,
			Adding:   true,
			Stages:   projectStages(),
			Journeys: journeys.All(),
			Error:    err.Error(),
		})
		return
	}
	http.Redirect(w, r, "/admin/projects/"+strconv.FormatInt(p.ID, 10), http.StatusSeeOther)
}

func (s *Server) projectSave(w http.ResponseWriter, r *http.Request) {
	if !sameOrigin(w, r) {
		return
	}
	u := s.requireAdmin(w, r)
	if u == nil {
		return
	}
	p, ok := s.loadProject(w, r)
	if !ok {
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad form", http.StatusBadRequest)
		return
	}
	j, g := splitPlacement(r.FormValue("placement"))
	err := s.store.UpdateProject(r.Context(), p.ID, r.FormValue("name"), r.FormValue("slug"), r.FormValue("one_liner"), r.FormValue("stage"), r.FormValue("notes"), r.FormValue("url"), j, g)
	if err != nil {
		roster, _ := s.store.ListUsers(r.Context())
		s.render(w, "projects", s.projectView(r.Context(), u, p, roster, err.Error(), ""))
		return
	}
	http.Redirect(w, r, "/admin/projects/"+strconv.FormatInt(p.ID, 10), http.StatusSeeOther)
}

func (s *Server) projectDelete(w http.ResponseWriter, r *http.Request) {
	if !sameOrigin(w, r) {
		return
	}
	u := s.requireAdmin(w, r)
	if u == nil {
		return
	}
	p, ok := s.loadProject(w, r)
	if !ok {
		return
	}
	if err := s.store.DeleteProject(r.Context(), p.ID); err != nil {
		projects, _ := s.store.ListProjects(r.Context())
		s.render(w, "projects", pageData{Title: "Projects", Nav: "projects", User: u, Projects: projects, Stages: projectStages(), Error: err.Error()})
		return
	}
	http.Redirect(w, r, "/admin/projects", http.StatusSeeOther)
}

func (s *Server) projectMemberAdd(w http.ResponseWriter, r *http.Request) {
	if !sameOrigin(w, r) {
		return
	}
	u := s.requireAdmin(w, r)
	if u == nil {
		return
	}
	p, ok := s.loadProject(w, r)
	if !ok {
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad form", http.StatusBadRequest)
		return
	}
	uid, _ := strconv.ParseInt(r.FormValue("user_id"), 10, 64)
	if err := s.store.AddProjectMember(r.Context(), p.ID, uid, r.FormValue("seat")); err != nil {
		roster, _ := s.store.ListUsers(r.Context())
		fresh, _ := s.store.GetProject(r.Context(), p.ID)
		s.render(w, "projects", s.projectView(r.Context(), u, fresh, roster, err.Error(), ""))
		return
	}
	http.Redirect(w, r, "/admin/projects/"+strconv.FormatInt(p.ID, 10), http.StatusSeeOther)
}

func (s *Server) projectMemberRemove(w http.ResponseWriter, r *http.Request) {
	if !sameOrigin(w, r) {
		return
	}
	u := s.requireAdmin(w, r)
	if u == nil {
		return
	}
	p, ok := s.loadProject(w, r)
	if !ok {
		return
	}
	uid, err := strconv.ParseInt(r.PathValue("uid"), 10, 64)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	if err := s.store.RemoveProjectMember(r.Context(), p.ID, uid); err != nil {
		roster, _ := s.store.ListUsers(r.Context())
		s.render(w, "projects", s.projectView(r.Context(), u, p, roster, err.Error(), ""))
		return
	}
	http.Redirect(w, r, "/admin/projects/"+strconv.FormatInt(p.ID, 10), http.StatusSeeOther)
}

func (s *Server) loadProject(w http.ResponseWriter, r *http.Request) (Project, bool) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.NotFound(w, r)
		return Project{}, false
	}
	p, err := s.store.GetProject(r.Context(), id)
	if err != nil {
		http.NotFound(w, r)
		return Project{}, false
	}
	return p, true
}

func (s *Server) internalProjects(w http.ResponseWriter, r *http.Request) {
	if !s.internalOK(r) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	projects, err := s.store.ListProjects(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	out := make([]map[string]any, 0, len(projects))
	for _, p := range projects {
		members := make([]map[string]any, 0, len(p.Members))
		for _, m := range p.Members {
			members = append(members, map[string]any{
				"user_id": m.UserID,
				"name":    m.Name,
				"phone":   m.Phone,
				"email":   m.Email,
				"seat":    m.Seat,
			})
		}
		out = append(out, map[string]any{
			"id":         p.ID,
			"name":       p.Name,
			"slug":       p.Slug,
			"one_liner":  p.OneLiner,
			"stage":      p.Stage,
			"url":        p.URL,
			"notes":      p.Notes,
			"journey":    p.Journey,
			"gate":       p.Gate,
			"proposal":   p.Proposal,
			"members":    members,
			"updated_at": p.UpdatedAt,
		})
	}
	writeJSON(w, map[string]any{"projects": out})
}

func (s *Server) projectView(ctx context.Context, u *User, p Project, roster []User, errMsg, flash string) pageData {
	drafts, _ := s.store.ListLegalDrafts(ctx, p.ID)
	return pageData{
		Title:          "Amend " + p.Name,
		Nav:            "projects",
		User:           u,
		Project:        &p,
		Roster:         roster,
		Stages:         projectStages(),
		Seats:          projectSeats(),
		Journeys:       journeys.All(),
		CanPlace:       s.place != nil,
		CanLegal:       s.legal != nil,
		Drafts:         drafts,
		LegalTemplates: legal.Specs(),
		Error:          errMsg,
		Flash:          flash,
	}
}

func placementFlash(ok string) string {
	switch ok {
	case "placed":
		return "Proposal saved. It has not moved the stamp — Accept, Amend, or Reject."
	case "accepted":
		return "Stamp moved. The proposal is closed."
	case "rejected":
		return "Proposal dropped. Committed journey and gate are unchanged."
	case "amended":
		return "You wrote the stamp. The proposal is closed."
	default:
		return ""
	}
}

func (s *Server) projectPlace(w http.ResponseWriter, r *http.Request) {
	if !sameOrigin(w, r) {
		return
	}
	u := s.requireAdmin(w, r)
	if u == nil {
		return
	}
	p, ok := s.loadProject(w, r)
	if !ok {
		return
	}
	if s.place == nil {
		roster, _ := s.store.ListUsers(r.Context())
		s.render(w, "projects", s.projectView(r.Context(), u, p, roster, "LLM is not configured on this process", ""))
		return
	}
	_ = r.ParseForm()
	req, err := placementRequest(p, r.FormValue("human_feedback"))
	if err != nil {
		roster, _ := s.store.ListUsers(r.Context())
		s.render(w, "projects", s.projectView(r.Context(), u, p, roster, err.Error(), ""))
		return
	}
	resp, err := s.place(r.Context(), req)
	if err != nil || resp.Status != contract.StatusSuccess {
		msg := resp.OutputText
		if msg == "" && err != nil {
			msg = err.Error()
		}
		roster, _ := s.store.ListUsers(r.Context())
		s.render(w, "projects", s.projectView(r.Context(), u, p, roster, "Placement did not save a stamp: "+msg, ""))
		return
	}
	prop, err := proposalFromStructured(resp.StructuredData)
	if err != nil {
		roster, _ := s.store.ListUsers(r.Context())
		s.render(w, "projects", s.projectView(r.Context(), u, p, roster, "Worker returned an invalid proposal (stamp untouched): "+err.Error(), ""))
		return
	}
	if err := s.store.SetProposal(r.Context(), p.ID, prop); err != nil {
		roster, _ := s.store.ListUsers(r.Context())
		s.render(w, "projects", s.projectView(r.Context(), u, p, roster, err.Error(), ""))
		return
	}
	http.Redirect(w, r, "/admin/projects/"+strconv.FormatInt(p.ID, 10)+"?ok=placed", http.StatusSeeOther)
}

func (s *Server) projectProposalAccept(w http.ResponseWriter, r *http.Request) {
	s.proposalAction(w, r, "accepted", func(ctx context.Context, id int64) error {
		return s.store.AcceptProposal(ctx, id)
	})
}

func (s *Server) projectProposalReject(w http.ResponseWriter, r *http.Request) {
	s.proposalAction(w, r, "rejected", func(ctx context.Context, id int64) error {
		return s.store.ClearProposal(ctx, id)
	})
}

func (s *Server) projectProposalAmend(w http.ResponseWriter, r *http.Request) {
	if !sameOrigin(w, r) {
		return
	}
	u := s.requireAdmin(w, r)
	if u == nil {
		return
	}
	p, ok := s.loadProject(w, r)
	if !ok {
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad form", http.StatusBadRequest)
		return
	}
	j, g := splitPlacement(r.FormValue("placement"))
	if err := s.store.CommitPlacement(r.Context(), p.ID, j, g, r.FormValue("mission")); err != nil {
		roster, _ := s.store.ListUsers(r.Context())
		s.render(w, "projects", s.projectView(r.Context(), u, p, roster, err.Error(), ""))
		return
	}
	http.Redirect(w, r, "/admin/projects/"+strconv.FormatInt(p.ID, 10)+"?ok=amended", http.StatusSeeOther)
}

func (s *Server) proposalAction(w http.ResponseWriter, r *http.Request, ok string, fn func(context.Context, int64) error) {
	if !sameOrigin(w, r) {
		return
	}
	u := s.requireAdmin(w, r)
	if u == nil {
		return
	}
	p, loaded := s.loadProject(w, r)
	if !loaded {
		return
	}
	if err := fn(r.Context(), p.ID); err != nil {
		roster, _ := s.store.ListUsers(r.Context())
		s.render(w, "projects", s.projectView(r.Context(), u, p, roster, err.Error(), ""))
		return
	}
	http.Redirect(w, r, "/admin/projects/"+strconv.FormatInt(p.ID, 10)+"?ok="+ok, http.StatusSeeOther)
}

func placementRequest(p Project, feedback string) (contract.Request, error) {
	ctx := map[string]any{
		"action":          "place_on_journey",
		"channel":         "desk",
		"human_feedback":  strings.TrimSpace(feedback),
		"journeys_locked": true,
		"project": map[string]any{
			"id":                p.ID,
			"name":              p.Name,
			"slug":              p.Slug,
			"one_liner":         p.OneLiner,
			"url":               p.URL,
			"notes":             p.Notes,
			"stage":             p.Stage,
			"committed_journey": p.Journey,
			"committed_gate":    p.Gate,
		},
	}
	raw, err := json.Marshal(ctx)
	if err != nil {
		return contract.Request{}, err
	}
	task := "Place project " + p.Name + " on a catalog journey and gate. Do not write the committed stamp."
	if fb := strings.TrimSpace(feedback); fb != "" {
		task += " Human feedback: " + fb
	}
	return contract.Request{TaskDescription: task, ContextData: raw}, nil
}

func (s *Server) internalJourneys(w http.ResponseWriter, r *http.Request) {
	if !s.internalOK(r) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	writeJSON(w, map[string]any{"journeys": journeys.All()})
}

func (s *Server) internalProposal(w http.ResponseWriter, r *http.Request) {
	if !s.internalOK(r) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.NotFound(w, r)
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
	prop, err := proposalFromStructured(sd)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := s.store.SetProposal(r.Context(), id, prop); err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	writeJSON(w, map[string]any{"ok": true, "committed": false})
}
