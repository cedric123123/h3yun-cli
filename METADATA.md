# 字段元数据接入

氚云公有云 OpenAPI 目录中没有本项目已核实、可直接读取完整字段定义的原生接口。v0.4.0 的 `schema fields` 因而是一个受约束的适配入口：它只调用使用者已经部署并在 operations 文件中声明为只读的自定义控制器。CLI 本身不部署控制器，也不猜测在线编程对象模型、ActionName 或字段结构。

## 服务端前提

控制器应当：

- 只读取调用者有权查看的指定表单元数据，不创建、更新或删除任何业务数据；
- 在服务端校验应用、组织和表单权限，只接受约定的表单编码参数；
- 返回标准 `Successful` / `ErrorMessage` / `ReturnData` envelope，不返回 EngineSecret、连接字符串或其他服务端秘密；
- 明确定义字段编码、显示名、类型、必填、选项、子表等结构。CLI 会原样保留这份契约，不做字段名转换。

`code`、`name`、`type`、`required`、`options` 是常见且便于 Agent 使用的字段语义，但不是 CLI 强制的键名。请以实际部署控制器的版本化契约为准。

## operations 映射

下面只是结构示例，路由值必须替换为真实部署值：

```json
{
  "schema.fields": {
    "action": "YOUR_VERIFIED_FIELDS_ACTION",
    "controller": "YOUR_DEPLOYED_METADATA_CONTROLLER",
    "app_code": "YOUR_APP_CODE",
    "description": "读取指定表单字段定义",
    "read_only": true,
    "required": ["SchemaCode"],
    "success_field": "status",
    "success_value": "ok"
  }
}
```

如果控制器没有 ReturnData 级业务状态，可省略 `success_field` 和 `success_value`。CLI 始终要求该操作存在且 `read_only=true`；`operations.example.json` 中的 `REPLACE_WITH_...` 占位路由会被拒绝，避免误报能力可用。

通过配置文件的 `operations_file` 或 `H3YUN_OPERATIONS` 指向映射文件。文件只保存路由和契约，不应保存密钥。EngineCode/EngineSecret 仍由调用进程环境提供。

## 调用

默认操作名为 `schema.fields`，默认表单参数名为 `SchemaCode`：

```sh
h3yun schema fields --schema D0000000000000000000
```

需要查看该命令本身的机器可读参数契约时，使用 `h3yun schema "schema fields" --compact`；双词路径作为一个参数传入，避免与实际执行命令混淆。

若真实控制器使用其他表单参数名，可显式指定：

```sh
h3yun schema fields --schema D0000000000000000000 --schema-param FormCode
```

也可用 `--operation` 选择另一个已配置的只读操作。参数名不能覆盖 ActionName、Controller、AppCode、EngineCode 或 EngineSecret。若映射还要求表单编码之外的参数，`schema fields` 会停止；此时应先核对契约，并按需通过 `ops run` 明确传入完整参数，而不是让 CLI 猜默认值。

先用 `--dry-run` 检查最终路由和包体，再进行真实调用。成功响应的 `meta.metadata_source` 为 `configured_custom_controller`，这只说明来源类型，不是对字段内容的背书。配置了 `success_field/success_value` 时，平台 envelope 成功但业务状态不匹配会返回 `CUSTOM_BUSINESS`。

## 验收清单

1. 在测试表单验证普通字段、必填字段、选项字段、人员/部门、关联和子表的返回结构。
2. 用无权访问的表单验证控制器拒绝越权读取，并确认错误中不泄露内部信息。
3. 确认调用不产生业务记录、流程、日志填报或其他写入副作用。
4. 固定控制器契约版本，并让 Agent 只使用响应中实际存在的字段编码。
5. 控制器升级后重新执行本地模拟和目标环境只读验收；不能把一个表单成功外推为所有表单可用。
