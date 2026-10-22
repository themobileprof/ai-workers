package admin

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/samuel/ai-workers/agents/internal/contract"
	"github.com/samuel/ai-workers/agents/internal/departments/files"
)

func filesFlash(ok string) string {
	switch ok {
	case "file_ok":
		return "Accepted. n8n will put it in Drive when the Shared Drive id is on Defaults."
	case "file_no":
		return "Turned away. Nothing was written to Drive."
	default:
		return ""
	}
}

func (s *Server) filesGET(w http.ResponseWriter, r *http.Request) {
	u := s.requireOffice(w, r)
	if u == nil {
		return
	}
	rows, err := s.store.ListFileFilings(r.Context(), 0, "")
	data := pageData{
		Title:   "Files",
		Nav:     "files",
		User:    u,
		Filings: rows,
		Flash:   filesFlash(r.URL.Query().Get("ok")),
	}
	if err != nil {
		data.Error = err.Error()
	}
	s.render(w, "files", data)
}

func (s *Server) filesStamp(status string) http.HandlerFunc {
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
		if err := s.store.SetFileFilingStatus(r.Context(), id, status); err != nil {
			rows, _ := s.store.ListFileFilings(r.Context(), 0, "")
			s.render(w, "files", pageData{Title: "Files", Nav: "files", User: u, Filings: rows, Error: err.Error()})
			return
		}
		ok := "file_ok"
		if status == "rejected" {
			ok = "file_no"
		}
		http.Redirect(w, r, filesNext(r, "/admin/files?ok="+ok), http.StatusSeeOther)
	}
}

func filesNext(r *http.Request, fallback string) string {
	_ = r.ParseForm()
	out := fallback
	if next := strings.TrimSpace(r.FormValue("next")); strings.HasPrefix(next, "/admin/") {
		out = next
		if strings.Contains(fallback, "ok=file_no") {
			out = withQuery(out, "ok", "file_no")
		} else if strings.Contains(fallback, "ok=file_ok") {
			out = withQuery(out, "ok", "file_ok")
		}
	}
	if strings.Contains(out, "/admin/projects/") {
		return withQuery(out, "tab", "files")
	}
	return out
}

func (s *Server) EnrichFiles(ctx context.Context, req contract.Request) contract.Request {
	if s == nil || s.store == nil {
		return req
	}
	data := contract.ContextObject(req.ContextData)
	projects, err := s.store.ListProjects(ctx)
	if err == nil {
		list := make([]map[string]any, 0, len(projects)+1)
		list = append(list, map[string]any{
			"slug": files.CompanySlug, "name": "TheMobileProf", "one_liner": "Office papers for the CAC company, not a bet.",
		})
		for _, p := range projects {
			list = append(list, map[string]any{"id": p.ID, "slug": p.Slug, "name": p.Name, "one_liner": p.OneLiner})
		}
		data["projects"] = list
	}
	company := []map[string]string{}
	bets := []map[string]string{}
	for _, f := range files.Folders() {
		row := map[string]string{"id": f.ID, "label": f.Label, "help": f.Help}
		company = append(company, row)
		if !f.Company {
			bets = append(bets, row)
		}
	}
	data["folders_company"] = company
	data["folders_project"] = bets
	data["cannot_file"] = true
	raw, _ := json.Marshal(data)
	req.ContextData = raw
	return req
}

func (s *Server) applyFilesResponse(ctx context.Context, req contract.Request, resp contract.Response) {
	if s == nil || s.store == nil || resp.StructuredData == nil {
		return
	}
	sd := resp.StructuredData
	if !boolVal(sd["save_filing"]) {
		return
	}
	data := contract.ContextObject(req.ContextData)
	d, err := filingFromStructured(sd, data)
	if err != nil {
		return
	}
	if d.Slug != files.CompanySlug {
		p, err := s.resolveFilesProject(ctx, data, sd)
		if err != nil {
			return
		}
		d.ProjectID = p.ID
		if d.Slug == "" {
			d.Slug = p.Slug
		}
	}
	saved, err := s.store.InsertFileFiling(ctx, d)
	if err != nil {
		return
	}
	if !boolVal(sd["unsure"]) {
		if err := s.store.SetFileFilingStatus(ctx, saved.ID, "accepted"); err == nil {
			saved, _ = s.store.GetFileFiling(ctx, saved.ID)
		}
	}
	sd["filing_id"] = saved.ID
	sd["path"] = saved.Path
	sd["status"] = saved.Status
}

