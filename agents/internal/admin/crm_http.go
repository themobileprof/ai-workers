package admin

import (
	"context"
	"net/http"
	"strconv"
	"strings"

	"github.com/samuel/ai-workers/agents/internal/contract"
)

func crmFlash(ok string) string {
	switch ok {
	case "saved":
		return "Contact saved. This does not send mail."
	case "logged":
		return "Logged. Nothing was sent."
	case "won":
		return "Marked won. They stay on the book for a later upsell."
	case "lost":
		return "Marked lost."
	case "nurture":
		return "Moved to nurture."
	default:
		return ""
	}
}

func (s *Server) canSeeCRM(u *User) bool {
	if u == nil {
		return false
	}
	return canOffice(u) || Handles(*u, "crm") || u.Role == "cofounder"
}

func (s *Server) requireCRM(w http.ResponseWriter, r *http.Request) *User {
	u := s.requireAdmin(w, r)
	if u == nil {
		return nil
	}
	if !s.canSeeCRM(u) {
		http.Redirect(w, r, afterLoginPath(*u), http.StatusSeeOther)
		return nil
	}
	return u
}

func filterCRMContacts(u *User, list []CRMContact, projects []Project) []CRMContact {
	if u == nil {
		return nil
	}
	if canOffice(u) || Handles(*u, "crm") {
		return list
	}
	seen := map[int64]bool{}
	for _, p := range projects {
		if canSeeBet(*u, p.Members) {
			seen[p.ID] = true
		}
	}
	var out []CRMContact
	for _, c := range list {
		if c.ProjectID > 0 && seen[c.ProjectID] {
			out = append(out, c)
		}
	}
	return out
}

func canSeeCRMContact(u *User, c CRMContact, projects []Project) bool {
	if u == nil {
		return false
	}
	if canOffice(u) || Handles(*u, "crm") {
		return true
	}
	if c.ProjectID == 0 {
		return false
	}
	for _, p := range projects {
		if p.ID == c.ProjectID && canSeeBet(*u, p.Members) {
			return true
		}
	}
	return false
}

func (s *Server) loadVisibleCRM(w http.ResponseWriter, r *http.Request, u *User) (CRMContact, bool) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.NotFound(w, r)
		return CRMContact{}, false
	}
	c, err := s.store.GetCRMContact(r.Context(), id)
	if err != nil {
		http.NotFound(w, r)
		return CRMContact{}, false
	}
	projects, _ := s.store.ListProjects(r.Context())
	if !canSeeCRMContact(u, c, projects) {
		http.Redirect(w, r, afterLoginPath(*u), http.StatusSeeOther)
		return CRMContact{}, false
	}
	return c, true
}

func (s *Server) crmGET(w http.ResponseWriter, r *http.Request) {
	u := s.requireCRM(w, r)
	if u == nil {
		return
	}
	s.render(w, "crm", s.crmView(r, u, "", crmFlash(r.URL.Query().Get("ok"))))
}

func (s *Server) crmView(r *http.Request, u *User, errMsg, flash string) pageData {
	list, err := s.store.ListCRMContacts(r.Context())
	projects, _ := s.store.ListProjects(r.Context())
	list = filterCRMContacts(u, list, projects)
	data := pageData{
		Title:    "CRM",
		Nav:      "crm",
		User:     u,
		Contacts: list,
		Projects: projects,
		Flash:    flash,
		Error:    errMsg,
	}
	if err != nil && data.Error == "" {
		data.Error = err.Error()
	}
	return data
}

func (s *Server) crmPOST(w http.ResponseWriter, r *http.Request) {
	if !sameOrigin(w, r) {
		return
	}
	u := s.requireCRM(w, r)
	if u == nil {
		return
	}
	_ = r.ParseForm()
	pid, _ := strconv.ParseInt(r.FormValue("project_id"), 10, 64)
	c, err := s.store.UpsertCRMContact(r.Context(), CRMContact{
		ProjectID:    pid,
		Name:         r.FormValue("name"),
		Company:      r.FormValue("company"),
		Email:        r.FormValue("email"),
		Phone:        r.FormValue("phone"),
		Stage:        r.FormValue("stage"),
		Offer:        r.FormValue("offer"),
		NextAction:   r.FormValue("next_action"),
		NextActionBy: strings.TrimSpace(r.FormValue("next_action_by")),
		Notes:        r.FormValue("notes"),
		Source:       "desk",
	})
	if err != nil {
		s.render(w, "crm", s.crmView(r, u, err.Error(), ""))
		return
	}
	if strings.TrimSpace(r.FormValue("notes")) != "" {
		_ = s.store.InsertCRMEvent(r.Context(), CRMEvent{
			ContactID: c.ID, Channel: "desk", Direction: "note",
			Summary: clipCRM(r.FormValue("notes"), 400), Body: r.FormValue("notes"),
		})
	}
	http.Redirect(w, r, adminNext(r, "/admin/crm/"+strconv.FormatInt(c.ID, 10)+"?ok=saved"), http.StatusSeeOther)
}

