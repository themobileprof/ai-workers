package admin

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/samuel/ai-workers/agents/internal/acadchat"
	"github.com/samuel/ai-workers/agents/internal/clubkonnect"
	"github.com/samuel/ai-workers/agents/internal/contract"
)

func rewardYes(s string) bool {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "yes", "y", "ok", "okay", "send", "go", "go ahead", "correct", "that's it", "thats it", "e correct", "na him":
		return true
	default:
		return false
	}
}

func rewardSeen(s string) bool {
	low := strings.ToLower(strings.TrimSpace(s))
	if low == "seen" || low == "got it" || low == "landed" || low == "received" || low == "i see it" || low == "it's in" || low == "its in" || low == "don see am" {
		return true
	}
	return strings.Contains(low, "seen") || strings.Contains(low, "got it")
}

func (s *Server) academyRewardTurn(ctx context.Context, in acadchat.Inbound) (string, bool) {
	if s == nil || s.store == nil || in.FromBot || in.Join {
		return "", false
	}
	rd, err := s.store.OpenRewardFor(ctx, in.FromID)
	if err != nil || rd.WinnerID == "" {
		return "", false
	}
	if in.Group {
		if ngPhoneLocal(in.Text) != "" {
			return displayName(CommunityStudent{Name: rd.WinnerName}, "Friend") + ", send that number in a private chat with me — not in the group.", true
		}
		return "", false
	}
	switch rd.Stage {
	case "await_phone":
		phone, net := parseNGPhoneNetwork(in.Text)
		if phone == "" {
			if strings.TrimSpace(rd.Phone) != "" && rewardYes(in.Text) {
				phone, net = rd.Phone, rd.Network
			} else {
				return s.askRewardPhone(rd), true
			}
		}
		if net == "" || ngNetworkCode(net) == "" {
			rd.Phone = phone
			rd.Stage = "await_phone"
			_ = s.store.UpsertRewardDay(ctx, rd)
			return "I have " + phone + ". Which network — MTN, Airtel, Glo, or 9mobile?", true
		}
		rd.Phone, rd.Network, rd.Stage = phone, net, "await_confirm"
		_ = s.store.UpsertRewardDay(ctx, rd)
		_, _ = s.store.UpsertCommunityStudent(ctx, CommunityStudent{TelegramID: in.FromID, Phone: phone, Network: net})
		return fmt.Sprintf("That looks like %s %s. Reply YES and I send ₦%d there, or send a different number.", ngNetworkName(net), phone, rd.Amount), true
	case "await_confirm":
		if !rewardYes(in.Text) {
			phone, net := parseNGPhoneNetwork(in.Text)
			if phone != "" {
				rd.Stage = "await_phone"
				_ = s.store.UpsertRewardDay(ctx, rd)
				in.Text = phone + " " + net
				return s.academyRewardTurn(ctx, in)
			}
			return fmt.Sprintf("Reply YES to send ₦%d to %s %s, or send a different number.", rd.Amount, ngNetworkName(rd.Network), rd.Phone), true
		}
		return s.sendRewardAirtime(ctx, rd), true
	case "sent":
		if !rewardSeen(in.Text) {
			return fmt.Sprintf("I already sent ₦%d to %s. Check the phone and reply SEEN when it lands.", rd.Amount, rd.Phone), true
		}
		rd.Stage = "verified"
		_ = s.store.UpsertRewardDay(ctx, rd)
		s.cheerRewardGroup(ctx, rd)
		return "Nice. See you in the group.", true
	}
	return "", false
}

func (s *Server) askRewardPhone(rd RewardDay) string {
	if rd.Phone != "" {
		return fmt.Sprintf("I have %s %s from last time. Reply YES to send ₦%d there, or send a new number.", ngNetworkName(rd.Network), rd.Phone, rd.Amount)
	}
	return fmt.Sprintf("I have ₦%d airtime for you. Reply with the phone number to credit (MTN, Airtel, Glo, or 9mobile).", rd.Amount)
}

