package growth

import (
	"context"
	"strings"

	"github.com/samuel/ai-workers/agents/internal/contract"
	"github.com/samuel/ai-workers/agents/internal/departments"
	"github.com/samuel/ai-workers/agents/internal/llm"
)

const deckOverlay = `You are the Growth clerk for TheMobileProf Technologies (Lagos), the only CAC-registered company in this office.
Projects (MomLaunchpad, Academy, Finchest, HomeGauge, Mechazone) are bets under that company, not separate companies.
A project "cofounder" is the public face of a bet. They are not automatically a director of TheMobileProf Technologies.
You draft pitch decks and business-plan outlines from the Card. You cannot send email, WhatsApp, or a PDF.
Do not invent waitlist, MAU, GMV, burn, valuation, or "investors said". Quote only context_data.project fields. If a fact is missing, write [TO BE COMPLETED].
Spend this month is project.burn_ngn only when that field is already on the Card. Never compute it.
Legal does not write this document. After Accept, a human may paste the pack into /legal for a claims review. Files product stores the PDF after /file.
The deck is Guy Kawasaki's 10-slide pitch (10/20/30: ten slides, twenty minutes, thirty-point type when they paste). Use exactly these ten titles, in this order. Do not add an 11th slide. Do not add a traction slide you cannot fill.
1. Title — bet name + TheMobileProf Technologies. Face if named. Not a second company.
2. Problem / Opportunity — Card hypothesis.
3. Value Proposition — Card one-liner (who pays, for what).
4. Underlying Magic — how it works, only if already on the Card. Else [TO BE COMPLETED].
5. Business Model — who pays, from the one-liner. Do not invent a price.
6. Go-to-Market Plan — mission, journey, gate, this week. Named places only.
7. Competitive Analysis — only rivals already named on the Card. Never invent a grid or scrape the web.
8. Management Team — Face names. They are not automatically directors of TheMobileProf Technologies.
9. Financials and Key Metrics — burn_ngn only if on the Card. Never invent MAU, waitlist, or GMV.
10. Current Status, Timeline, and Use of Funds — stage, evidence/notes, mission due. Use of funds only from Card spend.
You are in a copy session with the handler, not a one-shot wizard. Work ONE slide at a time.
context_data.slides is the live deck. context_data.current_slide is the slide in play (1–10).
Start (or stay) on that slide: propose short on-slide copy, then one speaker-note line, then ONE question (tighter? next? skip?).
When they like it, set that slide status to agreed and move current_slide forward. If they say skip, status skipped and move on.
If they say "business plan", write a longer outline with the same ten headings into body — still cannot send.
Do not dump all ten slides into output_text. The pack stores the full deck. Chat is the current slide plus your question.
Never invent Sequoia or a London VC. Audience is named people in context (Face, notes, CRM).
cannot_send is always true. Chat never Accepts. Customer WhatsApp is sales, not this session.
Nigerian/West African commercial English. Phone-short. No markdown tables.
task_type must be "deck".`

type Spec struct {
	ID    string
	Label string
	Help  string
}

func Specs() []Spec {
	return []Spec{
		{ID: "session", Label: "Deck session", Help: "Chat each Kawasaki slide. The pack updates every turn. Chat never Accepts."},
		{ID: "plan", Label: "Business plan", Help: "Longer outline with the same ten headings. Legal flags claims; it does not write this."},
		{ID: "audience", Label: "Audience", Help: "Who sits in the room. Named from notes, Face, or CRM."},
		{ID: "beats", Label: "Beats", Help: "Guy Kawasaki’s 10 slides from the Card."},
		{ID: "deck", Label: "Deck pack", Help: "Markdown for those 10 slides."},
	}
}

func WalkIDs() []string {
	return []string{"session"}
}

// KawasakiSlide is one of Guy Kawasaki's 10 pitch-deck slides.
type KawasakiSlide struct {
	N     int    `json:"n"`
	Title string `json:"title"`
	Card  string `json:"card"`
}

// KawasakiSlides is the 10/20/30 pitch, in order. The clerk must not add an 11th.
func KawasakiSlides() []KawasakiSlide {
	return []KawasakiSlide{
		{1, "Title", "Bet name + TheMobileProf Technologies. Face if named. Not a second company."},
		{2, "Problem / Opportunity", "Card hypothesis: who hurts, what must be true."},
		{3, "Value Proposition", "Card one-liner: who pays, for what."},
		{4, "Underlying Magic", "How it works, only if already on the Card. Else [TO BE COMPLETED]."},
		{5, "Business Model", "Who pays, from the one-liner. Do not invent a price."},
		{6, "Go-to-Market Plan", "Mission, journey, gate, this week. Named places only."},
		{7, "Competitive Analysis", "Only rivals already named on the Card. Never invent a grid."},
		{8, "Management Team", "Face names. They are not automatically directors of TheMobileProf Technologies."},
		{9, "Financials and Key Metrics", "burn_ngn only if on the Card. Never invent MAU, waitlist, or GMV."},
		{10, "Current Status, Timeline, and Use of Funds", "Stage, evidence/notes, mission due. Use of funds only from Card spend."},
	}
}

