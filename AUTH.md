# 浏览器授权模块

氚云公有云标准 OpenAPI 使用 EngineCode/EngineSecret。这里的 OAuth 模块是可选扩展，不是已核实的氚云公有云扫码入口。需要 OAuth 服务与 Bearer API 网关；项目不包含线上认证服务、二维码提供方或 OAuth 应用注册。

## 交互与协议

```powershell
h3yun --profile browser auth login
h3yun --profile browser auth status
h3yun --profile browser records list --schema YOUR_SCHEMA
h3yun --profile browser auth logout
```

login 启动 IPv4 回环监听并打开系统浏览器，用户在真实认证页面扫码/确认，CLI 收到回调后换取令牌并保存缓存。二维码由认证服务提供。--no-browser 把授权链接写到 stderr，需要同一机器能访问回调的浏览器；不是设备流，不支持 --device。

服务端必须支持 OAuth 公共客户端 Authorization Code + PKCE S256。注册回调为 `http://127.0.0.1:<port>/callback`，默认动态端口，可配置 callback_port 固定端口。token endpoint 接收 form-urlencoded 的 grant_type、code、client_id、redirect_uri、code_verifier，返回 access_token、Bearer token_type、正整数 expires_in。不保存 refresh token，过期重新登录，不自动重放写入。

API base_url 必须指向接受 Bearer 的 API/网关。网关验证令牌、audience、组织、用户权限，并在服务端持有 EngineSecret，转发到氚云公有云。附件下载由网关补充需要的 EngineCode。不能把用户 access_token 当作 EngineSecret，也不能把普通网页登录页当作 OAuth authorize endpoint。

配置示例（域名为占位，不能直接运行）：

```json
{
  "current_profile": "browser",
  "profiles": {
    "browser": {
      "base_url": "https://h3yun-gateway.example",
      "oauth": {
        "authorize_url": "https://identity.example/oauth/authorize",
        "token_url": "https://identity.example/oauth/token",
        "client_id": "registered-public-client-id",
        "scopes": ["h3yun.read"],
        "audience": "h3yun-gateway"
      }
    }
  }
}
```

## 会话行为

- 随机 state、常量时间比较、PKCE；回调只监听 127.0.0.1，校验 Host/方法/state/code。
- OAuth endpoint 必须 HTTPS，无 URL 凭据/query/fragment；仅显式 allow_http 的回环模拟服务允许 HTTP。ca_cert 支持企业 CA。
- token 交换与业务请求均拒绝重定向。错误不输出原始 token 响应。
- 缓存绑定 profile、API 地址、authorize/token URL、client_id、scope、audience，不跨配置复用。
- Windows 使用当前用户 DPAPI 加密。macOS 使用 Keychain 保存每个缓存绑定的随机密钥，文件保存 AES-256-GCM 密文；目录 0700、文件 0600。Keychain 失败不会降级明文，也不会替换无法读取的既有密钥。Linux 仍使用 0700/0600 的未加密文件，建议使用加密磁盘。
- 默认缓存位于系统用户配置目录的 h3yun/auth；可用 H3YUN_AUTH_DIR 指定。拒绝不安全权限及符号链接。
- auth status 仅表示本地未过期会话，不证明服务端权限或撤销状态；OAuth 回执包含 storage_backend。读缓存失败的 reason 表示本地状态无法确认，不代表服务端已登出。logout 仅删除所选本地会话，以及 Mac 上对应的 Keychain 密钥，不撤销服务端令牌，不清除环境变量密钥。
- 默认等待 5 分钟，--login-timeout 最大 30 分钟。扫码确认由使用者操作。

已用模拟服务验证回调、错误 state、PKCE、拒绝、超时、异常 token、重定向拒绝、缓存隔离、过期和清除；未连接真实扫码平台。

macOS 旧版明文会话不会自动导入，也不会静默继续使用：需重新执行 auth login，在成功授权并保存后替换所选缓存。原有文件在读取失败时保持不动。密钥还绑定缓存目录，移动目录或单独复制密文不能迁移登录，需重新授权。Keychain 锁定/拒绝访问时先由用户恢复系统访问，不要反复登录或自动删除凭据。

Save/Logout 使用同一会话锁，冲突时立即失败而不抢锁。进程崩溃残留锁需人工核实后处理。环境变量 EngineCode/EngineSecret 不写入 Keychain，也不写入配置文件。Mac 实际 Keychain 权限提示与可用性仍须真机测试；自动测试只使用内存模拟后端。

协议依据：[RFC 8252](https://www.rfc-editor.org/rfc/rfc8252)。
