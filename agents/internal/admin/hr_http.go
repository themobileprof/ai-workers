package admin

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/samuel/ai-workers/agents/internal/contract"
	"github.com/samuel/ai-workers/agents/internal/departments/hr"
)

func (s *Server) WithHR(fn PlaceFunc) {
	if s != nil {
		s.hr = fn
		s.WithWorker("hr", fn)
	}
}

func hrFlash(ok string) string {
	switch ok {
	case "role_drafted":
		return "JD saved as proposed. Accept opens the role. The clerk cannot hire."
	case "role_open":
		return "Role is open. Applications can match this JD."
	case "role_closed":
		return "Role closed. Nothing was emailed."
	case "app_filed":
		return "Application saved as proposed. Shortlist or turn away — the clerk cannot stamp it."
	case "app_yes":
		return "Shortlisted. Offer letter and payroll are not wired."
	case "app_no":
		return "Turned away on the desk. Nothing was emailed."
	default:
		return ""
	}
}

func (s *Server) hrGET(w http.ResponseWriter, r *http.Request) {
	u := s.requireDesk(w, r, "hr")
	if u == nil {
		return
	}
	s.render(w, "hr", s.hrView(r, u, "", hrFlash(r.URL.Query().Get("ok"))))
}

func (s *Server) hrView(r *http.Request, u *User, errMsg, flash string) pageData {
	roles, _ := s.store.ListHrRoles(r.Context())
	apps, _ := s.store.ListHrApplications(r.Context(), 0)
	projects, _ := s.store.ListProjects(r.Context())
	return pageData{
		Title:       "HR",
		Nav:         "hr",
		User:        u,
		HrRoles:     roles,
		HrApps:      apps,
		Projects:    projects,
		HrTemplates: hr.Specs(),
		CanHR:       s.hr != nil,
		Error:       errMsg,
		Flash:       flash,
	}
}

func (s *Server) hrRolePOST(w http.ResponseWriter, r *http.Request) {
	if !sameOrigin(w, r) {
		return
	}
	u := s.requireDesk(w, r, "hr")
	if u == nil {
		return
	}
	if s.hr == nil {
		s.render(w, "hr", s.hrView(r, u, "LLM is not configured on this process", ""))
		return
	}
	_ = r.ParseForm()
	pid, _ := strconv.ParseInt(r.FormValue("project_id"), 10, 64)
	title := strings.TrimSpace(r.FormValue("title"))
	template := strings.TrimSpace(r.FormValue("template"))
	notes := strings.TrimSpace(r.FormValue("human_feedback"))
	company := s.companyName(r)
	var project Project
	if pid > 0 {
		if p, err := s.store.GetProject(r.Context(), pid); err == nil {
			project = p
		}
	}
	req, err := hrRoleRequest(company, project, title, template, notes)
	if err != nil {
		s.render(w, "hr", s.hrView(r, u, err.Error(), ""))
		return
	}
	resp, err := s.hr(r.Context(), req)
	if err != nil || resp.Status != contract.StatusSuccess {
		msg := resp.OutputText
		if msg == "" && err != nil {
			msg = err.Error()
		}
		s.render(w, "hr", s.hrView(r, u, "HR did not save a JD: "+msg, ""))
		return
	}
	role, err := roleFromStructured(resp.StructuredData, pid)
	if err != nil {
		s.render(w, "hr", s.hrView(r, u, "Worker returned an invalid JD: "+err.Error(), ""))
		return
	}
	if title != "" {
		role.Title = title
	}
	saved, err := s.store.InsertHrRole(r.Context(), role)
	if err != nil {
		s.render(w, "hr", s.hrView(r, u, err.Error(), ""))
		return
	}
	s.linkJob(r.Context(), "hr", saved.Title, saved.JD, "hr_role", saved.ID, map[string]any{"template": saved.Template, "kind": saved.Kind})
	http.Redirect(w, r, "/admin/hr?ok=role_drafted", http.StatusSeeOther)
}

func (s *Server) hrRoleStamp(status string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !sameOrigin(w, r) {
			return
		}
		u := s.requireDesk(w, r, "hr")
		if u == nil {
			return
		}
		id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		if err := s.store.SetHrRoleStatus(r.Context(), id, status); err != nil {
			s.render(w, "hr", s.hrView(r, u, err.Error(), ""))
			return
		}
		mapped := "accepted"
		if status != "open" {
			mapped = "rejected"
		}
		_ = s.store.StampJobsByRef(r.Context(), "hr_role", id, mapped)
		ok := "role_open"
		if status != "open" {
			ok = "role_closed"
		}
		http.Redirect(w, r, "/admin/hr?ok="+ok, http.StatusSeeOther)
	}
}

