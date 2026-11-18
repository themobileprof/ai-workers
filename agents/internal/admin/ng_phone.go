package admin

import (
	"strings"
	"unicode"
)

// ClubKonnect network codes: MTN=01 Glo=02 9mobile/t2=03 Airtel=04
func ngNetworkCode(name string) string {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "mtn", "01":
		return "01"
	case "glo", "02":
		return "02"
	case "9mobile", "nine mobile", "9 mobile", "etisalat", "t2", "t2mobile", "03":
		return "03"
	case "airtel", "04":
		return "04"
	default:
		return ""
	}
}

func ngNetworkName(code string) string {
	switch ngNetworkCode(code) {
	case "01":
		return "MTN"
	case "02":
		return "Glo"
	case "03":
		return "9mobile"
	case "04":
		return "Airtel"
	default:
		return ""
	}
}

func ngPhoneLocal(raw string) string {
	var b strings.Builder
	for _, r := range raw {
		if unicode.IsDigit(r) {
			b.WriteRune(r)
		}
	}
	d := b.String()
	switch {
	case len(d) == 13 && strings.HasPrefix(d, "234"):
		return "0" + d[3:]
	case len(d) == 11 && strings.HasPrefix(d, "0"):
		return d
	case len(d) == 10:
		return "0" + d
	default:
		return ""
	}
}

func ngNetworkFromPrefix(phone string) string {
	p := ngPhoneLocal(phone)
	if len(p) < 4 {
		return ""
	}
	pre := p[:4]
	switch pre {
	case "0703", "0704", "0706", "0707", "0803", "0806", "0810", "0813", "0814", "0816", "0903", "0906", "0913", "0916":
		return "mtn"
	case "0705", "0805", "0807", "0811", "0815", "0905", "0915":
		return "glo"
	case "0809", "0817", "0818", "0908", "0909":
		return "9mobile"
	case "0701", "0708", "0802", "0808", "0812", "0901", "0902", "0904", "0907", "0912":
		return "airtel"
	default:
		return ""
	}
}

func parseNGPhoneNetwork(raw string) (phone, network string) {
	phone = ngPhoneLocal(raw)
	low := strings.ToLower(raw)
	for _, name := range []string{"9mobile", "etisalat", "airtel", "mtn", "glo"} {
		if strings.Contains(low, name) {
			network = name
			if name == "etisalat" {
				network = "9mobile"
			}
			break
		}
	}
	if network == "" {
		network = ngNetworkFromPrefix(phone)
	}
	return phone, network
}
