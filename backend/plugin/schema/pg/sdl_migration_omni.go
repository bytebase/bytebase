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
	schema.RegisterDiffSDLMigration(storepb.Engine_POSTGRES, pgDiffSDLMigration)
	schema.RegisterDiffSDLMigration(storepb.Engine_COCKROACHDB, pgDiffSDLMigration)
	schema.RegisterSDLDropAdvices(storepb.Engine_POSTGRES, pgSDLDropAdvices)
	schema.RegisterSDLDropAdvices(storepb.Engine_COCKROACHDB, pgSDLDropAdvices)
}

// pgDiffSDLMigration is the core migration function: two SDL texts in, migration SQL out.
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
	plan := catalog.GenerateMigration(from, to, diff)
	plan = plan.Filter(func(op catalog.MigrationOp) bool {
		return op.SchemaName != "bbdataarchive"
	})
	return plan.SQL(), nil
}

// pgSDLDropAdvices analyzes the SDL migration plan for destructive operations.
func pgSDLDropAdvices(userSDLText string, currentSchema *model.DatabaseMetadata) ([]*storepb.Advice, error) {
	sourceSDL, err := schema.MetadataToSDL(storepb.Engine_POSTGRES, currentSchema)
	if err != nil {
		return nil, err
	}
	migrationSQL, err := pgDiffSDLMigration(sourceSDL, userSDLText)
	if err != nil {
		return nil, err
	}
	if migrationSQL == "" {
		return nil, nil
	}

	// Re-run to get the plan for advice generation.
	from, _ := catalog.LoadSDL(strings.TrimSpace(sourceSDL))
	to, _ := catalog.LoadSDL(strings.TrimSpace(userSDLText))
	diff := catalog.Diff(from, to)
	plan := catalog.GenerateMigration(from, to, diff)
	plan = plan.Filter(func(op catalog.MigrationOp) bool {
		return op.SchemaName != "bbdataarchive"
	})

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