func filingFromStructured(sd, data map[string]any) (FileFiling, error) {
	known := map[string]bool{files.CompanySlug: true}
	if raw, ok := data["projects"].([]any); ok {
		for _, item := range raw {
			m, ok := item.(map[string]any)
			if !ok {
				continue
			}
			if s := strings.ToLower(strings.TrimSpace(strVal(m["slug"]))); s != "" {
				known[s] = true
			}
		}
	}
	files.ApplyCatalog(sd, known)
	if !boolVal(sd["save_filing"]) {
		return FileFiling{}, errFilingUnsure
	}
	slug := strings.ToLower(strings.TrimSpace(strVal(sd["project_slug"])))
	folder := strings.ToLower(strings.TrimSpace(strVal(sd["folder"])))
	name := firstNonEmpty(strVal(sd["filename"]), strVal(data["filename"]))
	return FileFiling{
		Slug:           slug,
		Folder:         folder,
		Filename:       name,
		MIME:           firstNonEmpty(strVal(data["file_mime"]), firstNonEmpty(strVal(data["image_mime"]), strVal(data["mime"]))),
		TelegramFileID: firstNonEmpty(strVal(data["telegram_file_id"]), firstNonEmpty(strVal(data["document_file_id"]), strVal(data["photo_file_id"]))),
		Caption:        firstNonEmpty(strVal(data["caption"]), strVal(sd["rationale"])),
		Rationale:      strVal(sd["rationale"]),
	}, nil
}

var errFilingUnsure = errString("unsure")

type errString string

func (e errString) Error() string { return string(e) }

func (s *Server) resolveFilesProject(ctx context.Context, data, sd map[string]any) (Project, error) {
	if id := int64Val(firstNonEmptyAny(sd["project_id"], data["project_id"])); id > 0 {
		return s.store.GetProject(ctx, id)
	}
	slug := firstNonEmpty(strVal(sd["project_slug"]), strVal(data["project_slug"]))
	if nested, ok := sd["project"].(map[string]any); ok && slug == "" {
		slug = strVal(nested["slug"])
	}
	if nested, ok := data["project"].(map[string]any); ok && slug == "" {
		slug = strVal(nested["slug"])
	}
	if slug == "" || slug == files.CompanySlug {
		return Project{}, errFilingUnsure
	}
	return s.store.GetProjectBySlug(ctx, slug)
}

func (s *Server) internalFileFilingsGET(w http.ResponseWriter, r *http.Request) {
	if !s.internalOK(r) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	var pid int64
	if raw := strings.TrimSpace(r.URL.Query().Get("project_id")); raw != "" {
		pid, _ = strconv.ParseInt(raw, 10, 64)
	}
	status := strings.TrimSpace(r.URL.Query().Get("status"))
	rows, err := s.store.ListFileFilings(r.Context(), pid, status)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	out := make([]map[string]any, 0, len(rows))
	for _, d := range rows {
		out = append(out, fileFilingJSON(d))
	}
	writeJSON(w, map[string]any{"filings": out})
}

func (s *Server) internalFileFilingsPOST(w http.ResponseWriter, r *http.Request) {
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
	data := body
	if nested, ok := body["context_data"].(map[string]any); ok {
		data = nested
	}
	d, err := filingFromStructured(sd, data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if d.Slug != files.CompanySlug {
		p, err := s.resolveFilesProject(r.Context(), data, sd)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		d.ProjectID = p.ID
		d.Slug = p.Slug
	}
	saved, err := s.store.InsertFileFiling(r.Context(), d)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	writeJSON(w, map[string]any{"ok": true, "committed": false, "status": saved.Status, "filing": fileFilingJSON(saved)})
}

func (s *Server) internalFilePlacedPOST(w http.ResponseWriter, r *http.Request) {
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
	if err := s.store.MarkFileFiled(r.Context(), id, strVal(body["drive_file_id"]), strVal(body["drive_url"]), strVal(body["error"])); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	d, err := s.store.GetFileFiling(r.Context(), id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	writeJSON(w, map[string]any{"ok": true, "filing": fileFilingJSON(d)})
}

func fileFilingJSON(d FileFiling) map[string]any {
	return map[string]any{
		"id":               d.ID,
		"project_id":       d.ProjectID,
		"project_name":     d.ProjectName,
		"slug":             d.Slug,
		"folder":           d.Folder,
		"folder_label":     d.FolderLabel(),
		"filename":         d.Filename,
		"mime":             d.MIME,
		"telegram_file_id": d.TelegramFileID,
		"caption":          d.Caption,
		"path":             d.Path,
		"rationale":        d.Rationale,
		"status":           d.Status,
		"drive_file_id":    d.DriveFileID,
		"drive_url":        d.DriveURL,
		"place_error":      d.PlaceError,
	}
}
