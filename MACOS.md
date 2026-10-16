# macOS 使用

发行包提供两个原生 Mach-O 可执行文件，最低 macOS 13，无需安装 Go、Python、Node 或额外运行时：

| 芯片 | 文件 |
|---|---|
| Apple Silicon（M 系列） | dist/h3yun-darwin-arm64 |
| Intel | dist/h3yun-darwin-amd64 |

终端运行 `uname -m` 查看当前终端架构。Apple Silicon 终端若在 Rosetta 下运行可能显示 x86_64，建议使用原生终端及 arm64 版本。

从解压后的 h3yun-cli 目录执行（Apple Silicon 示例）：

```sh
chmod +x ./dist/h3yun-darwin-arm64
./dist/h3yun-darwin-arm64 version
./dist/h3yun-darwin-arm64 schema 'records list' --compact
./dist/h3yun-darwin-arm64 records list --schema YOUR_SCHEMA --dry-run
sh scripts/smoke.sh "$PWD/dist/h3yun-darwin-arm64"
```

Intel 把文件名换成 h3yun-darwin-amd64。也可复制到自己维护的 PATH 目录并命名 h3yun；本项目不自动改系统 PATH。二进制未进行 Apple Developer ID 签名或 notarization，macOS 可能要求使用者按系统提示批准运行；不应宣称已完成签名分发。

单独的 `h3yun-v0.3.1-darwin-arm64.tar.gz` / `h3yun-v0.3.1-darwin-amd64.tar.gz` 保存执行权限。将对应包解压到新目录后，程序位于根目录的 `h3yun`，直接运行 `./h3yun version` 以及 `sh scripts/smoke.sh "$PWD/h3yun"`。源码和四平台程序一起放在完整 ZIP 包中。

## 凭据与多组织

与其他系统使用相同环境变量。下列是占位示例，不是内置密钥；实际建议由 Agent 宿主或凭据管理器注入，避免真实密钥进入 shell 历史。

```sh
export H3YUN_ENGINE_CODE='<使用者的 EngineCode>'
export H3YUN_ENGINE_SECRET='<使用者的 EngineSecret>'
./dist/h3yun-darwin-arm64 auth status
./dist/h3yun-darwin-arm64 records list --schema YOUR_SCHEMA --limit 100
```

profile 配置按 README，文件路径用 macOS 路径。脚本可通过参数数组和 stdin 调用，所有核心功能不依赖 PowerShell 或 Windows 路径。

## 浏览器与文件系统

可配置 OAuth 登录使用 macOS `open` 调用默认浏览器；回调为 127.0.0.1。公有云扫码端点尚未核实，默认 EngineCode/EngineSecret 模式不能凭空扫码登录。详见 AUTH.md。

macOS OAuth 缓存默认位于 `~/Library/Application Support/h3yun/auth`，目录 0700、文件 0600。v0.3.0 起文件是 AES-256-GCM 密文，加密密钥由 Keychain 管理，系统交互使用 `/usr/bin/security`。密钥按目录和会话绑定隔离；Keychain 不可用时报错，不落明文。旧版明文缓存需要重新登录替换；单独搬运缓存不能迁移凭据。环境变量模式不写凭据文件，也不操作 Keychain。

只有真实 OAuth 登录才会创建系统 Keychain 条目。`go test` 在 Mac 上使用模拟 Keychain，不会触碰用户系统凭据。真机验收应在专用测试 profile 下检查登录保存、读取、系统拒绝访问和 logout，不应使用实际业务账号做破坏性测试。

附件下载需要硬链接；APFS/HFS+ 本地卷可用，FAT/exFAT 不支持。路径含空格时使用引号。

## 源码构建与验证范围

安装 Go 1.22+ 后可在 Mac 原生构建：

```sh
go test ./... -cover
go vet ./...
go build -trimpath -o h3yun ./cmd/h3yun
sh scripts/smoke.sh "$PWD/h3yun"
```

当前交付在 Windows 上交叉编译了 arm64 与 amd64，并检查产物为对应 Mach-O 架构；没有真实 Mac 执行证据，也没有真实扫码服务联调证据。支持 Mac 的代码和产物已提供，真实 Mac 运行自检及系统授权提示仍需目标机器验证。
