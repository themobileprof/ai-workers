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

func meetingInviteeIdentity(v any) (name, email, phone string) {
	switch x := v.(type) {
	case string:
		for _, p := range strings.Fields(strings.TrimSpace(x)) {
			low := strings.ToLower(strings.Trim(p, "<>,;"))
			if strings.Contains(low, "@") {
				email = low
				continue
			}
			digits := normalizePhone(p)
			if len(digits) >= 8 {
				phone = digits
				continue
			}
			if name == "" {
				name = strings.Trim(p, ",;")
			} else {
				name += " " + strings.Trim(p, ",;")
			}
		}
	case map[string]any:
		name = firstNonEmpty(strVal(x["name"]), strVal(x["display_name"]))
		email = strings.ToLower(strVal(x["email"]))
		phone = normalizePhone(firstNonEmpty(strVal(x["phone"]), strVal(x["phone_number"])))
	}
	return strings.TrimSpace(name), strings.TrimSpace(email), strings.TrimSpace(phone)
}

func meetingInvitees(v any) []any {
	switch t := v.(type) {
	case []any:
		return t
	case []string:
		out := make([]any, 0, len(t))
		for _, s := range t {
			out = append(out, s)
		}
		return out
	case []map[string]any:
		out := make([]any, 0, len(t))
		for _, m := range t {
			out = append(out, m)
		}
		return out
	default:
		return nil
	}
}

func meetingProjectSlug(data map[string]any) string {
	if p, ok := data["project"].(map[string]any); ok {
		if s := strVal(p["slug"]); s != "" {
			return s
		}
	}
	return firstNonEmpty(strVal(data["project_slug"]), strVal(data["project"]))
}

func crmIsOfficePerson(users []User, name, email, phone string) bool {
	email = strings.ToLower(strings.TrimSpace(email))
	phone = normalizePhone(phone)
	name = strings.TrimSpace(name)
	for _, u := range users {
		if !u.Active {
			continue
		}
		if email != "" && strings.ToLower(strings.TrimSpace(u.Email)) == email {
			return true
		}
		if phone != "" && normalizePhone(u.Phone) != "" {
			staff := normalizePhone(u.Phone)
			if staff == phone || strings.HasSuffix(phone, staff) || strings.HasSuffix(staff, phone) {
				return true
			}
		}
		if name != "" && strings.EqualFold(strings.TrimSpace(u.Name), name) {
			return true
		}
	}
	return false
}

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
