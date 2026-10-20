package cli

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/cedric123123/h3yun-cli/internal/auth"
	"github.com/cedric123123/h3yun-cli/pkg/h3yun"
)

type Settings struct {
	BaseURL         string       `json:"base_url"`
	CACert          string       `json:"ca_cert,omitempty"`
	AllowHTTP       bool         `json:"allow_http,omitempty"`
	ReadOnly        bool         `json:"read_only,omitempty"`
	Timeout         string       `json:"timeout,omitempty"`
	MinInterval     string       `json:"min_interval,omitempty"`
	InvokePath      string       `json:"invoke_path,omitempty"`
	UploadPath      string       `json:"upload_path,omitempty"`
	DownloadPath    string       `json:"download_path,omitempty"`
	OperationsFile  string       `json:"operations_file,omitempty"`
	EngineCodeEnv   string       `json:"engine_code_env,omitempty"`
	EngineSecretEnv string       `json:"engine_secret_env,omitempty"`
	Profile         string       `json:"-"`
	OAuth           *auth.Config `json:"oauth,omitempty"`
}

type ConfigFile struct {
	Settings
	CurrentProfile  string              `json:"current_profile,omitempty"`
	PreviousProfile string              `json:"previous_profile,omitempty"`
	Profiles        map[string]Settings `json:"profiles,omitempty"`
}

func loadConfig(path string, env func(string) string) (ConfigFile, error) {
	var doc ConfigFile
	if path == "" {
		path = env("H3YUN_CONFIG")
	}
	if path == "" {
		return doc, nil
	}
	b, err := readFile(path)
	if err != nil {
		return doc, err
	}
	if err := strictJSON(b, &doc); err != nil {
		return doc, err
	}
	return doc, nil
}

func strictJSON(b []byte, into any) error {
	d := json.NewDecoder(bytes.NewReader(bytes.TrimPrefix(b, []byte{0xef, 0xbb, 0xbf})))
	d.DisallowUnknownFields()
	d.UseNumber()
	if err := d.Decode(into); err != nil {
		return h3yun.Fail("INPUT", "invalid JSON or unknown configuration property", 2)
	}
	var extra any
	if d.Decode(&extra) != io.EOF {
		return h3yun.Fail("INPUT", "expected exactly one JSON value", 2)
	}
	return nil
}

func readLimited(r io.Reader) ([]byte, error) {
	b, err := io.ReadAll(io.LimitReader(r, h3yun.MaxJSON+1))
	if err != nil || len(b) > h3yun.MaxJSON {
		return nil, h3yun.Fail("INPUT", "input unreadable or exceeds 16 MiB", 2)
	}
	return b, nil
}

func readFile(path string) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, h3yun.Fail("INPUT", "cannot open input/configuration file", 2)
	}
	defer f.Close()
	return readLimited(f)
}

