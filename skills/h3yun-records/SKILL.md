---
name: h3yun-records
description: 使用 h3yun CLI 查询、创建、更新或删除氚云表单数据，包含分页和批量创建。不用于搭建表单结构或执行审批。
---

# 表单记录

命令为 records get/list/create/create-many/update/delete。参数不确定时读取精确 leaf：`h3yun schema "records create" --compact`。CLI 使用宿主注入的凭据；同一链路固定 profile。

- schema 参数是业务表单编码，不是 CLI 的命令 Schema。字段编码和记录 ID 来自真实配置或返回，不从字段标签猜测。
- data 接受 JSON 对象；create-many 接受对象数组；用 --data - 或 --data @文件。CLI 自动处理平台字符串编码，不手动二次编码。
- create 默认草稿，--submit 才创建生效数据。update 不是审批/提交已有草稿。
- list 每页最多 500，--filter 与 offset/limit/count 互斥。按 meta.pagination 翻页并维持同一筛选，满页仅表示可能还有结果。
- 删除先确定精确 schema/id，实际执行使用 --yes；dry-run 可审查包体，不代替用户授权。
- 输出读取 ok/data/error/meta；批量失败保留成功数据。outcome_unknown 时先确认现状，不整批重放。
- 在任务需要时用返回记录 ID 做只读验证，再报告实际完成数量和失败项。
