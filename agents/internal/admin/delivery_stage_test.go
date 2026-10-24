package admin

import (
	"strings"
	"testing"
)

func TestEngagementFor(t *testing.T) {
	if g := EngagementFor("parked"); g.MaxOpenTasks != 0 || g.Pings {
		t.Fatalf("parked %+v", g)
	}
	if g := EngagementFor("killed"); g.Weight != 0 {
		t.Fatalf("killed %+v", g)
	}
	if g := EngagementFor("idea"); g.MaxOpenTasks != 1 || g.Pings || !g.IdeaTalk {
		t.Fatalf("idea %+v", g)
	}
	if g := EngagementFor("testing"); g.MaxOpenTasks != 3 || !g.Pings {
		t.Fatalf("testing %+v", g)
	}
	if g := EngagementFor("selling"); g.MaxOpenTasks != 5 {
		t.Fatalf("selling %+v", g)
	}
	if g := EngagementFor("fundable"); g.MaxOpenTasks != 4 {
		t.Fatalf("fundable %+v", g)
	}
	if mandateProjectSlug("lms") != "academy" || mandateProjectSlug("homegauge") != "homegauge" {
		t.Fatal("mandate slug")
	}
}

func TestApportionRespectsStageAndPace(t *testing.T) {
	founder := PersonLoad{UserID: 1, Name: "Sam", Hours: 6, Office: true, Projects: map[int64]bool{}}
	bdm := PersonLoad{UserID: 2, Name: "Ada", Hours: 8, Office: true, Projects: map[int64]bool{}}
	parked := BetLoad{ProjectID: 1, Slug: "finchest", Name: "Finchest", Stage: "parked", Weight: 0, Slots: 0}
	idea := BetLoad{ProjectID: 2, Slug: "mechazone", Name: "Mechazone", Stage: "idea", Weight: 1, Slots: 1}
	selling := BetLoad{ProjectID: 3, Slug: "academy", Name: "Academy", Stage: "selling", Weight: 5, Slots: 5}
	got := Apportion([]PersonLoad{founder, bdm}, []BetLoad{parked, idea, selling})
	if len(got) == 0 {
		t.Fatal("expected slots")
	}
	var hours float64
	bySlug := map[string]int{}
	for _, a := range got {
		if a.Slug == "finchest" {
			t.Fatal("parked bet received work")
		}
		hours += a.Hours
		bySlug[a.Slug]++
	}
	if hours > 14.01 {
		t.Fatalf("over pace %v", hours)
	}
	if bySlug["academy"] < bySlug["mechazone"] {
		t.Fatalf("selling should get more than idea: %v", bySlug)
	}
	if bySlug["mechazone"] > 1 {
		t.Fatalf("idea cap %d", bySlug["mechazone"])
	}
}

func TestApportionSeatedOnly(t *testing.T) {
	face := PersonLoad{UserID: 9, Name: "Ada", Hours: 4, Office: false, Projects: map[int64]bool{7: true}}
	other := BetLoad{ProjectID: 3, Slug: "academy", Weight: 5, Slots: 5}
	mine := BetLoad{ProjectID: 7, Slug: "homegauge", Weight: 3, Slots: 3}
	got := Apportion([]PersonLoad{face}, []BetLoad{other, mine})
	for _, a := range got {
		if a.Slug != "homegauge" {
			t.Fatalf("leaked %s", a.Slug)
		}
		if a.UserID != 9 {
			t.Fatal(a.UserID)
		}
	}
	if len(got) == 0 {
		t.Fatal("home gauge slots")
	}
}

func TestCanAssignOfficeAndSeat(t *testing.T) {
	owner := User{ID: 1, Role: "owner"}
	face := User{ID: 2, Role: "cofounder"}
	members := []ProjectMember{{UserID: 2}}
	if !canAssign(owner, 3, nil) || !canSeeBet(owner, nil) {
		t.Fatal("owner")
	}
	if canAssign(face, 3, nil) || canSeeBet(face, nil) {
		t.Fatal("unseated face")
	}
	if !canAssign(face, 3, members) {
		t.Fatal("seated")
	}
}

func TestMorningPing(t *testing.T) {
	if morningShouldSpeak(nil, nil, "", true) {
		t.Fatal("office with empty CRM stays quiet")
	}
	if !morningShouldSpeak(nil, nil, "CRM due today", true) {
		t.Fatal("owner still gets CRM due")
	}
	if morningShouldSpeak(nil, nil, "CRM due today", false) {
		t.Fatal("handler does not get the CRM card")
	}
	if !morningShouldSpeak([]string{"Academy: editor"}, nil, "", false) {
		t.Fatal("handler with tasks still pings")
	}
	text := morningPingText("Sam", []string{"Academy: editor"}, []string{"HomeGauge training 28 Oct"}, "CRM due today (Lagos)\n• Apex — call")
	if !strings.Contains(text, "Academy: editor") || !strings.Contains(text, "Coming up") || !strings.Contains(text, "CRM due today") {
		t.Fatal(text)
	}
	if !strings.Contains(text, "does not") && !strings.Contains(text, "Reply /today") {
		t.Fatal("footer")
	}
}