func (s *Server) hrApplicationPOST(w http.ResponseWriter, r *http.Request) {
	if !sameOrigin(w, r) {
		return
	}
	u := s.requireDesk(w, r, "hr")
	if u == nil {
		return
	}
	if s.hr == nil {
		s.render(w, "hr", s.hrView(r, u, "LLM is not configured on this process", ""))
		return
	}
	_ = r.ParseForm()
	rid, _ := strconv.ParseInt(r.FormValue("role_id"), 10, 64)
	paste := strings.TrimSpace(r.FormValue("human_feedback"))
	name := strings.TrimSpace(r.FormValue("applicant_name"))
	email := strings.TrimSpace(r.FormValue("applicant_email"))
	open, _ := s.store.ListOpenHrRoles(r.Context())
	var role HrRole
	if rid > 0 {
		if got, err := s.store.GetHrRole(r.Context(), rid); err == nil {
			role = got
		}
	}
	req, err := hrScoreRequest(s.companyName(r), role, open, name, email, paste)
	if err != nil {
		s.render(w, "hr", s.hrView(r, u, err.Error(), ""))
		return
	}
	resp, err := s.hr(r.Context(), req)
	if err != nil || resp.Status != contract.StatusSuccess {
		msg := resp.OutputText
		if msg == "" && err != nil {
			msg = err.Error()
		}
		s.render(w, "hr", s.hrView(r, u, "HR did not file an application: "+msg, ""))
		return
	}
	if rid == 0 {
		if slug := strVal(resp.StructuredData["role_slug"]); slug != "" {
			if got, err := s.store.GetHrRoleBySlug(r.Context(), slug); err == nil {
				rid = got.ID
			}
		}
	}
	app, err := applicationFromStructured(resp.StructuredData, rid, "desk")
	if err != nil {
		s.render(w, "hr", s.hrView(r, u, "Worker returned an invalid application: "+err.Error(), ""))
		return
	}
	if name != "" {
		app.Name = name
	}
	if email != "" {
		app.Email = email
	}
	saved, err := s.store.InsertHrApplication(r.Context(), app)
	if err != nil {
		s.render(w, "hr", s.hrView(r, u, err.Error(), ""))
		return
	}
	s.linkJob(r.Context(), "hr", saved.Name, saved.Rationale, "hr_application", saved.ID, map[string]any{"recommendation": saved.Recommendation, "score": saved.Score})
	http.Redirect(w, r, "/admin/hr?ok=app_filed", http.StatusSeeOther)
}

func (s *Server) hrApplicationStamp(status string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !sameOrigin(w, r) {
			return
		}
		u := s.requireDesk(w, r, "hr")
		if u == nil {
			return
		}
		id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		if err := s.store.SetHrApplicationStatus(r.Context(), id, status); err != nil {
			s.render(w, "hr", s.hrView(r, u, err.Error(), ""))
			return
		}
		mapped := "accepted"
		if status == "rejected" {
			mapped = "rejected"
		}
		_ = s.store.StampJobsByRef(r.Context(), "hr_application", id, mapped)
		ok := "app_yes"
		if status == "rejected" {
			ok = "app_no"
		}
		http.Redirect(w, r, "/admin/hr?ok="+ok, http.StatusSeeOther)
	}
}

func (s *Server) companyName(r *http.Request) string {
	company := "TheMobileProf Technologies"
	if settings, err := s.store.Settings(r.Context()); err == nil {
		if n := strings.TrimSpace(settings["company.name"]); n != "" {
			company = n
		}
	}
	return company
}

