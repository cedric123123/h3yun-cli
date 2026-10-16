# v0.3.1 验证记录

日期：2026-10-16。本地构建环境：Windows 11 amd64，Go 1.27.2。

- `go test ./... -count=1 -cover`、`go vet ./...`、`go mod verify` 全部通过。
- 本地语句覆盖率：CLI 85.9%，HTTP 客户端 82.9%，OAuth/存储 73.4%，本地状态 83.3%。
- `govulncheck v1.9.0 ./...` 使用 Go 1.27.2 和 2026-10-08 漏洞库扫描，结果为 `No vulnerabilities found.`。
- Windows amd64、Linux amd64、macOS arm64、macOS amd64 四个 CGO=0 程序构建成功；版本回执为 0.3.1，并包含 Git commit。
- 打包程序成功生成源码 ZIP、Windows ZIP、Linux tar.gz、两个 macOS tar.gz 和 `SHA256SUMS`；本地逐项校验通过，tar 内程序权限为可执行。
- 模块路径已改为 `github.com/cedric123123/h3yun-cli`，用于 Release 后的标准 `go install`。
- CI 将在 Windows、Linux、macOS 重跑测试；Linux 额外执行 race detector。Release 工作流只接受与源码版本一致的标签，并检查干净构建的 VCS revision 与 `vcs.modified=false` 后发布。

限制：本地没有真实氚云凭据，未做线上读写、权限或审批联调；macOS/Linux 运行验证、Linux race detector 和干净标签发行物 metadata 由 GitHub Actions 完成。发布包最终 SHA-256 以对应 GitHub Release 的 `SHA256SUMS` 为准。

## v0.3.0 历史验证记录

日期：2026-10-15。构建环境：Windows amd64，Go 1.27.0。

- 43 个测试函数（含表驱动用例）通过：`go test ./... -cover`。
- CLI 包语句覆盖率 86.0%，HTTP 客户端包 83.6%，OAuth/存储包 74.2%，本地状态工具包 83.3%（Windows 执行；不代表 Darwin 专属适配的运行覆盖率）。
- 新增超时边界（CLI 参数、环境变量、配置文件及 Go 客户端）、文档来源、提交语义、回执警告、501 条批量不自动拆分和子表附件参数测试。
- `go vet ./...` 通过。
- Windows amd64、Linux amd64、macOS arm64、macOS amd64 构建成功，CGO 关闭。Mac Keychain 依赖系统自带的 /usr/bin/security；不需要额外语言运行时。
- Windows 实际程序通过 version、profile switch Schema、离线写入预览。目标机器运行不能以交叉编译代替真机测试。
- 新增模拟密钥存储测试：长 token、随机 nonce、密文完整性、目录/profile 隔离、旧明文拒绝、Keychain 读取/写入失败与损坏密钥不被覆盖。
- 新增 profile 切换测试：dry-run 零写入、精确原始备份、切回、同名无操作、缺省/未知名称/环境覆盖拒绝、并发锁拒绝。本地临时写入与替换经过测试。
- go-keyring 固定为 v0.2.8；`go mod verify` 通过。许可证及来源记录随包提供；未引入 DWS 源码或网页接口。
- Darwin arm64 的 auth 与 cli 测试可执行文件交叉编译通过；未在 Windows 执行它们。两个 Mac 发行程序的 Go 构建信息确认对应架构、CGO=0 和 go-keyring v0.2.8。
- HTTP/TLS 模拟服务覆盖企业 CA、凭据头、中文附件、禁止重定向、失败下载清理、结果未知、部分成功、业务成功条件。
- OAuth 模拟服务覆盖真实本机 HTTP 回调、错误 state、S256 PKCE 换码、拒绝、超时、异常 token、禁止 token 重定向、缓存隔离、过期、logout、Bearer 请求及令牌脱敏。Windows DPAPI 保存/解密实际运行通过；未自动打开真实浏览器。
- 三个 Agent skills 从 v0.2.0 原样复制；上一版通过 skill-creator 的 quick_validate.py，本次未改动 skill 内容。

限制：无真实氚云凭据，未做线上读写、权限、审批联调。Mac/Linux 仅交叉编译，未在目标系统运行；未签名/notarize。公共 OpenAPI 扫码 OAuth 端点尚未核实，浏览器模块依赖真实提供方和 Bearer 网关，不能当作氚云公有云扫码已接通。命名操作样例不是已部署的服务端能力。

SHA-256：

```text
957605BAC1046758717814D774A4003898680976687A022B797733B3F393D94F  dist/h3yun.exe
D06175EF5B05FC9E172E32B1085741FB4620F5FD0D9A7982345EE7788EA979C8  dist/h3yun-linux-amd64
AE043E2265D6286E2013F04C63EE3F07B8661F16E5E064894803A7EC1F741F50  dist/h3yun-darwin-arm64
49FA0718D974B5DA654359B0DE84C4A92B7D5C047993816C1C3589A93E7E9A5A  dist/h3yun-darwin-amd64
```
