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
