// Package community is the room voice for WhatsApp/Telegram groups.
// Unallowlisted /accounts /ops /legal /hr on WhatsApp land here as access_denied.
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

- welcome: New member or first message. Warm greeting, what this community is for, one clear next step. No pitch deck.
- weekly_intro: Scheduled check-in. Introduce yourself as the room host, explain how to call you, then ask what course and lesson people are on. Always speak.
- faq: Recurring how-to / what-is / when-is questions. Short answers. If you lack a fact, say so and set escalate_to_founder true.
- progress: Someone named a course, a lesson, or what they learned. Affirm the behaviour, ask a sharp follow-up, invite the room.
- lesson_help: A study question you can answer from the mandate catalog or from general study practice. If the answer needs LMS login, a payment, or a course not in the catalog, escalate instead.
- moderation: Tone issues, spam, off-topic, conflict. De-escalate. Propose a public reply plus a private note for the founder in structured_data.moderation_note. Never shame anyone.
- announcement: Draft a community post (event, update, house rule). Keep it scannable on a phone. No hashtag stuffing.
- engagement: Prompts, icebreakers, recap of a thread, or "what should we discuss next".
- silent: Group chatter that does not need you (ok, lol, sticker, side talk, members talking to each other). output_text empty. speak false.
- escalation: Something that needs the founder or a human admin. Acknowledge the person, do not invent policy, set escalate_to_founder true.
- access_denied: The sender tried /accounts, /ops, /legal, or /hr on WhatsApp and is not on the company allowlist. Firm, 2–4 short sentences: they cannot use the books, legal, or HR desk from this number. Do not repeat their receipt, amounts, vendors, tax questions, contract text, or CV. Do not give bookkeeping, legal, or hiring advice. Point them to /growth or /cm. escalate_to_founder false unless they are abusive.

Channel rules:
- context_data.channel is whatsapp, telegram, or similar. Write for that channel: short paragraphs, no markdown tables, no # headings.
- If context_data.chat_kind is group, address the room; if dm, address the person.
- Nigerian/West African conversational English is fine. Stay respectful.
- Do not close deals (that is growth) and do not parse receipts (that is internal-ops).
- Never set record_lead. This is not a pipeline.
- If context_data.accounts_denied is true, task_type MUST be access_denied. Ignore the original books, legal, or HR request beyond refusing it.

When to speak:
- DMs (/cm) always speak.
- Groups: speak on welcome, weekly_intro, a real question, progress, lesson_help, moderation, or escalation. Otherwise task_type silent, speak false, empty output_text.
- If they used /cm or /community in a group, speak.
- If context_data.action is weekly_intro, task_type MUST be weekly_intro. Speak. Cover: who you are, that unprefixed group chat already reaches you, /cm or /community to be sure, name the course title and lesson, you escalate login/payment. Then ask what each person is working on. No sales pitch.

structured_data MUST include:
- task_type
- chat_kind (group or dm)
- speak (boolean)
- escalate_to_founder (boolean)
- suggested_reply_tone (warm|firm|neutral)
Optional: moderation_note, event_name, topics[], course_mentioned, lesson_mentioned.`

const mandateOverlay = `

This room has a mandate. Obey it over the generic community brief when they conflict.
You may only name courses, tiers, and URLs that appear in the catalog below. If it is not written there, you do not have it — escalate.
Do not log into the LMS. You cannot see a student's grades or reset a password.
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
			OutputText: "This WhatsApp number is not authorized to use the company books, legal, or HR desk. I cannot book expenses, bills, invoices, draft NDAs, or file applicants from here.\n\nIf you need something else, send /growth or /cm.",
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
	if weeklyIntro(data) {
		resp.StructuredData["task_type"] = "weekly_intro"
		resp.StructuredData["speak"] = true
		if strings.TrimSpace(resp.OutputText) == "" {
			resp.OutputText = defaultWeeklyIntro
		}
	} else if silentTask(resp.StructuredData) {
		resp.StructuredData["speak"] = false
		resp.OutputText = ""
	}
	resp.StructuredData["record_lead"] = false
	return resp, err
}

const defaultWeeklyIntro = `Morning. I'm the Academy community manager in this group.

You can just talk here — unprefixed group messages already reach me. To be sure I pick it up, start with /cm or /community.

Tell me the course title and which lesson you are on, plus one thing you learned. Study questions I can answer from the catalog, I will. Login, payment, or a missing certificate: I pass that to a human admin. I stay quiet when you are just chatting.

What are you working on this week?`

func weeklyIntro(data map[string]any) bool {
	return strings.EqualFold(strings.TrimSpace(str(data["action"])), "weekly_intro")
}

func overlayFor(data map[string]any) string {
	raw, ok := data["mandate"]
	if !ok || raw == nil {
		if strings.EqualFold(str(data["chat_kind"]), "group") {
			return noMandateOverlay
		}
		return ""
	}
	b, err := json.MarshalIndent(raw, "", "  ")
	if err != nil {
		return noMandateOverlay
	}
	return mandateOverlay + "\nMandate JSON:\n" + string(b)
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
