package cli

import (
	"strings"

	"github.com/cedric123123/h3yun-cli/pkg/h3yun"
)

func metadataRequest(v values, s Settings) (map[string]any, *Operation, error) {
	name := v.s("operation")
	if name == "" {
		name = "schema.fields"
	}
	parameter := v.s("schema-param")
	if parameter == "" {
		parameter = "SchemaCode"
	}
	if strings.TrimSpace(parameter) == "" || parameter != strings.TrimSpace(parameter) {
		return nil, nil, h3yun.Fail("INPUT", "schema-param must not be blank", 2)
	}
	for _, reserved := range []string{"ActionName", "Controller", "AppCode", "EngineCode", "EngineSecret"} {
		if strings.EqualFold(parameter, reserved) {
			return nil, nil, h3yun.Fail("INPUT", "schema-param cannot override routing or credentials", 2)
		}
	}
	ops, err := operations(s.OperationsFile)
	if err != nil {
		return nil, nil, err
	}
	op, ok := ops[name]
	if !ok || !op.ReadOnly {
		e := h3yun.Fail("CAPABILITY_UNCONFIGURED", "schema fields requires a verified deployed read-only operations mapping", 3)
		e.Hint = "Configure schema.fields in operations_file; see METADATA.md. Field discovery is not a native public OpenAPI method."
		return nil, nil, e
	}
	body := map[string]any{"ActionName": op.Action, "Controller": op.Controller, "AppCode": op.AppCode, parameter: v.s("schema")}
	for _, required := range op.Required {
		if body[required] == nil {
			return nil, nil, h3yun.Fail("INPUT", "metadata mapping requires additional parameters; use ops run for this contract", 2)
		}
	}
	return body, &op, nil
}

func remoteDiagnosis(response map[string]any, err error, schema string) (any, error) {
	data := map[string]any{"network_tested": true, "schema": schema, "remote_verified": false, "scope": "read_access_to_requested_schema", "credentials_valid": "undetermined"}
	if err == nil {
		returned, _ := response["ReturnData"].(map[string]any)
		if _, ok := returned["BizObjectArray"].([]any); !ok {
			err = h3yun.Fail("PROTOCOL", "probe response must contain ReturnData.BizObjectArray", 4)
		}
	}
	if err == nil {
		data["remote_verified"], data["credentials_valid"], data["status"] = true, "accepted_for_this_request", "read_succeeded"
	} else {
		data["status"] = "probe_failed"
		if e, ok := err.(*h3yun.Error); ok {
			switch e.Code {
			case "TRANSPORT":
				data["status"] = "network_or_tls_error"
			case "API":
				data["status"] = "platform_rejected_check_credentials_schema_and_permissions"
			case "HTTP":
				data["status"] = "http_error"
			case "PROTOCOL":
				data["status"] = "unexpected_response"
			}
		}
	}
	return data, err
}
