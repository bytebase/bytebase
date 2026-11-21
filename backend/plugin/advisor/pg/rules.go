// Package pg implements the SQL advisor rules for PostgreSQL.
package pg

import (
	"cmp"
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/bytebase/omni/pg/ast"
	pgv1review "github.com/bytebase/omni/pg/v1review"
	"github.com/bytebase/omni/v1review"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	pgparser "github.com/bytebase/bytebase/backend/plugin/parser/pg"
	"github.com/bytebase/bytebase/backend/store/model"
)

func init() {
	extract := advisor.NodeExtractor(pgparser.GetNode)
	byteOffsetLine := func(sql string, offset int) int32 {
		return pgparser.ByteOffsetToRunePosition(sql, offset).Line
	}
	e := advisor.NewV1Engine(extract, byteOffsetLine, advisor.V1PosContentStart)

	compatEntries := map[v1review.Code]advisor.V1CompatEntry{
		v1review.IncompatibleDropDatabase:  {Code: code.CompatibilityDropDatabase},
		v1review.IncompatibleDropTable:     {Code: code.CompatibilityDropTable},
		v1review.IncompatibleRenameTable:   {Code: code.CompatibilityRenameTable},
		v1review.IncompatibleRenameColumn:  {Code: code.CompatibilityRenameColumn},
		v1review.IncompatibleDropColumn:    {Code: code.CompatibilityDropColumn},
		v1review.IncompatibleAlterColumn:   {Code: code.CompatibilityAlterColumn},
		v1review.IncompatibleAddPrimaryKey: {Code: code.CompatibilityAddPrimaryKey},
		v1review.IncompatibleAddUniqueKey:  {Code: code.CompatibilityAddUniqueKey},
		v1review.IncompatibleAddForeignKey: {Code: code.CompatibilityAddForeignKey},
		v1review.IncompatibleAddCheck:      {Code: code.CompatibilityAddCheck},
	}

	advisor.Register(storepb.Engine_POSTGRES, storepb.SQLReviewRule_BUILTIN_WALK_THROUGH_CHECK, &advisor.BuiltinWalkThroughCheckAdvisor{})
	advisor.Register(storepb.Engine_POSTGRES, storepb.SQLReviewRule_BUILTIN_PRIOR_BACKUP_CHECK, pgPriorBackup(e))
	advisor.Register(storepb.Engine_POSTGRES, storepb.SQLReviewRule_STATEMENT_INSERT_ROW_LIMIT, pgInsertRowLimit(e))
	advisor.Register(storepb.Engine_POSTGRES, storepb.SQLReviewRule_STATEMENT_AFFECTED_ROW_LIMIT, pgAffectedRowLimit(e))
	advisor.Register(storepb.Engine_POSTGRES, storepb.SQLReviewRule_STATEMENT_DML_DRY_RUN, pgDMLDryRun(e))
	advisor.Register(storepb.Engine_POSTGRES, storepb.SQLReviewRule_STATEMENT_OBJECT_OWNER_CHECK, pgObjectOwnerCheck(e))

	advisor.Register(storepb.Engine_POSTGRES, storepb.SQLReviewRule_SYSTEM_CHARSET_ALLOWLIST,
		e.StringArrayStmt(pgv1review.SystemCharsetAllowlist, "", code.DisabledCharset, "", func(_ advisor.V1Stmt[ast.Node], f v1review.Finding) string {
			return fmt.Sprintf("\"\" used disabled encoding '%s'", f.Name)
		}, func(stmt advisor.V1Stmt[ast.Node], _ v1review.Finding) *storepb.Position {
			return &storepb.Position{Line: int32(stmt.BaseLine)}
		}))
	advisor.Register(storepb.Engine_POSTGRES, storepb.SQLReviewRule_SYSTEM_COLLATION_ALLOWLIST,
		e.StringArrayStmt(pgv1review.SystemCollationAllowlist, "", code.DisabledCollation, "", func(stmt advisor.V1Stmt[ast.Node], f v1review.Finding) string {
			return fmt.Sprintf("Use disabled collation \"%s\", related statement \"%s\"", f.Name, strings.TrimSpace(stmt.Text))
		}))
	advisor.Register(storepb.Engine_POSTGRES, storepb.SQLReviewRule_COLUMN_COMMENT,
		e.ColumnComment(
			func(_ advisor.Context, nodes []ast.Node, required bool, maxLength int) []v1review.Finding {
				return pgv1review.ColumnComment(nodes, required, maxLength)
			},
			0,
			false,
			func(f v1review.Finding) string {
				return fmt.Sprintf("`%s.%s`", f.Table, f.Name)
			},
			e.PosSearchOrStart,
		))
	advisor.Register(storepb.Engine_POSTGRES, storepb.SQLReviewRule_COLUMN_DEFAULT_DISALLOW_VOLATILE,
		e.SimpleStmt(pgv1review.ColumnDefaultDisallowVolatile, code.NoDefault, func(_ advisor.V1Stmt[ast.Node], f v1review.Finding) string {
			return fmt.Sprintf("Column %q.%q in schema %q has volatile DEFAULT", f.Table, f.Name, "public")
		}, e.PosSearchOrStart))
	advisor.Register(storepb.Engine_POSTGRES, storepb.SQLReviewRule_COLUMN_DISALLOW_CHANGE_TYPE,
		e.SimpleStmt(pgv1review.ColumnDisallowChangeType, code.ChangeColumnType, func(stmt advisor.V1Stmt[ast.Node], _ v1review.Finding) string {
			return fmt.Sprintf("The statement \"%s\" changes column type", strings.TrimSpace(e.TrimmedText(stmt.Text)))
		}, e.PosSearchOrStart))
	advisor.Register(storepb.Engine_POSTGRES, storepb.SQLReviewRule_COLUMN_MAXIMUM_CHARACTER_LENGTH,
		e.NumberStmt(pgv1review.ColumnMaximumCharacterLength, true, code.CharLengthExceedsLimit, func(_ advisor.V1Stmt[ast.Node], f v1review.Finding, maximum int) string {
			tableName := fmt.Sprintf("%q", f.Table)
			if f.Value != "" {
				tableName = fmt.Sprintf("%q.%q", f.Value, f.Table)
			}
			return fmt.Sprintf("The length of the CHAR column %q in table %s is bigger than %d, please use VARCHAR instead", f.Name, tableName, maximum)
		}, e.PosSearchOrStart))
	advisor.Register(storepb.Engine_POSTGRES, storepb.SQLReviewRule_COLUMN_NO_NULL, pgColumnNoNull(e))
	advisor.Register(storepb.Engine_POSTGRES, storepb.SQLReviewRule_COLUMN_REQUIRE_DEFAULT,
		e.SimpleStmt(pgv1review.ColumnRequireDefault, code.NoDefault, func(_ advisor.V1Stmt[ast.Node], f v1review.Finding) string {
			return fmt.Sprintf("Column %q.%q in schema %q doesn't have DEFAULT", f.Table, f.Name, "public")
		}, e.PosSearchOrStart))
	advisor.Register(storepb.Engine_POSTGRES, storepb.SQLReviewRule_COLUMN_REQUIRED, pgColumnRequired(e))
	advisor.Register(storepb.Engine_POSTGRES, storepb.SQLReviewRule_COLUMN_TYPE_DISALLOW_LIST,
		e.StringArrayStmt(pgv1review.ColumnTypeDisallowList, "", code.DisabledColumnType, "", func(_ advisor.V1Stmt[ast.Node], f v1review.Finding) string {
			return fmt.Sprintf("Disallow column type %s but column %q.%q is", strings.ToUpper(f.Value), f.Table, f.Name)
		}, e.PosSearchOrStart))
	advisor.Register(storepb.Engine_POSTGRES, storepb.SQLReviewRule_SYSTEM_COMMENT_LENGTH,
		e.NumberStmt(pgv1review.SystemCommentLength, false, code.CommentTooLong, func(_ advisor.V1Stmt[ast.Node], _ v1review.Finding, maximum int) string {
			return fmt.Sprintf("The length of comment should be within %d characters", maximum)
		}))
	advisor.Register(storepb.Engine_POSTGRES, storepb.SQLReviewRule_INDEX_CREATE_CONCURRENTLY, pgIndexCreateConcurrently(e))
	advisor.Register(storepb.Engine_POSTGRES, storepb.SQLReviewRule_INDEX_KEY_NUMBER_LIMIT,
		e.NumberStmt(pgv1review.IndexKeyNumberLimit, false, code.IndexKeyNumberExceedsLimit, func(_ advisor.V1Stmt[ast.Node], f v1review.Finding, limit int) string {
			return fmt.Sprintf("The number of keys of index %q in table %q should be not greater than %d", f.Name, f.Table, limit)
		}, e.PosSearchOrStart))
	advisor.Register(storepb.Engine_POSTGRES, storepb.SQLReviewRule_INDEX_NO_DUPLICATE_COLUMN,
		e.SimpleStmt(pgv1review.IndexNoDuplicateColumn, code.DuplicateColumnInIndex, func(_ advisor.V1Stmt[ast.Node], f v1review.Finding) string {
			return fmt.Sprintf("%s %q has duplicate column %q.%q", f.Value, f.Name, f.Table, f.Other)
		}, e.PosSearchOrStart))
	advisor.Register(storepb.Engine_POSTGRES, storepb.SQLReviewRule_INDEX_PRIMARY_KEY_TYPE_ALLOWLIST,
		e.StringArrayStmt(pgv1review.IndexPrimaryKeyTypeAllowlist, "", code.IndexPKType, "", func(_ advisor.V1Stmt[ast.Node], f v1review.Finding) string {
			return fmt.Sprintf("The column %q is one of the primary key, but its type %q is not in allowlist", f.Name, f.Value)
		}, e.PosSearchOrStart))
	advisor.Register(storepb.Engine_POSTGRES, storepb.SQLReviewRule_INDEX_TOTAL_NUMBER_LIMIT, pgIndexTotalNumberLimit(e))
	advisor.Register(storepb.Engine_POSTGRES, storepb.SQLReviewRule_STATEMENT_INSERT_DISALLOW_ORDER_BY_RAND,
		e.SimpleStmt(pgv1review.StatementInsertDisallowOrderByRand, code.InsertUseOrderByRand, func(stmt advisor.V1Stmt[ast.Node], _ v1review.Finding) string {
			return fmt.Sprintf("The INSERT statement uses ORDER BY random() or random_between(), related statement \"%s\"", e.TrimmedText(stmt.Text))
		}))
	advisor.Register(storepb.Engine_POSTGRES, storepb.SQLReviewRule_STATEMENT_INSERT_MUST_SPECIFY_COLUMN,
		e.SimpleStmt(pgv1review.StatementInsertMustSpecifyColumn, code.InsertNotSpecifyColumn, func(stmt advisor.V1Stmt[ast.Node], _ v1review.Finding) string {
			return fmt.Sprintf("The INSERT statement must specify columns but \"%s\" does not", e.TrimmedText(stmt.Text))
		}))
	advisor.Register(storepb.Engine_POSTGRES, storepb.SQLReviewRule_SCHEMA_BACKWARD_COMPATIBILITY,
		e.BackwardCompatibility(
			func(_ advisor.Context, nodes []ast.Node) []v1review.Finding {
				return pgv1review.SchemaBackwardCompatibility(nodes)
			},
			compatEntries,
			func(stmt advisor.V1Stmt[ast.Node], _ v1review.Finding, _ string) string {
				return fmt.Sprintf(`"%s" may cause incompatibility with the existing data and code`, e.TrimmedText(stmt.Text))
			},
		))
	advisor.Register(storepb.Engine_POSTGRES, storepb.SQLReviewRule_NAMING_COLUMN,
		e.NamingRegex(
			pgv1review.NamingColumn,
			code.NamingColumnConventionMismatch,
			"naming_payload is required for this rule",
			func(f v1review.Finding, format string) string {
				return fmt.Sprintf("\"%s\".\"%s\" mismatches column naming convention, naming format should be %q", f.Table, f.Name, format)
			},
			func(f v1review.Finding, maxLength int) string {
				return fmt.Sprintf("\"%s\".\"%s\" mismatches column naming convention, its length should be within %d characters", f.Table, f.Name, maxLength)
			},
			e.PosNameOrStart,
		))
	advisor.Register(storepb.Engine_POSTGRES, storepb.SQLReviewRule_NAMING_INDEX_FK,
		e.TemplateNaming(
			func(nodes []ast.Node, cfg v1review.TemplateConfig, _ v1review.Schema) ([]v1review.Finding, error) {
				return pgv1review.NamingForeignKey(nodes, cfg)
			},
			code.NamingFKConventionMismatch,
			"Internal error for foreign key naming convention rule",
			"Foreign key",
			"Failed to compile regex: %s",
			false,
			e.PosNameOrStart,
		))
	advisor.Register(storepb.Engine_POSTGRES, storepb.SQLReviewRule_NAMING_FULLY_QUALIFIED, pgNamingFullyQualified(e))
	advisor.Register(storepb.Engine_POSTGRES, storepb.SQLReviewRule_NAMING_INDEX_IDX,
		e.TemplateNaming(
			func(nodes []ast.Node, cfg v1review.TemplateConfig, schema v1review.Schema) ([]v1review.Finding, error) {
				return pgv1review.NamingIndex(nodes, cfg, schema), nil
			},
			code.NamingIndexConventionMismatch,
			"Internal error for index naming convention rule",
			"Index",
			"Failed to compile regex: %s",
			false,
		))
	advisor.Register(storepb.Engine_POSTGRES, storepb.SQLReviewRule_NAMING_INDEX_PK,
		e.TemplateNaming(
			func(nodes []ast.Node, cfg v1review.TemplateConfig, schema v1review.Schema) ([]v1review.Finding, error) {
				return pgv1review.NamingPrimaryKey(nodes, cfg, schema), nil
			},
			code.NamingPKConventionMismatch,
			"Internal error for primary key naming convention rule",
			"Primary key",
			"Failed to compile regex: %s",
			false,
			e.PosNameOrStart,
		))
	advisor.Register(storepb.Engine_POSTGRES, storepb.SQLReviewRule_NAMING_TABLE,
		e.NamingRegex(
			pgv1review.NamingTable,
			code.NamingTableConventionMismatch,
			"naming_payload is required for naming table rule",
			func(f v1review.Finding, format string) string {
				return fmt.Sprintf(`"%s" mismatches table naming convention, naming format should be %q`, f.Name, format)
			},
			func(f v1review.Finding, maxLength int) string {
				return fmt.Sprintf("\"%s\" mismatches table naming convention, its length should be within %d characters", f.Name, maxLength)
			},
		))
	advisor.Register(storepb.Engine_POSTGRES, storepb.SQLReviewRule_NAMING_INDEX_UK,
		e.TemplateNaming(
			func(nodes []ast.Node, cfg v1review.TemplateConfig, schema v1review.Schema) ([]v1review.Finding, error) {
				return pgv1review.NamingUniqueKey(nodes, cfg, schema), nil
			},
			code.NamingUKConventionMismatch,
			"Internal error for unique key naming convention rule",
			"Unique key",
			"Failed to compile regex for unique key naming convention: %s",
			true,
			e.PosNameOrStart,
		))
	advisor.Register(storepb.Engine_POSTGRES, storepb.SQLReviewRule_STATEMENT_ADD_CHECK_NOT_VALID,
		e.SimpleStmt(pgv1review.StatementAddCheckNotValid, code.StatementAddCheckWithValidation, func(advisor.V1Stmt[ast.Node], v1review.Finding) string {
			return "Adding check constraints with validation will block reads and writes. You can add check constraints not valid and then validate separately"
		}))
	advisor.Register(storepb.Engine_POSTGRES, storepb.SQLReviewRule_STATEMENT_ADD_FOREIGN_KEY_NOT_VALID,
		e.SimpleStmt(pgv1review.StatementAddForeignKeyNotValid, code.StatementAddFKWithValidation, func(advisor.V1Stmt[ast.Node], v1review.Finding) string {
			return "Adding foreign keys with validation will block reads and writes. You can add check foreign keys not valid and then validate separately"
		}))
	advisor.Register(storepb.Engine_POSTGRES, storepb.SQLReviewRule_STATEMENT_CHECK_SET_ROLE_VARIABLE,
		e.SimpleBatch(
			func(_ advisor.Context, nodes []ast.Node) []v1review.Finding {
				return pgv1review.StatementCheckSetRoleVariable(nodes)
			},
			code.StatementCheckSetRoleVariable,
			"",
			func(v1review.Finding) string {
				return "No SET ROLE statement found."
			},
			e.PosNil,
		))
	advisor.Register(storepb.Engine_POSTGRES, storepb.SQLReviewRule_STATEMENT_CREATE_SPECIFY_SCHEMA,
		e.SimpleStmt(pgv1review.StatementCreateSpecifySchema, code.StatementCreateWithoutSchemaName, func(_ advisor.V1Stmt[ast.Node], f v1review.Finding) string {
			return f.Name + " schema should be specified."
		}))
	advisor.Register(storepb.Engine_POSTGRES, storepb.SQLReviewRule_STATEMENT_DISALLOW_ADD_COLUMN_WITH_DEFAULT,
		e.SimpleStmt(pgv1review.StatementDisallowAddColumnWithDefault, code.StatementAddColumnWithDefault, func(advisor.V1Stmt[ast.Node], v1review.Finding) string {
			return "Adding column with DEFAULT will locked the whole table and rewriting each rows"
		}))
	advisor.Register(storepb.Engine_POSTGRES, storepb.SQLReviewRule_STATEMENT_DISALLOW_ADD_NOT_NULL,
		e.SimpleStmt(pgv1review.StatementDisallowAddNotNull, code.StatementAddNotNull, func(_ advisor.V1Stmt[ast.Node], f v1review.Finding) string {
			return fmt.Sprintf("Setting NOT NULL will block reads and writes. You can use CHECK (%q IS NOT NULL) instead", f.Name)
		}))
	advisor.Register(storepb.Engine_POSTGRES, storepb.SQLReviewRule_STATEMENT_DISALLOW_COMMIT,
		e.SimpleStmt(pgv1review.StatementDisallowCommit, code.StatementDisallowCommit, func(stmt advisor.V1Stmt[ast.Node], _ v1review.Finding) string {
			stmtText := e.TrimmedText(stmt.Text)
			if stmtText == "" {
				stmtText = "<unknown statement>"
			}
			return fmt.Sprintf("Commit is not allowed, related statement: \"%s\"", stmtText)
		}))
	advisor.Register(storepb.Engine_POSTGRES, storepb.SQLReviewRule_STATEMENT_DISALLOW_ON_DEL_CASCADE,
		e.SimpleStmt(pgv1review.StatementDisallowOnDelCascade, code.StatementDisallowCascade, func(advisor.V1Stmt[ast.Node], v1review.Finding) string {
			return "The CASCADE option is not permitted for ON DELETE clauses"
		}))
	advisor.Register(storepb.Engine_POSTGRES, storepb.SQLReviewRule_STATEMENT_DISALLOW_RM_TBL_CASCADE,
		e.SimpleStmt(pgv1review.StatementDisallowRmTblCascade, code.StatementDisallowCascade, func(advisor.V1Stmt[ast.Node], v1review.Finding) string {
			return "The use of CASCADE is not permitted when removing a table"
		}))
	advisor.Register(storepb.Engine_POSTGRES, storepb.SQLReviewRule_STATEMENT_DISALLOW_TRUNCATE,
		e.SimpleStmt(pgv1review.StatementDisallowTruncate, code.StatementDisallowTruncate, func(_ advisor.V1Stmt[ast.Node], f v1review.Finding) string {
			suffix := ""
			for _, option := range strings.Split(f.Value, ",") {
				switch option {
				case "CASCADE":
					suffix += " CASCADE will wipe dependent tables."
				case "RESTART IDENTITY":
					suffix += " RESTART IDENTITY will reset owned sequences."
				default:
				}
			}
			return fmt.Sprintf(
				`TRUNCATE TABLE %q is not allowed: TRUNCATE bypasses triggers. Unlike Oracle/MySQL/MSSQL, PG TRUNCATE is transactional and rolls back cleanly.%s`,
				f.Table, suffix,
			)
		}, e.PosFinding))
	advisor.Register(storepb.Engine_POSTGRES, storepb.SQLReviewRule_STATEMENT_MERGE_ALTER_TABLE, pgMergeAlterTable(e))
	advisor.Register(storepb.Engine_POSTGRES, storepb.SQLReviewRule_STATEMENT_WHERE_NO_LEADING_WILDCARD_LIKE,
		e.SimpleStmt(pgv1review.StatementWhereNoLeadingWildcardLike, code.StatementLeadingWildcardLike, func(stmt advisor.V1Stmt[ast.Node], _ v1review.Finding) string {
			return fmt.Sprintf("\"%s\" uses leading wildcard LIKE", e.TrimmedText(stmt.Text))
		}))
	advisor.Register(storepb.Engine_POSTGRES, storepb.SQLReviewRule_STATEMENT_SELECT_NO_SELECT_ALL,
		e.SimpleStmt(pgv1review.StatementSelectNoSelectAll, code.StatementSelectAll, func(stmt advisor.V1Stmt[ast.Node], f v1review.Finding) string {
			text := e.TrimmedText(stmt.Text)
			if f.End > f.Start && f.End <= len(stmt.Text) {
				text = strings.TrimSpace(stmt.Text[f.Start:f.End])
			}
			return fmt.Sprintf("\"%s\" uses SELECT all", text)
		}))
	advisor.Register(storepb.Engine_POSTGRES, storepb.SQLReviewRule_STATEMENT_NON_TRANSACTIONAL,
		e.Stmt(
			func(stmt advisor.V1Stmt[ast.Node]) []v1review.Finding {
				return pgv1review.StatementNonTransactional([]ast.Node{stmt.Node}, stmt.Text)
			},
			func(_ advisor.V1Stmt[ast.Node], _ v1review.Finding) (code.Code, string, bool) {
				return code.StatementNonTransactional, "This statement is non-transactional", true
			},
		))
	advisor.Register(storepb.Engine_POSTGRES, storepb.SQLReviewRule_STATEMENT_WHERE_DISALLOW_FUNCTIONS_AND_CALCULATIONS,
		e.WhereDisallowFuncAndCalc(func(checkCtx advisor.Context) func([]ast.Node) []v1review.Finding {
			schema := e.SchemaOf(checkCtx)
			var searchPath []string
			if checkCtx.DBSchema != nil {
				searchPath = model.NewDatabaseMetadata(checkCtx.DBSchema, nil, nil,
					storepb.Engine_POSTGRES, checkCtx.IsObjectCaseSensitive).GetSearchPathForCurrentUser(checkCtx.SessionUser)
			}
			if len(searchPath) == 0 {
				searchPath = []string{"public"}
			}
			return func(nodes []ast.Node) []v1review.Finding {
				return pgv1review.StatementWhereDisallowFunctionsAndCalculations(nodes, schema, searchPath)
			}
		}, false, e.PosFinding))
	advisor.Register(storepb.Engine_POSTGRES, storepb.SQLReviewRule_STATEMENT_WHERE_REQUIRE_SELECT,
		e.SimpleStmt(pgv1review.StatementWhereRequireSelect, code.StatementNoWhere, func(stmt advisor.V1Stmt[ast.Node], f v1review.Finding) string {
			text := strings.TrimSpace(stmt.Text)
			if start, end := f.Start, f.End; end > start && end <= len(stmt.Text) {
				if start > 0 && stmt.Text[start-1] == '(' && end < len(stmt.Text) && stmt.Text[end] == ')' {
					start--
					end++
				}
				text = stmt.Text[start:end]
			}
			return fmt.Sprintf("\"%s\" requires WHERE clause", text)
		}))
	advisor.Register(storepb.Engine_POSTGRES, storepb.SQLReviewRule_STATEMENT_WHERE_REQUIRE_UPDATE_DELETE,
		e.SimpleStmt(pgv1review.StatementWhereRequireUpdateDelete, code.StatementNoWhere, func(stmt advisor.V1Stmt[ast.Node], _ v1review.Finding) string {
			return fmt.Sprintf("\"%s\" requires WHERE clause", e.TrimmedText(stmt.Text))
		}))
	advisor.Register(storepb.Engine_POSTGRES, storepb.SQLReviewRule_TABLE_COMMENT,
		e.TableComment(
			func(_ advisor.Context, nodes []ast.Node, cfg *v1review.TableCommentConfig) []v1review.Finding {
				return pgv1review.TableComment(nodes, cfg)
			},
			func(f v1review.Finding) string {
				return fmt.Sprintf("`%s`", f.Name)
			},
		))
	advisor.Register(storepb.Engine_POSTGRES, storepb.SQLReviewRule_TABLE_DISALLOW_PARTITION,
		e.Stmt(
			func(stmt advisor.V1Stmt[ast.Node]) []v1review.Finding {
				return pgv1review.TableDisallowPartition([]ast.Node{stmt.Node})
			},
			func(stmt advisor.V1Stmt[ast.Node], f v1review.Finding) (code.Code, string, bool) {
				if f.Code != v1review.TablePartitioned {
					return 0, "", false
				}
				return code.CreateTablePartition, fmt.Sprintf("Table partition is forbidden, but %q creates", e.TrimmedText(stmt.Text)), true
			},
		))
	advisor.Register(storepb.Engine_POSTGRES, storepb.SQLReviewRule_TABLE_DROP_NAMING_CONVENTION,
		e.NamingRegex(
			pgv1review.TableDropNaming,
			code.TableDropNamingConventionMismatch,
			"naming_payload is required for table drop naming convention rule",
			func(f v1review.Finding, format string) string {
				return fmt.Sprintf("`%s` mismatches drop table naming convention, naming format should be %q", f.Name, format)
			},
			nil,
		))
	advisor.Register(storepb.Engine_POSTGRES, storepb.SQLReviewRule_TABLE_NO_FOREIGN_KEY,
		e.Stmt(
			func(stmt advisor.V1Stmt[ast.Node]) []v1review.Finding {
				return pgv1review.TableNoForeignKey([]ast.Node{stmt.Node})
			},
			func(stmt advisor.V1Stmt[ast.Node], f v1review.Finding) (code.Code, string, bool) {
				if f.Code != v1review.TableForeignKey {
					return 0, "", false
				}
				return code.TableHasFK, fmt.Sprintf("Foreign key is not allowed in the table %q.%q, related statement: \"%s\"", f.Value, f.Table, e.TrimmedText(stmt.Text)), true
			},
		))
	advisor.Register(storepb.Engine_POSTGRES, storepb.SQLReviewRule_TABLE_REQUIRE_PK, pgTableRequirePK(e))
}

