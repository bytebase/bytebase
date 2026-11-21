// Package oracle implements the SQL advisor rules for Oracle.
package oracle

import (
	"context"
	"fmt"
	"strings"

	"github.com/bytebase/omni/oracle/ast"
	oraclev1review "github.com/bytebase/omni/oracle/v1review"
	"github.com/bytebase/omni/v1review"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	plsqlparser "github.com/bytebase/bytebase/backend/plugin/parser/plsql"
)

func init() {
	extract := func(checkCtx advisor.Context) ([]advisor.V1Stmt[ast.Node], error) {
		return advisor.ExtractNodes(checkCtx.ParsedStatements, plsqlparser.GetNode), nil
	}
	byteOffsetLine := func(sql string, offset int) int32 {
		return plsqlparser.ByteOffsetToRunePosition(sql, offset).Line
	}
	e := advisor.NewV1Engine(extract, byteOffsetLine, advisor.V1PosFindingOrFirstLine)

	advisor.Register(storepb.Engine_ORACLE, storepb.SQLReviewRule_BUILTIN_PRIOR_BACKUP_CHECK, oraclePriorBackup(e))
	advisor.Register(storepb.Engine_ORACLE, storepb.SQLReviewRule_COLUMN_ADD_NOT_NULL_REQUIRE_DEFAULT,
		e.SimpleStmt(oraclev1review.ColumnAddNotNullRequireDefault, code.NotNullColumnWithNoDefault, func(_ advisor.V1Stmt[ast.Node], f v1review.Finding) string {
			return fmt.Sprintf("Adding not null column %q requires default.", f.Name)
		}))
	advisor.Register(storepb.Engine_ORACLE, storepb.SQLReviewRule_COLUMN_COMMENT,
		e.ColumnComment(
			func(checkCtx advisor.Context, nodes []ast.Node, required bool, maxLength int) []v1review.Finding {
				return oraclev1review.ColumnComment(nodes, checkCtx.CurrentDatabase, required, maxLength)
			},
			-1,
			true,
			func(f v1review.Finding) string {
				return normalizeIdentifierName(f.Table + "." + f.Name)
			},
		))
	advisor.Register(storepb.Engine_ORACLE, storepb.SQLReviewRule_COLUMN_MAXIMUM_CHARACTER_LENGTH,
		e.NumberStmt(oraclev1review.ColumnMaximumCharacterLength, true, code.CharLengthExceedsLimit, func(_ advisor.V1Stmt[ast.Node], _ v1review.Finding, maximum int) string {
			return fmt.Sprintf("The maximum character length is %d.", maximum)
		}))
	advisor.Register(storepb.Engine_ORACLE, storepb.SQLReviewRule_COLUMN_MAXIMUM_VARCHAR_LENGTH,
		e.NumberStmt(oraclev1review.ColumnMaximumVarcharLength, true, code.VarcharLengthExceedsLimit, func(_ advisor.V1Stmt[ast.Node], _ v1review.Finding, maximum int) string {
			return fmt.Sprintf("The maximum varchar length is %d.", maximum)
		}))
	advisor.Register(storepb.Engine_ORACLE, storepb.SQLReviewRule_COLUMN_NO_NULL,
		e.SimpleBatch(
			func(checkCtx advisor.Context, nodes []ast.Node) []v1review.Finding {
				return oraclev1review.ColumnNoNull(nodes, checkCtx.CurrentDatabase)
			},
			code.ColumnCannotNull,
			"",
			func(f v1review.Finding) string {
				return fmt.Sprintf("Column %q is nullable, which is not allowed.", f.Name)
			},
		))
	advisor.Register(storepb.Engine_ORACLE, storepb.SQLReviewRule_COLUMN_REQUIRED, oracleColumnRequired(e))
	advisor.Register(storepb.Engine_ORACLE, storepb.SQLReviewRule_COLUMN_REQUIRE_DEFAULT,
		e.SimpleBatch(
			func(checkCtx advisor.Context, nodes []ast.Node) []v1review.Finding {
				return oraclev1review.ColumnRequireDefault(nodes, checkCtx.CurrentDatabase)
			},
			code.NoDefault,
			"",
			func(f v1review.Finding) string {
				return fmt.Sprintf("Column %q doesn't have default value", f.Name)
			},
		))
	advisor.Register(storepb.Engine_ORACLE, storepb.SQLReviewRule_COLUMN_TYPE_DISALLOW_LIST,
		e.StringArrayStmt(oraclev1review.ColumnTypeDisallowList, "string_array_payload is required for column type disallow list rule", code.DisabledColumnType, "", func(_ advisor.V1Stmt[ast.Node], f v1review.Finding) string {
			return fmt.Sprintf("Disallow column type %s but column \"%s\" is", f.Value, f.Name)
		}))
	advisor.Register(storepb.Engine_ORACLE, storepb.SQLReviewRule_INDEX_KEY_NUMBER_LIMIT,
		e.NumberStmt(oraclev1review.IndexKeyNumberLimit, true, code.IndexKeyNumberExceedsLimit, func(_ advisor.V1Stmt[ast.Node], _ v1review.Finding, limit int) string {
			return fmt.Sprintf("Index key number should be less than or equal to %d", limit)
		}))
	advisor.Register(storepb.Engine_ORACLE, storepb.SQLReviewRule_STATEMENT_INSERT_MUST_SPECIFY_COLUMN,
		e.SimpleStmt(oraclev1review.StatementInsertMustSpecifyColumn, code.InsertNotSpecifyColumn, func(advisor.V1Stmt[ast.Node], v1review.Finding) string {
			return "INSERT statement should specify column name."
		}))
	advisor.Register(storepb.Engine_ORACLE, storepb.SQLReviewRule_NAMING_IDENTIFIER_CASE, oracleNamingIdentifierCase(e))
	advisor.Register(storepb.Engine_ORACLE, storepb.SQLReviewRule_NAMING_IDENTIFIER_NO_KEYWORD,
		e.SimpleStmt(oraclev1review.NamingIdentifierNoKeyword, code.NameIsKeywordIdentifier, func(_ advisor.V1Stmt[ast.Node], f v1review.Finding) string {
			return fmt.Sprintf("Identifier %q is a keyword and should be avoided", f.Name)
		}))
	advisor.Register(storepb.Engine_ORACLE, storepb.SQLReviewRule_NAMING_TABLE,
		e.NamingRegex(
			oraclev1review.NamingTable,
			code.NamingTableConventionMismatch,
			"naming_payload is required for this rule",
			func(f v1review.Finding, format string) string {
				return fmt.Sprintf(`"%s" mismatches table naming convention, naming format should be %q`, f.Name, format)
			},
			func(f v1review.Finding, maxLength int) string {
				return fmt.Sprintf("\"%s\" mismatches table naming convention, its length should be within %d characters", f.Name, maxLength)
			},
		))
	advisor.Register(storepb.Engine_ORACLE, storepb.SQLReviewRule_NAMING_TABLE_NO_KEYWORD,
		e.SimpleStmt(oraclev1review.NamingTableNoKeyword, code.NameIsKeywordIdentifier, func(_ advisor.V1Stmt[ast.Node], f v1review.Finding) string {
			return fmt.Sprintf("Table name %q is a keyword identifier and should be avoided.", f.Name)
		}))
	advisor.Register(storepb.Engine_ORACLE, storepb.SQLReviewRule_STATEMENT_SELECT_NO_SELECT_ALL,
		e.SimpleStmt(oraclev1review.StatementSelectNoSelectAll, code.StatementSelectAll, func(advisor.V1Stmt[ast.Node], v1review.Finding) string {
			return "Avoid using SELECT *."
		}))
	advisor.Register(storepb.Engine_ORACLE, storepb.SQLReviewRule_STATEMENT_DISALLOW_TRUNCATE,
		e.Stmt(
			func(stmt advisor.V1Stmt[ast.Node]) []v1review.Finding {
				return oraclev1review.StatementDisallowTruncate([]ast.Node{stmt.Node}, stmt.Text)
			},
			func(_ advisor.V1Stmt[ast.Node], f v1review.Finding) (code.Code, string, bool) {
				switch f.Code {
				case v1review.Truncate:
					return code.StatementDisallowTruncate, fmt.Sprintf(
						`TRUNCATE TABLE %q is not allowed: it issues an implicit COMMIT and cannot be rolled back. Any prior uncommitted work in the same transaction is also committed. Prior-backup treats this as DDL and does not produce row-level snapshots.`,
						f.Table,
					), true
				case v1review.TruncatePartition:
					return code.StatementDisallowTruncate, fmt.Sprintf(
						`ALTER TABLE %q TRUNCATE %s %q is not allowed: partition truncate shares the implicit-commit gap of TRUNCATE TABLE on Oracle. Prior-backup treats this as DDL and does not produce row-level snapshots.`,
						f.Table, f.Value, f.Name,
					), true
				default:
					return 0, "", false
				}
			},
		))
	advisor.Register(storepb.Engine_ORACLE, storepb.SQLReviewRule_STATEMENT_DML_DRY_RUN, oracleDMLDryRun(e))
	advisor.Register(storepb.Engine_ORACLE, storepb.SQLReviewRule_STATEMENT_WHERE_DISALLOW_FUNCTIONS_AND_CALCULATIONS,
		e.WhereDisallowFuncAndCalc(func(checkCtx advisor.Context) func([]ast.Node) []v1review.Finding {
			// Oracle sync stores only the connected owner's objects, under the anonymous schema "".
			schema := e.SchemaOf(checkCtx)
			currentDB := checkCtx.CurrentDatabase
			return func(nodes []ast.Node) []v1review.Finding {
				return oraclev1review.StatementWhereDisallowFunctionsAndCalculations(nodes, schema, "", currentDB)
			}
		}, true))
	advisor.Register(storepb.Engine_ORACLE, storepb.SQLReviewRule_TABLE_COMMENT,
		e.TableComment(
			func(checkCtx advisor.Context, nodes []ast.Node, cfg *v1review.TableCommentConfig) []v1review.Finding {
				return oraclev1review.TableComment(nodes, checkCtx.CurrentDatabase, cfg)
			},
			func(f v1review.Finding) string {
				return normalizeIdentifierName(f.Name)
			},
		))
	advisor.Register(storepb.Engine_ORACLE, storepb.SQLReviewRule_TABLE_NO_FOREIGN_KEY,
		e.SimpleBatch(
			func(checkCtx advisor.Context, nodes []ast.Node) []v1review.Finding {
				return oraclev1review.TableNoForeignKey(nodes, checkCtx.CurrentDatabase)
			},
			code.TableHasFK,
			v1review.TableForeignKey,
			func(f v1review.Finding) string {
				return fmt.Sprintf("Foreign key is not allowed in the table %s.", normalizeIdentifierName(f.Name))
			},
		))
	advisor.Register(storepb.Engine_ORACLE, storepb.SQLReviewRule_TABLE_REQUIRE_PK,
		e.SimpleBatch(
			func(checkCtx advisor.Context, nodes []ast.Node) []v1review.Finding {
				return oraclev1review.TableRequirePK(nodes, checkCtx.CurrentDatabase)
			},
			code.TableNoPK,
			v1review.TablePrimaryKeyMissing,
			func(f v1review.Finding) string {
				return fmt.Sprintf("Table %s requires PRIMARY KEY.", normalizeIdentifierName(f.Name))
			},
		))
	advisor.Register(storepb.Engine_ORACLE, storepb.SQLReviewRule_STATEMENT_WHERE_NO_LEADING_WILDCARD_LIKE,
		e.SimpleStmt(oraclev1review.StatementWhereNoLeadingWildcardLike, code.StatementLeadingWildcardLike, func(advisor.V1Stmt[ast.Node], v1review.Finding) string {
			return "Avoid using leading wildcard LIKE."
		}))
	advisor.Register(storepb.Engine_ORACLE, storepb.SQLReviewRule_STATEMENT_WHERE_REQUIRE_SELECT,
		e.SimpleStmt(oraclev1review.StatementWhereRequireSelect, code.StatementNoWhere, func(advisor.V1Stmt[ast.Node], v1review.Finding) string {
			return "WHERE clause is required for SELECT statement."
		}))
	advisor.Register(storepb.Engine_ORACLE, storepb.SQLReviewRule_STATEMENT_WHERE_REQUIRE_UPDATE_DELETE,
		e.SimpleStmt(oraclev1review.StatementWhereRequireUpdateDelete, code.StatementNoWhere, func(_ advisor.V1Stmt[ast.Node], f v1review.Finding) string {
			return fmt.Sprintf("WHERE clause is required for %s statement.", f.Name)
		}))
}

