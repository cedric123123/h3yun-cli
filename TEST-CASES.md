# 氚云 Agent CLI v0.3.0 完整测试用例

文档日期：2026-10-16。适用对象：当前仓库 v0.3.0 源码及对应发行程序。共 **140 条验收用例、14 个模块**，关联现有 **43 个 Go 测试函数**（其中包含表驱动子用例）。

本文件是可执行的测试设计与验收清单，不是“140 条全部通过”的报告。除第 2 节已实跑的自动回归外，以下验收用例初始状态统一为 **NOT_RUN**。自动测试关联表示有相关检查，**不等于整条用例及全部平台、边界变体已被覆盖**。测试中发现与预期不符，应记录缺陷，不修改预期来迁就现状。

## 1. 范围与明确边界

覆盖命令发现、JSON 输入、配置、环境切换、记录增删改查、附件、原始 API、自定义操作、HTTP/TLS、OAuth、会话安全、跨平台、发布及 Agent 工作流。

当前注册表有 19 个命令，另有根级 help/schema/version。8 个标准公共 API 已实现客户端；审批、组织、表单管理通过用户部署的控制器与映射扩展，并非已内置全部远程能力。

浏览器 OAuth + PKCE 客户端可用本地提供方验证，但尚无已核实的氚云公有云扫码 OpenAPI 授权端点。真实扫码需要提供方、应用注册和 Bearer 网关；条件不具备时应标 BLOCKED，不能用模拟成功代替真实扫码验收。

本用例依据当前源码及仓库 [README](README.md)、[API-CONTRACT](API-CONTRACT.md)、[AUTH](AUTH.md)、[MACOS](MACOS.md)、[AGENT](AGENT.md) 制定。真实平台行为以已获授权的沙箱实测为准；本次编写不新增对远程文档时效性的保证。

## 2. 本次已执行的基线

2026-10-16 在 Windows amd64、Go 1.27.0 环境重新运行：

| 检查 | 本次结果 | 解释 |
|---|---|---|
| go test ./... -count=1 -cover | PASS | 43 个测试函数及其子用例通过；主要为本地/模拟服务验证 |
| go vet ./... | PASS | 静态检查通过 |
| go mod verify | PASS | 依赖文件校验通过 |
| internal/cli 语句覆盖率 | 86.0% | Windows 本次测量 |
| pkg/h3yun 语句覆盖率 | 83.6% | Windows 本次测量 |
| internal/auth 语句覆盖率 | 74.2% | 不是 Mac 真机钥匙串覆盖率 |
| internal/localstate 语句覆盖率 | 83.3% | Windows 本次测量 |
| cmd/h3yun、scripts/package | 0.0% | 不隐去未覆盖入口 |
| 140 条验收用例逐条执行 | NOT_RUN | 本文件提供待执行清单，不据关联关系自动改为 PASS |
| 真实氚云读写/审批、Mac/Linux 真机 | NOT_RUN | 当前未完成这些环境的实测 |

历史交叉编译及发行物检查见 [VALIDATION.md](VALIDATION.md)，其日期与本次回归日期不同。交叉编译不能替代运行验收。

## 3. 执行约定

### 3.1 优先级、环境和状态

| 标记 | 含义 |
|---|---|
| P0 | 核心功能、安全、数据一致性；相关发布范围内必须通过 |
| P1 | 重要边界、平台适配、恢复能力 |
| P2 | 体验与优化基线 |
| L | 离线本地；不调用真实业务服务 |
| M | Go httptest 或等效受控模拟服务，使用假凭据 |
| F | 故障注入夹具：可控时序、文件错误、连接中断等；部分夹具需补建 |
| OS | 指定系统上的真实发行程序与系统集成测试 |
| R | 经批准的真实氚云测试租户/表单/身份；不是生产环境 |
| PASS / FAIL | 实际执行并满足全部断言 / 存在不符合项 |
| NOT_RUN / BLOCKED | 尚未执行 / 缺少必要环境、契约、账号或夹具 |
| N/A | 经评审确认不属于本次交付范围；必须写原因 |

环境用“/”连接表示需要分阶段或分平台执行，不是任选一种就算全通过。每条列出的输入变体均须执行；记录可细分为 `INP-008/limit-501/windows`。一个变体失败，整条不能标 PASS。

自动关联列的 UT 编号见第 7 节。它们仅指相关既有测试；“—”表示没有指定现成自动关联，不表示该用例不重要。本 CLI 没有 `--mock`、`--device`、`--jq` 或通用自动翻页参数；模拟与注入是在测试夹具中进行，不能虚构 CLI 开关。

### 3.2 安全前置条件

1. 每轮使用专属临时目录、配置副本及 `H3YUN_AUTH_DIR`。保存执行前文件清单/hash；不使用真实默认会话目录做破坏测试。
2. 测试进程隔离 H3YUN_*、profile 专用凭据变量及代理变量。不要修改用户全局环境或系统代理。
3. M 环境仅绑定回环地址，只用固定假凭据；HTTP 仅显式用于测试。正式连接不关闭 TLS 校验。
4. R 环境须有测试租户、可清理表单、隔离审批人和最小权限凭据。密钥由使用者在进程环境中注入，不写进文档、命令行、仓库或测试回执。
5. 涉及提交、审批、删除、组织维护时，先核对自动化、通知及业务规则。没有对应业务授权则 BLOCKED。
6. 网络中断后的写操作先查证实际状态；不得自动重复执行。批量部分成功逐条核对，不能整体重放。
7. F 使用隔离目录、子进程或专属测试卷，不填满系统磁盘、不损坏真实缓存、不批量清除用户钥匙串。
8. `127.0.0.1:7897` 可作为可选测试进程代理；先确认代理可用。回环加入 NO_PROXY；不要把真实凭据发给不可信代理。
9. 强制终止进程场景不要求它还能输出 JSON；要验收的是重启后的状态完整性和可恢复性。

### 3.3 通用断言和清理

每个正常结束的命令均记录退出码、stdout、stderr、持续时间、请求次数及本地/远程副作用。默认 stdout 应能解析为且仅为一个 JSON 对象；成功退出 0，失败退出非 0；保留 `ok/error/meta` 契约，不强制要求所有可选字段出现。

`auth status` 和 `doctor` 是本地检查，不证明远程认证有效。网络错误可重试提示不等于自动重放授权；尤其要检查 `outcome_unknown`。

业务 dry-run 应零网络、零远程变更、零下载文件；profile switch dry-run 不创建备份/锁文件；OAuth dry-run 不创建监听器或会话。读取本地输入不等于有副作用。

清理只针对本轮清单中的记录 ID、附件及精确文件路径。删除记录必须确认属于本轮；审批无法撤销时按测试平台流程处置，不假设有回滚 API。备份恢复前确认没有后续合法修改；锁目录删除前确认持有者已退出；禁止通配清空用户配置或会话目录。

## 4. 测试数据与操作模板

### 4.1 统一符号

| 符号 | 准备要求 |
|---|---|
| BIN | Windows 为项目 dist/h3yun.exe；Mac/Linux 为对应发行程序绝对路径 |
| RUN_ID | 每轮唯一标识，例如 QA-20261016-001；用于业务字段、目录和证据关联 |
| S / ID | 可清理的测试表单编码 / 实际创建返回的记录 ID；仅 M/dry-run 可用假值 |
| CHILD / CHILD_ROW_ID | 真实子表编码 / 已保存的子表行 ID，不是主记录 ID |
| ATT_FIELD / ATT_ID | 附件控件编码 / 实际返回的附件 ID |
| F-REC / F-PATCH | 单条创建 / 更新 JSON；替换为测试表单真实可写字段 |
| F-BATCH | 至少两个对象的 JSON 数组；失败变体含一个违反沙箱规则的对象 |
| F-FILTER | 完整 Filter 对象；运算符及字段以测试表单支持契约为准 |
| F-PRO | 下面多环境配置的本轮副本；prod只是测试名称，并非真实生产 |
| F-OPS | 下方仅供模拟的操作映射；R须替换成已核实部署值 |
| F-OAUTH | 下方本地 OAuth 夹具配置；不是可直接扫码的公共端点 |
| F-FILE | 中文空格文件名、真实小PNG、文本；另准备0/10MiB−1/10MiB/10MiB+1字节样本 |
| F-OUT | 专属临时目录中的新文件名，正常下载前必须不存在 |

大小均为二进制单位：1 MiB = 1,048,576 字节，1 GiB = 1,073,741,824 字节。JSON 边界按整个输入总字节计数，包括 BOM，必须用有效 JSON 字符串填充到目标尺寸，而不是填入无效随机字节。

F-REC 示例（字段是假名，R必须换成真实字段）：

