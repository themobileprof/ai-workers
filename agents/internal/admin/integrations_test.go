package admin

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func repoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("caller")
	}
	root := filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", ".."))
	if _, err := os.Stat(filepath.Join(root, "docker-compose.yml")); err != nil {
		t.Fatalf("repo root %s: %v", root, err)
	}
	return root
}

func claimedEnvs(t *testing.T) map[string]bool {
	t.Helper()
	got, err := loadIntegrations()
	if err != nil {
		t.Fatal(err)
	}
	out := map[string]bool{}
	for _, it := range got {
		for _, e := range it.ComposeEnvs {
			out[e] = true
		}
	}
	return out
}

func claimedCreds(t *testing.T) map[string]bool {
	t.Helper()
	got, err := loadIntegrations()
	if err != nil {
		t.Fatal(err)
	}
	out := map[string]bool{}
	for _, it := range got {
		for _, n := range it.N8nCredNames {
			out[n] = true
		}
	}
	return out
}

func isVendorEnv(name string) bool {
	for _, p := range []string{"PAYSTACK_", "DEEPSEEK_", "GEMINI_", "OPENAI_", "ANTHROPIC_", "LLM_"} {
		if strings.HasPrefix(name, p) || name == strings.TrimSuffix(p, "_") {
			return true
		}
	}
	return name == "PAYSTACK_SECRET_KEY"
}

func TestIntegrationsCoverVendorEnv(t *testing.T) {
	root := repoRoot(t)
	claimed := claimedEnvs(t)
	for _, rel := range []string{"docker-compose.yml", ".env.example"} {
		raw, err := os.ReadFile(filepath.Join(root, rel))
		if err != nil {
			t.Fatal(err)
		}
		for _, line := range strings.Split(string(raw), "\n") {
			s := strings.TrimSpace(line)
			if s == "" || strings.HasPrefix(s, "#") {
				continue
			}
			name := ""
			if i := strings.Index(s, ":"); i > 0 && !strings.Contains(s[:i], " ") {
				name = strings.TrimSpace(s[:i])
			}
			if i := strings.Index(s, "="); i > 0 && !strings.Contains(s[:i], " ") {
				name = strings.TrimSpace(s[:i])
			}
			if !isVendorEnv(name) {
				continue
			}
			if !claimed[name] {
				t.Errorf("%s: %s is not listed in TOOLS.md Env — add a vendor section", rel, name)
			}
		}
	}
}

func TestIntegrationsCoverN8nCredentials(t *testing.T) {
	root := repoRoot(t)
	claimed := claimedCreds(t)
	err := filepath.WalkDir(filepath.Join(root, "n8n", "workflows"), func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".json") {
			return err
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		var wf struct {
			Nodes []struct {
				Credentials map[string]struct {
					Name string `json:"name"`
				} `json:"credentials"`
			} `json:"nodes"`
		}
		if err := json.Unmarshal(raw, &wf); err != nil {
			t.Errorf("%s: %v", path, err)
			return nil
		}
		for _, n := range wf.Nodes {
			for _, c := range n.Credentials {
				if c.Name == "" {
					continue
				}
				if !claimed[c.Name] {
					t.Errorf("%s: n8n credential %q is not in TOOLS.md n8n credentials", filepath.Base(path), c.Name)
				}
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestSettingCatalogMatchesDefaults(t *testing.T) {
	keys := catalogSettingKeys()
	defaults := defaultSettings()
	if len(keys) != len(defaults) {
		t.Fatalf("catalog %d keys vs defaults %d — keep them in lockstep", len(keys), len(defaults))
	}
	for k := range keys {
		if _, ok := defaults[k]; !ok {
			t.Errorf("catalog key %s missing from defaultSettings()", k)
		}
	}
}

func TestIntegrationDeskKeysAreCatalogued(t *testing.T) {
	got, err := loadIntegrations()
	if err != nil {
		t.Fatal(err)
	}
	keys := catalogSettingKeys()
	for _, it := range got {
		for _, k := range it.DeskKeys {
			if !keys[k] {
				t.Errorf("%s desk key %s is not in settingCatalog()", it.ID, k)
			}
		}
	}
}

func TestCaddyfileAllowlistsN8n(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join(repoRoot(t), "Caddyfile"))
	if err != nil {
		t.Fatal(err)
	}
	s := string(raw)
	if !strings.Contains(s, "handle @n8nui") {
		t.Fatal("Caddyfile must proxy the n8n editor through @n8nui, not a catch-all")
	}
	if strings.Contains(s, "handle {\n\t\treverse_proxy 127.0.0.1:5678") {
		t.Fatal("Caddyfile must not catch-all to n8n")
	}
}

func TestN8nDepartmentCallsSendInternalToken(t *testing.T) {
	root := repoRoot(t)
	err := filepath.WalkDir(filepath.Join(root, "n8n", "workflows"), func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".json") {
			return err
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		var wf struct {
			Nodes []struct {
				Name       string         `json:"name"`
				Type       string         `json:"type"`
				Parameters map[string]any `json:"parameters"`
			} `json:"nodes"`
		}
		if err := json.Unmarshal(raw, &wf); err != nil {
			t.Errorf("%s: %v", path, err)
			return nil
		}
		for _, n := range wf.Nodes {
			if n.Type != "n8n-nodes-base.httpRequest" || n.Parameters == nil {
				continue
			}
			url, _ := n.Parameters["url"].(string)
			if !strings.Contains(url, "/departments/") {
				continue
			}
			if n.Parameters["sendHeaders"] != true {
				t.Errorf("%s %s: POST %s missing sendHeaders for X-Internal-Token", filepath.Base(path), n.Name, url)
				continue
			}
			headers, _ := n.Parameters["headerParameters"].(map[string]any)
			params, _ := headers["parameters"].([]any)
			found := false
			for _, p := range params {
				m, _ := p.(map[string]any)
				if strings.EqualFold(fmtString(m["name"]), "X-Internal-Token") {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("%s %s: POST %s missing X-Internal-Token header", filepath.Base(path), n.Name, url)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func fmtString(v any) string {
	s, _ := v.(string)
	return s
}

func TestToolsMarkdownParses(t *testing.T) {
	got, err := loadIntegrations()
	if err != nil {
		t.Fatal(err)
	}
	if n := len(got); n < 8 {
		t.Fatalf("expected a full vendor list, got %d", n)
	}
	if _, ok := integrationByID("zoho-books"); !ok {
		t.Fatal("zoho-books")
	}
	if _, ok := integrationByID("whatsapp"); !ok {
		t.Fatal("whatsapp")
	}
}
