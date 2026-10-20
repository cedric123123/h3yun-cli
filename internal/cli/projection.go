package cli

import (
	"strings"

	"github.com/cedric123123/h3yun-cli/pkg/h3yun"
)

type projection struct {
	whole    bool
	children map[string]*projection
}

func projectionPaths(fields string) ([]string, error) {
	paths := strings.Split(fields, ",")
	for i, path := range paths {
		paths[i] = strings.TrimSpace(path)
		for _, part := range strings.Split(paths[i], ".") {
			if strings.TrimSpace(part) == "" {
				return nil, h3yun.Fail("USAGE", "fields must contain nonempty comma-separated dot paths", 2)
			}
		}
	}
	return paths, nil
}

func projectObject(object map[string]any, fields string) map[string]any {
	paths, _ := projectionPaths(fields)
	root := &projection{children: map[string]*projection{}}
	for _, path := range paths {
		parts := strings.Split(path, ".")
		// Preserve the v0.3.x behavior for literal top-level keys containing dots.
		if _, exists := object[path]; exists {
			parts = []string{path}
		}
		node := root
		for _, part := range parts {
			if node.children[part] == nil {
				node.children[part] = &projection{children: map[string]*projection{}}
			}
			node = node.children[part]
		}
		node.whole = true
	}
	return projectValue(object, root).(map[string]any)
}

func projectValue(value any, selection *projection) any {
	if selection.whole {
		return value
	}
	switch v := value.(type) {
	case map[string]any:
		out := map[string]any{}
		for key, child := range selection.children {
			if item, ok := v[key]; ok {
				out[key] = projectValue(item, child)
			}
		}
		return out
	case []any:
		out := make([]any, len(v))
		for i, item := range v {
			out[i] = projectValue(item, selection)
		}
		return out
	default:
		// A requested child path cannot expose an unrelated scalar value.
		return nil
	}
}