func settings(path string, env func(string) string) (Settings, h3yun.Config, error) {
	doc, err := loadConfig(path, env)
	if err != nil {
		return Settings{}, h3yun.Config{}, err
	}
	s := doc.Settings
	profile := env("H3YUN_PROFILE")
	if profile == "" {
		profile = doc.CurrentProfile
	}
	if profile == "" && len(doc.Profiles) > 0 {
		return s, h3yun.Config{}, h3yun.Fail("PROFILE_REQUIRED", "select --profile or set current_profile; no profile is chosen implicitly", 3)
	}
	if profile != "" {
		selected, ok := doc.Profiles[profile]
		if !ok {
			return s, h3yun.Config{}, h3yun.Fail("PROFILE_NOT_FOUND", "selected profile does not exist in config", 3)
		}
		s = selected
		s.Profile = profile
		if s.OAuth == nil && (s.EngineCodeEnv == "" || s.EngineSecretEnv == "") {
			return s, h3yun.Config{}, h3yun.Fail("CONFIG", "named profiles require OAuth or explicit credential environment variable names", 3)
		}
	}
	if s.Timeout == "" {
		s.Timeout = "45s"
	}
	if s.InvokePath == "" {
		s.InvokePath = "/OpenApi/Invoke"
	}
	if s.UploadPath == "" {
		s.UploadPath = "/OpenApi/UploadAttachment"
	}
	if s.DownloadPath == "" {
		s.DownloadPath = "/Api/DownloadBizObjectFile"
	}
	if s.EngineCodeEnv == "" && s.OAuth == nil {
		s.EngineCodeEnv = "H3YUN_ENGINE_CODE"
	}
	if s.EngineSecretEnv == "" && s.OAuth == nil {
		s.EngineSecretEnv = "H3YUN_ENGINE_SECRET"
	}
	for name, dest := range map[string]*string{"H3YUN_BASE_URL": &s.BaseURL, "H3YUN_CA_CERT": &s.CACert, "H3YUN_TIMEOUT": &s.Timeout, "H3YUN_MIN_INTERVAL": &s.MinInterval, "H3YUN_OPERATIONS": &s.OperationsFile} {
		if profile != "" && name != "H3YUN_TIMEOUT" && name != "H3YUN_MIN_INTERVAL" {
			continue
		}
		if v := env(name); v != "" {
			*dest = v
		}
	}
	for name, dest := range map[string]*bool{"H3YUN_ALLOW_HTTP": &s.AllowHTTP, "H3YUN_READ_ONLY": &s.ReadOnly} {
		if profile != "" && name == "H3YUN_ALLOW_HTTP" {
			continue
		}
		if v := env(name); v != "" {
			b, err := strconv.ParseBool(v)
			if err != nil {
				return s, h3yun.Config{}, h3yun.Fail("CONFIG", name+" must be true or false", 3)
			}
			if name == "H3YUN_READ_ONLY" {
				*dest = *dest || b
			} else {
				*dest = b
			}
		}
	}
	if s.OAuth != nil && s.BaseURL == "" {
		return s, h3yun.Config{}, h3yun.Fail("CONFIG", "OAuth profiles require an explicit Bearer-capable API base_url; public H3yun OpenAPI uses engine credentials", 3)
	}
	if s.BaseURL == "" {
		s.BaseURL = "https://www.h3yun.com"
	}
	timeout, err := time.ParseDuration(s.Timeout)
	if err != nil || timeout <= 0 || timeout > h3yun.MaxTimeout {
		return s, h3yun.Config{}, h3yun.Fail("CONFIG", "timeout must be positive and at most 60s", 3)
	}
	cfg := h3yun.Config{BaseURL: s.BaseURL, EngineCode: env(s.EngineCodeEnv), EngineSecret: env(s.EngineSecretEnv), CACert: s.CACert, AllowHTTP: s.AllowHTTP, Timeout: timeout, InvokePath: s.InvokePath, UploadPath: s.UploadPath, DownloadPath: s.DownloadPath}
	if s.MinInterval != "" {
		interval, err := time.ParseDuration(s.MinInterval)
		if err != nil || interval < 0 || interval > time.Minute {
			return s, cfg, h3yun.Fail("CONFIG", "min_interval must be between 0s and 60s", 3)
		}
		cfg.MinInterval = interval
	}
	return s, cfg, nil
}

type Operation struct {
	Action       string   `json:"action"`
	Controller   string   `json:"controller"`
	AppCode      string   `json:"app_code"`
	Description  string   `json:"description"`
	ReadOnly     bool     `json:"read_only"`
	Required     []string `json:"required"`
	SuccessField string   `json:"success_field,omitempty"`
	SuccessValue any      `json:"success_value,omitempty"`
}

func operations(path string) (map[string]Operation, error) {
	if path == "" {
		return map[string]Operation{}, nil
	}
	b, err := readFile(path)
	if err != nil {
		return nil, err
	}
	var ops map[string]Operation
	if err := strictJSON(b, &ops); err != nil {
		return nil, err
	}
	if ops == nil {
		return nil, h3yun.Fail("CONFIG", "operations file must be a JSON object", 3)
	}
	for name, op := range ops {
		if name == "" || op.Action == "" || op.Controller == "" || op.AppCode == "" {
			return nil, h3yun.Fail("CONFIG", "each operation requires name, action, controller and app_code", 3)
		}
		if standardAction(op.Action) {
			return nil, h3yun.Fail("CONFIG", "custom operations cannot use reserved standard ActionName", 3)
		}
		for _, value := range []string{op.Action, op.Controller, op.AppCode} {
			if strings.HasPrefix(value, "REPLACE_WITH_") {
				return nil, h3yun.Fail("CAPABILITY_UNCONFIGURED", "replace example operation placeholders with verified deployed controller values", 3)
			}
		}
	}
	return ops, nil
}

func standardAction(action string) bool {
	switch action {
	case "LoadBizObject", "LoadBizObjects", "CreateBizObject", "CreateBizObjects", "UpdateBizObject", "RemoveBizObject":
		return true
	}
	return false
}
