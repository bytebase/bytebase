package pg

import (
	"fmt"
	"strings"

	"github.com/pkg/errors"

	"github.com/bytebase/omni/pg/catalog"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	"github.com/bytebase/bytebase/backend/plugin/schema"
	"github.com/bytebase/bytebase/backend/store/model"
)

func init() {
	schema.RegisterSDLMigration(storepb.Engine_POSTGRES, pgSDLMigration)
	schema.RegisterSDLMigration(storepb.Engine_COCKROACHDB, pgSDLMigration)
	schema.RegisterSDLDropAdvices(storepb.Engine_POSTGRES, pgSDLDropAdvices)
	schema.RegisterSDLDropAdvices(storepb.Engine_COCKROACHDB, pgSDLDropAdvices)
}

// convertDatabaseSchemaToSDL converts a model.DatabaseMetadata to SDL format string.
func convertDatabaseSchemaToSDL(dbMetadata *model.DatabaseMetadata) (string, error) {
	if dbMetadata == nil {
		return "", nil
	}

	metadata := dbMetadata.GetProto()
	if metadata == nil {
		return "", nil
	}

	return getSDLFormat(metadata)
}

// buildSDLCatalogs builds the from/to catalogs for an SDL migration.
func buildSDLCatalogs(userSDLText string, currentSchema *model.DatabaseMetadata) (*catalog.Catalog, *catalog.Catalog, error) {
	fromDDL, err := convertDatabaseSchemaToSDL(currentSchema)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to convert current schema to DDL")
	}

	from, err := catalog.LoadSDL(fromDDL)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "failed to load current schema into catalog")
	}

	to, err := catalog.LoadSDL(strings.TrimSpace(userSDLText))
	if err != nil {
		return nil, nil, errors.Wrapf(err, "failed to load user SDL into catalog")
	}

	return from, to, nil
}

// filterArchiveOps filters out bbdataarchive schema changes from a migration plan.
func filterArchiveOps(plan *catalog.MigrationPlan) *catalog.MigrationPlan {
	return plan.Filter(func(op catalog.MigrationOp) bool {
		return op.SchemaName != "bbdataarchive"
	})
}

// pgSDLMigration computes the migration SQL from a user-provided SDL text and
// the current database schema using the omni catalog engine.
func pgSDLMigration(userSDLText string, currentSchema *model.DatabaseMetadata) (string, error) {
	from, to, err := buildSDLCatalogs(userSDLText, currentSchema)
	if err != nil {
		return "", err
	}

	diff := catalog.Diff(from, to)
	if diff.IsEmpty() {
		return "", nil
	}

	plan := filterArchiveOps(catalog.GenerateMigration(from, to, diff))
	return plan.SQL(), nil
}

// pgSDLDropAdvices analyzes the SDL migration plan for destructive operations
// and returns warnings.
func pgSDLDropAdvices(userSDLText string, currentSchema *model.DatabaseMetadata) ([]*storepb.Advice, error) {
	from, to, err := buildSDLCatalogs(userSDLText, currentSchema)
	if err != nil {
		return nil, err
	}

	diff := catalog.Diff(from, to)
	if diff.IsEmpty() {
		return nil, nil
	}

	plan := filterArchiveOps(catalog.GenerateMigration(from, to, diff))

	var advices []*storepb.Advice
	for _, op := range plan.Ops {
		switch op.Type {
		case catalog.OpDropSchema:
			advices = append(advices, dropAdvice(fmt.Sprintf("Dropping schema '%s' will result in data loss.", op.ObjectName)))
		case catalog.OpDropTable:
			advices = append(advices, dropAdvice(fmt.Sprintf("Dropping table '%s.%s' will result in data loss.", op.SchemaName, op.ObjectName)))
		case catalog.OpDropColumn:
			advices = append(advices, dropAdvice(fmt.Sprintf("Dropping column from table '%s.%s' will result in data loss.", op.SchemaName, op.ParentObject)))
		case catalog.OpDropView:
			advices = append(advices, dropAdvice(fmt.Sprintf("Dropping view '%s.%s' will affect dependent objects.", op.SchemaName, op.ObjectName)))
		case catalog.OpDropFunction:
			advices = append(advices, dropAdvice(fmt.Sprintf("Dropping function '%s.%s' will affect dependent objects.", op.SchemaName, op.ObjectName)))
		case catalog.OpDropSequence:
			advices = append(advices, dropAdvice(fmt.Sprintf("Dropping sequence '%s.%s' may affect auto-increment columns.", op.SchemaName, op.ObjectName)))
		case catalog.OpDropType:
			advices = append(advices, dropAdvice(fmt.Sprintf("Dropping type '%s.%s' will affect columns using this type.", op.SchemaName, op.ObjectName)))
		case catalog.OpDropTrigger:
			advices = append(advices, dropAdvice(fmt.Sprintf("Dropping trigger '%s' on '%s.%s'.", op.ObjectName, op.SchemaName, op.ParentObject)))
		case catalog.OpDropConstraint:
			advices = append(advices, dropAdvice(fmt.Sprintf("Dropping constraint from table '%s.%s'.", op.SchemaName, op.ParentObject)))
		case catalog.OpAlterFunction:
			advices = append(advices, replaceAdvice(fmt.Sprintf("Function '%s.%s' definition will be replaced.", op.SchemaName, op.ObjectName)))
		default:
		}
	}

	return advices, nil
}

func dropAdvice(content string) *storepb.Advice {
	return &storepb.Advice{
		Status:  storepb.Advice_WARNING,
		Code:    code.SDLDropOperation.Int32(),
		Title:   "DROP operation detected",
		Content: content,
	}
}

func replaceAdvice(content string) *storepb.Advice {
	return &storepb.Advice{
		Status:  storepb.Advice_WARNING,
		Code:    code.SDLReplaceOperation.Int32(),
		Title:   "CREATE OR REPLACE operation detected",
		Content: content,
	}
}
