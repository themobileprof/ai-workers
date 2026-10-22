package admin

import "strings"

type BoardRollup struct {
	OpenTasks int
	Queued    int
	FilesOdd  int
	FilesGo   int
	JobsWait  int
	Grants    int
	Burn      string
}

func buildRollup(projects []Project, items []WorkItem, filings []FileFiling, jobs []DeskJob, grants []Grant) BoardRollup {
	r := BoardRollup{}
	for _, it := range items {
		if it.Open() {
			r.OpenTasks++
		}
		if it.Queued() {
			r.Queued++
		}
	}
	for _, d := range filings {
		if d.Proposed() {
			r.FilesOdd++
		}
		if d.Waiting() {
			r.FilesGo++
		}
	}
	for _, j := range jobs {
		if j.Proposed() {
			r.JobsWait++
		}
	}
	for _, g := range grants {
		if g.Open() {
			r.Grants++
		}
	}
	var burns []string
	for _, p := range projects {
		if n := strings.TrimSpace(p.BurnNGN); n != "" {
			burns = append(burns, p.Name+": "+n)
		}
	}
	r.Burn = strings.Join(burns, " · ")
	return r
}
