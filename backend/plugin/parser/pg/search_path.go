package pg

import "strings"

func searchPathForSelectedSchema(selectedSchema string, defaultSearchPath []string) []string {
	if selectedSchema == "" {
		return defaultSearchPath
	}
	if selectedSchema == "public" || strings.ContainsRune(selectedSchema, '\x00') {
		return []string{selectedSchema}
	}
	return []string{selectedSchema, "public"}
}
