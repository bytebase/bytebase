// Package redshift implements the SQL advisor rules for Redshift.
package redshift

import (
	"fmt"

	redshiftast "github.com/bytebase/omni/redshift/ast"
	redshiftv1review "github.com/bytebase/omni/redshift/v1review"
	"github.com/bytebase/omni/v1review"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	parserredshift "github.com/bytebase/bytebase/backend/plugin/parser/redshift"
)

type redshiftStmt struct {
	node  redshiftast.Node
	start *storepb.Position
}

func init() {
	extract := func(checkCtx advisor.Context) ([]advisor.V1Stmt[redshiftStmt], error) {
		stmts := make([]advisor.V1Stmt[redshiftStmt], 0, len(checkCtx.ParsedStatements))
		for _, stmt := range checkCtx.ParsedStatements {
			node, ok := parserredshift.GetNode(stmt.AST)
			if !ok {
				continue
			}
			stmts = append(stmts, advisor.V1Stmt[redshiftStmt]{
				Node:     redshiftStmt{node: node, start: stmt.Start},
				Text:     stmt.Text,
				BaseLine: stmt.BaseLine(),
			})
		}
		return stmts, nil
	}
	byteOffsetLine := func(sql string, offset int) int32 {
		return parserredshift.ByteOffsetToRunePosition(sql, offset).Line
	}
	e := advisor.NewV1Engine(extract, byteOffsetLine, advisor.V1PosContentStart)

	// V1 anchored every finding on where the splitter starts the statement,
	// which can include leading whitespace or comments before the node.
	posStmtStart := func(stmt advisor.V1Stmt[redshiftStmt], _ v1review.Finding) *storepb.Position {
		return stmt.Node.start
	}

	advisor.RegisterFunc(storepb.Engine_REDSHIFT, storepb.SQLReviewRule_TABLE_DROP_NAMING_CONVENTION,
		e.NamingRegex(
			func(nodes []redshiftStmt, cfg v1review.NamingConfig) ([]v1review.Finding, error) {
				return redshiftv1review.TableDropNaming([]redshiftast.Node{nodes[0].node}, cfg)
			},
			code.TableDropNamingConventionMismatch,
			"naming_payload is required for table drop naming convention rule",
			func(f v1review.Finding, format string) string {
				return fmt.Sprintf("`%s` mismatches drop table naming convention, naming format should be %q", f.Name, format)
			},
			nil,
			posStmtStart,
		))
}
