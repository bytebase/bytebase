package base

import (
	"context"
	"strings"

	"github.com/bytebase/omni/metadata"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
)

const maxRestoreCommentLength = 1000

type RestoreContext struct {
	InstanceID              string
	GetDatabaseMetadataFunc GetDatabaseMetadataFunc
	ListDatabaseNamesFunc   ListDatabaseNamesFunc
	IsCaseSensitive         bool
}

// RestoreTableMetadata holds the resolved schema name, table metadata proto,
// and optional primary key index proto needed to generate restore SQL.
type RestoreTableMetadata struct {
	Schema     string
	Table      *metadata.TableMetadata
	PrimaryKey *metadata.IndexMetadata
}

// ExtractStatementOptions configures splitting, range selection, per-statement
// matching, optional normalization, and joining for restore SQL extraction.
type ExtractStatementOptions struct {
	SplitSQL     func(string) ([]Statement, error)
	SplitErrMsg  string // defaults to "failed to split sql"
	ExclusiveEnd bool
	// SourceDatabase, if empty, is parsed from backupItem.SourceTable.Database
	// after SplitSQL succeeds.
	SourceDatabase string
	Match          func(stmtText, sourceDatabase, sourceTable string) (bool, error)
	MatchErrMsg    string // defaults to "failed to parse sql"
	Normalize      func(string) string
	JoinSeparator  string
}

// ValidateBackupItem ensures backupItem and its SourceTable and TargetTable
// fields are non-nil.
func ValidateBackupItem(backupItem *storepb.PriorBackupDetail_Item) error {
	if backupItem == nil {
		return errors.New("backup item is nil")
	}
	if backupItem.SourceTable == nil {
		return errors.New("backup item source table is nil")
	}
	if backupItem.TargetTable == nil {
		return errors.New("backup item target table is nil")
	}
	return nil
}

func parseBackupDatabase(resourceName, role string) (string, error) {
	_, _, db, err := common.GetDatabaseResourceName(resourceName)
	if err != nil {
		return "", errors.Wrapf(err, "failed to get %s database ID for %s", role, resourceName)
	}
	return db, nil
}

// ParseBackupSourceDatabase extracts the database ID from backupItem.SourceTable.Database.
func ParseBackupSourceDatabase(backupItem *storepb.PriorBackupDetail_Item) (string, error) {
	return parseBackupDatabase(backupItem.SourceTable.Database, "source")
}

// ParseBackupTargetDatabase extracts the database ID from backupItem.TargetTable.Database.
func ParseBackupTargetDatabase(backupItem *storepb.PriorBackupDetail_Item) (string, error) {
	return parseBackupDatabase(backupItem.TargetTable.Database, "target")
}

// ParseBackupItemDatabases extracts the source and target database IDs from backupItem.
func ParseBackupItemDatabases(backupItem *storepb.PriorBackupDetail_Item) (sourceDatabase, targetDatabase string, err error) {
	sourceDatabase, err = ParseBackupSourceDatabase(backupItem)
	if err != nil {
		return "", "", err
	}
	targetDatabase, err = ParseBackupTargetDatabase(backupItem)
	if err != nil {
		return "", "", err
	}
	return sourceDatabase, targetDatabase, nil
}

// ExtractSingleStatement extracts and joins the statements within backupItem's
// position range that match opts.Match, wrapping failures with the standard
// restore extraction prefix.
func ExtractSingleStatement(statement string, backupItem *storepb.PriorBackupDetail_Item, opts ExtractStatementOptions) (string, error) {
	sql, err := extractRangeStatements(statement, backupItem, opts)
	if err != nil {
		return "", errors.Errorf("failed to extract single SQL: %v", err)
	}
	return sql, nil
}

