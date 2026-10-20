package cli

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/cedric123123/h3yun-cli/internal/auth"
)

func TestProgressiveSchema(t *testing.T) {
	code, result, _ := invoke(t, []string{"schema"}, "", nil)
	root := result["data"].(map[string]any)
	if code != 0 || root["groups"] == nil || root["commands"] != nil {
		t.Fatal(result)
	}
	for _, spec := range commands() {
		code, result, raw := invoke(t, []string{"schema", "--cli-path", spec.Name, "--compact", "-f", "json"}, "", nil)
		if code != 0 {
			t.Fatal(raw)
		}
		leaf := result["data"].(map[string]any)
		params := leaf["parameters"].(map[string]any)
		for _, f := range spec.Flags {
			if params[f.Name] == nil {
				t.Fatal("schema omitted live flag", f.Name)
			}
		}
		if spec.Name == "records delete" && leaf["confirmation"] != "explicit_yes" {
			t.Fatal(leaf)
		}
	}
	code, result, raw := invoke(t, []string{"schema", "records", "--compact"}, "", nil)
	if code != 0 || len(result["data"].(map[string]any)["commands"].([]any)) != 7 {
		t.Fatal(raw)
	}
	code, _, raw = invoke(t, []string{"schema", "--all"}, "", nil)
	if code != 0 {
		t.Fatal(raw)
	}
}

func TestGlobalFlagsAndProjection(t *testing.T) {
	code, result, raw := invoke(t, []string{"--format", "pretty", "records", "--timeout", "20s", "get", "--schema", "s", "--id", "id", "--dry-run"}, "", nil)
	if code != 0 || result["ok"] != true || !strings.Contains(raw, "\n  ") {
		t.Fatal(raw)
	}
	code, result, raw = invoke(t, []string{"schema", "records get", "--fields", "parameters,effect"}, "", nil)
	if code != 0 || len(result["data"].(map[string]any)) != 2 || result["meta"] == nil {
		t.Fatal(raw)
	}
	code, _, _ = invoke(t, []string{"version", "--format", "csv"}, "", nil)
	if code != 2 {
		t.Fatal("unsupported format accepted")
	}
	code, _, _ = invoke(t, []string{"records", "create", "--schema", "s", "--data", "--profile", "--dry-run"}, "", nil)
	if code != 2 {
		t.Fatal("leaf value parsed as global flag")
	}
}

func TestProfilesIsolateEndpointCredentialsAndReadOnly(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	b := []byte(`{"profiles":{"prod":{"base_url":"https://prod.example","read_only":true,"engine_code_env":"PROD_CODE","engine_secret_env":"PROD_KEY"},"test":{"base_url":"https://test.example","engine_code_env":"TEST_CODE","engine_secret_env":"TEST_KEY"}}}`)
	if err := os.WriteFile(path, b, 0600); err != nil {
		t.Fatal(err)
	}
	env := map[string]string{"H3YUN_CONFIG": path, "H3YUN_BASE_URL": "https://wrong.example", "H3YUN_ENGINE_SECRET": "wrong-key", "H3YUN_READ_ONLY": "false"}
	code, result, raw := invoke(t, []string{"profile", "list"}, "", env)
	if code != 0 || len(result["data"].(map[string]any)["profiles"].([]any)) != 2 {
		t.Fatal(raw)
	}
	code, _, raw = invoke(t, []string{"profile", "show"}, "", env)
	if code != 3 || !strings.Contains(raw, "PROFILE_REQUIRED") {
		t.Fatal(raw)
	}
	env["H3YUN_PROFILE"] = "prod"
	s, cfg, err := settings("", func(k string) string { return env[k] })
	if err != nil || cfg.BaseURL != "https://prod.example" || !s.ReadOnly || cfg.EngineSecret != "" {
		t.Fatal(s, err)
	}
	code, _, raw = invoke(t, []string{"records", "create", "--schema", "s", "--data", "{}"}, "", env)
	if code != 3 || !strings.Contains(raw, "READ_ONLY") {
		t.Fatal(raw)
	}
	code, result, raw = invoke(t, []string{"--profile", "test", "profile", "show"}, "", env)
	if code != 0 || result["data"].(map[string]any)["base_url"] != "https://test.example" {
		t.Fatal(raw)
	}
}

func TestOAuthCachedBearerAndRedaction(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer test-session-token" || r.Header.Get("EngineSecret") != "" {
			t.Error("wrong authentication mode")
		}
		fmt.Fprint(w, `{"Successful":false,"ErrorMessage":"test-session-token"}`)
	}))
	defer server.Close()
	dir := t.TempDir()
	config := filepath.Join(dir, "config.json")
	store := filepath.Join(dir, "auth")
	oauth := auth.Config{AuthorizeURL: server.URL + "/authorize", TokenURL: server.URL + "/token", ClientID: "cli"}
	doc := ConfigFile{CurrentProfile: "dev", Profiles: map[string]Settings{"dev": {BaseURL: server.URL, AllowHTTP: true, OAuth: &oauth}}}
	b, _ := json.Marshal(doc)
	if err := os.WriteFile(config, b, 0600); err != nil {
		t.Fatal(err)
	}
	session := auth.Session{AccessToken: "test-session-token", ExpiresAt: time.Now().Add(time.Hour), Binding: auth.Binding("dev", server.URL, oauth)}
	if err := auth.Save(store, session); err != nil {
		t.Fatal(err)
	}
	env := map[string]string{"H3YUN_CONFIG": config, "H3YUN_AUTH_DIR": store}
	code, _, raw := invoke(t, []string{"records", "get", "--schema", "s", "--id", "id"}, "", env)
	if code != 5 || strings.Contains(raw, "test-session-token") || !strings.Contains(raw, "REDACTED") {
		t.Fatal(raw)
	}
	code, result, raw := invoke(t, []string{"auth", "status"}, "", env)
	if code != 0 || result["data"].(map[string]any)["authenticated"] != true {
		t.Fatal(raw)
	}
	code, _, raw = invoke(t, []string{"auth", "login", "--dry-run"}, "", env)
	if code != 0 || strings.Contains(raw, "test-session-token") {
		t.Fatal(raw)
	}
	code, _, raw = invoke(t, []string{"auth", "logout"}, "", env)
	if code != 0 {
		t.Fatal(raw)
	}
	code, result, raw = invoke(t, []string{"auth", "status"}, "", env)
	if code != 0 || result["data"].(map[string]any)["authenticated"] != false {
		t.Fatal(raw)
	}
}

func TestMissingOAuthDoesNotPretendToLogin(t *testing.T) {
	code, _, raw := invoke(t, []string{"auth", "login"}, "", nil)
	if code != 3 || !strings.Contains(raw, "AUTH_NOT_CONFIGURED") {
		t.Fatal(raw)
	}
	code, result, _ := invoke(t, []string{"records", "lst"}, "", nil)
	if code != 2 || result["error"].(map[string]any)["actions"] == nil {
		t.Fatal("missing discovery suggestions")
	}
}

func TestPublicCloudDefault(t *testing.T) {
	s, cfg, err := settings("", func(string) string { return "" })
	if err != nil || s.BaseURL != "https://www.h3yun.com" || cfg.BaseURL != s.BaseURL {
		t.Fatal(s, err)
	}
	code, result, raw := invoke(t, []string{"records", "get", "--schema", "s", "--id", "i", "--dry-run"}, "", nil)
	if code != 0 || result["data"].(map[string]any)["url"] != "https://www.h3yun.com/OpenApi/Invoke" {
		t.Fatal(raw)
	}
}