func (s *Server) crmPersonGET(w http.ResponseWriter, r *http.Request) {
	u := s.requireCRM(w, r)
	if u == nil {
		return
	}
	c, ok := s.loadVisibleCRM(w, r, u)
	if !ok {
		return
	}
	s.render(w, "crm", s.crmPersonView(r, u, c, "", crmFlash(r.URL.Query().Get("ok"))))
}

func (s *Server) crmPersonView(r *http.Request, u *User, c CRMContact, errMsg, flash string) pageData {
	events, err := s.store.ListCRMEvents(r.Context(), c.ID, 80)
	projects, _ := s.store.ListProjects(r.Context())
	data := pageData{
		Title:     c.Name,
		Nav:       "crm",
		User:      u,
		Contact:   &c,
		CRMEvents: events,
		Projects:  projects,
		Flash:     flash,
		Error:     errMsg,
	}
	if err != nil && data.Error == "" {
		data.Error = err.Error()
	}
	return data
}

func (s *Server) crmPersonPOST(w http.ResponseWriter, r *http.Request) {
	if !sameOrigin(w, r) {
		return
	}
	u := s.requireCRM(w, r)
	if u == nil {
		return
	}
	c, ok := s.loadVisibleCRM(w, r, u)
	if !ok {
		return
	}
	_ = r.ParseForm()
	pid, _ := strconv.ParseInt(r.FormValue("project_id"), 10, 64)
	saved, err := s.store.UpdateCRMContact(r.Context(), CRMContact{
		ID:           c.ID,
		ProjectID:    pid,
		Name:         r.FormValue("name"),
		Company:      r.FormValue("company"),
		Email:        r.FormValue("email"),
		Phone:        r.FormValue("phone"),
		Stage:        r.FormValue("stage"),
		Offer:        r.FormValue("offer"),
		NextAction:   r.FormValue("next_action"),
		NextActionBy: strings.TrimSpace(r.FormValue("next_action_by")),
		Notes:        r.FormValue("notes"),
	})
	if err != nil {
		s.render(w, "crm", s.crmPersonView(r, u, c, err.Error(), ""))
		return
	}
	http.Redirect(w, r, adminNext(r, "/admin/crm/"+strconv.FormatInt(saved.ID, 10)+"?ok=saved"), http.StatusSeeOther)
}

func (s *Server) crmLogPOST(w http.ResponseWriter, r *http.Request) {
	if !sameOrigin(w, r) {
		return
	}
	u := s.requireCRM(w, r)
	if u == nil {
		return
	}
	c, ok := s.loadVisibleCRM(w, r, u)
	if !ok {
		return
	}
	_ = r.ParseForm()
	body := strings.TrimSpace(r.FormValue("body"))
	summary := firstNonEmpty(strings.TrimSpace(r.FormValue("summary")), clipCRM(body, 400))
	if summary == "" && body == "" {
		s.render(w, "crm", s.crmPersonView(r, u, c, "Say what happened. This does not send anything.", ""))
		return
	}
	if err := s.store.InsertCRMEvent(r.Context(), CRMEvent{
		ContactID: c.ID,
		Channel:   r.FormValue("channel"),
		Direction: r.FormValue("direction"),
		Summary:   summary,
		Body:      body,
	}); err != nil {
		s.render(w, "crm", s.crmPersonView(r, u, c, err.Error(), ""))
		return
	}
	if body != "" {
		_, _ = s.store.UpsertCRMContact(r.Context(), CRMContact{
			Name: c.Name, Email: c.Email, Phone: c.Phone, Notes: body, Source: crmChannel(r.FormValue("channel")),
		})
	}
	http.Redirect(w, r, adminNext(r, "/admin/crm/"+strconv.FormatInt(c.ID, 10)+"?ok=logged"), http.StatusSeeOther)
}

func (s *Server) crmStamp(stage string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !sameOrigin(w, r) {
			return
		}
		u := s.requireCRM(w, r)
		if u == nil {
			return
		}
		c, ok := s.loadVisibleCRM(w, r, u)
		if !ok {
			return
		}
		if err := s.store.SetCRMStage(r.Context(), c.ID, stage); err != nil {
			s.render(w, "crm", s.crmPersonView(r, u, c, err.Error(), ""))
			return
		}
		http.Redirect(w, r, adminNext(r, "/admin/crm/"+strconv.FormatInt(c.ID, 10)+"?ok="+stage), http.StatusSeeOther)
	}
}

