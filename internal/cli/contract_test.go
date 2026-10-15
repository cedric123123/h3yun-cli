package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDocumentedTimeoutBounds(t *testing.T) {
	for _, tc := range []struct {
		timeout string
		want    int
	}{{"60s", 0}, {"1m", 0}, {"1ms", 0}, {"60.001s", 3}, {"10m", 3}, {"0s", 3}, {"-1s", 3}, {"invalid", 3}} {
		for _, fromEnv := range []bool{false, true} {
			args := []string{"records", "get", "--schema", "test", "--id", "test", "--dry-run"}
			env := map[string]string{}
			if fromEnv {
				env["H3YUN_TIMEOUT"] = tc.timeout
			} else {
				args = append(args, "--timeout", tc.timeout)
			}
			code, _, raw := invoke(t, args, "", env)
			if code != tc.want {
				t.Fatalf("timeout %s, env %v: %d %s", tc.timeout, fromEnv, code, raw)
			}
		}
	}
	config := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(config, []byte(`{"timeout":"61s"}`), 0600); err != nil {
		t.Fatal(err)
	}
	code, _, raw := invoke(t, []string{"doctor", "--config", config, "--dry-run"}, "", nil)
	if code != 3 {
		t.Fatal(raw)
	}
}

func TestContractSourcesAndSubmissionSemantics(t *testing.T) {
	for _, command := range []string{"records get", "records list", "records create", "records create-many", "records update", "records delete", "files upload", "files download", "custom invoke"} {
		code, result, raw := invoke(t, []string{"schema", command, "--compact"}, "", nil)
		if code != 0 {
			t.Fatal(raw)
		}
		contract := result["data"].(map[string]any)["api_contract"].(map[string]any)
		if contract["max_timeout_seconds"] != json.Number("60") || contract["live_verified"] != false || !strings.HasPrefix(contract["source"].(string), integrationDoc) {
			t.Fatal(raw)
		}
	}
	single := commandContract("records create")["submit_semantics"].(map[string]string)
	batch := commandContract("records create-many")["submit_semantics"].(map[string]string)
	if !strings.Contains(single["true"], "下一审批节点") || !strings.Contains(batch["true"], "不创建进行中") {
		t.Fatal(single, batch)
	}
	if commandContract("auth login") != nil {
		t.Fatal("invented public auth API")
	}
}

func TestDocumentedWarningsKeepRequestUnchanged(t *testing.T) {
	for _, tc := range []struct{ command, data, expected string }{
		{"create", `{"field":"value"}`, "下一审批节点"},
		{"create-many", `[{"field":"value"}]`, "不能当作批量发起审批"},
	} {
		code, result, raw := invoke(t, []string{"records", tc.command, "--schema", "test", "--data", tc.data, "--submit", "--dry-run"}, "", nil)
		if code != 0 || !strings.Contains(raw, tc.expected) {
			t.Fatal(raw)
		}
		if _, ok := result["meta"].(map[string]any)["warnings"].([]any); !ok {
			t.Fatal(raw)
		}
		body := result["data"].(map[string]any)["body"].(map[string]any)
		if body["IsSubmit"] != true || body["warnings"] != nil {
			t.Fatal(body)
		}
	}
	code, _, raw := invoke(t, []string{"api", "invoke", "--body", `{"ActionName":"UpdateBizObject"}`, "--dry-run"}, "", nil)
	if code != 0 || !strings.Contains(raw, "先触发作废规则") {
		t.Fatal(raw)
	}
	// 500 is guidance, not an invented server hard limit or automatic batch splitting.
	items := make([]any, 501)
	for i := range items {
		items[i] = map[string]any{"field": i}
	}
	code, result, raw := invoke(t, []string{"records", "create-many", "--schema", "test", "--data", encoded(items), "--dry-run"}, "", nil)
	if code != 0 {
		t.Fatal(raw)
	}
	if len(result["data"].(map[string]any)["body"].(map[string]any)["BizObjectArray"].([]any)) != 501 {
		t.Fatal("batch changed")
	}
}

func TestChildAttachmentContract(t *testing.T) {
	file := filepath.Join(t.TempDir(), "附件.txt")
	if err := os.WriteFile(file, []byte("test"), 0600); err != nil {
		t.Fatal(err)
	}
	code, result, raw := invoke(t, []string{"files", "upload", "--schema", "main", "--id", "child-row", "--child-schema", "child", "--field", "F0001", "--file", file, "--dry-run"}, "", nil)
	if code != 0 || !strings.Contains(raw, "子表行 ID") {
		t.Fatal(raw)
	}
	body := result["data"].(map[string]any)["body"].(map[string]any)
	query := body["query"].(map[string]any)
	for key, want := range map[string]string{"SchemaCode": "main", "ChildSchemaCode": "child", "BizObjectId": "child-row", "FilePropertyName": "F0001"} {
		if query[key].([]any)[0] != want {
			t.Fatal(query)
		}
	}
	if body["multipart_field"] != "media" {
		t.Fatal("official upload wire format changed")
	}
}
