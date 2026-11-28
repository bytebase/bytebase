package schema

import (
	"context"

	omnianalysis "github.com/bytebase/omni/analysis"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/store/model"
)

var walkThroughCodes = map[omnianalysis.WalkThroughCode]code.Code{
	omnianalysis.WalkThroughDDLSimulationFailed:                  code.DDLSimulationFailed,
	omnianalysis.WalkThroughNotCurrentDatabase:                   code.NotCurrentDatabase,
	omnianalysis.WalkThroughDatabaseIsDeleted:                    code.DatabaseIsDeleted,
	omnianalysis.WalkThroughCreateTableAs:                        code.StatementCreateTableAs,
	omnianalysis.WalkThroughTableExists:                          code.TableExists,
	omnianalysis.WalkThroughTableNotExists:                       code.TableNotExists,
	omnianalysis.WalkThroughColumnExists:                         code.ColumnExists,
	omnianalysis.WalkThroughColumnNotExists:                      code.ColumnNotExists,
	omnianalysis.WalkThroughInvalidColumnDefault:                 code.InvalidColumnDefault,
	omnianalysis.WalkThroughIndexExists:                          code.IndexExists,
	omnianalysis.WalkThroughIndexNotExists:                       code.IndexNotExists,
	omnianalysis.WalkThroughPrimaryKeyExists:                     code.PrimaryKeyExists,
	omnianalysis.WalkThroughSpatialIndexKeyNullable:              code.SpatialIndexKeyNullable,
	omnianalysis.WalkThroughAutoIncrementExists:                  code.AutoIncrementExists,
	omnianalysis.WalkThroughOnUpdateColumnNotDatetimeOrTimestamp: code.OnUpdateColumnNotDatetimeOrTimestamp,
	omnianalysis.WalkThroughTableHasFK:                           code.TableHasFK,
	omnianalysis.WalkThroughRelationExists:                       code.RelationExists,
	omnianalysis.WalkThroughSchemaNotExists:                      code.SchemaNotExists,
	omnianalysis.WalkThroughChangeColumnType:                     code.ChangeColumnType,
	omnianalysis.WalkThroughColumnCannotNull:                     code.ColumnCannotNull,
	omnianalysis.WalkThroughSyntaxError:                          code.StatementSyntaxError,
	omnianalysis.WalkThroughNameIsKeywordIdentifier:              code.NameIsKeywordIdentifier,
}

// OmniWalkThrough returns the registry's WalkThrough for an omni engine's
// walk-through. It replaces d with the schema after the script when the
// walk-through returns one.
func OmniWalkThrough(engine storepb.Engine, walker omnianalysis.WalkThrougher) func(context.Context, WalkThroughContext, *model.DatabaseMetadata, []base.AST) *storepb.Advice {
	return func(ctx context.Context, wtCtx WalkThroughContext, d *model.DatabaseMetadata, asts []base.AST) *storepb.Advice {
		result := walker.WalkThrough(ctx, omnianalysis.WalkThroughRequest{
			Database: &omnianalysis.Database{
				Name:                d.DatabaseName(),
				Metadata:            d.GetProto(),
				ObjectCaseSensitive: d.GetIsObjectCaseSensitive(),
			},
			Script:      wtCtx.RawSQL,
			SessionUser: wtCtx.SessionUser,
			SearchPath:  FormatSearchPath(d.GetConfiguredSearchPath()),
		}, base.OmniParsedStatements(asts))
		if result.Database != nil {
			d.ReplaceFrom(model.NewDatabaseMetadata(result.Database.Metadata, nil, d.GetConfig(), engine, result.Database.ObjectCaseSensitive))
		}
		if result.Finding == nil {
			return nil
		}
		return walkThroughAdvice(result.Finding, asts)
	}
}

// walkThroughAdvice converts a finding; one at a statement starts where that
// statement's AST does, at line 0 when the AST has no position.
func walkThroughAdvice(finding *omnianalysis.WalkThroughFinding, asts []base.AST) *storepb.Advice {
	c, ok := walkThroughCodes[finding.Code]
	if !ok {
		c = code.DDLSimulationFailed
	}
	status := storepb.Advice_WARNING
	if finding.Error {
		status = storepb.Advice_ERROR
	}
	start := &storepb.Position{Line: int32(finding.Start.Line), Column: int32(finding.Start.Column)}
	if finding.Statement >= 0 {
		start = &storepb.Position{Line: 0}
		if finding.Statement < len(asts) {
			if p := asts[finding.Statement].ASTStartPosition(); p != nil {
				start = &storepb.Position{Line: p.Line, Column: p.Column}
			}
		}
	}
	return &storepb.Advice{
		Status:        status,
		Code:          c.Int32(),
		Title:         finding.Title,
		Content:       finding.Content,
		StartPosition: start,
	}
}

// FormatSearchPath writes a configured PostgreSQL search path as omni's
// catalog takes it: "$user" for the current user, skipping an empty schema.
func FormatSearchPath(configured []model.PGSearchPathItem) []string {
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
