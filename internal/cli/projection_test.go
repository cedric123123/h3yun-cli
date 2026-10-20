package cli

import (
	"reflect"
	"strings"
	"testing"
)

func TestNestedProjection(t *testing.T) {
	input := map[string]any{
		"literal.dot": "legacy",
		"ReturnData": map[string]any{
			"BizObjectArray": []any{
				map[string]any{"ObjectId": "1", "Owner": map[string]any{"Name": "甲", "Secret": "x"}},
				map[string]any{"ObjectId": "2", "Owner": map[string]any{"Name": "乙", "Secret": "y"}},
			},
			"TotalCount": 2,
		},
		"ignored": true,
	}
	want := map[string]any{
		"literal.dot": "legacy",
		"ReturnData": map[string]any{"BizObjectArray": []any{
			map[string]any{"ObjectId": "1", "Owner": map[string]any{"Name": "甲"}},
			map[string]any{"ObjectId": "2", "Owner": map[string]any{"Name": "乙"}},
		}},
	}
	got := projectObject(input, "ReturnData.BizObjectArray.ObjectId,ReturnData.BizObjectArray.Owner.Name,literal.dot")
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got=%#v want=%#v", got, want)
	}
	if !reflect.DeepEqual(projectObject(input, "ReturnData,ReturnData.TotalCount"), map[string]any{"ReturnData": input["ReturnData"]}) {
		t.Fatal("parent selection must win")
	}
}

func TestProjectionValidationPreventsExecution(t *testing.T) {
	for _, value := range []string{"", "a.", ".a", "a,,b", "a..b"} {
		code, _, raw := invoke(t, []string{"records", "get", "--schema", "s", "--id", "i", "--fields", value}, "", nil)
		if code != 2 || !strings.Contains(raw, "nonempty") {
			t.Fatalf("%q: %d %s", value, code, raw)
		}
	}
}
