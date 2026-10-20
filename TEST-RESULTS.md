# TEST-RESULTS v0.4.0 — QA-20261020-001 工作区验证记录

执行日期：2026-10-20（北京时间）。执行环境：Windows 11 amd64、Go 1.27.2。
源码定位：基于 commit `570aed9` 的 v0.4.0 未提交工作区快照；这不是最终发布 commit，也不是 GitHub Release 发行物验证。
关联用例：[TEST-CASES.md](TEST-CASES.md)。本节只记录本轮实际执行结果；未执行的真实账号、目标系统和发行物项目保持 NOT_RUN。

## 1. 本轮已执行检查

| 检查 | 状态 | 实测结果 |
|---|---|---|
| `go test ./... -count=1 -cover` | **PASS** | exit 0；56 个测试函数及其表驱动子用例通过 |
| `go vet ./...` | **PASS** | exit 0 |
| `go mod verify` | **PASS** | exit 0；`all modules verified` |
| `govulncheck v1.9.0 ./...` | **PASS** | Go 1.27.2；漏洞库更新时间 2026-10-20 00:20:56 UTC；`No vulnerabilities found.` |
| `actionlint v1.7.12` | **PASS** | exit 0；GitHub Actions 工作流无诊断 |
| skill-creator `quick_validate.py` | **PASS** | h3yun-shared、h3yun-records、h3yun-files 均有效 |
| 四平台 CGO=0 交叉构建 | **PASS** | Windows amd64、Linux amd64、macOS arm64/amd64 均成功；Windows 程序返回 version=0.4.0 |
| 本地打包与 SHA-256 | **PASS** | 生成源码 ZIP、Windows ZIP、Linux/macOS 包及 SHA256SUMS；macOS tar 内程序权限为 0755，新文档与 skills 已包含 |
| internal/cli 覆盖率 | 记录 | 87.9% statements |
| pkg/h3yun 覆盖率 | 记录 | 84.2% statements |
| internal/auth 覆盖率 | 记录 | 73.4% statements |
| internal/localstate 覆盖率 | 记录 | 83.3% statements |
| cmd/h3yun、scripts/package 覆盖率 | 记录 | 0.0%；不隐去未覆盖入口 |

## 2. v0.4.0 新增自动回归

以下结果均来自本地逻辑或绑定回环地址的 `httptest` 服务，使用假凭据；不代表真实氚云租户已联调。

| 范围 | 状态 | 本轮实际覆盖 |
|---|---|---|
| 显式自动翻页 | **PASS** | 上限约束、Filter保留、末页缩小、短页/max停止、API/HTTP/协议/超页失败后的部分结果与next_offset、64MiB聚合上限 |
| 串行批量更新 | **PASS** | 成功顺序、大整数保真、业务失败failed_ids、HTTP/协议失败unknown_ids、取消后unprocessed_ids、dry-run及只读策略 |
| 字段元数据 | **PASS** | 缺映射、write映射、保留参数、额外required、已配置控制器线形、metadata_source及自定义业务成功条件 |
| 远程诊断 | **PASS** | 必须显式remote+schema；成功/平台失败/协议失败分类；不回显读取到的私有记录；scope限定为指定表单读取 |
| 点路径投影 | **PASS** | 嵌套对象、数组逐项、字面点号键、父级优先及非法路径在执行前拒绝 |
| 请求起点间隔 | **PASS** | 配置/环境覆盖与0..60s边界；同Client并发请求间隔；等待取消；取消的写请求未发送且outcome_unknown=false |

对应新增测试函数为 UT-044～UT-056。这里的 PASS 表示这些自动测试函数通过，不把它们自动换算成 29 条 V4 验收用例全部 PASS。

## 3. 明确未执行

| 项目 | 状态 | 原因/后续证据 |
|---|---|---|
| 真实氚云标准API读写、`--all`、`update-many` | **NOT_RUN** | 本轮未提供已授权测试租户、表单与真实凭据 |
| 已部署字段元数据控制器 | **NOT_RUN** | 未提供真实action/controller/app_code及字段契约，不能用模拟响应证明准确性 |
| `doctor --remote` 真实权限判定 | **NOT_RUN** | 只完成假服务分类测试；真实账号的有权/无权/不存在表单组合待测 |
| Windows最终发行程序、macOS/Linux真机 | **NOT_RUN** | 本轮执行对象是源码工作区，不是最终包；未在目标系统运行 |
| Linux race detector | **NOT_RUN** | Windows 本地未运行；由 GitHub Linux CI 执行 |
| 最终提交/标签发行物复验 | **NOT_RUN** | 本轮构建来自未提交工作区，VCS metadata 正确显示 modified=true；须在最终提交和标签上重跑并要求 modified=false |
| 169条验收用例逐条执行 | **NOT_RUN** | 自动关联不等于完整验收；R/OS/F变体仍按TEST-CASES执行 |

