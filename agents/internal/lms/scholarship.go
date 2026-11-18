package lms

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// ApproachingDays is how close a season end must be before the room gets a daily nudge.
const ApproachingDays = 21

const scholarshipMiniSlug = "scholarship"

var scholarshipRe = regexp.MustCompile(`(?i)\b(scholarship|unilag|altschool|discount code|course rep)\b`)

func LooksLikeScholarship(text string) bool {
	return scholarshipRe.MatchString(text)
}

// Season is the open Scholarship mini-course run. Aggregates only — never a named student.
type Season struct {
	Title           string
	Details         string
	StartsOn        time.Time
	EndsOn          time.Time
	DiscountName    string
	DiscountPercent int
	Openings        int
	Prize           string
	PathTitle       string
	Micros          []string
	CodesUsed       int
	MaxStudents     int
	CompletionPct   string
	TestsPassed     int
	AwardsGiven     int
	DaysLeft        int
	Approaching     bool
	Open            bool
}

func (s Season) HasDeadline() bool {
	return s.Open && !s.EndsOn.IsZero()
}

// Scholarship loads the public season plus admin code-use, lab ranking, and awards.
func (c *Client) Scholarship(ctx context.Context) (Season, error) {
	if c == nil {
		c = NewClient("")
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.scholarship.ok && time.Since(c.scholarship.at) < c.ttl {
		return c.scholarship.item, nil
	}
	got, err := c.fetchSeason(ctx)
	if err != nil {
		if c.scholarship.ok {
			return c.scholarship.item, nil
		}
		return Season{}, err
	}
	c.scholarship = cachedSeason{at: time.Now(), item: got, ok: got.Open}
	return got, nil
}

func (c *Client) fetchSeason(ctx context.Context) (Season, error) {
	mini, err := c.getJSON(ctx, "/api/mini-courses/"+scholarshipMiniSlug)
	if err != nil {
		return Season{}, err
	}
	s := parsePublicSeason(mini)
	if !s.Open {
		return s, nil
	}
	s.DaysLeft = daysUntil(s.EndsOn, time.Now())
	s.Approaching = s.DaysLeft >= 0 && s.DaysLeft <= ApproachingDays
	if !c.HasAuth() {
		return s, nil
	}
	rank, _ := c.authJSON(ctx, "/api/mini-courses/"+scholarshipMiniSlug+"/capstone/ranking")
	if rank != nil {
		if rs := asMap(rank["season"]); rs != nil {
			if s.DiscountName == "" {
				s.DiscountName = firstStr(rs, "discountName", "title")
			}
			if s.MaxStudents == 0 {
				s.MaxStudents = intish(rs["maxStudents"], rs["max_students"])
			}
			s.AwardsGiven += len(asList(rs, "winners"))
		}
		s.TestsPassed = len(asList(rank, "ranking"))
		s.AwardsGiven += len(asList(rank, "earlierWinners"))
		for _, past := range asList(rank, "pastSeasons") {
			s.AwardsGiven += len(asList(past, "winners"))
		}
	}
	code := ""
	if rank != nil {
		if rs := asMap(rank["season"]); rs != nil {
			code = firstStr(rs, "discountCode", "discount_code")
		}
	}
	if sp := c.matchSponsorship(ctx, s, code); sp != nil {
		s.CodesUsed = intish(sp["students_used"], sp["studentsUsed"], sp["usage_count"])
		if s.MaxStudents == 0 {
			s.MaxStudents = intish(sp["max_students"], sp["maxStudents"])
		}
		s.CompletionPct = percentString(firstAny(sp, "completion_rate", "completionRate"))
	}
	return s, nil
}

func parsePublicSeason(mini map[string]any) Season {
	s := Season{
		PathTitle: firstStr(mini, "title"),
		Open:      false,
	}
	for _, m := range asList(mini, "microCourses") {
		if t := firstStr(m, "title"); t != "" {
			s.Micros = append(s.Micros, t)
		}
	}
	raw := asMap(mini["season"])
	if raw == nil {
		return s
	}
	s.Open = true
	s.Title = firstStr(raw, "title", "discountName")
	s.Details = clip(plain(firstStr(raw, "details")), 800)
	s.DiscountName = firstStr(raw, "discountName", "title")
	s.DiscountPercent = intish(raw["discountPercent"], raw["discount_percent"])
	s.Openings = intish(raw["openings"])
	s.Prize = firstStr(raw, "prize")
	s.StartsOn = parseDay(firstStr(raw, "startsOn", "starts_on"))
	s.EndsOn = parseDay(firstStr(raw, "endsOn", "ends_on", "winnersAnnouncedOn"))
	return s
}

func (c *Client) matchSponsorship(ctx context.Context, season Season, code string) map[string]any {
	obj, err := c.authJSON(ctx, "/api/admin/sponsorships?status=active&limit=50")
	if err != nil {
		return nil
	}
	rows := asList(obj, "sponsorships")
	code = strings.ToUpper(strings.TrimSpace(code))
	if code != "" {
		for _, row := range rows {
			got := strings.ToUpper(firstStr(row, "discount_code", "discountCode"))
			if got != "" && got == code {
				return row
			}
		}
	}
	if !season.EndsOn.IsZero() {
		want := season.EndsOn.UTC().Format("2006-01-02")
		for _, row := range rows {
			end := parseDay(firstStr(row, "end_date", "endDate"))
			if !end.IsZero() && end.UTC().Format("2006-01-02") == want {
				return row
			}
		}
	}
	return nil
}

func parseDay(s string) time.Time {
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Time{}
	}
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t
	}
	if t, err := time.Parse("2006-01-02", s[:min(len(s), 10)]); err == nil {
		return t
	}
	return time.Time{}
}

