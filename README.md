# 氚云 Agent CLI

[![Release](https://img.shields.io/github/v/release/cedric123123/h3yun-cli?label=%E6%9C%80%E6%96%B0%E7%89%88%E6%9C%AC)](https://github.com/cedric123123/h3yun-cli/releases)
[![Platform](https://img.shields.io/badge/%E5%B9%B3%E5%8F%B0-Windows%20%7C%20macOS%20%7C%20Linux-blue)]()
[![Go](https://img.shields.io/badge/Go-1.22%2B-00ADD8?logo=go&logoColor=white)]()

> **氚云 Agent CLI** 是一个用 Go 实现的氚云（h3yun.com）公有云命令行工具，专为 AI Agent 与自动化场景设计。采用 DWS 风格的渐进式命令发现（`schema`）、leaf Schema 自描述契约、多 profile 多环境管理、结构化错误码，并附带可直接使用的 Agent Skills。所有输出均为 JSON envelope（`ok/data/error/meta`），无需解析人读文本即可集成。

Go 实现的氚云公有云 CLI，采用 DWS 风格的渐进命令发现、leaf Schema、profile、结构化错误及配套 Agent skills。默认连接 `https://www.h3yun.com`，源码和程序均不内置实际密钥。

v0.3.0 首批吸纳：借鉴 DWS 交互，独立实现 `profile switch`（预览、备份、切回）；固定引入 go-keyring v0.2.8，macOS 会话改为 Keychain 密钥 + AES-GCM 密文，禁止静默降级明文。Windows 保持 DPAPI；Linux 仍是仅文件权限保护。来源与许可见 [THIRD_PARTY_NOTICES.md](THIRD_PARTY_NOTICES.md)。


## 开始使用

Windows 程序：`dist/h3yun.exe`；Linux amd64：`dist/h3yun-linux-amd64`；macOS Apple Silicon：`dist/h3yun-darwin-arm64`；macOS Intel：`dist/h3yun-darwin-amd64`。Unix 程序运行前添加执行权限。下面从项目目录执行；加入 PATH 后可直接用 `h3yun`。Mac 安装和自检见 [MACOS.md](MACOS.md)。

```powershell
.\dist\h3yun.exe schema
.\dist\h3yun.exe schema records --compact
.\dist\h3yun.exe schema "records create" --compact --format json
.\dist\h3yun.exe records create --schema YOUR_SCHEMA --data '@examples/record.json' --dry-run
```

根 Schema 仅显示分组；组级显示命令摘要；leaf 显示参数、effect、risk、confirmation、constraints。`schema --all` 用于全量审计。Schema 和执行参数来自同一注册表。

## 公有云鉴权与扫码边界

官方 OpenAPI 使用 EngineCode、EngineSecret，由使用者或 Agent 宿主注入环境变量，不需要私有部署地址。

```powershell
$env:H3YUN_ENGINE_CODE = '<使用者自己的 EngineCode>'
# PowerShell 7，由使用者输入；无交互 Agent 由宿主注入环境变量。
$env:H3YUN_ENGINE_SECRET = Read-Host 'EngineSecret' -MaskInput
.\dist\h3yun.exe auth status
.\dist\h3yun.exe doctor
.\dist\h3yun.exe records list --schema YOUR_SCHEMA --limit 100
```

`auth status` 只检查本地凭据/会话；`doctor` 检查配置和 CA，均不验证真实权限。CLI 不读取 `.env`，不接受命令行 EngineSecret，不保存环境变量密钥。

**已实现浏览器 OAuth + PKCE 登录模块，但目前未核实到氚云公有云 OpenAPI 提供扫码 OAuth 端点。** DWS 的令牌和氚云网页登录 Cookie 不能当作 OpenAPI 凭据。默认配置执行 `auth login` 会返回 `AUTH_NOT_CONFIGURED`，不会误报扫码登录成功。

可选浏览器模块需要真实 OAuth 服务和接受 Bearer 的 API 网关；二维码由该服务提供。网关可以转发到氚云公有云，不要求氚云私有部署。本项目没有部署网关或注册 OAuth 应用。详见 [AUTH.md](AUTH.md)。当前直接使用公有云时采用环境变量鉴权。

## 全局参数与多环境

| 参数 | 行为 |
|---|---|
| `--format json` / `-f json` | 默认一个 JSON 对象；另支持 pretty |
| `--pretty` | 兼容 v0.1 的缩进 JSON |
| `--profile NAME` | 本次环境，不修改默认 profile |
| `--config PATH` | 不含密钥的配置文件 |
| `--timeout 30s` | HTTP 超时，默认 45s，最大 60s；与浏览器等待授权时限不同 |
| `--fields parameters,effect` | 筛选 data 顶层对象字段，保留 ok/error/meta；非对象数据不裁剪 |
| `--dry-run` | 放在业务命令后；不联网、不要求密钥 |
| `--help` | 放在具体命令后，返回该命令完整契约 |

format/profile/config/timeout/fields/pretty 可置于命令前、中、后。布尔参数用 `--submit` 或 `--submit=false`。不支持 `--jq`、table/csv/ndjson 等格式时明确报错，不假装兼容整个 DWS。

与 DWS 不同：为保持 v0.1 兼容，成功和失败均输出到 stdout 的 `ok/data/error/meta` envelope；不是 DWS 原样 payload/stderr 错误协议。浏览器登录 `--no-browser` 仅将临时授权 URL 写入 stderr。

多组织配置见 `profiles.example.json`，只保存每个组织的凭据环境变量名称，不保存值。

```powershell
$env:H3YUN_CONFIG = 'C:\config\h3yun.json'
h3yun profile list
h3yun --profile sandbox profile show
h3yun records list --profile sandbox --schema YOUR_SCHEMA
h3yun schema "records delete" --compact --fields parameters,confirmation
```

profile 选择顺序：命令行 → H3YUN_PROFILE → current_profile。存在 profiles 却没有选择时返回 PROFILE_REQUIRED，不选第一项。命名 profile 不借用通用 H3YUN_BASE_URL/ENGINE_SECRET；超时可覆盖，只读模式只能加强。

修改配置文件中的默认环境（只选择已有名称，不创建账号或更改凭据）：

```sh
h3yun profile switch --config profiles.json --name sandbox --dry-run
h3yun profile switch --config profiles.json --name sandbox
h3yun profile switch --config profiles.json --name -
```

必须显式提供 `--config` 或 H3YUN_CONFIG；切换时不能同时用 `--profile`/H3YUN_PROFILE 覆盖。实际更改会在配置同目录创建唯一的 `.h3yun-config-backup-*` 文件，并在 JSON 回执返回备份路径；相同环境不重写。原有配置内容语义保留，JSON 排版会规范化。并发写入被锁拒绝，dry-run 不写文件。备份不会自动清理。进程异常退出留下 `.lock` 目录时，应先核实没有运行中的写入者，再人工移除该精确锁目录。

旧单环境配置继续支持，默认公有云。非 profile 模式可用 H3YUN_BASE_URL 覆盖以适配测试服务。ca_cert/H3YUN_CA_CERT 追加企业 CA，不关闭 TLS 验证；HTTP 必须显式 allow_http/H3YUN_ALLOW_HTTP。配置中的相对文件路径以当前工作目录为基准。

## 已实现命令

| 命令 | 能力 |
|---|---|
| records get / list | 单条、批量查询；单页最多 500 条 |
| records create / create-many | 单条、批量创建；默认草稿，--submit 生效 |
| records update / delete | 更新、删除；删除执行需 --yes |
| files upload / download | 附件上传下载，支持子表参数 |
| custom invoke / api invoke | 已部署控制器、原始 OpenAPI 包体 |
| ops list / run | 将自定义接口映射为命名业务操作 |
| profile list / show / switch | 本地环境发现、查看、持久切换及切回 |
| auth login / status / logout | 可配置浏览器授权、本地状态、清除所选会话 |
| doctor | 本地配置检查 |

```powershell
h3yun records get --schema YOUR_SCHEMA --id RECORD_ID
h3yun records list --schema YOUR_SCHEMA --offset 0 --limit 100 --count
h3yun records list --schema YOUR_SCHEMA --filter '@examples/filter.json'
h3yun records create --schema YOUR_SCHEMA --data '@examples/record.json'
h3yun records create-many --schema YOUR_SCHEMA --data '@examples/records.json' --submit
h3yun records update --schema YOUR_SCHEMA --id RECORD_ID --data '@examples/record.json'
h3yun records delete --schema YOUR_SCHEMA --id RECORD_ID --dry-run
h3yun records delete --schema YOUR_SCHEMA --id RECORD_ID --yes
h3yun files upload --schema YOUR_SCHEMA --id RECORD_ID --field FILE_FIELD --file 'C:\data\合同.pdf'
h3yun files download --attachment-id ATTACHMENT_ID --output 'C:\data\新下载文件.pdf'
```

JSON 参数支持 @文件、- 标准输入、内联 JSON。支持 UTF-8 BOM，数字转发不丢精度，输入上限 16 MiB。表单编码和字段需由真实表单提供。Filter/BizObject 自动转成平台 JSON 字符串，批量记录转字符串数组。

列表不自动翻页，参考 meta.pagination.next_offset/has_more_possible；满页只能表示可能还有数据。--filter 与 offset/limit/count 互斥。更新记录不等同于审批或提交已有草稿。

单条 `records create --submit` 在流程表单中推进到下一审批节点；批量 `records create-many --submit` 直接创建生效数据，不可替代批量发起审批。创建/更新/删除可能触发自动化和业务规则，执行及 dry-run 的 `meta.warnings` 会提示这些影响。批量建议每批不超过 500 条（建议而非服务端硬上限），复杂数据应减小批次；CLI 不自动拆分或重放。

调用方应协调同一集成的所有 Agent/进程，将请求控制在文档所述每秒 1–2 次；CLI 当前没有跨进程限流器。查询返回的人/部门补充字段不要原样写回；CLI 不掌握表单元数据，不擅自删除或转换自定义字段。

附件上传最大 10 MiB。下载不覆盖已有文件，先临时保存再硬链接发布，支持 NTFS/ext4，不支持 FAT/exFAT；父目录需存在，上限 1 GiB。响应缺少 Content-Disposition 时拒绝保存。失败清理临时文件。

子表附件：`--schema` 传主表编码，`--child-schema` 传子表编码，`--id` 传已保存的子表行 ID，`--field` 只传子表内附件字段编码，不含子表前缀。普通主表附件无需 child-schema。

## 扩展审批与管理能力

标准 8 个公共 API 已实现，但审批、组织、表单管理不能假定存在原生公共 HTTP 方法。可在公有云应用中部署自定义 RestApiController，再通过 operations_file/H3YUN_OPERATIONS 映射；示例占位符会被拒绝执行。

```powershell
h3yun custom invoke --app YOUR_APP --controller DEPLOYED_CONTROLLER --action VERIFIED_ACTION --data '@params.json'
h3yun ops list
h3yun ops run --name workflow.approve --data '@approval.json' --dry-run
```

命名操作支持 required 参数和 success_field/success_value，匹配 ReturnData 的直接子字段。未部署控制器时不能宣称审批可用。本项目未部署任何服务端业务代码。

## Agent 与验证

退出码：0 成功，2 输入，3 配置/授权/策略，4 网络/协议，5 平台业务失败，6 本地输出。错误提供 code/message/retryable，适用时含 hint/actions/outcome_unknown。不自动重试；结果未知的写入先核实，批量失败保留部分成功数据。

随包提供 skills/h3yun-shared、skills/h3yun-records、skills/h3yun-files，由使用者复制到 Agent 的 skills 目录。本次未全局安装。技能参考 DWS 渐进发现习惯，按 skill-creator 规范组织。详见 [AGENT.md](AGENT.md)。

Go 1.22+；依赖固定在 go.mod/go.sum，macOS Keychain 适配使用 go-keyring。运行 `go test ./... -cover`、`go vet ./...`、`go mod verify` 或 `scripts/build.ps1`（Windows）/`sh scripts/build.sh`（Mac/Linux）。Mac/Linux 仅交叉编译；OAuth 用本地模拟服务验证，未用真实氚云账号联调。Keychain 单测使用模拟存储，不触及系统凭据。见 [VALIDATION.md](VALIDATION.md)。

## 依据

[官方认证](https://help.h3yun.com/contents/1005/1631.html)、[查询单条](https://help.h3yun.com/api/preview/1/1006/1632)、[批量查询](https://help.h3yun.com/contents/1007/1633.html)、[创建单条](https://help.h3yun.com/contents/1008/1634.html)、[批量创建](https://help.h3yun.com/api/preview/1/1009/1635)、[更新](https://help.h3yun.com/contents/1010/1636.html)、[删除](https://help.h3yun.com/api/preview/1/1011/1637)、[上传](https://help.h3yun.com/contents/1012/1638.html)、[下载](https://help.h3yun.com/api/preview/1/1013/1639)、[自定义接口](https://help.h3yun.com/239700988/)。

DWS 交互参考本机 leaf Schema、auth login Help 及共享 skill；核对日期 2026-10-15。本工具不是钉钉或氚云官方 CLI。

## 🐛 反馈与贡献

欢迎提交 Issue！如果你遇到 Bug、有功能建议或接口契约疑问，请到 [Issues](https://github.com/cedric123123/h3yun-cli/issues) 页面反馈。提交时请尽量附上：

1. **命令与完整参数**（脱敏后，不要包含 EngineCode/EngineSecret）
2. **完整 JSON 输出**（含 `error.code`、`meta`）
3. **退出码**（0/2/3/4/5/6）与预期行为的差异
4. **运行环境**：操作系统、CLI 版本（`h3yun --help` 顶部可见）

> ⚠️ 安全提醒：Issue 内容是公开的，请勿在反馈中粘贴真实凭据、表单业务数据或下载链接中的签名参数。涉及敏感问题时可先描述现象，由维护者引导私下提供细节。也欢迎 PR！

## 📄 许可

项目依赖的来源与许可见 [THIRD_PARTY_NOTICES.md](THIRD_PARTY_NOTICES.md) 及 [licenses/](licenses/) 目录。
