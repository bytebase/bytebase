package tidb

import (
	"strings"

	omniast "github.com/bytebase/omni/tidb/ast"
	"github.com/pkg/errors"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	base.RegisterExplainStatementFunc(storepb.Engine_TIDB, explainStatement)
}

// tidbJSONFormat is TiDB's name for the plan as a tree of JSON objects. Its
// other JSON-ish format, plain `json`, TiDB rejects outright.
//
// TiDB added tidb_json in v6.5.0 (December 2022) and does not know the name
// before it. Nothing here reads the server's version, so a server older than
// that answers an explain request with its own error rather than a plan.
const tidbJSONFormat = "tidb_json"

// explainStatement implements base.ExplainStatement. TiDB spells its structured
// plan FORMAT=tidb_json, unquoted so that what this builds parses again; any
// other format the caller names gets the engine's default plan, since the rest
// of TiDB's FORMAT values are shapes nothing downstream reads (the driver
// capability table refuses them).
func explainStatement(statement string, format base.ExplainFormat) (string, error) {
	node, err := singleStatement(statement)
	if err != nil {
		return "", err
	}
	planned := statement
	if explain, ok := node.(*omniast.ExplainStmt); ok {
		if explain.Analyze {
			return "", base.ErrExplainAnalyzeExecutes
		}
		if inner, ok := explainedText(explain, statement); ok {
			planned = inner
		} else {
			// Nothing to rebuild around, as for EXPLAIN FOR CONNECTION or the
			// EXPLAIN tbl_name that describes a table. Run what the user wrote
			// rather than plan a plan.
			return statement, nil
		}
	}
	if format == base.ExplainFormatJSON {
		return "EXPLAIN FORMAT=" + tidbJSONFormat + " " + planned, nil
	}
	return "EXPLAIN " + planned, nil
}

// DescribeExplain returns the output format of an EXPLAIN statement and whether
// it executes the statement it plans. It is not an explain at all unless it
// plans a statement: EXPLAIN tbl_name and DESCRIBE tbl_name parse here too and
// describe a table's columns, which is a result like any other.
func DescribeExplain(statement string) (format string, executed bool, ok bool) {
	node, err := singleStatement(statement)
	if err != nil {
		return "", false, false
	}
	explain, ok := node.(*omniast.ExplainStmt)
	if !ok {
		return "", false, false
	}
	if _, plans := explainedStart(explain.Stmt); !plans {
		return "", false, false
	}
	if strings.EqualFold(explain.Format, tidbJSONFormat) {
		return "json", explain.Analyze, true
	}
	// Every other format TiDB prints is a table or a tree of lines.
	return "text", explain.Analyze, true
}

// singleStatement returns the one statement in text. An explain request plans one
// statement at a time: both the drivers and the API gate split a multi-statement
// request first.
func singleStatement(text string) (omniast.Node, error) {
	list, err := ParseTiDBOmni(text)
	if err != nil {
		return nil, convertOmniParseError(err, base.Statement{Text: text})
	}
	if list == nil || len(list.Items) == 0 {
		return nil, errors.New("EXPLAIN needs a statement to plan")
	}
	if len(list.Items) > 1 {
		return nil, errors.New("EXPLAIN plans a single statement")
	}
	return list.Items[0], nil
}

// explainedText returns the text within text of the statement explain plans, and
// whether it could be located.
func explainedText(explain *omniast.ExplainStmt, text string) (string, bool) {
	// The parser reads the body of an executable comment, so a location can fall
	// inside one and the slice would leave the comment unclosed.
	if strings.Contains(text, "/*!") || strings.Contains(text, "/*T!") {
		return "", false
	}
	start, ok := explainedStart(explain.Stmt)
	if !ok {
		return "", false
	}
	// The plan ends where the EXPLAIN does, which is past clauses that the
	// planned statement's own location can leave out.
	if start < 0 || start >= explain.Loc.End || explain.Loc.End > len(text) {
		return "", false
	}
	return text[start:explain.Loc.End], true
}

// explainedStart returns where a statement that EXPLAIN can plan begins. The
// cases are what the grammar accepts after EXPLAIN; anything else it parses there
// describes a table rather than planning a statement.
func explainedStart(node omniast.Node) (int, bool) {
	switch n := node.(type) {
	case *omniast.SelectStmt:
		return n.Loc.Start, true
	// REPLACE parses as an InsertStmt.
	case *omniast.InsertStmt:
		return n.Loc.Start, true
	case *omniast.UpdateStmt:
		return n.Loc.Start, true
	case *omniast.DeleteStmt:
		return n.Loc.Start, true
	case *omniast.TableStmt:
		return n.Loc.Start, true
	case *omniast.ValuesStmt:
		return n.Loc.Start, true
	default:
		return 0, false
	}
}
