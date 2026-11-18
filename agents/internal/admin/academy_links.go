package admin

import (
	"context"
	"os"
	"strings"

	"github.com/samuel/ai-workers/agents/internal/clubkonnect"
)

const (
	joinOriginDefault     = "https://lms.themobileprof.com"
	lmsRegisterDefault    = "https://lms.themobileprof.com/join"
	telegramInviteDefault = "https://t.me/+6lTlfZkJ0TU3ZGE0"
	academyBotDefault     = "themobileprof_academy_bot"
)

func (s *Server) setting(ctx context.Context, key, fallback string) string {
	if s == nil || s.store == nil {
		return fallback
	}
	all, err := s.store.Settings(ctx)
	if err != nil {
		return fallback
	}
	if v := strings.TrimSpace(all[key]); v != "" {
		return v
	}
	return fallback
}

func (s *Server) joinOrigin() string {
	if v := strings.TrimSpace(os.Getenv("JOIN_ORIGIN")); v != "" {
		return strings.TrimRight(v, "/")
	}
	return joinOriginDefault
}

func (s *Server) lmsRegisterURL(ctx context.Context) string {
	return s.setting(ctx, "community.lms_register", lmsRegisterDefault)
}

func (s *Server) telegramInviteURL(ctx context.Context) string {
	return s.setting(ctx, "community.telegram_invite", telegramInviteDefault)
}

func (s *Server) academyBotUsername(ctx context.Context) string {
	if u := s.setting(ctx, "community.telegram_bot", ""); u != "" {
		return strings.TrimPrefix(u, "@")
	}
	if s != nil && s.acadBot != nil {
		if u, err := s.acadBot.Username(ctx); err == nil && u != "" {
			if s.store != nil {
				_ = s.store.SetSetting(ctx, "community.telegram_bot", u)
			}
			return u
		}
	}
	return academyBotDefault
}

func (s *Server) personalJoinURL(code string) string {
	code = strings.TrimSpace(code)
	if code == "" {
		return s.joinOrigin() + "/join"
	}
	return s.joinOrigin() + "/join/" + code
}

func (s *Server) telegramStartURL(ctx context.Context, code string) string {
	u := s.academyBotUsername(ctx)
	if u == "" {
		return ""
	}
	code = strings.TrimSpace(code)
	if code == "" {
		return "https://t.me/" + u
	}
	return "https://t.me/" + u + "?start=r_" + code
}

func (s *Server) rewardNaira() int {
	return clubkonnect.RewardAmount()
}

func displayName(st CommunityStudent, fallback string) string {
	if n := strings.TrimSpace(st.Name); n != "" {
		return n
	}
	if n := strings.TrimSpace(st.Username); n != "" {
		return n
	}
	if fallback != "" {
		return fallback
	}
	return "friend"
}