func (s *Server) sendRewardAirtime(ctx context.Context, rd RewardDay) string {
	if s.store != nil {
		if sent, err := s.store.AirtimeSentOnLagosDay(ctx, rd.WinnerID, time.Now()); err == nil && sent {
			return "I already sent airtime today. Tomorrow is a new day."
		}
	}
	cli := clubkonnect.FromEnv()
	if cli == nil {
		s.fileRewardFail(ctx, rd, "airtime is not configured")
		return "I could not complete that just now. A human admin will pick it up."
	}
	reqID := "tmp" + rd.Day.In(lagosLocation()).Format("20060102") + rd.WinnerID
	if len(reqID) > 40 {
		reqID = reqID[:40]
	}
	rd.RequestID = reqID
	got, err := cli.Buy(ctx, ngNetworkCode(rd.Network), rd.Phone, strconv.Itoa(rd.Amount), reqID)
	if err != nil {
		log.Printf("academy airtime: %v", err)
		s.fileRewardFail(ctx, rd, "airtime send failed")
		return "I could not complete that just now. Send the number again later today, or a human admin will pick it up."
	}
	rd.OrderID = got.OrderID
	rd.ClubStatus = got.Status
	rd.Stage = "sent"
	if err := s.store.UpsertRewardDay(ctx, rd); err != nil {
		log.Printf("academy airtime save: %v", err)
	}
	return fmt.Sprintf("Sent ₦%d to %s. Check the phone and reply SEEN when it lands.", rd.Amount, rd.Phone)
}

func (s *Server) cheerRewardGroup(ctx context.Context, rd RewardDay) {
	if s.acadBot == nil || strings.TrimSpace(rd.GroupChatID) == "" {
		return
	}
	who := strings.TrimSpace(rd.WinnerName)
	if who == "" {
		who = "today's winner"
	}
	text := who + " has the airtime. Keep asking questions, talk in here, and share your invite — tomorrow is a new day."
	if err := s.academySend(ctx, rd.GroupChatID, text); err != nil {
		log.Printf("academy reward cheer: %v", err)
	}
}

func (s *Server) fileRewardFail(ctx context.Context, rd RewardDay, why string) {
	s.FileInboundJob(ctx, "community", contract.Request{
		TaskDescription: "Airtime reward needs a human. " + why,
	}, contract.Response{
		Status:     contract.StatusSuccess,
		OutputText: why + " winner=" + rd.WinnerID,
		StructuredData: map[string]any{
			"task_type":           "escalation",
			"speak":               false,
			"escalate_to_founder": true,
		},
	})
}

func (s *Server) dailyEngage(ctx context.Context) map[string]any {
	now := time.Now().In(lagosLocation())
	yesterday := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, lagosLocation()).AddDate(0, 0, -1)
	existing, _ := s.store.GetRewardDay(ctx, yesterday)
	if existing.AnnouncedAt != nil {
		return map[string]any{"ok": true, "skipped": true, "day": yesterday.Format("2006-01-02"), "stage": existing.Stage}
	}
	list, err := s.store.ListCommunityMandates(ctx)
	if err != nil {
		return map[string]any{"ok": false, "error": err.Error()}
	}
	var groups []string
	seen := map[string]bool{}
	for _, m := range list {
		id := strings.TrimSpace(m.TelegramChatID)
		if !m.Active || id == "" || seen[id] {
			continue
		}
		seen[id] = true
		groups = append(groups, id)
	}
	groupID := ""
	if len(groups) > 0 {
		groupID = groups[0]
	}
	if !existing.Day.IsZero() {
		if existing.GroupChatID == "" {
			existing.GroupChatID = groupID
		}
		return s.announceDailyGroup(ctx, existing, existing.Stage != "none" && existing.WinnerID != "", groups)
	}
	scores, err := s.store.RankEngagement(ctx, yesterday)
	if err != nil {
		return map[string]any{"ok": false, "error": err.Error()}
	}
	staff := map[string]bool{}
	if users, err := s.store.ListUsers(ctx); err == nil {
		for _, u := range users {
			if tid := normalizeTelegramID(u.TelegramID); tid != "" {
				staff[tid] = true
			}
		}
	}
	recent, _ := s.store.RecentAirtimeWinners(ctx, now.AddDate(0, 0, -7))
	winner, has := pickWinner(scores, staff, recent)
	amount := s.rewardNaira()
	rd := RewardDay{Day: yesterday, GroupChatID: groupID, Amount: amount, Stage: "none"}
	if has {
		st, _ := s.store.LookupCommunityStudent(ctx, winner.TelegramID)
		name := displayName(st, winner.Name)
		rd.WinnerID = winner.TelegramID
		rd.WinnerName = name
		rd.CommunityPts = winner.Community
		rd.AcademyPts = winner.Academy
		rd.ReferralPts = winner.Referral
		rd.Phone, rd.Network = st.Phone, st.Network
		rd.Stage = "await_phone"
	}
	if err := s.store.UpsertRewardDay(ctx, rd); err != nil {
		return map[string]any{"ok": false, "error": err.Error()}
	}
	return s.announceDailyGroup(ctx, rd, has, groups)
}

