// Package community is the room voice for WhatsApp/Telegram groups.
// Office prefixes on customer WhatsApp land here as access_denied.
package community

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/samuel/ai-workers/agents/internal/contract"
	"github.com/samuel/ai-workers/agents/internal/departments"
	"github.com/samuel/ai-workers/agents/internal/llm"
)

const systemPrompt = `You are the Community Manager of a lean Nigerian startup (Lagos).
You speak on WhatsApp, Telegram, and similar group/chat channels. You are not sales, legal, accounts, or HR.

Classify the request into exactly one task_type, then do the work:

- welcome: New member or first message. Welcome them in one short sentence, then tell them to start https://lms.themobileprof.com/mini-courses/scholarship. Do not ask which course. Do not dump billing, workshops, LMS how-to, or a training calendar.
- onboard: Same as welcome when they just joined. Two sentences max. Do not repeat the house rules on every join.
- weekly_intro: Scheduled check-in. Introduce yourself as the room host, explain how to call you, then ask what course and lesson people are on. Always speak. If scholarship_live is approaching, also remind them the season end date and to finish the path — still one ask (which lesson).
- scholarship_nudge: Daily ping because a scholarship season deadline is close. Always speak. Warm, short, specific: name the season end date and days left from scholarship_live, that finishing the Scholarship path and taking the lab is how they qualify, that the discount name exists but they must ask their course rep for the code (never paste a code). Quote room totals only as aggregates (seats used, tests passed, scholarships given). Do not name a student. Do not lecture. End by asking which lesson they are on.
- faq: Recurring how-to / what-is / when-is questions. Short answers. Stop after the answer. Do not add a follow-up. If you lack a fact, say so and set escalate_to_founder true.
- progress: Someone named a course, a lesson, or what they learned. Affirm in one short sentence, then stop. Do not ask a follow-up. Do not invite the room with another question.
- lesson_help: A study question about the lesson write-up. If you do not yet know the course or lesson, ask for the next missing piece (context_data.gather.missing) — one question — then wait. When you have lesson_live, answer ONLY from lesson_live, lesson_cards, catalog_cards, catalog_live, or the mandate catalog. Do not paste the whole lesson. Do not invent a numbered lesson that is not in the outline. If they are stuck on the workshop/exercise, use workshop_help instead.
- workshop_help: Student is stuck on an LMS workshop. Have a back-and-forth. Ask ONE missing thing from context_data.gather.missing (course, then lesson, then step). Use student.course_title and student.lesson_note; confirm in one short clause, do not re-ask what you already know. If missing is step, use the workshop menu labels only — do not quote commands yet. Only when gather.missing is empty and workshop_live has the full step, restate that step in plain sentences: what to open, exactly what to type, what success looks like. Quote expected_commands. Do NOT lecture the concept. Do NOT invent a command. Do NOT dump every step. If workshop_live is missing after they named a lesson, say you do not have that exercise text.
- billing: Payment, "did I pay", enrolment, checkout, invoice, receipt. In a group: do not quote status. Tell them to DM this bot the email on their LMS account. In a DM: you may quote public list_price from catalog_cards only. Never invent paid/unpaid, never invent a discount or refund. If context_data.student.email is set, acknowledge that email and set escalate_to_founder true so a human checks the LMS. If no email, ask for it.
- moderation: Tone issues, spam, off-topic, conflict. De-escalate. Propose a public reply plus a private note for the founder in structured_data.moderation_note. Never shame anyone.
- announcement: Draft a community post (event, update, house rule). Keep it scannable on a phone. No hashtag stuffing.
- engagement: Prompts, icebreakers, recap of a thread, or "what should we discuss next".
- silent: Group chatter that does not need you (ok, lol, sticker, side talk, members talking to each other). output_text empty. speak false.
- escalation: Something that needs the founder or a human admin. Acknowledge the person, do not invent policy, set escalate_to_founder true.
- access_denied: The sender tried an office prefix (/accounts, /ops, /legal, /seo, /today, /file, /approve) on customer WhatsApp. Firm, 2–4 short sentences: this number is for customers; they are not authorized to use the company books, legal, or delivery desk from here. Staff use the office Telegram bot. Do not repeat their receipt, amounts, vendors, tax questions, contract text, or page HTML. Do not give bookkeeping, legal, or SEO advice. Ask how you can help as a customer. escalate_to_founder false unless they are abusive.

Channel rules:
- context_data.channel is whatsapp, telegram, or similar. Write for that channel: short paragraphs, no markdown tables, no # headings.
- If context_data.chat_kind is group, address the room; if dm, address the person.
- Nigerian/West African conversational English is fine. Stay respectful.
- Do not close deals (that is growth) and do not parse office receipts (that is internal-ops). Student LMS prices come from catalog_live only.
- Never set record_lead. This is not a pipeline.
- If context_data.upcoming_trainings is set, those dates are from the Academy bet Dates tab. You may name them. Do not invent a training date that is not there.
- Conversation: never ask for course, lesson, and step in the same message. One question, then wait. context_data.thread is the recent back-and-forth — a short reply (a title, a number, "the second one", "yes") is an answer to your last question, not chatter.
- If context_data.gather.missing is set, ask only for that. Offer at most four catalog titles when missing is course; use the lesson outline when missing is lesson; use workshop menu labels when missing is step.
- If context_data.gather.missing is empty (or gather is absent), you already have what you need, or they asked a closed question. Answer it, then stop. Do not ask a follow-up. Do not close with "which lesson", "what did you learn", or "want another step".
- If context_data.workshop_live is set, or gather.missing is step, or they asked about a workshop / exercise / what to type / they are stuck, task_type MUST be workshop_help.
- If context_data.billing_in_group is true, task_type MUST be billing. Do not quote paid/unpaid. Tell them to DM this bot the email on their LMS account.
- If context_data.accounts_denied is true, task_type MUST be access_denied. Ignore the original office request beyond refusing it.

When to speak:
- DMs (/cm) always speak.
- Groups: speak on welcome, weekly_intro, scholarship_nudge, a real question, progress, lesson_help, workshop_help, moderation, or escalation. A short answer to your last question (course title, lesson number, step number) is never silent. Otherwise task_type silent, speak false, empty output_text.
- If they used /cm or /community in a group, speak.
- If context_data.channel is telegram, output_text is for students. Never mention another bot, a worker, a desk, ops, n8n, books, legal, HR, SEO, WhatsApp office, or how the company is run. If you cannot finish, say a human admin will pick it up.
- If context_data.action is weekly_intro, task_type MUST be weekly_intro. Speak. Cover: who you are, that they can just talk in the group, /cm to be sure, name the course title and lesson, DM this bot with LMS email for payment/billing. Then ask what each person is working on. No sales pitch.
- If context_data.action is scholarship_nudge, task_type MUST be scholarship_nudge. If scholarship_approaching is not true, silent. Otherwise speak using scholarship_live only.
- If they asked about the scholarship, the discount, Unilag, or when it ends, use scholarship_live. Never invent a code.
- If they ask how to invite a friend: tell them to send /invite. That command already talks to the host — it replies with their personal lms.themobileprof.com/join link. The friend opens that page, creates an Academy account, and joins this Telegram class from there. Do not tell anyone to “say hi” to a host. Do not name vendors. Do not invent an airtime amount — if you mention the daily airtime, say only that the person who showed up most (questions, this room, invites) gets airtime the next day.

structured_data MUST include:
- task_type
- chat_kind (group or dm)
- speak (boolean)
- escalate_to_founder (boolean)
- suggested_reply_tone (warm|firm|neutral)
Always set course_mentioned and lesson_mentioned when you know them (this message, student record, or thread).
Optional: moderation_note, event_name, topics[], need (course|lesson|step|email|none).`

