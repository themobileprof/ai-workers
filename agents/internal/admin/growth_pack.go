package admin

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/samuel/ai-workers/agents/internal/departments/growth"
)

func (d GrowthDeck) PackDownloadPath() string {
	if d.ID <= 0 {
		return ""
	}
	return "/admin/growth/" + strconv.FormatInt(d.ID, 10) + "/pack.md"
}

func (d GrowthDeck) PackPath() string {
	if d.PackToken == "" || d.Status == "rejected" {
		return ""
	}
	return "/site/deck/" + d.PackToken
}

func (d GrowthDeck) PackFilename() string {
	slug := strings.ToLower(strings.TrimSpace(d.ProjectName))
	slug = strings.Map(func(r rune) rune {
		if r >= 'a' && r <= 'z' || r >= '0' && r <= '9' {
			return r
		}
		if r == ' ' || r == '-' {
			return '-'
		}
		return -1
	}, slug)
	slug = strings.Trim(slug, "-")
	if slug == "" {
		slug = "deck"
	}
	return fmt.Sprintf("%s-%s-%d.md", slug, d.Kind, d.ID)
}

func growthPackMarkdown(d GrowthDeck, p Project, siblings []GrowthDeck) string {
	var b strings.Builder
	kind := d.KindLabel()
	b.WriteString("# " + kind + " " + strconv.FormatInt(d.ID, 10) + "\n\n")
	name := strings.TrimSpace(p.Name)
	if name == "" {
		name = d.ProjectName
	}
	b.WriteString("Bet: **" + name + "**")
	if p.Slug != "" {
		b.WriteString(" (`" + p.Slug + "`)")
	}
	b.WriteString("\n\n")
	b.WriteString("Legal name: **" + seoCompanyName + "**. This bet is not a second company.\n\n")
	if p.OneLiner != "" {
		b.WriteString("One-liner: " + p.OneLiner + "\n\n")
	}
	if p.Hypothesis != "" {
		b.WriteString("Hypothesis: " + p.Hypothesis + "\n\n")
	}
	if p.Mission != "" {
		b.WriteString("Mission: " + p.Mission)
		if p.MissionDue != "" {
			b.WriteString(" · due " + p.MissionDue)
		}
		b.WriteString("\n\n")
	}
	if p.Notes != "" {
		b.WriteString("Evidence / this week: " + p.Notes + "\n\n")
	}
	if p.Stage != "" {
		b.WriteString("Stage: " + p.Stage + "\n\n")
	}
	if strings.TrimSpace(p.BurnNGN) != "" {
		b.WriteString("Spend this month (Card): " + p.BurnNGN + "\n\n")
	}
	b.WriteString("Kind: " + kind + " · status: " + d.Status + " · updated: " + d.UpdatedAt.UTC().Format(time.RFC3339) + "\n\n")
	b.WriteString("Guy Kawasaki 10-slide pitch (10/20/30). Ten slides, twenty minutes, thirty-point when you paste. Do not add an 11th slide.\n\n")
	b.WriteString("This pack does not send, email, or invent waitlist / MAU / burn. Missing facts stay [TO BE COMPLETED]. After Accept, export a PDF and `/file` it under this bet’s **product** folder. Paste into `/legal` if you need a claims review.\n\n")
	if aud := firstNonEmpty(d.Audience, latestLiveDeck(siblings, "audience").Body); aud != "" {
		b.WriteString("## Audience\n\n" + strings.TrimSpace(aud) + "\n\n")
	}
	if len(d.Slides) > 0 {
		b.WriteString(growth.SlidesMarkdown(d.Slides) + "\n\n")
	} else {
		writeDeckWalkSections(&b, d, siblings)
		if d.Body != "" {
			b.WriteString("## " + kind + "\n\n")
			b.WriteString(strings.TrimSpace(d.Body) + "\n\n")
		}
	}
	if d.Kind == "plan" && strings.TrimSpace(d.Body) != "" && len(d.Slides) > 0 {
		b.WriteString("## Business plan\n\n")
		b.WriteString(strings.TrimSpace(d.Body) + "\n\n")
	}
	return b.String()
}

func writeDeckWalkSections(b *strings.Builder, d GrowthDeck, siblings []GrowthDeck) {
	for _, id := range []string{"beats"} {
		if d.Kind == id {
			continue
		}
		sib := latestLiveDeck(siblings, id)
		if strings.TrimSpace(sib.Body) == "" {
			continue
		}
		b.WriteString("## " + lookupDeckSpec(id) + " (earlier step)\n\n")
		b.WriteString(strings.TrimSpace(sib.Body) + "\n\n")
	}
}

func lookupDeckSpec(id string) string {
	switch id {
	case "beats":
		return "Beats"
	case "audience":
		return "Audience"
	default:
		return id
	}
}

func (s *Server) growthPackDownload(w http.ResponseWriter, r *http.Request) {
	u := s.requireDesk(w, r, "growth")
	if u == nil {
		return
	}
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	d, err := s.store.GetGrowthDeck(r.Context(), id)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	p, _ := s.store.GetProject(r.Context(), d.ProjectID)
	sibs, _ := s.store.ListGrowthDecks(r.Context(), d.ProjectID)
	writeMarkdownFile(w, d.PackFilename(), growthPackMarkdown(d, p, sibs), true)
}

func (s *Server) growthPackPublic(w http.ResponseWriter, r *http.Request) {
	token := strings.TrimSpace(r.PathValue("token"))
	if token == "" {
		http.NotFound(w, r)
		return
	}
	d, err := s.store.GetGrowthDeckByPackToken(r.Context(), token)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	p, _ := s.store.GetProject(r.Context(), d.ProjectID)
	sibs, _ := s.store.ListGrowthDecks(r.Context(), d.ProjectID)
	writeMarkdownFile(w, d.PackFilename(), growthPackMarkdown(d, p, sibs), false)
}
