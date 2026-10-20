package cli

import "github.com/cedric123123/h3yun-cli/pkg/h3yun"

type Flag struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Description string `json:"description"`
	Default     any    `json:"default,omitempty"`
	Required    bool   `json:"required,omitempty"`
}

type Command struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Mutates     bool   `json:"mutates,omitempty"`
	Flags       []Flag `json:"flags"`
}

func str(name, desc string, required bool) Flag {
	return Flag{Name: name, Type: "string", Description: desc, Required: required}
}
func boolean(name, desc string) Flag {
	return Flag{Name: name, Type: "boolean", Description: desc, Default: false}
}

func commands() []Command {
	schema := str("schema", "氚云完整表单编码 SchemaCode", true)
	id := str("id", "业务记录 ObjectId", true)
	data := str("data", "JSON 对象；支持 @文件路径、- 标准输入或内联 JSON", true)
	return []Command{
		{Name: "schema fields", Description: "通过已配置的只读控制器获取字段元数据；需真实部署契约", Flags: []Flag{schema, {Name: "operation", Type: "string", Description: "已配置的只读操作名", Default: "schema.fields"}, {Name: "schema-param", Type: "string", Description: "控制器的表单编码参数名", Default: "SchemaCode"}}},
		{Name: "profile list", Description: "列出本地配置中的环境；不读取密钥、不联网", Flags: []Flag{}},
		{Name: "profile show", Description: "查看所选 profile 的非密钥设置；不联网", Flags: []Flag{}},
		{Name: "profile switch", Description: "切换配置文件默认环境；先备份，不改凭据、不联网", Mutates: true, Flags: []Flag{str("name", "已有 profile 名称；- 表示上一个默认环境", true)}},
		{Name: "auth status", Description: "检查所选环境凭据或缓存会话，不验证远程有效性", Flags: []Flag{}},
		{Name: "auth login", Description: "打开浏览器执行 OAuth PKCE 登录；需配置实际授权服务及 Bearer API 网关", Mutates: true, Flags: []Flag{boolean("no-browser", "不自动打开浏览器，授权 URL 写到 stderr"), str("login-timeout", "等待浏览器授权时限，默认 5m，最大 30m", false)}},
		{Name: "auth logout", Description: "仅清除所选 profile 的本地登录缓存，不撤销服务端令牌", Mutates: true, Flags: []Flag{}},
		{Name: "records get", Description: "查询单条记录 LoadBizObject", Flags: []Flag{schema, id}},
		{Name: "records list", Description: "分页查询 LoadBizObjects；默认单页，--all 显式翻页", Flags: []Flag{schema, str("filter", "完整筛选 JSON 对象（@file/-/inline）；与 offset/limit/count 互斥", false), {Name: "offset", Type: "integer", Default: 0, Description: "起始行，含此行"}, {Name: "limit", Type: "integer", Default: 100, Description: "页大小 1..500"}, boolean("count", "请求记录总数"), boolean("all", "显式连续翻页；须指定 max；请求间隔至少 700ms，失败即停"), {Name: "max", Type: "integer", Description: "all 模式最大累计条数，必填 1..100000；另有累计 JSON 64 MiB 上限"}}},
		{Name: "records create", Description: "创建单条记录，默认草稿；流程表单默认停在发起节点", Mutates: true, Flags: []Flag{schema, data, boolean("submit", "普通表单生效；流程表单跳过发起节点，进入下一审批节点")}},
		{Name: "records create-many", Description: "批量创建；建议每批不超过 500 条；失败可能部分成功，不自动重试", Mutates: true, Flags: []Flag{schema, str("data", "JSON 对象数组（@file/-/inline）", true), boolean("submit", "直接创建生效数据，包括流程表单；不能用于批量发起审批")}},
		{Name: "records update", Description: "更新单条记录 UpdateBizObject", Mutates: true, Flags: []Flag{schema, id, data}},
		{Name: "records update-many", Description: "串行逐条更新；非原子批量接口，失败即停，不重试", Mutates: true, Flags: []Flag{schema, str("data", "1..500 个 {id,data} 对象组成的数组；id 唯一，data 为更新字段对象", true)}},
		{Name: "records delete", Description: "删除指定单条记录；实际执行须传 --yes", Mutates: true, Flags: []Flag{schema, id, boolean("yes", "明确执行删除")}},
		{Name: "files upload", Description: "记录保存后上传附件，最多 10 MiB", Mutates: true, Flags: []Flag{schema, str("id", "主表记录 ID；指定 child-schema 时必须传子表行 ID", true), str("field", "附件控件编码；子表内只传字段名，不含子表编码前缀", true), str("child-schema", "子表编码 ChildSchemaCode；schema 仍为主表编码", false), str("file", "本地附件路径", true)}},
		{Name: "files download", Description: "下载附件到新文件，禁止覆盖已有文件", Flags: []Flag{str("attachment-id", "附件 ID", true), str("output", "本地新文件路径", true)}},
		{Name: "api invoke", Description: "原始 OpenAPI 包体，字段按服务端协议原样发送；视为可能写入", Mutates: true, Flags: []Flag{str("body", "完整请求 JSON 对象（@file/-/inline）", true), boolean("yes", "原始 RemoveBizObject 删除需明确确认")}},
		{Name: "custom invoke", Description: "调用已在氚云部署的 RestApiController；视为可能写入", Mutates: true, Flags: []Flag{str("action", "自定义 ActionName", true), str("controller", "已部署的控制器类名", true), str("app", "应用 AppCode", true), str("data", "自定义参数对象（@file/-/inline），不能覆盖路由字段", false)}},
		{Name: "ops list", Description: "列出使用者配置的命名操作；仅本地读取", Flags: []Flag{}},
		{Name: "ops run", Description: "执行已映射的审批、组织、表单等自定义操作", Mutates: true, Flags: []Flag{str("name", "operations 文件中已配置的操作名称", true), str("data", "参数 JSON 对象（@file/-/inline）", false)}},
		{Name: "doctor", Description: "默认仅本地检查；--remote --schema 对指定表单执行一次只读探测", Flags: []Flag{boolean("remote", "读取指定表单最多一条；不输出业务记录，不代表全局权限"), str("schema", "remote 模式必填的表单编码", false)}},
	}
}