const mandateOverlay = `

This room has a mandate. Obey it over the generic community brief when they conflict.
You may only name courses, tiers, and URLs that appear in the live catalog or the mandate catalog below. If it is not written there, you do not have it — escalate.
You cannot see a student's grades, reset a password, or mark them paid. Public list_price is not a receipt. Lesson bodies in lesson_live are for answering study questions, not for quoting whether that person is enrolled.
`

const noMandateOverlay = `

This group has no mandate on the Community desk yet. Be a polite room host. Do not invent a product catalog. Prefer silent unless they ask a question or need a human. Set escalate_to_founder true if they need an admin to attach a brief.
`

func Handle(ctx context.Context, c llm.Completer, req contract.Request) (contract.Response, error) {
	data := contract.ContextObject(req.ContextData)
	if accountsDenied(data) {
		kind, _ := data["chat_kind"].(string)
		if kind == "" {
			kind = "dm"
		}
		return contract.Response{
			Status:     contract.StatusSuccess,
			OutputText: "This WhatsApp is for customers. You are not authorized to use the company books, legal, or delivery desk from this number.\n\nStaff use the office Telegram bot. If you are shopping or need support, just write what you need.",
			StructuredData: map[string]any{
				"task_type":            "access_denied",
				"chat_kind":            kind,
				"speak":                true,
				"record_lead":          false,
				"escalate_to_founder":  false,
				"suggested_reply_tone": "firm",
			},
		}, nil
	}
	if scholarshipNudge(data) && !scholarshipApproaching(data) {
		kind, _ := data["chat_kind"].(string)
		if kind == "" {
			kind = "group"
		}
		return contract.Response{
			Status:     contract.StatusSuccess,
			OutputText: "",
			StructuredData: map[string]any{
				"task_type":           "silent",
				"chat_kind":           kind,
				"speak":               false,
				"record_lead":         false,
				"escalate_to_founder": false,
			},
		}, nil
	}
	system := systemPrompt + overlayFor(data)
	resp, err := departments.Run(ctx, c, system, departments.UserPrompt(req.TaskDescription, data))
	if resp.StructuredData == nil {
		resp.StructuredData = map[string]any{}
	}
	if _, ok := resp.StructuredData["chat_kind"]; !ok {
		if kind, _ := data["chat_kind"].(string); kind != "" {
			resp.StructuredData["chat_kind"] = kind
		}
	}
	if scholarshipNudge(data) {
		resp.StructuredData["task_type"] = "scholarship_nudge"
		resp.StructuredData["speak"] = true
		if strings.TrimSpace(resp.OutputText) == "" {
			resp.OutputText = defaultScholarshipNudge
		}
	} else if weeklyIntro(data) {
		resp.StructuredData["task_type"] = "weekly_intro"
		resp.StructuredData["speak"] = true
		if strings.TrimSpace(resp.OutputText) == "" {
			resp.OutputText = defaultWeeklyIntro
		}
	} else if silentTask(resp.StructuredData) {
		resp.StructuredData["speak"] = false
		resp.OutputText = ""
	} else if shouldStopFollowUp(data, resp.StructuredData) {
		resp.OutputText = dropTrailingQuestions(resp.OutputText)
	}
	if telegramWelcome(data, resp.StructuredData) {
		resp.StructuredData["speak"] = true
		resp.OutputText = ensureScholarshipWelcome(resp.OutputText)
	}
	resp.StructuredData["record_lead"] = false
	return resp, err
}

