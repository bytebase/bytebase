package tidb

import (
	"context"
	"fmt"
	"strings"

	"github.com/bytebase/omni/tidb/ast"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	advisorcode "github.com/bytebase/bytebase/backend/plugin/advisor/code"
)

var (
	_ advisor.Advisor = (*CollationAllowlistAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_TIDB, storepb.SQLReviewRule_SYSTEM_COLLATION_ALLOWLIST, &CollationAllowlistAdvisor{})
}

// CollationAllowlistAdvisor checks for collation allowlist.
type CollationAllowlistAdvisor struct {
}

// Check is Recipe A. Same wrapper-safety rationale as
// CharsetAllowlistAdvisor: EXPLAIN-DDL doesn't parse, the matched stmt
// types can't nest themselves.
func (*CollationAllowlistAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	stmts, err := getTiDBOmniNodes(checkCtx)
	if err != nil {
		return nil, err
	}
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	stringArrayPayload := checkCtx.Rule.GetStringArrayPayload()
	allowlist := make(map[string]bool)
	for _, collation := range stringArrayPayload.List {
		allowlist[strings.ToLower(collation)] = true
	}

	title := checkCtx.Rule.Type.String()
	var adviceList []*storepb.Advice

	emit := func(text, collation string, line int) {
		adviceList = append(adviceList, &storepb.Advice{
			Status:        level,
			Code:          advisorcode.DisabledCollation.Int32(),
			Title:         title,
			Content:       fmt.Sprintf("\"%s\" used disabled collation '%s'", text, collation),
			StartPosition: common.ConvertANTLRLineToPosition(line),
		})
	}

	for _, ostmt := range stmts {
		text := ostmt.TrimmedText()
		stmtLine := ostmt.FirstTokenLine()
		switch n := ostmt.Node.(type) {
		case *ast.CreateDatabaseStmt:
			if c := omniDatabaseOption(n.Options, omniOptionNamesCollate); c != "" {
				if _, ok := allowlist[c]; !ok {
					emit(text, c, stmtLine)
				}
			}
		case *ast.AlterDatabaseStmt:
			if c := omniDatabaseOption(n.Options, omniOptionNamesCollate); c != "" {
				if _, ok := allowlist[c]; !ok {
					emit(text, c, stmtLine)
				}
			}
		case *ast.CreateTableStmt:
			if c := omniTableOption(n.Options, omniOptionNamesCollate); c != "" {
				if _, ok := allowlist[c]; !ok {
					emit(text, c, stmtLine)
					continue
				}
			}
			for _, col := range n.Columns {
				if col == nil {
					continue
				}
				c := omniColumnCollate(col)
				if c == "" {
					continue
				}
				if _, ok := allowlist[c]; !ok {
					emit(text, c, ostmt.AbsoluteLine(col.Loc.Start))
					break
				}
			}
		case *ast.AlterTableStmt:
			// Pingcap parity: single ALTER TABLE → at most ONE advice. See
			// the matching comment in advisor_charset_allowlist.go for the
			// rationale (Codex P2 round-1 catch on PR #20217).
			var lastViolation string
			for _, cmd := range n.Commands {
				if cmd == nil {
					continue
				}
				switch cmd.Type {
				case ast.ATTableOption:
					if cmd.Option == nil {
						continue
					}
					if !omniOptionNameMatches(cmd.Option.Name, omniOptionNamesCollate) {
						continue
					}
					c := strings.ToLower(cmd.Option.Value)
					if c == "" {
						continue
					}
					if _, ok := allowlist[c]; !ok {
						lastViolation = c
					}
				case ast.ATConvertCharset:
					// `ALTER TABLE t CONVERT TO CHARACTER SET cs COLLATE col`.
					// Pingcap parses both charset and collation into
					// spec.Options (Tp=TableOptionCharset / Tp=TableOptionCollate).
					// Omni splits: charset on cmd.Name, collation on cmd.NewName
					// (empty if no COLLATE clause). cmd.Option is nil. Without
					// this case, mechanical migration silently misses the
					// COLLATE-on-CONVERT form (Codex P2 round-2 catch on
					// PR #20217).
					c := strings.ToLower(cmd.NewName)
					if c == "" {
						continue
					}
					if _, ok := allowlist[c]; !ok {
						lastViolation = c
					}
				case ast.ATAddColumn:
					for _, col := range addColumnTargets(cmd) {
						if col == nil {
							continue
						}
						c := omniColumnCollate(col)
						if c == "" {
							continue
						}
						if _, ok := allowlist[c]; !ok {
							lastViolation = c
							break // pingcap parity: only first violating column per spec
						}
					}
				case ast.ATChangeColumn, ast.ATModifyColumn:
					if cmd.Column == nil {
						continue
					}
					c := omniColumnCollate(cmd.Column)
					if c == "" {
						continue
					}
					if _, ok := allowlist[c]; !ok {
						lastViolation = c
					}
				default:
				}
			}
			if lastViolation != "" {
				emit(text, lastViolation, stmtLine)
			}
		default:
		}
	}

	return adviceList, nil
}
