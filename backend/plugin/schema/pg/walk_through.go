package pg

import (
	"context"

	"google.golang.org/protobuf/proto"

	"github.com/bytebase/omni/pg/catalog"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/plugin/schema"
	"github.com/bytebase/bytebase/backend/plugin/schema/internal/omnicatalog"
	"github.com/bytebase/bytebase/backend/store/model"
)

// WalkThroughWithContext performs DDL simulation using the omni catalog.
//  1. catalog.LoadMetadata(metadata) → load existing schema into catalog
//  2. catalog.Exec(userSQL) → execute user DDL, collect per-statement Changes
//  3. Map exec errors → *storepb.Advice
//  4. Merge Changes into original metadata → FinalMetadata for downstream rules
func WalkThroughWithContext(ctx context.Context, wtCtx schema.WalkThroughContext, d *model.DatabaseMetadata, _ []base.AST) *storepb.Advice {
	if wtCtx.RawSQL == "" {
		return nil
	}

	// Step 1: Load existing schema into two catalogs — one as snapshot, one for exec.
	catBefore := catalog.New()
	if wtCtx.SessionUser != "" {
		catBefore.SetSessionUser(wtCtx.SessionUser)
	}
	if searchPath := getConfiguredSearchPath(d); len(searchPath) > 0 {
		catBefore.SetSearchPath(searchPath)
	}
	if _, err := catBefore.LoadMetadata(ctx, d.GetProto(), catalog.LoadMetadataOptions{Full: true}); err != nil {
		return omnicatalog.DDLSimulationAdvice("Failed to load schema", err)
	}

	catAfter := catBefore.Clone()

	// Step 2: Execute user SQL on catAfter.
	results, execErr := catAfter.Exec(wtCtx.RawSQL, &catalog.ExecOptions{ContinueOnError: true})
	if execErr != nil {
		return omnicatalog.DDLSimulationAdvice("DDL simulation failed", execErr)
	}

	// Step 3: Report the first exec error.
	for _, r := range results {
		if r.Error == nil {
			continue
		}
		errCode := mapSQLSTATEToCode(r.Error)
		return &storepb.Advice{
			Status:  storepb.Advice_ERROR,
			Code:    errCode.Int32(),
			Title:   "DDL simulation failed",
			Content: r.Error.Error(),
			StartPosition: &storepb.Position{
				Line: int32(r.Line),
			},
		}
	}

	// Step 4: Diff before/after catalogs and apply to original metadata.
	diff := catalog.Diff(catBefore, catAfter)
	newProto := catalog.ApplyDiffToMetadata(d.GetProto(), catBefore, catAfter, diff)
	if !proto.Equal(d.GetProto(), newProto) {
		newMetadata := model.NewDatabaseMetadata(newProto, nil, d.GetConfig(), storepb.Engine_POSTGRES, d.GetIsObjectCaseSensitive())
		d.ReplaceFrom(newMetadata)
	}

	return nil
}

// mapSQLSTATEToCode converts an omni catalog error to a bytebase error code.
func mapSQLSTATEToCode(err error) code.Code {
	catErr, ok := err.(*catalog.Error)
	if !ok {
		return code.DDLSimulationFailed
	}

	switch catErr.Code {
	// Table/relation errors
	case catalog.CodeDuplicateTable:
		return code.TableExists
	case catalog.CodeUndefinedTable:
		return code.TableNotExists

	// Column errors
	case catalog.CodeDuplicateColumn:
		return code.ColumnExists
	case catalog.CodeUndefinedColumn, catalog.CodeInvalidColumnDefinition:
		return code.ColumnNotExists

	// Schema errors
	case catalog.CodeDuplicateSchema:
		return code.RelationExists
	case catalog.CodeUndefinedSchema:
		return code.SchemaNotExists

	// Index/constraint errors
	case catalog.CodeDuplicateObject, catalog.CodeUniqueViolation:
		return code.IndexExists
	case catalog.CodeUndefinedObject:
		return code.IndexNotExists

	// FK errors
	case catalog.CodeInvalidFK, catalog.CodeForeignKeyViolation:
		return code.TableHasFK

	// PK/table definition errors
	case catalog.CodeDuplicatePKey:
		return code.PrimaryKeyExists

	// Type/expression errors
	case catalog.CodeDatatypeMismatch:
		return code.ChangeColumnType
	case catalog.CodeNotNullViolation:
		return code.ColumnCannotNull

	// Syntax
	case catalog.CodeSyntaxError:
		return code.StatementSyntaxError

	// Naming
	case catalog.CodeReservedName:
		return code.NameIsKeywordIdentifier

	default:
		return code.DDLSimulationFailed
	}
}

func getConfiguredSearchPath(d *model.DatabaseMetadata) []string {
	return formatSearchPath(d.GetConfiguredSearchPath())
}

func formatSearchPath(configured []model.PGSearchPathItem) []string {
	if len(configured) == 0 {
		return nil
	}

	searchPath := make([]string, 0, len(configured))
	for _, item := range configured {
		if item.CurrentUser {
			searchPath = append(searchPath, "$user")
			continue
		}
		if item.Schema == "" {
			continue
		}
		searchPath = append(searchPath, item.Schema)
	}
	return searchPath
}
