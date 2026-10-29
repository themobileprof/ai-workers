package admin

import (
	"strconv"
	"strings"

	"github.com/samuel/ai-workers/agents/internal/journeys"
)

// RunStep is one base-kit action on a project. Same sequence for every bet.
type RunStep struct {
	ID     string
	Kind   string
	Title  string
	Why    string
	Detail string
	How    []string
	Href   string
	Button string
	Help   string
	Stamp  string
	Done   bool
	Now    bool
}

func (s RunStep) StampClass() string {
	switch s.Stamp {
	case "done":
		return "ok"
	case "now":
		return "brass"
	default:
		return "dark"
	}
}

func (s RunStep) StampLabel() string {
	switch s.Stamp {
	case "done":
		return "done"
	case "now":
		return "now"
	default:
		return "next"
	}
}

type runInput struct {
	Project   Project
	Drafts    []LegalDraft
	Briefs    []SeoBrief
	Decks     []GrowthDeck
	Filings   []FileFiling
	Items     []WorkItem
	Events    []ProjectEvent
	CanLegal  bool
	CanSEO    bool
	CanGrowth bool
}

func projectRun(in runInput) []RunStep {
	p := in.Project
	base := "/admin/projects/" + strconv.FormatInt(p.ID, 10)
	name := strings.TrimSpace(p.Name)
	if name == "" {
		name = "this project"
	}
	hasWork := len(in.Items) > 0
	placed := strings.TrimSpace(p.Journey) != "" && strings.TrimSpace(p.Gate) != ""

	placeWhy := "Ask placement. Accept on Decide is your yes. Chat never Accepts."
	if p.Proposal != nil {
		placeWhy = "A proposal is waiting. Open Decide and Accept or Reject."
	}

	steps := []RunStep{
		{
			ID: "card", Kind: "Must", Title: "Write the card",
			Why:  "Name, who pays, and this week’s line. Customers still buy from TheMobileProf.",
			How:  []string{"Open Card.", "Fill the one-liner if empty. Write this week’s line.", "Hypothesis: what must be true. Mission: named place, named people. Spend this month from Books if you have it.", "Click Save card."},
			Href: base + "?tab=card", Button: "Open Card",
			Help: "Write this week’s line on " + name,
			Done: strings.TrimSpace(p.OneLiner) != "" && strings.TrimSpace(p.Notes) != "",
		},
		{
			ID: "face", Kind: "Must", Title: "Seat a face",
			Why:  "Someone on People must own the week. Seat is not a login.",
			How:  []string{"They must already be on People.", "Pick them, seat cofounder or assistant.", "Click Seat them."},
			Href: base + "?tab=face", Button: "Open Face",
			Help: "Seat someone on Face for " + name,
			Done: len(p.Members) > 0,
		},
		{
			ID: "place", Kind: "Must", Title: "Place on a journey",
			Why:  placeWhy,
			How:  []string{"Open Decide. Click Ask placement if there is no proposal.", "Read the suggested gate.", "Click Accept if you agree — that writes the stage."},
			Href: base + "?tab=stamp", Button: "Open Decide",
			Help: "Ask placement for " + name,
			Done: placed && p.Proposal == nil,
		},
		{
			ID: "dates", Kind: "Must", Title: "Put a date on it",
			Why:  "A campus week, field week, or launch that survives This week changing.",
			How:  []string{"Open Dates.", "Fill What, start date, Where, Who, success.", "Leave Set as this week’s line ticked if this is the agenda. Save date."},
			Href: base + "?tab=dates", Button: "Open Dates",
			Help: "Add a date on " + name,
			Done: len(in.Events) > 0,
		},
		{
			ID: "work", Kind: "Must", Title: "Give someone work",
			Why:  "One Delivery task. Stage caps how many. Pace is hours/week on People.",
			How:  []string{"Open Work.", "Type the task, pick who (they should be on Face).", "Click Add task. Extra work sits on Backlog when the stage is full — Promote when a slot opens."},
			Href: base + "?tab=work", Button: "Open Work",
			Help: "Add a task on " + name,
			Done: hasWork,
		},
	}

	if placed {
		if g, ok := journeys.LookupGate(p.Journey, p.Gate); ok {
			steps = append(steps, RunStep{
				ID: "gate", Kind: "This gate", Title: g.Label,
				Why:    g.DoneWhen,
				Detail: g.Mission,
				How:    []string{"Write this week’s mission on Card if it is empty.", "Send a human if it needs the street. Bring names back.", "When the proof is real, Ask placement again — then Accept on Decide."},
				Href:   base + "?tab=card", Button: "This week’s line",
				Done: strings.TrimSpace(p.Notes) != "" && hasWork && len(p.Members) > 0,
			})
		}
	}

	legal := RunStep{
		ID: "legal", Kind: "As needed", Title: "Paper if someone is in",
		Why:  "NDA or contractor. Copy the text yourself. It does not email.",
		How:  []string{"Open Legal.", "Pick the paper and the other party. Ask draft.", "Accept or Turn away when it appears."},
		Href: base + "?tab=legal", Button: "Open Legal",
		Help: "Ask a legal draft for " + name,
		Done: len(in.Drafts) > 0,
	}
	if !in.CanLegal {
		legal.Detail = "Legal clerk is off until the LLM env is set, or you need the Legal desk tick."
	}
	steps = append(steps, legal)

	steps = append(steps, RunStep{
		ID: "files", Kind: "As needed", Title: "File the evidence",
		Why:  "Ops Telegram /file or a PDF. Accept on Files writes Drive. Chat never files.",
		How:  []string{"Send the file on ops Telegram with /file.", "A catalog match files itself. Odd paths stay here for Accept.", "Open Files on this project if you need to check."},
		Href: base + "?tab=files", Button: "Open Files",
		Done: len(in.Filings) > 0,
	})

	seo := RunStep{
		ID: "seo", Kind: "As needed", Title: "Name the public page",
		Why:  "Walk the designed page: facts, FAQ, schema, then an onsite pack. Offsite is a Delivery plan unless you Turn away.",
		How:  []string{"Put the public URL on Card.", "Open SEO. Do the step marked now.", "Download the pack or give Cursor the URL. Turn away if a fact is invented."},
		Href: base + "?tab=seo", Button: "Open SEO",
		Help: "Ask SEO for " + name,
		Done: len(in.Briefs) > 0,
	}
	if strings.TrimSpace(p.URL) == "" && !seo.Done {
		seo.Detail = "Skip until there is a public URL on Card."
	}
	if !in.CanSEO {
		if seo.Detail != "" {
			seo.Detail += " "
		}
		seo.Detail += "SEO clerk is off until the LLM env is set, or you need the SEO desk tick."
	}
	steps = append(steps, seo)

	deck := RunStep{
		ID: "deck", Kind: "As needed", Title: "Pitch deck from the Card",
		Why:  "Chat Growth one Kawasaki slide at a time from the Card. The pack updates every turn. Growth cannot send or invent traction.",
		How:  []string{"Write the Card one-liner first.", "Open Deck. Start a session and chat each slide.", "Download the pack or give Cursor the URL. Export a PDF and /file it under product. Turn away if a fact is invented."},
		Href: base + "?tab=deck", Button: "Open Deck",
		Help: "Ask a pitch deck for " + name,
		Done: len(in.Decks) > 0,
	}
	if strings.TrimSpace(p.OneLiner) == "" && !deck.Done {
		deck.Detail = "Skip until the Card has a one-liner."
	}
	if !in.CanGrowth {
		if deck.Detail != "" {
			deck.Detail += " "
		}
		deck.Detail += "Growth clerk is off until the LLM env is set, or you need the Growth desk tick."
	}
	steps = append(steps, deck)
	return finishRun(steps)
}