func pgNamingFullyQualified(e *advisor.V1Engine[ast.Node]) advisor.Func {
	return func(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
		stmts, nodes, level, title, err := e.Prepare(checkCtx)
		if err != nil {
			return nil, err
		}
		schema := v1review.Schema{Metadata: checkCtx.DBSchema}
		var adviceList []*storepb.Advice
		for i, stmt := range stmts {
			pos := e.PosContentEnd(stmt, v1review.Finding{})
			for _, f := range pgv1review.NamingFullyQualified(nodes[i:i+1:i+1], schema) {
				if f.Code != v1review.NameUnqualified {
					continue
				}
				adviceList = append(adviceList, &storepb.Advice{
					Status:        level,
					Code:          int32(code.NamingNotFullyQualifiedName),
					Title:         title,
					Content:       fmt.Sprintf("unqualified object name: '%s'", f.Name),
					StartPosition: pos,
				})
			}
		}
		return adviceList, nil
	}
}

func pgPriorBackup(e *advisor.V1Engine[ast.Node]) advisor.Func {
	return func(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
		if !checkCtx.EnablePriorBackup {
			return nil, nil
		}
		stmts, nodes, level, title, err := e.Prepare(checkCtx)
		if err != nil {
			return nil, err
		}
		schemaName := common.BackupDatabaseNameOfEngine(storepb.Engine_POSTGRES)
		facts := v1review.PriorBackupFacts{
			TotalSize:      checkCtx.StatementsTotalSize,
			MaxSize:        common.MaxSheetCheckSize,
			Database:       schemaName,
			DatabaseExists: checkCtx.OriginalMetadata.GetSchemaMetadata(schemaName) != nil,
		}
		var adviceList []*storepb.Advice
		for _, f := range pgv1review.PriorBackupCheck(nodes, facts) {
			advice := &storepb.Advice{Status: level, Title: title, Code: code.BuiltinPriorBackupCheck.Int32()}
			switch f.Code {
			case v1review.PriorBackupScriptTooLarge:
				advice.Content = fmt.Sprintf("The size of the SQL statements exceeds the maximum limit of %s bytes for backup", f.Value)
			case v1review.PriorBackupMixedDDLAndDML:
				stmt := stmts[f.Statement]
				advice.Content = fmt.Sprintf("Data change can only run DML, \"%s\" is not DML", e.TrimmedText(stmt.Text))
				advice.StartPosition = e.PosContentStart(stmt, f)
			case v1review.PriorBackupDataModifyingCTE:
				stmt := stmts[f.Statement]
				advice.Content = fmt.Sprintf("Prior backup does not cover rows changed by data-modifying WITH query %s", f.Value)
				switch nodes[f.Statement].(type) {
				case *ast.UpdateStmt, *ast.DeleteStmt:
					advice.Content = fmt.Sprintf("Prior backup does not support data-modifying WITH query %s; the task will fail unless prior backup is disabled or skipped", f.Value)
				default:
				}
				advice.StartPosition = e.PosContentStart(stmt, f)
			case v1review.PriorBackupDatabaseMissing:
				advice.Code = code.SchemaNotExists.Int32()
				advice.Content = fmt.Sprintf("Need schema %q to do prior backup but it does not exist", f.Name)
			case v1review.PriorBackupMixedDMLOnTable:
				advice.Content = fmt.Sprintf("The statement type is not the same for all statements on the same table %q", f.Table)
			default:
				continue
			}
			adviceList = append(adviceList, advice)
		}
		return adviceList, nil
	}
}

