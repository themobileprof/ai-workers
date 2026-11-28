package opschat

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/samuel/ai-workers/agents/internal/departments"
	"github.com/samuel/ai-workers/agents/internal/llm"
)

// DeskOrigin is the signed-in company desk.
const DeskOrigin = "https://workers.themobileprof.com"

const (
	IntentWork = "work"
	IntentDesk = "desk"
	IntentGap  = "gap"
	IntentPlan = "plan"
)

// BetRef is enough project identity for sticky bet matching.
type BetRef struct {
	ID   int64
	Slug string
	Name string
}

// PlanStep is one desk in an ordered multi-desk reply.
type PlanStep struct {
	Department string
	Why        string
}

// Conduct is how the office conductor routes one plain-English ops message.
type Conduct struct {
	Intent      string
	Department  string
	ProjectSlug string
	Plan        []PlanStep
	DeskAction  string
	DeskPath    string
	DeskClick   string
	GapMissing  string
	GapDesk     string
	GapSpecimen string
	Note        string
}

const conductOverlay = `You are the office conductor for TheMobileProf Technologies (Lagos). One CAC company. Ops Telegram is 1:1 for staff.
Route the message. Do not do the desk's work. Chat never Accepts, never Places an ad, never flips stage, never amends Defaults.

Pick exactly one intent:
- work: one live worker can do this in chat now. Set department.
- plan: the ask spans two or more desks. List plan[] in order (max 4). Set department to the FIRST step only. The clerk will run only that first worker this turn.
- desk: the human must stamp on the website (Accept, Place ads, mark ads mature, Decide stage, People, Defaults, Board). Do not pretend chat can do it. Set desk_action and a short desk_click.
- gap: no live worker covers this. Say what is missing, which desk it would live on, and a one-paragraph specimen. Do not invent a live path.

Departments (for work/plan only):
- delivery — field reportage and this week's work
- growth — sales copy, pitch deck, Search ads pack (draft only — Place is desk)
- legal — NDA/contract draft or claims review
- files — data-room classify
- accounts — receipt/bill/invoice for Books
- internal-ops — grants, mixed ops
- crm — follow a named person on the conversion book
- product-dev — Card hypothesis/mission/evidence
- community — Academy student rooms (usually the other bot)
- hr — hiring copy (rare)

project_slug: one slug from the bet catalog in the user message, or empty. Prefer the sticky bet when the text does not name another.

structured_data MUST include:
- intent (work|plan|desk|gap)
- department (required for work/plan; empty for desk/gap)
- project_slug (string, may be empty)
Optional: plan[{department,why}], desk_action, desk_click, gap_missing, gap_desk, gap_specimen, note
output_text: unused, keep short.
status: success.`

