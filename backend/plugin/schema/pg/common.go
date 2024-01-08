package pg

import "strings"

func escapePostgreSQLString(s string) string {
	return strings.ReplaceAll(s, "'", "''")
}

func unescapePostgreSQLString(s string) string {
	return strings.ReplaceAll(s, "''", "'")
}
