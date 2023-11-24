package mysql

import (
	"fmt"

	mysql "github.com/bytebase/mysql-parser"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

type CreateAndDropDatabaseChecker struct {
	*mysql.BaseMySQLParserListener

	Text    string
	Results []*storepb.PlanCheckRunResult_Result
}

func (checker *CreateAndDropDatabaseChecker) EnterQuery(ctx *mysql.QueryContext) {
	checker.Text = ctx.GetParser().GetTokenStream().GetTextFromRuleContext(ctx)
}

// EnterCreateDatabase is called when production createDatabase is entered.
func (checker *CreateAndDropDatabaseChecker) EnterCreateDatabase(_ *mysql.CreateDatabaseContext) {
	checker.Results = append(checker.Results, &storepb.PlanCheckRunResult_Result{
		Status:  storepb.PlanCheckRunResult_Result_ERROR,
		Code:    common.TaskTypeCreateDatabase.Int32(),
		Title:   "Cannot create database",
		Content: fmt.Sprintf(`The statement "%s" creates database`, checker.Text),
	})
}

// EnterDropDatabase is called when production dropDatabase is entered.
func (checker *CreateAndDropDatabaseChecker) EnterDropDatabase(_ *mysql.DropDatabaseContext) {
	checker.Results = append(checker.Results, &storepb.PlanCheckRunResult_Result{
		Status:  storepb.PlanCheckRunResult_Result_ERROR,
		Code:    common.TaskTypeDropDatabase.Int32(),
		Title:   "Cannot drop database",
		Content: fmt.Sprintf(`The statement "%s" drops database`, checker.Text),
	})
}

type StatementTypeChecker struct {
	*mysql.BaseMySQLParserListener

	IsDDL     bool
	IsDML     bool
	IsExplain bool
	Text      string
}

func (checker *StatementTypeChecker) EnterQuery(ctx *mysql.QueryContext) {
	checker.Text = ctx.GetParser().GetTokenStream().GetTextFromRuleContext(ctx)
}

// DDL from g4
// alterStatement
// createStatement
// dropStatement
// renameTableStatement
// truncateTableStatement
// importStatement
// EnterAlterStatement is called when production alterStatement is entered.
func (checker *StatementTypeChecker) EnterAlterStatement(_ *mysql.AlterStatementContext) {
	checker.IsDDL = true
}

// EnterCreateStatement is called when production createStatement is entered.
func (checker *StatementTypeChecker) EnterCreateStatement(_ *mysql.CreateStatementContext) {
	checker.IsDDL = true
}

// EnterDropStatement is called when production dropStatement is entered.
func (checker *StatementTypeChecker) EnterDropStatement(_ *mysql.DropStatementContext) {
	checker.IsDDL = true
}

// EnterRenameTableStatement is called when production renameTableStatement is entered.
func (checker *StatementTypeChecker) EnterRenameTableStatement(_ *mysql.RenameTableStatementContext) {
	checker.IsDDL = true
}

// EnterTruncateTableStatement is called when production truncateTableStatement is entered.
func (checker *StatementTypeChecker) EnterTruncateTableStatement(_ *mysql.TruncateTableStatementContext) {
	checker.IsDDL = true
}

// EnterImportStatement is called when production importStatement is entered.
func (checker *StatementTypeChecker) EnterImportStatement(_ *mysql.ImportStatementContext) {
	checker.IsDDL = true
}

// EnterExplainStatement is called when production explainStatement is entered.
func (checker *StatementTypeChecker) EnterExplainStatement(_ *mysql.ExplainStatementContext) {
	checker.IsExplain = true
}

// DML
// EnterCallStatement is called when production callStatement is entered.
func (checker *StatementTypeChecker) EnterCallStatement(_ *mysql.CallStatementContext) {
	checker.IsDML = true
}

// EnterDeleteStatement is called when production deleteStatement is entered.
func (checker *StatementTypeChecker) EnterDeleteStatement(_ *mysql.DeleteStatementContext) {
	checker.IsDML = true
}

// EnterDoStatement is called when production doStatement is entered.
func (checker *StatementTypeChecker) EnterDoStatement(_ *mysql.DoStatementContext) {
	checker.IsDML = true
}

// EnterHandlerStatement is called when production handlerStatement is entered.
func (checker *StatementTypeChecker) EnterHandlerStatement(_ *mysql.HandlerStatementContext) {
	checker.IsDML = true
}

// EnterInsertStatement is called when production insertStatement is entered.
func (checker *StatementTypeChecker) EnterInsertStatement(_ *mysql.InsertStatementContext) {
	checker.IsDML = true
}

// EnterLoadDataFileTail is called when production loadDataFileTail is entered.
func (checker *StatementTypeChecker) EnterLoadDataFileTail(_ *mysql.LoadDataFileTailContext) {
	checker.IsDML = true
}

// EnterReplaceStatement is called when production replaceStatement is entered.
func (checker *StatementTypeChecker) EnterReplaceStatement(_ *mysql.ReplaceStatementContext) {
	checker.IsDML = true
}

// EnterSelectStatement is called when production selectStatement is entered.
func (checker *StatementTypeChecker) EnterSelectStatement(_ *mysql.SelectStatementContext) {
	checker.IsDML = true
}

// EnterUpdateStatement is called when production updateStatement is entered.
func (checker *StatementTypeChecker) EnterUpdateStatement(_ *mysql.UpdateStatementContext) {
	checker.IsDML = true
}

// EnterTransactionOrLockingStatement is called when production transactionOrLockingStatement is entered.
func (checker *StatementTypeChecker) EnterTransactionOrLockingStatement(_ *mysql.TransactionOrLockingStatementContext) {
	checker.IsDML = true
}

// EnterReplicationStatement is called when production replicationStatement is entered.
func (checker *StatementTypeChecker) EnterReplicationStatement(_ *mysql.ReplicationStatementContext) {
	checker.IsDML = true
}

// EnterPreparedStatement is called when production preparedStatement is entered.
func (checker *StatementTypeChecker) EnterPreparedStatement(_ *mysql.PreparedStatementContext) {
	checker.IsDML = true
}

type SDLTypeChecker struct {
	*mysql.BaseMySQLParserListener

	baseLine int
	Results  []*storepb.PlanCheckRunResult_Result
}

// for SDL.
// EnterDropTable is called when production dropTable is entered.
func (checker *SDLTypeChecker) EnterDropTable(ctx *mysql.DropTableContext) {
	if ctx.TableRefList() == nil {
		return
	}

	for _, tableRef := range ctx.TableRefList().AllTableRef() {
		_, tableName := NormalizeMySQLTableRef(tableRef)
		checker.Results = append(checker.Results, &storepb.PlanCheckRunResult_Result{
			Status:  storepb.PlanCheckRunResult_Result_WARNING,
			Code:    common.TaskTypeDropTable.Int32(),
			Title:   "Plan to drop table",
			Content: fmt.Sprintf("Plan to drop table `%s`", tableName),
			Report: &storepb.PlanCheckRunResult_Result_SqlReviewReport_{
				SqlReviewReport: &storepb.PlanCheckRunResult_Result_SqlReviewReport{
					Line:   int32(checker.baseLine + ctx.GetStart().GetLine()),
					Detail: "",
					Code:   0,
				},
			},
		})
	}
}

// EnterDropIndex is called when production dropIndex is entered.
func (checker *SDLTypeChecker) EnterDropIndex(ctx *mysql.DropIndexContext) {
	if ctx.IndexRef() == nil || ctx.TableRef() == nil {
		return
	}

	_, _, indexName := NormalizeIndexRef(ctx.IndexRef())
	_, tableName := NormalizeMySQLTableRef(ctx.TableRef())
	checker.Results = append(checker.Results, &storepb.PlanCheckRunResult_Result{
		Status:  storepb.PlanCheckRunResult_Result_WARNING,
		Code:    common.TaskTypeDropIndex.Int32(),
		Title:   "Plan to drop index",
		Content: fmt.Sprintf("Plan to drop index `%s` on table `%s`", indexName, tableName),
		Report: &storepb.PlanCheckRunResult_Result_SqlReviewReport_{
			SqlReviewReport: &storepb.PlanCheckRunResult_Result_SqlReviewReport{
				Line:   int32(checker.baseLine + ctx.GetStart().GetLine()),
				Detail: "",
				Code:   0,
			},
		},
	})
}

// EnterAlterTable is called when production alterTable is entered.
func (checker *SDLTypeChecker) EnterAlterTable(ctx *mysql.AlterTableContext) {
	if ctx.TableRef() == nil {
		// todo: maybe need to do error handle.
		return
	}

	_, tableName := NormalizeMySQLTableRef(ctx.TableRef())

	if ctx.AlterTableActions() == nil {
		return
	}
	if ctx.AlterTableActions().AlterCommandList() == nil {
		return
	}
	if ctx.AlterTableActions().AlterCommandList().AlterList() == nil {
		return
	}

	for _, alterListItem := range ctx.AlterTableActions().AlterCommandList().AlterList().AllAlterListItem() {
		if alterListItem == nil {
			continue
		}

		switch {
		case alterListItem.DROP_SYMBOL() != nil:
			switch {
			// drop column.
			case alterListItem.ColumnInternalRef() != nil:
				columnName := NormalizeMySQLColumnInternalRef(alterListItem.ColumnInternalRef())
				checker.Results = append(checker.Results, &storepb.PlanCheckRunResult_Result{
					Status:  storepb.PlanCheckRunResult_Result_WARNING,
					Code:    common.TaskTypeDropColumn.Int32(),
					Title:   "Plan to drop column",
					Content: fmt.Sprintf("Plan to drop column `%s` on table `%s`", columnName, tableName),
					Report: &storepb.PlanCheckRunResult_Result_SqlReviewReport_{
						SqlReviewReport: &storepb.PlanCheckRunResult_Result_SqlReviewReport{
							Line:   int32(checker.baseLine + alterListItem.GetStart().GetLine()),
							Detail: "",
							Code:   0,
						},
					},
				})
			// drop primary key.
			case alterListItem.PRIMARY_SYMBOL() != nil && alterListItem.KEY_SYMBOL() != nil:
				checker.Results = append(checker.Results, &storepb.PlanCheckRunResult_Result{
					Status:  storepb.PlanCheckRunResult_Result_WARNING,
					Code:    common.TaskTypeDropPrimaryKey.Int32(),
					Title:   "Plan to drop primary key",
					Content: fmt.Sprintf("Plan to drop primary key on table `%s`", tableName),
					Report: &storepb.PlanCheckRunResult_Result_SqlReviewReport_{
						SqlReviewReport: &storepb.PlanCheckRunResult_Result_SqlReviewReport{
							Line:   int32(checker.baseLine + ctx.GetStart().GetLine()),
							Detail: "",
							Code:   0,
						},
					},
				})
			// drop foreign key.
			case alterListItem.FOREIGN_SYMBOL() != nil && alterListItem.KEY_SYMBOL() != nil && alterListItem.ColumnInternalRef() != nil:
				name := NormalizeMySQLColumnInternalRef(alterListItem.ColumnInternalRef())
				checker.Results = append(checker.Results, &storepb.PlanCheckRunResult_Result{
					Status:  storepb.PlanCheckRunResult_Result_WARNING,
					Code:    common.TaskTypeDropForeignKey.Int32(),
					Title:   "Plan to drop foreign key",
					Content: fmt.Sprintf("Plan to drop foreign key `%s` on table `%s`", name, tableName),
					Report: &storepb.PlanCheckRunResult_Result_SqlReviewReport_{
						SqlReviewReport: &storepb.PlanCheckRunResult_Result_SqlReviewReport{
							Line:   int32(checker.baseLine + ctx.GetStart().GetLine()),
							Detail: "",
							Code:   0,
						},
					},
				})
			// drop check.
			case alterListItem.CHECK_SYMBOL() != nil && alterListItem.Identifier() != nil:
				constraintName := NormalizeMySQLIdentifier(alterListItem.Identifier())
				checker.Results = append(checker.Results, &storepb.PlanCheckRunResult_Result{
					Status:  storepb.PlanCheckRunResult_Result_WARNING,
					Code:    common.TaskTypeDropCheck.Int32(),
					Title:   "Plan to drop check constraint",
					Content: fmt.Sprintf("Plan to drop check constraint `%s` on table `%s`", constraintName, tableName),
					Report: &storepb.PlanCheckRunResult_Result_SqlReviewReport_{
						SqlReviewReport: &storepb.PlanCheckRunResult_Result_SqlReviewReport{
							Line:   int32(checker.baseLine + ctx.GetStart().GetLine()),
							Detail: "",
							Code:   0,
						},
					},
				})
			}
		default:
		}
	}
}
