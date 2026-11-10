package mongodb

import (
	"strings"

	"github.com/bytebase/omni/mongo/ast"
	"github.com/pkg/errors"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	base.RegisterExplainStatementFunc(storepb.Engine_MONGODB, explainStatement)
}

// explainCursorMethod is how a mongosh statement asks for its own plan: a
// cursor method on the end of the query, not a keyword in front of it.
const explainCursorMethod = "explain"

// planVerbosity is the one verbosity that does not run the statement it plans,
// and is also MongoDB's default. The other two, executionStats and
// allPlansExecution, execute the query to measure it.
const planVerbosity = "queryPlanner"

// explainableMethods are the operations MongoDB will plan. A write, a findOne
// or a metadata read is not among them: either there is no plan to make, or
// making it would mean running the statement.
var explainableMethods = map[string]bool{
	"aggregate":              true,
	"count":                  true,
	"countDocuments":         true,
	"distinct":               true,
	"estimatedDocumentCount": true,
	"find":                   true,
}

// explainStatement implements base.ExplainStatement. MongoDB states a plan as
// one document and has no second format, so the format a caller asks for has
// nowhere to go; the driver capability table offers this engine JSON alone.
//
// A statement that already asks for its own plan has its verbosity replaced
// rather than kept, which is what keeps an explain request from running the
// query it is meant to describe.
func explainStatement(statement string, _ base.ExplainFormat) (string, error) {
	collection, err := singleCollectionStatement(statement)
	if err != nil {
		return "", err
	}
	if !explainableMethods[collection.Method] {
		return "", errors.Errorf("MongoDB plans aggregate, count, countDocuments, distinct, estimatedDocumentCount and find; %q is none of those", collection.Method)
	}

	for _, cursor := range collection.CursorMethods {
		if cursor.Method != explainCursorMethod {
			continue
		}
		return statement[collection.Loc.Start:cursor.Loc.Start] +
			explainCall() +
			statement[cursor.Loc.End:collection.Loc.End], nil
	}
	return statement[collection.Loc.Start:collection.Loc.End] + "." + explainCall(), nil
}

func explainCall() string {
	return explainCursorMethod + `("` + planVerbosity + `")`
}

// DescribeExplain returns the output format of a statement that asks for its
// own plan, and whether producing that plan runs the statement.
//
// MongoDB has one plan format, so the format is JSON or the statement is not an
// explain at all. A verbosity other than queryPlanner measures the query by
// running it, which the caller needs to know: such a plan is an executed one.
func DescribeExplain(statement string) (format string, executed bool, ok bool) {
	collection, err := singleCollectionStatement(statement)
	if err != nil {
		return "", false, false
	}
	for _, cursor := range collection.CursorMethods {
		if cursor.Method != explainCursorMethod {
			continue
		}
		return "json", explainExecutes(statement, cursor), true
	}
	return "", false, false
}

// explainExecutes reports whether the verbosity the caller named runs the
// statement. An omitted verbosity is queryPlanner, which does not.
func explainExecutes(statement string, cursor ast.CursorMethod) bool {
	if len(cursor.Args) == 0 {
		return false
	}
	loc := cursor.Args[0].GetLoc()
	if loc.Start < 0 || loc.End > len(statement) || loc.Start >= loc.End {
		// A verbosity this parser cannot read is assumed to measure, since
		// calling an executed plan an estimate is the costlier mistake.
		return true
	}
	return strings.Trim(statement[loc.Start:loc.End], `"'`) != planVerbosity
}

// singleCollectionStatement returns the one query over a collection in text. An
// explain request plans one statement at a time: both the driver and the API
// gate split a multi-statement request first.
func singleCollectionStatement(statement string) (*ast.CollectionStatement, error) {
	statements, err := ParseMongoShell(statement)
	if err != nil {
		return nil, err
	}
	if len(statements) == 0 {
		return nil, errors.New("a plan needs a statement to plan")
	}
	if len(statements) > 1 {
		return nil, errors.New("a plan covers a single statement")
	}
	node, ok := GetOmniNode(statements[0].AST)
	if !ok || node == nil {
		return nil, errors.New("the statement cannot be planned")
	}
	collection, ok := node.(*ast.CollectionStatement)
	if !ok {
		return nil, errors.New("MongoDB plans a query over a collection, and this statement queries none")
	}
	return collection, nil
}