func pgColumnNoNull(e *advisor.V1Engine[ast.Node]) advisor.Func {
	return func(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
		stmts, nodes, level, title, err := e.Prepare(checkCtx)
		if err != nil {
			return nil, err
		}
		texts := make([]string, len(stmts))
		for i, stmt := range stmts {
			texts[i] = stmt.Text
		}
		var schema v1review.Schema
		if md := checkCtx.OriginalMetadata; md != nil {
			schema = v1review.Schema{Metadata: md.GetProto(), ObjectCaseSensitive: md.GetIsObjectCaseSensitive()}
		}
		var adviceList []*storepb.Advice
		for _, f := range pgv1review.ColumnNoNull(nodes, texts, schema) {
			stmt := stmts[f.Statement]
			line := e.ContentStartLine(stmt.Text)
			if f.Start != 0 || f.End != 0 {
				line = int32(1 + strings.Count(stmt.Text[:f.Start], "\n"))
			}
			adviceList = append(adviceList, &storepb.Advice{
				Status:        level,
				Code:          code.ColumnCannotNull.Int32(),
				Title:         title,
				Content:       fmt.Sprintf("Column %q in %q.%q cannot have NULL value", f.Name, f.Schema, f.Table),
				StartPosition: &storepb.Position{Line: line + int32(stmt.BaseLine)},
			})
		}
		return adviceList, nil
	}
}

