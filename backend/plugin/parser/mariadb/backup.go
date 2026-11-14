package mariadb

import (
	"context"
	"log/slog"

	"github.com/pkg/errors"

	"github.com/bytebase/omni/mariadb/analysis"
	"github.com/bytebase/omni/mariadb/ast"
	mariadbparser "github.com/bytebase/omni/mariadb/parser"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/store/model"
)

func init() {
	base.RegisterTransformDMLToSelect(storepb.Engine_MARIADB, TransformDMLToSelect)
}

type StatementType = analysis.DMLType

const (
	StatementTypeUnknown = analysis.DMLTypeUnknown
	StatementTypeUpdate  = analysis.DMLTypeUpdate
	StatementTypeInsert  = analysis.DMLTypeInsert
	StatementTypeDelete  = analysis.DMLTypeDelete
)

type TableReference = analysis.TableReference

type StatementInfo = analysis.DMLTarget

func TransformDMLToSelect(ctx context.Context, tCtx base.TransformContext, statement string, sourceDatabase string, targetDatabase string, tablePrefix string) ([]base.BackupStatement, error) {
	backups, err := analysis.TransformDMLToSelect(statement, analysis.BackupOptions{
		DefaultDatabase: sourceDatabase,
		BackupDatabase:  targetDatabase,
		TablePrefix:     tablePrefix,
		Catalog:         newBackupCatalog(ctx, tCtx.InstanceID, tCtx.GetDatabaseMetadataFunc, tCtx.ListDatabaseNamesFunc),
		Columns: func(database, table string) ([]string, []string, error) {
			return classifyColumns(ctx, tCtx.GetDatabaseMetadataFunc, tCtx.ListDatabaseNamesFunc, tCtx.IsCaseSensitive, tCtx.InstanceID, database, table)
		},
	})
	if err != nil || backups == nil {
		return nil, err
	}
	return convertBackupStatements(statement, backups)
}

// convertBackupStatements gives each backup the positions SplitSQL gives the
// first and last segment it covers.
func convertBackupStatements(statement string, backups []analysis.BackupStatement) ([]base.BackupStatement, error) {
	segments := mariadbparser.Split(statement)
	list, err := SplitSQL(statement)
	if err != nil {
		return nil, err
	}
	starts := make(map[int]*storepb.Position, len(segments))
	ends := make(map[int]*storepb.Position, len(segments))
	for i, segment := range segments {
		starts[segment.ByteStart] = list[i].Start
		ends[segment.ByteEnd] = list[i].End
	}
	result := make([]base.BackupStatement, 0, len(backups))
	for _, backup := range backups {
		result = append(result, base.BackupStatement{
			Statement:       backup.Statement,
			SourceTableName: backup.SourceTable,
			TargetTableName: backup.BackupTable,
			StartPosition:   starts[backup.ByteStart],
			EndPosition:     ends[backup.ByteEnd],
		})
	}
	return result, nil
}

// ExtractTables extracts table references from a DML statement.
func ExtractTables(databaseName string, node ast.Node, fullSQL string, offset int, dbMetadata *model.DatabaseMetadata) ([]StatementInfo, error) {
	var database analysis.Database
	if dbMetadata != nil {
		database = &catalogDatabase{database: dbMetadata}
	}
	return analysis.ExtractTables(databaseName, node, fullSQL, offset, database)
}

// newBackupCatalog reads the instance's metadata for prior backup. Unlike the
// query span's catalog, it returns lookup errors unchanged.
func newBackupCatalog(ctx context.Context, instanceID string, getDatabaseMetadataFunc base.GetDatabaseMetadataFunc, listDatabaseNamesFunc base.ListDatabaseNamesFunc) analysis.Catalog {
	var catalog analysis.Catalog
	if listDatabaseNamesFunc != nil {
		catalog.ListDatabaseNames = func() ([]string, error) {
			return listDatabaseNamesFunc(ctx, instanceID)
		}
	}
	if getDatabaseMetadataFunc != nil {
		catalog.GetDatabase = func(name string) (analysis.Database, error) {
			_, database, err := getDatabaseMetadataFunc(ctx, instanceID, name)
			if err != nil {
				return nil, err
			}
			if database == nil {
				return nil, nil
			}
			return &catalogDatabase{database: database}, nil
		}
	}
	return catalog
}

// classifyColumns returns a table's generated and other columns, and reports
// a missing callback or database in Bytebase's words.
func classifyColumns(ctx context.Context, getDatabaseMetadataFunc base.GetDatabaseMetadataFunc, listDatabaseNamesFunc base.ListDatabaseNamesFunc, isCaseSensitive bool, instanceID string, database, table string) ([]string, []string, error) {
	if getDatabaseMetadataFunc == nil {
		return nil, nil, errors.New("GetDatabaseMetadataFunc is not set")
	}
	generated, normal, err := analysis.ClassifyColumns(newBackupCatalog(ctx, instanceID, getDatabaseMetadataFunc, listDatabaseNamesFunc), isCaseSensitive, database, table)
	var notFound *analysis.NotFoundError
	if errors.As(err, &notFound) {
		slog.Debug("failed to get database metadata", slog.String("instanceID", instanceID), slog.String("database", database))
		return nil, nil, errors.Errorf("failed to get database metadata for InstanceID %q, Database %q", instanceID, database)
	}
	return generated, normal, err
}

// dmlKeywordStart returns the offset of the UPDATE or DELETE keyword of the
// statement at loc. omni starts the node at its WITH clause, if any, and keeps
// no CTE on it, so the clause is the text between the two.
func dmlKeywordStart(sql string, loc ast.Loc) int {
	if loc.Start < 0 || loc.End > len(sql) || loc.Start >= loc.End {
		return loc.Start
	}
	tokens := mariadbparser.Tokenize(sql[loc.Start:loc.End])
	if len(tokens) == 0 || mariadbparser.TokenName(tokens[0].Type) != "WITH" {
		return loc.Start
	}
	depth := 0
	for _, token := range tokens[1:] {
		switch token.Type {
		case '(':
			depth++
		case ')':
			depth--
		default:
			if depth != 0 {
				continue
			}
			if name := mariadbparser.TokenName(token.Type); name == "UPDATE" || name == "DELETE" {
				return loc.Start + token.Loc
			}
		}
	}
	return loc.Start
}
