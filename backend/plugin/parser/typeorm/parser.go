// Package typeorm defines the sql extractor for typeorm migration scripts.
package typeorm

import (
	"regexp"
	"strings"

	"github.com/pkg/errors"
)

// Parse parses the typescript migration file.
func Parse(s string) ([]string, error) {
	upIndex := strings.Index(s, "public async up(")
	if upIndex < 0 {
		return nil, errors.Errorf("unable to find up() function")
	}
	s = s[upIndex:]
	downIndex := strings.Index(s, "public async down(")
	if downIndex >= 0 {
		s = s[:downIndex]
	}
	pattern := `await queryRunner\.query\(\s*(.+?)\s*[,]?\s*\);`

	// Compile the regular expression
	re := regexp.MustCompile(pattern)

	// Find all matched statements
	matches := re.FindAllStringSubmatch(s, -1)
	var stmts []string
	for _, m := range matches {
		stmt := m[1]
		if len(stmt) <= 2 {
			return nil, errors.Errorf("invalid statement %s", stmt)
		}
		if stmt[0] != stmt[len(stmt)-1] {
			return nil, errors.Errorf("invalid statement %s", stmt)
		}
		stmts = append(stmts, stmt[1:len(stmt)-1])
	}
	return stmts, nil
}