func lagosDate(t time.Time) time.Time {
	loc, err := time.LoadLocation("Africa/Lagos")
	if err != nil {
		loc = time.FixedZone("WAT", 3600)
	}
	t = t.In(loc)
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, loc)
}

func daysUntil(end, now time.Time) int {
	if end.IsZero() {
		return -1
	}
	d := int(lagosDate(end).Sub(lagosDate(now)).Hours() / 24)
	return d
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// ScholarshipBlock is student-safe overlay. Never includes a discount code or a named student.
func ScholarshipBlock(s Season) string {
	if !s.HasDeadline() {
		return ""
	}
	var b strings.Builder
	b.WriteString("Live Scholarship season (LMS). Use these facts only. Never invent a discount code. Never name a student. Never quote whether a person is paid. Tell them to ask their course rep for the code.\n")
	if s.Title != "" {
		b.WriteString("- season: " + s.Title + "\n")
	}
	if s.PathTitle != "" {
		b.WriteString("- path: " + s.PathTitle + " at https://lms.themobileprof.com/mini-courses/scholarship\n")
	}
	if !s.EndsOn.IsZero() {
		b.WriteString("- ends: " + lagosDate(s.EndsOn).Format("2 January 2006") + "\n")
		if s.DaysLeft == 0 {
			b.WriteString("- days_left: last day\n")
		} else if s.DaysLeft > 0 {
			b.WriteString(fmt.Sprintf("- days_left: %d\n", s.DaysLeft))
		}
	}
	if s.Approaching {
		b.WriteString("- approaching_deadline: true. Daily nudge in the room. Encourage finishing the path and taking the lab so they can qualify.\n")
	}
	if s.DiscountName != "" {
		b.WriteString("- discount_name: " + s.DiscountName)
		if s.DiscountPercent > 0 {
			b.WriteString(fmt.Sprintf(" (%d%% off paid courses)", s.DiscountPercent))
		}
		b.WriteString(". Code is not in this overlay — course reps give it out.\n")
	}
	if s.Openings > 0 {
		b.WriteString(fmt.Sprintf("- openings: %d", s.Openings))
		if s.Prize != "" {
			b.WriteString(" places, prize " + s.Prize)
		}
		b.WriteString(" for students who finish at the top.\n")
	}
	if s.Details != "" {
		b.WriteString("- details: " + s.Details + "\n")
	}
	if len(s.Micros) > 0 {
		b.WriteString("- finish these micros then the scholarship lab test: " + strings.Join(s.Micros, "; ") + "\n")
	}
	b.WriteString("- room totals (aggregates only):\n")
	if s.MaxStudents > 0 {
		b.WriteString(fmt.Sprintf("  - discount seats used: %d of %d\n", s.CodesUsed, s.MaxStudents))
	} else {
		b.WriteString(fmt.Sprintf("  - discount seats used: %d\n", s.CodesUsed))
	}
	if s.CompletionPct != "" {
		b.WriteString("  - sponsored-student course completion: " + s.CompletionPct + "%\n")
	}
	b.WriteString(fmt.Sprintf("  - scholarship lab tests passed: %d\n", s.TestsPassed))
	b.WriteString(fmt.Sprintf("  - scholarships given: %d\n", s.AwardsGiven))
	return b.String()
}

func percentString(v any) string {
	switch t := v.(type) {
	case string:
		return strings.TrimSpace(strings.TrimSuffix(t, "%"))
	case float64:
		return strconv.FormatFloat(t, 'f', 2, 64)
	default:
		s := strings.TrimSpace(fmt.Sprint(v))
		if s == "" || s == "<nil>" {
			return ""
		}
		return s
	}
}
