package admin

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"strconv"
	"strings"
	"time"
)

func officeDeliveryChannel(ch string) bool {
	switch strings.ToLower(strings.TrimSpace(ch)) {
	case "telegram", "notes", "fathom", "meeting":
		return true
	default:
		return false
	}
}

func slugFromMeetingTitle(title string, projects []Project) string {
	low := strings.ToLower(strings.TrimSpace(title))
	if low == "" {
		return ""
	}
	best := ""
	bestN := 0
	for _, p := range projects {
		for _, needle := range []string{p.Slug, p.Name} {
			n := strings.ToLower(strings.TrimSpace(needle))
			if len(n) < 3 {
				continue
			}
			if strings.Contains(low, n) && len(n) > bestN {
				best = p.Slug
				bestN = len(n)
			}
		}
	}
	return best
}

func fathomSecretBytes(secret string) ([]byte, error) {
	secret = strings.TrimSpace(secret)
	if secret == "" {
		return nil, errors.New("missing Fathom webhook secret")
	}
	if strings.HasPrefix(secret, "whsec_") {
		secret = strings.TrimPrefix(secret, "whsec_")
	}
	raw, err := base64.StdEncoding.DecodeString(secret)
	if err != nil || len(raw) == 0 {
		return []byte(secret), nil
	}
	return raw, nil
}

// VerifyFathomWebhook checks Standard Webhooks HMAC (webhook-id / timestamp / signature).
func VerifyFathomWebhook(secret, id, ts, sigHeader, rawBody string, now time.Time) error {
	id = strings.TrimSpace(id)
	ts = strings.TrimSpace(ts)
	sigHeader = strings.TrimSpace(sigHeader)
	if id == "" || ts == "" || sigHeader == "" {
		return errors.New("missing Fathom webhook headers")
	}
	sec, err := strconv.ParseInt(ts, 10, 64)
	if err != nil {
		return errors.New("bad Fathom webhook timestamp")
	}
	if now.IsZero() {
		now = time.Now()
	}
	delta := now.Unix() - sec
	if delta < 0 {
		delta = -delta
	}
	if delta > 300 {
		return errors.New("stale Fathom webhook")
	}
	key, err := fathomSecretBytes(secret)
	if err != nil {
		return err
	}
	mac := hmac.New(sha256.New, key)
	mac.Write([]byte(id + "." + ts + "." + rawBody))
	expected := []byte(base64.StdEncoding.EncodeToString(mac.Sum(nil)))
	for _, part := range strings.Fields(sigHeader) {
		sig := part
		if i := strings.Index(part, ","); i >= 0 {
			sig = part[i+1:]
		}
		got := []byte(strings.TrimSpace(sig))
		if len(got) == len(expected) && hmac.Equal(expected, got) {
			return nil
		}
	}
	return errors.New("bad Fathom webhook signature")
}
