package admin

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/samuel/ai-workers/agents/internal/contract"
	"github.com/samuel/ai-workers/agents/internal/lms"
)

func (s *Server) EnrichCommunity(ctx context.Context, req contract.Request) contract.Request {
	if s == nil || s.store == nil {
		return req
	}
	data := contract.ContextObject(req.ContextData)
	if s.lms == nil {
		s.lms = lms.NewClient("")
	}

	tg := strings.TrimSpace(strVal(data["from"]))
	if tg == "" {
		tg = strings.TrimSpace(strVal(data["telegram_id"]))
	}
	var st CommunityStudent
	if tg != "" {
		if got, err := s.store.LookupCommunityStudent(ctx, tg); err == nil && got.TelegramID != "" {
			st = got
			data["student"] = map[string]any{
				"telegram_id":  st.TelegramID,
				"name":         st.Name,
				"username":     st.Username,
				"email":        st.Email,
				"course_title": st.CourseTitle,
				"lesson_note":  st.LessonNote,
				"last_ask":     st.LastAsk,
			}
		}
	}

	task := req.TaskDescription
	billing := boolVal(data["billing_ask"]) || boolVal(data["billing_in_group"])
	cat, err := s.lms.Get(ctx)
	if err == nil {
		matched := lms.Match(cat, task)
		if len(matched) == 0 && st.CourseTitle != "" {
			matched = lms.Match(cat, st.CourseTitle)
		}
		data["catalog_live"] = lms.PromptBlock(cat, matched)
		if cards := lms.Cards(matched); len(cards) > 0 {
			data["catalog_cards"] = cards
		}
		wantWorkshop := !billing && (lms.LooksLikeWorkshop(task) || st.LastAsk == "workshop")
		stepN := lms.WorkshopStepIndex(task)
		if stepN == 0 && st.LastAsk == "workshop" {
			stepN = lms.BareStepIndex(task)
		}
		lessonHint := task
		if st.LessonNote != "" {
			lessonHint = strings.TrimSpace(task + " " + st.LessonNote)
		}
		var lessonBits []string
		var lessonCards []map[string]any
		var workshopBits []string
		var focused []lms.Lesson
		var courseTitles []string
		for _, co := range matched {
			courseTitles = append(courseTitles, co.Title)
			lessons, lerr := s.lms.Lessons(ctx, co.ID)
			if lerr != nil || len(lessons) == 0 {
				continue
			}
			focus := lms.MatchLessons(lessons, lessonHint)
			if len(focus) > 0 {
				hydrated := s.lms.HydrateWorkshops(ctx, focus)
				byID := map[string]lms.Lesson{}
				for _, l := range hydrated {
					byID[l.ID] = l
				}
				for i := range lessons {
					if h, ok := byID[lessons[i].ID]; ok {
						lessons[i] = h
					}
				}
				for i := range focus {
					if h, ok := byID[focus[i].ID]; ok {
						focus[i] = h
					}
				}
			}
			if block := lms.LessonBlock(co.Title, lessons, focus); block != "" {
				lessonBits = append(lessonBits, block)
			}
			lessonCards = append(lessonCards, lms.LessonCards(lessons, focus)...)
			if wantWorkshop {
				for _, l := range focus {
					if ws := lms.WorkshopOverlay(l, stepN); ws != "" {
						workshopBits = append(workshopBits, ws)
					}
				}
			}
			focused = append(focused, focus...)
		}
		if len(lessonBits) > 0 {
			data["lesson_live"] = strings.Join(lessonBits, "\n")
			data["lessons_authed"] = true
		}
		if len(lessonCards) > 0 {
			data["lesson_cards"] = lessonCards
		}
		if len(workshopBits) > 0 {
			data["workshop_live"] = strings.Join(workshopBits, "\n")
		}
		gathering := !billing && (wantWorkshop || st.LastAsk == "lesson" || st.LastAsk == "workshop")
		if gathering {
			missing := "course"
			course := ""
			if len(courseTitles) > 0 {
				course = courseTitles[0]
				missing = "lesson"
			} else if st.CourseTitle != "" {
				course = st.CourseTitle
				missing = "lesson"
			}
			lesson := ""
			if len(focused) > 0 {
				lesson = focused[0].Title
				missing = ""
			} else if st.LessonNote != "" && course != "" {
				lesson = st.LessonNote
			}
			if wantWorkshop && missing == "" {
				needStep := false
				for _, l := range focused {
					if !l.Workshop.HasSteps() {
						continue
					}
					if len(l.Workshop.Steps) > 1 && (stepN <= 0 || stepN > len(l.Workshop.Steps)) {
						needStep = true
					}
				}
				if needStep {
					missing = "step"
				}
			}
			data["gather"] = map[string]any{
				"course":  course,
				"lesson":  lesson,
				"step":    stepN,
				"missing": missing,
			}
		}
	}
	if season, err := s.lms.Scholarship(ctx); err == nil && season.HasDeadline() {
		data["scholarship"] = map[string]any{
			"title":          season.Title,
			"ends":           season.EndsOn.UTC().Format("2 January 2006"),
			"days_left":      season.DaysLeft,
			"approaching":    season.Approaching,
			"discount_name":  season.DiscountName,
			"openings":       season.Openings,
			"codes_used":     season.CodesUsed,
			"tests_passed":   season.TestsPassed,
			"awards_given":   season.AwardsGiven,
			"completion_pct": season.CompletionPct,
			"path":           season.PathTitle,
		}
		data["scholarship_approaching"] = season.Approaching
		if block := lms.ScholarshipBlock(season); block != "" {
			data["scholarship_live"] = block
		}
	}
	if p, err := s.store.GetProjectBySlug(ctx, "academy"); err == nil {
		events, _ := s.store.ListProjectEvents(ctx, p.ID, true)
		var upcoming []map[string]any
		for _, e := range events {
			if !e.Planned() {
				continue
			}
			upcoming = append(upcoming, map[string]any{
				"title":    e.Title,
				"kind":     e.Kind,
				"when":     e.WhenLabel(),
				"venue":    e.Venue,
				"audience": e.Audience,
			})
		}
		if len(upcoming) > 0 {
			data["upcoming_trainings"] = upcoming
		}
	}
	raw, _ := json.Marshal(data)
	req.ContextData = raw
	return req
}

func (s *Server) applyCommunityStudent(ctx context.Context, fromID, username, name, chatID, email string, resp contract.Response) {
	if s == nil || s.store == nil || strings.TrimSpace(fromID) == "" {
		return
	}
	st := CommunityStudent{
		TelegramID: fromID,
		Username:   username,
		Name:       name,
		Email:      email,
		ChatID:     chatID,
	}
	if resp.StructuredData != nil {
		st.CourseTitle = strVal(resp.StructuredData["course_mentioned"])
		st.LessonNote = strVal(resp.StructuredData["lesson_mentioned"])
		switch strings.ToLower(strVal(resp.StructuredData["task_type"])) {
		case "workshop_help":
			st.LastAsk = "workshop"
		case "lesson_help":
			st.LastAsk = "lesson"
		case "billing":
			st.LastAsk = "billing"
		}
	}
	_, _ = s.store.UpsertCommunityStudent(ctx, st)
}
