package admin

import (
	"crypto/subtle"
	"encoding/json"
	"net/http"
	"os"
	"strings"
)

func (s *Server) joinGET(w http.ResponseWriter, r *http.Request) {
	code := strings.ToLower(strings.TrimSpace(r.PathValue("code")))
	dest := s.personalJoinURL(code)
	if q := r.URL.RawQuery; q != "" {
		if strings.Contains(dest, "?") {
			dest += "&" + q
		} else {
			dest += "?" + q
		}
	}
	http.Redirect(w, r, dest, http.StatusFound)
}

func (s *Server) joinSyncOK(r *http.Request) bool {
	want := strings.TrimSpace(os.Getenv("JOIN_SYNC_SECRET"))
	got := strings.TrimSpace(r.Header.Get("X-Join-Token"))
	if want == "" || got == "" {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(want), []byte(got)) == 1
}

type joinHitBody struct {
	Code        string `json:"code"`
	Landing     string `json:"landing"`
	Referer     string `json:"referer"`
	UTMSource   string `json:"utm_source"`
	UTMMedium   string `json:"utm_medium"`
	UTMCampaign string `json:"utm_campaign"`
	UTMContent  string `json:"utm_content"`
	UTMTerm     string `json:"utm_term"`
	UserAgent   string `json:"user_agent"`
}

func (s *Server) joinHitPOST(w http.ResponseWriter, r *http.Request) {
	if !s.joinSyncOK(r) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	var body joinHitBody
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<16)).Decode(&body); err != nil {
		http.Error(w, "bad json", http.StatusBadRequest)
		return
	}
	if s.store != nil {
		s.store.RecordJoinHit(r.Context(), JoinHit{
			Code:        body.Code,
			Landing:     body.Landing,
			Referer:     body.Referer,
			UTMSource:   body.UTMSource,
			UTMMedium:   body.UTMMedium,
			UTMCampaign: body.UTMCampaign,
			UTMContent:  body.UTMContent,
			UTMTerm:     body.UTMTerm,
			UserAgent:   body.UserAgent,
		})
	}
	writeJSON(w, map[string]any{"ok": true})
}

func (s *Server) joinReferredPOST(w http.ResponseWriter, r *http.Request) {
	if !s.joinSyncOK(r) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	var body struct {
		Code  string `json:"code"`
		Email string `json:"email"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<14)).Decode(&body); err != nil {
		http.Error(w, "bad json", http.StatusBadRequest)
		return
	}
	ok := false
	if s.store != nil {
		ok = s.store.ConfirmLMSReferral(r.Context(), body.Code, body.Email)
	}
	writeJSON(w, map[string]any{"ok": ok})
}

func (s *Server) joinAliasPOST(w http.ResponseWriter, r *http.Request) {
	if !s.joinSyncOK(r) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	var body struct {
		Code  string `json:"code"`
		Email string `json:"email"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<14)).Decode(&body); err != nil {
		http.Error(w, "bad json", http.StatusBadRequest)
		return
	}
	if s.store != nil {
		_ = s.store.BindReferralCodeByEmail(r.Context(), body.Email, body.Code)
	}
	writeJSON(w, map[string]any{"ok": true})
}
