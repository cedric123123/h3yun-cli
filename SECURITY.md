# 安全策略

## 支持范围

仅最新 GitHub Release 接收安全修复。发现问题后，请先确认最新版本是否仍可复现。

## 私密报告漏洞

请使用 GitHub 的[私密漏洞报告](https://github.com/cedric123123/h3yun-cli/security/advisories/new)，不要创建公开 Issue。

报告中建议包含：

- 受影响的 CLI 版本、操作系统和架构；
- 最小复现步骤、预期结果与实际结果；
- 已脱敏的 JSON 回执、退出码和必要日志；
- 影响范围及你已采取的临时缓解措施。

请勿提交 EngineCode、EngineSecret、Bearer Token、Cookie、真实表单数据或带签名的附件地址。若凭据可能已经泄露，请立即在对应平台撤销或轮换；不要等待本项目回复。

维护者会先确认报告是否收到，再通过私密 Advisory 协作验证和修复。公开时间应在修复版本可用后协调确定。

## 安全边界

本 CLI 不内置凭据，不读取 `.env`，也不接受命令行 EngineSecret。`auth status` 和 `doctor` 只做本地检查，不能证明远程权限有效。浏览器授权需要使用者自行提供经过验证的 OAuth 服务；DWS Token 和网页登录 Cookie 不能作为氚云 OpenAPI 凭据。
