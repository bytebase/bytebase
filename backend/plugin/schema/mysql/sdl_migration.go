package mysql

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/pkg/errors"

	"github.com/bytebase/omni/mysql/catalog"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	"github.com/bytebase/bytebase/backend/plugin/schema"
	"github.com/bytebase/bytebase/backend/plugin/schema/internal/omnicatalog"
	"github.com/bytebase/bytebase/backend/store/model"
)

// sdlDatabaseContext is the synthetic database both SDL inputs are loaded under,
// matching the metadata diff so unqualified DDL resolves consistently.
const sdlDatabaseContext = omnicatalog.Database

// unsupportedSDLVersion returns why the declarative path refuses a MySQL server
// older than 8.0 (omni canonicalizes against the 8.0 stored form), or "" when accepted.
func unsupportedSDLVersion(engineVersion string) string {
	version := strings.TrimSpace(engineVersion)
	major, _, _ := strings.Cut(version, ".")
	if n, err := strconv.Atoi(major); err == nil && n < 8 {
		return fmt.Sprintf("declarative schema change requires MySQL 8.0 or later, but this instance runs MySQL %s", version)
	}
	return ""
}

func buildMigrationPlan(sourceText, targetText string, sessionCtx *schema.SDLSessionContextMap) (*catalog.MigrationPlan, error) {
	return catalog.BuildSDLMigrationPlan(sourceText, targetText, catalog.SDLMigrationOptions{
		Database:             sdlDatabaseContext,
		SourceSessionContext: toCatalogSessionContextMap(sessionCtx),
	})
}

func toCatalogSessionContextMap(m *schema.SDLSessionContextMap) *catalog.SessionContextMap {
	if m == nil {
		return nil
	}
	conv := func(src map[string]schema.SDLSessionContext) map[string]catalog.SessionContext {
		if len(src) == 0 {
			return nil
		}
		dst := make(map[string]catalog.SessionContext, len(src))
		for name, c := range src {
			dst[name] = catalog.SessionContext{
				SQLMode:             c.SQLMode,
				CharacterSetClient:  c.CharacterSetClient,
				CollationConnection: c.CollationConnection,
				TimeZone:            c.TimeZone,
			}
		}
		return dst
	}
	return &catalog.SessionContextMap{
		Functions:  conv(m.Functions),
		Procedures: conv(m.Procedures),
		Triggers:   conv(m.Triggers),
		Events:     conv(m.Events),
	}
}

func mysqlDiffSDLMigrationWithContext(sourceSDL, targetSDL, engineVersion string, sessionCtx *schema.SDLSessionContextMap) (string, error) {
	if reason := unsupportedSDLVersion(engineVersion); reason != "" {
		return "", errors.New(reason)
	}
	plan, err := buildMigrationPlan(sourceSDL, targetSDL, sessionCtx)
	if err != nil || plan == nil {
		return "", err
	}
	return catalog.StripDatabaseContext(plan.SQL(), sdlDatabaseContext), nil
}

// mysqlTableDropMessages maps table-scoped destructive ops whose omni MigrationOp
// carries the table name in ParentObject and the sub-object name only in op.SQL.
var mysqlTableDropMessages = map[catalog.MigrationOpType]string{
	catalog.OpDropColumn:     "Dropping a column from table '%s' will result in data loss. (%s)",
	catalog.OpModifyColumn:   "Modifying a column on table '%s' may result in data loss. (%s)",
	catalog.OpDropIndex:      "Dropping an index on table '%s'. (%s)",
	catalog.OpDropConstraint: "Dropping a constraint on table '%s'. (%s)",
	catalog.OpDropForeignKey: "Dropping a foreign key on table '%s'. (%s)",
	catalog.OpDropCheck:      "Dropping a check constraint on table '%s'. (%s)",
}

var mysqlDropToCreate = map[catalog.MigrationOpType]catalog.MigrationOpType{
	catalog.OpDropView:      catalog.OpCreateView,
	catalog.OpDropFunction:  catalog.OpCreateFunction,
	catalog.OpDropProcedure: catalog.OpCreateProcedure,
	catalog.OpDropTrigger:   catalog.OpCreateTrigger,
	catalog.OpDropEvent:     catalog.OpCreateEvent,
}

var mysqlReplaceLabels = map[catalog.MigrationOpType]string{
	catalog.OpCreateView:      "View",
	catalog.OpCreateFunction:  "Function",
	catalog.OpCreateProcedure: "Procedure",
	catalog.OpCreateTrigger:   "Trigger",
	catalog.OpCreateEvent:     "Event",
}

type opNameKey struct {
	op   catalog.MigrationOpType
	name string
}

