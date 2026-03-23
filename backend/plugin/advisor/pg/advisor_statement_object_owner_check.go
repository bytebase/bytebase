package pg

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"

	"github.com/bytebase/omni/pg/ast"

	"github.com/bytebase/bytebase/backend/common/log"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	"github.com/bytebase/bytebase/backend/store/model"
)

var (
	_ advisor.Advisor = (*StatementObjectOwnerCheckAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_POSTGRES, storepb.SQLReviewRule_STATEMENT_OBJECT_OWNER_CHECK, &StatementObjectOwnerCheckAdvisor{})
}

const (
	pgDatabaseOwner = "pg_database_owner"
)

type StatementObjectOwnerCheckAdvisor struct {
}

func (*StatementObjectOwnerCheckAdvisor) Check(ctx context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	dbMetadata := model.NewDatabaseMetadata(checkCtx.DBSchema, nil, nil, storepb.Engine_POSTGRES, checkCtx.IsObjectCaseSensitive)
	currentRole := checkCtx.DBSchema.Owner
	if !checkCtx.TenantMode {
		currentRole, err = getCurrentUser(ctx, checkCtx.Driver)
		if err != nil {
			slog.Debug("Failed to get current user", log.BBError(err))
			currentRole = checkCtx.DBSchema.Owner
		}
	}

	rule := &statementObjectOwnerCheckRule{
		OmniBaseRule: OmniBaseRule{
			Level: level,
			Title: checkCtx.Rule.Type.String(),
		},
		dbMetadata:  dbMetadata,
		currentRole: currentRole,
	}

	return RunOmniRules(checkCtx.ParsedStatements, []OmniRule{rule}), nil
}

type statementObjectOwnerCheckRule struct {
	OmniBaseRule
	dbMetadata  *model.DatabaseMetadata
	currentRole string
}

// Name returns the rule name.
func (*statementObjectOwnerCheckRule) Name() string {
	return "statement.object-owner-check"
}

func (r *statementObjectOwnerCheckRule) OnStatement(node ast.Node) {
	switch n := node.(type) {
	case *ast.VariableSetStmt:
		// SET ROLE handling - no-op for now (same as ANTLR version)
	case *ast.AlterSeqStmt:
		if n.Sequence != nil {
			r.checkSchemaOwnership(omniSchemaName(n.Sequence))
		}
	case *ast.AlterTableStmt:
		if n.Relation != nil {
			schemaName := omniSchemaName(n.Relation)
			tableName := omniTableName(n.Relation)
			r.checkTableOwnership(schemaName, tableName)
		}
	case *ast.AlterTypeStmt:
		parts := omniListStrings(n.TypeName)
		var schemaName string
		if len(parts) >= 2 {
			schemaName = parts[0]
		}
		r.checkSchemaOwnership(schemaName)
	case *ast.CreateExtensionStmt:
		schemaName := defaultSchema
		if n.Options != nil {
			for _, item := range n.Options.Items {
				if elem, ok := item.(*ast.DefElem); ok && elem.Defname == "schema" {
					if s, ok := elem.Arg.(*ast.String); ok {
						schemaName = s.Str
					}
				}
			}
		}
		r.checkSchemaOwnership(schemaName)
	case *ast.CreateFunctionStmt:
		schemaName := omniExtractSchemaFromFuncname(n.Funcname)
		r.checkSchemaOwnership(schemaName)
	case *ast.IndexStmt:
		if n.Relation != nil {
			r.checkSchemaOwnership(omniSchemaName(n.Relation))
		}
	case *ast.CreateSchemaStmt:
		owner := r.dbMetadata.GetProto().GetOwner()
		if owner != r.currentRole {
			r.AddAdvice(&storepb.Advice{
				Status:  r.Level,
				Title:   r.Title,
				Content: fmt.Sprintf("Database \"%s\" is owned by \"%s\", but the current role is \"%s\".", r.dbMetadata.GetProto().GetName(), owner, r.currentRole),
				Code:    code.StatementObjectOwnerCheck.Int32(),
				StartPosition: &storepb.Position{
					Line:   r.ContentStartLine(),
					Column: 0,
				},
			})
		}
	case *ast.CreateSeqStmt:
		if n.Sequence != nil {
			r.checkSchemaOwnership(omniSchemaName(n.Sequence))
		}
	case *ast.CreateStmt:
		if n.Relation != nil {
			r.checkSchemaOwnership(omniSchemaName(n.Relation))
		}
	case *ast.CreateTrigStmt:
		if n.Relation != nil {
			r.checkSchemaOwnership(omniSchemaName(n.Relation))
		}
	case *ast.DefineStmt:
		if n.Kind == ast.OBJECT_TYPE {
			parts := omniListStrings(n.Defnames)
			var schemaName string
			if len(parts) >= 2 {
				schemaName = parts[0]
			}
			r.checkSchemaOwnership(schemaName)
		}
	case *ast.CreateTableAsStmt:
		// Handles CREATE MATERIALIZED VIEW
		if n.Objtype == ast.OBJECT_MATVIEW && n.Into != nil && n.Into.Rel != nil {
			r.checkSchemaOwnership(omniSchemaName(n.Into.Rel))
		}
	case *ast.AlterFunctionStmt:
		// DROP FUNCTION is handled by DropStmt in omni AST
		// AlterFunctionStmt handles ALTER FUNCTION
		if n.Func != nil {
			schemaName := omniExtractSchemaFromObjectWithArgs(n.Func)
			r.checkSchemaOwnership(schemaName)
		}
	case *ast.DropStmt:
		r.handleDropStmt(n)
	case *ast.RenameStmt:
		r.handleRenameStmt(n)
	default:
	}
}