```json
{"F_TEXT":"QA-20261016-001","F_NUMBER":12.34}
```

F-PATCH 示例：`{"F_TEXT":"QA-20261016-001-updated"}`。F-BATCH 为两个不同 RUN_ID 后缀的 F-REC 对象。精度用例另加 `9007199254740993`。

F-FILTER 示例：

```json
{"FromRowNum":0,"ToRowNum":2,"RequireCount":true,"Matcher":{"Type":"And","Matchers":[]},"SortByCollection":[],"ReturnItems":[]}
```

F-RAW 示例（只用于 M/dry-run，R必须替换真实编码和 ID）：

```json
{"ActionName":"LoadBizObject","SchemaCode":"TEST_SCHEMA","BizObjectId":"TEST_ID"}
```

### 4.2 多环境配置 F-PRO

```json
{
  "current_profile": "prod",
  "profiles": {
    "prod": {
      "base_url": "https://prod.example.test",
      "engine_code_env": "QA_PROD_CODE",
      "engine_secret_env": "QA_PROD_SECRET"
    },
    "sandbox": {
      "base_url": "https://sandbox.example.test",
      "engine_code_env": "QA_SANDBOX_CODE",
      "engine_secret_env": "QA_SANDBOX_SECRET",
      "read_only": true
    }
  }
}
```

上述域名不是真实服务。M 测试时改为夹具回环 URL，并仅在本轮副本中设置 allow_http；R 改为已授权服务地址。sandbox 只读用于策略验证，需要执行写入的独立用例使用单独、明确允许写的测试配置，不关闭真实环境保护。不同 profile 和通用变量注入不同假值，才能检测串用。

### 4.3 操作映射 F-OPS

```json
{
  "test.read": {
    "action": "TestRead",
    "controller": "MockController",
    "app_code": "MOCK",
    "description": "local fixture only",
    "read_only": true,
    "required": ["id"]
  },
  "test.write": {
    "action": "TestWrite",
    "controller": "MockController",
    "app_code": "MOCK",
    "description": "local fixture only",
    "read_only": false,
    "required": ["id"],
    "success_field": "result",
    "success_value": "success"
  }
}
```

通过测试配置 operations_file 或该测试进程 H3YUN_OPERATIONS 指向文件。R 环境的审批/组织/表单每个动作需单独补充：控制器版本、动作契约、参数、所需角色、成功条件、状态前后对照及恢复办法。不得直接执行示例占位符。

### 4.4 OAuth 配置 F-OAUTH

```json
{
  "base_url": "http://127.0.0.1:18080",
  "allow_http": true,
  "oauth": {
    "authorize_url": "http://127.0.0.1:18081/authorize",
    "token_url": "http://127.0.0.1:18081/token",
    "client_id": "qa-local-client",
    "scopes": ["qa.read"],
    "callback_port": 0
  }
}
```

这些端口是示意值，不表示已有服务。夹具须动态选未占用端口并改写配置；callback_port=0允许系统分配回调端口。提供方需实现授权/换码并验 PKCE，网关需验证假 Bearer。真实提供方使用 HTTPS、实际注册信息及允许的回调，不复制本地 allow_http 设置。

### 4.5 命令模板

下列为参数模板，尖括号需替换；不要直接把模板粘贴进 shell。F-REC 等表示路径，`@` 要与路径组成一个参数；Windows/Unix 按各自 shell 规则引用路径，优先文件输入。

```text
<BIN> schema "records create" --compact
<BIN> records get --schema <S> --id <ID>
<BIN> records list --schema <S> --offset 0 --limit 100 --count
<BIN> records create --schema <S> --data @<F-REC> --dry-run
<BIN> records create-many --schema <S> --data @<F-BATCH> --dry-run
<BIN> records update --schema <S> --id <ID> --data @<F-PATCH> --dry-run
<BIN> records delete --schema <S> --id <ID> --dry-run
<BIN> files upload --schema <S> --id <ID> --field <ATT_FIELD> --file <F-FILE> --dry-run
<BIN> files download --attachment-id <ATT_ID> --output <F-OUT> --dry-run
<BIN> profile switch --config <F-PRO> --name sandbox --dry-run
<BIN> auth login --config <F-OAUTH> --no-browser
```

仅在用例批准的 M/R 范围内移除业务 dry-run；真实删除额外传 --yes。表中的 create/get 等短名均指以上对应业务命令，不是新命令。

## 5. 模块与执行顺序

建议顺序：CLI/INP/CFG → PRO → M 环境 REC/FIL/EXT/NET → M 环境 OAU/SEC → PLT 真机 → R 环境 E2E → REL 发布检查；NFR 在隔离环境单独执行。所有模块均可按具体前置条件独立重跑。

| 模块 | 范围 | 用例数 |
|---|---|---:|
| CLI | 命令发现与 Agent 输出 | 10 |
| INP | 输入解析与边界 | 10 |
| CFG | 配置、鉴权来源与诊断 | 10 |
| PRO | 环境切换与恢复 | 12 |
| REC | 表单记录标准API | 14 |
| FIL | 附件与本地文件 | 14 |
| EXT | 原始API与可扩展能力 | 8 |
| NET | 网络、协议和失败恢复 | 12 |
| OAU | 浏览器OAuth与扫码边界 | 14 |
| SEC | 会话存储与安全 | 12 |
| PLT | 跨平台和安装体验 | 8 |
| REL | 构建、依赖与交付 | 6 |
| E2E | 真实沙箱端到端验收 | 6 |
| NFR | 性能、稳定性与可恢复性 | 4 |

## 6. 详细测试用例

全表初始执行状态为 NOT_RUN；关联 UT 的本次运行结果见第 2 节，不能替代逐条验收记录。

### CLI：命令发现与 Agent 输出

| ID / 名称 | 优先级 | 环境 | 前置差异与操作步骤 | 预期结果 | 自动关联（部分） |
|---|---|---|---|---|---|
| CLI-001 无凭据本地发现 | P0 | L | 清空测试进程凭据；分别执行空参数、help、version、schema、ops list（不指定映射）。 | 退出0、ok=true；version为0.3.0；不要求密钥、不发请求。 | UT-001 |
| CLI-002 渐进 Schema | P0 | L | 依次执行 schema、schema records --compact、schema "records create" --compact、schema --all。 | 根只列组；组列摘要；leaf列参数；全量恰有19条已注册命令且无重名。 | UT-016 |
| CLI-003 命令帮助同源 | P1 | L | 遍历19命令的 --help，与对应 leaf schema 对比。 | 参数、必填、类型、默认值及风险一致；删除需确认、raw条件确认、profile switch本地副作用明确。 | UT-016 |
| CLI-004 未知命令纠错 | P1 | L | 执行 records nonexistent、unknown；执行 schema nonexistent。 | 退出2、UNKNOWN_COMMAND；hint/actions引导发现已注册命令，不联网。 | UT-016 |
| CLI-005 全局参数位置 | P0 | L | 将 config/profile/format/fields/timeout/pretty逐一放在命令前、中、后；比较同一dry-run结果，另测 -f json。 | 合法位置输出语义一致；参数值不会被当成另一全局选项。 | UT-017 |
| CLI-006 格式支持边界 | P1 | L | 同一schema用 --format json、--format pretty、--pretty；另用 --format csv、--jq。 | 支持格式均可解析单个JSON；不支持格式/参数退出2，不能静默忽略。 | UT-017、UT-003 |
| CLI-007 字段投影 | P1 | L | schema "records create" --fields parameters,effect；schema --all --fields parameters；再对失败命令投影。 | 对象data只保留指定顶层字段；数组不裁剪；ok/error/meta不被删除。 | UT-017 |
| CLI-008 统一回执 | P0 | M/F | 分别触发成功、输入错、策略错、网络错、平台错、文件输出错；逐项保存stdout/stderr/退出码。 | 正常结束stdout仅一个JSON且meta.contract_version=1；ok与退出码一致；失败含code/message/retryable；不要求所有可选字段都出现。 | UT-001、UT-003、UT-005、UT-029、UT-009、UT-032 |
| CLI-009 退出码分类 | P0 | M/F | 对应构造：成功；缺id；无凭据；HTTP500；Successful=false；下载到不可写目录。 | 分别退出0/2/3/4/5/6；Agent不把HTTP200直接当业务成功。 | UT-003、UT-005、UT-029、UT-026、UT-032 |
| CLI-010 接口能力与风险说明 | P1 | L | 检查schema --all和各写入dry-run回执，对照API-CONTRACT.md。 | 8个标准API映射明确；api_contract及warnings表达提交差异、副作用、不重试；没有宣称原生审批/组织/扫码已经可用。 | UT-013、UT-014、UT-020 |

### INP：输入解析与边界

