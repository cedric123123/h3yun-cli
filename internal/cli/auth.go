package cli

import (
	"context"
	"fmt"
	"os"
	"time"

	"h3yun-cli/internal/auth"
	"h3yun-cli/pkg/h3yun"
)

type secretKey struct{}

func rememberSecret(ctx context.Context, value string) {
	if p, ok := ctx.Value(secretKey{}).(*[]string); ok && value != "" {
		*p = append(*p, value)
	}
}

func oauthConfig(s Settings) (auth.Config, error) {
	if s.OAuth == nil {
		return auth.Config{}, h3yun.Fail("AUTH_NOT_CONFIGURED", "configure a real OAuth provider and Bearer-capable API/gateway; EngineSecret cannot provide QR authorization", 3)
	}
	c := *s.OAuth
	c.CACert = s.CACert
	c.AllowHTTP = s.AllowHTTP
	if err := auth.Validate(c); err != nil {
		return c, h3yun.Fail("AUTH_CONFIG", err.Error(), 3)
	}
	if _, err := h3yun.Endpoint(s.BaseURL, s.InvokePath, s.AllowHTTP); err != nil {
		return c, err
	}
	return c, nil
}

func cachedSession(s Settings, env func(string) string) (auth.Session, error) {
	c, err := oauthConfig(s)
	if err != nil {
		return auth.Session{}, err
	}
	dir, err := auth.StoreDir(env("H3YUN_AUTH_DIR"))
	if err != nil {
		return auth.Session{}, h3yun.Fail("AUTH_STORE", err.Error(), 3)
	}
	session, err := auth.Load(dir, auth.Binding(s.Profile, s.BaseURL, c))
	if err != nil {
		return auth.Session{}, h3yun.Fail("AUTH_REQUIRED", "session missing, expired or unreadable; run auth login for this profile", 3)
	}
	return session, nil
}

func authCommand(ctx context.Context, command string, v values, s Settings, cfg h3yun.Config, env func(string) string) (any, error) {
	if command == "auth status" {
		if cfg.EngineCode != "" || cfg.EngineSecret != "" {
			return map[string]any{"authenticated": false, "credentials_present": cfg.EngineCode != "" && cfg.EngineSecret != "", "source": "environment", "remote_verified": false}, nil
		}
		if s.OAuth == nil {
			return map[string]any{"authenticated": false, "source": "none", "remote_verified": false}, nil
		}
		session, err := cachedSession(s, env)
		if err != nil {
			return map[string]any{"authenticated": false, "source": "oauth_cache", "storage_backend": auth.StorageBackend(), "remote_verified": false, "reason": "session_missing_expired_or_unreadable", "action": "h3yun auth login"}, nil
		}
		return map[string]any{"authenticated": true, "source": "oauth_cache", "storage_backend": auth.StorageBackend(), "expires_at": session.ExpiresAt, "remote_verified": false}, nil
	}
	c, err := oauthConfig(s)
	if err != nil {
		return nil, err
	}
	binding := auth.Binding(s.Profile, s.BaseURL, c)
	if v.b("dry-run") {
		return map[string]any{"dry_run": true, "flow": "authorization_code_pkce", "action": command, "authorize_url": c.AuthorizeURL, "token_url": c.TokenURL, "client_id": c.ClientID, "profile": s.Profile, "api_auth": "Bearer", "requires_provider_and_gateway": true}, nil
	}
	dir, err := auth.StoreDir(env("H3YUN_AUTH_DIR"))
	if err != nil {
		return nil, h3yun.Fail("AUTH_STORE", err.Error(), 3)
	}
	if command == "auth logout" {
		if err := auth.Logout(dir, binding); err != nil {
			return nil, h3yun.Fail("AUTH_STORE", "cannot remove selected session", 3)
		}
		return map[string]any{"local_session_removed": true, "server_token_revoked": false, "environment_credentials_unchanged": true}, nil
	}
	limit := 5 * time.Minute
	if v.s("login-timeout") != "" {
		limit, err = time.ParseDuration(v.s("login-timeout"))
		if err != nil || limit <= 0 || limit > 30*time.Minute {
			return nil, h3yun.Fail("INPUT", "login-timeout must be positive and at most 30m", 2)
		}
	}
	loginCtx, cancel := context.WithTimeout(ctx, limit)
	defer cancel()
	open := auth.OpenBrowser
	if v.b("no-browser") {
		open = func(u string) error {
			_, err := fmt.Fprintln(os.Stderr, "Open this authorization URL in a browser on this machine:\n"+u)
			return err
		}
	}
	session, err := auth.Login(loginCtx, c, open)
	if err != nil {
		return nil, h3yun.Fail("AUTH_LOGIN", err.Error(), 3)
	}
	session.Binding = binding
	if err := auth.Save(dir, session); err != nil {
		return nil, h3yun.Fail("AUTH_STORE", err.Error(), 3)
	}
	return map[string]any{"authenticated": true, "profile": s.Profile, "expires_at": session.ExpiresAt, "source": "browser_oauth", "storage_backend": auth.StorageBackend(), "api_verified": false}, nil
}
