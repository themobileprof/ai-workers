package admin

import (
	"context"
	"net/http"
	"os"
	"regexp"
	"strings"

	"github.com/samuel/ai-workers/agents/internal/contract"
)

var (
	episodeEmailRe = regexp.MustCompile(`(?i)[a-z0-9._%+\-]+@[a-z0-9.\-]+\.[a-z]{2,}`)
	episodePhoneRe = regexp.MustCompile(`(?:\+?234|0)[\s\-()]*\d(?:[\s\-()]*\d){9}`)
)

var episodeDropKeys = map[string]bool{
	"from": true, "phone": true, "email": true, "chat_id": true, "thread": true,
	"student": true, "profile_name": true, "image_base64": true, "images": true,
	"telegram_file_id": true, "photo_file_id": true, "document_file_id": true,
	"actor_name": true, "filename": true,
}

var episodeKeepKeys = map[string]bool{
	"project_slug": true, "action": true, "task_type": true, "ref_kind": true,
	"looks": true, "current_step": true, "current_slide": true, "cannot_send": true,
	"cannot_spend": true, "job_title": true, "hypothesis": true, "mission": true,
	"one_liner": true, "card_hypothesis": true, "funnel_steps": true,
}

func skipEpisode(desk, source, channel, taskType string) bool {
	ch := strings.ToLower(strings.TrimSpace(channel))
	src := strings.ToLower(strings.TrimSpace(source))
	if ch == "whatsapp" || src == "whatsapp" {
		return true
	}
	if strings.HasPrefix(ch, "acad") || strings.HasPrefix(src, "acad") || ch == "academy" {
		return true
	}
	if strings.EqualFold(desk, "community") && src != "desk" {
		return true
	}
	switch strings.ToLower(strings.TrimSpace(taskType)) {
	case "ads_glance", "access_denied", "weekly_intro", "scholarship_nudge":
		return true
	}
	return false
}

func skipEpisodeChannel(data map[string]any) bool {
	ch := strings.ToLower(strVal(data["channel"]))
	if ch == "whatsapp" {
		return true
	}
	if strings.EqualFold(strVal(data["chat_kind"]), "group") {
		return true
	}
	if strings.HasPrefix(ch, "acad") {
		return true
	}
	return false
}

func redactEpisodeText(s string) string {
	s = episodeEmailRe.ReplaceAllString(s, "[email]")
	s = episodePhoneRe.ReplaceAllString(s, "[phone]")
	return strings.TrimSpace(s)
}

func formReason(r *http.Request) string {
	if r == nil {
		return ""
	}
	_ = r.ParseForm()
	return clipRunes(redactEpisodeText(r.FormValue("reason")), 200)
}

func episodeInput(data map[string]any) map[string]any {
	out := map[string]any{}
	for k, v := range data {
		if episodeDropKeys[k] {
			continue
		}
		if !episodeKeepKeys[k] {
			continue
		}
		out[k] = redactEpisodeValue(v)
	}
	return out
}

func episodeProposal(resp contract.Response) map[string]any {
	out := map[string]any{}
	sd := resp.StructuredData
	if sd == nil {
		sd = map[string]any{}
	}
	for _, k := range []string{"task_type", "action", "title", "cannot_send", "cannot_spend", "save_deck", "project_slug", "prompt_version"} {
		if v, ok := sd[k]; ok {
			out[k] = redactEpisodeValue(v)
		}
	}
	if t := clipRunes(redactEpisodeText(resp.OutputText), 400); t != "" {
		out["output"] = t
	}
	return out
}

func redactEpisodeValue(v any) any {
	switch t := v.(type) {
	case string:
		return redactEpisodeText(t)
	case []string:
		out := make([]string, 0, len(t))
		for _, s := range t {
			out = append(out, redactEpisodeText(s))
		}
		return out
	case []any:
		out := make([]any, 0, len(t))
		for _, item := range t {
			if s, ok := item.(string); ok {
				out = append(out, redactEpisodeText(s))
			}
		}
		return out
	default:
		return v
	}
}