| ID / 名称 | 优先级 | 环境 | 前置差异与操作步骤 | 预期结果 | 自动关联（部分） |
|---|---|---|---|---|---|
| INP-001 三种JSON输入 | P0 | L | 同一F-REC分别以内联、@文件和 --data - stdin传给create --dry-run。 | 三种body语义一致；均不联网；特殊字符不丢失。 | UT-002、UT-007 |
| INP-002 BOM与Unicode | P1 | L | 准备UTF-8 BOM、无BOM两份F-REC，字段含中文、emoji、引号、换行；分别dry-run。 | 内容正确解码、JSON输出合法；不把BOM当业务字段。 | UT-007 |
| INP-003 JSON形状与尾部垃圾 | P0 | L | 对象命令分别输入null、数组、字符串、损坏JSON、连续两个JSON；批量输入对象、空数组、混合数组。 | 无效形状退出2；错误前不发请求、不写业务数据。 | UT-003 |
| INP-004 必填与额外参数 | P0 | L | 按leaf schema逐项移除必填参数；添加 --unknown、尾部位置参数、缺值选项。 | 退出2；没有网络请求；不把缺值静默替换为危险默认值。 | UT-003 |
| INP-005 数值精度 | P0 | M | F-REC含9007199254740993及12.34；分别create/update/create-many，解析服务端收到的嵌套JSON。 | 大整数文本不被float64舍入；小数不被整型截断。 | UT-002 |
| INP-006 JSON输入16MiB | P1 | L | 生成合法JSON使总字节数分别为16MiB−1、16MiB、16MiB+1；从文件和stdin输入dry-run。 | 不超限且形状合法者接受；超限退出2；BOM计入总字节；不崩溃。 | — |
| INP-007 文件来源错误 | P1 | L/F | @不存在文件、不可读文件、目录；可读文件置于带空格中文路径。 | 前三者结构化输入错误；合法中文路径成功；没有额外文件创建。 | — |
| INP-008 布尔与整数 | P1 | L | create分别 --submit、--submit=false、--submit=bad；list用limit=1/500/0/501、offset=0/−1/非数字。 | submit真实映射布尔；非法值退出2；页大小1..500且offset非负。 | UT-003、UT-002 |
| INP-009 路由与凭据注入 | P0 | L | custom data尝试覆盖ActionName/Controller/AppCode；api body添加EngineSecret；使用不存在的 --engine-secret。 | 拒绝路由/凭据注入且退出2；密钥只能由支持的鉴权渠道提供。 | UT-006、UT-003 |
| INP-010 Filter互斥 | P0 | L | list --filter F-FILTER分别叠加 --offset 0、--limit 100、--count、--count=false。 | 只要显式传互斥选项即拒绝；独立filter有效，不能悄悄覆盖用户过滤。 | UT-003 |

### CFG：配置、鉴权来源与诊断

| ID / 名称 | 优先级 | 环境 | 前置差异与操作步骤 | 预期结果 | 自动关联（部分） |
|---|---|---|---|---|---|
| CFG-001 默认公有云 | P0 | L | 不指定配置与base_url；records get --schema S --id ID --dry-run；执行doctor。 | 默认https://www.h3yun.com及公开API路径；doctor仅本地检查，不宣称密钥有效。 | UT-021 |
| CFG-002 配置来源优先级 | P1 | L | 准备base_url不同的两份F-PRO；同时指定H3YUN_CONFIG与 --config，分别profile show及dry-run。 | 显式config优先；未给路径不猜测读取当前目录文件。 | UT-018 |
| CFG-003 配置严格校验 | P0 | L | 配置含engine_secret、未知字段、损坏JSON、多个JSON；另测合法BOM配置。 | 非法配置退出2，不接受落盘密钥；合法BOM配置可读。 | UT-007 |
| CFG-004 凭据组合 | P0 | M | 仅EngineCode、仅EngineSecret、均缺失、二者齐备逐项执行get；齐备时使用假凭据及M。 | 不完整时退出3且不联网；齐备头正确；auth status仅credentials_present，remote_verified=false。 | UT-005、UT-004 |
| CFG-005 profile选择顺序 | P0 | L | F-PRO设置current_profile；叠加H3YUN_PROFILE，再叠加 --profile；分别show。 | 命令行 > 环境变量 > 文件默认；缺默认且多profile报PROFILE_REQUIRED，未知名报PROFILE_NOT_FOUND。 | UT-018 |
| CFG-006 跨profile凭据隔离 | P0 | M | 给profile A/B、通用变量设置不同假密钥及不同M地址；依次请求A/B，再移除A凭据。 | 仅使用所选profile的专属变量和端点；A缺凭据不得借用B或通用值。 | UT-018 |
| CFG-007 只读不可弱化 | P0 | M | profile read_only=true且环境H3YUN_READ_ONLY=false；再反向设定；分别get/create/delete/raw。 | 任一只读来源为true就阻止远程写入；get允许；raw即使查询Action也保守视作写。 | UT-018、UT-005 |
| CFG-008 超时三来源 | P0 | L/M | 配置、H3YUN_TIMEOUT、--timeout分别测试1ms/45s/60s/0/−1s/60.001s/文本；检查覆盖顺序。 | 有效范围(0,60s]，默认45s；非法配置退出3；CLI参数最终优先；Go客户端同样守边界。 | UT-012、UT-024 |
| CFG-009 端点和TLS配置 | P0 | L/M | base_url测试带userinfo、query、fragment、不支持scheme及HTTP；HTTP分别显式允许/未允许；测试自定义路径。 | 不安全或无效端点拒绝；默认HTTPS；显式测试HTTP才允许；端点构造不跳到另一主机。 | UT-025 |
| CFG-010 诊断不做远程认证 | P1 | L | 用假密钥执行doctor/auth status/profile show；设置无效CA文件再执行doctor；统计M命中次数。 | 诊断零请求、不泄密；错误CA可识别；不以本地凭据存在证明真实权限。 | UT-021、UT-001 |

### PRO：环境切换与恢复

| ID / 名称 | 优先级 | 环境 | 前置差异与操作步骤 | 预期结果 | 自动关联（部分） |
|---|---|---|---|---|---|
| PRO-001 预览零写入 | P0 | L | 复制F-PRO到专属目录，记录目录清单及文件hash；profile switch --config F-PRO --name sandbox --dry-run。 | 退出0且would_change=true；原文件字节、会话文件、目录清单均不变；零网络请求。 | UT-022 |
| PRO-002 切换与精确备份 | P0 | L | F-PRO原始排版带缩进；切换sandbox；比较回执backup文件和操作前原始字节。 | current_profile=sandbox、previous_profile=prod；备份逐字节一致；其余配置语义和凭据变量名不变。 | UT-022 |
| PRO-003 切回上一个环境 | P0 | L | 完成PRO-002后执行 --name -，再次执行一次。 | 在两个默认环境间切回；previous_profile更新正确；每次实际变更都有独立备份。 | UT-022 |
| PRO-004 同名幂等 | P1 | L | 当前sandbox，再切sandbox；比较文件hash及目录清单。 | changed=false；无重写、无新增备份、会话不变。 | UT-022 |
| PRO-005 缺配置路径 | P0 | L | 清空H3YUN_CONFIG，不传 --config，执行switch。 | 退出3；不自动选当前目录或用户目录配置。 | UT-023 |
| PRO-006 不存在的目标 | P1 | L | --name unknown；移除previous_profile再 --name -。 | 退出3、PROFILE_NOT_FOUND；原配置不变。 | UT-023 |
| PRO-007 临时profile覆盖冲突 | P0 | L | switch同时带 --profile；另一轮设置H3YUN_PROFILE。 | 拒绝持久切换，退出3；不造成命令环境与默认环境混淆。 | UT-023 |
| PRO-008 并发锁 | P0 | F | 测试夹具持有F-PRO.lock，发起切换；再释放并重试；另做两个进程竞争。 | 持锁时CONFIG_LOCKED退出3，原配置不变；释放后可用；并发不写坏JSON。 | UT-023、UT-043 |
| PRO-009 备份/替换失败 | P0 | F | 在隔离目录注入备份创建、写入或原子替换失败；不得填满真实磁盘。 | 退出6；不报告changed成功；原配置保持完整；已完成备份可恢复，失败的临时文件按实现清理。 | UT-043 |
| PRO-010 目标类型与路径别名 | P1 | OS/F | 将配置路径指向目录或符号链接；再用指向同一父目录的别名进行锁竞争。 | 目标须常规文件，目标符号链接拒绝；父目录别名规范化，不绕过合作锁。 | — |
| PRO-011 外部修改检测 | P0 | F | 通过同步测试夹具，在读原配置后、最终替换前修改文件内容或替换其文件身份。 | 返回CONFIG_CHANGED，不覆盖外部修改；原始备份保留；需重新读取后决定下一步。 | — |
| PRO-012 中断恢复 | P1 | F | 隔离子进程在持锁/备份后终止；检查配置及备份；确认写入进程已退出后仅清理该测试锁目录。 | 配置不是半段JSON；备份可查；遗留锁不自动强占；人工核对后可继续切换。 | — |

