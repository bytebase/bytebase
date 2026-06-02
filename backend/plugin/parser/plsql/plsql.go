package plsql

import (
	"regexp"
	"strconv"
	"strings"

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

		list, err := ParsePLSQLOmni(stmt.Text)
		if err != nil {
			return nil, convertOmniError(err, stmt)
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
				AST: &OmniAST{
					Node:          raw.Stmt,
					Text:          stmt.Text,
					StartPosition: stmt.Start,
				},
			})
		}
	}

	return result, nil
}

type Version struct {
	First  int
	Second int
}

// GTE returns true if the version is greater than or equal to the base version.
func (v *Version) GTE(base *Version) bool {
	if v.First > base.First {
		return true
	}
	if v.First == base.First {
		return v.Second >= base.Second
	}
	return false
}

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

// IsOracleKeyword returns true if the given text is an Oracle keyword.
func IsOracleKeyword(text string) bool {
	if len(text) == 0 {
		return false
	}

	return oracleKeywords[strings.ToUpper(text)] || oracleReservedWords[strings.ToUpper(text)]
}
