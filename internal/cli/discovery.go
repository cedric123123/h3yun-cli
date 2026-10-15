package cli

import (
	"sort"
	"strings"

	"h3yun-cli/pkg/h3yun"
)

func leaf(spec Command, compact bool) map[string]any {
	params := map[string]any{}
	for _, f := range append(append([]Flag{}, spec.Flags...), commonFlags()...) {
		p := map[string]any{"type": f.Type, "required": f.Required, "description": f.Description}
		if f.Default != nil {
			p["default"] = f.Default
		}
		params[f.Name] = p
	}
	effect, risk, confirmation := "read", "low", "not_required"
	if spec.Mutates {
		effect, risk = "write", "medium"
	}
	if spec.Name == "records delete" {
		effect, risk, confirmation = "destructive", "high", "explicit_yes"
	}
	constraints := map[string]any{}
	if spec.Name == "records list" {
		constraints["mutually_exclusive"] = [][]string{{"filter", "offset"}, {"filter", "limit"}, {"filter", "count"}}
		constraints["limit"] = map[string]int{"minimum": 1, "maximum": 500}
	}
	if spec.Name == "api invoke" {
		confirmation = "conditional_remove_requires_yes"
	}
	if spec.Name == "profile switch" {
		constraints["config_source"] = "explicit --config or H3YUN_CONFIG required"
		constraints["profile_override"] = "--profile and H3YUN_PROFILE must be unset"
		constraints["scope"] = "local configuration only; no remote request or credential mutation"
		constraints["backup"] = "unique backup before changes; no implicit cleanup"
	}
	d := map[string]any{"cli_path": spec.Name, "canonical_path": strings.ReplaceAll(spec.Name, " ", "."), "description": spec.Description, "parameters": params, "effect": effect, "risk": risk, "confirmation": confirmation, "availability": "available", "constraints": constraints}
	if contract := commandContract(spec.Name); contract != nil {
		d["api_contract"] = contract
	}
	if !compact {
		d["result"] = map[string]any{"envelope": "ok/data/error/meta", "error_codes": discovery()["exit_codes"], "retry_policy": "no automatic retry"}
		d["source"] = "runtime command registry"
	}
	return d
}

func schemaQuery(args []string) (any, error) {
	path := ""
	compact, all := false, false
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--compact":
			compact = true
		case "--all":
			all = true
		case "--cli-path":
			i++
			if i >= len(args) {
				return nil, h3yun.Fail("USAGE", "--cli-path requires a value", 2)
			}
			path = args[i]
		default:
			if strings.HasPrefix(args[i], "-") || path != "" {
				return nil, h3yun.Fail("USAGE", "use schema [path] [--compact] or schema --all", 2)
			}
			path = args[i]
		}
	}
	if all && path != "" {
		return nil, h3yun.Fail("USAGE", "--all cannot be combined with a path", 2)
	}
	if all {
		result := []any{}
		for _, c := range commands() {
			result = append(result, leaf(c, compact))
		}
		return result, nil
	}
	if path != "" {
		for _, c := range commands() {
			if c.Name == path {
				return leaf(c, compact), nil
			}
		}
		result := []any{}
		for _, c := range commands() {
			if strings.HasPrefix(c.Name, path+" ") {
				result = append(result, map[string]any{"cli_path": c.Name, "description": c.Description, "effect": leaf(c, true)["effect"]})
			}
		}
		if len(result) == 0 {
			return nil, unknownCommand(path)
		}
		return map[string]any{"group": path, "commands": result}, nil
	}
	counts := map[string]int{}
	for _, c := range commands() {
		counts[strings.Fields(c.Name)[0]]++
	}
	names := []string{}
	for name := range counts {
		names = append(names, name)
	}
	sort.Strings(names)
	groups := []any{}
	for _, name := range names {
		groups = append(groups, map[string]any{"name": name, "tool_count": counts[name]})
	}
	return map[string]any{"name": "h3yun", "version": Version, "platform": "h3yun_public_cloud", "default_base_url": "https://www.h3yun.com", "groups": groups, "next": "h3yun schema <group> --compact; h3yun schema \"<group> <command>\" --compact", "extended_capabilities": "workflow/org/forms require deployed custom controllers and verified operations mappings", "authentication": map[string]string{"engine_keys": "available", "public_cloud_qr_oauth": "no documented public OpenAPI authorization endpoint", "browser_oauth": "implemented client; requires configured OAuth provider and Bearer-capable API/gateway"}}, nil
}