### REC：表单记录标准API

| ID / 名称 | 优先级 | 环境 | 前置差异与操作步骤 | 预期结果 | 自动关联（部分） |
|---|---|---|---|---|---|
| REC-001 单条查询 | P0 | M/R | get --schema S --id ID；M返回一条记录；R使用测试数据真实ID。 | ActionName=LoadBizObject，SchemaCode/BizObjectId正确；成功返回完整业务回执，零写副作用。 | UT-026 |
| REC-002 分页查询与头 | P0 | M | list --schema S --offset 10 --limit 2；M返回两条。 | Filter是JSON字符串，FromRowNum=10、ToRowNum=12；认证头正确；一次请求；next_offset=12且has_more_possible=true。 | UT-004 |
| REC-003 空页与不足一页 | P1 | M | 分别返回0、1条，对limit=2检查meta；再返回2条。 | 空/不足页无更多的可能性提示；满页仅表示可能有更多，不假称总数已知、不自动翻页。 | UT-004 |
| REC-004 完整Filter | P1 | M/R | list --schema S --filter @F-FILTER；含条件、排序、返回字段；M核对嵌套对象。 | Filter正确序列化，条件原样发送；R返回值满足测试条件，具体运算符由真实表单协议确认。 | UT-002 |
| REC-005 单条默认草稿 | P0 | M/R | create --schema S --data @F-REC；R操作前记录业务规则。 | ActionName=CreateBizObject；BizObject字符串；IsSubmit=false；R核对草稿/发起节点状态，返回ID纳入清理清单。 | UT-002、UT-013 |
| REC-006 单条提交语义 | P0 | M/R | 普通表单及流程表单各执行create --submit；R仅用隔离审批人。 | IsSubmit=true；普通数据生效，流程进入下一审批节点；存在副作用warning，不能只看HTTP成功判通过。 | UT-013、UT-014 |
| REC-007 批量默认草稿 | P0 | M/R | create-many --schema S --data @F-BATCH（2对象），未传submit。 | CreateBizObjects；BizObjectArray为字符串数组；IsSubmit=false；每个返回ID可追溯，无自动重放。 | UT-002 |
| REC-008 批量提交区别 | P0 | M/R | 普通/流程测试表分别create-many --submit并检查实际数据和流程状态。 | 直接创建生效数据；不是逐条发起审批；文档/Schema/warnings一致。 | UT-013、UT-014 |
| REC-009 批量500与501 | P1 | M | 分别用500、501个合法对象；保持输入小于16MiB；统计请求次数。 | 500为建议而非CLI硬上限；501不被悄悄拆批；一次请求、不重试，含批量注意事项。 | UT-014 |
| REC-010 批量部分失败 | P0 | M/R | M返回Successful=false及部分成功ID；R可用一条合法一条违反测试校验的数据。 | 退出5、ok=false，保留ReturnData/部分成功信息；Agent不得把全部记录重放，先核对已创建ID。 | UT-026 |
| REC-011 更新记录 | P0 | M/R | update --schema S --id ID --data @F-PATCH；再get比对。 | UpdateBizObject、BizObjectId及字符串BizObject正确；指定字段变化；不伪称提交草稿或完成审批。 | UT-002、UT-013 |
| REC-012 删除显式确认 | P0 | M/R | 分别delete无 --yes、--dry-run、--yes；前三轮用M，R只删本次创建ID。 | 无确认退出3/CONFIRMATION_REQUIRED且零请求；dry-run只预览；yes仅删除指定记录，保留副作用警告。 | UT-005、UT-011、UT-014 |
| REC-013 业务规则/权限失败 | P0 | M/R | 用不存在ID、无权表单、必填缺失、只读账号；M分别构造业务失败，R限测试权限。 | 服务端业务失败退出5；保留可诊断信息且脱敏；不自动修复字段、不自动提权。 | UT-026、UT-008 |
| REC-014 查询数据安全写回 | P1 | M/R | 查询含人/部门扩展字段的数据；仅选择目标可写字段构造patch；M对原始对象转发另做对比。 | CLI不擅自删改未知字段；Agent不将整份查询结果盲目写回；R核对目标字段及关联关系。 | UT-014 |

### FIL：附件与本地文件

| ID / 名称 | 优先级 | 环境 | 前置差异与操作步骤 | 预期结果 | 自动关联（部分） |
|---|---|---|---|---|---|
| FIL-001 主表上传协议 | P0 | M/R | 保存记录后upload --schema S --id ID --field ATT_FIELD --file F-FILE；M抓multipart。 | 正确附件端点、认证头、media部分、原始文件名和字节；R附件挂到指定控件。 | UT-030 |
| FIL-002 子表上传 | P0 | M/R | upload传主表S、--child-schema CHILD、--id CHILD_ROW_ID、--field ATT_FIELD。 | 主/子表编码及已保存子行ID准确；field不带子表前缀；R附件不得挂错主记录。 | UT-015 |
| FIL-003 中文路径与文件名 | P1 | OS/M | 上传名为“测试 合同（甲）.txt”的文件；测试路径含空格。 | 上传字节和文件名可正确解析；不发生shell拆词、乱码或路径截断。 | UT-030 |
| FIL-004 上传10MiB边界 | P1 | M | 分别上传10MiB−1、10MiB、10MiB+1文件，统计M命中。 | 不超限者可上传；超限在本地拒绝、不发业务请求。 | UT-030 |
| FIL-005 上传文件类型错误 | P1 | L/F | 分别传不存在、目录、不可读文件路径，执行实际上传。 | 结构化错误、非零退出、不崩溃、不把目录内容当附件。 | — |
| FIL-006 空文件与内容差异 | P1 | M/R | 上传0字节、真实小PNG、中文文本；M先验证字节，R以平台接受策略为准。 | CLI不损坏内容；平台拒绝时返回业务失败而非假成功；记录各类型限制，不推测空文件必然可用。 | — |
| FIL-007 下载成功与摘要 | P0 | M/R | download --attachment-id ATT_ID --output F-OUT；M返回disposition及已知二进制内容。 | 输出新文件hash与源一致；回执成功；不得用服务端文件名改写用户指定路径。 | UT-031 |
| FIL-008 下载禁止覆盖 | P0 | M/F | 预先创建F-OUT记录hash，再下载；另在临时写入后/发布前创建同名文件。 | 不覆盖已有文件，非零退出；原hash不变；竞争窗口也不能覆盖。 | UT-031 |
| FIL-009 目录/权限/文件系统 | P1 | OS/F | 输出到不存在父目录、不可写目录、FAT/exFAT隔离卷；对照NTFS/ext4。 | 明确失败且不报保存成功；支持硬链接的目标可用；不悄悄降级为覆盖式写入。 | UT-032 |
| FIL-010 缺Content-Disposition | P0 | M | 返回HTTP200但缺少Content-Disposition，body为HTML或JSON错误页。 | 拒绝作为附件保存；无最终文件，临时文件清理。 | UT-032 |
| FIL-011 截断与连接中断 | P0 | M | 发送比Content-Length短的内容；另一轮传一部分后断开。 | 失败并清理临时文件；F-OUT不存在；不把部分内容作为成功附件。 | UT-035 |
| FIL-012 下载1GiB边界 | P1 | M/F | 用按需生成流测试1GiB及1GiB+1；另测试预告超限长度；预留专属测试磁盘空间。 | 不超限合法响应可保存；超限拒绝；失败不留最终文件，资源使用可监测。 | — |
| FIL-013 附件dry-run | P0 | L/M | upload/download分别加 --dry-run；使用受控路径，记录M命中和目录清单。 | 只返回请求/路径预览；无网络、无下载或临时文件，不要求真实密钥。 | UT-011、UT-015 |
| FIL-014 下载查询参数与凭据 | P0 | M | 假EngineCode含需编码字符；检查download请求URL、头及对用户输出。 | EngineCode按接口要求URL编码；不把EngineSecret写入URL/日志；禁止跨站跳转；真实凭据不进测试证据。 | UT-031、UT-027 |

### EXT：原始API与可扩展能力

