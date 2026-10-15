# Third-party notices and adoption record

## Included dependency

`github.com/zalando/go-keyring v0.2.8` is pinned in go.mod/go.sum and imported by the macOS credential adapter. Its original source is not modified or vendored. macOS uses `/usr/bin/security`; the application stores only a 32-byte encryption key there, and stores AES-256-GCM ciphertext in its own session directory.

- Repository: https://github.com/zalando/go-keyring/tree/v0.2.8
- Root license: MIT, Copyright (c) 2016 Zalando SE. Full text: licenses/go-keyring-MIT.txt.
- Darwin implementation includes Apache-2.0-licensed code, Copyright 2013 Google Inc. All Rights Reserved. Full license: licenses/Apache-2.0.txt. See upstream keyring_darwin.go for the original notice.
- Internal shellescape includes MIT code, Copyright 2016 Alessio Treglia. Full text: licenses/shellescape-MIT.txt.

The Go module graph also includes github.com/danieljoos/wincred v1.2.3, github.com/godbus/dbus/v5 v5.2.2, and golang.org/x/sys v0.27.0. Their upstream license files are included in licenses. This application's Windows backend remains its own DPAPI implementation; Linux remains the documented permission-protected file backend. Test dependencies are recorded in go.sum and not vendored into this distribution.

## Design reference, not incorporated code

DingTalk-Real-AI/dingtalk-workspace-cli (Apache-2.0) inspired explicit profile selection, a persisted previous-profile switch, and scoped session handling. Our implementation is independent and preserves the existing h3yun JSON contract. No DWS source, credentials, endpoint catalog or bundled proprietary SDK was copied.

Reference: https://github.com/DingTalk-Real-AI/dingtalk-workspace-cli

Not incorporated: Cobra, cli/oauth, fast-h3yun, h3yun-cmax or other unofficial web-login implementations. Public H3yun OpenAPI authentication remains user-provided EngineCode/EngineSecret. Keychain storage does not create a new authorization provider.
