package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"h3yun-cli/internal/localstate"
)

func TestProfileSwitchDryRunBackupAndPrevious(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	initial := []byte(`{"current_profile":"prod","profiles":{"prod":{"engine_code_env":"PROD_CODE","engine_secret_env":"PROD_SECRET"},"test":{"engine_code_env":"TEST_CODE","engine_secret_env":"TEST_SECRET","read_only":true}}}`)
	if err := os.WriteFile(path, initial, 0600); err != nil {
		t.Fatal(err)
	}
	args := []string{"profile", "switch", "--config", path, "--name", "test"}
	code, result, raw := invoke(t, append(append([]string{}, args...), "--dry-run"), "", nil)
	if code != 0 || result["data"].(map[string]any)["would_change"] != true {
		t.Fatal(raw)
	}
	after, _ := os.ReadFile(path)
	entries, _ := os.ReadDir(dir)
	if !bytes.Equal(initial, after) || len(entries) != 1 {
		t.Fatal("dry-run changed files")
	}
	code, result, raw = invoke(t, args, "", nil)
	if code != 0 {
		t.Fatal(raw)
	}
	backup := result["data"].(map[string]any)["backup"].(string)
	b, err := os.ReadFile(backup)
	if err != nil || !bytes.Equal(b, initial) {
		t.Fatal("backup mismatch", err)
	}
	doc, err := loadConfig(path, func(string) string { return "" })
	if err != nil || doc.CurrentProfile != "test" || doc.PreviousProfile != "prod" || !doc.Profiles["test"].ReadOnly || doc.Profiles["prod"].EngineSecretEnv != "PROD_SECRET" {
		t.Fatal(doc, err)
	}
	code, _, raw = invoke(t, []string{"profile", "switch", "--name", "-"}, "", map[string]string{"H3YUN_CONFIG": path})
	if code != 0 {
		t.Fatal(raw)
	}
	doc, _ = loadConfig(path, func(string) string { return "" })
	if doc.CurrentProfile != "prod" || doc.PreviousProfile != "test" {
		t.Fatal(doc)
	}
	entries, _ = os.ReadDir(dir)
	code, result, raw = invoke(t, []string{"profile", "switch", "--name", "prod", "--config", path}, "", nil)
	newEntries, _ := os.ReadDir(dir)
	if code != 0 || result["data"].(map[string]any)["changed"] != false || len(entries) != len(newEntries) {
		t.Fatal(raw)
	}
}

func TestProfileSwitchRejectsMissingOverrideAndLock(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	initial := []byte(`{"profiles":{"test":{"engine_code_env":"C","engine_secret_env":"S"}}}`)
	if err := os.WriteFile(path, initial, 0600); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		args []string
		env  map[string]string
	}{
		{[]string{"profile", "switch", "--name", "test"}, nil},
		{[]string{"profile", "switch", "--name", "missing", "--config", path}, nil},
		{[]string{"profile", "switch", "--name", "-", "--config", path}, nil},
		{[]string{"profile", "switch", "--name", "test", "--config", path}, map[string]string{"H3YUN_PROFILE": "test"}},
	} {
		code, _, raw := invoke(t, tc.args, "", tc.env)
		if code != 3 {
			t.Fatal(raw)
		}
	}
	unlock, err := localstate.Lock(path)
	if err != nil {
		t.Fatal(err)
	}
	defer unlock()
	code, result, raw := invoke(t, []string{"profile", "switch", "--name", "test", "--config", path}, "", nil)
	if code != 3 || result["error"].(map[string]any)["code"] != "CONFIG_LOCKED" {
		t.Fatal(raw)
	}
	after, _ := os.ReadFile(path)
	if !bytes.Equal(initial, after) {
		t.Fatal("rejected operation modified file")
	}
}