| ID / 名称 | 优先级 | 环境 | 前置差异与操作步骤 | 预期结果 | 自动关联（部分） |
|---|---|---|---|---|---|
| EXT-001 raw透传 | P0 | M | api invoke --body @F-RAW；M比较请求JSON。 | 保留ActionName及用户协议字段，不误作records自动包装；回执仍统一；视为潜在写入。 | UT-005 |
| EXT-002 raw删除确认 | P0 | L/M | raw RemoveBizObject分别无yes、dry-run、yes。 | 无yes退出3；dry-run零请求；yes才发送一次；只读策略仍优先。 | UT-005 |
| EXT-003 自定义路由 | P0 | M | custom invoke --app MOCK --controller MockController --action TestRead --data '{"id":"1"}'。 | AppCode/Controller/ActionName正确、data展平；保留标准动作名拒绝混用；未配置控制器不虚构能力。 | UT-006 |
| EXT-004 操作发现与占位符 | P0 | L | ops list无配置、合法F-OPS、operations.example.json依次执行。 | 无配置返回空映射；合法映射可发现；REPLACE_WITH_占位符报CAPABILITY_UNCONFIGURED，不可执行。 | UT-001、UT-010 |
| EXT-005 required和未知操作 | P0 | L/M | ops run --name test.read缺id/有id；再调用不存在name。 | 缺required退出2；未知能力退出3；合法参数映射到已配置动作；验证失败零请求。 | UT-006、UT-005 |
| EXT-006 自定义业务成功条件 | P0 | M | F-OPS配置success_field=result,success_value=success；分别返回成功、error、缺字段、类型不匹配。 | 仅标准Successful与配置业务条件都满足才成功；否则退出5，保留业务返回；只匹配ReturnData直接子字段。 | UT-009 |
| EXT-007 映射只读策略 | P0 | M | 在read_only环境分别执行F-OPS read_only=true/false动作及custom。 | 允许显式只读映射，阻止写映射和通用custom；配置标签不是服务端权限证明，R仍核对最小权限。 | UT-018、UT-005 |
| EXT-008 审批/组织/表单扩展验收 | P0 | R | 先取得已部署控制器契约和测试角色；分别映射审批查询/操作、组织查询/维护、表单查询/维护；逐动作先dry-run再验收。 | 每项按真实服务端前后状态/权限/审计验证；未部署或无契约标BLOCKED，不因CLI能发请求标PASS，也不冒充内置功能。 | — |

### NET：网络、协议和失败恢复

| ID / 名称 | 优先级 | 环境 | 前置差异与操作步骤 | 预期结果 | 自动关联（部分） |
|---|---|---|---|---|---|
| NET-001 标准成功/业务失败 | P0 | M | 分别返回Successful=true和false的标准JSON，失败ReturnData含部分信息。 | 成功退出0；业务失败退出5且ReturnData保留；HTTP状态不能代替Successful判断。 | UT-026 |
| NET-002 非法服务端协议 | P0 | M | 返回HTML、损坏JSON、空体、多个JSON、缺Successful或Successful为字符串。 | 退出4类协议错误、不崩溃；错误正文不泄密；不得把缺成功标志视为成功。 | UT-028 |
| NET-003 服务端JSON体积 | P1 | M | 产生16MiB内合法响应及16MiB+1响应；同时观察内存和请求次数。 | 超限明确协议失败；响应读取有界；不自动再次拉取。 | UT-028 |
| NET-004 HTTP错误与限流 | P0 | M | 对读写各返回401/403/404/429/500/503；429附Retry-After。 | 非零、http_status正确且无自动重试；权限错不当成功；限流不造成重试风暴，按回执与业务状态判断后续。 | UT-029 |
| NET-005 重定向凭据保护 | P0 | M | 返回301/302/307/308到第二个本地服务器；分别测试标准API、上传、下载。 | 不跟随跳转；第二服务零命中、零EngineSecret/Bearer；返回可诊断失败。 | UT-027 |
| NET-006 超时且不重试 | P0 | M | M延迟超过短timeout；读与写各执行一次并记录命中数。 | 超时后退出4；一次请求，不自动重放；进程在合理时间内退出。 | UT-033 |
| NET-007 写入结果未知 | P0 | M/R | M收到写入并记录成功后，断连/返回异常响应；R通过受控网关模拟时必须另行批准。 | 写失败适当标outcome_unknown；Agent先按业务唯一标识/记录ID查证；不能直接重复创建或重复审批。 | UT-029、UT-033 |
| NET-008 DNS/连接失败 | P1 | M/F | 使用专用不可解析域名、已关闭本地端口、主动重置连接的测试服务。 | 结构化网络错误、非零退出；不泄漏凭据；不无限等待、不自动切换其他profile。 | UT-029 |
| NET-009 企业CA与主机验证 | P0 | M | 本地TLS服务用测试CA；无CA/正确CA/错误CA/主机名不匹配依次调用。 | 只有链及主机验证正确才通过；额外CA不关闭TLS验证，不引入跳过校验选项。 | UT-034 |
| NET-010 CA输入异常 | P1 | L/M | 配置CA为不存在文件、目录、非PEM、空文件；分别doctor和业务调用。 | 本地明确失败；不静默退回不安全连接；不发送带密钥的业务请求。 | UT-034 |
| NET-011 代理与回环绕过 | P1 | OS/M | 仅对测试进程设置HTTP_PROXY/HTTPS_PROXY为http://127.0.0.1:7897，NO_PROXY含127.0.0.1,localhost；由受控代理观察目标测试请求；再停代理。 | 正常代理路径可用；停代理明确失败；OAuth回环不经外部代理；不修改系统全局代理，不用真实密钥测试抓包。 | — |
| NET-012 请求取消与释放 | P1 | M/F | 延迟请求时取消调用context；CLI独立进程发送支持的中断；检查连接、临时文件和子进程。 | 函数调用及时终止；正常返回时输出结构化错误；强制杀进程不强求JSON，重启不得误报业务成功。 | UT-033 |

### OAU：浏览器OAuth与扫码边界

| ID / 名称 | 优先级 | 环境 | 前置差异与操作步骤 | 预期结果 | 自动关联（部分） |
|---|---|---|---|---|---|
| OAU-001 默认不伪造扫码 | P0 | L | 无oauth配置执行auth login；同时检查schema根authentication。 | 退出3、AUTH_NOT_CONFIGURED；不打开浏览器、不生成假二维码、不借用DWS token或网页登录Cookie。 | UT-020 |
| OAU-002 登录预览 | P0 | L/M | 合法F-OAUTH配置下auth login --dry-run、auth logout --dry-run；比较缓存目录及M命中。 | 返回PKCE流程、提供方及网关要求；不监听、不换码、不改会话、不联网。 | — |
| OAU-003 授权地址校验 | P0 | L | 分别缺client_id、非法callback_port、端点含userinfo/query/fragment、HTTP非回环；另测显式HTTP回环。 | 非法配置拒绝；仅明确测试允许的回环HTTP可用；正式提供方须HTTPS。 | UT-038 |
| OAU-004 PKCE成功链路 | P0 | M | 本地提供方记录授权URL，回调正确state/code，再返回有效Bearer token。 | state随机；S256 challenge与换码verifier相符；redirect_uri完全一致；token不出现在stdout/URL中；成功保存所选会话。 | UT-036 |
| OAU-005 回调防伪 | P0 | M | 向监听器分别发错误state、重复state、缺code、重复code、非GET、错误Host；最后发正确回调。 | 恶意/畸形回调拒绝，不提前换码或结束为成功；正确回调可完成；只监听127.0.0.1。 | UT-036 |
| OAU-006 拒绝授权 | P0 | M | 正确state回调error=access_denied；预先保留有效旧会话并比对。 | 登录失败退出3，不写新凭据；旧会话不因失败被覆盖；浏览器不提示已成功认证。 | UT-037 |
| OAU-007 等待超时/取消 | P1 | M | auth login --login-timeout 100ms不回调；另测试0、负数、超过30m及非法时长。 | 等待超时退出3；非法时长退出2；回调端口释放，无新增会话；等待默认5m、上限30m。 | UT-037 |
| OAU-008 自动浏览器与无浏览器模式 | P1 | OS/M | 正常启动系统浏览器；随后auth login --no-browser手动用同机浏览器访问stderr URL；模拟浏览器启动失败。 | 自动/手动路径均可完成；stdout只JSON；启动失败引导no-browser，不能把不能回调的另一台机器扫码当已登录。 | UT-036 |
| OAU-009 回调端口占用 | P1 | M | 夹具提前占用指定callback_port，再登录；释放后重试。 | 占用时结构化失败，不监听公网或偷换已配置端口；释放后可正常绑定。 | — |
| OAU-010 换码HTTP失败/跳转 | P0 | M | token端点依次401、500、302到第二服务、超时。 | 不保存token；不跟随重定向；失败正文不回显秘密；第二服务无换码凭据。 | UT-037 |
| OAU-011 token格式与上限 | P0 | M | 返回非JSON、缺token、非Bearer、expires_in=0/负数/超过31536000、token含CRLF、响应超过1MiB。 | 拒绝所有非法响应，退出3且不保存；合法Bearer大小与到期范围内才接受。 | UT-037 |
| OAU-012 Bearer调用与失效 | P0 | M | 成功登录后请求测试网关；再让网关返回401。 | 只使用所选绑定token；Authorization为Bearer；401不伪装成功、不偷偷使用EngineSecret或自动刷新重放。 | UT-019 |
| OAU-013 本地状态与到期余量 | P0 | M | 缓存有效、缺失、损坏、已过期、距过期≤30秒分别执行status和业务get。 | status始终remote_verified=false；不可用缓存authenticated=false；业务需重新登录，退出3；不自动刷新。 | UT-039、UT-019 |
| OAU-014 真实扫码验收门槛 | P0 | R/OS | 只有提供方、客户端注册、回调、Bearer网关齐备时，分别在Mac/Windows浏览器扫码后查一条测试记录。 | 真实用户身份/组织/权限一致，网关可用；没有可核实氚云授权端点时标BLOCKED，不把模拟PKCE通过算公有云扫码可用。 | — |

