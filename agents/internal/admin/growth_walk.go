package admin

import (
	"strconv"
	"strings"

	"github.com/samuel/ai-workers/agents/internal/departments/growth"
)

func liveDeckKinds(decks []GrowthDeck) map[string]bool {
	out := map[string]bool{}
	for _, d := range decks {
		if d.Status == "rejected" || d.Kind == "" {
			continue
		}
		out[d.Kind] = true
	}
	return out
}

func nextDeckWalkAction(decks []GrowthDeck) string {
	live := liveDeckKinds(decks)
	for _, id := range growth.WalkIDs() {
		if !live[id] {
			return id
		}
	}
	return ""
}

func nextDeckWalkLabel(after string) string {
	ids := growth.WalkIDs()
	for i, id := range ids {
		if id == after && i+1 < len(ids) {
			if spec, ok := growth.Lookup(ids[i+1]); ok {
				return spec.Label
			}
			return ids[i+1]
		}
	}
	return ""
}

func resolveDeckAction(action string, decks []GrowthDeck) string {
	action = strings.ToLower(strings.TrimSpace(action))
	if action == "" || action == "walk" || action == "next" {
		if next := nextDeckWalkAction(decks); next != "" {
			return next
		}
		return "session"
	}
	if _, ok := growth.Lookup(action); ok {
		return action
	}
	if next := nextDeckWalkAction(decks); next != "" {
		return next
	}
	return "session"
}

func deckWalk(decks []GrowthDeck, p Project) []RunStep {
	live := liveDeckKinds(decks)
	nowSet := false
	var steps []RunStep
	ready := cardHasDeckFacts(p)
	for _, id := range growth.WalkIDs() {
		spec, _ := growth.Lookup(id)
		st := RunStep{
			ID:     id,
			Kind:   "Walk",
			Title:  spec.Label,
			Why:    spec.Help,
			How:    deckWalkHow(id),
			Button: "Ask this step",
			Help:   "Ask Growth " + spec.Label,
			Done:   live[id],
		}
		if !ready && !st.Done {
			st.Detail = "Skip until the Card has a one-liner."
		}
		if st.Done {
			st.Stamp = "done"
		} else if !ready {
			st.Stamp = "later"
		} else if !nowSet {
			st.Now = true
			st.Stamp = "now"
			nowSet = true
		} else {
			st.Stamp = "later"
		}
		steps = append(steps, st)
	}
	return steps
}

func deckWalkHow(id string) []string {
	if id == "plan" {
		return []string{
			"Same ten Kawasaki headings, longer outline. Not a CAC filing.",
			"Ask this step. Paste into /legal if you need a claims review.",
			"Export a PDF and /file it under this bet’s product folder.",
		}
	}
	return []string{
		"Start a session. Name who sits in the room.",
		"Chat each Kawasaki slide. The pack URL updates after every reply.",
		"Say next, skip, or rewrite a line. Accept when the ten slides are right. Chat never Accepts.",
	}
}

func liveProposedSession(decks []GrowthDeck) GrowthDeck {
	for _, d := range decks {
		if d.Status == "proposed" && d.Kind == "session" {
			return d
		}
	}
	for _, d := range decks {
		if d.Status == "proposed" && len(d.Slides) > 0 {
			return d
		}
	}
	return GrowthDeck{}
}

func seedSessionSlides(p Project) []growth.DeckSlide {
	out := growth.EmptySlides()
	for i := range out {
		out[i].Copy = strings.TrimSpace(kawasakiFill(p, out[i].N))
	}
	return out
}

func latestLiveDeck(decks []GrowthDeck, kind string) GrowthDeck {
	for _, d := range decks {
		if d.Kind == kind && d.Status != "rejected" {
			return d
		}
	}
	return GrowthDeck{}
}

func deckWalkContext(decks []GrowthDeck) map[string]any {
	out := map[string]any{
		"order":   growth.WalkIDs(),
		"next":    nextDeckWalkAction(decks),
		"company": seoCompanyName,
	}
	for _, id := range growth.WalkIDs() {
		if d := latestLiveDeck(decks, id); d.ID > 0 {
			out[id] = map[string]any{
				"id":    d.ID,
				"title": d.Title,
				"body":  d.Body,
			}
		}
	}
	return out
}

