package mysql

import "strings"

var columnTypeCanonicalSynonyms = map[string]string{
	// Display width default, the display width attribute is deprecated for integer data types, but exist in
	// MySQL 5.7.
	"tinyint(4)":   "tinyint",
	"smallint(6)":  "smallint",
	"mediumint(9)": "mediumint",
	"int(11)":      "int",
	"bigint(20)":   "bigint",

	// Numeric Data Type Synonyms.
	"integer":        "int",
	"boolean":        "tinyint(1)",
	"decimal(10, 0)": "decimal",
	"dec":            "decimal",

	// TODO(zp): handle other column type synonyms and considering handling
	// variant display width match, for example, dec(20, 0) are equivalent with decimal(20, 0).
}

// GetColumnTypeCanonicalSynonym returns the canonical synonyms of the given column types,
// returns the original one if no synonym was found.
func GetColumnTypeCanonicalSynonym(columnType string) string {
	if canonical, ok := columnTypeCanonicalSynonyms[strings.ToLower(columnType)]; ok {
		return canonical
	}

	return columnType
}
