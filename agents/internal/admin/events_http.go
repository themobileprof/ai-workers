package admin

import (
	"net/http"
	"strconv"
	"strings"
)

func eventFlash(ok string) string {
	switch ok {
	case "dated":
		return "Date saved on this bet. Add Delivery tasks and click Delivered when it actually happened."
	case "week":
		return "Date saved and copied onto This week. Delivery still holds the tasks."
	case "delivered":
		return "Marked delivered."
	case "skipped":
		return "Skipped. The date stays on the card as history."
	case "event_task":
		return "Task filed on Delivery for this date."
	default:
		return ""
	}
}

func (s *Server) projectEventPOST(w http.ResponseWriter, r *http.Request) {
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
	ev, err := cleanEvent(r.FormValue("title"), r.FormValue("kind"), r.FormValue("starts_on"), r.FormValue("ends_on"), r.FormValue("venue"), r.FormValue("audience"), r.FormValue("outcome"))
	if err != nil {
		roster, _ := s.store.ListUsers(r.Context())
		s.render(w, "projects", s.projectView(r.Context(), u, p, roster, err.Error(), ""))
		return
	}
	ev.ProjectID = p.ID
	saved, err := s.store.InsertProjectEvent(r.Context(), ev)
	if err != nil {
		roster, _ := s.store.ListUsers(r.Context())
		s.render(w, "projects", s.projectView(r.Context(), u, p, roster, err.Error(), ""))
		return
	}
	okq := "dated"
	if r.FormValue("set_week") == "on" || r.FormValue("set_week") == "true" {
		if err := s.store.SetProjectNotes(r.Context(), p.ID, saved.ThisWeekLine()); err == nil {
			okq = "week"
		}
	}
	http.Redirect(w, r, projectURL(p.ID, "dates", okq), http.StatusSeeOther)
}

func (s *Server) projectEventStamp(status string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
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
		id, err := strconv.ParseInt(r.PathValue("eid"), 10, 64)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		if err := s.store.SetProjectEventStatus(r.Context(), p.ID, id, status); err != nil {
			roster, _ := s.store.ListUsers(r.Context())
			s.render(w, "projects", s.projectView(r.Context(), u, p, roster, err.Error(), ""))
			return
		}
		okq := "delivered"
		if status == "skipped" {
			okq = "skipped"
		}
		http.Redirect(w, r, projectURL(p.ID, "dates", okq), http.StatusSeeOther)
	}
}

func (s *Server) projectEventTaskPOST(w http.ResponseWriter, r *http.Request) {
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
	id, err := strconv.ParseInt(r.PathValue("eid"), 10, 64)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	ev, err := s.store.GetProjectEvent(r.Context(), p.ID, id)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	if !ev.Planned() {
		roster, _ := s.store.ListUsers(r.Context())
		s.render(w, "projects", s.projectView(r.Context(), u, p, roster, "that date is closed", ""))
		return
	}
	_ = r.ParseForm()
	title := strings.TrimSpace(r.FormValue("title"))
	if title == "" {
		title = ev.Title
	}
	eng := EngagementFor(p.Stage)
	open, _ := s.store.CountOpenWork(r.Context(), p.ID)
	if eng.MaxOpenTasks <= 0 || open >= eng.MaxOpenTasks {
		roster, _ := s.store.ListUsers(r.Context())
		s.render(w, "projects", s.projectView(r.Context(), u, p, roster, p.Name+" has no open task slots at this stage", ""))
		return
	}
	due := ev.EndsOn.Format("2006-01-02")
	if ev.EndsOn.IsZero() {
		due = ev.StartsOn.Format("2006-01-02")
	}
	_, err = s.store.InsertWorkItem(r.Context(), WorkItem{
		ProjectID: p.ID,
		Title:     title,
		Hours:     parsePace(r.FormValue("hours")),
		Due:       due,
		Kind:      "task",
	})
	if err != nil {
		roster, _ := s.store.ListUsers(r.Context())
		s.render(w, "projects", s.projectView(r.Context(), u, p, roster, err.Error(), ""))
		return
	}
	http.Redirect(w, r, projectURL(p.ID, "dates", "event_task"), http.StatusSeeOther)
}