func lessonLines(accepted, rejected []DeskEpisode) map[string]any {
	var acc, rej []string
	for _, e := range accepted {
		line := lessonLine(e)
		if line != "" {
			acc = append(acc, line)
		}
	}
	for _, e := range rejected {
		line := lessonLine(e)
		if line != "" {
			rej = append(rej, line)
		}
	}
	if len(acc) == 0 && len(rej) == 0 {
		return nil
	}
	out := map[string]any{}
	if len(acc) > 0 {
		out["accepted"] = acc
	}
	if len(rej) > 0 {
		out["turned_away"] = rej
	}
	return out
}

func lessonLine(e DeskEpisode) string {
	head := strings.TrimSpace(e.TaskType)
	if e.ProjectSlug != "" {
		if head != "" {
			head += " · " + e.ProjectSlug
		} else {
			head = e.ProjectSlug
		}
	}
	body := ""
	switch e.HumanAct {
	case "rejected":
		body = strings.TrimSpace(e.Reason)
	case "accepted", "edited_accepted":
		body = strings.TrimSpace(e.Title)
		if e.Reason != "" {
			body = e.Reason
		}
	}
	body = redactEpisodeText(body)
	if body == "" {
		return ""
	}
	if head == "" {
		return clipRunes(body, 160)
	}
	return clipRunes(head+": "+body, 160)
}

func (s *Server) attachEpisodeLessons(ctx context.Context, data map[string]any) {
	if s == nil || s.store == nil || data == nil || skipEpisodeChannel(data) {
		return
	}
	desk := strings.TrimSpace(strVal(data["department"]))
	if desk == "" {
		return
	}
	if d, ok := DeskByDepartment(desk); ok {
		desk = d.Slug
	}
	if desk == "community" {
		return
	}
	slug := firstNonEmpty(strVal(data["project_slug"]), nestedEpisodeSlug(data))
	acc, rej, err := s.store.ListEpisodeLessons(ctx, desk, slug, 3, 2)
	if err != nil {
		return
	}
	lessons := lessonLines(acc, rej)
	if lessons == nil {
		return
	}
	data["lessons"] = lessons
}

func nestedEpisodeSlug(data map[string]any) string {
	if p, ok := data["project"].(map[string]any); ok {
		return strVal(p["slug"])
	}
	return ""
}

func (s *Server) openStaffEpisode(ctx context.Context, job DeskJob, req contract.Request, resp contract.Response) {
	if s == nil || s.store == nil || (job.ID == 0 && job.RefID == 0) {
		return
	}
	data := contract.ContextObject(req.ContextData)
	sd := resp.StructuredData
	if sd == nil {
		sd = map[string]any{}
	}
	taskType := firstNonEmpty(strVal(sd["task_type"]), strVal(job.Payload["task_type"]))
	if skipEpisode(job.Desk, job.Source, strVal(data["channel"]), taskType) {
		return
	}
	slug := firstNonEmpty(strVal(sd["project_slug"]), firstNonEmpty(strVal(data["project_slug"]), nestedEpisodeSlug(data)))
	_, _ = s.store.InsertEpisode(ctx, DeskEpisode{
		Desk:        job.Desk,
		ProjectSlug: slug,
		TaskType:    taskType,
		Source:      job.Source,
		Title:       job.Title,
		PromptVer:   strVal(sd["prompt_version"]),
		Model:       strings.TrimSpace(os.Getenv("LLM_PROVIDER")),
		Input:       episodeInput(data),
		Proposal:    episodeProposal(resp),
		HumanAct:    "proposed",
		JobID:       job.ID,
		RefKind:     job.RefKind,
		RefID:       job.RefID,
	})
}

