# 公有云接口核对

核对日期：2026-10-20。使用者提供的 [氚专 Apifox 文档](https://h3yun-pro-doc-openapi.apifox.cn/) 是本次对照来源，不等同于氚云官方支持承诺；存在差异时同时参考氚云帮助中心。读取各接口 `.md` 版参数定义，而不只看示例。以下是文档核对结果，未进行真实账号调用。

## 已封装的接口

| CLI | API / 文档 |
|---|---|
| records get | [LoadBizObject](https://h3yun-pro-doc-openapi.apifox.cn/188676261e0.md) |
| records list | [LoadBizObjects](https://h3yun-pro-doc-openapi.apifox.cn/188676262e0.md) |
| records create | [CreateBizObject](https://h3yun-pro-doc-openapi.apifox.cn/188676263e0.md) |
| records create-many | [CreateBizObjects](https://h3yun-pro-doc-openapi.apifox.cn/188676264e0.md) |
| records update | [UpdateBizObject](https://h3yun-pro-doc-openapi.apifox.cn/188676265e0.md) |
| records delete | [RemoveBizObject](https://h3yun-pro-doc-openapi.apifox.cn/188676266e0.md) |
| files upload | [UploadAttachment](https://h3yun-pro-doc-openapi.apifox.cn/188676270e0.md) |
| files download | [DownloadBizObjectFile](https://h3yun-pro-doc-openapi.apifox.cn/188676267e0.md) |
| custom invoke | [自定义接口](https://h3yun-pro-doc-openapi.apifox.cn/188676268e0.md) |

前六项以及自定义接口使用 `/OpenApi/Invoke`。附件使用各自路径，不是 Invoke 的 ActionName。Filter/BizObject 采用 JSON 字符串，BizObjectArray 为字符串数组；CLI 的对象输入会完成序列化。

## v0.2.1 行为变化

- CLI 配置及 Go 客户端同时拒绝大于 60 秒的 HTTP 超时；默认仍为 45 秒。浏览器授权等待时间独立，不受此限制。
- 精确 leaf Schema 包含来源、核对日期、鉴权、频率说明及尚未真实联调的标志。写入与预览回执通过 `meta.warnings` 提醒业务副作用，不向服务器包体增加字段。
- 单条提交在流程表单中会推进到下一审批节点。批量提交直接生效，不负责创建审批中的流程；需要自定义接口实现该类批量流程行为。
- 批量 500 条是建议值，不被伪装为服务端硬上限。CLI 不自动拆批，避免重试或拆批造成难以核对的部分成功。
- 更新生效数据会触发作废及重新生效规则；删除生效数据会触发作废规则。普通 CRUD 不能等同于审批处理。

## v0.4.0 组合能力

- `records list --all --max N` 是 CLI 对 `LoadBizObjects` 的显式有界循环，不是新的服务端接口。N 必填且为 1..100000，单页 1..500，累计序列化记录不超过 64 MiB，请求起点至少间隔 700ms。任何一页失败即停且不重试，已完成页、失败响应（如有）和 next_offset 会保留。到达 max 仅表示本地边界已到；连续页不是一致性快照。
- `records update-many` 是 CLI 串行调用 `UpdateBizObject`，每次 1..500 个唯一 `{id,data}`，至少间隔 700ms，失败即停。它不是原子服务端批量接口；回执将记录分为 succeeded、failed、unknown、unprocessed，unknown 必须先核实再决定是否继续。
- `--min-interval` / `H3YUN_MIN_INTERVAL` / `min_interval` 仅调度当前客户端的请求起点，范围 0s..60s；上述多请求命令最低 700ms。它不是租户级或跨进程限流器。
- `--fields` 只对 CLI JSON envelope 的 `data` 做点路径投影，保留嵌套结构并逐项处理数组；不改变 OpenAPI 包体、服务端字段选择、权限或传输量。
- `doctor --remote --schema CODE` 调用一次 `LoadBizObjects`，最多请求一条并丢弃业务行。成功只证明指定表单在该次调用中可读；平台拒绝无法单独区分密钥、表单编码与权限问题，不能推断全局凭据有效或写入能力。
- `schema fields` 只路由到 operations 文件中已验证、`read_only=true` 的自定义控制器映射。目录中没有已核实的原生字段元数据 OpenAPI，CLI 不规范化控制器的 ReturnData。参见 [METADATA.md](METADATA.md)。

## 调用方责任与待核对差异

1. 频率：文档给出每秒 1–2 次。CLI 可限制一个客户端内的请求起点，但 Agent 宿主仍须统一调度多个进程；没有跨进程限流，也没有自动重试。
2. 子表附件：记录先保存；主表 schema、子表 child-schema、子表行 id、无子表前缀的 field 分别传入。
3. 上传 multipart 字段：Apifox 为 `file`，[官方示例](https://help.h3yun.com/contents/1012/1638.html) 为 `media`，本版保留后者和 10 MiB 限制。未证明两种格式都兼容；不通过重发上传试探。
4. 下载编码：Apifox 为 multipart/form-data，[官方示例](https://help.h3yun.com/api/preview/1/1013/1639) 使用 FormUrlEncodedContent。本版保留 urlencoded，并要求 Content-Disposition。不能在表单提交事务尚未结束时假定刚上传附件已可下载。
5. 多选人员/部门/关联字段：入门页文字描述与数组示例不一致。本版保持用户输入，不猜测转换规则；需通过真实表单契约确认。人/部门查询补充对象不应写回，CLI 不会仅凭 `Object` 后缀自动删字段。
6. 部分变更接口的 Apifox 响应 schema 是空对象，不代表业务成功；本版仍依据平台 `Successful` 判断成功，保留失败时的 ReturnData。

## 能力边界

目录没有提供可直接接入的公共扫码 OAuth 端点；EngineCode/EngineSecret 仍由使用者提供。现有可选 PKCE 模块不是氚云扫码服务，需要独立的真实授权服务和 Bearer 网关。未在本次核对中注册应用或部署网关。

审批处理、组织管理、表单设计及字段元数据等不能仅凭上述目录声明已实现，需要实际可用接口或已部署自定义控制器。`ops` 是调用映射，不是服务端功能本身。CLI、源码、示例配置和发行包均不内置 EngineCode/EngineSecret。所有新增验证均为离线或本地模拟服务测试。
