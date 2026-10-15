// Package auth implements public-client OAuth authorization-code + S256 PKCE.
// An actual authorization server and Bearer-capable API/gateway are required.
package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

type Config struct {
	AuthorizeURL string   `json:"authorize_url"`
	TokenURL     string   `json:"token_url"`
	ClientID     string   `json:"client_id"`
	Scopes       []string `json:"scopes,omitempty"`
	Audience     string   `json:"audience,omitempty"`
	CallbackPort int      `json:"callback_port,omitempty"`
	CACert       string   `json:"-"`
	AllowHTTP    bool     `json:"-"`
}

type Session struct {
	AccessToken string    `json:"access_token"`
	ExpiresAt   time.Time `json:"expires_at"`
	Binding     string    `json:"binding"`
}

func Binding(profile, base string, c Config) string {
	data, _ := json.Marshal([]any{profile, base, c.AuthorizeURL, c.TokenURL, c.ClientID, c.Scopes, c.Audience})
	hash := sha256.Sum256(data)
	return fmt.Sprintf("%x", hash)
}

func Validate(c Config) error {
	if c.ClientID == "" || c.AuthorizeURL == "" || c.TokenURL == "" {
		return fmt.Errorf("OAuth requires client_id, authorize_url and token_url")
	}
	if c.CallbackPort < 0 || c.CallbackPort > 65535 {
		return fmt.Errorf("invalid callback port")
	}
	for _, raw := range []string{c.AuthorizeURL, c.TokenURL} {
		u, err := url.Parse(raw)
		if err != nil || u.Host == "" || u.User != nil || u.Fragment != "" || u.RawQuery != "" || (u.Scheme != "https" && !(c.AllowHTTP && u.Scheme == "http" && net.ParseIP(u.Hostname()).IsLoopback())) {
			return fmt.Errorf("OAuth endpoints require HTTPS, no URL credentials/query/fragment; HTTP is test-only on explicit loopback")
		}
	}
	return nil
}

func random() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func httpClient(c Config) (*http.Client, error) {
	tr := http.DefaultTransport.(*http.Transport).Clone()
	tr.TLSClientConfig = &tls.Config{MinVersion: tls.VersionTLS12}
	if c.CACert != "" {
		pem, err := os.ReadFile(c.CACert)
		if err != nil {
			return nil, fmt.Errorf("cannot read OAuth CA certificate")
		}
		pool, err := x509.SystemCertPool()
		if err != nil {
			pool = x509.NewCertPool()
		}
		if !pool.AppendCertsFromPEM(pem) {
			return nil, fmt.Errorf("invalid OAuth CA certificate")
		}
		tr.TLSClientConfig.RootCAs = pool
	}
	return &http.Client{Transport: tr, Timeout: 30 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}, nil
}

// Login starts a loopback-only listener before opening the system browser.
// open is injectable for tests; it must not log the PKCE verifier or token.
func Login(ctx context.Context, c Config, open func(string) error) (Session, error) {
	if err := Validate(c); err != nil {
		return Session{}, err
	}
	client, err := httpClient(c)
	if err != nil {
		return Session{}, err
	}
	defer client.CloseIdleConnections()
	state, err := random()
	if err != nil {
		return Session{}, fmt.Errorf("cannot generate state")
	}
	verifier, err := random()
	if err != nil {
		return Session{}, fmt.Errorf("cannot generate PKCE verifier")
	}
	listener, err := net.Listen("tcp4", fmt.Sprintf("127.0.0.1:%d", c.CallbackPort))
	if err != nil {
		return Session{}, fmt.Errorf("cannot start loopback callback listener")
	}
	redirect := "http://" + listener.Addr().String() + "/callback"
	type callback struct {
		code   string
		denied bool
	}
	result := make(chan callback, 1)
	mux := http.NewServeMux()
	mux.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		if r.Method != "GET" || r.Host != listener.Addr().String() {
			http.Error(w, "Invalid callback", 400)
			return
		}
		q := r.URL.Query()
		if len(q["state"]) != 1 || subtle.ConstantTimeCompare([]byte(q.Get("state")), []byte(state)) != 1 {
			http.Error(w, "Invalid state", 400)
			return
		}
		if q.Get("error") != "" {
			select {
			case result <- callback{denied: true}:
			default:
			}
			fmt.Fprint(w, "Authorization was denied. Return to the CLI.")
			return
		}
		if len(q["code"]) != 1 || q.Get("code") == "" {
			http.Error(w, "Missing code", 400)
			return
		}
		select {
		case result <- callback{code: q.Get("code")}:
		default:
		}
		fmt.Fprint(w, "Authorization response received. Return to the CLI to check completion.")
	})
	server := &http.Server{Handler: mux, ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 10 * time.Second, WriteTimeout: 10 * time.Second, MaxHeaderBytes: 16 << 10}
	defer server.Close()
	go server.Serve(listener)
	u, _ := url.Parse(c.AuthorizeURL)
	q := u.Query()
	digest := sha256.Sum256([]byte(verifier))
	q.Set("response_type", "code")
	q.Set("client_id", c.ClientID)
	q.Set("redirect_uri", redirect)
	q.Set("state", state)
	q.Set("code_challenge", base64.RawURLEncoding.EncodeToString(digest[:]))
	q.Set("code_challenge_method", "S256")
	if len(c.Scopes) > 0 {
		q.Set("scope", strings.Join(c.Scopes, " "))
	}
	if c.Audience != "" {
		q.Set("audience", c.Audience)
	}
	u.RawQuery = q.Encode()
	if err := open(u.String()); err != nil {
		return Session{}, fmt.Errorf("cannot open browser; use --no-browser in a terminal with access to this machine's loopback callback")
	}
	var reply callback
	select {
	case reply = <-result:
	case <-ctx.Done():
		return Session{}, fmt.Errorf("authorization cancelled or timed out")
	}
	if reply.denied {
		return Session{}, fmt.Errorf("authorization denied by user or provider")
	}
	form := url.Values{"grant_type": {"authorization_code"}, "code": {reply.code}, "client_id": {c.ClientID}, "redirect_uri": {redirect}, "code_verifier": {verifier}}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.TokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return Session{}, fmt.Errorf("cannot construct token exchange")
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	res, err := client.Do(req)
	if err != nil {
		return Session{}, fmt.Errorf("token exchange failed; no credentials saved")
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		return Session{}, fmt.Errorf("token exchange returned HTTP %d; body omitted", res.StatusCode)
	}
	var token struct {
		AccessToken string `json:"access_token"`
		TokenType   string `json:"token_type"`
		ExpiresIn   int64  `json:"expires_in"`
	}
	b, err := io.ReadAll(io.LimitReader(res.Body, (1<<20)+1))
	if err != nil || len(b) > 1<<20 {
		return Session{}, fmt.Errorf("invalid token response")
	}
	if json.Unmarshal(b, &token) != nil || token.AccessToken == "" || !strings.EqualFold(token.TokenType, "Bearer") || token.ExpiresIn <= 0 || token.ExpiresIn > 31536000 || strings.ContainsAny(token.AccessToken, "\r\n") {
		return Session{}, fmt.Errorf("token response must contain access_token, Bearer token_type and positive expires_in")
	}
	return Session{AccessToken: token.AccessToken, ExpiresAt: time.Now().Add(time.Duration(token.ExpiresIn) * time.Second)}, nil
}
