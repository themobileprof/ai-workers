package admin

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// Integration is one third-party system this office talks to.
// The catalog lives in TOOLS.md at the repo root (GitHub-visible).
// integrations_test.go fails the build if compose/.env.example or n8n
// credentials mention a tool that is missing from that file.
type Integration struct {
	ID           string
	Name         string
	URL          string
	Role         string
	Where        string
	Secrets      string
	Status       string
	ComposeEnvs  []string
	N8nCredNames []string
	DeskKeys     []string
}

const (
	statusLive     = "live"
	statusOptional = "code-ready"
	statusHost     = "host"
)

func toolsMarkdownPath() string {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		return "TOOLS.md"
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", "..", "TOOLS.md"))
}

func allIntegrations() []Integration {
	got, err := loadIntegrations()
	if err != nil {
		return nil
	}
	return got
}

func loadIntegrations() ([]Integration, error) {
	raw, err := os.ReadFile(toolsMarkdownPath())
	if err != nil {
		return nil, err
	}
	return parseToolsMarkdown(string(raw))
}

func parseToolsMarkdown(raw string) ([]Integration, error) {
	status := ""
	var out []Integration
	var cur *Integration
	flush := func() error {
		if cur == nil {
			return nil
		}
		if cur.ID == "" {
			return fmt.Errorf("TOOLS.md: %q is missing **Id:**", cur.Name)
		}
		if cur.Status == "" {
			return fmt.Errorf("TOOLS.md: %s is missing a Live/Host/Code-ready section", cur.ID)
		}
		out = append(out, *cur)
		cur = nil
		return nil
	}
	for _, line := range strings.Split(raw, "\n") {
		s := strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(s, "## "):
			if err := flush(); err != nil {
				return nil, err
			}
			status = toolsStatusHeading(strings.TrimSpace(strings.TrimPrefix(s, "## ")))
		case strings.HasPrefix(s, "### "):
			if err := flush(); err != nil {
				return nil, err
			}
			cur = &Integration{
				Name:   strings.TrimSpace(strings.TrimPrefix(s, "### ")),
				Status: status,
			}
		case cur != nil && strings.HasPrefix(s, "- **"):
			key, val, ok := splitToolsField(s)
			if !ok {
				continue
			}
			switch key {
			case "id":
				if toks := toolsTokens(val); len(toks) == 1 {
					cur.ID = toks[0]
				} else {
					cur.ID = strings.Trim(val, "`")
				}
			case "url":
				cur.URL = val
			case "role":
				cur.Role = val
			case "where":
				cur.Where = val
			case "secrets":
				cur.Secrets = val
			case "env":
				cur.ComposeEnvs = toolsTokens(val)
			case "n8n credentials":
				cur.N8nCredNames = toolsTokens(val)
			case "desk keys":
				cur.DeskKeys = toolsTokens(val)
			}
		}
	}
	if err := flush(); err != nil {
		return nil, err
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("TOOLS.md: no vendors parsed")
	}
	return out, nil
}

func toolsStatusHeading(h string) string {
	switch strings.ToLower(h) {
	case "live":
		return statusLive
	case "host":
		return statusHost
	case "code-ready":
		return statusOptional
	default:
		return ""
	}
}

func splitToolsField(line string) (key, val string, ok bool) {
	line = strings.TrimPrefix(line, "- **")
	i := strings.Index(line, ":**")
	if i < 0 {
		return "", "", false
	}
	key = strings.ToLower(strings.TrimSpace(line[:i]))
	val = strings.TrimSpace(line[i+3:])
	return key, val, true
}

func toolsTokens(s string) []string {
	var out []string
	for {
		a := strings.Index(s, "`")
		if a < 0 {
			break
		}
		rest := s[a+1:]
		b := strings.Index(rest, "`")
		if b < 0 {
			break
		}
		tok := strings.TrimSpace(rest[:b])
		if tok != "" {
			out = append(out, tok)
		}
		s = rest[b+1:]
	}
	return out
}

func integrationByID(id string) (Integration, bool) {
	for _, it := range allIntegrations() {
		if it.ID == id {
			return it, true
		}
	}
	return Integration{}, false
}