### SEC：会话存储与安全

| ID / 名称 | 优先级 | 环境 | 前置差异与操作步骤 | 预期结果 | 自动关联（部分） |
|---|---|---|---|---|---|
| SEC-001 环境凭据不落盘 | P0 | OS/M | 注入唯一假EngineSecret，做status/doctor/get/dry-run；扫描本次专属配置、日志、备份、缓存和回执。 | 无EngineSecret明文落盘；不在命令行传真实秘密；扫描证据只保留命中数、不得输出真实值。 | UT-007、UT-008 |
| SEC-002 结构化脱敏 | P0 | M | M在嵌套对象、数组及错误信息字符串中回显假密钥/Bearer；另用单字符假秘密测试。 | 已知秘密的字符串值被[REDACTED]替代；JSON键和语法不损坏；stdout/stderr不泄露token。 | UT-008、UT-019 |
| SEC-003 请求头注入 | P0 | M | 假EngineCode/EngineSecret/Bearer包含CR/LF，调用客户端。 | 拒绝非法头，不产生额外请求头、不联网泄漏其他数据；错误不打印凭据。 | UT-025、UT-037 |
| SEC-004 Windows DPAPI | P0 | OS | Windows专属测试目录登录保存；同用户读取；检查原始文件；另一隔离用户尝试读取副本。 | 同用户可解密；缓存不含明文token；其他用户无法解密；不操作真实用户缓存。 | UT-040、UT-039 |
| SEC-005 Mac Keychain加密 | P0 | OS/M | Mac专属目录保存长假token，多次保存；检查Keychain条目及.session原始内容；读取回验。 | Keychain存AES256密钥，文件AES-GCM密文；随机nonce导致密文不同；长token完整；无明文回退。 | UT-041 |
| SEC-006 Keychain失败安全 | P0 | OS/F | 模拟Keychain拒绝读/写、锁定、不可用；损坏密钥后再保存；对比原文件和条目。 | 错误明确，不回退明文、不静默重置已有损坏密钥；旧会话不被不完整文件替换。 | UT-042 |
| SEC-007 密文篡改/旧明文 | P0 | M/OS | 翻转密文、截断、复制旧版明文缓存；尝试Load。 | 完整性验证失败或拒绝旧明文；不能认证成功；需明确重新登录，不自动迁移不可信内容。 | UT-041 |
| SEC-008 绑定与目录隔离 | P0 | M/OS | 同一token复制到不同profile/base_url/client_id/scopes/audience/提供方；Mac另复制到其他目录。 | 绑定不一致时不能复用；Mac目录绑定防串用；一环境登录/退出不破坏另一环境。 | UT-039、UT-041 |
| SEC-009 文件权限和链接 | P0 | OS/F | Unix会话目录改为0755、文件0644；用会话文件/目录符号链接；再还原合法0700/0600；另用>2MiB缓存。 | 不安全权限、链接或超限缓存拒绝；正确权限可用；Windows按DPAPI语义单独验收，不假称POSIX位等同ACL。 | UT-039 |
| SEC-010 会话并发与替换 | P0 | F | 夹具持有单会话.lock，执行Save/Logout；释放后操作；模拟临时写失败。 | 锁竞争失败而非覆盖竞争者；不产半文件；失败不报告成功；另一个profile不受影响。 | UT-043、UT-039 |
| SEC-011 登出范围 | P0 | M/OS | A/B两有效会话，选A logout两次；检查A文件/Keychain条目、B及环境密钥。 | A本地会话清除且幂等；B不变；environment_credentials_unchanged=true、server_token_revoked=false；不宣称服务端撤销。 | UT-039 |
| SEC-012 Linux存储边界 | P0 | OS | Linux保存假会话并检查权限及StorageBackend；切换到权限宽松目录尝试读取。 | 明确owner_only_file_unencrypted；目录0700/文件0600，宽松权限拒绝；文档不可称Linux具备Keychain/DPAPI加密。 | UT-039 |

### PLT：跨平台和安装体验

| ID / 名称 | 优先级 | 环境 | 前置差异与操作步骤 | 预期结果 | 自动关联（部分） |
|---|---|---|---|---|---|
| PLT-001 Windows发行物 | P0 | OS | Windows amd64从专属目录运行发行exe：version/schema/doctor及一条dry-run；中文空格路径重复。 | 无需Go运行时；版本/JSON/路径正常；默认目录及DPAPI正常，不能以go run替代发行物验收。 | — |
| PLT-002 Mac Apple Silicon | P0 | OS | Apple Silicon解压arm64包，确认执行权限，运行version/schema/dry-run；执行OAU及SEC的Mac场景。 | 原生arm64可运行；Keychain与浏览器正常；交叉编译结果不是本用例PASS证据。 | — |
| PLT-003 Mac Intel | P0 | OS | Intel Mac解压amd64包，同PLT-002执行；记录系统版本。 | 原生amd64可运行；不依赖Rosetta替代原生验收；钥匙串正常。 | — |
| PLT-004 Linux发行物 | P1 | OS | Linux amd64解压/赋执行权限，运行version/schema/dry-run；在ext4做附件和会话测试。 | 无需Go运行时；权限、硬链接下载正常；明确Linux会话非加密。 | — |
| PLT-005 shell与参数引用 | P1 | OS | PowerShell7、zsh、bash分别用@文件和stdin输入含中文/引号JSON；记录实际参数。 | 跨shell解析业务内容一致；推荐@文件减少引号歧义；文档命令与对应shell一致。 | — |
| PLT-006 Mac安全提示与依赖 | P1 | OS | 干净Mac按MACOS.md安装；检查/usr/bin/security及首次打开提示；不绕过系统安全保护。 | 文档明确未签名/未notarize；由用户按系统提示批准；不要求额外语言运行时，不自动移除隔离属性。 | — |
| PLT-007 平台默认缓存位置 | P1 | OS | 测试用户不设H3YUN_AUTH_DIR观察默认位置；另设专属目录重做登录/状态/退出。 | 各平台使用用户配置目录；覆盖路径生效；不串到其他用户/真实缓存；清理仅限测试目录和对应条目。 | UT-039 |
| PLT-008 支持系统最低版本 | P1 | OS | 根据MACOS.md和实际构建工具链支持范围，在声称最低版本与当前常用版本执行烟测；记录Go及OS版本。 | 仅对实际跑过的组合标PASS；不以go.mod最低语言版本推断二进制兼容旧OS。 | — |

### REL：构建、依赖与交付

| ID / 名称 | 优先级 | 环境 | 前置差异与操作步骤 | 预期结果 | 自动关联（部分） |
|---|---|---|---|---|---|
| REL-001 自动回归 | P0 | L | 执行 go test ./... -count=1 -cover，保留完整报告和环境信息。 | 所有43个测试函数通过（含子用例）；不把函数数当验收场景数；覆盖率记录各包。 | UT-001～UT-043 |
| REL-002 静态与依赖校验 | P0 | L | 分别执行go vet ./...、go mod verify并各自检查退出码；检查go.mod/go.sum。 | 均退出0；go-keyring固定v0.2.8；无未经批准的新增依赖。 | — |
| REL-003 四平台构建 | P0 | L | 按scripts/build.ps1或build.sh构建四目标；读取go version -m；额外交叉编译Darwin auth/cli测试。 | Windows/Linux amd64、Darwin amd64/arm64产物齐全且架构正确；CGO=0；交叉测试程序只记编译通过。 | — |
| REL-004 发行包完整性 | P0 | L | 检查zip和两个Mac tar.gz目录、二进制hash、README/AUTH/MACOS/许可和skills；验证tar执行位。 | 与构建产物hash一致；许可证齐全；无真实凭据、缓存、锁、临时测试程序；平台说明相符。 | — |
| REL-005 开源吸纳与授权 | P1 | L | 核对THIRD_PARTY_NOTICES.md、LICENSES及模块许可；确认DWS是设计参考而不是复制未授权源码。 | 来源、版本和许可可追溯；未把DWS token能力或未核实网页API包装成氚云授权；无遗漏必要版权文本。 | — |
| REL-006 文档与Agent契约 | P1 | L | 逐项比对README/API-CONTRACT/AUTH/MACOS/AGENT和3个skills的命令示例与schema；核对发布包版本。 | 示例无未实现选项；风险、只读、退出码、待联调限制一致；本测试文档后续加入包时更新清单和hash，不覆盖旧包冒称未变。 | UT-013、UT-016 |