type deckSeed struct {
	Audience string
	Beats    string
	Title    string
}

func cardHasDeckFacts(p Project) bool {
	return strings.TrimSpace(p.OneLiner) != ""
}

func loadDeckSeed(p Project, decks []GrowthDeck) deckSeed {
	return buildDeckSeed(p, decks, deckAskForm{})
}

func buildDeckSeed(p Project, decks []GrowthDeck, form deckAskForm) deckSeed {
	seed := deckSeed{
		Audience: strings.TrimSpace(form.Audience),
		Beats:    strings.TrimSpace(form.Beats),
		Title:    strings.TrimSpace(form.Title),
	}
	if seed.Audience == "" {
		if d := latestLiveDeck(decks, "audience"); strings.TrimSpace(d.Body) != "" {
			seed.Audience = d.Body
		} else if d := latestLiveDeck(decks, "audience"); strings.TrimSpace(d.Audience) != "" {
			seed.Audience = d.Audience
		} else {
			seed.Audience = faceAudience(p)
		}
	}
	if seed.Beats == "" {
		if d := latestLiveDeck(decks, "beats"); strings.TrimSpace(d.Body) != "" {
			seed.Beats = d.Body
		} else {
			seed.Beats = cardBeats(p)
		}
	}
	if seed.Title == "" {
		seed.Title = strings.TrimSpace(p.Name) + " deck"
	}
	return seed
}

func faceAudience(p Project) string {
	var names []string
	for _, m := range p.Members {
		line := strings.TrimSpace(m.Name)
		if m.Seat != "" {
			line += " (" + m.Seat + ")"
		}
		if line != "" {
			names = append(names, line)
		}
	}
	if len(names) == 0 {
		return ""
	}
	return "Named on Face: " + strings.Join(names, ", ") + ". A Face without Growth ticked cannot Accept."
}

func cardBeats(p Project) string {
	var lines []string
	for _, s := range growth.KawasakiSlides() {
		body := strings.TrimSpace(kawasakiFill(p, s.N))
		if body == "" {
			lines = append(lines, fmtKawasakiLine(s, "[TO BE COMPLETED]"))
			continue
		}
		lines = append(lines, fmtKawasakiLine(s, clipRunes(body, 180)))
	}
	return strings.Join(lines, "\n")
}

func fmtKawasakiLine(s growth.KawasakiSlide, body string) string {
	return strconv.Itoa(s.N) + ". " + s.Title + " — " + body
}

func kawasakiFill(p Project, n int) string {
	switch n {
	case 1:
		name := strings.TrimSpace(p.Name)
		if name == "" {
			return seoCompanyName
		}
		return name + " · " + seoCompanyName
	case 2:
		return p.Hypothesis
	case 3, 5:
		return p.OneLiner
	case 4:
		return ""
	case 6:
		return strings.TrimSpace(strings.Join(nonEmpty(
			p.Mission,
			strings.TrimSpace(p.Journey+" "+p.Gate),
			p.Notes,
		), " · "))
	case 7:
		return ""
	case 8:
		return faceNames(p)
	case 9:
		if strings.TrimSpace(p.BurnNGN) == "" {
			return ""
		}
		return "Spend this month (Card): " + p.BurnNGN
	case 10:
		return strings.TrimSpace(strings.Join(nonEmpty(p.Stage, p.Notes, p.MissionDue), " · "))
	default:
		return ""
	}
}

func faceNames(p Project) string {
	var names []string
	for _, m := range p.Members {
		line := strings.TrimSpace(m.Name)
		if m.Seat != "" {
			line += " (" + m.Seat + ")"
		}
		if line != "" {
			names = append(names, line)
		}
	}
	return strings.Join(names, ", ")
}

func nonEmpty(parts ...string) []string {
	var out []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}
