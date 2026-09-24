package admin

import (
	"crypto/subtle"
	"net/http"
	"strings"
)

// InternalRequestToken reads X-Internal-Token, or Bearer from Authorization.
func InternalRequestToken(r *http.Request) string {
	if r == nil {
		return ""
	}
	got := strings.TrimSpace(r.Header.Get("X-Internal-Token"))
	if got == "" {
		if auth := r.Header.Get("Authorization"); strings.HasPrefix(auth, "Bearer ") {
			got = strings.TrimSpace(strings.TrimPrefix(auth, "Bearer "))
		}
	}
	return got
}

// InternalTokenOK is a constant-time compare. An empty configured token never matches.
func InternalTokenOK(got, want string) bool {
	if want == "" {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(got), []byte(want)) == 1
}
