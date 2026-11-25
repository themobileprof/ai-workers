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

	"github.com/samuel/ai-workers/agents/internal/acadchat"
	"github.com/samuel/ai-workers/agents/internal/contract"
	"github.com/samuel/ai-workers/agents/internal/opschat"
)

func (s *Server) academyTelegramPOST(w http.ResponseWriter, r *http.Request) {
	if s == nil || s.acadBot == nil {
		http.Error(w, "academy telegram is not configured", http.StatusServiceUnavailable)
		return
	}
	want := s.acadBot.Secret()
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
	go s.handleAcademyTelegram(raw)
}

func (s *Server) handleAcademyTelegram(raw []byte) {
	in, ok := acadchat.ParseUpdate(raw)
	if !ok {
		return
	}
	ctx := context.Background()
	if in.MigrateTo != "" {
		_ = s.store.MigrateTelegramChat(ctx, in.ChatID, in.MigrateTo)
	}
	if in.Join && in.Group {
		_ = s.store.BindLMSMandateChat(ctx, in.ChatID)
		if in.MigrateTo != "" {
			_ = s.store.BindLMSMandateChat(ctx, in.MigrateTo)
		}
	}
	reply, err := s.academyTurn(ctx, in)
	if err != nil {
		log.Printf("academy telegram: %v", err)
		reply = "I could not finish that. Which course are you on?"
	}
	if strings.TrimSpace(reply) == "" {
		return
	}
	if err := s.academySend(ctx, in.ChatID, reply); err != nil {
		log.Printf("academy telegram send: %v", err)
	}
}

func (s *Server) academySend(ctx context.Context, chatID, text string) error {
	if s == nil || s.acadBot == nil {
		return nil
	}
	err := s.acadBot.Send(ctx, chatID, text)
	to := opschat.TelegramMigrateTo(err)
	if to == "" || to == chatID {
		return err
	}
	_ = s.store.MigrateTelegramChat(ctx, chatID, to)
	return s.acadBot.Send(ctx, to, text)
}

// telegramDestAfterMigrate is the chat Telegram actually delivered to. A basic
// group id that was upgraded still sits on the lms mandate; sendMessage then
// returns migrate_to_chat_id for TheMobileProf Community. Posting that id and
// the supergroup id in one loop posted the same copy twice in Community.
func telegramDestAfterMigrate(from, migrateTo string) string {
	from = strings.TrimSpace(from)
	to := strings.TrimSpace(migrateTo)
	if to != "" && to != from {
		return to
	}
	return from
}

func (s *Server) academySendEach(ctx context.Context, ids []string, text string) error {
	if s == nil || s.acadBot == nil {
		return nil
	}
	sent := map[string]bool{}
	var sendErr error
	for _, id := range ids {
		id = strings.TrimSpace(id)
		if id == "" || sent[id] {
			continue
		}
		err := s.acadBot.Send(ctx, id, text)
		to := opschat.TelegramMigrateTo(err)
		dest := telegramDestAfterMigrate(id, to)
		if to != "" && to != id {
			_ = s.store.MigrateTelegramChat(ctx, id, to)
			if sent[dest] {
				sent[id] = true
				continue
			}
			err = s.acadBot.Send(ctx, dest, text)
		}
		if err != nil {
			log.Printf("academy telegram send: %v", err)
			sendErr = err
			continue
		}
		sent[dest] = true
		sent[id] = true
	}
	return sendErr
}

func (s *Server) academyTurn(ctx context.Context, in acadchat.Inbound) (string, error) {
	if text, ok := s.academyRewardTurn(ctx, in); ok {
		return text, nil
	}
	d := acadchat.Decide(in)
	switch d.Kind {
	case acadchat.KindSkip:
		return "", nil
	case acadchat.KindHelp, acadchat.KindRefuse:
		return d.Reply, nil
	case acadchat.KindStart, acadchat.KindInvite:
		return s.academyInviteTurn(ctx, in, d)
	case acadchat.KindOnboard:
		if in.FromBot {
			return "", nil
		}
		s.applyCommunityStudent(ctx, in.FromID, in.Username, in.Name, in.ChatID, "", contract.Response{})
		s.store.RecordEngagement(ctx, in.FromID, engageCommunity, 1, time.Now())
		return s.academyOnboardText(ctx, in), nil
	case acadchat.KindLink:
		_, err := s.store.UpsertCommunityStudent(ctx, CommunityStudent{
			TelegramID: in.FromID,
			Username:   in.Username,
			Name:       in.Name,
			Email:      d.Email,
			ChatID:     in.ChatID,
		})
		if err != nil {
			return "", err
		}
		req := s.academyRequest(ctx, in, d, CommunityMandate{}, nil)
		s.FileInboundJob(ctx, "community", req, contract.Response{
			Status:     contract.StatusSuccess,
			OutputText: "LMS email on file: " + d.Email,
			StructuredData: map[string]any{
				"task_type":           "billing",
				"speak":               true,
				"escalate_to_founder": true,
			},
		})
		return "Saved " + d.Email + ". I can quote the public LMS list price. A human admin will check whether that account is paid. Do not send card numbers here.", nil
	}

	m, _, _ := s.store.LookupCommunityMandate(ctx, "", in.ChatID)
	thread, _ := s.store.ListOpsChat(ctx, "acad:"+in.ChatID+":"+in.FromID, 8)
	req := s.academyRequest(ctx, in, d, m, thread)
	req = s.EnrichCommunity(ctx, req)
	resp, err := s.callDepartment(ctx, "community", req)
	if err != nil && strings.TrimSpace(resp.OutputText) == "" {
		return "", err
	}
	s.applyCommunityStudent(ctx, in.FromID, in.Username, in.Name, in.ChatID, "", resp)
	s.recordAcademyEngagement(ctx, in, resp)
	_ = s.store.AppendOpsChat(ctx, "acad:"+in.ChatID+":"+in.FromID, "you", d.Task)
	text := strings.TrimSpace(resp.OutputText)
	speak := true
	if resp.StructuredData != nil {
		if _, ok := resp.StructuredData["speak"]; ok {
			speak = boolVal(resp.StructuredData["speak"])
		}
	}
	if !speak && text == "" {
		return "", nil
	}
	_ = s.store.AppendOpsChat(ctx, "acad:"+in.ChatID+":"+in.FromID, "assistant", text)
	return text, nil
}

