package admin

import (
	"context"
	"net/http"
	"strconv"
	"strings"

	"github.com/samuel/ai-workers/agents/internal/contract"
)

func grantFlash(ok string) string {
	switch ok {
	case "saved":
		return "Grant row saved. This does not submit an application."
	case "done":
		return "Marked done."
	case "dropped":
		return "Dropped. Nothing was sent."
	default:
		return ""
	}
}

func (s *Server) grantsGET(w http.ResponseWriter, r *http.Request) {
	u := s.requireOffice(w, r)
	if u == nil {
		return
	}
	s.render(w, "grants", s.grantsView(r, u, "", grantFlash(r.URL.Query().Get("ok"))))
}

func (s *Server) grantsView(r *http.Request, u *User, errMsg, flash string) pageData {
	list, err := s.store.ListGrants(r.Context())
	projects, _ := s.store.ListProjects(r.Context())
	data := pageData{
		Title:    "Grants",
		Nav:      "grants",
		User:     u,
		Grants:   list,
		Projects: projects,
		Flash:    flash,
		Error:    errMsg,
	}
	if err != nil && data.Error == "" {
		data.Error = err.Error()
	}
	return data
}

func (s *Server) grantsPOST(w http.ResponseWriter, r *http.Request) {
	if !sameOrigin(w, r) {
		return
	}
	u := s.requireOffice(w, r)
	if u == nil {
		return
	}
	_ = r.ParseForm()
	pid, _ := strconv.ParseInt(r.FormValue("project_id"), 10, 64)
	_, err := s.store.InsertGrant(r.Context(), Grant{
		ProjectID:  pid,
		Title:      r.FormValue("title"),
		Funder:     r.FormValue("funder"),
		Deadline:   strings.TrimSpace(r.FormValue("deadline")),
		Fit:        r.FormValue("fit"),
		NextAction: r.FormValue("next_action"),
		Notes:      r.FormValue("notes"),
	})
	if err != nil {
		s.render(w, "grants", s.grantsView(r, u, err.Error(), ""))
		return
	}
	http.Redirect(w, r, adminNext(r, "/admin/grants?ok=saved"), http.StatusSeeOther)
}

func (s *Server) grantStamp(status string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !sameOrigin(w, r) {
			return
		}
		u := s.requireOffice(w, r)
		if u == nil {
			return
		}
		id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		if err := s.store.SetGrantStatus(r.Context(), id, status); err != nil {
			s.render(w, "grants", s.grantsView(r, u, err.Error(), ""))
			return
		}
		http.Redirect(w, r, adminNext(r, "/admin/grants?ok="+status), http.StatusSeeOther)
	}
}

func adminNext(r *http.Request, fallback string) string {
	_ = r.ParseForm()
	next := strings.TrimSpace(r.FormValue("next"))
	if !strings.HasPrefix(next, "/admin/") {
		return fallback
	}
	if i := strings.Index(fallback, "ok="); i >= 0 {
		ok := fallback[i+3:]
		if j := strings.IndexAny(ok, "&?#"); j >= 0 {
			ok = ok[:j]
		}
		if ok != "" {
			return withQuery(next, "ok", ok)
		}
	}
	return next
}

func (s *Server) applyGrantResponse(ctx context.Context, req contract.Request, resp contract.Response) {
	if s == nil || s.store == nil || resp.StructuredData == nil {
		return
	}
	sd := resp.StructuredData
	if !boolVal(sd["save_grant"]) {
		return
	}
	title := firstNonEmpty(strVal(sd["title"]), strVal(sd["funder"]))
	if title == "" {
		return
	}
	data := contract.ContextObject(req.ContextData)
	pid := int64Val(sd["project_id"])
	if pid == 0 {
		if slug := firstNonEmpty(strVal(sd["project_slug"]), strVal(data["project_slug"])); slug != "" {
			if p, err := s.store.GetProjectBySlug(ctx, slug); err == nil {
				pid = p.ID
			}
		}
	}
	g, err := s.store.InsertGrant(ctx, Grant{
		ProjectID:  pid,
		Title:      title,
		Funder:     strVal(sd["funder"]),
		Deadline:   strVal(sd["deadline"]),
		Fit:        firstNonEmpty(strVal(sd["fit"]), strVal(sd["eligibility"])),
		NextAction: strVal(sd["next_action"]),
		Notes:      strVal(sd["notes"]),
	})
	if err != nil {
		return
	}
	sd["grant_id"] = g.ID
}
