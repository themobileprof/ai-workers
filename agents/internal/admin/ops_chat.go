package admin

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/samuel/ai-workers/agents/internal/contract"
	"github.com/samuel/ai-workers/agents/internal/opschat"
)

func (s *Server) opsTelegramPOST(w http.ResponseWriter, r *http.Request) {
	if s == nil || s.opsBot == nil {
		http.Error(w, "telegram is not configured", http.StatusServiceUnavailable)
		return
	}
	want := s.opsBot.Secret()
	got := strings.TrimSpace(r.Header.Get("X-Telegram-Bot-Api-Secret-Token"))
	if want == "" || subtle.ConstantTimeCompare([]byte(got), []byte(want)) != 1 {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	raw, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "bad body", http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"ok":true}`))
	go s.handleOpsTelegram(raw)
}

type opsOut struct {
	Text         string
	JobID        int64
	CallbackID   string
	CallbackNote string
	EditMsgID    int64
}

func (s *Server) handleOpsTelegram(raw []byte) {
	in, ok := opschat.ParseUpdate(raw)
	if !ok {
		return
	}
	ctx := context.Background()
	reply, err := s.opsTurn(ctx, in)
	if err != nil {
		log.Printf("ops telegram: %v", err)
		reply = "The worker did not answer. Try again, or prefix a desk (/today, /growth, /legal)."
	}
	if reply == "" {
		return
	}
	if err := s.opsBot.Send(ctx, in.ChatID, reply); err != nil {
		log.Printf("ops telegram send: %v", err)
	}
}

func (s *Server) opsTurn(ctx context.Context, in opschat.Inbound) (string, error) {
	u, known, err := s.store.LookupUserByTelegram(ctx, in.FromID)
	if err != nil {
		return "", err
	}
	if !known || !u.Active {
		return "This bot is the office. Your Telegram id is " + in.FromID + ". Paste it on People.", nil
	}

	current := s.store.OpsChatDesk(ctx, in.ChatID)
	d := opschat.Decide(in, current)
	switch d.Kind {
	case opschat.KindSkip:
		return "", nil
	case opschat.KindHelp, opschat.KindGreet:
		go func() {
			c, cancel := context.WithTimeout(context.Background(), 8*time.Second)
			defer cancel()
			s.rememberOpsChat(c, in.ChatID)
		}()
		return opschat.HelpText(u.Name, staffPrefixes(u)), nil
	case opschat.KindNote:
		return d.Reply, nil
	case opschat.KindOutbox:
		if !staffCanApprove(u) {
			return "Only owner and BDM can /approve or /stop a pending info@ draft.", nil
		}
		return s.runOutbox(ctx, d.Action, in.ChatID), nil
	}

	dept := d.Department
	if dept == "" {
		dept, err = opschat.Classify(ctx, s.guide, d.Task, in.ReplyTo)
		if err != nil {
			return "", err
		}
	}
	if !opschat.KnownDepartment(dept) {
		dept = staffDefaultDept(u)
	}
	if !staffMay(u, dept) {
		return "That desk is not on your People card. You may use: " + strings.Join(staffPrefixes(u), ", "), nil
	}

	_ = s.store.AppendOpsChat(ctx, in.ChatID, "you", displayTask(in, d.Task))
	thread, _ := s.store.ListOpsChat(ctx, in.ChatID, 12)
	req := s.opsRequest(ctx, u, in, d, dept, thread)
	resp, err := s.callDepartment(ctx, dept, req)
	if err != nil && strings.TrimSpace(resp.OutputText) == "" {
		return "", err
	}
	text := strings.TrimSpace(resp.OutputText)
	if text == "" {
		text = "I heard you. Chat never Accepts — that stays on the desk."
	}
	if audit := s.maybeBooksWrite(ctx, dept, resp); audit != "" {
		text = strings.TrimSpace(text + "\n\n" + audit)
	}
	if d.Action == "report" && in.HasDocument {
		if extra := s.fileAlongReport(ctx, u, in, d, thread); extra != "" {
			text = strings.TrimSpace(text + "\n\n" + extra)
		}
	}
	_ = s.store.AppendOpsChat(ctx, in.ChatID, "assistant", text)
	sticky := dept
	if d.Action == "report" {
		sticky = "delivery"
	}
	_ = s.store.SetOpsChatDesk(ctx, in.ChatID, u.ID, sticky)
	return text, nil
}

func (s *Server) fileAlongReport(ctx context.Context, u User, in opschat.Inbound, d opschat.Decision, thread []OpsChatLine) string {
	if s == nil || !in.HasDocument {
		return ""
	}
	fd := opschat.Decision{Kind: opschat.KindWork, Department: "files", Task: d.Task}
	req := s.opsRequest(ctx, u, in, fd, "files", thread)
	resp, err := s.callDepartment(ctx, "files", req)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(resp.OutputText)
}

func displayTask(in opschat.Inbound, task string) string {
	if strings.TrimSpace(in.Text) != "" {
		return in.Text
	}
	return task
}

func (s *Server) opsRequest(ctx context.Context, u User, in opschat.Inbound, d opschat.Decision, dept string, thread []OpsChatLine) contract.Request {
	data := map[string]any{
		"channel":     "telegram",
		"chat_kind":   "dm",
		"department":  dept,
		"chat_id":     in.ChatID,
		"from":        in.FromID,
		"actor_id":    u.ID,
		"actor_name":  u.Name,
		"actor_role":  u.Role,
		"cannot_send": true,
		"cannot_hire": true,
	}
	if in.ReplyTo != "" {
		data["reply_to"] = in.ReplyTo
	}
	if d.Action != "" {
		data["action"] = d.Action
	} else if dept == "delivery" {
		data["action"] = "report"
	}
	if in.PhotoFileID != "" {
		data["photo_file_id"] = in.PhotoFileID
		data["telegram_file_id"] = in.PhotoFileID
	}
	if in.DocumentFileID != "" {
		data["document_file_id"] = in.DocumentFileID
		data["telegram_file_id"] = in.DocumentFileID
		data["filename"] = in.Filename
		data["file_mime"] = in.FileMIME
	}
	if in.HasPhoto && s.opsBot != nil {
		if b64, mime, err := s.opsBot.Download(ctx, in.PhotoFileID); err == nil {
			data["image_base64"] = b64
			data["image_mime"] = mime
		}
	}
	if len(thread) > 0 {
		lines := make([]string, 0, len(thread))
		for _, m := range thread {
			lines = append(lines, m.Author+": "+m.Body)
		}
		data["thread"] = lines
	}
	s.attachEpisodeLessons(ctx, data)
	raw, _ := json.Marshal(data)
	return contract.Request{TaskDescription: d.Task, ContextData: raw}
}

func (s *Server) callDepartment(ctx context.Context, dept string, req contract.Request) (contract.Response, error) {
	if dept == "delivery" {
		req = s.EnrichDelivery(ctx, req)
	}
	if dept == "files" {
		req = s.EnrichFiles(ctx, req)
	}
	if dept == "product-dev" {
		req = s.EnrichProduct(ctx, req)
	}
	if dept == "growth" {
		req = s.EnrichGrowth(ctx, req)
	}
	if dept == "community" {
		req = s.EnrichCommunity(ctx, req)
	}
	fn := s.deptFunc(dept)
	if fn == nil {
		return contract.Fail("no worker for " + dept), nil
	}
	resp, err := fn(ctx, req)
	if err == nil {
		s.FileInboundJob(ctx, dept, req, resp)
	}
	return resp, err
}

func (s *Server) deptFunc(dept string) PlaceFunc {
	if s.workers != nil {
		if fn := s.workers[dept]; fn != nil {
			return fn
		}
	}
	if d, ok := DeskByDepartment(dept); ok {
		return s.workerFor(d)
	}
	if dept == "files" && s.workers != nil {
		return s.workers["files"]
	}
	if dept == "hr" {
		return s.hr
	}
	return nil
}