## 4. 本轮结论

v0.4.0 新增代码在当前 Windows 工作区通过单元/模拟服务回归、静态检查和依赖完整性检查；本轮未发现自动回归失败。发布结论仍取决于最终 commit 的CI、构建/打包/校验和，以及真实氚云测试租户和目标系统验收。没有真实账号测试证据，因此不能宣称字段发现、远程诊断、批量更新或自动翻页已在氚云线上验证。

---

# TEST-RESULTS v0.3.0 — QA-20261016-001 历史执行记录

执行日期：2026-10-16（北京时间）。执行环境：Windows 11 amd64、Go 1.27.0、bash（PortableGit）。
测试程序：本地构建 `h3yun.exe`（源码 commit `08714a0`）。该记录对应 v0.3.0 测试时点，不代表后续分支或 Release 的当前状态。
关联用例：[TEST-CASES.md](TEST-CASES.md)。本文件只记录实际执行的用例及结果，未执行的用例保持 NOT_RUN。

## 1. 自动回归基线（复现）

| 检查 | 结果 | 备注 |
|---|---|---|
| go test ./... -count=1 -cover | PASS (exit 0) | auth 74.2% / cli 86.0% / localstate 83.3% / h3yun 83.6%；cmd、scripts/package 0.0% |
| go vet ./... | PASS (exit 0) | — |
| go mod verify | PASS (exit 0) | all modules verified |
| 测试函数计数 | 43 | 与 TEST-CASES 第 7 节索引一致 |

## 2. CLI 模块（10 条）

| ID | 状态 | 实测要点 |
|---|---|---|
| CLI-001 | **PASS** | 空参/help/version/schema/ops list 均 exit 0；version=0.3.0；无凭据零请求；ops list 无映射返回空对象 |
| CLI-002 | **PASS** | 根列组+authentication；组级列 commands；leaf 含 api_contract/effects；`--all` 恰 19 条命令、cli_path 无重名 |
| CLI-003 | **PASS** | 19 条命令逐一对比 `--help` 与 leaf schema：parameters 集合与 risk 全部一致 |
| CLI-004 | **PASS** | `records nonexistent`/`unknown`/`schema nonexistent` 均 exit 2、UNKNOWN_COMMAND、含 hint+actions、零联网 |
| CLI-005 | **PASS** | config/format/timeout 放命令前/中/后三位置 data 语义完全一致；`-f json` 短格式可用 |
| CLI-006 | **PASS** | json/pretty/--pretty 均可解析单个 JSON；csv/jq exit 2、USAGE、不静默忽略 |
| CLI-007 | **PASS** | `--fields parameters,effect` 后 data 仅剩该两键；envelope 顶层 ok/data/meta 保留；meta.contract_version=1 |
| CLI-008 | **部分执行** | 成功/输入错/无凭据路径已覆盖（见 CLI-009 及 INP/CFG）；HTTP 500、平台业务失败、输出错误属 M/F 环境变体 → 见 NET 模块，本模块对应变体 NOT_RUN |
| CLI-009 | **部分执行** | 实测：成功 0、缺 id 2、无凭据 3 ✅；「下载到不可写目录」用不存在盘符与只读目录实测 exit 4/PROTOCOL——因为无凭据先于本地输出检查触发，与文档预期"退出 6"不一致，**记观察项 OBS-001**（真实不可写输出场景需 M 环境带假凭据复测） |
| CLI-010 | **PASS** | schema --all 含 19 命令、写命令 effect=write、删除/切换带 confirmation；api_contract 标注 live_verified=false、限流责任、60s 上限；无宣称原生审批/扫码可用 |

## 3. INP 模块（10 条）

| ID | 状态 | 实测要点 |
|---|---|---|
| INP-001 | **PASS** | 内联/@文件/stdin 三种 dry-run 语义一致、零联网 |
| INP-002 | **PASS** | UTF-8 BOM 文件正确解码（中文/emoji/引号/换行保留）；无 BOM 正常。附注：损坏 JSON 报 INPUT 属预期（首测的 BOM 文件因生成脚本换行损坏被正确拒绝，重测规范文件通过） |
| INP-003 | **PASS** | null/数组/字符串/损坏 JSON/双 JSON 均 exit 2 且零请求；批量接受数组、拒绝空数组和对象 |
| INP-004 | **PASS** | 缺 schema/缺 id/未知 flag/多余位置参数/缺值选项均 exit 2 |
| INP-005 | **部分执行** | 9007199254740993 与 12.34 在 dry-run 回执中完整保留；服务端接收侧需 M/R 环境核实 → 部分变体 NOT_RUN |
| INP-006 | **PASS** | 精确 16777215/16777216 字节均接受，16777217 拒绝 exit 2 INPUT；不崩溃 |
| INP-007 | **PASS** | @不存在文件/@目录 exit 2 结构化错误；中文+空格合法路径成功 |
| INP-008 | **PASS** | --submit/--submit=false 合法、=bad 拒绝；limit 1/500 接受、0/501 拒绝；offset 0 接受、-1/abc 拒绝 |
| INP-009 | **PASS** | custom data 覆盖 ActionName/Controller/AppCode 拒绝 exit 2；api body 注入 EngineSecret 拒绝且不回显；--engine-secret 未知选项 exit 2 |
| INP-010 | **PASS** | --filter 叠加 offset/limit/count 任一即拒绝 exit 2；filter 独立可用 |