func pgColumnRequired(e *advisor.V1Engine[ast.Node]) advisor.Func {
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
			findings := pgv1review.ColumnRequired(nodes[i:i+1:i+1], list)
			for j := 0; j < len(findings); {
				f := findings[j]
				var missingColumns []string
				for ; j < len(findings) && findings[j].Start == f.Start && findings[j].End == f.End; j++ {
					missingColumns = append(missingColumns, findings[j].Name)
				}
				adviceList = append(adviceList, &storepb.Advice{
					Status:        level,
					Code:          code.NoRequiredColumn.Int32(),
					Title:         title,
					Content:       fmt.Sprintf("Table %q requires columns: %s", f.Table, strings.Join(missingColumns, ", ")),
					StartPosition: e.PosSearchOrStart(stmt, f),
				})
			}
		}
		return adviceList, nil
	}
}

func pgIndexCreateConcurrently(e *advisor.V1Engine[ast.Node]) advisor.Func {
	return func(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
		stmts, nodes, level, title, err := e.Prepare(checkCtx)
		if err != nil {
			return nil, err
		}
		var adviceList []*storepb.Advice
		for _, f := range pgv1review.IndexCreateConcurrently(nodes) {
			advice := &storepb.Advice{
				Status:        level,
				Title:         title,
				StartPosition: e.PosSearchOrStart(stmts[f.Statement], f),
			}
			switch f.Code {
			case v1review.IndexCreatedNotConcurrently:
				advice.Code = code.CreateIndexUnconcurrently.Int32()
				advice.Content = "Creating indexes will block writes on the table, unless use CONCURRENTLY"
			case v1review.IndexDroppedNotConcurrently:
				advice.Code = code.DropIndexUnconcurrently.Int32()
				advice.Content = "Droping indexes will block writes on the table, unless use CONCURRENTLY"
			default:
				continue
			}
			adviceList = append(adviceList, advice)
		}
		return adviceList, nil
	}
}

