package base

import (
	"context"
	"log/slog"
	"slices"

	omnianalysis "github.com/bytebase/omni/analysis"
	"github.com/pkg/errors"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
)

type TransformContext struct {
	Version                 any
	InstanceID              string
	GetDatabaseMetadataFunc GetDatabaseMetadataFunc
	ListDatabaseNamesFunc   ListDatabaseNamesFunc
	// ListReferencingDatabasesFunc lists the databases that may hold a foreign
	// key into another database. MySQL-family prior backup reads only these
	// for keys that cascade into its tables; nil makes it read every database.
	ListReferencingDatabasesFunc ListDatabaseNamesFunc
	IsCaseSensitive              bool
	DatabaseName                 string
}

// ReferencingDatabases is ListReferencingDatabasesFunc as omni's MySQL-family
// BackupOptions.ReferencingDatabases takes it, listed once for every table a
// backup checks. The list does not depend on the database asked about.
func (c TransformContext) ReferencingDatabases(ctx context.Context) func(string) ([]string, error) {
	if c.ListReferencingDatabasesFunc == nil {
		return nil
	}
	var names []string
	var err error
	listed := false
	return func(string) ([]string, error) {
		if !listed {
			names, err = c.ListReferencingDatabasesFunc(ctx, c.InstanceID)
			listed = true
		}
		return names, err
	}
}

type BackupStatement struct {
	Statement       string
	SourceSchema    string
	SourceTableName string
	TargetTableName string

	StartPosition *storepb.Position
	EndPosition   *storepb.Position
}

// ConvertBackupStatements maps engine backup items onto Bytebase BackupStatements
// using positions from splitSQL aligned with each segment's byte offsets.
func ConvertBackupStatements[S, B any](
	statement string,
	splitSQL func(string) ([]Statement, error),
	segs []S,
	segBounds func(S) (start, end int),
	backups []B,
	extract func(B) (stmt, srcTable, bakTable string, start, end int),
) ([]BackupStatement, error) {
	list, err := splitSQL(statement)
	if err != nil {
		return nil, err
	}
	starts := make(map[int]*storepb.Position, len(segs))
	ends := make(map[int]*storepb.Position, len(segs))
	for i, seg := range segs {
		s, e := segBounds(seg)
		starts[s] = list[i].Start
		ends[e] = list[i].End
	}
	result := make([]BackupStatement, 0, len(backups))
	for _, b := range backups {
		stmt, src, bak, s, e := extract(b)
		result = append(result, BackupStatement{
			Statement:       stmt,
			SourceTableName: src,
			TargetTableName: bak,
			StartPosition:   starts[s],
			EndPosition:     ends[e],
		})
	}
	return result, nil
}

// ConvertOffsetBackupStatements maps engine backup items with byte offsets onto
// Bytebase BackupStatements in a single ascending pass over statement.
// StartPosition always uses a 0-based column; when zeroBasedEnd is true,
// EndPosition also uses a 0-based column.
func ConvertOffsetBackupStatements[B any](
	statement string,
	backups []B,
	extract func(B) (stmt, srcSchema, srcTable, bakTable string, startOff, endOff int),
	zeroBasedEnd bool,
) []BackupStatement {
	if len(backups) == 0 {
		return nil
	}
	offsets := make([]int, 0, len(backups)*2)
	for _, b := range backups {
		_, _, _, _, startOff, endOff := extract(b)
		offsets = append(offsets, startOff, endOff)
	}
	slices.Sort(offsets)
	mapper := NewByteOffsetPositionMapper(statement)
	positions := make(map[int]*storepb.Position, len(offsets))
	for _, offset := range slices.Compact(offsets) {
		positions[offset] = mapper.Position(offset)
	}

	result := make([]BackupStatement, 0, len(backups))
	for _, b := range backups {
		stmt, srcSchema, srcTable, bakTable, startOff, endOff := extract(b)
		start, end := positions[startOff], positions[endOff]
		endCol := end.Column
		if zeroBasedEnd {
			endCol = max(endCol-1, 0)
		}
		result = append(result, BackupStatement{
			Statement:       stmt,
			SourceSchema:    srcSchema,
			SourceTableName: srcTable,
			TargetTableName: bakTable,
			StartPosition:   &storepb.Position{Line: start.Line, Column: max(start.Column-1, 0)},
			EndPosition:     &storepb.Position{Line: end.Line, Column: endCol},
		})
	}
	return result
}

// BackupCatalogFuncs builds ListDatabaseNames and GetDatabase closures backed
// by Bytebase metadata callbacks for unnamed-schema engines. Unlike query span
// catalogs, metadata lookup errors are returned unchanged.
func BackupCatalogFuncs[D any](
	ctx context.Context,
	instanceID string,
	getDB GetDatabaseMetadataFunc,
	listDBs ListDatabaseNamesFunc,
	wrap func(*UnnamedSchemaDatabase) D,
) (list func() ([]string, error), get func(string) (D, error)) {
	if listDBs != nil {
		list = func() ([]string, error) {
			return listDBs(ctx, instanceID)
		}
	}
	if getDB != nil {
		get = func(name string) (D, error) {
			var zero D
			_, database, err := getDB(ctx, instanceID, name)
			if err != nil {
				return zero, err
			}
			if database == nil {
				return zero, nil
			}
			return wrap(&UnnamedSchemaDatabase{Database: database}), nil
		}
	}
	return list, get
}

// NewColumnClassifier adapts an engine's ClassifyColumns function into a
// backup column callback, validating getDB and translating omni's
// ResourceNotFoundError into Bytebase's wording with a debug log.
func NewColumnClassifier[C, T any](
	getDB GetDatabaseMetadataFunc,
	instanceID string,
	catalog C,
	isCaseSensitive bool,
	classify func(C, bool, string, string) (T, error),
) func(database, table string) (T, error) {
	return func(database, table string) (T, error) {
		var zero T
		if getDB == nil {
			return zero, errors.New("GetDatabaseMetadataFunc is not set")
		}
		columns, err := classify(catalog, isCaseSensitive, database, table)
		var notFound *omnianalysis.ResourceNotFoundError
		if errors.As(err, &notFound) {
			slog.Debug("failed to get database metadata", slog.String("instanceID", instanceID), slog.String("database", database))
			return zero, errors.Errorf("failed to get database metadata for InstanceID %q, Database %q", instanceID, database)
		}
		return columns, err
	}
}
