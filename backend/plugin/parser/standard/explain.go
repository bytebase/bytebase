package standard

import (
	"regexp"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/plugin/parser/tokenizer"
)

var explainKeyword = regexp.MustCompile(`(?i)^\s*EXPLAIN\b`)

// ExplainStatement explains statement by prefixing EXPLAIN, which returns the
// engine's default plan, so format goes unused. A statement that already is
// an EXPLAIN is refused rather than given a second one: telling whether it runs
// its statement would take the engine's own parser.
func ExplainStatement(statement, _ string) (string, error) {
	// A statement the tokenizer cannot read is prefixed anyway, and the database
	// rejects it.
	if text, err := tokenizer.StandardRemoveQuotedTextAndComment(statement); err == nil && explainKeyword.MatchString(text) {
		return "", errors.New("the statement is already an EXPLAIN; run it to see its plan")
	}
	return "EXPLAIN " + statement, nil
}
