package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func invoke(t *testing.T, args []string, input string, env map[string]string) (int, map[string]any, string) {
	t.Helper()
	var out bytes.Buffer
	code := Run(context.Background(), args, strings.NewReader(input), &out, func(k string) string { return env[k] })
	var data map[string]any
	d := json.NewDecoder(strings.NewReader(out.String()))
	d.UseNumber()
	if err := d.Decode(&data); err != nil {
		t.Fatalf("invalid output %q: %v", out.String(), err)
	}
	return code, data, out.String()
}

func TestLocalCommandsWithoutCredentials(t *testing.T) {
	for _, args := range [][]string{nil, {"help"}, {"schema", "--pretty"}, {"version"}, {"records", "create", "--help"}, {"ops", "list"}} {
		code, data, _ := invoke(t, args, "", nil)
		if code != 0 || data["ok"] != true {
			t.Fatalf("%v: %+v", args, data)
		}
	}
	_, result, _ := invoke(t, []string{"version"}, "", nil)
	version := result["data"].(map[string]any)
	if version["version"] != Version || version["commit"] == "" {
		t.Fatalf("version metadata missing: %+v", version)
	}
}

func TestDryRunWireShapes(t *testing.T) {
	for _, tc := range []struct {
		args                 []string
		input, action, field string
		array                bool
	}{
		{[]string{"records", "create", "--schema", "s", "--data", "-", "--dry-run"}, `{"金额":9007199254740993}`, "CreateBizObject", "BizObject", false},
		{[]string{"records", "update", "--schema", "s", "--id", "id", "--data", "-", "--dry-run"}, `{"name":"张三"}`, "UpdateBizObject", "BizObject", false},
		{[]string{"records", "create-many", "--schema", "s", "--data", "-", "--dry-run"}, `[{"name":"A"},{"name":"B"}]`, "CreateBizObjects", "BizObjectArray", true},
	} {
		code, data, raw := invoke(t, tc.args, tc.input, nil)
		if code != 0 {
			t.Fatal(raw)
		}
		body := data["data"].(map[string]any)["body"].(map[string]any)
		if body["ActionName"] != tc.action {
			t.Fatal(body)
		}
		if tc.array {
			for _, item := range body[tc.field].([]any) {
				if _, ok := item.(string); !ok {
					t.Fatal("not a JSON string array")
				}
			}
		} else if _, ok := body[tc.field].(string); !ok {
			t.Fatal("not a serialized object")
		}
		if strings.Contains(tc.input, "9007199254740993") && !strings.Contains(raw, "9007199254740993") {
			t.Fatal("lost numeric precision")
		}
		if tc.action != "UpdateBizObject" && body["IsSubmit"] != false {
			t.Fatal("must default to draft")
		}
	}
}

func TestInputFailures(t *testing.T) {
	for _, args := range [][]string{
		{"records", "get", "--schema", "s"},
		{"records", "get", "--schema", "s", "--id", "x", "--unknown"},
		{"records", "list", "--schema", "s", "--limit", "501", "--dry-run"},
		{"records", "list", "--schema", "s", "--limit", "0", "--dry-run"},
		{"records", "list", "--schema", "s", "--offset", "-1", "--dry-run"},
		{"records", "list", "--schema", "s", "--filter", `{}`, "--offset", "0", "--dry-run"},
		{"records", "create", "--schema", "s", "--data", `null`, "--dry-run"},
		{"records", "create", "--schema", "s", "--data", `{} {}`, "--dry-run"},
		{"records", "create-many", "--schema", "s", "--data", `[]`, "--dry-run"},
		{"records", "create-many", "--schema", "s", "--data", `[{} , "bad"]`, "--dry-run"},
		{"records", "get", "--schema", "s", "--id", "x", "extra"},
		{"api", "invoke", "--body", `{"ActionName":"LoadBizObject","EngineSecret":"bad"}`, "--dry-run"},
	} {
		code, _, raw := invoke(t, args, "", nil)
		if code != 2 {
			t.Fatalf("%v: %d %s", args, code, raw)
		}
	}
}

