package admin

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"testing"
	"time"
)

func TestOfficeDeliveryChannel(t *testing.T) {
	if !officeDeliveryChannel("notes") || !officeDeliveryChannel("Fathom") || !officeDeliveryChannel("telegram") {
		t.Fatal("office channels")
	}
	if officeDeliveryChannel("whatsapp") || officeDeliveryChannel("email") {
		t.Fatal("customer channels stay person-by-phone")
	}
}

func TestSlugFromMeetingTitle(t *testing.T) {
	bets := []Project{
		{Slug: "academy", Name: "Academy LMS"},
		{Slug: "homegauge", Name: "HomeGauge"},
	}
	if got := slugFromMeetingTitle("Academy weekly stand-up", bets); got != "academy" {
		t.Fatalf("academy title: %q", got)
	}
	if got := slugFromMeetingTitle("homegauge / rates with a bank", bets); got != "homegauge" {
		t.Fatalf("slug in title: %q", got)
	}
	if got := slugFromMeetingTitle("Catch-up with a vendor", bets); got != "" {
		t.Fatalf("no bet in title: %q", got)
	}
}

func TestMeetingInviteeIdentity(t *testing.T) {
	name, email, phone := meetingInviteeIdentity("Chinedu chinedu@apex.ng")
	if name != "Chinedu" || email != "chinedu@apex.ng" || phone != "" {
		t.Fatalf("%q %q %q", name, email, phone)
	}
	name, email, phone = meetingInviteeIdentity(map[string]any{"name": "Ada", "email": "Ada@Apex.ng", "phone": "08031112222"})
	if name != "Ada" || email != "ada@apex.ng" || phone != "2348031112222" {
		t.Fatalf("map %q %q %q", name, email, phone)
	}
}

func TestCRMIsOfficePerson(t *testing.T) {
	staff := []User{
		{Name: "Samuel", Email: "samuel@themobileprof.com", Phone: "2348033954301", Active: true},
		{Name: "Biz", Email: "biz@themobileprof.com", Active: true},
	}
	if !crmIsOfficePerson(staff, "Samuel", "", "") || !crmIsOfficePerson(staff, "", "samuel@themobileprof.com", "") {
		t.Fatal("staff by name or email")
	}
	if crmIsOfficePerson(staff, "Chinedu", "chinedu@apex.ng", "") {
		t.Fatal("customer is not staff")
	}
}

func TestMeetingProjectSlug(t *testing.T) {
	if got := meetingProjectSlug(map[string]any{"project": map[string]any{"slug": "academy", "name": "Academy"}}); got != "academy" {
		t.Fatalf("%q", got)
	}
	if got := meetingProjectSlug(map[string]any{"meeting_title": "Catch-up"}); got != "" {
		t.Fatalf("no bet %q", got)
	}
}

func TestVerifyFathomWebhook(t *testing.T) {
	now := time.Unix(1_712_345_678, 0)
	body := `{"recording_id":123,"title":"Academy weekly"}`
	secret := "whsec_" + base64.StdEncoding.EncodeToString([]byte("fathom-test-secret"))
	id := "msg_test_1"
	ts := "1712345678"
	mac := hmac.New(sha256.New, []byte("fathom-test-secret"))
	mac.Write([]byte(id + "." + ts + "." + body))
	sig := "v1," + base64.StdEncoding.EncodeToString(mac.Sum(nil))
	if err := VerifyFathomWebhook(secret, id, ts, sig, body, now); err != nil {
		t.Fatal(err)
	}
	if err := VerifyFathomWebhook(secret, id, ts, "v1,AAAA", body, now); err == nil {
		t.Fatal("bad signature")
	}
	if err := VerifyFathomWebhook(secret, id, "100", sig, body, now); err == nil {
		t.Fatal("stale timestamp")
	}
}
