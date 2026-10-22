package admin

import (
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestLagosDelayOfficeHours(t *testing.T) {
	loc := lagosLocation()
	noon := time.Date(2026, 10, 22, 12, 0, 0, 0, loc)
	if got := LagosDelay(noon); got != 15*time.Minute {
		t.Fatalf("noon delay %s", got)
	}
	night := time.Date(2026, 10, 22, 21, 0, 0, 0, loc)
	if got := LagosDelay(night); got != 60*time.Minute {
		t.Fatalf("night delay %s", got)
	}
}

func TestBuildRollup(t *testing.T) {
	r := buildRollup(
		[]Project{{Name: "Academy", BurnNGN: "120000"}},
		[]WorkItem{{Status: "open"}, {Status: "queued"}, {Status: "done"}},
		[]FileFiling{{Status: "proposed"}, {Status: "accepted"}},
		[]DeskJob{{Status: "proposed"}, {Status: "accepted"}},
		[]Grant{{Status: "open"}, {Status: "done"}},
	)
	if r.OpenTasks != 1 || r.Queued != 1 || r.FilesOdd != 1 || r.FilesGo != 1 || r.JobsWait != 1 || r.Grants != 1 {
		t.Fatalf("%+v", r)
	}
	if r.Burn != "Academy: 120000" {
		t.Fatalf("burn %q", r.Burn)
	}
}

func TestJobAutoAt(t *testing.T) {
	at := time.Date(2026, 10, 22, 10, 0, 0, 0, time.UTC)
	j := DeskJob{Payload: map[string]any{"auto_at": at.Format(time.RFC3339)}}
	if !jobAutoAt(j).Equal(at) {
		t.Fatalf("auto_at %s", jobAutoAt(j))
	}
	if !jobAutoAt(DeskJob{}).IsZero() {
		t.Fatal("empty")
	}
}

func TestAdminNext(t *testing.T) {
	r, err := http.NewRequest("POST", "/admin/grants/1/done", strings.NewReader("next=/admin/projects/3?tab=card"))
	if err != nil {
		t.Fatal(err)
	}
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	got := adminNext(r, "/admin/grants?ok=done")
	if !strings.Contains(got, "/admin/projects/3") || !strings.Contains(got, "ok=done") {
		t.Fatalf("%s", got)
	}
	r2, _ := http.NewRequest("POST", "/admin/grants", nil)
	if adminNext(r2, "/admin/grants?ok=saved") != "/admin/grants?ok=saved" {
		t.Fatal("fallback")
	}
}
