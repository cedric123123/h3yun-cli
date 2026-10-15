package h3yun

import (
	"context"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func clientFor(t *testing.T, server *httptest.Server) *Client {
	t.Helper()
	c, err := New(Config{BaseURL: server.URL + "/tenant", EngineCode: "test-engine", EngineSecret: "test-secret", AllowHTTP: true, Timeout: time.Second, InvokePath: "/OpenApi/Invoke", UploadPath: "/OpenApi/UploadAttachment", DownloadPath: "/Api/DownloadBizObjectFile"})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(c.Close)
	return c
}

func TestClientTimeoutContract(t *testing.T) {
	for _, duration := range []time.Duration{0, -time.Second, 60*time.Second + time.Nanosecond, 10 * time.Minute, time.Minute} {
		cfg := Config{BaseURL: "https://example.test", EngineCode: "test", EngineSecret: "test", Timeout: duration, InvokePath: "/invoke", UploadPath: "/upload", DownloadPath: "/download"}
		client, err := New(cfg)
		if duration == time.Minute {
			if err != nil {
				t.Fatal(err)
			}
			client.Close()
		} else if err == nil {
			client.Close()
			t.Fatalf("accepted timeout %v", duration)
		}
	}
}

func TestEndpoint(t *testing.T) {
	for _, tc := range []struct {
		base, path string
		http, ok   bool
	}{
		{"https://example.test/reverse/proxy/", "/OpenApi/Invoke", false, true},
		{"http://example.test", "/OpenApi/Invoke", false, false},
		{"http://example.test", "/OpenApi/Invoke", true, true},
		{"https://user:pass@example.test", "/OpenApi/Invoke", false, false},
		{"https://example.test?token=secret", "/OpenApi/Invoke", false, false},
		{"https://example.test", "//evil.test/x", false, false},
		{"https://example.test", "/../x", false, false},
		{"https://example.test", "/%2e%2e/x", false, false},
	} {
		_, err := Endpoint(tc.base, tc.path, tc.http)
		if (err == nil) != tc.ok {
			t.Errorf("%+v: %v", tc, err)
		}
	}
	got, _ := Endpoint("https://example.test/reverse/proxy/", "/OpenApi/Invoke", false)
	if got != "https://example.test/reverse/proxy/OpenApi/Invoke" {
		t.Fatal(got)
	}
}

func TestInvokeProtocolAndPartialResults(t *testing.T) {
	count := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count++
		if r.URL.Path != "/tenant/OpenApi/Invoke" || r.Method != "POST" {
			t.Error(r.URL.Path, r.Method)
		}
		if r.Header.Get("EngineCode") != "test-engine" || r.Header.Get("EngineSecret") != "test-secret" {
			t.Error("missing authentication")
		}
		fmt.Fprint(w, `{"Successful":false,"ErrorMessage":"partial failure","ReturnData":{"ObjectIds":["created-id"],"number":9007199254740993}}`)
	}))
	defer srv.Close()
	result, err := clientFor(t, srv).Invoke(context.Background(), map[string]any{"ActionName": "CreateBizObjects"}, true)
	if err == nil || err.(*Error).Exit != 5 || count != 1 {
		t.Fatalf("%v count=%d", err, count)
	}
	data := result["ReturnData"].(map[string]any)
	if data["number"].(json.Number).String() != "9007199254740993" {
		t.Fatal(data)
	}
	if len(data["ObjectIds"].([]any)) != 1 {
		t.Fatal("lost partial result")
	}
}

func TestRejectRedirectWithoutLeakingCredentials(t *testing.T) {
	hits := 0
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { hits++ }))
	defer target.Close()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { http.Redirect(w, r, target.URL, 307) }))
	defer srv.Close()
	_, err := clientFor(t, srv).Invoke(context.Background(), map[string]any{}, false)
	if err == nil || hits != 0 || err.(*Error).HTTPStatus != 307 {
		t.Fatalf("%v hits=%d", err, hits)
	}
}

func TestProtocolErrors(t *testing.T) {
	for _, body := range []string{"<html>login</html>", "null", `{}`, `{"Successful":"true"}`, `{"Successful":true} {}`, strings.Repeat("x", MaxJSON+1)} {
		if _, err := ParseResponse(strings.NewReader(body)); err == nil {
			t.Fatal("accepted invalid response")
		}
	}
}

func TestHTTPFailureAndUnknownWriteOutcome(t *testing.T) {
	hits := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		w.WriteHeader(503)
		fmt.Fprint(w, "sensitive body")
	}))
	defer srv.Close()
	_, err := clientFor(t, srv).Invoke(context.Background(), map[string]any{}, true)
	e := err.(*Error)
	if hits != 1 || !e.OutcomeUnknown || strings.Contains(e.Message, "sensitive") {
		t.Fatalf("%+v hits=%d", e, hits)
	}
}