func pgIndexTotalNumberLimit(e *advisor.V1Engine[ast.Node]) advisor.Func {
	return func(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
		level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
		if err != nil {
			return nil, err
		}
		limit, err := advisor.RequireNumberPayload(checkCtx.Rule, "number_payload is required for this rule")
		if err != nil {
			return nil, err
		}
		stmts, nodes, err := e.Nodes(checkCtx)
		if err != nil {
			return nil, err
		}
		findings, err := pgv1review.IndexTotalNumberLimit(nodes, limit, advisor.V1SchemaOf(checkCtx.FinalMetadata))
		if err != nil {
			advisor.ReadMissingSchema(checkCtx.FinalMetadata, err)
			return nil, err
		}
		title := checkCtx.Rule.Type.String()
		var adviceList []*storepb.Advice
		for _, f := range findings {
			adviceList = append(adviceList, &storepb.Advice{
				Status:        level,
				Code:          code.IndexCountExceedsLimit.Int32(),
				Title:         title,
				Content:       fmt.Sprintf("The count of index in table %q.%q should be no more than %d, but found %s", f.Schema, f.Table, limit, f.Value),
				StartPosition: e.PosContentEnd(stmts[f.Statement], f),
			})
		}
		return adviceList, nil
	}
}

func pgMergeAlterTable(e *advisor.V1Engine[ast.Node]) advisor.Func {
	return func(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
		stmts, nodes, level, title, err := e.Prepare(checkCtx)
		if err != nil {
			return nil, err
		}
		var adviceList []*storepb.Advice
		for _, f := range pgv1review.StatementMergeAlterTable(nodes) {
			adviceList = append(adviceList, &storepb.Advice{
				Status:  level,
				Code:    code.StatementRedundantAlterTable.Int32(),
				Title:   title,
				Content: fmt.Sprintf("There are %s statements to modify table `%s`", f.Value, f.Table),
				// V1 placed the finding on its last statement's last line relative to that
				// statement, without the statement's base line, and ordered findings by that line.
				StartPosition: &storepb.Position{Line: e.ContentEndLine(stmts[f.Statement].Text)},
			})
		}
		slices.SortStableFunc(adviceList, func(a, b *storepb.Advice) int {
			return cmp.Compare(a.StartPosition.Line, b.StartPosition.Line)
		})
		return adviceList, nil
	}
}