### E2E：真实沙箱端到端验收

| ID / 名称 | 优先级 | 环境 | 前置差异与操作步骤 | 预期结果 | 自动关联（部分） |
|---|---|---|---|---|---|
| E2E-001 记录全生命周期 | P0 | R | 专属测试表：create草稿→get→list定位→update→get核对→delete --yes→再次查询；全过程记ID。 | 字段/状态正确，删除对象仅本轮数据；最终不存在按平台真实协议判断；全过程无重复创建。 | — |
| E2E-002 附件全生命周期 | P0 | R/OS | 创建主记录与子行→上传主/子附件→获取真实附件ID→下载到新文件→比对hash→按批准方式清理。 | 附件归属正确、字节一致；缺disposition等协议差异如实报缺陷，不修改安全策略强行通过。 | — |
| E2E-003 流程副作用核实 | P0 | R | 隔离流程表逐一验证单条草稿/单条submit/批量submit；检查审批任务、自动化、通知及审计。 | 三种语义符合记录模块预期；不向真实业务人员发通知；所有副作用有证据和清理记录。 | — |
| E2E-004 最小权限与组织隔离 | P0 | R | 测试账号A/B分属不同权限范围；读取/修改各自和对方测试记录；分别选profile。 | 服务端阻止越权；CLI正确传回失败；无跨组织数据或凭据串用。 | — |
| E2E-005 Agent发现到执行 | P0 | M/R | Agent从schema根→组→leaf，构造create dry-run；确认范围后执行、读取结果；故意触发输入错和结果未知。 | 遵守JSON/退出码、必填及风险；不凭猜测发审批命令；不自动重试未知结果写入；删除先确认授权范围。 | — |
| E2E-006 配置切换与登录综合 | P1 | R/OS | 两个测试profile分别检查status→读数据→持久切换→再读→切回→仅登出选中OAuth profile。 | 每次操作与目标环境一致；备份可恢复；会话隔离；若无真实OAuth提供方，该分支BLOCKED，EngineKey分支仍可独立验收。 | — |

### NFR：性能、稳定性与可恢复性

| ID / 名称 | 优先级 | 环境 | 前置差异与操作步骤 | 预期结果 | 自动关联（部分） |
|---|---|---|---|---|---|
| NFR-001 本地命令响应基线 | P2 | OS | 冷启动5次、热启动30次version/schema/leaf；记录机器、P50/P95、输出字节及内存。 | 输出始终合法；建议本机P95<1秒作为待确认体验目标，不冒称既有SLA；明显退化提交性能缺陷。 | — |
| NFR-002 边界资源占用 | P1 | M/F | 运行16MiB JSON、10MiB上传、1GiB流下载，记录峰值内存/句柄/结束时间；在受控环境重复。 | 无崩溃、死锁、无界增长；下载不整块载入内存；性能指标与环境一起记录，不向真实平台压测。 | — |
| NFR-003 调用频率责任 | P1 | M/R | M并发两个调用方记录请求节奏；R仅由宿主限流器按同一集成每秒1–2次协调，逐次执行。 | 明确CLI无跨进程限流、自动拆分或重试；Agent宿主负责限流；不把服务端429当可无限重试。 | — |
| NFR-004 连续回归与故障后恢复 | P1 | M/F | 模拟网络错、协议错、写失败、锁冲突各100轮，再执行正常读写；比较临时文件/句柄基线。 | 无持续增长或状态污染；正常请求可恢复；失败写入先核对业务状态，遗留锁按精确目标人工处理。 | — |

## 7. 现有自动测试索引

以下 43 个函数来自当前源码。关联清单只提供入口，不声称其覆盖对应整条验收用例；平台专属系统行为仍按 OS 场景实测。

| 编号 | 文件 | 测试函数 |
|---|---|---|
| UT-001 | [internal/cli/cli_test.go](internal/cli/cli_test.go) | `TestLocalCommandsWithoutCredentials` |
| UT-002 | [internal/cli/cli_test.go](internal/cli/cli_test.go) | `TestDryRunWireShapes` |
| UT-003 | [internal/cli/cli_test.go](internal/cli/cli_test.go) | `TestInputFailures` |
| UT-004 | [internal/cli/cli_test.go](internal/cli/cli_test.go) | `TestListWirePaginationAndCredentialHeaders` |
| UT-005 | [internal/cli/cli_test.go](internal/cli/cli_test.go) | `TestReadOnlyAndDeleteConfirmation` |
| UT-006 | [internal/cli/cli_test.go](internal/cli/cli_test.go) | `TestCustomAndOperationsMapping` |
| UT-007 | [internal/cli/cli_test.go](internal/cli/cli_test.go) | `TestBOMInputAndConfigRejectsSecrets` |
| UT-008 | [internal/cli/cli_test.go](internal/cli/cli_test.go) | `TestSecretsRedactedFromServerErrors` |
| UT-009 | [internal/cli/cli_test.go](internal/cli/cli_test.go) | `TestCustomBusinessFailure` |
| UT-010 | [internal/cli/cli_test.go](internal/cli/cli_test.go) | `TestExampleOperationsCannotBeExecuted` |
| UT-011 | [internal/cli/cli_test.go](internal/cli/cli_test.go) | `TestDryRunNeverContactsServer` |
| UT-012 | [internal/cli/contract_test.go](internal/cli/contract_test.go) | `TestDocumentedTimeoutBounds` |
| UT-013 | [internal/cli/contract_test.go](internal/cli/contract_test.go) | `TestContractSourcesAndSubmissionSemantics` |
| UT-014 | [internal/cli/contract_test.go](internal/cli/contract_test.go) | `TestDocumentedWarningsKeepRequestUnchanged` |
| UT-015 | [internal/cli/contract_test.go](internal/cli/contract_test.go) | `TestChildAttachmentContract` |
| UT-016 | [internal/cli/dws_test.go](internal/cli/dws_test.go) | `TestProgressiveSchema` |
| UT-017 | [internal/cli/dws_test.go](internal/cli/dws_test.go) | `TestGlobalFlagsAndProjection` |
| UT-018 | [internal/cli/dws_test.go](internal/cli/dws_test.go) | `TestProfilesIsolateEndpointCredentialsAndReadOnly` |
| UT-019 | [internal/cli/dws_test.go](internal/cli/dws_test.go) | `TestOAuthCachedBearerAndRedaction` |
| UT-020 | [internal/cli/dws_test.go](internal/cli/dws_test.go) | `TestMissingOAuthDoesNotPretendToLogin` |
| UT-021 | [internal/cli/dws_test.go](internal/cli/dws_test.go) | `TestPublicCloudDefault` |
| UT-022 | [internal/cli/profile_test.go](internal/cli/profile_test.go) | `TestProfileSwitchDryRunBackupAndPrevious` |
| UT-023 | [internal/cli/profile_test.go](internal/cli/profile_test.go) | `TestProfileSwitchRejectsMissingOverrideAndLock` |
| UT-024 | [pkg/h3yun/client_test.go](pkg/h3yun/client_test.go) | `TestClientTimeoutContract` |
| UT-025 | [pkg/h3yun/client_test.go](pkg/h3yun/client_test.go) | `TestEndpoint` |
| UT-026 | [pkg/h3yun/client_test.go](pkg/h3yun/client_test.go) | `TestInvokeProtocolAndPartialResults` |
| UT-027 | [pkg/h3yun/client_test.go](pkg/h3yun/client_test.go) | `TestRejectRedirectWithoutLeakingCredentials` |
| UT-028 | [pkg/h3yun/client_test.go](pkg/h3yun/client_test.go) | `TestProtocolErrors` |
| UT-029 | [pkg/h3yun/client_test.go](pkg/h3yun/client_test.go) | `TestHTTPFailureAndUnknownWriteOutcome` |
| UT-030 | [pkg/h3yun/client_test.go](pkg/h3yun/client_test.go) | `TestUploadMultipart` |
| UT-031 | [pkg/h3yun/client_test.go](pkg/h3yun/client_test.go) | `TestDownloadAndNoOverwrite` |
| UT-032 | [pkg/h3yun/client_test.go](pkg/h3yun/client_test.go) | `TestFailedDownloadDoesNotCreateFile` |
| UT-033 | [pkg/h3yun/client_test.go](pkg/h3yun/client_test.go) | `TestTimeoutNotRetried` |
| UT-034 | [pkg/h3yun/client_test.go](pkg/h3yun/client_test.go) | `TestEnterpriseCA` |
| UT-035 | [pkg/h3yun/client_test.go](pkg/h3yun/client_test.go) | `TestTruncatedDownloadCleansUp` |
| UT-036 | [internal/auth/oauth_test.go](internal/auth/oauth_test.go) | `TestBrowserPKCEFlow` |
| UT-037 | [internal/auth/oauth_test.go](internal/auth/oauth_test.go) | `TestLoginDenialTimeoutAndTokenFailure` |
| UT-038 | [internal/auth/oauth_test.go](internal/auth/oauth_test.go) | `TestEndpointValidation` |
| UT-039 | [internal/auth/oauth_test.go](internal/auth/oauth_test.go) | `TestSessionStorageIsolationExpiryAndLogout` |
| UT-040 | [internal/auth/oauth_test.go](internal/auth/oauth_test.go) | `TestSealRoundTrip` |
| UT-041 | [internal/auth/keychain_crypto_test.go](internal/auth/keychain_crypto_test.go) | `TestKeychainCipherIsolationTamperingAndLongTokens` |
| UT-042 | [internal/auth/keychain_crypto_test.go](internal/auth/keychain_crypto_test.go) | `TestKeychainFailureDoesNotFallbackOrReplaceKeys` |
| UT-043 | [internal/localstate/files_test.go](internal/localstate/files_test.go) | `TestLockAndReplace` |