func commonFlags() []Flag {
	return []Flag{str("config", "不含密钥的 JSON 配置路径，也可通过 H3YUN_CONFIG 设置", false), str("profile", "本次请求的命名环境，不修改默认环境", false), str("format", "json 或 pretty；默认 json；简写 -f", false), str("fields", "逗号分隔的 data 相对点路径；保留嵌套结构与 ok/error/meta；数组字段可逐项投影", false), str("timeout", "每次请求超时，例如 30s", false), str("min-interval", "同一客户端请求起点最小间隔 0s..60s；多请求操作至少 700ms；非跨进程限速", false), boolean("dry-run", "校验并预览；不联网、不要求密钥、不写下载文件"), boolean("pretty", "缩进输出 JSON"), boolean("help", "显示当前命令机器可读帮助")}
}

func discovery() map[string]any {
	return map[string]any{
		"name": "h3yun", "version": Version, "commit": h3yun.Commit, "contract_version": 1,
		"commands": commands(), "common_flags": commonFlags(),
		"local_commands": []string{"schema", "help", "version", "profile list", "profile show", "profile switch", "auth status", "auth logout", "ops list", "doctor (without --remote)"},
		"syntax":         "h3yun <group> <command> [flags]; flags follow the command; booleans use --flag or --flag=false",
		"environment":    []string{"H3YUN_CONFIG", "H3YUN_BASE_URL", "H3YUN_ENGINE_CODE", "H3YUN_ENGINE_SECRET", "H3YUN_CA_CERT", "H3YUN_TIMEOUT", "H3YUN_MIN_INTERVAL", "H3YUN_ALLOW_HTTP", "H3YUN_READ_ONLY", "H3YUN_OPERATIONS", "H3YUN_PROFILE"},
		"exit_codes":     map[string]string{"0": "success", "2": "invalid arguments/input", "3": "configuration/auth/policy", "4": "transport/HTTP/protocol", "5": "H3yun API failure (may include partial results)", "6": "local output or aggregate-limit failure"},
		"response":       map[string]string{"ok": "boolean", "data": "API envelope or local result; inspect ReturnData even if ok=false", "error": "code/message/http_status/outcome_unknown", "meta": "contract_version/command/dry_run/request_policy/pagination/batch"},
		"capabilities":   map[string]any{"standard_openapi": "8 documented public cloud methods implemented; real-account verification pending", "files": "upload/download implemented", "custom_controllers": "implemented", "field_metadata": "requires a verified deployed read-only controller mapping", "workflows_organization_forms": "requires verified API contract or deployed custom controller plus operations mapping; no built-in remote methods assumed"},
		"retry_policy":   "No automatic retries. A failed write may already have committed; reconcile before retrying.",
	}
}
