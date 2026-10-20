package cli

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/cedric123123/h3yun-cli/pkg/h3yun"
)

func metadataEnv(t *testing.T, serverURL string, operation string) map[string]string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "operations.json")
	if err := os.WriteFile(path, []byte(operation), 0600); err != nil {
		t.Fatal(err)
	}
	return map[string]string{"H3YUN_OPERATIONS": path, "H3YUN_BASE_URL": serverURL, "H3YUN_ALLOW_HTTP": "true", "H3YUN_ENGINE_CODE": "engine", "H3YUN_ENGINE_SECRET": "secret"}
}

func TestSchemaFieldsRequiresVerifiedReadOnlyMapping(t *testing.T) {
	code, discovered, raw := invoke(t, []string{"schema", "schema fields", "--compact"}, "", nil)
	if code != 0 {
		t.Fatal(raw)
	}
	params := discovered["data"].(map[string]any)["parameters"].(map[string]any)
	if params["operation"].(map[string]any)["default"] != "schema.fields" || params["schema-param"].(map[string]any)["default"] != "SchemaCode" {
		t.Fatal(params)
	}
	for _, tc := range []struct {
		name, operation, parameter string
		code                       int
	}{
		{"missing", `{}`, "", 3},
		{"write mapping", `{"schema.fields":{"action":"Fields","controller":"Metadata","app_code":"app","read_only":false}}`, "", 3},
		{"reserved parameter", `{"schema.fields":{"action":"Fields","controller":"Metadata","app_code":"app","read_only":true}}`, "EngineSecret", 2},
		{"extra required", `{"schema.fields":{"action":"Fields","controller":"Metadata","app_code":"app","read_only":true,"required":["TenantId"]}}`, "", 2},
	} {
		t.Run(tc.name, func(t *testing.T) {
			env := metadataEnv(t, "http://unused.test", tc.operation)
			args := []string{"schema", "fields", "--schema", "D0001", "--dry-run"}
			if tc.parameter != "" {
				args = append(args, "--schema-param", tc.parameter)
			}
			code, _, raw := invoke(t, args, "", env)
			if code != tc.code {
				t.Fatalf("code=%d: %s", code, raw)
			}
		})
	}
}

func TestSchemaFieldsWireAndSuccessCondition(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		if body["ActionName"] != "DescribeFields" || body["Controller"] != "MetadataController" || body["AppCode"] != "app" || body["FormCode"] != "D0001" {
			t.Error(body)
		}
		fmt.Fprint(w, `{"Successful":true,"ReturnData":{"status":"ok","fields":[{"code":"F1","name":"金额","type":"Number","required":true,"options":["A"],"future_property":{"kept":true}}]}}`)
	}))
	defer srv.Close()
	op := `{"schema.fields":{"action":"DescribeFields","controller":"MetadataController","app_code":"app","read_only":true,"required":["FormCode"],"success_field":"status","success_value":"ok"}}`
	env := metadataEnv(t, srv.URL, op)
	code, response, raw := invoke(t, []string{"schema", "fields", "--schema", "D0001", "--schema-param", "FormCode"}, "", env)
	if code != 0 || calls.Load() != 1 || !strings.Contains(raw, `"fields"`) || !strings.Contains(raw, `"future_property"`) {
		t.Fatal(code, calls.Load(), raw)
	}
	meta := response["meta"].(map[string]any)
	if meta["metadata_source"] != "configured_custom_controller" {
		t.Fatal(meta)
	}

	// The configured success condition is authoritative even when transport and
	// the platform envelope succeed.
	op = strings.Replace(op, `"success_value":"ok"`, `"success_value":"different"`, 1)
	env = metadataEnv(t, srv.URL, op)
	code, response, _ = invoke(t, []string{"schema", "fields", "--schema", "D0001", "--schema-param", "FormCode"}, "", env)
	if code != 5 || response["error"].(map[string]any)["code"] != "CUSTOM_BUSINESS" {
		t.Fatal(response)
	}
}

