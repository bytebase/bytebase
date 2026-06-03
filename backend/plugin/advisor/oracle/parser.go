// Package oracle is the advisor for oracle database.
package oracle

import (
	"fmt"
	"strings"
)

// normalizeIdentifierName normalizes the identifier name to the format of "schema"."table"."column".
// Including table name and column name.
func normalizeIdentifierName(name string) string {
	list := strings.Split(name, ".")
	var result []string
	for _, item := range list {
		result = append(result, fmt.Sprintf("\"%s\"", item))
	}
	return strings.Join(result, ".")
}

func lastIdentifier(name string) string {
	list := strings.Split(name, ".")
	return list[len(list)-1]
}