func extractRangeStatements(statement string, backupItem *storepb.PriorBackupDetail_Item, opts ExtractStatementOptions) (string, error) {
	if backupItem == nil {
		return "", errors.New("backup item is nil")
	}
	list, err := opts.SplitSQL(statement)
	if err != nil {
		msg := opts.SplitErrMsg
		if msg == "" {
			msg = "failed to split sql"
		}
		return "", errors.Wrap(err, msg)
	}
	start, end := statementRangeIndices(list, backupItem.StartPosition, backupItem.EndPosition, opts.ExclusiveEnd)

	sourceDatabase := opts.SourceDatabase
	if sourceDatabase == "" {
		sourceDatabase, err = ParseBackupSourceDatabase(backupItem)
		if err != nil {
			return "", err
		}
	}

	matchErrMsg := opts.MatchErrMsg
	if matchErrMsg == "" {
		matchErrMsg = "failed to parse sql"
	}

	var result []string
	for i := start; i <= end; i++ {
		matched, err := opts.Match(list[i].Text, sourceDatabase, backupItem.SourceTable.Table)
		if err != nil {
			return "", errors.Wrap(err, matchErrMsg)
		}
		if matched {
			text := list[i].Text
			if opts.Normalize != nil {
				text = opts.Normalize(text)
			}
			result = append(result, text)
		}
	}
	return strings.Join(result, opts.JoinSeparator), nil
}

// ParseForRestore validates that backupItem is non-nil and parses statement,
// wrapping failures with the standard restore extraction prefix.
func ParseForRestore[S any](statement string, backupItem *storepb.PriorBackupDetail_Item, parse func(string) (S, error)) (S, error) {
	var zero S
	if backupItem == nil {
		return zero, errors.Errorf("failed to extract single SQL: %v", errors.New("backup item is nil"))
	}
	stmts, err := parse(statement)
	if err != nil {
		return zero, errors.Errorf("failed to extract single SQL: %v", errors.Wrap(err, "failed to parse statement"))
	}
	return stmts, nil
}

// CheckRestoreDMLs validates the result of finding DML nodes in extracted
// restore SQL, wrapping parse errors and rejecting an empty node slice.
func CheckRestoreDMLs[N any](nodes []N, err error) ([]N, error) {
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse statement")
	}
	if len(nodes) == 0 {
		return nil, errors.New("no DML statement found in extracted SQL")
	}
	return nodes, nil
}

// ClassifyRestoreColumns classifies columns for sourceTable via classify,
// applying Bytebase's metadata checks and wrapping errors with the source table.
func ClassifyRestoreColumns[C, T any](
	rCtx RestoreContext,
	catalog C,
	sourceTable *storepb.PriorBackupDetail_Item_Table,
	sourceDatabase string,
	classify func(C, bool, string, string) (T, error),
) (T, error) {
	cols, err := NewColumnClassifier(rCtx.GetDatabaseMetadataFunc, rCtx.InstanceID, catalog, rCtx.IsCaseSensitive, classify)(sourceDatabase, sourceTable.Table)
	if err != nil {
		var zero T
		return zero, errors.Wrapf(err, "failed to classify columns for %s.%s", sourceTable.Database, sourceTable.Table)
	}
	return cols, nil
}