// ConductRoute classifies intent, first desk, and optional bet.
func ConductRoute(ctx context.Context, c llm.Completer, task, replyTo, stickyBet string, bets []BetRef) (Conduct, error) {
	out := Conduct{Intent: IntentWork, Department: "internal-ops", ProjectSlug: stickyBet}
	if hint, ok := DeskOnly(task, bets, stickyBet); ok {
		return hint, nil
	}
	if c == nil {
		out.Department = "internal-ops"
		out.ProjectSlug = MatchBet(task, bets, stickyBet)
		return out, nil
	}
	user := task
	if strings.TrimSpace(replyTo) != "" {
		user = "They replied to:\n" + replyTo + "\n\nTheir text:\n" + task
	}
	if stickyBet != "" {
		user += "\n\nSticky bet slug: " + stickyBet
	}
	if cat := betCatalog(bets); cat != "" {
		user += "\n\nBet catalog (slug — name):\n" + cat
	}
	resp, err := departments.Run(ctx, c, conductOverlay, user)
	if err != nil {
		return out, err
	}
	sd := resp.StructuredData
	if sd == nil {
		sd = map[string]any{}
	}
	out.Intent = normalizeIntent(strVal(sd["intent"]))
	out.Department = strings.TrimSpace(strings.ToLower(strVal(sd["department"])))
	out.ProjectSlug = strings.TrimSpace(strings.ToLower(strVal(sd["project_slug"])))
	out.DeskAction = strings.TrimSpace(strings.ToLower(strVal(sd["desk_action"])))
	out.DeskClick = strings.TrimSpace(strVal(sd["desk_click"]))
	out.GapMissing = strings.TrimSpace(strVal(sd["gap_missing"]))
	out.GapDesk = strings.TrimSpace(strVal(sd["gap_desk"]))
	out.GapSpecimen = strings.TrimSpace(strVal(sd["gap_specimen"]))
	out.Note = strings.TrimSpace(strVal(sd["note"]))
	out.Plan = parsePlan(sd["plan"])
	if out.ProjectSlug == "" {
		out.ProjectSlug = MatchBet(task, bets, stickyBet)
	} else if !knownBet(out.ProjectSlug, bets) {
		out.ProjectSlug = MatchBet(task, bets, stickyBet)
	}
	switch out.Intent {
	case IntentDesk:
		if out.DeskAction == "" {
			out.DeskAction = "board"
		}
		out.DeskPath = DeskPath(out.DeskAction, findBet(out.ProjectSlug, bets))
		if out.DeskClick == "" {
			out.DeskClick = defaultDeskClick(out.DeskAction)
		}
		out.Department = ""
	case IntentGap:
		if out.GapMissing == "" {
			out.GapMissing = strings.TrimSpace(task)
		}
		if out.GapDesk == "" {
			out.GapDesk = "product-dev"
		}
		out.Department = ""
	case IntentPlan:
		if len(out.Plan) == 0 && KnownDepartment(out.Department) {
			out.Plan = []PlanStep{{Department: out.Department, Why: "first step"}}
		}
		if out.Department == "" && len(out.Plan) > 0 {
			out.Department = out.Plan[0].Department
		}
		if !KnownDepartment(out.Department) {
			out.Intent = IntentWork
			out.Department = "internal-ops"
		}
	default:
		out.Intent = IntentWork
		if !KnownDepartment(out.Department) {
			out.Department = strings.TrimSpace(strings.ToLower(strVal(sd["route_to"])))
		}
		if !KnownDepartment(out.Department) {
			out.Department = "internal-ops"
		}
	}
	return out, nil
}

// Classify returns exactly one department id (work path).
func Classify(ctx context.Context, c llm.Completer, task, replyTo string) (string, error) {
	got, err := ConductRoute(ctx, c, task, replyTo, "", nil)
	if err != nil {
		return "", err
	}
	if got.Department == "" {
		return "internal-ops", nil
	}
	return got.Department, nil
}

func normalizeIntent(s string) string {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case IntentDesk, "stamp", "website", "dashboard":
		return IntentDesk
	case IntentGap, "missing", "build":
		return IntentGap
	case IntentPlan, "multi", "several":
		return IntentPlan
	default:
		return IntentWork
	}
}

func parsePlan(v any) []PlanStep {
	raw, err := json.Marshal(v)
	if err != nil || len(raw) == 0 || string(raw) == "null" {
		return nil
	}
	var rows []map[string]any
	if err := json.Unmarshal(raw, &rows); err != nil {
		return nil
	}
	var out []PlanStep
	for _, row := range rows {
		dept := strings.TrimSpace(strings.ToLower(strVal(row["department"])))
		if !KnownDepartment(dept) {
			continue
		}
		out = append(out, PlanStep{Department: dept, Why: strings.TrimSpace(strVal(row["why"]))})
		if len(out) == 4 {
			break
		}
	}
	return out
}

func betCatalog(bets []BetRef) string {
	var lines []string
	for _, b := range bets {
		if b.Slug == "" {
			continue
		}
		name := b.Name
		if name == "" {
			name = b.Slug
		}
		lines = append(lines, "- "+b.Slug+" — "+name)
	}
	return strings.Join(lines, "\n")
}

func knownBet(slug string, bets []BetRef) bool {
	slug = strings.TrimSpace(strings.ToLower(slug))
	if slug == "" {
		return false
	}
	for _, b := range bets {
		if strings.EqualFold(b.Slug, slug) {
			return true
		}
	}
	return false
}

func findBet(slug string, bets []BetRef) BetRef {
	slug = strings.TrimSpace(strings.ToLower(slug))
	for _, b := range bets {
		if strings.EqualFold(b.Slug, slug) {
			return b
		}
	}
	return BetRef{}
}

