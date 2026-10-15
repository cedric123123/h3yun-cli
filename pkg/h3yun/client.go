// Package h3yun implements the documented H3yun OpenAPI transport.
package h3yun

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const MaxJSON = 16 << 20
const MaxUpload = 10 << 20
const MaxTimeout = 60 * time.Second
const Version = "0.3.0"

type Error struct {
	Code           string   `json:"code"`
	Message        string   `json:"message"`
	HTTPStatus     int      `json:"http_status,omitempty"`
	OutcomeUnknown bool     `json:"outcome_unknown,omitempty"`
	Exit           int      `json:"-"`
	Retryable      bool     `json:"retryable"`
	Hint           string   `json:"hint,omitempty"`
	Actions        []string `json:"actions,omitempty"`
}

func (e *Error) Error() string { return e.Message }
func Fail(code, message string, exit int) *Error {
	return &Error{Code: code, Message: message, Exit: exit}
}

type Config struct {
	BaseURL      string
	EngineCode   string
	EngineSecret string
	AccessToken  string
	CACert       string
	AllowHTTP    bool
	Timeout      time.Duration
	InvokePath   string
	UploadPath   string
	DownloadPath string
}

type Client struct {
	cfg  Config
	http *http.Client
}

// Endpoint preserves a reverse-proxy base path and rejects URL credentials,
// query strings and fragments. Credentials are never forwarded on redirects.
func Endpoint(base, path string, allowHTTP bool) (string, error) {
	u, err := url.Parse(base)
	if err != nil || u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" || u.Opaque != "" {
		return "", Fail("CONFIG", "base URL must contain scheme, host and optional path only", 3)
	}
	if u.Scheme != "https" && !(allowHTTP && u.Scheme == "http") {
		return "", Fail("CONFIG", "HTTPS required; explicitly set allow_http for trusted HTTP deployments", 3)
	}
	if !strings.HasPrefix(path, "/") || strings.HasPrefix(path, "//") || strings.ContainsAny(path, "?#\\") || strings.Contains(path, "://") {
		return "", Fail("CONFIG", "API path must be a relative path starting with one slash", 3)
	}
	for _, segment := range strings.Split(path, "/") {
		decoded, err := url.PathUnescape(segment)
		if err != nil || decoded == ".." || decoded == "." || strings.ContainsAny(decoded, "/\\") {
			return "", Fail("CONFIG", "invalid API path segment", 3)
		}
	}
	u.Path = strings.TrimRight(u.Path, "/") + path
	u.RawPath = ""
	return u.String(), nil
}

func New(cfg Config) (*Client, error) {
	if cfg.AccessToken == "" && (cfg.EngineCode == "" || cfg.EngineSecret == "") {
		return nil, Fail("AUTH_CONFIG", "set H3YUN_ENGINE_CODE and H3YUN_ENGINE_SECRET in the invoking process environment", 3)
	}
	if strings.ContainsAny(cfg.EngineCode+cfg.EngineSecret+cfg.AccessToken, "\r\n") {
		return nil, Fail("AUTH_CONFIG", "invalid credential header characters", 3)
	}
	if cfg.Timeout <= 0 || cfg.Timeout > MaxTimeout {
		return nil, Fail("CONFIG", "timeout must be positive and at most 60s", 3)
	}
	for _, path := range []string{cfg.InvokePath, cfg.UploadPath, cfg.DownloadPath} {
		if _, err := Endpoint(cfg.BaseURL, path, cfg.AllowHTTP); err != nil {
			return nil, err
		}
	}
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.TLSClientConfig = &tls.Config{MinVersion: tls.VersionTLS12}
	if cfg.CACert != "" {
		pem, err := os.ReadFile(cfg.CACert)
		if err != nil {
			return nil, Fail("CONFIG", "cannot read CA certificate file", 3)
		}
		pool, err := x509.SystemCertPool()
		if err != nil {
			pool = x509.NewCertPool()
		}
		if !pool.AppendCertsFromPEM(pem) {
			return nil, Fail("CONFIG", "CA file contains no PEM certificates", 3)
		}
		transport.TLSClientConfig.RootCAs = pool
	}
	return &Client{cfg: cfg, http: &http.Client{Timeout: cfg.Timeout, Transport: transport,
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}}, nil
}

