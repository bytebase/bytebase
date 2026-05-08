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
	_ advisor.Advisor = (*CharsetAllowlistAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_TIDB, storepb.SQLReviewRule_SYSTEM_CHARSET_ALLOWLIST, &CharsetAllowlistAdvisor{})
}

// CharsetAllowlistAdvisor checks for charset allowlist.
type CharsetAllowlistAdvisor struct {
}

// Check is Recipe A. Empirically verified safe: EXPLAIN doesn't accept
// DDL in either pingcap or omni grammar (parse-error before reaching the
// advisor), and CreateDatabase/CreateTable/AlterDatabase/AlterTable
// can't nest themselves. No wrapper-statement risk.
func (*CharsetAllowlistAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
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
	for _, charset := range stringArrayPayload.List {
		allowlist[strings.ToLower(charset)] = true
	}

	title := checkCtx.Rule.Type.String()
	var adviceList []*storepb.Advice

	emit := func(text, charset string, line int) {
		adviceList = append(adviceList, &storepb.Advice{
			Status:        level,
			Code:          advisorcode.DisabledCharset.Int32(),
			Title:         title,
			Content:       fmt.Sprintf("\"%s\" used disabled charset '%s'", text, charset),
			StartPosition: common.ConvertANTLRLineToPosition(line),
		})
	}

	for _, ostmt := range stmts {
		text := ostmt.TrimmedText()
		stmtLine := ostmt.FirstTokenLine()
		switch n := ostmt.Node.(type) {
		case *ast.CreateDatabaseStmt:
			if cs := omniDatabaseOption(n.Options, omniOptionNamesCharset); cs != "" {
				if _, ok := allowlist[cs]; !ok {
					emit(text, cs, stmtLine)
				}
			}
		case *ast.AlterDatabaseStmt:
			if cs := omniDatabaseOption(n.Options, omniOptionNamesCharset); cs != "" {
				if _, ok := allowlist[cs]; !ok {
					emit(text, cs, stmtLine)
				}
			}
		case *ast.CreateTableStmt:
			// Table-level charset first; pingcap parity stops at the first
			// violation per top-level statement (table-level OR column-level,
			// not both).
			if cs := omniTableOption(n.Options, omniOptionNamesCharset); cs != "" {
				if _, ok := allowlist[cs]; !ok {
					emit(text, cs, stmtLine)
					continue
				}
			}
			for _, col := range n.Columns {
				if col == nil {
					continue
				}
				cs := omniColumnCharset(col)
				if cs == "" {
					continue
				}
				if _, ok := allowlist[cs]; !ok {
					emit(text, cs, ostmt.AbsoluteLine(col.Loc.Start))
					break
				}
			}
		case *ast.AlterTableStmt:
			// Pingcap parity: a single ALTER TABLE statement emits AT MOST
			// ONE advice regardless of how many commands violate. Pingcap's
			// pattern set `code`/`disabledCharset` inside the spec loop
			// without break, with one append after the loop — multiple
			// violations overwrite, last violation wins. Mirroring exactly
			// (Codex P2 round-1 catch on PR #20217).
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
					if !omniOptionNameMatches(cmd.Option.Name, omniOptionNamesCharset) {
						continue
					}
					cs := strings.ToLower(cmd.Option.Value)
					if cs == "" {
						continue
					}
					if _, ok := allowlist[cs]; !ok {
						lastViolation = cs
					}
				case ast.ATConvertCharset:
					// `ALTER TABLE t CONVERT TO CHARACTER SET cs [COLLATE c]`.
					// Pingcap parses this as AlterTableOption with the
					// charset in spec.Options (Tp=TableOptionCharset), so the
					// pingcap-typed advisor flagged it. Omni splits into
					// ATConvertCharset with charset on cmd.Name (and optional
					// collation on cmd.NewName); cmd.Option is nil. Without
					// this case, mechanical migration silently misses the
					// CONVERT form (Codex P2 round-2 catch on PR #20217).
					cs := strings.ToLower(cmd.Name)
					if cs == "" {
						continue
					}
					if _, ok := allowlist[cs]; !ok {
						lastViolation = cs
					}
				case ast.ATAddColumn:
					for _, col := range addColumnTargets(cmd) {
						if col == nil {
							continue
						}
						cs := omniColumnCharset(col)
						if cs == "" {
							continue
						}
						if _, ok := allowlist[cs]; !ok {
							lastViolation = cs
							break // pingcap parity: only first violating column per spec
						}
					}
				case ast.ATChangeColumn, ast.ATModifyColumn:
					if cmd.Column == nil {
						continue
					}
					cs := omniColumnCharset(cmd.Column)
					if cs == "" {
						continue
					}
					if _, ok := allowlist[cs]; !ok {
						lastViolation = cs
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