// MatchBet finds a project slug named in text, else returns sticky.
func MatchBet(text string, bets []BetRef, sticky string) string {
	low := strings.ToLower(text)
	var best BetRef
	bestLen := 0
	for _, b := range bets {
		slug := strings.ToLower(strings.TrimSpace(b.Slug))
		name := strings.ToLower(strings.TrimSpace(b.Name))
		if slug != "" && strings.Contains(low, slug) && len(slug) > bestLen {
			best, bestLen = b, len(slug)
		}
		if name != "" && strings.Contains(low, name) && len(name) > bestLen {
			best, bestLen = b, len(name)
		}
	}
	if best.Slug != "" {
		return best.Slug
	}
	return strings.TrimSpace(strings.ToLower(sticky))
}

var (
	deskPlaceRe    = regexp.MustCompile(`(?i)\b((place|book)\b[\w\s]{0,40}\b(ads?|google|meta|facebook)\b|\b(ads?|google search|meta|facebook)\b[\w\s]{0,40}\b(place|book)\b)`)
	deskMatureRe   = regexp.MustCompile(`(?i)\b(mark\b[\w\s]{0,40}\bmature\b|ads\s+spend\s+mature|\bmature\s+(the\s+)?(ads|spend|card)\b)`)
	deskAcceptRe   = regexp.MustCompile(`(?i)\b(accept|turn away)\b[\w\s]{0,30}\b(job|pack|draft|proposal|ads|deck)\b|\baccept\s+on\s+(growth|legal|delivery|product|the desk)\b`)
	deskDecideRe   = regexp.MustCompile(`(?i)\b(change|set|stamp)\s+(the\s+)?stage\b|\bopen\s+decide\b|\b(go|pivot|kill)\s+the\s+bet\b`)
	deskCardRe     = regexp.MustCompile(`(?i)\b(edit|open)\s+(the\s+)?card\b|\bthis\s+week'?s\s+(mission|line)\b`)
	deskPeopleRe   = regexp.MustCompile(`(?i)\b(open|add)\s+people\b|\bseat\s+(them|someone|a\s+cofounder|an\s+assistant)\b`)
	deskDefaultsRe = regexp.MustCompile(`(?i)\b(open|amend)\s+defaults\b|\bzoho\s+ids\b|\bads\s+ceiling\b`)
	deskBoardRe    = regexp.MustCompile(`(?i)\bopen\s+board\b|\bshow\s+(me\s+)?(the\s+)?board\b`)
)

// DeskOnly detects website-only stamps without calling a worker.
func DeskOnly(text string, bets []BetRef, sticky string) (Conduct, bool) {
	text = strings.TrimSpace(text)
	if text == "" {
		return Conduct{}, false
	}
	slug := MatchBet(text, bets, sticky)
	bet := findBet(slug, bets)
	switch {
	case deskPlaceRe.MatchString(text):
		return deskConduct("place", bet, "Open Growth and click Place on Google Search or Meta — chat never Places."), true
	case deskMatureRe.MatchString(text):
		return deskConduct("mature", bet, "On the project Card, set Ads spend to mature after winning Search segments are listed."), true
	case deskAcceptRe.MatchString(text):
		return deskConduct("accept", bet, "Open the desk job and click Accept or Turn away. Chat never Accepts."), true
	case deskDecideRe.MatchString(text):
		return deskConduct("decide", bet, "Open Decide on the project and Accept the stage. Chat never stamps GO, PIVOT, or KILL."), true
	case deskCardRe.MatchString(text):
		return deskConduct("card", bet, "Edit the Card on the project page."), true
	case deskPeopleRe.MatchString(text):
		return deskConduct("people", bet, "Open People, then seat them on the project Face tab."), true
	case deskDefaultsRe.MatchString(text):
		return deskConduct("defaults", bet, "Amend Defaults on the desk. Chat cannot change vendor ids."), true
	case deskBoardRe.MatchString(text):
		return deskConduct("board", bet, "Open Board for every bet."), true
	default:
		return Conduct{}, false
	}
}

func deskConduct(action string, bet BetRef, click string) Conduct {
	return Conduct{
		Intent:      IntentDesk,
		ProjectSlug: bet.Slug,
		DeskAction:  action,
		DeskPath:    DeskPath(action, bet),
		DeskClick:   click,
	}
}

