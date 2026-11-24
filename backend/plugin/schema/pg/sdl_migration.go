package pg

import (
	"fmt"

	"github.com/bytebase/omni/pg/catalog"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/schema"
	"github.com/bytebase/bytebase/backend/plugin/schema/internal/omnicatalog"
	"github.com/bytebase/bytebase/backend/store/model"
)

func init() {
	schema.Register(schema.Engine{
		DiffSDLMigration:      pgDiffSDLMigration,
		DiffMetadataMigration: diffMetadataMigration,
		SDLDropAdvices:        pgSDLDropAdvices,
	}, storepb.Engine_POSTGRES, storepb.Engine_COCKROACHDB)
}

func buildMigrationPlan(sourceText, targetText string) (*catalog.MigrationPlan, error) {
	return catalog.BuildSDLMigrationPlan(sourceText, targetText, catalog.SDLMigrationOptions{
		SkipSchemas: []string{common.BackupDatabaseNameOfEngine(storepb.Engine_POSTGRES)},
	})
}

func pgDiffSDLMigration(sourceSDL, targetSDL, _ string, _ *schema.SDLSessionContextMap) (string, error) {
	plan, err := buildMigrationPlan(sourceSDL, targetSDL)
	if err != nil || plan == nil {
		return "", err
	}
	return plan.SQL(), nil
}

func pgSDLDropAdvices(userSDLText string, currentSchema *model.DatabaseMetadata, _ string) ([]*storepb.Advice, error) {
	return omnicatalog.PlanSDLDropAdvices(
		storepb.Engine_POSTGRES,
		userSDLText,
		currentSchema,
		buildMigrationPlan,
		func(plan *catalog.MigrationPlan) []*storepb.Advice {
			var advices []*storepb.Advice
			for _, op := range plan.Ops {
				switch op.Type {
				case catalog.OpDropSchema:
					advices = append(advices, omnicatalog.SDLDropAdvice(fmt.Sprintf("Dropping schema '%s' will result in data loss.", op.ObjectName)))
				case catalog.OpDropTable:
					advices = append(advices, omnicatalog.SDLDropAdvice(fmt.Sprintf("Dropping table '%s.%s' will result in data loss.", op.SchemaName, op.ObjectName)))
				case catalog.OpDropColumn:
					advices = append(advices, omnicatalog.SDLDropAdvice(fmt.Sprintf("Dropping column from table '%s.%s' will result in data loss.", op.SchemaName, op.ParentObject)))
				case catalog.OpDropView:
					advices = append(advices, omnicatalog.SDLDropAdvice(fmt.Sprintf("Dropping view '%s.%s' will affect dependent objects.", op.SchemaName, op.ObjectName)))
				case catalog.OpDropFunction:
					advices = append(advices, omnicatalog.SDLDropAdvice(fmt.Sprintf("Dropping function '%s.%s' will affect dependent objects.", op.SchemaName, op.ObjectName)))
				case catalog.OpDropSequence:
					advices = append(advices, omnicatalog.SDLDropAdvice(fmt.Sprintf("Dropping sequence '%s.%s' may affect auto-increment columns.", op.SchemaName, op.ObjectName)))
				case catalog.OpDropType:
					advices = append(advices, omnicatalog.SDLDropAdvice(fmt.Sprintf("Dropping type '%s.%s' will affect columns using this type.", op.SchemaName, op.ObjectName)))
				case catalog.OpDropTrigger:
					advices = append(advices, omnicatalog.SDLDropAdvice(fmt.Sprintf("Dropping trigger '%s' on '%s.%s'.", op.ObjectName, op.SchemaName, op.ParentObject)))
				case catalog.OpDropConstraint:
					advices = append(advices, omnicatalog.SDLDropAdvice(fmt.Sprintf("Dropping constraint from table '%s.%s'.", op.SchemaName, op.ParentObject)))
				case catalog.OpAlterFunction:
					advices = append(advices, omnicatalog.SDLReplaceAdvice(fmt.Sprintf("Function '%s.%s' definition will be replaced.", op.SchemaName, op.ObjectName)))
				default:
				}
			}
			return advices
		},
	)
}