func hrRoleRequest(company string, p Project, title, template, notes string) (contract.Request, error) {
	ctx := map[string]any{
		"action":      "draft_jd",
		"channel":     "desk",
		"cannot_hire": true,
		"cannot_send": true,
		"company":     map[string]any{"name": company},
	}
	if spec, ok := hr.Lookup(template); ok {
		ctx["template"] = spec.ID
		ctx["kind"] = spec.Kind
	}
	if title != "" {
		ctx["role_title"] = title
	}
	if p.ID > 0 {
		ctx["project"] = map[string]any{"id": p.ID, "name": p.Name, "slug": p.Slug, "one_liner": p.OneLiner}
		ctx["project_slug"] = p.Slug
	}
	raw, err := json.Marshal(ctx)
	if err != nil {
		return contract.Request{}, err
	}
	task := "Draft a job description for TheMobileProf Technologies"
	if title != "" {
		task += ": " + title
	}
	if p.Name != "" {
		task += " on project " + p.Name
	}
	task += ". Fill the catalog skeleton. Leave blanks as [TO BE COMPLETED]. Do not hire. Do not send."
	if notes != "" {
		task += " Human notes: " + notes
	}
	return contract.Request{TaskDescription: task, ContextData: raw}, nil
}

func hrScoreRequest(company string, role HrRole, open []HrRole, name, email, paste string) (contract.Request, error) {
	ctx := map[string]any{
		"action":      "ingest",
		"channel":     "desk",
		"cannot_hire": true,
		"cannot_send": true,
		"company":     map[string]any{"name": company},
		"open_roles":  openRolesJSON(open),
	}
	if name != "" {
		ctx["applicant_name"] = name
	}
	if email != "" {
		ctx["applicant_email"] = email
	}
	if role.ID > 0 {
		ctx["role"] = map[string]any{"id": role.ID, "slug": role.Slug, "title": role.Title, "kind": role.Kind, "jd": role.JD}
		ctx["role_slug"] = role.Slug
		ctx["template"] = role.Template
	}
	raw, err := json.Marshal(ctx)
	if err != nil {
		return contract.Request{}, err
	}
	task := "Parse and score this application against the open JD. Recommend shortlist or reject with reasons. Do not hire. Do not email."
	if name != "" {
		task += " Applicant name: " + name + "."
	}
	if paste != "" {
		task += "\n" + paste
	}
	return contract.Request{TaskDescription: task, ContextData: raw}, nil
}

func openRolesJSON(roles []HrRole) []map[string]any {
	out := make([]map[string]any, 0, len(roles))
	for _, r := range roles {
		out = append(out, map[string]any{
			"id": r.ID, "slug": r.Slug, "title": r.Title, "kind": r.Kind, "jd": r.JD, "project": r.ProjectName,
		})
	}
	return out
}

func (s *Server) internalHrRoles(w http.ResponseWriter, r *http.Request) {
	if !s.internalOK(r) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	roles, err := s.store.ListHrRoles(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	out := make([]map[string]any, 0, len(roles))
	for _, role := range roles {
		out = append(out, map[string]any{
			"id": role.ID, "project_id": role.ProjectID, "project_name": role.ProjectName,
			"slug": role.Slug, "title": role.Title, "kind": role.Kind, "template": role.Template,
			"jd": role.JD, "status": role.Status, "updated_at": role.UpdatedAt,
		})
	}
	writeJSON(w, map[string]any{"roles": out})
}

func (s *Server) internalHrApplicationsPOST(w http.ResponseWriter, r *http.Request) {
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
	roleID := int64Val(firstNonEmptyAny(body["role_id"], sd["role_id"]))
	if roleID == 0 {
		slug := firstNonEmpty(strVal(body["role_slug"]), strVal(sd["role_slug"]))
		if slug != "" {
			if role, err := s.store.GetHrRoleBySlug(r.Context(), slug); err == nil {
				roleID = role.ID
			}
		}
	}
	source := firstNonEmpty(strVal(body["source"]), "email")
	app, err := applicationFromStructured(sd, roleID, source)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	saved, err := s.store.InsertHrApplication(r.Context(), app)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	s.linkJob(r.Context(), "hr", saved.Name, saved.Rationale, "hr_application", saved.ID, map[string]any{"source": saved.Source, "recommendation": saved.Recommendation})
	writeJSON(w, map[string]any{"ok": true, "committed": false, "status": saved.Status, "application": hrAppJSON(saved)})
}

func hrAppJSON(a HrApplication) map[string]any {
	return map[string]any{
		"id": a.ID, "role_id": a.RoleID, "role_title": a.RoleTitle, "role_slug": a.RoleSlug,
		"name": a.Name, "email": a.Email, "phone": a.Phone, "source": a.Source,
		"score": a.Score, "recommendation": a.Recommendation, "reasons": a.Reasons,
		"status": a.Status, "rationale": a.Rationale, "created_at": a.CreatedAt,
	}
}
