# 贡献指南

感谢参与改进 h3yun CLI。提交前请先搜索现有 Issue，并为行为变化补充测试和文档。

## 本地验证

源码最低要求为 Go 1.22；官方发布使用 Go 1.27.2。提交 Pull Request 前运行：

```sh
go mod verify
go test ./... -count=1 -cover
go vet ./...
```

Linux 环境还应运行 `go test -race ./... -count=1`。Windows 可运行 `scripts/build.ps1`，macOS/Linux 可运行 `sh scripts/build.sh` 检查四平台交叉构建。

## 变更要求

- 不提交 EngineCode、EngineSecret、令牌、Cookie 或真实业务数据；测试使用本地模拟服务和假凭据。
- CLI 参数、输出或风险语义变化时，同步更新命令注册表、Schema、README、Agent Skill 和相关测试。
- 新增远程写操作时必须提供 dry-run、明确副作用，并遵守现有确认与只读策略。
- 新增依赖时说明用途，固定版本，并更新第三方许可与通知。
- 不把未核实的审批、组织或浏览器授权接口描述为已支持能力。

安全漏洞请遵循 [SECURITY.md](SECURITY.md)，不要在公开 Issue 或 Pull Request 中披露。