func unknownCommand(path string) error {
	e := h3yun.Fail("UNKNOWN_COMMAND", "command path is not registered", 2)
	group := strings.Fields(path)
	if len(group) > 0 {
		for _, c := range commands() {
			if strings.HasPrefix(c.Name, group[0]+" ") {
				e.Actions = append(e.Actions, "h3yun schema \""+c.Name+"\" --compact")
			}
		}
	}
	if len(e.Actions) == 0 {
		e.Actions = []string{"h3yun schema"}
	}
	e.Hint = "Use a registered path; no request was sent."
	return e
}

// Extract globals while preserving values of leaf flags, even when a value
// happens to look like a global option. Globals may precede or follow commands.
func globalArgs(args []string) ([]string, map[string]string, error) {
	global := map[string]string{}
	kinds := map[string]string{}
	for _, c := range commands() {
		for _, f := range c.Flags {
			kinds[f.Name] = f.Type
		}
	}
	gk := map[string]string{}
	for _, f := range commonFlags() {
		gk[f.Name] = f.Type
		kinds[f.Name] = f.Type
	}
	remaining := []string{}
	for i := 0; i < len(args); i++ {
		token := args[i]
		if token == "-f" {
			token = "--format"
		}
		if !strings.HasPrefix(token, "--") {
			remaining = append(remaining, token)
			continue
		}
		name, value, equals := strings.Cut(strings.TrimPrefix(token, "--"), "=")
		if _, ok := gk[name]; ok && name != "help" && name != "dry-run" {
			if !equals {
				if gk[name] == "boolean" {
					value = "true"
				} else {
					i++
					if i >= len(args) {
						return nil, nil, h3yun.Fail("USAGE", "missing global flag value", 2)
					}
					value = args[i]
				}
			}
			global[name] = value
			continue
		}
		remaining = append(remaining, token)
		if !equals && kinds[name] != "" && kinds[name] != "boolean" {
			i++
			if i >= len(args) {
				return nil, nil, h3yun.Fail("USAGE", "missing flag value", 2)
			}
			remaining = append(remaining, args[i])
		}
	}
	if f := global["format"]; f != "" && f != "json" && f != "pretty" {
		return nil, nil, h3yun.Fail("USAGE", "supported formats: json, pretty", 2)
	}
	return remaining, global, nil
}

func globalEnv(global map[string]string, env func(string) string) func(string) string {
	overrides := map[string]string{}
	for key, name := range map[string]string{"config": "H3YUN_CONFIG", "profile": "H3YUN_PROFILE", "timeout": "H3YUN_TIMEOUT"} {
		if value, ok := global[key]; ok {
			overrides[name] = value
		}
	}
	return func(key string) string {
		if value, ok := overrides[key]; ok {
			return value
		}
		return env(key)
	}
}

func profileList(path string, env func(string) string) (any, error) {
	doc, err := loadConfig(path, env)
	if err != nil {
		return nil, err
	}
	current := env("H3YUN_PROFILE")
	if current == "" {
		current = doc.CurrentProfile
	}
	names := []string{}
	for name := range doc.Profiles {
		names = append(names, name)
	}
	sort.Strings(names)
	items := []any{}
	for _, name := range names {
		s := doc.Profiles[name]
		items = append(items, map[string]any{"name": name, "base_url": s.BaseURL, "is_current": name == current, "read_only": s.ReadOnly})
	}
	return map[string]any{"profiles": items, "current_profile": current}, nil
}
