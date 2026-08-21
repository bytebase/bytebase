package pg

import (
	"fmt"

	"github.com/pkg/errors"

	"github.com/bytebase/omni/pg/catalog"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	"github.com/bytebase/bytebase/backend/plugin/schema"
	"github.com/bytebase/bytebase/backend/store/model"
)

func init() {
	schema.RegisterDiffSDLMigration(storepb.Engine_POSTGRES, pgDiffSDLMigration)
	schema.RegisterDiffSDLMigration(storepb.Engine_COCKROACHDB, pgDiffSDLMigration)
	schema.RegisterDiffMetadataMigration(storepb.Engine_POSTGRES, pgDiffMetadataMigration)
	schema.RegisterDiffMetadataMigration(storepb.Engine_COCKROACHDB, pgDiffCockroachMetadataMigration)
	schema.RegisterSDLDropAdvices(storepb.Engine_POSTGRES, pgSDLDropAdvices)
	schema.RegisterSDLDropAdvices(storepb.Engine_COCKROACHDB, pgSDLDropAdvices)
}

// loadCatalog parses a schema text and returns a catalog reflecting it.
// LoadSDL is the canonical, dependency-aware path. LoadSQL is tried only as
// a fallback for raw dumps containing non-SDL statements (e.g. SET, SELECT)
// that LoadSDL legitimately rejects. When both fail, the LoadSDL error is
// returned because it is far more diagnostic than LoadSQL's order-dependent
// failures (e.g. "relation X does not exist" on a forward FK reference).
func loadCatalog(text string) (*catalog.Catalog, error) {
	c, sdlErr := catalog.LoadSDL(text)
	if sdlErr == nil {
		return c, nil
	}
	if c, err := catalog.LoadSQL(text); err == nil {
		return c, nil
	}
	return nil, sdlErr
}

// buildMigrationPlan loads two schema texts into catalogs, diffs them, and
// returns the filtered migration plan. Returns nil if there are no changes.
func buildMigrationPlan(sourceText, targetText string) (*catalog.MigrationPlan, error) {
	from, err := loadCatalog(sourceText)
	if err != nil {
		return nil, errors.Wrap(err, "failed to load source schema")
	}
	to, err := loadCatalog(targetText)
	if err != nil {
		return nil, errors.Wrap(err, "failed to load target schema")
	}
	diff := catalog.Diff(from, to)
	if diff.IsEmpty() {
		return nil, nil
	}
	plan := catalog.GenerateMigration(from, to, diff)
	plan = plan.Filter(func(op catalog.MigrationOp) bool {
		return op.SchemaName != "bbdataarchive"
	})
	return plan, nil
}

// pgDiffSDLMigration is the core migration function: two schema texts in, migration SQL
// out. The engine version is unused: PostgreSQL canonicalizes identically across versions.
// The session-context map is unused: PostgreSQL routines carry no per-object session
// context (it is a MySQL-only concern), so the recreate is always bare.
func pgDiffSDLMigration(sourceSDL, targetSDL string, _ string, _ *schema.SDLSessionContextMap) (string, error) {
	plan, err := buildMigrationPlan(sourceSDL, targetSDL)
	if err != nil {
		return "", err
	}
	if plan == nil {
		return "", nil
	}
	return plan.SQL(), nil
}

// pgSDLDropAdvices analyzes the SDL migration plan for destructive operations.
// engineVersion is unused: PostgreSQL canonicalizes identically across versions.
func pgSDLDropAdvices(userSDLText string, currentSchema *model.DatabaseMetadata, _ string) ([]*storepb.Advice, error) {
	sourceSDL, err := schema.MetadataToSDL(storepb.Engine_POSTGRES, currentSchema)
	if err != nil {
		return nil, err
	}
	plan, err := buildMigrationPlan(sourceSDL, userSDLText)
	if err != nil {
		return nil, err
	}
	if plan == nil {
		return nil, nil
	}

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
		case catalog.OpDropExtension:
			advices = append(advices, undeclaredExtensionAdvice(op.ObjectName))
		case catalog.OpAlterFunction:
			advices = append(advices, replaceAdvice(fmt.Sprintf("Function '%s.%s' definition will be replaced.", op.SchemaName, op.ObjectName)))
		default:
		}
	}

	return advices, nil
}

// undeclaredExtensionAdvice reports an extension that is installed on the database but
// absent from the SDL. This is an ERROR, not a warning: the catalog does not track which
// objects an extension owns, so a plan that drops the extension also drops every object
// its script materialized, and PostgreSQL refuses those ("cannot drop function armor(bytea)
// because extension pgcrypto requires it", SQLSTATE 2BP01) — the rollout can never succeed.
// Failing the check with the one-line fix beats letting the task die mid-transaction.
func undeclaredExtensionAdvice(name string) *storepb.Advice {
	return &storepb.Advice{
		Status: storepb.Advice_ERROR,
		Code:   code.SDLUndeclaredExtension.Int32(),
		Title:  "Undeclared extension",
		Content: fmt.Sprintf(
			"Extension %s is installed on the database but not declared in the SDL.\n\n"+
				"Add this line to keep it:\n"+
				"  CREATE EXTENSION IF NOT EXISTS %s;\n\n"+
				"An undeclared extension makes the differ treat the extension's own functions and "+
				"types as orphans, and PostgreSQL will not drop them individually. To remove the "+
				"extension itself, run DROP EXTENSION outside the declarative pipeline.",
			wtQuoteIdent(name), wtQuoteIdent(name))
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