func TestDoctorRemoteScopeAndRedaction(t *testing.T) {
	for _, tc := range []struct {
		name, response, status string
		code, httpStatus       int
	}{
		{"success", `{"Successful":true,"ReturnData":{"BizObjectArray":[{"ObjectId":"private-record"}]}}`, "read_succeeded", 0, 200},
		{"api", `{"Successful":false,"ErrorMessage":"denied"}`, "platform_rejected_check_credentials_schema_and_permissions", 5, 200},
		{"protocol", `{"Successful":true,"ReturnData":{}}`, "unexpected_response", 4, 200},
		{"http", `{"error":"upstream"}`, "http_error", 4, 500},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var calls atomic.Int32
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				var body map[string]any
				json.NewDecoder(r.Body).Decode(&body)
				if body["ActionName"] != "LoadBizObjects" || body["SchemaCode"] != "D0001" {
					t.Error(body)
				}
				w.WriteHeader(tc.httpStatus)
				fmt.Fprint(w, tc.response)
			}))
			defer srv.Close()
			env := map[string]string{"H3YUN_BASE_URL": srv.URL, "H3YUN_ALLOW_HTTP": "true", "H3YUN_ENGINE_CODE": "engine", "H3YUN_ENGINE_SECRET": "secret"}
			code, response, raw := invoke(t, []string{"doctor", "--remote", "--schema", "D0001"}, "", env)
			if code != tc.code || calls.Load() != 1 || strings.Contains(raw, "private-record") {
				t.Fatal(code, calls.Load(), raw)
			}
			data := response["data"].(map[string]any)
			if data["status"] != tc.status || data["scope"] != "read_access_to_requested_schema" {
				t.Fatal(data)
			}
		})
	}
	code, _, raw := invoke(t, []string{"doctor", "--remote"}, "", nil)
	if code != 2 || !strings.Contains(raw, "requires --schema") {
		t.Fatal(raw)
	}
	code, _, raw = invoke(t, []string{"doctor", "--schema", "D0001"}, "", nil)
	if code != 2 || !strings.Contains(raw, "requires --remote") {
		t.Fatal(raw)
	}
	data, err := remoteDiagnosis(nil, &h3yun.Error{Code: "TRANSPORT", Message: "offline", Exit: 4}, "D0001")
	if err == nil || data.(map[string]any)["status"] != "network_or_tls_error" {
		t.Fatal(data, err)
	}
}

func TestMinIntervalConfigAndOverride(t *testing.T) {
	config := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(config, []byte(`{"min_interval":"150ms"}`), 0600); err != nil {
		t.Fatal(err)
	}
	_, cfg, err := settings(config, func(string) string { return "" })
	if err != nil || cfg.MinInterval.String() != "150ms" {
		t.Fatal(cfg.MinInterval, err)
	}
	_, cfg, err = settings(config, func(k string) string {
		if k == "H3YUN_MIN_INTERVAL" {
			return "2s"
		}
		return ""
	})
	if err != nil || cfg.MinInterval.String() != "2s" {
		t.Fatal(cfg.MinInterval, err)
	}
	env := map[string]string{"H3YUN_MIN_INTERVAL": "2s"}
	code, result, raw := invoke(t, []string{"--config", config, "--min-interval", "3s", "records", "get", "--schema", "s", "--id", "i", "--dry-run"}, "", env)
	if code != 0 || result["meta"].(map[string]any)["request_policy"].(map[string]any)["min_interval"] != "3s" {
		t.Fatal(raw)
	}
	if code, _, raw := invoke(t, []string{"records", "get", "--schema", "s", "--id", "i", "--dry-run", "--min-interval", ""}, "", nil); code != 2 {
		t.Fatal(raw)
	}
	for _, value := range []string{"0s", "60s"} {
		_, cfg, err := settings("", func(k string) string {
			if k == "H3YUN_MIN_INTERVAL" {
				return value
			}
			return ""
		})
		if err != nil || cfg.MinInterval != mustDuration(t, value) {
			t.Fatal(value, cfg.MinInterval, err)
		}
	}
	for _, value := range []string{"bad", "-1s", "61s"} {
		if _, _, err := settings("", func(k string) string {
			if k == "H3YUN_MIN_INTERVAL" {
				return value
			}
			return ""
		}); err == nil {
			t.Fatal("accepted", value)
		}
	}
}

func mustDuration(t *testing.T, value string) time.Duration {
	t.Helper()
	d, err := time.ParseDuration(value)
	if err != nil {
		t.Fatal(err)
	}
	return d
}
