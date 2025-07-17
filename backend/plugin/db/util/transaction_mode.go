package util

import (
	"context"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

// ExecuteWithTransactionMode executes a statement with the appropriate transaction mode.
// This is a generic implementation that can be used by any database engine.
func ExecuteWithTransactionMode(
	ctx context.Context,
	driver db.Driver,
	statement string,
	opts db.ExecuteOptions,
	engine storepb.Engine,
) (int64, error) {
	// Parse transaction mode from the script
	transactionMode, cleanedStatement := base.ParseTransactionMode(statement)
	
	// Apply engine-specific defaults when transaction mode is not specified
	if transactionMode == base.TransactionModeUnspecified {
		transactionMode = common.GetDefaultTransactionMode(engine, opts.TaskType)
	}

	// Execute based on transaction mode
	if transactionMode == base.TransactionModeOff {
		return ExecuteInAutoCommitMode(ctx, driver, cleanedStatement, opts)
	}
	return ExecuteInTransactionMode(ctx, driver, cleanedStatement, opts)
}

// ExecuteInTransactionMode executes statements within a single transaction.
// This is a generic implementation that can be used by any database engine.
func ExecuteInTransactionMode(
	ctx context.Context,
	driver db.Driver,
	statement string,
	opts db.ExecuteOptions,
) (int64, error) {
	// Use the original Execute method which typically handles transactions
	return driver.Execute(ctx, statement, opts)
}

// ExecuteInAutoCommitMode executes statements sequentially in auto-commit mode.
// This is a generic implementation that can be used by any database engine.
func ExecuteInAutoCommitMode(
	ctx context.Context,
	driver db.Driver,
	statement string,
	opts db.ExecuteOptions,
) (int64, error) {
	// For auto-commit mode, we need each driver to implement its own logic
	// This is a fallback that just calls the original Execute method
	return driver.Execute(ctx, statement, opts)
}