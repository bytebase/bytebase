package pg

import (
	"context"

	"github.com/bytebase/omni/pg/ast"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

// ExtractAccessTablesOption configures access table extraction behavior.
type ExtractAccessTablesOption struct {
	// DefaultDatabase is the default database name to use when table reference doesn't specify one.
	DefaultDatabase string
	// DefaultSchema is the default schema name to use when table reference doesn't specify one.
	DefaultSchema string
	// SkipMetadataValidation skips metadata lookup and validation.
	// When true, returns all table references found in SQL without checking if they exist.
	SkipMetadataValidation bool
	// GetDatabaseMetadata is the function to get database metadata (optional, only used when SkipMetadataValidation is false).
	GetDatabaseMetadata base.GetDatabaseMetadataFunc
	// Ctx is the context for metadata lookup (optional, only used when SkipMetadataValidation is false).
	Ctx context.Context
	// InstanceID is the instance ID for metadata lookup (optional, only used when SkipMetadataValidation is false).
	InstanceID string
}

// ExtractAccessTables extracts all table/view references from a SQL statement.
// This is a lightweight version that doesn't perform full query span analysis.
func ExtractAccessTables(statement string, option ExtractAccessTablesOption) ([]base.ColumnResource, error) {
	omniStmts, err := ParsePg(statement)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to parse PostgreSQL statement")
	}

	if len(omniStmts) != 1 {
		return nil, errors.Errorf("expected exactly 1 statement, got %d", len(omniStmts))
	}

	searchPath := []string{option.DefaultSchema}
	if option.DefaultSchema == "" {
		searchPath = []string{"public"}
	}

	extractor := &accessTableExtractor{
		defaultDatabase:        option.DefaultDatabase,
		searchPath:             searchPath,
		getDatabaseMetadata:    option.GetDatabaseMetadata,
		ctx:                    option.Ctx,
		instanceID:             option.InstanceID,
		skipMetadataValidation: option.SkipMetadataValidation,
	}

	// Walk the omni AST to find all RangeVar nodes (table references)
	ast.Inspect(omniStmts[0].AST, func(n ast.Node) bool {
		if extractor.err != nil {
			return false
		}
		rv, ok := n.(*ast.RangeVar)
		if !ok {
			return true
		}
		extractor.processRangeVar(rv)
		return true
	})

	if extractor.err != nil {
		return nil, errors.Wrapf(extractor.err, "failed to extract access tables")
	}

	return extractor.accessTables, nil
}

type accessTableExtractor struct {
	err                    error
	defaultDatabase        string
	searchPath             []string
	accessTables           []base.ColumnResource
	getDatabaseMetadata    base.GetDatabaseMetadataFunc
	ctx                    context.Context
	instanceID             string
	skipMetadataValidation bool
}

func (a *accessTableExtractor) processRangeVar(rv *ast.RangeVar) {
	resource := base.ColumnResource{
		Database: a.defaultDatabase,
	}

	if rv.Catalogname != "" {
		resource.Database = rv.Catalogname
		resource.Schema = rv.Schemaname
		resource.Table = rv.Relname
	} else if rv.Schemaname != "" {
		resource.Schema = rv.Schemaname
		resource.Table = rv.Relname
	} else {
		resource.Table = rv.Relname
	}

	if a.skipMetadataValidation {
		if resource.Schema == "" && !isSystemResource(resource) {
			if len(a.searchPath) > 0 {
				resource.Schema = a.searchPath[0]
			}
		}
		a.accessTables = append(a.accessTables, resource)
		return
	}

	if !isSystemResource(resource) {
		searchPath := a.searchPath
		if resource.Schema != "" {
			searchPath = []string{resource.Schema}
		}

		_, databaseMetadata, err := a.getDatabaseMetadata(a.ctx, a.instanceID, a.defaultDatabase)
		if err != nil {
			a.err = errors.Wrapf(err, "failed to get database metadata for database: %s", a.defaultDatabase)
			return
		}
		// Access pseudo table or table/view we do not sync, return directly.
		if databaseMetadata == nil {
			return
		}
		schemaName, name := databaseMetadata.SearchObject(searchPath, resource.Table)
		if schemaName == "" && name == "" {
			return
		}
		resource.Schema = schemaName
	}

	a.accessTables = append(a.accessTables, resource)
}

// isMixedQuery checks whether the query accesses the user table and system table at the same time.
func isMixedQuery(m base.SourceColumnSet) (bool, bool) {
	hasSystem, hasUser := false, false
	for table := range m {
		if isSystemResource(table) {
			hasSystem = true
		} else {
			hasUser = true
		}
	}

	if hasSystem && hasUser {
		return false, true
	}

	return !hasUser && hasSystem, false
}

func isSystemResource(resource base.ColumnResource) bool {
	// User can access the system table/view by name directly without database/schema name.
	// For example: `SELECT * FROM pg_database`, which will access the system table `pg_database`.
	// Additionally, user can create a table/view with the same name with system table/view and access them
	// by specify the schema name, for example:
	// `CREATE TABLE pg_database(id INT); SELECT * FROM public.pg_database;` which will access the user table `pg_database`.
	if IsSystemSchema(resource.Schema) {
		return true
	}
	if resource.Schema == "" && IsSystemView(resource.Table) {
		return true
	}
	if resource.Schema == "" && IsSystemTable(resource.Table) {
		return true
	}
	return false
}
