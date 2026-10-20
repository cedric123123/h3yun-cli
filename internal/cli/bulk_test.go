package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/cedric123123/h3yun-cli/pkg/h3yun"
)

func bulkClient(t *testing.T, handler http.HandlerFunc) *h3yun.Client {
	t.Helper()
	s := httptest.NewServer(handler)
	t.Cleanup(s.Close)
	c, err := h3yun.New(h3yun.Config{BaseURL: s.URL, EngineCode: "e", EngineSecret: "s", AllowHTTP: true, Timeout: time.Second, InvokePath: "/invoke", UploadPath: "/upload", DownloadPath: "/download"})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(c.Close)
	return c
}

func TestListAllBoundedAndPartial(t *testing.T) {
	for _, mode := range []string{"short", "max", "api", "http429", "http500", "protocol", "oversized"} {
		t.Run(mode, func(t *testing.T) {
			var calls atomic.Int32
			c := bulkClient(t, func(w http.ResponseWriter, r *http.Request) {
				n := calls.Add(1)
				var body map[string]any
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
					t.Error(err)
				}
				var filter map[string]any
				json.Unmarshal([]byte(body["Filter"].(string)), &filter)
				expectedFrom := float64(7 + (int(n)-1)*2)
				if filter["FromRowNum"] != expectedFrom || filter["Marker"] != "preserved" {
					t.Errorf("filter: %v", filter)
				}
				if n == 1 {
					fmt.Fprint(w, `{"Successful":true,"ReturnData":{"BizObjectArray":[{"ObjectId":"a"},{"ObjectId":"b"}],"TotalCount":99}}`)
					return
				}
				if filter["ToRowNum"] != float64(10) {
					t.Error("last request must respect remaining max", filter)
				}
				switch mode {
				case "short":
					fmt.Fprint(w, `{"Successful":true,"ReturnData":{"BizObjectArray":[]}}`)
				case "max":
					fmt.Fprint(w, `{"Successful":true,"ReturnData":{"BizObjectArray":[{"ObjectId":"c"}]}}`)
				case "api":
					fmt.Fprint(w, `{"Successful":false,"ErrorMessage":"denied"}`)
				case "http429":
					w.WriteHeader(429)
					fmt.Fprint(w, `{"error":"rate limited"}`)
				case "http500":
					w.WriteHeader(500)
					fmt.Fprint(w, `{"error":"upstream failed"}`)
				case "protocol":
					fmt.Fprint(w, `{"Successful":true,"ReturnData":{}}`)
				case "oversized":
					fmt.Fprint(w, `{"Successful":true,"ReturnData":{"BizObjectArray":[{},{}]}}`)
				}
			})
			meta := map[string]any{}
			result, err := listAll(context.Background(), c, "schema", map[string]any{"FromRowNum": 7, "ToRowNum": 9, "Marker": "preserved"}, 3, meta)
			wantError := mode != "short" && mode != "max"
			if (err != nil) != wantError || calls.Load() != 2 {
				t.Fatalf("calls=%d err=%v", calls.Load(), err)
			}
			rows := result.(map[string]any)["ReturnData"].(map[string]any)["BizObjectArray"].([]any)
			if result.(map[string]any)["ReturnData"].(map[string]any)["TotalCount"] != json.Number("99") {
				t.Fatal("first-page ReturnData metadata was lost")
			}
			wantRows := 2
			if mode == "max" {
				wantRows = 3
			}
			p := meta["pagination"].(map[string]any)
			if len(rows) != wantRows || p["next_offset"] != 7+wantRows || p["partial"] != wantError || p["complete"] != (mode == "short") {
				t.Fatalf("%v %v", rows, p)
			}
		})
	}
}

func TestListAllAggregateLimit(t *testing.T) {
	var calls atomic.Int32
	row := strings.Repeat("x", 9<<20)
	c := bulkClient(t, func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		json.NewEncoder(w).Encode(map[string]any{"Successful": true, "ReturnData": map[string]any{"BizObjectArray": []any{map[string]any{"payload": row}}}})
	})
	meta := map[string]any{}
	_, err := listAll(context.Background(), c, "s", map[string]any{"FromRowNum": 0, "ToRowNum": 1}, 10, meta)
	if e, ok := err.(*h3yun.Error); !ok || e.Code != "OUTPUT_LIMIT" {
		t.Fatalf("%v", err)
	}
	p := meta["pagination"].(map[string]any)
	if calls.Load() != 8 || p["returned"] != 7 || p["next_offset"] != 7 {
		t.Fatal(p, calls.Load())
	}
}