// DeskPath is the deep link for a desk-only action.
func DeskPath(action string, bet BetRef) string {
	base := DeskOrigin
	id := bet.ID
	switch strings.ToLower(strings.TrimSpace(action)) {
	case "place", "growth", "ads":
		if id > 0 {
			return fmt.Sprintf("%s/admin/growth?project_id=%d", base, id)
		}
		return base + "/admin/growth"
	case "mature", "card":
		if id > 0 {
			return fmt.Sprintf("%s/admin/projects/%d?tab=card", base, id)
		}
		return base + "/admin/projects"
	case "decide":
		if id > 0 {
			return fmt.Sprintf("%s/admin/projects/%d?tab=decide", base, id)
		}
		return base + "/admin/projects"
	case "accept", "job":
		return base + "/admin/desks"
	case "delivery":
		return base + "/admin/delivery"
	case "crm":
		return base + "/admin/crm"
	case "people":
		return base + "/admin/users"
	case "defaults", "settings":
		return base + "/admin/settings"
	case "board", "home":
		return base + "/admin/"
	case "legal":
		return base + "/admin/legal"
	default:
		if id > 0 {
			return fmt.Sprintf("%s/admin/projects/%d", base, id)
		}
		return base + "/admin/"
	}
}

func defaultDeskClick(action string) string {
	switch action {
	case "place":
		return "Click Place on Google Search or Meta. Do not also book the same campaign in the Ads UI."
	case "mature":
		return "Set Ads spend to mature on the Card when winning Search segments are ready."
	case "accept":
		return "Click Accept or Turn away on the job."
	case "decide":
		return "Accept the stage on Decide."
	case "people":
		return "Add or amend the People card, then seat them on Face."
	case "defaults":
		return "Amend the Defaults group you need."
	default:
		return "Open that page on the desk. Chat never stamps."
	}
}

// FormatDeskReply is the Telegram text for a desk-only intent.
func FormatDeskReply(c Conduct) string {
	var b strings.Builder
	b.WriteString("That stays on the desk — chat never stamps it.\n\n")
	if c.DeskClick != "" {
		b.WriteString(c.DeskClick)
		b.WriteString("\n")
	}
	if c.ProjectSlug != "" {
		b.WriteString("Bet: ")
		b.WriteString(c.ProjectSlug)
		b.WriteString("\n")
	}
	if c.DeskPath != "" {
		b.WriteString("\n")
		b.WriteString(c.DeskPath)
	}
	return strings.TrimSpace(b.String())
}

// FormatGapReply is the Telegram text when no worker covers the ask.
func FormatGapReply(c Conduct) string {
	var b strings.Builder
	b.WriteString("That is not a live worker yet.\n\n")
	if c.GapMissing != "" {
		b.WriteString("Missing: ")
		b.WriteString(c.GapMissing)
		b.WriteString("\n")
	}
	if c.GapDesk != "" {
		b.WriteString("Would live on: ")
		b.WriteString(c.GapDesk)
		b.WriteString("\n")
	}
	if c.GapSpecimen != "" {
		b.WriteString("\nSpecimen:\n")
		b.WriteString(c.GapSpecimen)
	}
	b.WriteString("\n\nI filed this as a guide note. Chat will not pretend the path exists.")
	return strings.TrimSpace(b.String())
}

func strVal(v any) string {
	switch t := v.(type) {
	case string:
		return strings.TrimSpace(t)
	case []any:
		if len(t) == 0 {
			return ""
		}
		s, _ := t[0].(string)
		return strings.TrimSpace(s)
	default:
		b, _ := json.Marshal(v)
		return strings.Trim(string(b), `"`)
	}
}

// FormatPlanPreamble lists the ordered desks, then says which one runs now.
func FormatPlanPreamble(c Conduct) string {
	if len(c.Plan) == 0 {
		return strings.TrimSpace(c.Note)
	}
	var b strings.Builder
	b.WriteString("This spans more than one desk. Order:\n")
	for i, step := range c.Plan {
		b.WriteString(fmt.Sprintf("%d. %s", i+1, step.Department))
		if step.Why != "" {
			b.WriteString(" — ")
			b.WriteString(step.Why)
		}
		b.WriteByte('\n')
	}
	b.WriteString("\nRunning only the first now (")
	b.WriteString(c.Department)
	b.WriteString("). Say when you want the next.")
	if c.Note != "" {
		b.WriteString("\n")
		b.WriteString(c.Note)
	}
	return strings.TrimSpace(b.String())
}
