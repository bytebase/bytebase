package plsql

import (
	"regexp"
	"strconv"

	"github.com/bytebase/omni/oracle/analysis"
	"github.com/bytebase/omni/oracle/ast"
	"github.com/pkg/errors"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	base.RegisterParseStatementsFunc(storepb.Engine_ORACLE, parsePLSQLStatements)
	base.RegisterGetStatementTypes(storepb.Engine_ORACLE, GetStatementTypes)
}

// parsePLSQLStatements is the ParseStatementsFunc for Oracle (PL/SQL).
// Returns []ParsedStatement with both text and AST populated.
func parsePLSQLStatements(statement string) ([]base.ParsedStatement, error) {
	stmts, err := SplitSQL(statement)
	if err != nil {
		return nil, err
	}

	var result []base.ParsedStatement
	for _, stmt := range stmts {
		if stmt.Empty {
			result = append(result, base.ParsedStatement{Statement: stmt})
			continue
		}

		list, err := ParsePLSQL(stmt.Text)
		if err != nil {
			return nil, convertError(err, stmt)
		}
		if list == nil || len(list.Items) == 0 {
			result = append(result, base.ParsedStatement{Statement: stmt})
			continue
		}
		for _, node := range list.Items {
			raw, ok := node.(*ast.RawStmt)
			if !ok {
				continue
			}
			result = append(result, base.ParsedStatement{
				Statement: stmt,
				AST: &AST{
					Node:          raw.Stmt,
					Text:          stmt.Text,
					StartPosition: stmt.Start,
				},
			})
		}
	}

	return result, nil
}

// Version is an Oracle release, such as 12.2.
type Version = analysis.Version

func ParseVersion(banner string) (*Version, error) {
	re := regexp.MustCompile(`(\d+)\.(\d+)`)
	match := re.FindStringSubmatch(banner)
	if len(match) >= 3 {
		firstVersion, err := strconv.Atoi(match[1])
		if err != nil {
			return nil, errors.Errorf("failed to parse first version from banner: %s", banner)
		}
		secondVersion, err := strconv.Atoi(match[2])
		if err != nil {
			return nil, errors.Errorf("failed to parse second version from banner: %s", banner)
		}
		return &Version{First: firstVersion, Second: secondVersion}, nil
	}
	return nil, errors.Errorf("failed to parse version from banner: %s", banner)
}
