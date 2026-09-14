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

func claimedEnvs() map[string]bool {
	out := map[string]bool{}
	for _, it := range allIntegrations() {
		for _, e := range it.ComposeEnvs {
			out[e] = true
		}
	}
	return out
}

func claimedCreds() map[string]bool {
	out := map[string]bool{}
	for _, it := range allIntegrations() {
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
	claimed := claimedEnvs()
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
				t.Errorf("%s: %s is not listed in allIntegrations().ComposeEnvs — add a Tools row", rel, name)
			}
		}
	}
}

func TestIntegrationsCoverN8nCredentials(t *testing.T) {
	root := repoRoot(t)
	claimed := claimedCreds()
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
					t.Errorf("%s: n8n credential %q is not in allIntegrations().N8nCredNames", filepath.Base(path), c.Name)
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
	keys := catalogSettingKeys()
	for _, it := range allIntegrations() {
		for _, k := range it.DeskKeys {
			if !keys[k] {
				t.Errorf("%s desk key %s is not in settingCatalog()", it.ID, k)
			}
		}
	}
}

func TestIntegrationsPageParses(t *testing.T) {
	if _, err := New(nil, ""); err != nil {
		t.Fatal(err)
	}
	if n := len(allIntegrations()); n < 8 {
		t.Fatalf("expected a full vendor list, got %d", n)
	}
}
