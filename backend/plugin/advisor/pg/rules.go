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
	e := advisor.NewV1Engine(advisor.NodeExtractor(pgparser.GetNode), nil, advisor.V1PosContentStart)

	advisor.RegisterRules(storepb.Engine_POSTGRES, map[storepb.SQLReviewRule_Type]advisor.Advisor{
		storepb.SQLReviewRule_BUILTIN_WALK_THROUGH_CHECK: &advisor.BuiltinWalkThroughCheckAdvisor{},
		storepb.SQLReviewRule_BUILTIN_PRIOR_BACKUP_CHECK: e.PriorBackup(advisor.V1PriorBackupConfig[ast.Node]{
			Engine: storepb.Engine_POSTGRES,
			Facts: func(_ context.Context, checkCtx advisor.Context) v1review.PriorBackupFacts {
				schemaName := common.BackupDatabaseNameOfEngine(storepb.Engine_POSTGRES)
				return v1review.PriorBackupFacts{
					TotalSize:      checkCtx.StatementsTotalSize,
					MaxSize:        common.MaxSheetCheckSize,
					Database:       schemaName,
					DatabaseExists: checkCtx.OriginalMetadata.GetSchemaMetadata(schemaName) != nil,
				}
			},
			Check: func(_ advisor.Context, _ []advisor.V1Stmt[ast.Node], nodes []ast.Node, facts v1review.PriorBackupFacts) ([]v1review.Finding, error) {
				return pgv1review.PriorBackupCheck(nodes, facts), nil
			},
			Render: func(stmts []advisor.V1Stmt[ast.Node], nodes []ast.Node, f v1review.Finding) (code.Code, string, *storepb.Position, bool) {
				switch f.Code {
				case v1review.PriorBackupMixedDDLAndDML:
					stmt := stmts[f.Statement]
					return code.BuiltinPriorBackupCheck, fmt.Sprintf("Data change can only run DML, \"%s\" is not DML", e.TrimmedText(stmt.Text)), e.PosContentStart(stmt, f), true
				case v1review.PriorBackupDataModifyingCTE:
					stmt := stmts[f.Statement]
					msg := fmt.Sprintf("Prior backup does not cover rows changed by data-modifying WITH query %s", f.Value)
					switch nodes[f.Statement].(type) {
					case *ast.UpdateStmt, *ast.DeleteStmt:
						msg = fmt.Sprintf("Prior backup does not support data-modifying WITH query %s; the task will fail unless prior backup is disabled or skipped", f.Value)
					default:
					}
					return code.BuiltinPriorBackupCheck, msg, e.PosContentStart(stmt, f), true
				case v1review.PriorBackupDatabaseMissing:
					return code.SchemaNotExists, fmt.Sprintf("Need schema %q to do prior backup but it does not exist", f.Name), nil, true
				case v1review.PriorBackupMixedDMLOnTable:
					return code.BuiltinPriorBackupCheck, fmt.Sprintf("The statement type is not the same for all statements on the same table %q", f.Table), nil, true
				default:
					return 0, "", nil, false
				}
			},
		}),
		storepb.SQLReviewRule_STATEMENT_INSERT_ROW_LIMIT:   pgInsertRowLimit(e),
		storepb.SQLReviewRule_STATEMENT_AFFECTED_ROW_LIMIT: pgAffectedRowLimit(e),
		storepb.SQLReviewRule_STATEMENT_DML_DRY_RUN:        pgDMLDryRun(e),
		storepb.SQLReviewRule_STATEMENT_OBJECT_OWNER_CHECK: pgObjectOwnerCheck(e),

		storepb.SQLReviewRule_SYSTEM_CHARSET_ALLOWLIST: e.StringArrayStmt(pgv1review.SystemCharsetAllowlist, "", code.DisabledCharset, "",
			e.FmtName("\"\" used disabled encoding '%s'"),
			func(stmt advisor.V1Stmt[ast.Node], _ v1review.Finding) *storepb.Position {
				return &storepb.Position{Line: int32(stmt.BaseLine)}
			}),
		storepb.SQLReviewRule_SYSTEM_COLLATION_ALLOWLIST: e.StringArrayStmt(pgv1review.SystemCollationAllowlist, "", code.DisabledCollation, "", func(stmt advisor.V1Stmt[ast.Node], f v1review.Finding) string {
			return fmt.Sprintf("Use disabled collation \"%s\", related statement \"%s\"", f.Name, strings.TrimSpace(stmt.Text))
		}),
		storepb.SQLReviewRule_COLUMN_COMMENT: e.ColumnComment(
			func(_ advisor.Context, nodes []ast.Node, required bool, maxLength int) []v1review.Finding {
				return pgv1review.ColumnComment(nodes, required, maxLength)
			},
			0, false,
			func(f v1review.Finding) string { return fmt.Sprintf("`%s.%s`", f.Table, f.Name) },
			e.PosSearchOrStart,
		),
		storepb.SQLReviewRule_COLUMN_DEFAULT_DISALLOW_VOLATILE: e.SimpleStmt(pgv1review.ColumnDefaultDisallowVolatile, code.NoDefault,
			e.FmtTableCol(`Column %q.%q in schema "public" has volatile DEFAULT`), e.PosSearchOrStart),
		storepb.SQLReviewRule_COLUMN_DISALLOW_CHANGE_TYPE: e.SimpleStmt(pgv1review.ColumnDisallowChangeType, code.ChangeColumnType, func(stmt advisor.V1Stmt[ast.Node], _ v1review.Finding) string {
			return fmt.Sprintf("The statement \"%s\" changes column type", strings.TrimSpace(e.TrimmedText(stmt.Text)))
		}, e.PosSearchOrStart),
		storepb.SQLReviewRule_COLUMN_MAXIMUM_CHARACTER_LENGTH: e.NumberStmt(pgv1review.ColumnMaximumCharacterLength, true, code.CharLengthExceedsLimit, func(_ advisor.V1Stmt[ast.Node], f v1review.Finding, maximum int) string {
			tableName := fmt.Sprintf("%q", f.Table)
			if f.Value != "" {
				tableName = fmt.Sprintf("%q.%q", f.Value, f.Table)
			}
			return fmt.Sprintf("The length of the CHAR column %q in table %s is bigger than %d, please use VARCHAR instead", f.Name, tableName, maximum)
		}, e.PosSearchOrStart),
		storepb.SQLReviewRule_COLUMN_NO_NULL: e.BatchStmts(
			func(checkCtx advisor.Context, stmts []advisor.V1Stmt[ast.Node], nodes []ast.Node) []v1review.Finding {
				texts := make([]string, len(stmts))
				for i, stmt := range stmts {
					texts[i] = stmt.Text
				}
				return pgv1review.ColumnNoNull(nodes, texts, advisor.V1SchemaOf(checkCtx.OriginalMetadata))
			},
			func(stmts []advisor.V1Stmt[ast.Node], f v1review.Finding) (code.Code, string, *storepb.Position, bool) {
				return code.ColumnCannotNull, fmt.Sprintf("Column %q in %q.%q cannot have NULL value", f.Name, f.Schema, f.Table), e.PosFinding(stmts[f.Statement], f), true
			},
		),
		storepb.SQLReviewRule_COLUMN_REQUIRE_DEFAULT: e.SimpleStmt(pgv1review.ColumnRequireDefault, code.NoDefault,
			e.FmtTableCol(`Column %q.%q in schema "public" doesn't have DEFAULT`), e.PosSearchOrStart),
		storepb.SQLReviewRule_COLUMN_REQUIRED: e.ColumnRequired(advisor.V1ColumnRequiredConfig[ast.Node]{
			Fn:        pgv1review.ColumnRequired,
			SameGroup: func(a, b v1review.Finding) bool { return a.Start == b.Start && a.End == b.End },
			Pos:       e.PosSearchOrStart,
		}),
		storepb.SQLReviewRule_COLUMN_TYPE_DISALLOW_LIST: e.StringArrayStmt(pgv1review.ColumnTypeDisallowList, "", code.DisabledColumnType, "", func(_ advisor.V1Stmt[ast.Node], f v1review.Finding) string {
			return fmt.Sprintf("Disallow column type %s but column %q.%q is", strings.ToUpper(f.Value), f.Table, f.Name)
		}, e.PosSearchOrStart),
		storepb.SQLReviewRule_SYSTEM_COMMENT_LENGTH: e.NumberStmt(pgv1review.SystemCommentLength, false, code.CommentTooLong,
			e.FmtLimit("The length of comment should be within %d characters")),
		storepb.SQLReviewRule_INDEX_CREATE_CONCURRENTLY: e.BatchNodes(pgv1review.IndexCreateConcurrently,
			func(stmts []advisor.V1Stmt[ast.Node], f v1review.Finding) (code.Code, string, *storepb.Position, bool) {
				switch f.Code {
				case v1review.IndexCreatedNotConcurrently:
					return code.CreateIndexUnconcurrently, "Creating indexes will block writes on the table, unless use CONCURRENTLY", e.PosSearchOrStart(stmts[f.Statement], f), true
				case v1review.IndexDroppedNotConcurrently:
					return code.DropIndexUnconcurrently, "Droping indexes will block writes on the table, unless use CONCURRENTLY", e.PosSearchOrStart(stmts[f.Statement], f), true
				default:
					return 0, "", nil, false
				}
			},
		),
		storepb.SQLReviewRule_INDEX_KEY_NUMBER_LIMIT: e.NumberStmt(pgv1review.IndexKeyNumberLimit, false, code.IndexKeyNumberExceedsLimit, func(_ advisor.V1Stmt[ast.Node], f v1review.Finding, limit int) string {
			return fmt.Sprintf("The number of keys of index %q in table %q should be not greater than %d", f.Name, f.Table, limit)
		}, e.PosSearchOrStart),
		storepb.SQLReviewRule_INDEX_NO_DUPLICATE_COLUMN: e.SimpleStmt(pgv1review.IndexNoDuplicateColumn, code.DuplicateColumnInIndex, func(_ advisor.V1Stmt[ast.Node], f v1review.Finding) string {
			return fmt.Sprintf("%s %q has duplicate column %q.%q", f.Value, f.Name, f.Table, f.Other)
		}, e.PosSearchOrStart),
		storepb.SQLReviewRule_INDEX_PRIMARY_KEY_TYPE_ALLOWLIST: e.StringArrayStmt(pgv1review.IndexPrimaryKeyTypeAllowlist, "", code.IndexPKType, "", func(_ advisor.V1Stmt[ast.Node], f v1review.Finding) string {
			return fmt.Sprintf("The column %q is one of the primary key, but its type %q is not in allowlist", f.Name, f.Value)
		}, e.PosSearchOrStart),
		storepb.SQLReviewRule_INDEX_TOTAL_NUMBER_LIMIT: e.IndexTotalNumberLimit(pgv1review.IndexTotalNumberLimit, func(f v1review.Finding, limit int) string {
			return fmt.Sprintf("The count of index in table %q.%q should be no more than %d, but found %s", f.Schema, f.Table, limit, f.Value)
		}, e.PosContentEnd),
		storepb.SQLReviewRule_STATEMENT_INSERT_DISALLOW_ORDER_BY_RAND: e.SimpleStmt(pgv1review.StatementInsertDisallowOrderByRand, code.InsertUseOrderByRand,
			e.FmtStmt("The INSERT statement uses ORDER BY random() or random_between(), related statement \"%s\"")),
		storepb.SQLReviewRule_STATEMENT_INSERT_MUST_SPECIFY_COLUMN: e.SimpleStmt(pgv1review.StatementInsertMustSpecifyColumn, code.InsertNotSpecifyColumn,
			e.FmtStmt("The INSERT statement must specify columns but \"%s\" does not")),
		storepb.SQLReviewRule_SCHEMA_BACKWARD_COMPATIBILITY: e.BackwardCompatibility(
			func(_ advisor.Context, nodes []ast.Node) []v1review.Finding {
				return pgv1review.SchemaBackwardCompatibility(nodes)
			},
			nil,
			func(stmt advisor.V1Stmt[ast.Node], _ v1review.Finding, _ string) string {
				return fmt.Sprintf(`"%s" may cause incompatibility with the existing data and code`, e.TrimmedText(stmt.Text))
			},
		),
		storepb.SQLReviewRule_NAMING_COLUMN: e.NamingRule(
			pgv1review.NamingColumn, code.NamingColumnConventionMismatch, "", "column", true,
			func(f v1review.Finding) string { return fmt.Sprintf(`"%s"."%s"`, f.Table, f.Name) }, e.PosNameOrStart,
		),
		storepb.SQLReviewRule_NAMING_INDEX_FK: e.TemplateNaming(advisor.V1TemplateNamingConfig[ast.Node]{
			FnNoSchema: pgv1review.NamingForeignKey, Code: code.NamingFKConventionMismatch, Kind: "Foreign key", Pos: e.PosNameOrStart,
		}),
		storepb.SQLReviewRule_NAMING_FULLY_QUALIFIED: e.StmtCtx(
			func(_ context.Context, checkCtx advisor.Context, _ []ast.Node) (func([]ast.Node) []v1review.Finding, error) {
				schema := v1review.Schema{Metadata: checkCtx.DBSchema}
				return func(nodes []ast.Node) []v1review.Finding {
					return pgv1review.NamingFullyQualified(nodes, schema)
				}, nil
			},
			func(_ advisor.Context, _ advisor.V1Stmt[ast.Node], f v1review.Finding) (code.Code, string, bool) {
				if f.Code != v1review.NameUnqualified {
					return 0, "", false
				}
				return code.NamingNotFullyQualifiedName, fmt.Sprintf("unqualified object name: '%s'", f.Name), true
			},
			e.PosContentEnd,
		),
		storepb.SQLReviewRule_NAMING_INDEX_IDX: e.TemplateNaming(advisor.V1TemplateNamingConfig[ast.Node]{
			FnNoErr: pgv1review.NamingIndex, Code: code.NamingIndexConventionMismatch, Kind: "Index",
		}),
		storepb.SQLReviewRule_NAMING_INDEX_PK: e.TemplateNaming(advisor.V1TemplateNamingConfig[ast.Node]{
			FnNoErr: pgv1review.NamingPrimaryKey, Code: code.NamingPKConventionMismatch, Kind: "Primary key", Pos: e.PosNameOrStart,
		}),
		storepb.SQLReviewRule_NAMING_TABLE: e.NamingRule(
			pgv1review.NamingTable, code.NamingTableConventionMismatch, "naming_payload is required for naming table rule", "table", true,
			advisor.DoubleQuotedName,
		),
		storepb.SQLReviewRule_NAMING_INDEX_UK: e.TemplateNaming(advisor.V1TemplateNamingConfig[ast.Node]{
			FnNoErr: pgv1review.NamingUniqueKey, Code: code.NamingUKConventionMismatch, Kind: "Unique key",
			InvalidFmt: "Failed to compile regex for unique key naming convention: %s", NilInvalidPos: true, Pos: e.PosNameOrStart,
		}),
		storepb.SQLReviewRule_STATEMENT_ADD_CHECK_NOT_VALID: e.SimpleConst(pgv1review.StatementAddCheckNotValid, code.StatementAddCheckWithValidation,
			"Adding check constraints with validation will block reads and writes. You can add check constraints not valid and then validate separately"),
		storepb.SQLReviewRule_STATEMENT_ADD_FOREIGN_KEY_NOT_VALID: e.SimpleConst(pgv1review.StatementAddForeignKeyNotValid, code.StatementAddFKWithValidation,
			"Adding foreign keys with validation will block reads and writes. You can add check foreign keys not valid and then validate separately"),
		storepb.SQLReviewRule_STATEMENT_CHECK_SET_ROLE_VARIABLE: e.SimpleBatchNodes(pgv1review.StatementCheckSetRoleVariable, code.StatementCheckSetRoleVariable, "",
			func(v1review.Finding) string { return "No SET ROLE statement found." }, e.PosNil),
		storepb.SQLReviewRule_STATEMENT_CREATE_SPECIFY_SCHEMA: e.SimpleStmt(pgv1review.StatementCreateSpecifySchema, code.StatementCreateWithoutSchemaName,
			e.FmtName("%s schema should be specified.")),
		storepb.SQLReviewRule_STATEMENT_DISALLOW_ADD_COLUMN_WITH_DEFAULT: e.SimpleConst(pgv1review.StatementDisallowAddColumnWithDefault, code.StatementAddColumnWithDefault,
			"Adding column with DEFAULT will locked the whole table and rewriting each rows"),
		storepb.SQLReviewRule_STATEMENT_DISALLOW_ADD_NOT_NULL: e.SimpleStmt(pgv1review.StatementDisallowAddNotNull, code.StatementAddNotNull,
			e.FmtName("Setting NOT NULL will block reads and writes. You can use CHECK (%q IS NOT NULL) instead")),
		storepb.SQLReviewRule_STATEMENT_DISALLOW_COMMIT: e.SimpleStmt(pgv1review.StatementDisallowCommit, code.StatementDisallowCommit, func(stmt advisor.V1Stmt[ast.Node], _ v1review.Finding) string {
			return fmt.Sprintf("Commit is not allowed, related statement: \"%s\"", cmp.Or(e.TrimmedText(stmt.Text), "<unknown statement>"))
		}),
		storepb.SQLReviewRule_STATEMENT_DISALLOW_ON_DEL_CASCADE: e.SimpleConst(pgv1review.StatementDisallowOnDelCascade, code.StatementDisallowCascade,
			"The CASCADE option is not permitted for ON DELETE clauses"),
		storepb.SQLReviewRule_STATEMENT_DISALLOW_RM_TBL_CASCADE: e.SimpleConst(pgv1review.StatementDisallowRmTblCascade, code.StatementDisallowCascade,
			"The use of CASCADE is not permitted when removing a table"),
		storepb.SQLReviewRule_STATEMENT_DISALLOW_TRUNCATE: e.SimpleStmt(pgv1review.StatementDisallowTruncate, code.StatementDisallowTruncate, func(_ advisor.V1Stmt[ast.Node], f v1review.Finding) string {
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
		}, e.PosFinding),
		storepb.SQLReviewRule_STATEMENT_MERGE_ALTER_TABLE: pgMergeAlterTable(e),
		storepb.SQLReviewRule_STATEMENT_WHERE_NO_LEADING_WILDCARD_LIKE: e.SimpleStmt(pgv1review.StatementWhereNoLeadingWildcardLike, code.StatementLeadingWildcardLike,
			e.FmtStmt("\"%s\" uses leading wildcard LIKE")),
		storepb.SQLReviewRule_STATEMENT_SELECT_NO_SELECT_ALL: e.SimpleStmt(pgv1review.StatementSelectNoSelectAll, code.StatementSelectAll, func(stmt advisor.V1Stmt[ast.Node], f v1review.Finding) string {
			text := e.TrimmedText(stmt.Text)
			if f.End > f.Start && f.End <= len(stmt.Text) {
				text = strings.TrimSpace(stmt.Text[f.Start:f.End])
			}
			return fmt.Sprintf("\"%s\" uses SELECT all", text)
		}),
		storepb.SQLReviewRule_STATEMENT_NON_TRANSACTIONAL: e.Stmt(pgv1review.StatementNonTransactional,
			func(_ advisor.V1Stmt[ast.Node], _ v1review.Finding) (code.Code, string, bool) {
				return code.StatementNonTransactional, "This statement is non-transactional", true
			},
		),
		storepb.SQLReviewRule_STATEMENT_WHERE_DISALLOW_FUNCTIONS_AND_CALCULATIONS: e.WhereDisallowFuncAndCalc(func(checkCtx advisor.Context) func([]ast.Node) []v1review.Finding {
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
		}, false, e.PosFinding),
		storepb.SQLReviewRule_STATEMENT_WHERE_REQUIRE_SELECT: e.SimpleStmt(pgv1review.StatementWhereRequireSelect, code.StatementNoWhere, func(stmt advisor.V1Stmt[ast.Node], f v1review.Finding) string {
			text := strings.TrimSpace(stmt.Text)
			if start, end := f.Start, f.End; end > start && end <= len(stmt.Text) {
				if start > 0 && stmt.Text[start-1] == '(' && end < len(stmt.Text) && stmt.Text[end] == ')' {
					start--
					end++
				}
				text = stmt.Text[start:end]
			}
			return fmt.Sprintf("\"%s\" requires WHERE clause", text)
		}),
		storepb.SQLReviewRule_STATEMENT_WHERE_REQUIRE_UPDATE_DELETE: e.SimpleStmt(pgv1review.StatementWhereRequireUpdateDelete, code.StatementNoWhere,
			e.FmtStmt("\"%s\" requires WHERE clause")),
		storepb.SQLReviewRule_TABLE_COMMENT: e.TableComment(
			func(_ advisor.Context, nodes []ast.Node, cfg *v1review.TableCommentConfig) []v1review.Finding {
				return pgv1review.TableComment(nodes, cfg)
			},
			advisor.BacktickName,
		),
		storepb.SQLReviewRule_TABLE_DISALLOW_PARTITION: e.SimpleStmt(pgv1review.TableDisallowPartition, code.CreateTablePartition,
			e.FmtStmt("Table partition is forbidden, but %q creates")),
		storepb.SQLReviewRule_TABLE_DROP_NAMING_CONVENTION: e.NamingRule(
			pgv1review.TableDropNaming, code.TableDropNamingConventionMismatch, "naming_payload is required for table drop naming convention rule", "drop table", false,
			advisor.BacktickName,
		),
		storepb.SQLReviewRule_TABLE_NO_FOREIGN_KEY: e.SimpleStmt(pgv1review.TableNoForeignKey, code.TableHasFK, func(stmt advisor.V1Stmt[ast.Node], f v1review.Finding) string {
			return fmt.Sprintf("Foreign key is not allowed in the table %q.%q, related statement: \"%s\"", f.Value, f.Table, e.TrimmedText(stmt.Text))
		}),
		storepb.SQLReviewRule_TABLE_REQUIRE_PK: e.BatchStmts(
			func(checkCtx advisor.Context, stmts []advisor.V1Stmt[ast.Node], nodes []ast.Node) []v1review.Finding {
				startLines := make([]int, len(stmts))
				for i, stmt := range stmts {
					startLines[i] = stmt.BaseLine + int(e.ContentStartLine(stmt.Text))
				}
				return pgv1review.TableRequirePK(nodes, startLines, advisor.V1SchemaPtrOf(checkCtx.OriginalMetadata), advisor.V1SchemaPtrOf(checkCtx.FinalMetadata))
			},
			func(stmts []advisor.V1Stmt[ast.Node], f v1review.Finding) (code.Code, string, *storepb.Position, bool) {
				if f.Code != v1review.TablePrimaryKeyMissing {
					return 0, "", nil, false
				}
				content := fmt.Sprintf("Table %q.%q requires PRIMARY KEY", f.Value, f.Name)
				line := 0
				if f.Statement >= 0 {
					stmt := stmts[f.Statement]
					line = stmt.BaseLine + int(e.ContentStartLine(stmt.Text))
					if text := e.TrimmedText(stmt.Text); text != "" {
						content = fmt.Sprintf("%s, related statement: %q", content, text)
					}
				}
				return code.TableNoPK, content, &storepb.Position{Line: int32(line)}, true
			},
		),
	})
}

func pgMergeAlterTable(e *advisor.V1Engine[ast.Node]) advisor.Func {
	base := e.BatchNodes(pgv1review.StatementMergeAlterTable,
		func(stmts []advisor.V1Stmt[ast.Node], f v1review.Finding) (code.Code, string, *storepb.Position, bool) {
			// V1 placed the finding on its last statement's last line relative to that
			// statement, without the statement's base line, and ordered findings by that line.
			return code.StatementRedundantAlterTable,
				fmt.Sprintf("There are %s statements to modify table `%s`", f.Value, f.Table),
				&storepb.Position{Line: e.ContentEndLine(stmts[f.Statement].Text)},
				true
		},
	)
	return func(ctx context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
		adviceList, err := base(ctx, checkCtx)
		if err != nil {
			return nil, err
		}
		slices.SortStableFunc(adviceList, func(a, b *storepb.Advice) int {
			return cmp.Compare(a.StartPosition.Line, b.StartPosition.Line)
		})
		return adviceList, nil
	}
}
