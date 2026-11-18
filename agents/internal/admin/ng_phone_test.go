package admin

import "testing"

func TestNGPhoneLocal(t *testing.T) {
	if ngPhoneLocal("0803 123 4567") != "08031234567" {
		t.Fatal(ngPhoneLocal("0803 123 4567"))
	}
	if ngPhoneLocal("+2348031234567") != "08031234567" {
		t.Fatal(ngPhoneLocal("+2348031234567"))
	}
	if ngPhoneLocal("not a phone") != "" {
		t.Fatal("empty")
	}
}

func TestNGNetworkFromPrefix(t *testing.T) {
	if ngNetworkFromPrefix("08031234567") != "mtn" || ngNetworkCode("mtn") != "01" {
		t.Fatal("mtn")
	}
	if ngNetworkFromPrefix("07075240400") != "mtn" {
		t.Fatal("0707")
	}
	if ngNetworkFromPrefix("08021234567") != "airtel" || ngNetworkCode("airtel") != "04" {
		t.Fatal("airtel")
	}
	phone, net := parseNGPhoneNetwork("airtel 0802 111 2222")
	if phone != "08021112222" || net != "airtel" {
		t.Fatal(phone, net)
	}
}