func (r *statementObjectOwnerCheckRule) handleDropStmt(n *ast.DropStmt) {
	// For DROP INDEX, DROP TYPE, etc., check schema ownership via object names
	names := omniDropObjectNames(n)
	for _, parts := range names {
		var schemaName string
		if len(parts) >= 2 {
			schemaName = parts[0]
		}
		r.checkSchemaOwnership(schemaName)
	}
}

func (r *statementObjectOwnerCheckRule) handleRenameStmt(n *ast.RenameStmt) {
	switch n.RenameType {
	case ast.OBJECT_SCHEMA:
		// ALTER SCHEMA RENAME
		if n.Subname != "" {
			r.checkSchemaOwnership(n.Subname)
		}
	case ast.OBJECT_INDEX:
		// ALTER INDEX RENAME
		if n.Relation != nil {
			r.checkSchemaOwnership(omniSchemaName(n.Relation))
		}
	default:
	}
}

func (r *statementObjectOwnerCheckRule) checkSchemaOwnership(schemaName string) {
	if schemaName == "" {
		schemaName = defaultSchema
	}

	schemaMeta := r.dbMetadata.GetSchemaMetadata(schemaName)
	if schemaMeta == nil {
		return
	}

	owner := schemaMeta.GetProto().GetOwner()
	if owner == pgDatabaseOwner {
		owner = r.dbMetadata.GetProto().GetOwner()
	}
	if owner != r.currentRole {
		r.AddAdvice(&storepb.Advice{
			Status:  r.Level,
			Title:   r.Title,
			Content: fmt.Sprintf("Schema \"%s\" is owned by \"%s\", but the current role is \"%s\".", schemaName, owner, r.currentRole),
			Code:    code.StatementObjectOwnerCheck.Int32(),
			StartPosition: &storepb.Position{
				Line:   r.ContentStartLine(),
				Column: 0,
			},
		})
	}
}

func (r *statementObjectOwnerCheckRule) checkTableOwnership(schemaName, tableName string) {
	if schemaName == "" {
		schemaName = defaultSchema
	}

	schemaMeta := r.dbMetadata.GetSchemaMetadata(schemaName)
	if schemaMeta == nil {
		return
	}

	tableMeta := schemaMeta.GetTable(tableName)
	if tableMeta == nil {
		return
	}

	owner := tableMeta.GetOwner()
	if owner == pgDatabaseOwner {
		owner = r.dbMetadata.GetProto().GetOwner()
	}
	if owner != r.currentRole {
		r.AddAdvice(&storepb.Advice{
			Status:  r.Level,
			Title:   r.Title,
			Content: fmt.Sprintf("Table \"%s\" is owned by \"%s\", but the current role is \"%s\".", tableName, owner, r.currentRole),
			Code:    code.StatementObjectOwnerCheck.Int32(),
			StartPosition: &storepb.Position{
				Line:   r.ContentStartLine(),
				Column: 0,
			},
		})
	}
}

// omniExtractSchemaFromFuncname extracts the schema name from a function's qualified name list.
func omniExtractSchemaFromFuncname(funcname *ast.List) string {
	parts := omniListStrings(funcname)
	if len(parts) >= 2 {
		return parts[0]
	}
	return ""
}

// omniExtractSchemaFromObjectWithArgs extracts the schema from an ObjectWithArgs node.
func omniExtractSchemaFromObjectWithArgs(obj *ast.ObjectWithArgs) string {
	if obj == nil || obj.Objname == nil {
		return ""
	}
	parts := omniListStrings(obj.Objname)
	if len(parts) >= 2 {
		return parts[0]
	}
	return ""
}

func getCurrentUser(ctx context.Context, driver *sql.DB) (string, error) {
	var user string
	err := driver.QueryRowContext(ctx, "SELECT CURRENT_USER").Scan(&user)
	if err != nil {
		return "", err
	}
	return user, nil
}