// normalizeIdentifierName normalizes the identifier name to the format of "schema"."table"."column".
func normalizeIdentifierName(name string) string {
	list := strings.Split(name, ".")
	result := make([]string, len(list))
	for i, item := range list {
		result[i] = fmt.Sprintf("%q", item)
	}
	return strings.Join(result, ".")
}

func oracleColumnRequired(e *advisor.V1Engine[ast.Node]) advisor.Func {
	return func(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
		level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
		if err != nil {
			return nil, err
		}
		list, err := advisor.RequireStringArrayPayload(checkCtx.Rule, "string_array_payload is required for column required rule")
		if err != nil {
			return nil, err
		}
		stmts, nodes, err := e.Nodes(checkCtx)
		if err != nil {
			return nil, err
		}
		title := checkCtx.Rule.Type.String()
		var adviceList []*storepb.Advice
		for i, stmt := range stmts {
			findings := oraclev1review.ColumnRequired(nodes[i:i+1:i+1], list)
			if len(findings) == 0 {
				continue
			}
			missingColumns := make([]string, len(findings))
			for j, f := range findings {
				missingColumns[j] = fmt.Sprintf("%q", f.Name)
			}
			adviceList = append(adviceList, &storepb.Advice{
				Status:        level,
				Code:          code.NoRequiredColumn.Int32(),
				Title:         title,
				Content:       fmt.Sprintf("Table %q requires columns: %s", findings[0].Table, strings.Join(missingColumns, ", ")),
				StartPosition: e.PosFindingOrFirstLine(stmt, findings[0]),
			})
		}
		return adviceList, nil
	}
}

