package admin

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestInternalTokenOK(t *testing.T) {
	if InternalTokenOK("secret", "") {
		t.Fatal("empty configured token must never match")
	}
	if InternalTokenOK("", "secret") {
		t.Fatal("empty request token")
	}
	if InternalTokenOK("secret", "secret!") {
		t.Fatal("mismatch")
	}
	if !InternalTokenOK("secret", "secret") {
		t.Fatal("match")
	}
}

func TestInternalRequestToken(t *testing.T) {
	h := httptest.NewRequest(http.MethodGet, "/internal/v1/settings", nil)
	h.Header.Set("X-Internal-Token", " from-header ")
	if got := InternalRequestToken(h); got != "from-header" {
		t.Fatalf("header %q", got)
	}

	b := httptest.NewRequest(http.MethodGet, "/internal/v1/settings", nil)
	b.Header.Set("Authorization", "Bearer from-bearer")
	if got := InternalRequestToken(b); got != "from-bearer" {
		t.Fatalf("bearer %q", got)
	}

	both := httptest.NewRequest(http.MethodGet, "/internal/v1/settings", nil)
	both.Header.Set("X-Internal-Token", "header-wins")
	both.Header.Set("Authorization", "Bearer ignored")
	if got := InternalRequestToken(both); got != "header-wins" {
		t.Fatalf("prefer X-Internal-Token %q", got)
	}
}
