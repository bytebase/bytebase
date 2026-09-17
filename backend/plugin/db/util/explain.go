package util

import (
	"regexp"

	"github.com/pkg/errors"

	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/plugin/db"
)

var explainKeyword = regexp.MustCompile(`(?i)^EXPLAIN\b`)

// PrefixExplain is the explain of an engine that plans a statement prefixed
// with EXPLAIN, which returns a text plan.
func PrefixExplain(defaultFormat v1pb.QueryOption_ExplainFormat) db.Explain {
	return db.Explain{
		Formats:       []v1pb.QueryOption_ExplainFormat{v1pb.QueryOption_TEXT},
		DefaultFormat: defaultFormat,
		Statement:     explainByPrefix,
	}
}

// explainByPrefix refuses a statement that already is an EXPLAIN rather than
// prefixing a second one: telling whether that EXPLAIN runs its statement would
// take the engine's parser.
func explainByPrefix(statement string, _ v1pb.QueryOption_ExplainFormat) (string, error) {
	// An unclosed literal leaves nothing to match, and the database rejects the
	// doubled EXPLAIN instead.
	text, _ := RemoveCommentsAndTrim(statement)
	if explainKeyword.MatchString(text) {
		return "", errors.New("the statement is already an EXPLAIN; run it to see its plan")
	}
	return "EXPLAIN " + statement, nil
}