func TestListWirePaginationAndCredentialHeaders(t *testing.T) {
	hits := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		if r.Header.Get("EngineCode") != "engine-123" || r.Header.Get("EngineSecret") != "secret-456" {
			t.Error("missing headers")
		}
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		var filter map[string]any
		if err := json.Unmarshal([]byte(body["Filter"].(string)), &filter); err != nil {
			t.Error(err)
		}
		if filter["FromRowNum"] != float64(10) || filter["ToRowNum"] != float64(12) {
			t.Error(filter)
		}
		fmt.Fprint(w, `{"Successful":true,"ReturnData":{"BizObjectArray":[{"ObjectId":"1"},{"ObjectId":"2"}]}}`)
	}))
	defer srv.Close()
	env := map[string]string{"H3YUN_BASE_URL": srv.URL, "H3YUN_ALLOW_HTTP": "true", "H3YUN_ENGINE_CODE": "engine-123", "H3YUN_ENGINE_SECRET": "secret-456"}
	code, result, raw := invoke(t, []string{"records", "list", "--schema", "s", "--offset", "10", "--limit", "2"}, "", env)
	if code != 0 || hits != 1 {
		t.Fatal(raw, hits)
	}
	p := result["meta"].(map[string]any)["pagination"].(map[string]any)
	if p["next_offset"].(json.Number).String() != "12" || p["has_more_possible"] != true {
		t.Fatal(p)
	}
}

func TestReadOnlyAndDeleteConfirmation(t *testing.T) {
	for _, tc := range []struct {
		args []string
		env  map[string]string
		code string
	}{
		{[]string{"records", "delete", "--schema", "s", "--id", "id"}, nil, "CONFIRMATION_REQUIRED"},
		{[]string{"records", "create", "--schema", "s", "--data", `{}`}, map[string]string{"H3YUN_READ_ONLY": "true"}, "READ_ONLY"},
		{[]string{"api", "invoke", "--body", `{"ActionName":"RemoveBizObject"}`}, nil, "CONFIRMATION_REQUIRED"},
		{[]string{"api", "invoke", "--body", `{"ActionName":"LoadBizObject"}`}, map[string]string{"H3YUN_READ_ONLY": "true"}, "READ_ONLY"},
		{[]string{"records", "get", "--schema", "s", "--id", "id"}, nil, "AUTH_CONFIG"},
		{[]string{"ops", "run", "--name", "workflow.approve"}, nil, "CAPABILITY_UNCONFIGURED"},
	} {
		code, result, raw := invoke(t, tc.args, "", tc.env)
		if code != 3 || result["error"].(map[string]any)["code"] != tc.code {
			t.Fatal(raw)
		}
	}
}

func TestCustomAndOperationsMapping(t *testing.T) {
	path := filepath.Join(t.TempDir(), "ops.json")
	content := `{"workflow.get":{"action":"GetWorkflow","controller":"ExistingController","app_code":"app","description":"test mapping only","read_only":true,"required":["workflowId"]}}`
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
	env := map[string]string{"H3YUN_OPERATIONS": path}
	code, result, raw := invoke(t, []string{"ops", "run", "--name", "workflow.get", "--data", `{"workflowId":"wf"}`, "--dry-run"}, "", env)
	if code != 0 {
		t.Fatal(raw)
	}
	body := result["data"].(map[string]any)["body"].(map[string]any)
	if body["Controller"] != "ExistingController" || body["workflowId"] != "wf" {
		t.Fatal(body)
	}
	code, _, _ = invoke(t, []string{"ops", "run", "--name", "workflow.get", "--dry-run"}, "", env)
	if code != 2 {
		t.Fatal("accepted missing required argument")
	}
	code, _, _ = invoke(t, []string{"custom", "invoke", "--action", "Example", "--controller", "C", "--app", "A", "--data", `{"Controller":"override"}`, "--dry-run"}, "", nil)
	if code != 2 {
		t.Fatal("routing override accepted")
	}
}

