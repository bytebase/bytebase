package model

import "strings"

// PGSearchPathItem is a parsed PostgreSQL search_path entry.
type PGSearchPathItem struct {
	Schema      string
	CurrentUser bool
}

// ParsePGConfiguredSearchPath parses a PostgreSQL search_path setting while preserving
// non-system configured entries, including the special $user placeholder.
func ParsePGConfiguredSearchPath(searchPath string) []PGSearchPathItem {
	if searchPath == "" {
		return []PGSearchPathItem{}
	}

	var result []PGSearchPathItem
	for _, token := range splitPGSearchPath(searchPath) {
		item, ok := parsePGSearchPathItem(token)
		if !ok {
			continue
		}
		result = append(result, item)
	}
	return result
}

// ResolvePGSearchPath resolves a parsed PostgreSQL search_path for a current user.
// When schemaExists is provided, schemas missing from metadata are omitted.
func ResolvePGSearchPath(configured []PGSearchPathItem, currentUser string, schemaExists func(string) bool) []string {
	if len(configured) == 0 {
		return []string{}
	}

	currentUser = strings.TrimSpace(currentUser)

	var result []string
	for _, item := range configured {
		schema, ok := item.resolve(currentUser)
		if !ok {
			continue
		}
		if schemaExists != nil && !schemaExists(schema) {
			continue
		}
		result = append(result, schema)
	}
	return result
}

func (i PGSearchPathItem) resolve(currentUser string) (string, bool) {
	if i.CurrentUser {
		if currentUser == "" {
			return "", false
		}
		return currentUser, true
	}
	if i.Schema == "" {
		return "", false
	}
	return i.Schema, true
}

func parsePGSearchPathItem(token string) (PGSearchPathItem, bool) {
	token = strings.TrimSpace(token)
	if token == "" {
		return PGSearchPathItem{}, false
	}

	schema := strings.ToLower(token)
	switch {
	case len(token) >= 2 && token[0] == '"' && token[len(token)-1] == '"':
		schema = strings.ReplaceAll(token[1:len(token)-1], `""`, `"`)
	case len(token) >= 2 && token[0] == '\'' && token[len(token)-1] == '\'':
		schema = strings.ReplaceAll(token[1:len(token)-1], `''`, `'`)
	default:
	}

	if schema == "$user" {
		return PGSearchPathItem{CurrentUser: true}, true
	}
	if isPGSystemPath(schema) {
		return PGSearchPathItem{}, false
	}
	if schema == "" {
		return PGSearchPathItem{}, false
	}
	return PGSearchPathItem{Schema: schema}, true
}

func splitPGSearchPath(searchPath string) []string {
	var parts []string
	start := 0
	var quote byte

	for i := 0; i < len(searchPath); i++ {
		switch {
		case quote != 0:
			if searchPath[i] != quote {
				continue
			}
			if i+1 < len(searchPath) && searchPath[i+1] == quote {
				i++
				continue
			}
			quote = 0
		case searchPath[i] == '"' || searchPath[i] == '\'':
			quote = searchPath[i]
		case searchPath[i] == ',':
			parts = append(parts, searchPath[start:i])
			start = i + 1
		default:
		}
	}
	parts = append(parts, searchPath[start:])
	return parts
}

// isPGSystemPath checks if a path is a PostgreSQL system schema.
func isPGSystemPath(path string) bool {
	systemSchemas := []string{"pg_catalog", "information_schema", "pg_toast", "pg_temp_1", "pg_temp_2", "pg_global"}
	for _, schema := range systemSchemas {
		if strings.EqualFold(path, schema) {
			return true
		}
	}
	return false
}
