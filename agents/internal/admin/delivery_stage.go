package admin

import "strings"

// StageEngagement is how hard a bet may run. Academy is not special — the stamp is.
type StageEngagement struct {
	MaxOpenTasks int
	Weight       int
	Pings        bool
	IdeaTalk     bool
}

func EngagementFor(stage string) StageEngagement {
	switch strings.ToLower(strings.TrimSpace(stage)) {
	case "killed", "parked":
		return StageEngagement{}
	case "idea":
		return StageEngagement{MaxOpenTasks: 1, Weight: 1, Pings: false, IdeaTalk: true}
	case "testing":
		return StageEngagement{MaxOpenTasks: 3, Weight: 3, Pings: true, IdeaTalk: true}
	case "selling":
		return StageEngagement{MaxOpenTasks: 5, Weight: 5, Pings: true, IdeaTalk: true}
	case "fundable":
		return StageEngagement{MaxOpenTasks: 4, Weight: 4, Pings: true, IdeaTalk: true}
	default:
		return StageEngagement{MaxOpenTasks: 1, Weight: 1, IdeaTalk: true}
	}
}

func mandateProjectSlug(slug string) string {
	switch strings.ToLower(strings.TrimSpace(slug)) {
	case "lms", "academy-lms":
		return "academy"
	default:
		return strings.ToLower(strings.TrimSpace(slug))
	}
}

// PersonLoad is one human’s remaining hours this week.
type PersonLoad struct {
	UserID   int64
	Name     string
	Hours    float64
	Office   bool
	Projects map[int64]bool
}

// BetLoad is one bet’s remaining task slots after the stamp cap.
type BetLoad struct {
	ProjectID int64
	Slug      string
	Name      string
	Stage     string
	Weight    int
	Slots     int
}

// ProposedAssign is a slot the PM may fill — Go caps; the LLM only names the work.
type ProposedAssign struct {
	ProjectID int64
	Slug      string
	UserID    int64
	Hours     float64
}

func Apportion(people []PersonLoad, bets []BetLoad) []ProposedAssign {
	var out []ProposedAssign
	remain := make([]float64, len(people))
	for i, p := range people {
		if p.Hours > 0 {
			remain[i] = p.Hours
		}
	}
	type betState struct {
		BetLoad
		left int
	}
	var live []betState
	totalW := 0
	for _, b := range bets {
		if b.Weight <= 0 || b.Slots <= 0 {
			continue
		}
		live = append(live, betState{BetLoad: b, left: b.Slots})
		totalW += b.Weight
	}
	if totalW == 0 {
		return out
	}
	for i := range live {
		share := 0
		for _, p := range people {
			if p.Hours <= 0 {
				continue
			}
			if !p.Office && (p.Projects == nil || !p.Projects[live[i].ProjectID]) {
				continue
			}
			share++
		}
		if share == 0 {
			live[i].left = 0
		}
	}
	progress := true
	for progress {
		progress = false
		for i := range live {
			if live[i].left <= 0 {
				continue
			}
			best := -1
			for pi, p := range people {
				if remain[pi] < 0.5 {
					continue
				}
				if !p.Office && (p.Projects == nil || !p.Projects[live[i].ProjectID]) {
					continue
				}
				if best < 0 || remain[pi] > remain[best] {
					best = pi
				}
			}
			if best < 0 {
				continue
			}
			h := 1.0
			if remain[best] < h {
				h = remain[best]
			}
			if h < 0.5 {
				continue
			}
			out = append(out, ProposedAssign{
				ProjectID: live[i].ProjectID,
				Slug:      live[i].Slug,
				UserID:    people[best].UserID,
				Hours:     h,
			})
			remain[best] -= h
			live[i].left--
			progress = true
		}
	}
	return out
}

func canAssign(u User, projectID int64, members []ProjectMember) bool {
	if u.Role == "owner" || u.Role == "bdm" {
		return true
	}
	for _, m := range members {
		if m.UserID == u.ID {
			return true
		}
	}
	return false
}

func canSeeBet(u User, members []ProjectMember) bool {
	return canAssign(u, 0, members)
}