// mysqlSDLDropAdvices emits a WARNING advice for each destructive or in-place-replace
// operation in the SDL migration plan, or one ERROR advice for MySQL < 8.0.
func mysqlSDLDropAdvices(userSDLText string, currentSchema *model.DatabaseMetadata, engineVersion string) ([]*storepb.Advice, error) {
	if reason := unsupportedSDLVersion(engineVersion); reason != "" {
		return []*storepb.Advice{{
			Status:  storepb.Advice_ERROR,
			Code:    code.Unsupported.Int32(),
			Title:   "MySQL version not supported",
			Content: reason,
		}}, nil
	}
	return omnicatalog.PlanSDLDropAdvices(
		storepb.Engine_MYSQL,
		userSDLText,
		currentSchema,
		func(sourceSDL, targetSDL string) (*catalog.MigrationPlan, error) {
			return buildMigrationPlan(sourceSDL, targetSDL, nil)
		},
		func(plan *catalog.MigrationPlan) []*storepb.Advice {
			return collectMySQLSDLDropAdvices(plan, currentSchema)
		},
	)
}

// collectMySQLSDLDropAdvices classifies plan ops into DROP vs REPLACE warnings.
// Omni renders view/event redefinitions as a lone OpCreate* and routine/trigger
// redefinitions as a paired OpDrop* + OpCreate*, so we cross-reference per-kind
// drop/create name sets (case-insensitive) and currentSchema.
func collectMySQLSDLDropAdvices(plan *catalog.MigrationPlan, currentSchema *model.DatabaseMetadata) []*storepb.Advice {
	dropped := make(map[opNameKey]bool)
	created := make(map[opNameKey]bool)
	for _, op := range plan.Ops {
		if createOp, ok := mysqlDropToCreate[op.Type]; ok {
			dropped[opNameKey{createOp, strings.ToLower(op.ObjectName)}] = true
		}
		if _, ok := mysqlReplaceLabels[op.Type]; ok {
			created[opNameKey{op.Type, strings.ToLower(op.ObjectName)}] = true
		}
	}

	var advices []*storepb.Advice
	for _, op := range plan.Ops {
		if op.Type == catalog.OpDropTable {
			advices = append(advices, omnicatalog.SDLDropAdvice(fmt.Sprintf("Dropping table '%s' will result in data loss.", op.ObjectName)))
			continue
		}
		if format, ok := mysqlTableDropMessages[op.Type]; ok {
			advices = append(advices, omnicatalog.SDLDropAdvice(fmt.Sprintf(format, op.ParentObject, catalog.StripDatabaseContext(op.SQL, sdlDatabaseContext))))
			continue
		}
		if createOp, ok := mysqlDropToCreate[op.Type]; ok {
			if !created[opNameKey{createOp, strings.ToLower(op.ObjectName)}] {
				advices = append(advices, omnicatalog.SDLDropAdvice(mysqlDropObjectMessage(op)))
			}
			continue
		}
		if label, ok := mysqlReplaceLabels[op.Type]; ok {
			if dropped[opNameKey{op.Type, strings.ToLower(op.ObjectName)}] || objectExistsInSchema(currentSchema, op.Type, op.ObjectName) {
				advices = append(advices, omnicatalog.SDLReplaceAdvice(fmt.Sprintf("%s '%s' definition will be replaced.", label, op.ObjectName)))
			}
		}
	}
	return advices
}

func mysqlDropObjectMessage(op catalog.MigrationOp) string {
	switch op.Type {
	case catalog.OpDropView:
		return fmt.Sprintf("Dropping view '%s' will affect dependent objects.", op.ObjectName)
	case catalog.OpDropFunction:
		return fmt.Sprintf("Dropping function '%s' will affect dependent objects.", op.ObjectName)
	case catalog.OpDropProcedure:
		return fmt.Sprintf("Dropping procedure '%s' will affect dependent objects.", op.ObjectName)
	case catalog.OpDropTrigger:
		return fmt.Sprintf("Dropping trigger '%s' on table '%s'.", op.ObjectName, op.ParentObject)
	case catalog.OpDropEvent:
		return fmt.Sprintf("Dropping event '%s'.", op.ObjectName)
	default:
		return ""
	}
}

func objectExistsInSchema(currentSchema *model.DatabaseMetadata, createOp catalog.MigrationOpType, name string) bool {
	if currentSchema == nil {
		return false
	}
	sm := currentSchema.GetSchemaMetadata("")
	if sm == nil {
		return false
	}
	switch createOp {
	case catalog.OpCreateView:
		return sm.GetView(name) != nil
	case catalog.OpCreateFunction:
		return sm.GetFunction(name) != nil
	case catalog.OpCreateProcedure:
		return sm.GetProcedure(name) != nil
	case catalog.OpCreateTrigger:
		for _, table := range sm.GetProto().GetTables() {
			for _, trigger := range table.GetTriggers() {
				if strings.EqualFold(trigger.GetName(), name) {
					return true
				}
			}
		}
		return false
	case catalog.OpCreateEvent:
		for _, event := range sm.GetProto().GetEvents() {
			if strings.EqualFold(event.GetName(), name) {
				return true
			}
		}
		return false
	default:
		return false
	}
}