func TestUpdateManyProgressAndUnknown(t *testing.T) {
	for _, mode := range []string{"success", "api", "http", "protocol", "cancel"} {
		t.Run(mode, func(t *testing.T) {
			var calls atomic.Int32
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			c := bulkClient(t, func(w http.ResponseWriter, r *http.Request) {
				n := calls.Add(1)
				var body map[string]any
				json.NewDecoder(r.Body).Decode(&body)
				if body["ActionName"] != "UpdateBizObject" || body["BizObjectId"] != []string{"a", "b", "c"}[n-1] || body["BizObject"] != `{"amount":9007199254740993}` {
					t.Error(body)
				}
				if n == 1 && mode == "cancel" {
					cancel()
				}
				if n == 2 {
					switch mode {
					case "api":
						fmt.Fprint(w, `{"Successful":false,"ErrorMessage":"denied"}`)
						return
					case "http":
						w.WriteHeader(500)
						return
					case "protocol":
						fmt.Fprint(w, `{"unexpected":true}`)
						return
					}
				}
				fmt.Fprint(w, `{"Successful":true}`)
			})
			items, err := parseUpdates(`[{"id":"a","data":{"amount":9007199254740993}},{"id":"b","data":{"amount":9007199254740993}},{"id":"c","data":{"amount":9007199254740993}}]`, strings.NewReader(""))
			if err != nil {
				t.Fatal(err)
			}
			meta := map[string]any{}
			result, err := updateMany(ctx, c, "s", items, meta)
			data := result.(map[string]any)
			if mode == "success" {
				if err != nil || calls.Load() != 3 || !reflect.DeepEqual(data["succeeded_ids"], []string{"a", "b", "c"}) {
					t.Fatal(data, err)
				}
				return
			}
			if err == nil {
				t.Fatal("expected failure")
			}
			if mode == "cancel" {
				if calls.Load() != 1 || len(data["unprocessed_ids"].([]string)) < 2 {
					t.Fatal(data)
				}
				return
			}
			key := "unknown_ids"
			if mode == "api" {
				key = "failed_ids"
			}
			if calls.Load() != 2 || !reflect.DeepEqual(data[key], []string{"b"}) || !reflect.DeepEqual(data["succeeded_ids"], []string{"a"}) || !reflect.DeepEqual(data["unprocessed_ids"], []string{"c"}) {
				t.Fatal(data)
			}
		})
	}
}

func TestBulkValidationDryRunAndReadOnly(t *testing.T) {
	for _, input := range []string{`[]`, `[{}]`, `[{"id":"a","data":{}}]`, `[{"id":"a","data":{"x":1},"extra":1}]`, `[{"id":"a","data":{"x":1}},{"id":"a","data":{"x":2}}]`, `[null]`, `[{"id":" ","data":{"x":1}}]`, `[{"id":" a","data":{"x":1}}]`} {
		if code, _, raw := invoke(t, []string{"records", "update-many", "--schema", "s", "--data", input, "--dry-run"}, "", nil); code != 2 {
			t.Fatal(code, raw)
		}
	}
	items := make([]map[string]any, 501)
	for i := range items {
		items[i] = map[string]any{"id": fmt.Sprintf("id-%03d", i), "data": map[string]any{"x": i}}
	}
	encodedItems, _ := json.Marshal(items)
	if code, _, raw := invoke(t, []string{"records", "update-many", "--schema", "s", "--data", string(encodedItems), "--dry-run"}, "", nil); code != 2 {
		t.Fatal(code, raw)
	}
	encodedItems, _ = json.Marshal(items[:500])
	if code, _, raw := invoke(t, []string{"records", "update-many", "--schema", "s", "--data", string(encodedItems), "--dry-run"}, "", nil); code != 0 {
		t.Fatal(code, raw)
	}
	for _, flags := range [][]string{{"--all"}, {"--max", "1"}, {"--all", "--max", "0"}, {"--all", "--max", "100001"}} {
		args := append([]string{"records", "list", "--schema", "s", "--dry-run"}, flags...)
		if code, _, raw := invoke(t, args, "", nil); code != 2 {
			t.Fatal(code, raw)
		}
	}
	code, data, raw := invoke(t, []string{"records", "list", "--schema", "s", "--all", "--max", "1", "--dry-run"}, "", nil)
	if code != 0 || data["meta"].(map[string]any)["request_policy"].(map[string]any)["min_interval"] != "700ms" {
		t.Fatal(raw)
	}
	body := data["data"].(map[string]any)["body"].(map[string]any)
	if !strings.Contains(body["Filter"].(string), `"ToRowNum":1`) {
		t.Fatal(body)
	}
	args := []string{"records", "update-many", "--schema", "s", "--data", `[{"id":"a","data":{"x":1}}]`}
	if code, _, raw = invoke(t, args, "", map[string]string{"H3YUN_READ_ONLY": "true"}); code != 3 || !strings.Contains(raw, "READ_ONLY") {
		t.Fatal(code, raw)
	}
	code, data, raw = invoke(t, append(args, "--dry-run", "--min-interval", "2s"), "", nil)
	if code != 0 || data["meta"].(map[string]any)["request_policy"].(map[string]any)["min_interval"] != "2s" {
		t.Fatal(raw)
	}
}
