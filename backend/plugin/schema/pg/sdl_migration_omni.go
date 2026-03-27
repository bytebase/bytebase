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
	schema.RegisterDiffMigration(storepb.Engine_POSTGRES, pgSchemaDiffMigration)
	schema.RegisterDiffMigration(storepb.Engine_COCKROACHDB, pgSchemaDiffMigration)
	schema.RegisterDiffSchemaTextMigration(storepb.Engine_POSTGRES, pgDiffSchemaTextMigration)
	schema.RegisterDiffSchemaTextMigration(storepb.Engine_COCKROACHDB, pgDiffSchemaTextMigration)
	schema.RegisterDiffSDLMigration(storepb.Engine_POSTGRES, pgDiffSDLMigration)
	schema.RegisterDiffSDLMigration(storepb.Engine_COCKROACHDB, pgDiffSDLMigration)
}

// catalogFromMetadata builds an omni Catalog from database metadata.
func catalogFromMetadata(meta *model.DatabaseMetadata) (*catalog.Catalog, error) {
	if meta == nil {
		return catalog.LoadSDL("")
	}
	proto := meta.GetProto()
	if proto == nil {
		return catalog.LoadSDL("")
	}
	ddl, err := getSDLFormat(proto)
	if err != nil {
		return nil, errors.Wrap(err, "failed to convert metadata to DDL")
	}
	return catalog.LoadSDL(ddl)
}

// pgDiffSDLMigration computes migration SQL between two SDL texts using omni.
func pgDiffSDLMigration(sourceSDL, targetSDL string) (string, error) {
	from, err := catalog.LoadSDL(strings.TrimSpace(sourceSDL))
	if err != nil {
		return "", errors.Wrap(err, "failed to load source SDL")
	}
	to, err := catalog.LoadSDL(strings.TrimSpace(targetSDL))
	if err != nil {
		return "", errors.Wrap(err, "failed to load target SDL")
	}
	diff := catalog.Diff(from, to)
	if diff.IsEmpty() {
		return "", nil
	}
	plan := filterArchiveOps(catalog.GenerateMigration(from, to, diff))
	return plan.SQL(), nil
}

// pgDiffSchemaTextMigration computes migration SQL from source metadata to target schema text.
func pgDiffSchemaTextMigration(sourceSchema *model.DatabaseMetadata, targetSchemaText string) (string, error) {
	from, err := catalogFromMetadata(sourceSchema)
	if err != nil {
		return "", errors.Wrap(err, "failed to load source schema")
	}
	to, err := catalog.LoadSDL(strings.TrimSpace(targetSchemaText))
	if err != nil {
		return "", errors.Wrap(err, "failed to load target schema text")
	}
	diff := catalog.Diff(from, to)
	if diff.IsEmpty() {
		return "", nil
	}
	plan := filterArchiveOps(catalog.GenerateMigration(from, to, diff))
	return plan.SQL(), nil
}

// pgSchemaDiffMigration computes migration SQL between two metadata states using omni.
func pgSchemaDiffMigration(oldSchema, newSchema *model.DatabaseMetadata) (string, error) {
	from, err := catalogFromMetadata(oldSchema)
	if err != nil {
		return "", errors.Wrap(err, "failed to load source schema")
	}
	to, err := catalogFromMetadata(newSchema)
	if err != nil {
		return "", errors.Wrap(err, "failed to load target schema")
	}
	diff := catalog.Diff(from, to)
	if diff.IsEmpty() {
		return "", nil
	}
	plan := filterArchiveOps(catalog.GenerateMigration(from, to, diff))
	return plan.SQL(), nil
}

// buildSDLCatalogs builds the from/to catalogs for an SDL migration.
func buildSDLCatalogs(userSDLText string, currentSchema *model.DatabaseMetadata) (*catalog.Catalog, *catalog.Catalog, error) {
	var fromDDL string
	if currentSchema != nil {
		proto := currentSchema.GetProto()
		if proto != nil {
			var ddlErr error
			fromDDL, ddlErr = getSDLFormat(proto)
			if ddlErr != nil {
				return nil, nil, errors.Wrap(ddlErr, "failed to convert current schema to DDL")
			}
		}
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
