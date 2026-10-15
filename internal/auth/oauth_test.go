package auth

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestBrowserPKCEFlow(t *testing.T) {
	var challenge, redirect string
	var hit int
	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hit++
		r.ParseForm()
		sum := sha256.Sum256([]byte(r.Form.Get("code_verifier")))
		if r.Form.Get("code") != "one-time-code" || r.Form.Get("grant_type") != "authorization_code" || r.Form.Get("redirect_uri") != redirect || base64.RawURLEncoding.EncodeToString(sum[:]) != challenge {
			t.Error("invalid PKCE exchange")
		}
		fmt.Fprint(w, `{"access_token":"test-bearer-token","token_type":"Bearer","expires_in":3600}`)
	}))
	defer provider.Close()
	c := Config{AuthorizeURL: provider.URL + "/authorize", TokenURL: provider.URL + "/token", ClientID: "public-cli", AllowHTTP: true}
	session, err := Login(context.Background(), c, func(raw string) error {
		u, _ := url.Parse(raw)
		q := u.Query()
		challenge = q.Get("code_challenge")
		redirect = q.Get("redirect_uri")
		if q.Get("code_challenge_method") != "S256" || q.Get("client_id") != "public-cli" || q.Get("response_type") != "code" {
			t.Error(q)
		}
		bad, err := http.Get(redirect + "?state=wrong&code=bad")
		if err != nil {
			return err
		}
		bad.Body.Close()
		if bad.StatusCode != 400 {
			t.Error("wrong state accepted")
		}
		res, err := http.Get(redirect + "?" + url.Values{"state": {q.Get("state")}, "code": {"one-time-code"}}.Encode())
		if err == nil {
			res.Body.Close()
		}
		return err
	})
	if err != nil || session.AccessToken != "test-bearer-token" || hit != 1 {
		t.Fatal(session.ExpiresAt, err, hit)
	}
}

func TestLoginDenialTimeoutAndTokenFailure(t *testing.T) {
	for _, mode := range []string{"denied", "timeout", "bad-token", "redirect"} {
		t.Run(mode, func(t *testing.T) {
			hits := 0
			provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				hits++
				if mode == "redirect" {
					w.Header().Set("Location", "https://example.invalid/leak")
					w.WriteHeader(302)
					return
				}
				fmt.Fprint(w, `{"access_token":"secret-no-expiry","token_type":"Bearer"}`)
			}))
			defer provider.Close()
			ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
			defer cancel()
			_, err := Login(ctx, Config{AuthorizeURL: provider.URL, TokenURL: provider.URL, ClientID: "test", AllowHTTP: true}, func(raw string) error {
				if mode == "timeout" {
					return nil
				}
				u, _ := url.Parse(raw)
				q := u.Query()
				params := url.Values{"state": {q.Get("state")}, "code": {"code"}}
				if mode == "denied" {
					params.Set("error", "access_denied")
				}
				r, e := http.Get(q.Get("redirect_uri") + "?" + params.Encode())
				if e == nil {
					r.Body.Close()
				}
				return e
			})
			if err == nil {
				t.Fatal("accepted failure")
			}
			if (mode == "denied" || mode == "timeout") && hits != 0 {
				t.Fatal("exchanged rejected code")
			}
		})
	}
}

func TestEndpointValidation(t *testing.T) {
	for _, endpoint := range []string{"http://evil.example/authorize", "https://user:secret@example.test/auth", "https://example.test/auth?client_secret=secret", "file:///tmp/auth"} {
		if Validate(Config{AuthorizeURL: endpoint, TokenURL: "https://example.test/token", ClientID: "x", AllowHTTP: true}) == nil {
			t.Fatal(endpoint)
		}
	}
}

func TestSessionStorageIsolationExpiryAndLogout(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "sessions")
	key := Binding("prod", "https://api.example", Config{ClientID: "app"})
	s := Session{AccessToken: "test-stored-secret", ExpiresAt: time.Now().Add(time.Hour), Binding: key}
	if err := Save(dir, s); err != nil {
		t.Fatal(err)
	}
	loaded, err := Load(dir, key)
	if err != nil || loaded.AccessToken != s.AccessToken {
		t.Fatal(err)
	}
	other := Binding("test", "https://api.example", Config{ClientID: "app"})
	if _, err := Load(dir, other); !os.IsNotExist(err) {
		t.Fatal("session reused across profiles", err)
	}
	s.ExpiresAt = time.Now().Add(-time.Second)
	if err := Save(dir, s); err != nil {
		t.Fatal("cannot replace existing session", err)
	}
	if _, err := Load(dir, key); err == nil {
		t.Fatal("accepted expired session")
	}
	if err := Logout(dir, key); err != nil {
		t.Fatal(err)
	}
	if err := Logout(dir, key); err != nil {
		t.Fatal("logout should be idempotent")
	}
	if _, err := Load(dir, key); !os.IsNotExist(err) {
		t.Fatal(err)
	}
	if err := Save(dir, Session{Binding: strings.Repeat("../", 22)}); err == nil {
		t.Fatal("invalid binding accepted")
	}
}

func TestSealRoundTrip(t *testing.T) {
	value := []byte("secret-session")
	dir, binding := t.TempDir(), strings.Repeat("a", 64)
	encrypted, err := protectSession(dir, binding, value)
	if err != nil {
		t.Fatal(err)
	}
	plain, err := unprotectSession(dir, binding, encrypted)
	if err != nil || string(plain) != string(value) {
		t.Fatal(err)
	}
}
