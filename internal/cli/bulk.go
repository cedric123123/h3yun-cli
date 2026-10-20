package cli

import (
	"context"
	"io"
	"strings"

	"github.com/cedric123123/h3yun-cli/pkg/h3yun"
)

const maxAggregateBytes = 64 << 20

// listAll is bounded by both record count and serialized row bytes. It never
// retries or silently returns success after a page failure.
func listAll(ctx context.Context, c *h3yun.Client, schema string, filter map[string]any, maximum int, meta map[string]any) (any, error) {
	from, _ := integer(filter["FromRowNum"])
	to, _ := integer(filter["ToRowNum"])
	pageSize, start := to-from, from
	rows := []any{}
	returnData := map[string]any{}
	pages, attempts, bytes := 0, 0, 0
	finish := func(reason string, complete bool, failed map[string]any, err error) (any, error) {
		meta["pagination"] = map[string]any{"offset": start, "limit": pageSize, "returned": len(rows), "next_offset": from, "pages": pages, "requests": attempts, "max": maximum, "complete": complete, "has_more_possible": !complete, "stop_reason": reason, "partial": err != nil, "snapshot": false}
		returnData["BizObjectArray"] = rows
		result := map[string]any{"Successful": err == nil, "ReturnData": returnData}
		if failed != nil {
			result["failed_response"] = failed
		}
		return result, err
	}
	for len(rows) < maximum {
		size := pageSize
		if remaining := maximum - len(rows); size > remaining {
			size = remaining
		}
		if from > int(^uint(0)>>1)-size {
			return finish("offset_overflow", false, nil, h3yun.Fail("INPUT", "pagination offset overflow", 2))
		}
		filter["FromRowNum"], filter["ToRowNum"] = from, from+size
		attempts++
		response, err := c.Invoke(ctx, map[string]any{"ActionName": "LoadBizObjects", "SchemaCode": schema, "Filter": encoded(filter)}, false)
		if err != nil {
			return finish("page_failed", false, response, err)
		}
		data, _ := response["ReturnData"].(map[string]any)
		page, ok := data["BizObjectArray"].([]any)
		if !ok || len(page) > size {
			return finish("invalid_page", false, response, h3yun.Fail("PROTOCOL", "list response must contain an array no larger than the requested page", 4))
		}
		if pages == 0 {
			for key, value := range data {
				if key != "BizObjectArray" {
					returnData[key] = value
				}
			}
		}
		pageBytes := len(encoded(page))
		if pageBytes > maxAggregateBytes-bytes {
			return finish("output_limit", false, nil, h3yun.Fail("OUTPUT_LIMIT", "aggregate rows exceed 64 MiB; resume from next_offset with a smaller --max", 6))
		}
		rows = append(rows, page...)
		bytes += pageBytes
		pages++
		from += len(page)
		if len(page) < size {
			return finish("end_of_data", true, nil, nil)
		}
	}
	return finish("max_reached", false, nil, nil)
}

type updateItem struct {
	ID   string
	Data map[string]any
}

func parseUpdates(source string, stdin io.Reader) ([]updateItem, error) {
	value, err := readJSON(source, stdin, true)
	if err != nil {
		return nil, err
	}
	items := value.([]any)
	if len(items) > 500 {
		return nil, h3yun.Fail("INPUT", "update-many accepts at most 500 items per invocation", 2)
	}
	updates := make([]updateItem, 0, len(items))
	seen := map[string]bool{}
	for _, item := range items {
		object := item.(map[string]any)
		id, okID := object["id"].(string)
		data, okData := object["data"].(map[string]any)
		if len(object) != 2 || !okID || strings.TrimSpace(id) == "" || id != strings.TrimSpace(id) || !okData || len(data) == 0 || seen[id] {
			return nil, h3yun.Fail("INPUT", "each update must contain only unique nonempty id and nonempty data object", 2)
		}
		seen[id] = true
		updates = append(updates, updateItem{id, data})
	}
	return updates, nil
}

func updateRequest(schema string, item updateItem) map[string]any {
	return map[string]any{"ActionName": "UpdateBizObject", "SchemaCode": schema, "BizObjectId": item.ID, "BizObject": encoded(item.Data)}
}

func updateMany(ctx context.Context, c *h3yun.Client, schema string, items []updateItem, meta map[string]any) (any, error) {
	succeeded, failed, unknown, unprocessed := []string{}, []string{}, []string{}, []string{}
	for i, item := range items {
		response, err := c.Invoke(ctx, updateRequest(schema, item), true)
		if err == nil {
			succeeded = append(succeeded, item.ID)
			continue
		}
		attempted := i + 1
		if e, ok := err.(*h3yun.Error); ok && e.Code == "CANCELED" {
			unprocessed = append(unprocessed, item.ID)
			attempted = i
		} else if ok && e.OutcomeUnknown {
			unknown = append(unknown, item.ID)
		} else {
			failed = append(failed, item.ID)
		}
		for _, rest := range items[i+1:] {
			unprocessed = append(unprocessed, rest.ID)
		}
		meta["batch"] = map[string]any{"atomic": false, "complete": false, "stop_on_error": true, "attempted": attempted, "total": len(items)}
		return map[string]any{"succeeded_ids": succeeded, "failed_ids": failed, "unknown_ids": unknown, "unprocessed_ids": unprocessed, "failed_response": response}, err
	}
	meta["batch"] = map[string]any{"atomic": false, "complete": true, "stop_on_error": true, "attempted": len(items), "total": len(items)}
	return map[string]any{"succeeded_ids": succeeded, "failed_ids": failed, "unknown_ids": unknown, "unprocessed_ids": unprocessed}, nil
}