func (c *Client) Close() { c.http.CloseIdleConnections() }

func (c *Client) request(ctx context.Context, path, contentType string, body io.Reader, query url.Values, write bool) (*http.Response, error) {
	endpoint, err := Endpoint(c.cfg.BaseURL, path, c.cfg.AllowHTTP)
	if err != nil {
		return nil, err
	}
	if len(query) > 0 {
		endpoint += "?" + query.Encode()
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, body)
	if err != nil {
		return nil, Fail("REQUEST", "could not construct request", 2)
	}
	req.Header.Set("Content-Type", contentType)
	if c.cfg.AccessToken != "" {
		req.Header.Set("Authorization", "Bearer "+c.cfg.AccessToken)
	} else {
		req.Header.Set("EngineCode", c.cfg.EngineCode)
		req.Header.Set("EngineSecret", c.cfg.EngineSecret)
	}
	req.Header.Set("User-Agent", "h3yun-cli/"+Version)
	res, err := c.http.Do(req)
	if err != nil {
		return nil, &Error{Code: "TRANSPORT", Message: "request failed or timed out; check network, TLS and endpoint; writes are not retried", OutcomeUnknown: write, Exit: 4}
	}
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		res.Body.Close()
		return nil, &Error{Code: "HTTP", Message: fmt.Sprintf("HTTP status %d; response body omitted", res.StatusCode), HTTPStatus: res.StatusCode, OutcomeUnknown: write && res.StatusCode >= 500, Exit: 4}
	}
	return res, nil
}

// ParseResponse retains ReturnData even on failure (batch calls may partially succeed).
func ParseResponse(body io.Reader) (map[string]any, error) {
	b, err := io.ReadAll(io.LimitReader(body, MaxJSON+1))
	if err != nil || len(b) > MaxJSON {
		return nil, Fail("PROTOCOL", "unreadable or oversized JSON response", 4)
	}
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.UseNumber()
	var envelope map[string]any
	if dec.Decode(&envelope) != nil || envelope == nil {
		return nil, Fail("PROTOCOL", "expected a JSON response object", 4)
	}
	var extra any
	if dec.Decode(&extra) != io.EOF {
		return nil, Fail("PROTOCOL", "unexpected data after JSON response", 4)
	}
	success, exists := envelope["Successful"].(bool)
	if !exists {
		return envelope, Fail("PROTOCOL", "response is missing boolean Successful", 4)
	}
	if !success {
		msg, _ := envelope["ErrorMessage"].(string)
		if msg == "" {
			msg = "H3yun reported failure; inspect ReturnData for partial results"
		}
		return envelope, Fail("API", msg, 5)
	}
	return envelope, nil
}

func markUnknown(err error, write bool) error {
	if e, ok := err.(*Error); ok && write && e.Code == "PROTOCOL" {
		e.OutcomeUnknown = true
	}
	return err
}

func (c *Client) Invoke(ctx context.Context, body map[string]any, write bool) (map[string]any, error) {
	b, err := json.Marshal(body)
	if err != nil {
		return nil, Fail("INPUT", "request cannot be encoded as JSON", 2)
	}
	res, err := c.request(ctx, c.cfg.InvokePath, "application/json; charset=utf-8", bytes.NewReader(b), nil, write)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	data, err := ParseResponse(res.Body)
	return data, markUnknown(err, write)
}

func UploadInfo(path string) (os.FileInfo, error) {
	info, err := os.Stat(path)
	if err != nil || !info.Mode().IsRegular() {
		return nil, Fail("INPUT", "upload requires a readable regular file", 2)
	}
	if info.Size() > MaxUpload {
		return nil, Fail("INPUT", "upload exceeds documented 10 MiB maximum", 2)
	}
	return info, nil
}