// ResolveRestoreTableMetadata resolves the source table's schema, table proto,
// and optional primary key proto for restore SQL generation. When defaultSchema
// is empty it reads the unnamed schema (""); otherwise it uses sourceTable.Schema
// falling back to defaultSchema. When the database metadata is nil, notSyncedErr
// formats the returned error if non-nil.
func ResolveRestoreTableMetadata(
	ctx context.Context,
	rCtx RestoreContext,
	sourceDatabase string,
	sourceTable *storepb.PriorBackupDetail_Item_Table,
	defaultSchema string,
	notSyncedErr func(database string) error,
) (RestoreTableMetadata, error) {
	if rCtx.GetDatabaseMetadataFunc == nil {
		return RestoreTableMetadata{}, errors.New("GetDatabaseMetadataFunc is required")
	}
	_, dbMetadata, err := rCtx.GetDatabaseMetadataFunc(ctx, rCtx.InstanceID, sourceDatabase)
	if err != nil {
		return RestoreTableMetadata{}, errors.Wrapf(err, "failed to get database metadata for %s", sourceDatabase)
	}
	if dbMetadata == nil {
		if notSyncedErr != nil {
			return RestoreTableMetadata{}, notSyncedErr(sourceDatabase)
		}
		return RestoreTableMetadata{}, errors.Errorf("database metadata for database %q not found (database not synced)", sourceDatabase)
	}

	schema, scope, missingPrefix, missingSuffix := "", sourceDatabase, "no ", ""
	if defaultSchema != "" {
		schema = sourceTable.Schema
		if schema == "" {
			schema = defaultSchema
		}
		scope, missingPrefix, missingSuffix = schema, "", " not found"
	}

	schemaMetadata := dbMetadata.GetSchemaMetadata(schema)
	if schemaMetadata == nil {
		return RestoreTableMetadata{}, errors.Errorf("%sschema metadata%s for %s", missingPrefix, missingSuffix, scope)
	}
	tableMetadata := schemaMetadata.GetTable(sourceTable.Table)
	if tableMetadata == nil {
		return RestoreTableMetadata{}, errors.Errorf("%stable metadata%s for %s.%s", missingPrefix, missingSuffix, scope, sourceTable.Table)
	}

	var primaryKey *metadata.IndexMetadata
	if pk := tableMetadata.GetPrimaryKey(); pk != nil {
		primaryKey = pk.GetProto()
	}
	return RestoreTableMetadata{
		Schema:     schema,
		Table:      tableMetadata.GetProto(),
		PrimaryKey: primaryKey,
	}, nil
}

// FormatRestoreComment truncates sql to the restore comment length limit,
// appending "..." when truncated.
func FormatRestoreComment(sql string) string {
	comment, truncated := common.TruncateString(sql, maxRestoreCommentLength)
	if truncated {
		return comment + "..."
	}
	return comment
}

// FilterDMLsInRange returns the items whose [start, end] positions fall within
// backupItem's [StartPosition, EndPosition].
func FilterDMLsInRange[T any](
	items []T,
	backupItem *storepb.PriorBackupDetail_Item,
	bounds func(T) (start, end *storepb.Position),
) []T {
	var result []T
	for _, item := range items {
		start, end := bounds(item)
		if positionInRange(start, end, backupItem.StartPosition, backupItem.EndPosition) {
			result = append(result, item)
		}
	}
	return result
}

// StatementNotFoundError reports that no DML statement matched backupItem's
// position range.
func StatementNotFoundError(backupItem *storepb.PriorBackupDetail_Item) error {
	return errors.Errorf("could not find statement at position (line %d:%d - %d:%d)",
		backupItem.StartPosition.Line, backupItem.StartPosition.Column,
		backupItem.EndPosition.Line, backupItem.EndPosition.Column)
}

// positionInRange reports whether [start, end] falls within
// [targetStart, targetEnd] in line:column order.
func positionInRange(start, end, targetStart, targetEnd *storepb.Position) bool {
	return equalOrGreaterPosition(start, targetStart) && equalOrLessPosition(end, targetEnd)
}

// statementRangeIndices returns the inclusive [start, end] indices in list covering
// [startPos, endPos]. When exclusiveEnd is true, a statement whose Start >= endPos
// ends the range at i - 1; otherwise it ends at i.
func statementRangeIndices(list []Statement, startPos, endPos *storepb.Position, exclusiveEnd bool) (start, end int) {
	start = 0
	end = len(list) - 1
	for i, item := range list {
		if equalOrLessPosition(item.Start, startPos) {
			start = i
		}
	}
	for i := len(list) - 1; i >= 0; i-- {
		if equalOrGreaterPosition(list[i].Start, endPos) {
			if exclusiveEnd {
				end = i - 1
			} else {
				end = i
			}
		}
	}
	return start, end
}

// equalOrLessPosition reports whether a is at or before b in line:column order.
func equalOrLessPosition(a, b *storepb.Position) bool {
	if a.Line < b.Line {
		return true
	}
	return a.Line == b.Line && a.Column <= b.Column
}

// equalOrGreaterPosition reports whether a is at or after b in line:column order.
func equalOrGreaterPosition(a, b *storepb.Position) bool {
	if a.Line > b.Line {
		return true
	}
	return a.Line == b.Line && a.Column >= b.Column
}