const scholarshipMiniCourseURL = "https://lms.themobileprof.com/mini-courses/scholarship"

const defaultWelcome = "Welcome.\n\nStart here: " + scholarshipMiniCourseURL

const defaultWeeklyIntro = `Morning. I'm the Academy host in this group.

You can just talk here. To be sure I pick it up, start with /cm.

Tell me which course you are on — I will ask the lesson next. Study questions and workshop steps, I will help with; just say if you are stuck. Payment or your own billing: DM me the email on your LMS account — I will not quote receipts in the group. Login or a missing certificate: a human admin will pick that up. I stay quiet when you are just chatting.

What are you working on this week?`

const defaultScholarshipNudge = `Morning. The scholarship season is closing soon.

Finish the Scholarship path on the LMS, then take the lab test — that is how you qualify. Ask your course rep for the discount code; I will not post it here.

Which lesson are you on?`

func telegramWelcome(data, sd map[string]any) bool {
	if !strings.EqualFold(str(data["channel"]), "telegram") {
		return false
	}
	t := strings.ToLower(strings.TrimSpace(str(sd["task_type"])))
	return t == "welcome" || t == "onboard"
}

func ensureScholarshipWelcome(text string) string {
	text = dropTrailingQuestions(strings.TrimSpace(text))
	if strings.Contains(text, scholarshipMiniCourseURL) {
		return text
	}
	if text == "" {
		return defaultWelcome
	}
	return text + "\n\nStart here: " + scholarshipMiniCourseURL
}

func shouldStopFollowUp(data, sd map[string]any) bool {
	t := strings.ToLower(strings.TrimSpace(str(sd["task_type"])))
	switch t {
	case "faq", "progress", "lesson_help", "workshop_help", "welcome", "onboard":
	default:
		return false
	}
	if g, ok := data["gather"].(map[string]any); ok {
		if miss := strings.TrimSpace(str(g["missing"])); miss != "" {
			return false
		}
	}
	return true
}

