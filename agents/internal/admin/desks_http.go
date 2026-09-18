package admin

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/samuel/ai-workers/agents/internal/contract"
)

func (s *Server) WithWorker(dept string, fn PlaceFunc) {
	if s == nil || fn == nil {
		return
	}
	if s.workers == nil {
		s.workers = map[string]PlaceFunc{}
	}
	s.workers[dept] = fn
}

func (s *Server) workerFor(desk Desk) PlaceFunc {
	if s.workers != nil {
		if fn := s.workers[desk.Department]; fn != nil {
			return fn
		}
	}
	switch desk.Slug {
	case "legal":
		return s.legal
	case "hr":
		return s.hr
	case "product-dev":
		return s.place
	default:
		return nil
	}
}

func (s *Server) requireDesk(w http.ResponseWriter, r *http.Request, slug string) *User {
	u := s.requireAdmin(w, r)
	if u == nil {
		return nil
	}
	if !Handles(*u, slug) {
		http.Error(w, "not assigned to this desk", http.StatusForbidden)
		return nil
	}
	return u
}

func (s *Server) requirePeople(w http.ResponseWriter, r *http.Request) *User {
	u := s.requireAdmin(w, r)
	if u == nil {
		return nil
	}
	if !isOwner(u) {
		http.Redirect(w, r, "/admin/", http.StatusSeeOther)
		return nil
	}
	return u
}

func (s *Server) desksIndex(w http.ResponseWriter, r *http.Request) {
	u := s.requireAdmin(w, r)
	if u == nil {
		return
	}
	desks := visibleDesks(u)
	if len(desks) == 1 {
		http.Redirect(w, r, desks[0].Path(), http.StatusSeeOther)
		return
	}
	jobs, _ := s.store.ListDeskJobs(r.Context(), "", 20)
	var mine []DeskJob
	for _, j := range jobs {
		if Handles(*u, j.Desk) {
			mine = append(mine, j)
		}
	}
	s.render(w, "desks", pageData{
		Title: "Desks",
		Nav:   "desks",
		User:  u,
		Jobs:  mine,
		Flash: deskFlash(r.URL.Query().Get("ok")),
	})
}

func (s *Server) deskGET(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")
	desk, ok := LookupDesk(slug)
	if !ok {
		http.NotFound(w, r)
		return
	}
	u := s.requireDesk(w, r, slug)
	if u == nil {
		return
	}
	if id, _ := strconv.ParseInt(r.URL.Query().Get("job"), 10, 64); id > 0 {
		s.deskJobView(w, r, u, desk, id, "", deskFlash(r.URL.Query().Get("ok")))
		return
	}
	s.render(w, "desks", s.deskListView(r, u, desk, "", deskFlash(r.URL.Query().Get("ok"))))
}

func (s *Server) deskListView(r *http.Request, u *User, desk Desk, errMsg, flash string) pageData {
	jobs, _ := s.store.ListDeskJobs(r.Context(), desk.Slug, 40)
	return pageData{
		Title:  desk.Title,
		Nav:    "desk-" + desk.Slug,
		User:   u,
		Desk:   &desk,
		Jobs:   jobs,
		CanAsk: s.workerFor(desk) != nil,
		Error:  errMsg,
		Flash:  flash,
	}
}

func (s *Server) deskJobView(w http.ResponseWriter, r *http.Request, u *User, desk Desk, jobID int64, errMsg, flash string) {
	job, err := s.store.GetDeskJob(r.Context(), jobID)
	if err != nil || job.Desk != desk.Slug {
		http.NotFound(w, r)
		return
	}
	msgs, _ := s.store.ListJobMessages(r.Context(), job.ID)
	jobs, _ := s.store.ListDeskJobs(r.Context(), desk.Slug, 40)
	s.render(w, "desks", pageData{
		Title:    desk.Title,
		Nav:      "desk-" + desk.Slug,
		User:     u,
		Desk:     &desk,
		Jobs:     jobs,
		Job:      &job,
		Messages: msgs,
		CanAsk:   s.workerFor(desk) != nil,
		Error:    errMsg,
		Flash:    flash,
	})
}

