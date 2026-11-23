package admin

import "testing"

func TestPickWinnerSimpleScore(t *testing.T) {
	scores := []EngageScore{
		{TelegramID: "2", Name: "Bola", Community: 8, Academy: 2, Referral: 0, Total: 10},
		{TelegramID: "1", Name: "Ada", Community: 3, Academy: 4, Referral: 5, Total: 12},
		{TelegramID: "9", Name: "Staff", Community: 8, Academy: 8, Referral: 15, Total: 31},
	}
	got, ok := pickWinner(scores, map[string]bool{"9": true}, nil)
	if !ok || got.Name != "Ada" {
		t.Fatalf("%+v ok=%v", got, ok)
	}
}

func TestPickWinnerTieBreaksOnReferrals(t *testing.T) {
	scores := []EngageScore{
		{TelegramID: "2", Name: "Bola", Community: 4, Academy: 4, Referral: 0, Total: 8},
		{TelegramID: "1", Name: "Ada", Community: 2, Academy: 1, Referral: 5, Total: 8},
	}
	got, ok := pickWinner(scores, nil, nil)
	if !ok || got.Name != "Ada" {
		t.Fatalf("%+v", got)
	}
}

func TestPickWinnerRecentWinnerNeedsToLead(t *testing.T) {
	scores := []EngageScore{
		{TelegramID: "1", Name: "Ada", Community: 8, Academy: 4, Referral: 4, Total: 16},
		{TelegramID: "2", Name: "Bola", Community: 6, Academy: 2, Referral: 2, Total: 10},
	}
	got, ok := pickWinner(scores, nil, map[string]bool{"1": true})
	if !ok || got.Name != "Bola" {
		t.Fatalf("cooldown should prefer Bola: %+v", got)
	}
	got, ok = pickWinner(scores, nil, map[string]bool{"1": true, "2": true})
	if !ok || got.Name != "Ada" {
		t.Fatalf("both recent, raw lead still wins: %+v", got)
	}
	lead := []EngageScore{
		{TelegramID: "1", Name: "Ada", Community: 8, Academy: 8, Referral: 8, Total: 24},
		{TelegramID: "2", Name: "Bola", Community: 6, Academy: 2, Referral: 2, Total: 10},
	}
	got, ok = pickWinner(lead, nil, map[string]bool{"1": true})
	if !ok || got.Name != "Ada" {
		t.Fatalf("compelling lead still wins: %+v", got)
	}
}

func TestSkipCheckinBecauseQuiet(t *testing.T) {
	if !skipCheckinBecauseQuiet(nil, nil) {
		t.Fatal("empty class is quiet")
	}
	staffOnly := []EngageScore{{TelegramID: "9", Name: "Staff", Total: 12}}
	if !skipCheckinBecauseQuiet(staffOnly, map[string]bool{"9": true}) {
		t.Fatal("staff chatter is still a quiet class")
	}
	students := []EngageScore{{TelegramID: "1", Name: "Ada", Total: 3}}
	if skipCheckinBecauseQuiet(students, map[string]bool{"9": true}) {
		t.Fatal("a student day still gets the morning check-in")
	}
}

func TestReferralCodeStable(t *testing.T) {
	a := referralCodeFor("123")
	if a != referralCodeFor("123") || len(a) != 6 || a == referralCodeFor("124") {
		t.Fatal(a)
	}
}
