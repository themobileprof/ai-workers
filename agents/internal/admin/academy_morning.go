package admin

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/samuel/ai-workers/agents/internal/llm"
)

type dailyMorningPosts struct {
	Hook    string
	Opener  string
	Airtime string
	FromLLM bool
}

func (s *Server) dailyGroupPosts(ctx context.Context, rd RewardDay, has bool, beat string) dailyMorningPosts {
	beat = dailyEngageBeat(beat)
	hook := morningHook(rd.Day)
	fallback := dailyMorningPosts{
		Hook:    hook,
		Opener:  morningFallbackOpener(hook),
		Airtime: morningFallbackAirtime(s, rd, has),
	}
	if s == nil || s.guide == nil {
		return fallback
	}
	if ctx == nil {
		ctx = context.Background()
	}
	llmCtx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	raw, err := s.guide.Complete(llmCtx, llm.Request{
		System: morningHostSystem(beat),
		Messages: []llm.Message{{
			Role:    "user",
			Content: morningHostUser(s, rd, has, hook, beat),
		}},
		Plain: true,
	})
	if err != nil {
		log.Printf("academy morning llm: %v", err)
		return fallback
	}
	opener, airtime := parseMorningLLM(raw)
	if beat == "checkin" {
		if opener == "" {
			opener = strings.TrimSpace(raw)
		}
		if morningOpenerOK(opener, hook) {
			fallback.Opener = opener
			fallback.FromLLM = true
		}
		return fallback
	}
	if airtime == "" {
		airtime = strings.TrimSpace(raw)
	}
	if morningAirtimeOK(airtime, rd, has) {
		fallback.Airtime = airtime
		fallback.FromLLM = true
	}
	return fallback
}

func (s *Server) dailyGroupText(ctx context.Context, rd RewardDay, has bool) string {
	p := s.dailyGroupPosts(ctx, rd, has, "airtime")
	c := s.dailyGroupPosts(ctx, rd, has, "checkin")
	return strings.TrimSpace(c.Opener + "\n\n" + p.Airtime)
}

func morningHook(d time.Time) string {
	if d.IsZero() {
		d = time.Now().In(lagosLocation())
	}
	day := time.Date(d.Year(), d.Month(), d.Day(), 0, 0, 0, 0, lagosLocation())
	switch (day.Unix() / 86400) % 3 {
	case 0:
		return "greet"
	case 1:
		return "progress"
	default:
		return "questions"
	}
}

func morningFallbackOpener(hook string) string {
	switch hook {
	case "progress":
		return "Morning class. Where is everyone right now — which course, and which lesson?"
	case "questions":
		return "Morning class. Stuck on a course, lesson, workshop, or quiz? Ask here. I will talk it through."
	default:
		return "Morning class. Hope the day is treating you kind — drop a hello if you are around."
	}
}