func Lookup(id string) (Spec, bool) {
	id = strings.ToLower(strings.TrimSpace(id))
	for _, s := range Specs() {
		if s.ID == id {
			return s, true
		}
	}
	return Spec{}, false
}

func LooksLike(data map[string]any, task string) bool {
	if strings.EqualFold(strings.TrimSpace(str(data["channel"])), "whatsapp") {
		return false
	}
	if strings.EqualFold(str(data["ref_kind"]), "growth_deck") {
		return true
	}
	if _, ok := data["slides"]; ok {
		return true
	}
	a := strings.ToLower(strings.TrimSpace(str(data["action"])))
	if _, ok := Lookup(a); ok {
		return true
	}
	switch a {
	case "walk", "next", "pitch", "slides", "session", "business-plan", "business_plan":
		return true
	}
	if strings.EqualFold(str(data["task_type"]), "deck") {
		return true
	}
	lower := strings.ToLower(task)
	for _, k := range []string{"pitch deck", "pitchdeck", "slide deck", "investor deck", "business plan", "slides for"} {
		if strings.Contains(lower, k) {
			return true
		}
	}
	return strings.Contains(lower, "deck") && !strings.Contains(lower, "linkedin")
}

func Infer(data map[string]any, task string) string {
	return infer(data, task)
}

func infer(data map[string]any, task string) string {
	if data == nil {
		data = map[string]any{}
	}
	if raw, ok := data["action"].(string); ok {
		a := strings.ToLower(strings.TrimSpace(raw))
		if _, ok := Lookup(a); ok {
			return a
		}
		switch a {
		case "walk", "next", "pitch", "slides", "deck":
			return "session"
		case "business-plan", "business_plan":
			return "plan"
		}
	}
	lower := strings.ToLower(task + " " + str(data["kind"]))
	switch {
	case strings.Contains(lower, "business plan"):
		return "plan"
	case strings.Contains(lower, "audience") || strings.Contains(lower, "who sits") || strings.Contains(lower, "who is in the room"):
		return "audience"
	case strings.Contains(lower, "beat") || strings.Contains(lower, "slide title"):
		return "session"
	case strings.Contains(lower, "pitch") || strings.Contains(lower, "deck") || strings.Contains(lower, "slides"):
		return "session"
	default:
		return "session"
	}
}

func handleDeck(ctx context.Context, c llm.Completer, req contract.Request, data map[string]any) (contract.Response, error) {
	action := infer(data, req.TaskDescription)
	system := deckOverlay + "\n\nAction: " + action + `

structured_data MUST include:
- action: session | plan
- project_slug (from context if present)
- audience (named people / room; never invent)
- title (short pack title)
- current_slide: 1–10
- slides: [{n, title, copy, notes, status: draft|agreed|skipped}] — merge into the live ten; keep titles
- body: optional longer outline only when they asked for a business plan
- save_deck: true when context has a project
- cannot_send: true
- record_lead: false
- needs_human: true if they asked you to email the deck, invent traction, or file at CAC
output_text: the current slide’s proposed copy, then ONE question. Do not dump all ten slides.`

	resp, err := departments.Run(ctx, c, system, departments.UserPrompt(req.TaskDescription, data))
	if resp.StructuredData == nil {
		resp.StructuredData = map[string]any{}
	}
	resp.StructuredData["task_type"] = "deck"
	resp.StructuredData["action"] = action
	resp.StructuredData["cannot_send"] = true
	resp.StructuredData["record_lead"] = false
	if resp.StructuredData["current_slide"] == nil && data["current_slide"] != nil {
		resp.StructuredData["current_slide"] = data["current_slide"]
	}
	if projectID := intFrom(data); projectID > 0 || str(nestedSlug(data)) != "" {
		resp.StructuredData["save_deck"] = true
	}
	return resp, err
}

func str(v any) string {
	s, _ := v.(string)
	return s
}

func nestedSlug(data map[string]any) string {
	if s := str(data["project_slug"]); s != "" {
		return s
	}
	if nested, ok := data["project"].(map[string]any); ok {
		return str(nested["slug"])
	}
	return str(data["project"])
}

func intFrom(data map[string]any) int64 {
	switch v := data["project_id"].(type) {
	case int64:
		return v
	case int:
		return int64(v)
	case float64:
		return int64(v)
	}
	if nested, ok := data["project"].(map[string]any); ok {
		switch v := nested["id"].(type) {
		case int64:
			return v
		case int:
			return int64(v)
		case float64:
			return int64(v)
		}
	}
	return 0
}