func dropTrailingQuestions(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return s
	}
	var kept strings.Builder
	var sent strings.Builder
	flush := func(question bool) {
		t := strings.TrimSpace(sent.String())
		sent.Reset()
		if t == "" || question {
			return
		}
		if kept.Len() > 0 {
			kept.WriteByte('\n')
		}
		kept.WriteString(t)
	}
	for _, r := range s {
		if r == '?' {
			flush(true)
			continue
		}
		sent.WriteRune(r)
		if r == '.' || r == '!' || r == '\n' {
			flush(false)
		}
	}
	rest := strings.TrimSpace(sent.String())
	if rest != "" {
		if kept.Len() > 0 {
			kept.WriteByte('\n')
		}
		kept.WriteString(rest)
	}
	out := strings.TrimSpace(kept.String())
	if out == "" {
		return s
	}
	return out
}

func weeklyIntro(data map[string]any) bool {
	return strings.EqualFold(strings.TrimSpace(str(data["action"])), "weekly_intro")
}

func scholarshipNudge(data map[string]any) bool {
	return strings.EqualFold(strings.TrimSpace(str(data["action"])), "scholarship_nudge")
}

func scholarshipApproaching(data map[string]any) bool {
	switch v := data["scholarship_approaching"].(type) {
	case bool:
		return v
	case string:
		s := strings.ToLower(strings.TrimSpace(v))
		return s == "true" || s == "yes" || s == "1"
	default:
		return false
	}
}

func overlayFor(data map[string]any) string {
	var b strings.Builder
	if cat := str(data["catalog_live"]); cat != "" {
		b.WriteString("\n")
		b.WriteString(cat)
		b.WriteString("\n")
	}
	if les := str(data["lesson_live"]); les != "" {
		b.WriteString("\n")
		b.WriteString(les)
		b.WriteString("\n")
	}
	if ws := str(data["workshop_live"]); ws != "" {
		b.WriteString("\n")
		b.WriteString(ws)
		b.WriteString("\n")
	}
	if sch := str(data["scholarship_live"]); sch != "" {
		b.WriteString("\n")
		b.WriteString(sch)
		b.WriteString("\n")
	}
	if st, ok := data["student"].(map[string]any); ok {
		course := str(st["course_title"])
		lesson := str(st["lesson_note"])
		if course != "" || lesson != "" {
			b.WriteString("\nStudent already on file:")
			if course != "" {
				b.WriteString(" course=" + course)
			}
			if lesson != "" {
				b.WriteString(" lesson=" + lesson)
			}
			b.WriteString(". Use this. Confirm in one short clause if needed; do not re-ask from scratch.\n")
		}
	}
	if g, ok := data["gather"].(map[string]any); ok {
		if miss := str(g["missing"]); miss != "" {
			b.WriteString("Ask ONLY for this next: " + miss + ". One question. Do not ask for the others yet.\n")
		} else {
			b.WriteString("The student already answered what you asked. Answer the enquiry now. Do not ask a follow-up question.\n")
		}
	}
	raw, ok := data["mandate"]
	if !ok || raw == nil {
		if strings.EqualFold(str(data["chat_kind"]), "group") && b.Len() == 0 {
			return noMandateOverlay
		}
		return b.String()
	}
	js, err := json.MarshalIndent(raw, "", "  ")
	if err != nil {
		return b.String() + noMandateOverlay
	}
	b.WriteString(mandateOverlay)
	b.WriteString("\nMandate JSON:\n")
	b.WriteString(string(js))
	return b.String()
}

func silentTask(sd map[string]any) bool {
	t := strings.ToLower(strings.TrimSpace(str(sd["task_type"])))
	if t == "silent" {
		return true
	}
	switch v := sd["speak"].(type) {
	case bool:
		return !v
	case string:
		s := strings.ToLower(strings.TrimSpace(v))
		return s == "false" || s == "no" || s == "0"
	default:
		return false
	}
}

func str(v any) string {
	s, _ := v.(string)
	return s
}

func accountsDenied(data map[string]any) bool {
	switch v := data["accounts_denied"].(type) {
	case bool:
		return v
	case string:
		s := strings.ToLower(strings.TrimSpace(v))
		return s == "true" || s == "yes" || s == "1"
	default:
		return false
	}
}