func (s *Server) announceDailyGroup(ctx context.Context, rd RewardDay, has bool, groups []string) map[string]any {
	out := map[string]any{"ok": true, "day": rd.Day.In(lagosLocation()).Format("2006-01-02"), "winner": rd.WinnerName, "stage": rd.Stage}
	groupText := s.dailyGroupText(ctx, rd, has)
	if len(groups) == 0 {
		out["ok"] = false
		out["error"] = "no Telegram class on the mandate"
		return out
	}
	if s.acadBot == nil {
		out["ok"] = false
		out["error"] = "academy telegram is not configured"
		return out
	}
	var sendErr error
	for _, id := range groups {
		if err := s.academySend(ctx, id, groupText); err != nil {
			log.Printf("academy daily group: %v", err)
			sendErr = err
		}
	}
	if sendErr != nil {
		out["ok"] = false
		out["error"] = sendErr.Error()
		return out
	}
	if err := s.store.MarkRewardAnnounced(ctx, rd.Day); err != nil {
		log.Printf("academy daily announced: %v", err)
	}
	if has {
		dm := s.askRewardPhone(rd)
		if err := s.acadBot.Send(ctx, rd.WinnerID, dm); err != nil {
			log.Printf("academy daily dm: %v", err)
			_ = s.academySend(ctx, groups[0], rd.WinnerName+", open a private chat with me and send a hello so I can send the airtime.")
		}
	}
	out["posted"] = true
	return out
}

func (s *Server) dailyGroupText(ctx context.Context, rd RewardDay, has bool) string {
	naira := fmt.Sprintf("₦%d", rd.Amount)
	if rd.Amount <= 0 {
		naira = "₦500"
	}
	invite := s.personalJoinURL("")
	how := "How to be in the running:\n" +
		"1. Ask a lesson or workshop question here (or in a private chat with me).\n" +
		"2. Talk in this class — share one thing you learned.\n" +
		"3. Bring a friend: type /invite and I send you a link. They create their Academy account on that page, then join this class from there."
	var b strings.Builder
	b.WriteString("Morning class.\n\n")
	if has {
		b.WriteString("Yesterday ")
		b.WriteString(rd.WinnerName)
		b.WriteString(" showed up most. ")
		b.WriteString(rd.WinnerName)
		b.WriteString(", I just wrote you privately for the ")
		b.WriteString(naira)
		b.WriteString(" airtime.\n\nEveryone else: ")
		b.WriteString(naira)
		b.WriteString(" is still on tomorrow. ")
		b.WriteString(how)
		b.WriteString("\n\nShare: ")
		b.WriteString(invite)
	} else {
		b.WriteString("I have ")
		b.WriteString(naira)
		b.WriteString(" airtime to give out. Nobody earned yesterday's card — it is still here today.\n\n")
		b.WriteString(how)
		b.WriteString("\n\nThe person who shows up most today gets ")
		b.WriteString(naira)
		b.WriteString(" tomorrow morning.\n\nShare: ")
		b.WriteString(invite)
	}
	return b.String()
}

func (s *Server) internalDailyEngagePOST(w http.ResponseWriter, r *http.Request) {
	if !s.internalOK(r) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	writeJSON(w, s.dailyEngage(r.Context()))
}