func (s *Server) recordAcademyEngagement(ctx context.Context, in acadchat.Inbound, resp contract.Response) {
	if s == nil || s.store == nil || in.FromBot || strings.TrimSpace(in.FromID) == "" {
		return
	}
	now := time.Now()
	if in.Group {
		s.store.RecordEngagement(ctx, in.FromID, engageCommunity, 1, now)
	}
	task := ""
	if resp.StructuredData != nil {
		task = strings.ToLower(strVal(resp.StructuredData["task_type"]))
	}
	switch task {
	case "lesson_help", "workshop_help", "progress":
		s.store.RecordEngagement(ctx, in.FromID, engageAcademy, 2, now)
	}
}

func (s *Server) academyInviteTurn(ctx context.Context, in acadchat.Inbound, d acadchat.Decision) (string, error) {
	st, err := s.store.UpsertCommunityStudent(ctx, CommunityStudent{
		TelegramID: in.FromID,
		Username:   in.Username,
		Name:       in.Name,
		ChatID:     in.ChatID,
	})
	if err != nil {
		return "", err
	}
	if d.Kind == acadchat.KindInvite {
		return s.inviteShareText(st), nil
	}
	who := displayName(st, in.Name)
	var invitedBy string
	if d.RefCode != "" {
		if ref, ok := s.store.ConfirmReferral(ctx, d.RefCode, in.FromID, who, time.Now()); ok {
			invitedBy = displayName(ref, "")
		}
	}
	lms := s.lmsRegisterURL(ctx)
	group := s.ensureGroupInvite(ctx, in.ChatID)
	var b strings.Builder
	if invitedBy != "" {
		b.WriteString(invitedBy + " sent you.\n\n")
	}
	b.WriteString("Welcome " + who + ".\n\n")
	b.WriteString("Create your Academy account: " + lms + "\n")
	if group != "" {
		b.WriteString("Telegram class: " + group + "\n")
	}
	b.WriteString("\nTo invite a friend, send /invite and share the link I give you.")
	return strings.TrimSpace(b.String()), nil
}

func (s *Server) inviteShareText(st CommunityStudent) string {
	url := s.personalJoinURL(st.ReferralCode)
	return "Your invite link:\n\n" + url + "\n\nSend that to a friend. They create their Academy account on that page, then join the Telegram class from there."
}

func (s *Server) ensureGroupInvite(ctx context.Context, hintChat string) string {
	if link := s.telegramInviteURL(ctx); link != "" {
		return link
	}
	chatID := strings.TrimSpace(hintChat)
	if !strings.HasPrefix(chatID, "-") {
		chatID = ""
		list, _ := s.store.ListCommunityMandates(ctx)
		for _, m := range list {
			if m.Active && m.TelegramChatID != "" {
				chatID = m.TelegramChatID
				break
			}
		}
	}
	if s.acadBot == nil || !strings.HasPrefix(chatID, "-") {
		return ""
	}
	link, err := s.acadBot.CreateInvite(ctx, chatID)
	if err != nil || link == "" {
		return ""
	}
	_ = s.store.SetSetting(ctx, "community.telegram_invite", link)
	return link
}

func (s *Server) academyOnboardText(_ context.Context, in acadchat.Inbound) string {
	who := strings.Join(in.JoinNames, ", ")
	if who == "" {
		who = strings.TrimSpace(in.Name)
	}
	if who == "" {
		who = "there"
	}
	return "Welcome " + who + ". Which course are you on?"
}

func (s *Server) academyRequest(_ context.Context, in acadchat.Inbound, d acadchat.Decision, m CommunityMandate, thread []OpsChatLine) contract.Request {
	kind := "dm"
	if in.Group {
		kind = "group"
	}
	data := map[string]any{
		"channel":          "telegram",
		"chat_kind":        kind,
		"department":       "community",
		"chat_id":          in.ChatID,
		"telegram_chat_id": in.ChatID,
		"from":             in.FromID,
		"username":         in.Username,
		"actor_name":       in.Name,
		"cannot_send":      true,
	}
	if acadchat.LooksLikePayment(in.Text) {
		data["billing_ask"] = true
		if in.Group {
			data["billing_in_group"] = true
		}
	}
	if m.ID > 0 {
		data["mandate"] = m.JSON()
	}
	if len(thread) > 0 {
		lines := make([]string, 0, len(thread))
		for _, t := range thread {
			lines = append(lines, t.Author+": "+t.Body)
		}
		data["thread"] = lines
	}
	raw, _ := json.Marshal(data)
	return contract.Request{TaskDescription: d.Task, ContextData: raw}
}
