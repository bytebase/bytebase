package mysql

import (
	"github.com/bytebase/parser/mysql"
)

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
func (checker *StatementTypeChecker) EnterAlterStatement(ctx *mysql.AlterStatementContext) {
	if parent, ok := ctx.GetParent().(*mysql.SimpleStatementContext); ok {
		if _, ok := parent.GetParent().(*mysql.QueryContext); ok {
			checker.IsDDL = true
		}
	}
}

// EnterCreateStatement is called when production createStatement is entered.
func (checker *StatementTypeChecker) EnterCreateStatement(ctx *mysql.CreateStatementContext) {
	if parent, ok := ctx.GetParent().(*mysql.SimpleStatementContext); ok {
		if _, ok := parent.GetParent().(*mysql.QueryContext); ok {
			checker.IsDDL = true
		}
	}
}

// EnterDropStatement is called when production dropStatement is entered.
func (checker *StatementTypeChecker) EnterDropStatement(ctx *mysql.DropStatementContext) {
	if parent, ok := ctx.GetParent().(*mysql.SimpleStatementContext); ok {
		if _, ok := parent.GetParent().(*mysql.QueryContext); ok {
			checker.IsDDL = true
		}
	}
}

// EnterRenameTableStatement is called when production renameTableStatement is entered.
func (checker *StatementTypeChecker) EnterRenameTableStatement(ctx *mysql.RenameTableStatementContext) {
	if parent, ok := ctx.GetParent().(*mysql.SimpleStatementContext); ok {
		if _, ok := parent.GetParent().(*mysql.QueryContext); ok {
			checker.IsDDL = true
		}
	}
}

// EnterTruncateTableStatement is called when production truncateTableStatement is entered.
func (checker *StatementTypeChecker) EnterTruncateTableStatement(ctx *mysql.TruncateTableStatementContext) {
	if parent, ok := ctx.GetParent().(*mysql.SimpleStatementContext); ok {
		if _, ok := parent.GetParent().(*mysql.QueryContext); ok {
			checker.IsDDL = true
		}
	}
}

// EnterImportStatement is called when production importStatement is entered.
func (checker *StatementTypeChecker) EnterImportStatement(ctx *mysql.ImportStatementContext) {
	if parent, ok := ctx.GetParent().(*mysql.SimpleStatementContext); ok {
		if _, ok := parent.GetParent().(*mysql.QueryContext); ok {
			checker.IsDDL = true
		}
	}
}

// EnterExplainStatement is called when production explainStatement is entered.
func (checker *StatementTypeChecker) EnterExplainStatement(ctx *mysql.ExplainStatementContext) {
	if parent, ok := ctx.GetParent().(*mysql.SimpleStatementContext); ok {
		if _, ok := parent.GetParent().(*mysql.QueryContext); ok {
			checker.IsDDL = true
		}
	}
}

// DML
// EnterCallStatement is called when production callStatement is entered.
func (checker *StatementTypeChecker) EnterCallStatement(ctx *mysql.CallStatementContext) {
	if parent, ok := ctx.GetParent().(*mysql.SimpleStatementContext); ok {
		if _, ok := parent.GetParent().(*mysql.QueryContext); ok {
			checker.IsDML = true
		}
	}
}

// EnterDeleteStatement is called when production deleteStatement is entered.
func (checker *StatementTypeChecker) EnterDeleteStatement(ctx *mysql.DeleteStatementContext) {
	if parent, ok := ctx.GetParent().(*mysql.SimpleStatementContext); ok {
		if _, ok := parent.GetParent().(*mysql.QueryContext); ok {
			checker.IsDML = true
		}
	}
}

// EnterDoStatement is called when production doStatement is entered.
func (checker *StatementTypeChecker) EnterDoStatement(ctx *mysql.DoStatementContext) {
	if parent, ok := ctx.GetParent().(*mysql.SimpleStatementContext); ok {
		if _, ok := parent.GetParent().(*mysql.QueryContext); ok {
			checker.IsDML = true
		}
	}
}

// EnterHandlerStatement is called when production handlerStatement is entered.
func (checker *StatementTypeChecker) EnterHandlerStatement(ctx *mysql.HandlerStatementContext) {
	if parent, ok := ctx.GetParent().(*mysql.SimpleStatementContext); ok {
		if _, ok := parent.GetParent().(*mysql.QueryContext); ok {
			checker.IsDML = true
		}
	}
}

// EnterInsertStatement is called when production insertStatement is entered.
func (checker *StatementTypeChecker) EnterInsertStatement(ctx *mysql.InsertStatementContext) {
	if parent, ok := ctx.GetParent().(*mysql.SimpleStatementContext); ok {
		if _, ok := parent.GetParent().(*mysql.QueryContext); ok {
			checker.IsDML = true
		}
	}
}

// EnterLoadDataFileTail is called when production loadDataFileTail is entered.
func (checker *StatementTypeChecker) EnterLoadDataFileTail(ctx *mysql.LoadDataFileTailContext) {
	if parent, ok := ctx.GetParent().(*mysql.SimpleStatementContext); ok {
		if _, ok := parent.GetParent().(*mysql.QueryContext); ok {
			checker.IsDML = true
		}
	}
}

// EnterReplaceStatement is called when production replaceStatement is entered.
func (checker *StatementTypeChecker) EnterReplaceStatement(ctx *mysql.ReplaceStatementContext) {
	if parent, ok := ctx.GetParent().(*mysql.SimpleStatementContext); ok {
		if _, ok := parent.GetParent().(*mysql.QueryContext); ok {
			checker.IsDML = true
		}
	}
}

// EnterSelectStatement is called when production selectStatement is entered.
func (checker *StatementTypeChecker) EnterSelectStatement(ctx *mysql.SelectStatementContext) {
	if parent, ok := ctx.GetParent().(*mysql.SimpleStatementContext); ok {
		if _, ok := parent.GetParent().(*mysql.QueryContext); ok {
			checker.IsDML = true
		}
	}
}

// EnterUpdateStatement is called when production updateStatement is entered.
func (checker *StatementTypeChecker) EnterUpdateStatement(ctx *mysql.UpdateStatementContext) {
	if parent, ok := ctx.GetParent().(*mysql.SimpleStatementContext); ok {
		if _, ok := parent.GetParent().(*mysql.QueryContext); ok {
			checker.IsDML = true
		}
	}
}

// EnterTransactionOrLockingStatement is called when production transactionOrLockingStatement is entered.
func (checker *StatementTypeChecker) EnterTransactionOrLockingStatement(ctx *mysql.TransactionOrLockingStatementContext) {
	if parent, ok := ctx.GetParent().(*mysql.SimpleStatementContext); ok {
		if _, ok := parent.GetParent().(*mysql.QueryContext); ok {
			checker.IsDML = true
		}
	}
}

// EnterReplicationStatement is called when production replicationStatement is entered.
func (checker *StatementTypeChecker) EnterReplicationStatement(ctx *mysql.ReplicationStatementContext) {
	if parent, ok := ctx.GetParent().(*mysql.SimpleStatementContext); ok {
		if _, ok := parent.GetParent().(*mysql.QueryContext); ok {
			checker.IsDML = true
		}
	}
}

// EnterPreparedStatement is called when production preparedStatement is entered.
func (checker *StatementTypeChecker) EnterPreparedStatement(ctx *mysql.PreparedStatementContext) {
	if parent, ok := ctx.GetParent().(*mysql.SimpleStatementContext); ok {
		if _, ok := parent.GetParent().(*mysql.QueryContext); ok {
			checker.IsDML = true
		}
	}
}