## 4. CFG 模块

本模块 P0 用例（CFG-001~010）在本次执行会话中未逐条实跑（时间约束），保持 **NOT_RUN**。
其关联自动测试 UT-004/005/007/012/018/021/024/025 均已在基线中 PASS，但这不等于逐条验收。

## 5. PRO 模块（12 条）

| ID | 状态 | 实测要点 |
|---|---|---|
| PRO-001 | **PASS** | dry-run would_change=true；原文件 sha1 不变；目录零新增；零网络 |
| PRO-002 | **PASS** | current=sandbox/previous=prod/changed=true；备份文件 `.h3yun-config-backup-*` 与切换前文件**逐字节一致**；sandbox read_only 等语义保留 |
| PRO-003 | **PASS** | `--name -` 在 prod↔sandbox 间来回切换，previous_profile 每次正确更新 |
| PRO-004 | **PASS** | 同名再切 changed=false、文件未重写、无新增备份 |
| PRO-005 | **PASS** | 无 --config 且清空 H3YUN_CONFIG → exit 3，不猜测当前目录配置 |
| PRO-006 | **PASS** | --name unknown → exit 3 PROFILE_NOT_FOUND；原配置不变 |
| PRO-007 | **PASS** | --profile 与 H3YUN_PROFILE 叠加 switch 均拒绝 exit 3 CONFIG；配置未被改 |
| PRO-008 | **部分执行** | 预置 `.lock` 目录 → CONFIG_LOCKED exit 3；移除后切换成功。两进程真实竞争变体 NOT_RUN |
| PRO-009 | **BLOCKED** | 备份/替换失败需故障注入夹具（磁盘满/权限错误），本机未建夹具 |
| PRO-010 | **BLOCKED** | 目录/符号链接目标与父目录别名竞争需 OS 级夹具，Windows本次未建 |
| PRO-011 | **BLOCKED** | 外部修改检测需同步测试夹具注入时序，未建 |
| PRO-012 | **BLOCKED** | 中断恢复需隔离子进程强杀夹具，未建 |

## 6. 未执行模块

REC / FIL / EXT / NET / OAU / SEC / PLT / REL / E2E / NFR：**全部 NOT_RUN**。
- REC/FIL/EXT/NET/OAU/SEC 的 M 类变体需要 httptest 夹具驱动的端到端脚本（关联 UT 已在基线 PASS）。
- OS 类需 Mac/Linux 真机；R 类需已授权测试租户；OAU-014 真实扫码按文档约定应标 BLOCKED（无已核实端点）。

## 7. 观察项与建议

| 编号 | 内容 | 建议 |
|---|---|---|
| OBS-001 | CLI-009f「下载到不可写目录预期 exit 6」实测先触发无凭据(exit 3)或协议错(exit 4)：本地输出检查在鉴权之后。文档预期与实际顺序可能不符，或需 M 环境带假凭据+HTTP 200 响应才能构造 exit 6 | 在 M 环境补测；若确认顺序即如此，修订用例预期描述 |
| OBS-002 | INP-002 首测 BOM 文件被拒是测试数据生成问题（heredoc 把 `\n` 变成真实换行导致 JSON 损坏），CLI 行为正确 | 无需改代码；记录避免复测误判 |

## 8. 汇总

| 状态 | 数量 | 明细 |
|---|---|---|
| PASS | **24** | CLI 8 / INP 9 / PRO 7；只统计完整通过的验收用例 |
| 部分执行 | **4** | CLI-008、CLI-009、INP-005、PRO-008（已通过部分变体，仍有 M/F/R 变体待补） |
| BLOCKED | 4 | PRO-009~012（缺故障注入夹具） |
| NOT_RUN | **108** | 140 条总数减去 24 PASS、4 部分执行和 4 BLOCKED；包含 CFG 及后续未执行模块 |

> 自动回归基线是 4 项独立检查，不计入 140 条验收用例的状态统计。本报告遵循 TEST-CASES 第 8 节约定：未填入任何真实凭据；证据为本文件记录的退出码与回执摘要；未把关联 UT 的通过改记为用例 PASS。