func finishRun(steps []RunStep) []RunStep {
	mustLeft := 0
	for _, s := range steps {
		if s.Kind == "Must" && !s.Done {
			mustLeft++
		}
	}
	nowSet := false
	for i := range steps {
		if steps[i].Done {
			steps[i].Stamp = "done"
			steps[i].Now = false
			continue
		}
		if steps[i].Kind == "As needed" && mustLeft > 0 {
			steps[i].Stamp = "later"
			continue
		}
		if !nowSet {
			steps[i].Now = true
			steps[i].Stamp = "now"
			nowSet = true
			continue
		}
		steps[i].Stamp = "later"
	}
	return steps
}

func runMustLeft(steps []RunStep) int {
	n := 0
	for _, s := range steps {
		if s.Kind == "Must" && !s.Done {
			n++
		}
	}
	return n
}

// RunLabel is a short gap list for Board and the project wall (no extra queries).
func (p Project) RunLabel() string {
	var gaps []string
	if strings.TrimSpace(p.OneLiner) == "" || strings.TrimSpace(p.Notes) == "" {
		gaps = append(gaps, "Card")
	}
	if len(p.Members) == 0 {
		gaps = append(gaps, "Face")
	}
	if strings.TrimSpace(p.Journey) == "" || p.Proposal != nil {
		gaps = append(gaps, "Decide")
	}
	if p.NextEvent == nil {
		gaps = append(gaps, "Dates")
	}
	if len(gaps) == 0 {
		return ""
	}
	return strconv.Itoa(len(gaps)) + " of the base kit still empty"
}
