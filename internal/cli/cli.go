package cli

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/url"
	"strconv"
	"strings"

	"h3yun-cli/pkg/h3yun"
)

const Version = h3yun.Version

type Result struct {
	OK    bool           `json:"ok"`
	Data  any            `json:"data,omitempty"`
	Error *h3yun.Error   `json:"error,omitempty"`
	Meta  map[string]any `json:"meta"`
}

type values struct {
	strings map[string]*string
	bools   map[string]*bool
	ints    map[string]*int
	set     map[string]bool
}

func (v values) s(k string) string {
	if p := v.strings[k]; p != nil {
		return *p
	}
	return ""
}
func (v values) b(k string) bool {
	if p := v.bools[k]; p != nil {
		return *p
	}
	return false
}
func (v values) n(k string) int {
	if p := v.ints[k]; p != nil {
		return *p
	}
	return 0
}

func parse(spec Command, args []string) (values, error) {
	v := values{strings: map[string]*string{}, bools: map[string]*bool{}, ints: map[string]*int{}, set: map[string]bool{}}
	fs := flag.NewFlagSet(spec.Name, flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	for _, f := range append(append([]Flag{}, spec.Flags...), commonFlags()...) {
		switch f.Type {
		case "string":
			v.strings[f.Name] = fs.String(f.Name, "", f.Description)
		case "boolean":
			v.bools[f.Name] = fs.Bool(f.Name, false, f.Description)
		case "integer":
			d, _ := f.Default.(int)
			v.ints[f.Name] = fs.Int(f.Name, d, f.Description)
		}
	}
	if err := fs.Parse(args); err != nil {
		return v, h3yun.Fail("USAGE", "invalid flags; run this command with --help", 2)
	}
	if fs.NArg() != 0 {
		return v, h3yun.Fail("USAGE", "unexpected positional arguments; flags must follow the command", 2)
	}
	fs.Visit(func(f *flag.Flag) { v.set[f.Name] = true })
	if !v.b("help") {
		for _, f := range spec.Flags {
			if f.Required && strings.TrimSpace(v.s(f.Name)) == "" {
				return v, h3yun.Fail("USAGE", "required flag --"+f.Name+" is missing", 2)
			}
		}
	}
	return v, nil
}

func readJSON(source string, stdin io.Reader, array bool) (any, error) {
	var b []byte
	var err error
	switch {
	case source == "-":
		b, err = readLimited(stdin)
	case strings.HasPrefix(source, "@"):
		b, err = readFile(source[1:])
	default:
		b = []byte(source)
	}
	if err != nil {
		return nil, err
	}
	if len(b) > h3yun.MaxJSON {
		return nil, h3yun.Fail("INPUT", "JSON input exceeds 16 MiB", 2)
	}
	var data any
	if err := strictJSON(b, &data); err != nil {
		return nil, err
	}
	if array {
		items, ok := data.([]any)
		if !ok || len(items) == 0 {
			return nil, h3yun.Fail("INPUT", "expected a nonempty JSON array of objects", 2)
		}
		for _, item := range items {
			if _, ok := item.(map[string]any); !ok {
				return nil, h3yun.Fail("INPUT", "every batch item must be a JSON object", 2)
			}
		}
	} else if _, ok := data.(map[string]any); !ok {
		return nil, h3yun.Fail("INPUT", "expected a JSON object", 2)
	}
	return data, nil
}

func encoded(v any) string { b, _ := json.Marshal(v); return string(b) }

func integer(v any) (int, bool) {
	switch n := v.(type) {
	case int:
		return n, true
	case json.Number:
		i, err := strconv.Atoi(n.String())
		return i, err == nil
	}
	return 0, false
}

func makeFilter(v values, stdin io.Reader) (map[string]any, error) {
	f := map[string]any{"FromRowNum": v.n("offset"), "ToRowNum": v.n("offset") + v.n("limit"), "RequireCount": v.b("count"), "ReturnItems": []any{}, "SortByCollection": []any{}, "Matcher": map[string]any{"Type": "And", "Matchers": []any{}}}
	if v.s("filter") != "" {
		if v.set["offset"] || v.set["limit"] || v.set["count"] {
			return nil, h3yun.Fail("USAGE", "--filter is mutually exclusive with --offset/--limit/--count", 2)
		}
		input, err := readJSON(v.s("filter"), stdin, false)
		if err != nil {
			return nil, err
		}
		for k, value := range input.(map[string]any) {
			f[k] = value
		}
	}
	from, ok1 := integer(f["FromRowNum"])
	to, ok2 := integer(f["ToRowNum"])
	if !ok1 || !ok2 || from < 0 || to <= from || to-from > 500 {
		return nil, h3yun.Fail("INPUT", "filter must use integer row bounds: 0 <= FromRowNum < ToRowNum, page size <= 500", 2)
	}
	if _, ok := f["RequireCount"].(bool); !ok {
		return nil, h3yun.Fail("INPUT", "RequireCount must be boolean", 2)
	}
	for _, key := range []string{"ReturnItems", "SortByCollection"} {
		if _, ok := f[key].([]any); !ok {
			return nil, h3yun.Fail("INPUT", key+" must be an array", 2)
		}
	}
	if _, ok := f["Matcher"].(map[string]any); !ok {
		return nil, h3yun.Fail("INPUT", "Matcher must be an object", 2)
	}
	return f, nil
}

func mergeCustom(body map[string]any, source string, stdin io.Reader) error {
	if source == "" {
		return nil
	}
	data, err := readJSON(source, stdin, false)
	if err != nil {
		return err
	}
	for key, value := range data.(map[string]any) {
		for _, reserved := range []string{"ActionName", "Controller", "AppCode", "EngineCode", "EngineSecret"} {
			if strings.EqualFold(key, reserved) {
				return h3yun.Fail("INPUT", "custom parameters cannot contain routing or credential fields", 2)
			}
		}
		body[key] = value
	}
	return nil
}

func run(ctx context.Context, command string, v values, stdin io.Reader, env func(string) string, meta map[string]any) (any, error) {
	if command == "profile list" {
		return profileList(v.s("config"), env)
	}
	if command == "profile switch" {
		if v.b("dry-run") {
			meta["dry_run"] = true
		}
		return switchProfile(v, env)
	}
	s, cfg, err := settings(v.s("config"), env)
	if err != nil {
		return nil, err
	}
	meta["profile"] = s.Profile
	if command == "profile show" {
		return s, nil
	}
	if strings.HasPrefix(command, "auth ") {
		return authCommand(ctx, command, v, s, cfg, env)
	}
	if command == "ops list" {
		return operations(s.OperationsFile)
	}
	if !v.b("dry-run") && s.OAuth != nil && cfg.EngineCode == "" && cfg.EngineSecret == "" {
		session, err := cachedSession(s, env)
		if err != nil {
			return nil, err
		}
		cfg.AccessToken = session.AccessToken
		rememberSecret(ctx, session.AccessToken)
	}
	if command == "doctor" {
		if v.b("dry-run") {
			meta["dry_run"] = true
			if cfg.BaseURL != "" {
				if _, err := h3yun.Endpoint(cfg.BaseURL, cfg.InvokePath, cfg.AllowHTTP); err != nil {
					return nil, err
				}
			}
			return map[string]any{"base_url_configured": cfg.BaseURL != "", "credentials_present": cfg.EngineCode != "" && cfg.EngineSecret != "", "network_tested": false}, nil
		}
		c, err := h3yun.New(cfg)
		if err != nil {
			return nil, err
		}
		c.Close()
		return map[string]any{"configuration_valid": true, "network_tested": false, "credentials_present": true, "base_url": cfg.BaseURL, "read_only": s.ReadOnly}, nil
	}
	path := cfg.InvokePath
	body := map[string]any{}
	write := true
	var filter map[string]any
	var fields url.Values
	var operation *Operation
	switch command {
	case "records get", "records delete":
		action := "LoadBizObject"
		write = false
		if command == "records delete" {
			action = "RemoveBizObject"
			write = true
		}
		body = map[string]any{"ActionName": action, "SchemaCode": v.s("schema"), "BizObjectId": v.s("id")}
	case "records list":
		filter, err = makeFilter(v, stdin)
		if err != nil {
			return nil, err
		}
		write = false
		body = map[string]any{"ActionName": "LoadBizObjects", "SchemaCode": v.s("schema"), "Filter": encoded(filter)}
	case "records create", "records update", "records create-many":
		data, err := readJSON(v.s("data"), stdin, command == "records create-many")
		if err != nil {
			return nil, err
		}
		body["SchemaCode"] = v.s("schema")
		if command == "records create-many" {
			items := []string{}
			for _, item := range data.([]any) {
				items = append(items, encoded(item))
			}
			body["ActionName"] = "CreateBizObjects"
			body["BizObjectArray"] = items
			body["IsSubmit"] = v.b("submit")
		} else {
			body["BizObject"] = encoded(data)
			if command == "records update" {
				body["ActionName"] = "UpdateBizObject"
				body["BizObjectId"] = v.s("id")
			} else {
				body["ActionName"] = "CreateBizObject"
				body["IsSubmit"] = v.b("submit")
			}
		}
	case "api invoke":
		data, err := readJSON(v.s("body"), stdin, false)
		if err != nil {
			return nil, err
		}
		body = data.(map[string]any)
		action, ok := body["ActionName"].(string)
		if !ok || strings.TrimSpace(action) == "" {
			return nil, h3yun.Fail("INPUT", "body requires nonempty string ActionName", 2)
		}
		for key := range body {
			if strings.EqualFold(key, "EngineCode") || strings.EqualFold(key, "EngineSecret") {
				return nil, h3yun.Fail("INPUT", "credentials must be supplied through environment variables, not JSON body", 2)
			}
		}
		// Even standard-looking raw calls remain conservative for read-only policy.
	case "custom invoke":
		if standardAction(v.s("action")) {
			return nil, h3yun.Fail("INPUT", "custom action cannot use reserved standard ActionName", 2)
		}
		body = map[string]any{"ActionName": v.s("action"), "Controller": v.s("controller"), "AppCode": v.s("app")}
		if err := mergeCustom(body, v.s("data"), stdin); err != nil {
			return nil, err
		}
	case "ops run":
		ops, err := operations(s.OperationsFile)
		if err != nil {
			return nil, err
		}
		op, ok := ops[v.s("name")]
		if !ok {
			return nil, h3yun.Fail("CAPABILITY_UNCONFIGURED", "operation not configured; supply a verified deployed controller mapping through H3YUN_OPERATIONS", 3)
		}
		write = !op.ReadOnly
		operation = &op
		body = map[string]any{"ActionName": op.Action, "Controller": op.Controller, "AppCode": op.AppCode}
		if err := mergeCustom(body, v.s("data"), stdin); err != nil {
			return nil, err
		}
		for _, key := range op.Required {
			if body[key] == nil {
				return nil, h3yun.Fail("INPUT", "required operation parameter missing: "+key, 2)
			}
		}
		meta["operation"] = v.s("name")
	case "files upload":
		path = cfg.UploadPath
		info, err := h3yun.UploadInfo(v.s("file"))
		if err != nil {
			return nil, err
		}
		fields = url.Values{"SchemaCode": {v.s("schema")}, "BizObjectId": {v.s("id")}, "FilePropertyName": {v.s("field")}}
		if v.s("child-schema") != "" {
			fields.Set("ChildSchemaCode", v.s("child-schema"))
		}
		body = map[string]any{"file": v.s("file"), "bytes": info.Size(), "multipart_field": "media", "query": fields}
	case "files download":
		path = cfg.DownloadPath
		write = false
		body = map[string]any{"attachmentId": v.s("attachment-id"), "EngineCode": "<from environment>", "output": v.s("output")}
	default:
		return nil, h3yun.Fail("USAGE", "unknown command", 2)
	}
	if warnings := operationWarnings(command, body); len(warnings) > 0 {
		meta["warnings"] = warnings
	}
	endpoint := "<H3YUN_BASE_URL>" + path
	if cfg.BaseURL != "" {
		endpoint, err = h3yun.Endpoint(cfg.BaseURL, path, cfg.AllowHTTP)
		if err != nil {
			return nil, err
		}
	}
	if v.b("dry-run") {
		meta["dry_run"] = true
		authentication := "EngineCode/EngineSecret from environment; omitted"
		if s.OAuth != nil && cfg.EngineCode == "" && cfg.EngineSecret == "" {
			authentication = "Bearer token from configured OAuth session; omitted"
		}
		return map[string]any{"method": "POST", "url": endpoint, "body": body, "mutates": write, "blocked_by_read_only": s.ReadOnly && write, "authentication": authentication}, nil
	}
	if s.ReadOnly && write {
		return nil, h3yun.Fail("READ_ONLY", "write or unclassified operation blocked by read-only mode", 3)
	}
	if body["ActionName"] == "RemoveBizObject" && !v.b("yes") {
		return nil, h3yun.Fail("CONFIRMATION_REQUIRED", "delete requires --yes; inspect --dry-run first", 3)
	}
	c, err := h3yun.New(cfg)
	if err != nil {
		return nil, err
	}
	defer c.Close()
	if command == "files upload" {
		return c.Upload(ctx, v.s("file"), fields)
	}
	if command == "files download" {
		return c.Download(ctx, v.s("attachment-id"), v.s("output"))
	}
	result, err := c.Invoke(ctx, body, write)
	if err == nil && operation != nil && operation.SuccessField != "" {
		returned, _ := result["ReturnData"].(map[string]any)
		value, exists := returned[operation.SuccessField]
		if !exists || encoded(value) != encoded(operation.SuccessValue) {
			return result, h3yun.Fail("CUSTOM_BUSINESS", "custom operation success condition did not match; inspect ReturnData", 5)
		}
	}
	if err == nil && filter != nil {
		data, _ := result["ReturnData"].(map[string]any)
		rows, ok := data["BizObjectArray"].([]any)
		if !ok {
			return result, h3yun.Fail("PROTOCOL", "list response must contain ReturnData.BizObjectArray array", 4)
		}
		from, _ := integer(filter["FromRowNum"])
		to, _ := integer(filter["ToRowNum"])
		meta["pagination"] = map[string]any{"offset": from, "limit": to - from, "returned": len(rows), "next_offset": to, "has_more_possible": len(rows) == to-from}
	}
	return result, err
}

// Run emits one JSON value to stdout on both success and failure, never prompts,
// and returns a stable process exit code. Secrets are redacted before output.
func Run(ctx context.Context, args []string, stdin io.Reader, stdout io.Writer, env func(string) string) int {
	args, globals, globalErr := globalArgs(args)
	env = globalEnv(globals, env)
	secrets := []string{}
	ctx = context.WithValue(ctx, secretKey{}, &secrets)
	originalEnv := env
	env = func(key string) string {
		value := originalEnv(key)
		switch key {
		case "H3YUN_CONFIG", "H3YUN_PROFILE", "H3YUN_TIMEOUT", "H3YUN_BASE_URL", "H3YUN_CA_CERT", "H3YUN_OPERATIONS", "H3YUN_ALLOW_HTTP", "H3YUN_READ_ONLY", "H3YUN_AUTH_DIR":
		default:
			if value != "" {
				secrets = append(secrets, value)
			}
		}
		return value
	}
	if len(args) == 0 {
		args = []string{"help"}
	}
	meta := map[string]any{"contract_version": 1}
	var data any
	var err error = globalErr
	pretty := globals["format"] == "pretty" || globals["pretty"] == "true"
	command := "help"
	if len(args) > 0 {
		command = args[0]
	}
	if len(args) > 1 && (command == "records" || command == "files" || command == "api" || command == "custom" || command == "ops" || command == "profile" || command == "auth") {
		command += " " + args[1]
		args = append([]string{command}, args[2:]...)
	}
	meta["command"] = command
	if err == nil {
		switch command {
		case "schema", "help", "--help", "-h", "version", "--version":
			if command == "version" || command == "--version" {
				if len(args) > 1 {
					err = h3yun.Fail("USAGE", "unexpected version arguments", 2)
				}
				data = map[string]any{"version": Version}
			} else {
				data, err = schemaQuery(args[1:])
			}
		default:
			found := false
			for _, spec := range commands() {
				if spec.Name != command {
					continue
				}
				found = true
				var v values
				v, err = parse(spec, args[1:])
				pretty = pretty || v.b("pretty")
				if err == nil {
					if v.b("help") {
						data = leaf(spec, false)
					} else {
						data, err = run(ctx, command, v, stdin, env, meta)
					}
				}
				break
			}
			if !found {
				err = unknownCommand(command)
			}
		}
	}
	r := Result{OK: err == nil, Data: data, Meta: meta}
	exit := 0
	if err != nil {
		if e, ok := err.(*h3yun.Error); ok {
			if e.Hint == "" {
				switch e.Code {
				case "USAGE", "INPUT":
					e.Hint = "Inspect the exact leaf schema; no automatic retry."
					e.Actions = []string{"h3yun schema \"" + command + "\" --compact"}
				case "AUTH_REQUIRED", "AUTH_NOT_CONFIGURED", "AUTH_CONFIG":
					e.Hint = "Public H3yun uses user-supplied EngineCode/EngineSecret. Browser login requires a configured OAuth provider and Bearer API."
					e.Actions = []string{"h3yun auth status"}
				case "PROFILE_REQUIRED", "PROFILE_NOT_FOUND":
					e.Hint = "Select a profile from local configuration."
					e.Actions = []string{"h3yun profile list"}
				case "HTTP", "TRANSPORT", "PROTOCOL":
					e.Hint = "Inspect outcome_unknown and reconcile writes before retrying."
				}
			}
			r.Error = e
			exit = e.Exit
		} else {
			r.Error = h3yun.Fail("INTERNAL", "unexpected internal error", 6)
			exit = 6
		}
	}
	// Redact string values before serialization, never JSON keys or syntax.
	serialized, marshalErr := json.Marshal(r)
	var safe any
	if marshalErr == nil {
		decoder := json.NewDecoder(strings.NewReader(string(serialized)))
		decoder.UseNumber()
		marshalErr = decoder.Decode(&safe)
	}
	if marshalErr != nil {
		safe = map[string]any{"ok": false, "error": map[string]any{"code": "INTERNAL", "message": "could not encode output"}}
		exit = 6
	}
	secrets = append(secrets, env("H3YUN_ENGINE_SECRET"), env("H3YUN_ENGINE_CODE"))
	safe = redact(safe, secrets)
	if fields := globals["fields"]; fields != "" {
		root, _ := safe.(map[string]any)
		object, ok := root["data"].(map[string]any)
		if ok {
			projected := map[string]any{}
			for _, field := range strings.Split(fields, ",") {
				field = strings.TrimSpace(field)
				if value, exists := object[field]; exists {
					projected[field] = value
				}
			}
			root["data"] = projected
		}
	}
	var b []byte
	if pretty {
		b, err = json.MarshalIndent(safe, "", "  ")
	} else {
		b, err = json.Marshal(safe)
	}
	if err != nil {
		b = []byte(`{"ok":false,"error":{"code":"INTERNAL","message":"could not encode output"}}`)
		exit = 6
	}
	if _, err := fmt.Fprintln(stdout, string(b)); err != nil {
		return 6
	}
	return exit
}

func redact(value any, secrets []string) any {
	switch v := value.(type) {
	case string:
		for _, secret := range secrets {
			if secret != "" {
				v = strings.ReplaceAll(v, secret, "[REDACTED]")
			}
		}
		return v
	case map[string]any:
		for key, child := range v {
			v[key] = redact(child, secrets)
		}
	case []any:
		for i, child := range v {
			v[i] = redact(child, secrets)
		}
	}
	return value
}