func TestUploadMultipart(t *testing.T) {
	file := filepath.Join(t.TempDir(), "附件.txt")
	if err := os.WriteFile(file, []byte("附件内容"), 0600); err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/tenant/OpenApi/UploadAttachment" || r.URL.Query().Get("ChildSchemaCode") != "child" {
			t.Error(r.URL)
		}
		if err := r.ParseMultipartForm(MaxUpload + 1024); err != nil {
			t.Fatal(err)
		}
		f, header, err := r.FormFile("media")
		if err != nil {
			t.Fatal(err)
		}
		defer f.Close()
		b, _ := io.ReadAll(f)
		if string(b) != "附件内容" || header.Filename != "附件.txt" {
			t.Error(string(b), header.Filename)
		}
		if !strings.Contains(header.Header.Get("Content-Disposition"), `filename="附件.txt"`) {
			t.Error("must use filename= for compatibility with documented .NET upload parser")
		}
		fmt.Fprint(w, `{"Successful":true,"ReturnData":{"AttachmentId":"a"}}`)
	}))
	defer srv.Close()
	_, err := clientFor(t, srv).Upload(context.Background(), file, url.Values{"ChildSchemaCode": {"child"}})
	if err != nil {
		t.Fatal(err)
	}
}

func TestDownloadAndNoOverwrite(t *testing.T) {
	hits := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		r.ParseForm()
		if r.Form.Get("attachmentId") != "attachment" || r.Form.Get("EngineCode") != "test-engine" {
			t.Error(r.Form)
		}
		w.Header().Set("Content-Disposition", `attachment; filename="../../unsafe.txt"`)
		fmt.Fprint(w, "content")
	}))
	defer srv.Close()
	c := clientFor(t, srv)
	target := filepath.Join(t.TempDir(), "chosen.txt")
	if _, err := c.Download(context.Background(), "attachment", target); err != nil {
		t.Fatal(err)
	}
	if _, err := c.Download(context.Background(), "attachment", target); err == nil {
		t.Fatal("overwrote target")
	}
	b, _ := os.ReadFile(target)
	if string(b) != "content" || hits != 1 {
		t.Fatal(string(b), hits)
	}
}

func TestFailedDownloadDoesNotCreateFile(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"Successful":false,"ErrorMessage":"not found"}`)
	}))
	defer srv.Close()
	target := filepath.Join(t.TempDir(), "never.txt")
	if _, err := clientFor(t, srv).Download(context.Background(), "x", target); err == nil {
		t.Fatal("accepted API failure")
	}
	if _, err := os.Stat(target); !os.IsNotExist(err) {
		t.Fatal("created failed output")
	}
}

func TestTimeoutNotRetried(t *testing.T) {
	hits := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		time.Sleep(100 * time.Millisecond)
		fmt.Fprint(w, `{"Successful":true}`)
	}))
	c := clientFor(t, srv)
	c.http.Timeout = 10 * time.Millisecond
	_, err := c.Invoke(context.Background(), map[string]any{}, true)
	srv.Close()
	if err == nil || !err.(*Error).OutcomeUnknown || hits != 1 {
		t.Fatalf("%v %d", err, hits)
	}
}

func TestEnterpriseCA(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { fmt.Fprint(w, `{"Successful":true}`) }))
	defer srv.Close()
	path := filepath.Join(t.TempDir(), "ca.pem")
	if err := os.WriteFile(path, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: srv.Certificate().Raw}), 0600); err != nil {
		t.Fatal(err)
	}
	cfg := Config{BaseURL: srv.URL, EngineCode: "test-code", EngineSecret: "test-secret", CACert: path, Timeout: time.Second, InvokePath: "/OpenApi/Invoke", UploadPath: "/OpenApi/UploadAttachment", DownloadPath: "/Api/DownloadBizObjectFile"}
	c, err := New(cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	if _, err := c.Invoke(context.Background(), map[string]any{}, false); err != nil {
		t.Fatal(err)
	}
	cfg.CACert = filepath.Join(t.TempDir(), "missing.pem")
	if _, err := New(cfg); err == nil {
		t.Fatal("accepted missing CA certificate")
	}
}

func TestTruncatedDownloadCleansUp(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Disposition", "attachment; filename=test.txt")
		w.Header().Set("Content-Length", "100")
		fmt.Fprint(w, "short")
	}))
	defer srv.Close()
	dir := t.TempDir()
	if _, err := clientFor(t, srv).Download(context.Background(), "id", filepath.Join(dir, "result.txt")); err == nil {
		t.Fatal("accepted truncated download")
	}
	entries, err := os.ReadDir(dir)
	if err != nil || len(entries) != 0 {
		t.Fatal("left partial files", entries, err)
	}
}