func morningFallbackAirtime(s *Server, rd RewardDay, has bool) string {
	naira := morningNaira(rd)
	invite := "https://lms.themobileprof.com/join"
	if s != nil {
		invite = s.personalJoinURL("")
	}
	how := "How to be in the running:\n" +
		"1. Ask a lesson or workshop question here (or in a private chat with me).\n" +
		"2. Talk in this class — share one thing you learned.\n" +
		"3. Bring a friend: type /invite and I send you a link. They create their Academy account on that page, then join this class from there."
	var b strings.Builder
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

func morningNaira(rd RewardDay) string {
	if rd.Amount <= 0 {
		return "₦500"
	}
	return fmt.Sprintf("₦%d", rd.Amount)
}

func morningHostSystem(beat string) string {
	base := `You are the Academy host in a student Telegram class for TheMobileProf Academy (Lagos).
Write one short phone message. Nigerian/West African conversational English is fine. Warm. No markdown headings, no hashtags, no bullet emoji spam.
Students must never hear about other bots, workers, desks, ops, n8n, books, legal, HR, vendors, ranking, or scores.
Do not invent an airtime amount, a winner, a URL, or a fourth way to earn. Do not explain how the winner is picked. Do not ask for phone numbers in the group.
`
	if beat == "airtime" {
		return base + "This message is the airtime note only — not a greeting. Output exactly:\nAIRTIME:\n(the message)"
	}
	return base + "This message is a check-in only — no airtime, no naira, no winner. Output exactly:\nOPENER:\n(the message)"
}

func morningHostUser(s *Server, rd RewardDay, has bool, hook, beat string) string {
	naira := morningNaira(rd)
	invite := "https://lms.themobileprof.com/join"
	if s != nil {
		invite = s.personalJoinURL("")
	}
	weekday := time.Now().In(lagosLocation()).Weekday().String()
	var b strings.Builder
	b.WriteString("Today is ")
	b.WriteString(weekday)
	b.WriteString(". Word this differently from a stiff memo.\n\n")
	if beat != "airtime" {
		b.WriteString("CHECK-IN hook (use only this one): ")
		switch hook {
		case "progress":
			b.WriteString("progress — Ask which course and which lesson people are on right now. One question. Invite a short reply in the group.\n")
		case "questions":
			b.WriteString("questions — Invite anyone stuck on a course, lesson, workshop, or quiz to ask here. You will talk it through. Do not dump a lesson.\n")
		default:
			b.WriteString("greet — A short fun Lagos morning greeting, then invite the room to talk. Not a joke about money or airtime.\n")
		}
		return b.String()
	}
	b.WriteString("AIRTIME facts you must include, reworded:\n")
	b.WriteString("- Amount (exact): ")
	b.WriteString(naira)
	b.WriteString(" airtime\n")
	if has {
		b.WriteString("- Winner: ")
		b.WriteString(rd.WinnerName)
		b.WriteString(" showed up most yesterday. You already wrote them privately about the airtime. Tell everyone else ")
		b.WriteString(naira)
		b.WriteString(" is still on tomorrow.\n")
	} else {
		b.WriteString("- Nobody earned yesterday. The card is still here today. The person who shows up most today gets it tomorrow morning.\n")
	}
	b.WriteString("- Eligibility (all three, any wording): (1) a lesson or workshop question here or in a private chat with you (2) talk in this class / share one thing learned (3) type /invite, share the link, friend creates an Academy account on that page then joins this class from there.\n")
	b.WriteString("- Share URL (exact): ")
	b.WriteString(invite)
	b.WriteString("\n")
	return b.String()
}

func parseMorningLLM(raw string) (opener, airtime string) {
	raw = strings.TrimSpace(raw)
	raw = strings.TrimPrefix(raw, "```")
	raw = strings.TrimSuffix(raw, "```")
	raw = strings.TrimSpace(raw)
	low := strings.ToLower(raw)
	i := strings.Index(low, "opener:")
	j := strings.Index(low, "airtime:")
	if i >= 0 && j > i {
		opener = strings.TrimSpace(raw[i+len("opener:") : j])
		airtime = strings.TrimSpace(raw[j+len("airtime:"):])
		return opener, airtime
	}
	if j >= 0 {
		return "", strings.TrimSpace(raw[j+len("airtime:"):])
	}
	if i >= 0 {
		return strings.TrimSpace(raw[i+len("opener:"):]), ""
	}
	return "", raw
}

func morningOpenerOK(s, hook string) bool {
	s = strings.TrimSpace(s)
	if s == "" || len(s) > 1200 || !studentFacingMorningOK(s) {
		return false
	}
	low := strings.ToLower(s)
	switch hook {
	case "progress":
		return strings.Contains(low, "course") || strings.Contains(low, "lesson")
	case "questions":
		return strings.Contains(low, "lesson") || strings.Contains(low, "workshop") || strings.Contains(low, "quiz") || strings.Contains(low, "question")
	default:
		return true
	}
}

func morningAirtimeOK(s string, rd RewardDay, has bool) bool {
	s = strings.TrimSpace(s)
	if s == "" || len(s) > 2500 || !studentFacingMorningOK(s) {
		return false
	}
	if !strings.Contains(s, "/invite") {
		return false
	}
	naira := strconv.Itoa(rd.Amount)
	if rd.Amount <= 0 {
		naira = "500"
	}
	if !strings.Contains(s, naira) {
		return false
	}
	low := strings.ToLower(s)
	if has && (rd.WinnerName == "" || !strings.Contains(low, strings.ToLower(rd.WinnerName))) {
		return false
	}
	if !strings.Contains(low, "lesson") && !strings.Contains(low, "workshop") {
		return false
	}
	return true
}

func studentFacingMorningOK(s string) bool {
	low := strings.ToLower(s)
	for _, bad := range []string{"clubkonnect", "n8n", "worker", "desk", "score", "ranking", "zoho", "paystack", "ops bot"} {
		if strings.Contains(low, bad) {
			return false
		}
	}
	return true
}