func (s *Server) deskAskPOST(w http.ResponseWriter, r *http.Request) {
	if !sameOrigin(w, r) {
		return
	}
	desk, u := s.loadDeskUser(w, r)
	if u == nil {
		return
	}
	_ = r.ParseForm()
	task := strings.TrimSpace(r.FormValue("task"))
	if task == "" {
		s.render(w, "desks", s.deskListView(r, u, desk, "Ask needs a task", ""))
		return
	}
	fn := s.workerFor(desk)
	if fn == nil {
		s.render(w, "desks", s.deskListView(r, u, desk, "This worker is not wired on this process", ""))
		return
	}
	resp, err := fn(r.Context(), deskRequest(desk, task, nil, nil))
	if err != nil || resp.Status != contract.StatusSuccess {
		msg := resp.OutputText
		if msg == "" && err != nil {
			msg = err.Error()
		}
		s.render(w, "desks", s.deskListView(r, u, desk, "Worker did not start a job: "+msg, ""))
		return
	}
	job, err := s.store.InsertDeskJob(r.Context(), jobFromWorker(desk.Slug, "desk", task, resp.StructuredData, resp.OutputText))
	if err != nil {
		s.render(w, "desks", s.deskListView(r, u, desk, err.Error(), ""))
		return
	}
	_, _ = s.store.InsertJobMessage(r.Context(), JobMessage{JobID: job.ID, AuthorKind: "handler", UserID: u.ID, Body: task})
	if resp.OutputText != "" {
		_, _ = s.store.InsertJobMessage(r.Context(), JobMessage{JobID: job.ID, AuthorKind: "worker", Body: resp.OutputText})
	}
	http.Redirect(w, r, desk.Path()+"?job="+strconv.FormatInt(job.ID, 10)+"&ok=asked", http.StatusSeeOther)
}

func (s *Server) deskChatPOST(w http.ResponseWriter, r *http.Request) {
	if !sameOrigin(w, r) {
		return
	}
	desk, u := s.loadDeskUser(w, r)
	if u == nil {
		return
	}
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	_ = r.ParseForm()
	body := strings.TrimSpace(r.FormValue("body"))
	if body == "" {
		s.deskJobView(w, r, u, desk, id, "Message is empty", "")
		return
	}
	job, err := s.store.GetDeskJob(r.Context(), id)
	if err != nil || job.Desk != desk.Slug {
		http.NotFound(w, r)
		return
	}
	if _, err := s.store.InsertJobMessage(r.Context(), JobMessage{JobID: job.ID, AuthorKind: "handler", UserID: u.ID, Body: body}); err != nil {
		s.deskJobView(w, r, u, desk, id, err.Error(), "")
		return
	}
	if job.Proposed() {
		_ = s.store.SetDeskJobStatus(r.Context(), job.ID, "open")
	}
	fn := s.workerFor(desk)
	if fn == nil {
		http.Redirect(w, r, desk.Path()+"?job="+strconv.FormatInt(job.ID, 10)+"&ok=noted", http.StatusSeeOther)
		return
	}
	thread, _ := s.store.ListJobMessages(r.Context(), job.ID)
	resp, err := fn(r.Context(), deskRequest(desk, body, &job, thread))
	reply := resp.OutputText
	if reply == "" && err != nil {
		reply = err.Error()
	}
	if reply == "" {
		reply = "No reply from the worker."
	}
	_, _ = s.store.InsertJobMessage(r.Context(), JobMessage{JobID: job.ID, AuthorKind: "worker", Body: reply})
	http.Redirect(w, r, desk.Path()+"?job="+strconv.FormatInt(job.ID, 10)+"&ok=chat", http.StatusSeeOther)
}

func (s *Server) deskStamp(status string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !sameOrigin(w, r) {
			return
		}
		desk, u := s.loadDeskUser(w, r)
		if u == nil {
			return
		}
		id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		job, err := s.store.GetDeskJob(r.Context(), id)
		if err != nil || job.Desk != desk.Slug {
			http.NotFound(w, r)
			return
		}
		if err := s.applyJobStamp(r.Context(), job, status); err != nil {
			s.deskJobView(w, r, u, desk, id, err.Error(), "")
			return
		}
		ok := "job_ok"
		if status == "rejected" {
			ok = "job_no"
		}
		http.Redirect(w, r, desk.Path()+"?job="+strconv.FormatInt(job.ID, 10)+"&ok="+ok, http.StatusSeeOther)
	}
}

func (s *Server) applyJobStamp(ctx context.Context, job DeskJob, status string) error {
	if err := s.store.SetDeskJobStatus(ctx, job.ID, status); err != nil {
		return err
	}
	switch job.RefKind {
	case "legal_draft":
		mapped := "approved"
		if status == "rejected" {
			mapped = "rejected"
		}
		if status == "accepted" || status == "rejected" {
			_ = s.store.SetLegalStatus(ctx, job.RefID, mapped)
		}
	case "hr_role":
		mapped := "open"
		if status == "rejected" {
			mapped = "closed"
		}
		if status == "accepted" || status == "rejected" {
			_ = s.store.SetHrRoleStatus(ctx, job.RefID, mapped)
		}
	case "hr_application":
		mapped := "shortlisted"
		if status == "rejected" {
			mapped = "rejected"
		}
		if status == "accepted" || status == "rejected" {
			_ = s.store.SetHrApplicationStatus(ctx, job.RefID, mapped)
		}
	case "project":
		if status == "accepted" {
			_ = s.store.AcceptProposal(ctx, job.RefID)
		}
		if status == "rejected" {
			_ = s.store.ClearProposal(ctx, job.RefID)
		}
	}
	kind := "system"
	text := "Handler accepted this job. The worker did not stamp it."
	if status == "rejected" {
		text = "Handler turned this job away. Nothing was sent."
	}
	_, _ = s.store.InsertJobMessage(ctx, JobMessage{JobID: job.ID, AuthorKind: kind, Body: text})
	return nil
}

