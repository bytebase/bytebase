package pg

import (
	omnipg "github.com/bytebase/omni/pg"
	"github.com/bytebase/omni/pg/analysis"
	"github.com/pkg/errors"
)

// ExtractPrependStatement returns the first SET role or SET search_path
// statement in statement, trimmed and terminated with a semicolon, or "" if
// none is present.
func ExtractPrependStatement(statement string) (string, error) {
	stmts, err := omnipg.Parse(statement)
	if err != nil {
		return "", errors.Wrap(err, "failed to parse statement")
	}
	prepend := analysis.PrependStatements(stmts)
	if prepend == "" {
		return "", nil
	}
	return prepend + ";", nil
}
