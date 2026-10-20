---
name: h3yun-records
description: 使用 h3yun CLI 查询、创建、串行批量更新或删除氚云表单数据，包含有界分页。不用于搭建表单结构或执行审批。
---

# 表单记录

命令为 records get/list/create/create-many/update/update-many/delete。参数不确定时读取精确 leaf：`h3yun schema "records create" --compact`。CLI 使用宿主注入的凭据；同一链路固定 profile。

- schema 参数是业务表单编码，不是 CLI 的命令 Schema。字段编码和记录 ID 来自真实配置或返回，不从字段标签猜测。
- data 接受 JSON 对象；create-many 接受对象数组；用 --data - 或 --data @文件。CLI 自动处理平台字符串编码，不手动二次编码。
- create 默认草稿，--submit 才创建生效数据。update 不是审批/提交已有草稿。
- list 默认一页、每页最多 500。需要 CLI 连续翻页时必须同时给 `--all --max N`，N 为 1..100000；累计上限 64 MiB、请求间隔至少 700ms、失败即停且不重试。它不是一致性快照，按 meta.pagination 的 partial/stop_reason/next_offset 判断完成度和续跑位置。
- update-many 接受 1..500 个唯一 `{id,data}`，串行调用单条更新、非原子、失败即停。分别处理 succeeded_ids、failed_ids、unknown_ids、unprocessed_ids；未知结果先查现状，不整批重放。
- 删除先确定精确 schema/id，实际执行使用 --yes；dry-run 可审查包体，不代替用户授权。
- 输出读取 ok/data/error/meta；批量失败保留成功数据。outcome_unknown 时先确认现状，不整批重放。
- `--fields` 是 data 相对点路径输出投影，数组逐项处理；它不减少服务端返回量，也不改变权限。频率仍需宿主跨进程协调。
- 在任务需要时用返回记录 ID 做只读验证，再报告实际完成数量和失败项。
