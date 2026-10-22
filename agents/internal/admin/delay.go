package admin

import (
	"strings"
	"time"
)

func lagosLocation() *time.Location {
	loc, err := time.LoadLocation("Africa/Lagos")
	if err != nil {
		return time.FixedZone("WAT", 3600)
	}
	return loc
}

// LagosDelay is 15 minutes during office hours, 60 minutes after hours.
func LagosDelay(now time.Time) time.Duration {
	t := now.In(lagosLocation())
	if t.Hour() >= 8 && t.Hour() < 20 {
		return 15 * time.Minute
	}
	return 60 * time.Minute
}

func jobAutoAt(j DeskJob) time.Time {
	raw := strings.TrimSpace(strVal(j.Payload["auto_at"]))
	if raw == "" {
		return time.Time{}
	}
	t, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		return time.Time{}
	}
	return t
}

func stampAutoAt(sd map[string]any, now time.Time) {
	if sd == nil {
		return
	}
	sd["auto_at"] = now.Add(LagosDelay(now)).UTC().Format(time.RFC3339)
}