func TestBOMInputAndConfigRejectsSecrets(t *testing.T) {
	file := filepath.Join(t.TempDir(), "record.json")
	if err := os.WriteFile(file, []byte("\xef\xbb\xbf"+`{"中文":"值"}`), 0600); err != nil {
		t.Fatal(err)
	}
	code, _, raw := invoke(t, []string{"records", "create", "--schema", "s", "--data", "@" + file, "--dry-run"}, "", nil)
	if code != 0 {
		t.Fatal(raw)
	}
	config := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(config, []byte(`{"engine_secret":"forbidden"}`), 0600); err != nil {
		t.Fatal(err)
	}
	code, _, _ = invoke(t, []string{"doctor", "--config", config}, "", nil)
	if code != 2 {
		t.Fatal("accepted secret in config file")
	}
}

func TestSecretsRedactedFromServerErrors(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{"Successful": false, "ErrorMessage": "echo " + r.Header.Get("EngineSecret"), "ReturnData": map[string]any{"secret": r.Header.Get("EngineSecret")}})
	}))
	defer srv.Close()
	env := map[string]string{"H3YUN_BASE_URL": srv.URL, "H3YUN_ALLOW_HTTP": "true", "H3YUN_ENGINE_CODE": "test-code", "H3YUN_ENGINE_SECRET": `special-"\secret`}
	code, _, raw := invoke(t, []string{"records", "get", "--schema", "s", "--id", "i"}, "", env)
	if code != 5 || strings.Contains(raw, "special-") || !strings.Contains(raw, "[REDACTED]") {
		t.Fatal(raw)
	}
	// A short credential must not corrupt JSON syntax or property names.
	code, result, _ := invoke(t, []string{"version"}, "", map[string]string{"H3YUN_ENGINE_SECRET": "e"})
	if code != 0 || result["ok"] != true {
		t.Fatal("redaction corrupted envelope")
	}
}

func TestCustomBusinessFailure(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"Successful":true,"ReturnData":{"result":"error","message":"permission denied"}}`)
	}))
	defer srv.Close()
	path := filepath.Join(t.TempDir(), "ops.json")
	if err := os.WriteFile(path, []byte(`{"workflow.approve":{"action":"Approve","controller":"Verified","app_code":"app","read_only":false,"required":["id"],"success_field":"result","success_value":"success"}}`), 0600); err != nil {
		t.Fatal(err)
	}
	env := map[string]string{"H3YUN_OPERATIONS": path, "H3YUN_BASE_URL": srv.URL, "H3YUN_ALLOW_HTTP": "true", "H3YUN_ENGINE_CODE": "test-code", "H3YUN_ENGINE_SECRET": "test-secret"}
	code, response, raw := invoke(t, []string{"ops", "run", "--name", "workflow.approve", "--data", `{"id":"work-item"}`}, "", env)
	if code != 5 || response["error"].(map[string]any)["code"] != "CUSTOM_BUSINESS" || !strings.Contains(raw, "permission denied") {
		t.Fatal(raw)
	}
}

func TestExampleOperationsCannotBeExecuted(t *testing.T) {
	code, response, raw := invoke(t, []string{"ops", "list"}, "", map[string]string{"H3YUN_OPERATIONS": "../../operations.example.json"})
	if code != 3 || response["error"].(map[string]any)["code"] != "CAPABILITY_UNCONFIGURED" {
		t.Fatal(raw)
	}
}

func TestDryRunNeverContactsServer(t *testing.T) {
	hits := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { hits++ }))
	defer srv.Close()
	code, _, raw := invoke(t, []string{"records", "delete", "--schema", "s", "--id", "id", "--dry-run"}, "", map[string]string{"H3YUN_BASE_URL": srv.URL, "H3YUN_ALLOW_HTTP": "true"})
	if code != 0 || hits != 0 {
		t.Fatal(raw, hits)
	}
	code, _, raw = invoke(t, []string{"doctor", "--dry-run"}, "", nil)
	if code != 0 {
		t.Fatal(raw)
	}
}