### 7.1 自动回归运行方法

在项目根目录，用已安装的 Go 执行。每条分别记录退出码，避免仅检查最后一条命令的退出码而漏掉前面的失败。

```sh
go test ./... -count=1 -cover
go vet ./...
go mod verify
go test ./internal/cli -run '^TestProfileSwitch' -count=1 -v
go test ./internal/auth -run '^TestKeychain' -count=1 -v
```

Windows 若 Go 未加入 PATH，可在 PowerShell 中使用 `& 'C:\Program Files\Go\bin\go.exe'` 替代上述 go。读取每次的 `$LASTEXITCODE`；Unix shell读取 `$?`。不要把 $? 的 PowerShell布尔语义当成进程数值退出码。

Keychain 单测的通过只说明模拟存储/加密逻辑通过，不表示真实 Mac 系统钥匙串已经验收。真实浏览器、真实云账号、进程中断和本地文件系统故障需要对应夹具或人工执行。按需新增自动用例时，应保留本文件的案例 ID 作为追溯标识。

### 7.2 标准API追溯

| 标准API | CLI | 主用例 |
|---|---|---|
| LoadBizObject | records get | REC-001、E2E-001 |
| LoadBizObjects | records list | REC-002～004、E2E-001 |
| CreateBizObject | records create | REC-005～006、E2E-001/003 |
| CreateBizObjects | records create-many | REC-007～010、E2E-003 |
| UpdateBizObject | records update | REC-011、E2E-001 |
| RemoveBizObject | records delete | REC-012、E2E-001 |
| UploadAttachment | files upload | FIL-001～006、E2E-002 |
| DownloadBizObjectFile | files download | FIL-007～012/014、E2E-002 |

自定义控制器业务操作另见 EXT-003～008；CLI只能提供调用与成功条件映射，不能证明服务端已经实现所有审批/组织/表单能力。

## 8. 执行记录模板

每个案例、输入变体、系统架构分别一条记录。以下模板初始为未执行，可复制到测试管理工具；不得填入完整密钥、token、authorization code或回调URL。

| 字段 | 填写内容 |
|---|---|
| 用例 ID / 变体 | 例如 FIL-008 / publish-race / darwin-arm64 |
| RUN_ID | 本轮唯一标识 |
| 优先级 / 初始状态 | P0 / NOT_RUN |
| 程序版本与 SHA-256 | 实际运行二进制，不只记录源码版本 |
| 源码定位 | commit（若有）；未提交时记录工作区快照/hash，不虚构commit |
| 系统/架构/文件系统/shell | 例如具体macOS版本、arm64、APFS、zsh |
| 环境 / 前置条件 | L/M/F/OS/R；测试租户、夹具版本、角色均去敏 |
| 输入 / 操作 | 参数与数据文件标识、变体；凭据仅记录变量名 |
| 预期 | 引用本案例并列出全部变体断言 |
| 实际 | 退出码、回执摘要、请求次数、文件hash、业务前后状态 |
| 证据路径 | 去敏stdout/stderr、截图、服务端测试日志、报告 |
| 最终状态 | PASS / FAIL / BLOCKED / N/A；不得遗漏说明 |
| 缺陷 / 阻塞原因 | 缺陷编号、复现条件或缺失提供方/设备/权限 |
| 清理 | 本轮ID和文件是否清理；不可逆副作用及残留 |
| 执行人 / 时间 | 实际执行人及带时区时间 |

证据建议放独立测试结果目录，不加入发行包；目录名带 RUN_ID。保留真实业务数据时遵守组织数据策略，不把未经脱敏的回执提交 Git。输出“脱敏通过”也要确认没有额外截图、抓包或调试日志泄漏。

## 9. 进入、退出与发布验收标准

### 9.1 进入条件

- 明确源码/发行物版本、架构和待验范围；本地基线通过。
- 准备专属临时目录、测试数据及夹具；OS场景有真机或等效原生运行环境。
- R场景取得测试账号权限和数据/流程副作用授权，具备追踪与清理能力。
- OAuth真实扫码场景提供已核实端点、应用注册和Bearer网关。缺失时仅该分支阻塞，不伪造功能。

### 9.2 核心发布门槛

- 相关范围内P0全部实测通过；不能有未解决的凭据泄漏、越权、误删、静默覆盖、重复写入或数据损坏。
- P1失败逐项评审并披露；需真实平台才能验证的项目不得以“单元测试通过”豁免。
- 43个既有自动函数全部通过，静态检查、依赖校验、架构与许可证检查通过；修改源码后重新跑，不能沿用旧结果。
- 宣称支持某系统时，至少该系统对应原生发行物烟测通过；Mac还需Keychain和浏览器验收。仅交叉编译应标“待真机验证”。
- 真实表单/附件/权限路径完成E2E验收，才可对该租户场景宣称已联调。
- 如仅发布EngineCode/EngineSecret模式，可明确排除未接通的OAuth/扫码功能；但不得宣传已支持氚云直接扫码授权。
- 无法覆盖的审批/组织/表单自定义动作必须列出缺失契约或部署项，不用“支持全部”替代逐项证据。
- 发行物hash、文档、schema、skills与执行记录保持一致；测试文档新增后若重新打包，应产生新的清单/hash并保留旧发行物。

### 9.3 缺陷分级与复测

| 级别 | 示例 | 处理 |
|---|---|---|
| S0 安全/数据阻断 | 密钥泄漏、跨组织读写、重复审批、覆盖用户文件 | 停止相关真实测试；保留去敏证据，修复并做安全回归 |
| S1 核心不可用 | 注册命令无法执行、协议错误识别失败、Mac无法启动 | 阻断相关功能/平台发布 |
| S2 重要缺陷 | 边界值、诊断信息、恢复路径异常 | 评审风险与范围；明确修复或限制 |
| S3 体验问题 | 本地命令慢、提示不清楚 | 记录基线，安排优化 |

修复后至少重跑失败用例全部变体、相邻模块P0、43个既有自动测试及受影响平台烟测。涉及传输、鉴权、存储的改动，不只复测“正常成功”路径。

## 10. 当前待验收重点

1. 真实氚云账号的8个标准API及提交/自动化副作用。
2. Mac arm64、Mac amd64、Linux原生运行；Mac真实Keychain授权拒绝/锁定/登出行为。
3. 提供方和网关齐备后的真实浏览器扫码；目前不可宣称氚云公有云直连扫码已接通。
4. 已部署控制器的审批、组织、表单管理逐动作权限和状态验收。
5. 超限输入、下载发布竞态、外部配置修改、进程中断等尚需扩展夹具的场景。
6. Agent宿主的跨进程限流与未知写入结果核对；CLI本身不负责自动限流或重试。

以上为待验证事项，不代表已观察到缺陷；也不能因缺环境而默认通过。

