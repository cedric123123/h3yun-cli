package cli

// These are documented semantics, not a claim of live-account verification.
// Keep them visible in compact Schema as well as in write receipts.
const integrationDoc = "https://h3yun-pro-doc-openapi.apifox.cn/"

func commandContract(command string) map[string]any {
	pages := map[string]string{
		"records get": "188676261e0", "records list": "188676262e0",
		"records create": "188676263e0", "records create-many": "188676264e0",
		"records update": "188676265e0", "records delete": "188676266e0",
		"files upload": "188676270e0", "files download": "188676267e0",
		"custom invoke": "188676268e0",
	}
	page, ok := pages[command]
	if !ok {
		return nil
	}
	c := map[string]any{
		"source": integrationDoc + page + ".md", "checked_on": "2026-10-15",
		"authentication": "EngineCode/EngineSecret", "max_timeout_seconds": 60,
		"request_rate":  "1-2 requests/second per integration; caller must coordinate all processes; CLI has no shared rate limiter",
		"live_verified": false,
	}
	switch command {
	case "records create":
		c["submit_semantics"] = map[string]string{"false": "普通表单草稿；流程表单停在发起节点", "true": "普通表单生效；流程表单推进到下一审批节点"}
	case "records create-many":
		c["submit_semantics"] = map[string]string{"false": "草稿", "true": "直接生效，不创建进行中的审批数据"}
		c["recommended_batch_size"] = 500
		c["batch_limit_is_advisory"] = true
	case "records list":
		c["return_items"] = "Filter.ReturnItems selects values; unselected fields may still appear as null"
	case "files upload":
		c["child_row"] = "With --child-schema, --id is the child row ID and --field excludes the child schema prefix"
		c["wire_format"] = "multipart field media (official example); Apifox uses file; compatibility needs live verification"
		c["official_source"] = "https://help.h3yun.com/contents/1012/1638.html"
	case "files download":
		c["wire_format"] = "application/x-www-form-urlencoded (official example); Apifox uses multipart/form-data"
		c["official_source"] = "https://help.h3yun.com/api/preview/1/1013/1639"
	}
	return c
}

func operationWarnings(command string, body map[string]any) []string {
	action, _ := body["ActionName"].(string)
	var warnings []string
	switch action {
	case "CreateBizObject", "CreateBizObjects":
		warnings = append(warnings, "创建会触发自动化；生效数据还会触发业务规则。")
		if submit, _ := body["IsSubmit"].(bool); submit {
			if action == "CreateBizObject" {
				warnings = append(warnings, "单条提交：普通表单生效；流程表单跳过发起节点，进入下一审批节点。")
			} else {
				warnings = append(warnings, "批量提交直接创建生效数据，不能当作批量发起审批；进行中流程需自定义接口。")
			}
		}
		if action == "CreateBizObjects" {
			warnings = append(warnings, "建议每批不超过 500 条，复杂记录需更少；失败可能部分成功，不要整批自动重放。")
		}
	case "UpdateBizObject":
		warnings = append(warnings, "更新会触发自动化；更新生效数据会先触发作废规则，再触发生效规则；不等同于审批。")
	case "RemoveBizObject":
		warnings = append(warnings, "删除会触发自动化；删除生效数据还会触发作废业务规则。")
	}
	if command == "files upload" {
		warnings = append(warnings, "先保存记录再上传；子表附件必须使用子表行 ID 和不含子表前缀的字段编码。")
	}
	return warnings
}
