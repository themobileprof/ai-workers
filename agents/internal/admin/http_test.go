package admin

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSameOrigin(t *testing.T) {
	ok := httptest.NewRequest(http.MethodPost, "https://workers.themobileprof.com/admin/login", nil)
	ok.Host = "workers.themobileprof.com"
	ok.Header.Set("Origin", "https://workers.themobileprof.com")
	if rec := httptest.NewRecorder(); !sameOrigin(rec, ok) {
		t.Fatal("expected same origin")
	}

	bad := httptest.NewRequest(http.MethodPost, "https://workers.themobileprof.com/admin/login", nil)
	bad.Host = "workers.themobileprof.com"
	bad.Header.Set("Origin", "https://evil.example")
	rec := httptest.NewRecorder()
	if sameOrigin(rec, bad) {
		t.Fatal("cross origin must fail")
	}
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status %d", rec.Code)
	}

	ref := httptest.NewRequest(http.MethodPost, "https://workers.themobileprof.com/admin/logout", nil)
	ref.Host = "workers.themobileprof.com"
	ref.Header.Set("Referer", "https://workers.themobileprof.com/admin/")
	if rec := httptest.NewRecorder(); !sameOrigin(rec, ref) {
		t.Fatal("referer same host should pass")
	}
}

func TestTemplatesParse(t *testing.T) {
	if _, err := New(nil, ""); err != nil {
		t.Fatal(err)
	}
}

func TestHasStoredCap(t *testing.T) {
	u := User{Role: "owner", Caps: []string{"whatsapp_accounts"}}
	if !hasStoredCap(u, "whatsapp_accounts") || hasStoredCap(u, "web_admin") {
		t.Fatal("stored caps only")
	}
	if !HasCap(u, "web_admin") {
		t.Fatal("owner has all caps for auth")
	}
}

func TestRemovalBlocked(t *testing.T) {
	owner := User{ID: 1, Role: "owner"}
	bookkeeper := User{ID: 2, Role: "accounts"}
	if err := removalBlocked(&owner, owner, 1); err == nil {
		t.Fatal("must not remove self")
	}
	if err := removalBlocked(&owner, owner, 2); err == nil {
		t.Fatal("must not remove self even with another owner")
	}
	if err := removalBlocked(&bookkeeper, owner, 1); err == nil {
		t.Fatal("must not remove last owner")
	}
	if err := removalBlocked(&owner, bookkeeper, 1); err != nil {
		t.Fatal(err)
	}
	if err := lastOwnerLocked(owner, "viewer", true, 1); err == nil {
		t.Fatal("must not demote last owner")
	}
	if err := lastOwnerLocked(owner, "owner", false, 1); err == nil {
		t.Fatal("must not deactivate last owner")
	}
	if err := lastOwnerLocked(owner, "viewer", true, 2); err != nil {
		t.Fatal(err)
	}
}
