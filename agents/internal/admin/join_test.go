package admin

import (
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/samuel/ai-workers/agents/internal/acadchat"
)

func TestJoinRedirectsToLMS(t *testing.T) {
	s := &Server{}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/join/abc123", nil)
	req.SetPathValue("code", "abc123")
	s.joinGET(rec, req)
	if rec.Code != 302 {
		t.Fatal(rec.Code, rec.Body.String())
	}
	loc := rec.Header().Get("Location")
	if !strings.Contains(loc, "lms.themobileprof.com/join/abc123") {
		t.Fatal(loc)
	}
	if strings.Contains(strings.ToLower(loc), "workers.themobileprof.com") {
		t.Fatal(loc)
	}
}

func TestPersonalJoinURLIsLMS(t *testing.T) {
	s := &Server{}
	got := s.personalJoinURL("ab12cd")
	if got != "https://lms.themobileprof.com/join/ab12cd" {
		t.Fatal(got)
	}
	if !strings.HasPrefix(s.personalJoinURL(""), "https://lms.themobileprof.com/join") {
		t.Fatal(s.personalJoinURL(""))
	}
}

func TestDailyGroupTextNamesWinnerWithoutVendor(t *testing.T) {
	s := &Server{}
	got := s.dailyGroupText(nil, RewardDay{WinnerName: "Ada", Amount: 500}, true)
	if !strings.Contains(got, "Ada") || !strings.Contains(got, "₦500") || !strings.Contains(got, "/invite") {
		t.Fatal(got)
	}
	low := strings.ToLower(got)
	for _, bad := range []string{"clubkonnect", "n8n", "worker", "desk"} {
		if strings.Contains(low, bad) {
			t.Fatal(bad)
		}
	}
}

func TestDailyGroupTextRemindsAirtimeWhenNobodyWon(t *testing.T) {
	s := &Server{}
	got := s.dailyGroupText(nil, RewardDay{Amount: 500}, false)
	if !strings.Contains(got, "₦500") || !strings.Contains(got, "Nobody earned") {
		t.Fatal(got)
	}
	if !strings.Contains(got, "lesson or workshop") || !strings.Contains(got, "/invite") || !strings.Contains(got, "Talk in this class") {
		t.Fatal(got)
	}
	low := strings.ToLower(got)
	for _, bad := range []string{"clubkonnect", "n8n", "worker", "desk", "score"} {
		if strings.Contains(low, bad) {
			t.Fatal(bad)
		}
	}
}

func TestRewardReplies(t *testing.T) {
	if !rewardYes("YES") || !rewardSeen("seen") || rewardYes("08031234567") {
		t.Fatal("parse")
	}
}

func TestInviteTurnCopyIsStudentFacing(t *testing.T) {
	d := acadchat.Decide(acadchat.Inbound{ChatID: "1", FromID: "2", Text: "/start r_ab", Group: false})
	if d.Kind != acadchat.KindStart {
		t.Fatalf("%+v", d)
	}
}

func TestInviteShareTextIsTheLMSLink(t *testing.T) {
	s := &Server{}
	got := s.inviteShareText(CommunityStudent{ReferralCode: "ab12cd"})
	if !strings.Contains(got, "https://lms.themobileprof.com/join/ab12cd") {
		t.Fatal(got)
	}
	low := strings.ToLower(got)
	for _, bad := range []string{"say hi", "workers.themobileprof.com", "n8n", "clubkonnect"} {
		if strings.Contains(low, bad) {
			t.Fatal(bad)
		}
	}
}