func (s *Server) applyCRMResponse(ctx context.Context, req contract.Request, resp contract.Response) {
	if s == nil || s.store == nil || resp.StructuredData == nil {
		return
	}
	sd := resp.StructuredData
	task := strings.ToLower(strVal(sd["task_type"]))
	if task == "noise" || task == "access_denied" {
		return
	}
	data := contract.ContextObject(req.ContextData)
	waPhone := strVal(data["from"])
	whatsappLead := strVal(data["channel"]) == "whatsapp" && len(normalizePhone(waPhone)) >= 8
	if !boolVal(sd["record_lead"]) && !whatsappLead && task != "file_thread" && task != "important" && task != "qualify_lead" && task != "update_record" && task != "follow_up" {
		return
	}
	email := firstNonEmpty(strVal(sd["email"]), strVal(data["from"]))
	if email != "" && !strings.Contains(email, "@") {
		email = ""
	}
	phone := strVal(sd["phone"])
	if phone == "" && strVal(data["channel"]) == "whatsapp" {
		phone = strVal(data["from"])
	}
	name := firstNonEmpty(firstNonEmpty(strVal(sd["contact_name"]), strVal(sd["name"])), strVal(data["profile_name"]))
	company := strVal(sd["company_name"])
	pid := int64Val(sd["project_id"])
	exist, _ := data["existing_contact"].(map[string]any)
	known := false
	if exist != nil {
		name = firstNonEmpty(name, firstNonEmpty(strVal(exist["contact_name"]), strVal(exist["name"])))
		company = firstNonEmpty(company, firstNonEmpty(strVal(exist["company_name"]), strVal(exist["company"])))
		email = firstNonEmpty(email, strVal(exist["email"]))
		phone = firstNonEmpty(phone, strVal(exist["phone"]))
		if pid == 0 {
			pid = int64Val(exist["project_id"])
		}
		if id := int64Val(exist["contact_id"]); id > 0 {
			if got, err := s.store.GetCRMContact(ctx, id); err == nil {
				known = true
				name = firstNonEmpty(name, got.Name)
				company = firstNonEmpty(company, got.Company)
				email = firstNonEmpty(email, got.Email)
				phone = firstNonEmpty(phone, got.Phone)
				if pid == 0 {
					pid = got.ProjectID
				}
			}
		}
	}
	name = crmLookupQuery(name)
	if hit, ok, err := s.store.LookupCRMContact(ctx, email, phone, name); err != nil {
		return
	} else if ok {
		known = true
		name = hit.Name
		company = firstNonEmpty(company, hit.Company)
		email = firstNonEmpty(email, hit.Email)
		phone = firstNonEmpty(phone, hit.Phone)
		if pid == 0 {
			pid = hit.ProjectID
		}
	}
	if task == "follow_up" && !known {
		return
	}
	if pid == 0 {
		if slug := firstNonEmpty(strVal(sd["project_slug"]), strVal(data["project_slug"])); slug != "" {
			if p, err := s.store.GetProjectBySlug(ctx, slug); err == nil {
				pid = p.ID
			}
		}
	}
	ch := crmChannel(firstNonEmpty(strVal(data["channel"]), "ops"))
	dir := "note"
	if task != "follow_up" && (ch == "email" || ch == "whatsapp") {
		dir = "in"
	}
	c, err := s.store.UpsertCRMContact(ctx, CRMContact{
		ProjectID:    pid,
		Name:         name,
		Company:      company,
		Email:        email,
		Phone:        phone,
		Stage:        strVal(sd["pipeline_stage"]),
		Offer:        firstNonEmpty(strVal(sd["offer"]), strVal(sd["product"])),
		NextAction:   strVal(sd["next_action"]),
		NextActionBy: strVal(sd["next_action_by"]),
		Notes:        strVal(sd["notes"]),
		Source:       ch,
	})
	if err != nil {
		return
	}
	sd["crm_contact_id"] = c.ID
	body := firstNonEmpty(strVal(data["correspondence"]), strVal(sd["notes"]))
	summary := firstNonEmpty(firstNonEmpty(strVal(sd["notes"]), clipCRM(strVal(resp.OutputText), 400)), firstNonEmpty(task, ch))
	if task == "follow_up" {
		summary = firstNonEmpty(strVal(sd["next_action"]), "Follow-up proposed")
		body = clipCRM(strVal(resp.OutputText), 2000)
	}
	_ = s.store.InsertCRMEvent(ctx, CRMEvent{
		ContactID: c.ID, Channel: ch, Direction: dir, Summary: summary, Body: body,
	})
}

func (s *Server) internalCRMContactsGET(w http.ResponseWriter, r *http.Request) {
	if !s.internalOK(r) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	email := r.URL.Query().Get("email")
	phone := r.URL.Query().Get("phone")
	q := crmLookupQuery(r.URL.Query().Get("q"))
	c, ok, err := s.store.LookupCRMContact(r.Context(), email, phone, q)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	if !ok {
		writeJSON(w, map[string]any{"ok": true, "known": false})
		return
	}
	events, _ := s.store.ListCRMEvents(r.Context(), c.ID, 8)
	timeline := make([]map[string]any, 0, len(events))
	for _, e := range events {
		timeline = append(timeline, map[string]any{
			"channel": e.Channel, "direction": e.Direction, "summary": e.Summary,
			"when": e.WhenLabel(),
		})
	}
	writeJSON(w, map[string]any{
		"ok": true, "known": true,
		"contact": map[string]any{
			"id": c.ID, "name": c.Name, "company": c.Company, "email": c.Email, "phone": c.Phone,
			"stage": c.Stage, "offer": c.Offer, "next_action": c.NextAction, "next_action_by": c.NextActionBy,
			"notes": c.Notes, "project_id": c.ProjectID, "project": c.ProjectName, "source": c.Source,
			"last_channel": c.LastChannel, "last_at": c.LastAt, "timeline": timeline,
		},
	})
}