func (s *Server) stampStaffEpisode(ctx context.Context, job DeskJob, status, reason string) {
	if s == nil || s.store == nil {
		return
	}
	act := status
	if status == "accepted" {
		act = s.episodeActForJob(ctx, job)
	}
	if status == "rejected" {
		act = "rejected"
	}
	_ = s.store.StampEpisodes(ctx, job.ID, job.RefKind, job.RefID, act, reason)
}

func (s *Server) stampStaffEpisodeRef(ctx context.Context, refKind string, refID int64, status, reason string) {
	if s == nil || s.store == nil || refKind == "" || refID <= 0 {
		return
	}
	act := status
	if status == "approved" {
		act = "accepted"
	}
	_ = s.store.StampEpisodes(ctx, 0, refKind, refID, act, reason)
}

func (s *Server) episodeActForJob(ctx context.Context, job DeskJob) string {
	msgs, err := s.store.ListJobMessages(ctx, job.ID)
	if err != nil {
		return "accepted"
	}
	sawWorker := false
	for _, m := range msgs {
		if m.Worker() {
			sawWorker = true
			continue
		}
		if sawWorker && m.Handler() {
			return "edited_accepted"
		}
	}
	return "accepted"
}

func (s *Server) episodesGET(w http.ResponseWriter, r *http.Request) {
	u := s.requireAdmin(w, r)
	if u == nil {
		return
	}
	if !canOffice(u) {
		http.Redirect(w, r, "/admin/docs", http.StatusSeeOther)
		return
	}
	rows, err := s.store.ListEpisodes(r.Context(), 200)
	data := pageData{Title: "Episodes", Nav: "episodes", User: u, Episodes: rows, Docs: allDocs()}
	if err != nil {
		data.Error = err.Error()
	}
	s.render(w, "episodes", data)
}

func (s *Server) episodesMarkdown(w http.ResponseWriter, r *http.Request) {
	u := s.requireAdmin(w, r)
	if u == nil {
		return
	}
	if !canOffice(u) {
		http.Error(w, "owner or bdm only", http.StatusForbidden)
		return
	}
	rows, err := s.store.ListEpisodes(r.Context(), 200)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	w.Header().Set("Content-Type", "text/markdown; charset=utf-8")
	w.Header().Set("Content-Disposition", "inline; filename=\"episodes.md\"")
	_, _ = w.Write([]byte(ExportEpisodesMarkdown(rows)))
}

func ExportEpisodesMarkdown(rows []DeskEpisode) string {
	var b strings.Builder
	b.WriteString("# Staff episodes\n\n")
	b.WriteString("Labeled Ask → proposal → Accept or Turn away on staff desks. Academy Telegram and customer WhatsApp are not in this file. Fold a repeating Turn-away reason into `faq.md` or a Go checker. Do not paste phones or student names into git.\n\n")
	if len(rows) == 0 {
		b.WriteString("No episodes yet.\n")
		return b.String()
	}
	for _, e := range rows {
		b.WriteString("## ")
		b.WriteString(e.Desk)
		if e.ProjectSlug != "" {
			b.WriteString(" · ")
			b.WriteString(e.ProjectSlug)
		}
		if e.TaskType != "" {
			b.WriteString(" · ")
			b.WriteString(e.TaskType)
		}
		b.WriteString("\n\n")
		b.WriteString("- Act: ")
		b.WriteString(e.ActLabel())
		b.WriteString("\n")
		if e.Title != "" {
			b.WriteString("- Title: ")
			b.WriteString(redactEpisodeText(e.Title))
			b.WriteString("\n")
		}
		if e.Reason != "" {
			b.WriteString("- Reason: ")
			b.WriteString(redactEpisodeText(e.Reason))
			b.WriteString("\n")
		}
		if e.Outcome != "" {
			b.WriteString("- Outcome: ")
			b.WriteString(redactEpisodeText(e.Outcome))
			b.WriteString("\n")
		}
		b.WriteString("- When: ")
		b.WriteString(e.CreatedAt.Format("2006-01-02 15:04"))
		b.WriteString("\n\n")
	}
	return b.String()
}
