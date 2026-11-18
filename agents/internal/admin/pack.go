package admin

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"net/http"
)

const seoCompanyName = "TheMobileProf Technologies"

func newPackToken() (string, error) {
	buf := make([]byte, 24)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

func writeMarkdownFile(w http.ResponseWriter, name, body string, download bool) {
	w.Header().Set("Content-Type", "text/markdown; charset=utf-8")
	w.Header().Set("X-Robots-Tag", "noindex")
	w.Header().Set("Cache-Control", "no-store")
	disp := "inline"
	if download {
		disp = "attachment"
	}
	w.Header().Set("Content-Disposition", disp+`; filename="`+name+`"`)
	_, _ = w.Write([]byte(body))
}

func (s *Server) resolveSeoProject(ctx context.Context, body, sd map[string]any) (Project, error) {
	if id := int64Val(firstNonEmptyAny(body["project_id"], sd["project_id"])); id > 0 {
		return s.store.GetProject(ctx, id)
	}
	slug := firstNonEmpty(strVal(body["project_slug"]), strVal(sd["project_slug"]))
	if slug == "" {
		if nested, ok := sd["project"].(map[string]any); ok {
			slug = strVal(nested["slug"])
			if id := int64Val(nested["id"]); id > 0 {
				return s.store.GetProject(ctx, id)
			}
		}
		if nested, ok := body["project"].(map[string]any); ok {
			slug = strVal(nested["slug"])
			if id := int64Val(nested["id"]); id > 0 {
				return s.store.GetProject(ctx, id)
			}
		}
	}
	if slug == "" {
		return Project{}, errors.New("project is required")
	}
	return s.store.GetProjectBySlug(ctx, slug)
}