func (c *Client) Upload(ctx context.Context, path string, fields url.Values) (map[string]any, error) {
	if _, err := UploadInfo(path); err != nil {
		return nil, err
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, Fail("INPUT", "cannot open upload file", 2)
	}
	defer f.Close()
	contents, err := io.ReadAll(io.LimitReader(f, MaxUpload+1))
	if err != nil || len(contents) > MaxUpload {
		return nil, Fail("INPUT", "cannot read upload file or file exceeds size limit", 2)
	}
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	header := make(textproto.MIMEHeader)
	// Match the documented UTF-8 filename parameter. Older .NET deployments
	// may not recognize the RFC 5987 filename* emitted by FormatMediaType.
	filename := filepath.Base(path)
	if strings.ContainsAny(filename, "\r\n") {
		return nil, Fail("INPUT", "upload filename cannot contain line breaks", 2)
	}
	escape := strings.NewReplacer("\\", "\\\\", "\"", "\\\"")
	header.Set("Content-Disposition", `form-data; name="media"; filename="`+escape.Replace(filename)+`"`)
	contentType := mime.TypeByExtension(filepath.Ext(path))
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	header.Set("Content-Type", contentType)
	part, err := mw.CreatePart(header)
	if err != nil {
		return nil, Fail("INPUT", "cannot create multipart upload", 2)
	}
	if _, err = part.Write(contents); err != nil {
		return nil, Fail("INPUT", "cannot encode upload", 2)
	}
	if err = mw.Close(); err != nil {
		return nil, Fail("INPUT", "cannot finish upload encoding", 2)
	}
	res, err := c.request(ctx, c.cfg.UploadPath, mw.FormDataContentType(), &buf, fields, true)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	data, err := ParseResponse(res.Body)
	return data, markUnknown(err, true)
}

// Download never uses server-provided filenames or overwrites an existing file.
func (c *Client) Download(ctx context.Context, id, output string) (map[string]any, error) {
	if _, err := os.Lstat(output); err == nil || !os.IsNotExist(err) {
		return nil, Fail("OUTPUT", "output path already exists or is inaccessible", 6)
	}
	form := url.Values{"attachmentId": {id}, "EngineCode": {c.cfg.EngineCode}}
	res, err := c.request(ctx, c.cfg.DownloadPath, "application/x-www-form-urlencoded", strings.NewReader(form.Encode()), nil, false)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	// The documented download endpoint returns Content-Disposition for files.
	// Without it, reject JSON/HTML error pages rather than saving them as attachments.
	if res.Header.Get("Content-Disposition") == "" {
		if strings.Contains(strings.ToLower(res.Header.Get("Content-Type")), "json") {
			data, err := ParseResponse(res.Body)
			if err != nil {
				return data, err
			}
		}
		return nil, Fail("PROTOCOL", "download response has no Content-Disposition; verify endpoint and response contract", 4)
	}
	// Stage first; a failed transfer must not leave a success-looking destination.
	temp, err := os.CreateTemp(filepath.Dir(output), ".h3yun-download-*")
	if err != nil {
		return nil, Fail("OUTPUT", "cannot create temporary download file", 6)
	}
	defer os.Remove(temp.Name())
	n, copyErr := io.Copy(temp, io.LimitReader(res.Body, (1<<30)+1))
	closeErr := temp.Close()
	if copyErr != nil || closeErr != nil || n > 1<<30 {
		return nil, Fail("OUTPUT", "download failed or exceeded 1 GiB", 6)
	}
	// Hard-link creation is atomic and fails if output already exists.
	if err := os.Link(temp.Name(), output); err != nil {
		return nil, Fail("OUTPUT", "cannot publish file: target exists or filesystem does not support hard links", 6)
	}
	abs, _ := filepath.Abs(output)
	return map[string]any{"path": abs, "bytes": n, "attachment_id": id}, nil
}
