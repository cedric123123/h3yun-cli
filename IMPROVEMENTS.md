# v0.4.0 改进评估

本次按“提升 Agent 的可发现性、长任务可恢复性和写入可核对性”评估改进建议。优先实现不需要虚构服务端能力、能够保持 JSON 契约并可用本地模拟服务完整验证的部分。

| 建议 | v0.4.0 结论 | 说明 |
|---|---|---|
| 字段发现 | 已实现受约束入口 | `schema fields` 仅调用真实部署且映射为 `read_only=true` 的自定义控制器。没有把字段发现声明为原生 OpenAPI，详见 METADATA.md。 |
| 自动翻页 | 已实现，必须显式有界 | `records list --all --max N`，N 为 1..100000；单页 1..500、累计 64 MiB、至少 700ms、失败即停、不重试、不是一致性快照。 |
| 请求间隔 | 已实现单客户端调度 | 配置 `min_interval`、环境变量 `H3YUN_MIN_INTERVAL` 或参数 `--min-interval`，范围 0s..60s；all/update-many 最低 700ms。跨进程仍由宿主协调。 |
| 批量更新进度 | 已实现串行组合命令 | `records update-many` 逐条调用单条更新，非原子、失败即停，分别返回 succeeded/failed/unknown/unprocessed ID。 |
| 嵌套字段裁剪 | 已实现点路径 | `--fields` 对 envelope 的 data 做点路径投影并逐项处理数组。平台原始响应仍是默认输出，因此没有另加 raw 模式。 |
| 凭据持久化 | 不为静态密钥新增存储 | EngineCode/EngineSecret 继续由使用者或宿主注入，不进入 CLI、配置或发行包。可选 OAuth 会话沿用 Windows DPAPI、macOS Keychain + AES-GCM 和 Linux 权限文件的既有边界，但公有云扫码端点仍未核实。 |
| 远程诊断 | 已实现最小只读探测 | `doctor --remote --schema CODE` 只读最多一条且不输出记录；只证明指定表单本次可读，不能证明全局、写入或审批权限。 |
| CSV/JSONL 导入导出 | 本版暂缓 | 当前 JSON、stdin 和文件输入已经保持数字精度与平台结构。表单类型、子表和附件的扁平化规则尚无可靠通用契约，贸然转换会产生数据损失。 |
| 查询自动重试 | 本版暂缓 | CLI 继续报告 retryable、结果未知和续跑位置，不自动重试。多页读取在源数据变化时不是快照，自动重试可能放大重复或遗漏；应由 Agent 按任务语义决定。 |

## 关键安全边界

- 所有多请求操作都保留“无自动重试”原则。读任务根据 `meta.pagination` 续跑，写任务先核实 unknown ID。
- 700ms 是这两个组合命令的客户端下限，不等于平台配额保证；多个 Agent、进程或其他集成共享凭据时仍需统一调度到文档建议的每秒 1–2 次。
- `--all` 到达 max 时返回成功但 `complete=false`；只有 `stop_reason=end_of_data` 才表示按本次连续读取观察到末页。
- 字段元数据、审批、组织和表单管理仍取决于真实部署控制器。operations 映射只能描述调用，不能创造服务端能力。
- 源码、配置示例和发行包不包含实际密钥；文档示例只使用占位符。