func oracleNamingIdentifierCase(e *advisor.V1Engine[ast.Node]) advisor.Func {
	return func(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
		level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
		if err != nil {
			return nil, err
		}
		namingCasePayload := checkCtx.Rule.GetNamingCasePayload()
		if namingCasePayload == nil {
			return nil, errors.New("naming_case_payload is required for this rule")
		}
		cfg := v1review.CaseConfig{Upper: namingCasePayload.Upper}
		stmts, nodes, err := e.Nodes(checkCtx)
		if err != nil {
			return nil, err
		}
		title := checkCtx.Rule.Type.String()
		var adviceList []*storepb.Advice
		for i, stmt := range stmts {
			for _, f := range oraclev1review.NamingIdentifierCase(nodes[i:i+1:i+1], cfg) {
				content := fmt.Sprintf("Identifier %q should be lower case", f.Name)
				if cfg.Upper {
					content = fmt.Sprintf("Identifier %q should be upper case", f.Name)
				}
				adviceList = append(adviceList, &storepb.Advice{
					Status:        level,
					Code:          code.NamingCaseMismatch.Int32(),
					Title:         title,
					Content:       content,
					StartPosition: e.PosFindingOrFirstLine(stmt, f),
				})
			}
		}
		return adviceList, nil
	}
}