func pgTableRequirePK(e *advisor.V1Engine[ast.Node]) advisor.Func {
	return func(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
		stmts, nodes, level, title, err := e.Prepare(checkCtx)
		if err != nil {
			return nil, err
		}
		startLines := make([]int, len(stmts))
		for i, stmt := range stmts {
			startLines[i] = stmt.BaseLine + int(e.ContentStartLine(stmt.Text))
		}
		var adviceList []*storepb.Advice
		for _, f := range pgv1review.TableRequirePK(nodes, startLines, advisor.V1SchemaPtrOf(checkCtx.OriginalMetadata), advisor.V1SchemaPtrOf(checkCtx.FinalMetadata)) {
			if f.Code != v1review.TablePrimaryKeyMissing {
				continue
			}
			content := fmt.Sprintf("Table %q.%q requires PRIMARY KEY", f.Value, f.Name)
			line := 0
			if f.Statement >= 0 {
				line = startLines[f.Statement]
				if text := e.TrimmedText(stmts[f.Statement].Text); text != "" {
					content = fmt.Sprintf("%s, related statement: %q", content, text)
				}
			}
			adviceList = append(adviceList, &storepb.Advice{
				Status:        level,
				Code:          code.TableNoPK.Int32(),
				Title:         title,
				Content:       content,
				StartPosition: &storepb.Position{Line: int32(line)},
			})
		}
		return adviceList, nil
	}
}