func (s *Server) loadDeskUser(w http.ResponseWriter, r *http.Request) (Desk, *User) {
	desk, ok := LookupDesk(r.PathValue("slug"))
	if !ok {
		http.NotFound(w, r)
		return Desk{}, nil
	}
	u := s.requireDesk(w, r, desk.Slug)
	if u == nil {
		return desk, nil
	}
	return desk, u
}

func (d Desk) Path() string { return "/admin/desks/" + d.Slug }

func deskRequest(desk Desk, task string, job *DeskJob, thread []JobMessage) contract.Request {
	ctx := map[string]any{
		"channel":     "desk",
		"department":  desk.Department,
		"cannot_send": true,
		"cannot_hire": true,
	}
	if job != nil {
		ctx["job_id"] = job.ID
		ctx["job_title"] = job.Title
		ctx["job_status"] = job.Status
		if job.Payload != nil {
			ctx["job"] = job.Payload
		}
	}
	if len(thread) > 0 {
		lines := make([]string, 0, len(thread))
		start := 0
		if len(thread) > 12 {
			start = len(thread) - 12
		}
		for _, m := range thread[start:] {
			who := m.AuthorKind
			if m.AuthorName != "" && m.Handler() {
				who = m.AuthorName
			}
			lines = append(lines, who+": "+m.Body)
		}
		ctx["thread"] = lines
	}
	raw, _ := json.Marshal(ctx)
	return contract.Request{TaskDescription: task, ContextData: raw}
}

func deskFlash(ok string) string {
	switch ok {
	case "asked":
		return "Worker started a proposed job. Chat here; Accept still belongs to you."
	case "chat":
		return "Worker replied. This chat does not stamp Accept."
	case "noted":
		return "Note saved. This worker is not wired on this process, so there is no reply."
	case "job_ok":
		return "Accepted. The worker did not send or hire."
	case "job_no":
		return "Turned away. Nothing was emailed."
	default:
		return ""
	}
}

func (s *Server) linkJob(ctx context.Context, desk, title, summary, kind string, refID int64, payload map[string]any) {
	if _, ok := LookupDesk(desk); !ok {
		return
	}
	j, err := s.store.InsertDeskJob(ctx, DeskJob{
		Desk: desk, Title: title, Status: "proposed", Source: "desk",
		Summary: summary, RefKind: kind, RefID: refID, Payload: payload,
	})
	if err != nil {
		return
	}
	if summary != "" {
		_, _ = s.store.InsertJobMessage(ctx, JobMessage{JobID: j.ID, AuthorKind: "worker", Body: summary})
	}
}

func (s *Server) FileInboundJob(ctx context.Context, dept string, req contract.Request, resp contract.Response) {
	if s == nil || s.store == nil || resp.Status != contract.StatusSuccess {
		return
	}
	if ctx == nil {
		ctx = context.Background()
	}
	data := contract.ContextObject(req.ContextData)
	sd := resp.StructuredData
	if sd == nil {
		sd = map[string]any{}
	}
	if !shouldFileInboundJob(dept, data, sd) {
		return
	}
	desk, ok := DeskByDepartment(dept)
	if !ok {
		return
	}
	source := firstNonEmpty(strVal(data["channel"]), "inbound")
	job, err := s.store.InsertDeskJob(ctx, jobFromWorker(desk.Slug, source, req.TaskDescription, sd, resp.OutputText))
	if err != nil {
		return
	}
	if resp.OutputText != "" {
		_, _ = s.store.InsertJobMessage(ctx, JobMessage{JobID: job.ID, AuthorKind: "worker", Body: resp.OutputText})
	}
}

func (s *Server) internalJobsPOST(w http.ResponseWriter, r *http.Request) {
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
	dept := firstNonEmpty(strVal(body["department"]), strVal(sd["department"]))
	desk, ok := DeskByDepartment(dept)
	if !ok {
		if d, found := LookupDesk(strVal(body["desk"])); found {
			desk, ok = d, true
		}
	}
	if !ok {
		http.Error(w, "desk is required", http.StatusBadRequest)
		return
	}
	source := firstNonEmpty(strVal(body["source"]), "n8n")
	task := firstNonEmpty(strVal(body["task_description"]), strVal(body["title"]))
	output := firstNonEmpty(strVal(body["output_text"]), strVal(sd["summary"]))
	job, err := s.store.InsertDeskJob(r.Context(), jobFromWorker(desk.Slug, source, task, sd, output))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if output != "" {
		_, _ = s.store.InsertJobMessage(r.Context(), JobMessage{JobID: job.ID, AuthorKind: "worker", Body: output})
	}
	writeJSON(w, map[string]any{"ok": true, "committed": false, "status": job.Status, "job_id": job.ID})
}