func oracleDMLDryRun(e *advisor.V1Engine[ast.Node]) advisor.Func {
	return func(ctx context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
		level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
		if err != nil {
			return nil, err
		}
		// BYT-8855: Skip DML dry run if there are DDL statements mixed in, because DML
		// statements often reference objects created by DDL statements, causing false positives.
		if advisor.ContainsDDL(checkCtx.DBType, checkCtx.ParsedStatements) {
			return nil, nil
		}
		prober, err := advisor.ProbeFor(checkCtx)
		if err != nil {
			return nil, err
		}
		stmts, nodes, err := e.Nodes(checkCtx)
		if err != nil {
			return nil, err
		}
		probes := make(map[int]v1review.Probe)
		for n, i := range oraclev1review.StatementDMLDryRunProbes(nodes) {
			if n >= common.MaximumLintExplainSize {
				break
			}
			// BEGIN opens a PL/SQL block in Oracle, so the dry run runs alone.
			if _, err := prober.Probe(ctx, db.ProbeContext{Autocommit: true}, fmt.Sprintf("EXPLAIN PLAN FOR %s", stmts[i].Text)); err != nil {
				probes[i] = v1review.Probe{Error: err.Error()}
			} else {
				probes[i] = v1review.Probe{}
			}
		}
		title := checkCtx.Rule.Type.String()
		var adviceList []*storepb.Advice
		for _, f := range oraclev1review.StatementDMLDryRun(nodes, probes) {
			line := stmts[f.Statement].BaseLine + 1
			adviceList = append(adviceList, &storepb.Advice{
				Status:        level,
				Code:          code.StatementDMLDryRunFailed.Int32(),
				Title:         title,
				Content:       fmt.Sprintf("Failed to dry run statement at line %d: %v", line, f.Value),
				StartPosition: base.ConvertANTLRLineToPosition(line),
			})
		}
		return adviceList, nil
	}
}

func oraclePriorBackup(e *advisor.V1Engine[ast.Node]) advisor.Func {
	return func(ctx context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
		if !checkCtx.EnablePriorBackup {
			return nil, nil
		}
		_, nodes, level, title, err := e.Prepare(checkCtx)
		if err != nil {
			return nil, err
		}
		databaseName := common.BackupDatabaseNameOfEngine(storepb.Engine_ORACLE)
		facts := v1review.PriorBackupFacts{
			TotalSize:      checkCtx.StatementsTotalSize,
			MaxSize:        common.MaxSheetCheckSize,
			Database:       databaseName,
			DatabaseExists: advisor.DatabaseExists(ctx, checkCtx, databaseName),
		}
		var adviceList []*storepb.Advice
		for _, f := range oraclev1review.PriorBackupCheck(nodes, e.SchemaOf(checkCtx), facts) {
			var adviceCode code.Code
			var content string
			switch f.Code {
			case v1review.PriorBackupScriptTooLarge:
				adviceCode, content = code.BuiltinPriorBackupCheck, fmt.Sprintf("The size of the SQL statements exceeds the maximum limit of %s bytes for backup", f.Value)
			case v1review.PriorBackupDatabaseMissing:
				adviceCode, content = code.DatabaseNotExists, fmt.Sprintf("Need database %q to do prior backup but it does not exist", f.Name)
			case v1review.PriorBackupDDLInBatch:
				adviceCode, content = code.BuiltinPriorBackupCheck, "Prior backup cannot deal with mixed DDL and DML statements"
			case v1review.PriorBackupMixedDMLOnTable:
				adviceCode, content = code.BuiltinPriorBackupCheck, fmt.Sprintf("Prior backup cannot handle mixed DML statements on the same table %q", f.Table)
			case v1review.PriorBackupUncopyableColumn:
				adviceCode, content = code.BuiltinPriorBackupCheck, fmt.Sprintf("Prior backup cannot copy table %q.%q: column(s) %s use the LONG/LONG RAW type, which CREATE TABLE AS SELECT does not support (ORA-00997). Disable prior backup for this issue or migrate the column type", f.Schema, f.Table, f.Value)
			default:
				continue
			}
			adviceList = append(adviceList, &storepb.Advice{
				Status:  level,
				Title:   title,
				Content: content,
				Code:    adviceCode.Int32(),
			})
		}
		return adviceList, nil
	}
}
