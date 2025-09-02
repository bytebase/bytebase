package pg

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/log"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/parser/pg/legacy/ast"
	"github.com/bytebase/bytebase/backend/store/model"
)

var (
	_ advisor.Advisor = (*StatementObjectOwnerCheckAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_POSTGRES, advisor.SchemaRuleStatementObjectOwnerCheck, &StatementObjectOwnerCheckAdvisor{})
}

const (
	pgDatabaseOwner = "pg_database_owner"
)

type StatementObjectOwnerCheckAdvisor struct {
}

func (*StatementObjectOwnerCheckAdvisor) Check(ctx context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	var adviceList []*storepb.Advice
	stmtList, ok := checkCtx.AST.([]ast.Node)
	if !ok {
		return nil, errors.Errorf("failed to convert to Node")
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}
	title := string(checkCtx.Rule.Type)

	dbMetadata := model.NewDatabaseMetadata(checkCtx.DBSchema, checkCtx.IsObjectCaseSensitive, true /* IsDetailCaseSensitive */)
	currentRole := checkCtx.DBSchema.Owner
	defaultSchema := "public"
	if !checkCtx.UsePostgresDatabaseOwner {
		currentRole, err = getCurrentUser(ctx, checkCtx.Driver)
		if err != nil {
			slog.Debug("Failed to get current user", log.BBError(err))
			currentRole = checkCtx.DBSchema.Owner
		}
	}

	for _, stmt := range stmtList {
		switch n := stmt.(type) {
		case *ast.VariableSetStmt:
			if n.Name == "role" {
				currentRole = n.GetRoleName()
			}
		case *ast.AlterSequenceStmt:
			// todo: use sequence owner instead of schema owner
			if n.Name == nil {
				continue
			}
			schemaName := n.Name.Schema
			if schemaName == "" {
				schemaName = defaultSchema
			}
			schemaMeta := dbMetadata.GetSchema(schemaName)
			if schemaMeta == nil {
				continue
			}
			owner := schemaMeta.GetOwner()
			if owner == pgDatabaseOwner {
				owner = dbMetadata.GetOwner()
			}
			if owner != currentRole {
				adviceList = append(adviceList, &storepb.Advice{
					Status:        level,
					Title:         title,
					Content:       fmt.Sprintf("Sequence \"%s\" is owned by \"%s\", but the current role is \"%s\".", n.Name.Name, owner, currentRole),
					Code:          advisor.StatementObjectOwnerCheck.Int32(),
					StartPosition: common.ConvertPGParserLineToPosition(stmt.LastLine()),
				})
			}
		case *ast.AlterTableStmt:
			if n.Table == nil {
				continue
			}
			schemaName := n.Table.Schema
			if schemaName == "" {
				schemaName = defaultSchema
			}
			schemaMeta := dbMetadata.GetSchema(schemaName)
			if schemaMeta == nil {
				continue
			}
			tableMeta := schemaMeta.GetTable(n.Table.Name)
			if tableMeta == nil {
				continue
			}
			owner := tableMeta.GetOwner()
			if owner == pgDatabaseOwner {
				owner = dbMetadata.GetOwner()
			}
			if owner != currentRole {
				adviceList = append(adviceList, &storepb.Advice{
					Status:        level,
					Title:         title,
					Content:       fmt.Sprintf("Table \"%s\" is owned by \"%s\", but the current role is \"%s\".", n.Table.Name, owner, currentRole),
					Code:          advisor.StatementObjectOwnerCheck.Int32(),
					StartPosition: common.ConvertPGParserLineToPosition(stmt.LastLine()),
				})
			}
		case *ast.AlterTypeStmt:
			// todo: use type owner instead of schema owner
			if n.Type == nil {
				continue
			}
			schemaName := n.Type.Schema
			if schemaName == "" {
				schemaName = defaultSchema
			}
			schemaMeta := dbMetadata.GetSchema(schemaName)
			if schemaMeta == nil {
				continue
			}
			owner := schemaMeta.GetOwner()
			if owner == pgDatabaseOwner {
				owner = dbMetadata.GetOwner()
			}
			if owner != currentRole {
				adviceList = append(adviceList, &storepb.Advice{
					Status:        level,
					Title:         title,
					Content:       fmt.Sprintf("Type \"%s\" is owned by \"%s\", but the current role is \"%s\".", n.Type.Name, owner, currentRole),
					Code:          advisor.StatementObjectOwnerCheck.Int32(),
					StartPosition: common.ConvertPGParserLineToPosition(stmt.LastLine()),
				})
			}
		case *ast.CreateExtensionStmt:
			schemaName := n.Schema
			if schemaName == "" {
				schemaName = defaultSchema
			}
			schemaMeta := dbMetadata.GetSchema(schemaName)
			if schemaMeta == nil {
				continue
			}
			owner := schemaMeta.GetOwner()
			if owner == pgDatabaseOwner {
				owner = dbMetadata.GetOwner()
			}
			if owner != currentRole {
				adviceList = append(adviceList, &storepb.Advice{
					Status:        level,
					Title:         title,
					Content:       fmt.Sprintf("Schema \"%s\" is owned by \"%s\", but the current role is \"%s\".", schemaName, owner, currentRole),
					Code:          advisor.StatementObjectOwnerCheck.Int32(),
					StartPosition: common.ConvertPGParserLineToPosition(stmt.LastLine()),
				})
			}
		case *ast.CreateFunctionStmt:
			if n.Function == nil {
				continue
			}
			schemaName := n.Function.Schema
			if schemaName == "" {
				schemaName = defaultSchema
			}
			schemaMeta := dbMetadata.GetSchema(schemaName)
			if schemaMeta == nil {
				continue
			}
			owner := schemaMeta.GetOwner()
			if owner == pgDatabaseOwner {
				owner = dbMetadata.GetOwner()
			}
			if owner != currentRole {
				adviceList = append(adviceList, &storepb.Advice{
					Status:        level,
					Title:         title,
					Content:       fmt.Sprintf("Schema \"%s\" is owned by \"%s\", but the current role is \"%s\".", schemaName, owner, currentRole),
					Code:          advisor.StatementObjectOwnerCheck.Int32(),
					StartPosition: common.ConvertPGParserLineToPosition(stmt.LastLine()),
				})
			}
		case *ast.CreateIndexStmt:
			if n.Index == nil || n.Index.Table == nil {
				continue
			}
			schemaName := n.Index.Table.Schema
			if schemaName == "" {
				schemaName = defaultSchema
			}
			schemaMeta := dbMetadata.GetSchema(schemaName)
			if schemaMeta == nil {
				continue
			}
			owner := schemaMeta.GetOwner()
			if owner == pgDatabaseOwner {
				owner = dbMetadata.GetOwner()
			}
			if owner != currentRole {
				adviceList = append(adviceList, &storepb.Advice{
					Status:        level,
					Title:         title,
					Content:       fmt.Sprintf("Schema \"%s\" is owned by \"%s\", but the current role is \"%s\".", schemaName, owner, currentRole),
					Code:          advisor.StatementObjectOwnerCheck.Int32(),
					StartPosition: common.ConvertPGParserLineToPosition(stmt.LastLine()),
				})
			}
		case *ast.CreateSchemaStmt:
			owner := dbMetadata.GetOwner()
			if owner != currentRole {
				adviceList = append(adviceList, &storepb.Advice{
					Status:        level,
					Title:         title,
					Content:       fmt.Sprintf("Database \"%s\" is owned by \"%s\", but the current role is \"%s\".", dbMetadata.GetName(), owner, currentRole),
					Code:          advisor.StatementObjectOwnerCheck.Int32(),
					StartPosition: common.ConvertPGParserLineToPosition(stmt.LastLine()),
				})
			}
		case *ast.CreateSequenceStmt:
			if n.SequenceDef.SequenceName == nil {
				continue
			}
			schemaName := n.SequenceDef.SequenceName.Schema
			if schemaName == "" {
				schemaName = defaultSchema
			}
			schemaMeta := dbMetadata.GetSchema(schemaName)
			if schemaMeta == nil {
				continue
			}
			owner := schemaMeta.GetOwner()
			if owner == pgDatabaseOwner {
				owner = dbMetadata.GetOwner()
			}
			if owner != currentRole {
				adviceList = append(adviceList, &storepb.Advice{
					Status:        level,
					Title:         title,
					Content:       fmt.Sprintf("Schema \"%s\" is owned by \"%s\", but the current role is \"%s\".", schemaName, owner, currentRole),
					Code:          advisor.StatementObjectOwnerCheck.Int32(),
					StartPosition: common.ConvertPGParserLineToPosition(stmt.LastLine()),
				})
			}
		case *ast.CreateTableStmt:
			if n.Name == nil {
				continue
			}
			schemaName := n.Name.Schema
			if schemaName == "" {
				schemaName = defaultSchema
			}
			schemaMeta := dbMetadata.GetSchema(schemaName)
			if schemaMeta == nil {
				continue
			}
			owner := schemaMeta.GetOwner()
			if owner == pgDatabaseOwner {
				owner = dbMetadata.GetOwner()
			}
			if owner != currentRole {
				adviceList = append(adviceList, &storepb.Advice{
					Status:        level,
					Title:         title,
					Content:       fmt.Sprintf("Schema \"%s\" is owned by \"%s\", but the current role is \"%s\".", schemaName, owner, currentRole),
					Code:          advisor.StatementObjectOwnerCheck.Int32(),
					StartPosition: common.ConvertPGParserLineToPosition(stmt.LastLine()),
				})
			}
		case *ast.CreateTriggerStmt:
			if n.Trigger == nil || n.Trigger.Table == nil {
				continue
			}
			schemaName := n.Trigger.Table.Schema
			if schemaName == "" {
				schemaName = defaultSchema
			}
			schemaMeta := dbMetadata.GetSchema(schemaName)
			if schemaMeta == nil {
				continue
			}
			owner := schemaMeta.GetOwner()
			if owner == pgDatabaseOwner {
				owner = dbMetadata.GetOwner()
			}
			if owner != currentRole {
				adviceList = append(adviceList, &storepb.Advice{
					Status:        level,
					Title:         title,
					Content:       fmt.Sprintf("Schema \"%s\" is owned by \"%s\", but the current role is \"%s\".", schemaName, owner, currentRole),
					Code:          advisor.StatementObjectOwnerCheck.Int32(),
					StartPosition: common.ConvertPGParserLineToPosition(stmt.LastLine()),
				})
			}
		case *ast.CreateTypeStmt:
			if n.Type == nil || n.Type.TypeName() == nil {
				continue
			}
			schemaName := n.Type.TypeName().Schema
			if schemaName == "" {
				schemaName = defaultSchema
			}
			schemaMeta := dbMetadata.GetSchema(schemaName)
			if schemaMeta == nil {
				continue
			}
			owner := schemaMeta.GetOwner()
			if owner == pgDatabaseOwner {
				owner = dbMetadata.GetOwner()
			}
			if owner != currentRole {
				adviceList = append(adviceList, &storepb.Advice{
					Status:        level,
					Title:         title,
					Content:       fmt.Sprintf("Schema \"%s\" is owned by \"%s\", but the current role is \"%s\".", schemaName, owner, currentRole),
					Code:          advisor.StatementObjectOwnerCheck.Int32(),
					StartPosition: common.ConvertPGParserLineToPosition(stmt.LastLine()),
				})
			}
		case *ast.CreateViewStmt:
			if n.Name == nil {
				continue
			}
			schemaName := n.Name.Schema
			if schemaName == "" {
				schemaName = defaultSchema
			}
			schemaMeta := dbMetadata.GetSchema(schemaName)
			if schemaMeta == nil {
				continue
			}
			owner := schemaMeta.GetOwner()
			if owner == pgDatabaseOwner {
				owner = dbMetadata.GetOwner()
			}
			if owner != currentRole {
				adviceList = append(adviceList, &storepb.Advice{
					Status:        level,
					Title:         title,
					Content:       fmt.Sprintf("Schema \"%s\" is owned by \"%s\", but the current role is \"%s\".", schemaName, owner, currentRole),
					Code:          advisor.StatementObjectOwnerCheck.Int32(),
					StartPosition: common.ConvertPGParserLineToPosition(stmt.LastLine()),
				})
			}
		case *ast.DropFunctionStmt:
			for _, funcName := range n.FunctionList {
				if funcName == nil {
					continue
				}
				schemaName := funcName.Schema
				if schemaName == "" {
					schemaName = defaultSchema
				}
				schemaMeta := dbMetadata.GetSchema(schemaName)
				if schemaMeta == nil {
					continue
				}
				owner := schemaMeta.GetOwner()
				if owner == pgDatabaseOwner {
					owner = dbMetadata.GetOwner()
				}
				if owner != currentRole {
					adviceList = append(adviceList, &storepb.Advice{
						Status:        level,
						Title:         title,
						Content:       fmt.Sprintf("Schema \"%s\" is owned by \"%s\", but the current role is \"%s\".", schemaName, owner, currentRole),
						Code:          advisor.StatementObjectOwnerCheck.Int32(),
						StartPosition: common.ConvertPGParserLineToPosition(stmt.LastLine()),
					})
				}
			}
		case *ast.DropIndexStmt:
			for _, indexDef := range n.IndexList {
				if indexDef == nil {
					continue
				}
				schemaName := defaultSchema
				if indexDef.Table != nil && indexDef.Table.Schema != "" {
					schemaName = indexDef.Table.Schema
				}
				schemaMeta := dbMetadata.GetSchema(schemaName)
				if schemaMeta == nil {
					continue
				}
				owner := schemaMeta.GetOwner()
				if owner == pgDatabaseOwner {
					owner = dbMetadata.GetOwner()
				}
				if owner != currentRole {
					adviceList = append(adviceList, &storepb.Advice{
						Status:        level,
						Title:         title,
						Content:       fmt.Sprintf("Schema \"%s\" is owned by \"%s\", but the current role is \"%s\".", schemaName, owner, currentRole),
						Code:          advisor.StatementObjectOwnerCheck.Int32(),
						StartPosition: common.ConvertPGParserLineToPosition(stmt.LastLine()),
					})
				}
			}
		case *ast.DropSchemaStmt:
			for _, schemaName := range n.SchemaList {
				schemaMeta := dbMetadata.GetSchema(schemaName)
				if schemaMeta == nil {
					continue
				}
				owner := schemaMeta.GetOwner()
				if owner == pgDatabaseOwner {
					owner = dbMetadata.GetOwner()
				}
				if owner != currentRole {
					adviceList = append(adviceList, &storepb.Advice{
						Status:        level,
						Title:         title,
						Content:       fmt.Sprintf("Schema \"%s\" is owned by \"%s\", but the current role is \"%s\".", schemaName, owner, currentRole),
						Code:          advisor.StatementObjectOwnerCheck.Int32(),
						StartPosition: common.ConvertPGParserLineToPosition(stmt.LastLine()),
					})
				}
			}
		case *ast.DropSequenceStmt:
			for _, seqName := range n.SequenceNameList {
				if seqName == nil {
					continue
				}
				schemaName := seqName.Schema
				if schemaName == "" {
					schemaName = defaultSchema
				}
				schemaMeta := dbMetadata.GetSchema(schemaName)
				if schemaMeta == nil {
					continue
				}
				owner := schemaMeta.GetOwner()
				if owner == pgDatabaseOwner {
					owner = dbMetadata.GetOwner()
				}
				if owner != currentRole {
					adviceList = append(adviceList, &storepb.Advice{
						Status:        level,
						Title:         title,
						Content:       fmt.Sprintf("Schema \"%s\" is owned by \"%s\", but the current role is \"%s\".", schemaName, owner, currentRole),
						Code:          advisor.StatementObjectOwnerCheck.Int32(),
						StartPosition: common.ConvertPGParserLineToPosition(stmt.LastLine()),
					})
				}
			}
		case *ast.DropTableStmt:
			for _, table := range n.TableList {
				if table == nil {
					continue
				}
				schemaName := table.Schema
				if schemaName == "" {
					schemaName = defaultSchema
				}
				schemaMeta := dbMetadata.GetSchema(schemaName)
				if schemaMeta == nil {
					continue
				}
				switch table.Type {
				case ast.TableTypeBaseTable:
					tableMeta := schemaMeta.GetTable(table.Name)
					if tableMeta == nil {
						continue
					}
					owner := tableMeta.GetOwner()
					if owner == pgDatabaseOwner {
						owner = dbMetadata.GetOwner()
					}
					if owner != currentRole {
						adviceList = append(adviceList, &storepb.Advice{
							Status:        level,
							Title:         title,
							Content:       fmt.Sprintf("Table \"%s\" is owned by \"%s\", but the current role is \"%s\".", table.Name, owner, currentRole),
							Code:          advisor.StatementObjectOwnerCheck.Int32(),
							StartPosition: common.ConvertPGParserLineToPosition(stmt.LastLine()),
						})
					}
				default:
					// todo: use view owner instead of schema owner
					owner := schemaMeta.GetOwner()
					if owner == pgDatabaseOwner {
						owner = dbMetadata.GetOwner()
					}
					if owner != currentRole {
						adviceList = append(adviceList, &storepb.Advice{
							Status:        level,
							Title:         title,
							Content:       fmt.Sprintf("Schema \"%s\" is owned by \"%s\", but the current role is \"%s\".", schemaName, owner, currentRole),
							Code:          advisor.StatementObjectOwnerCheck.Int32(),
							StartPosition: common.ConvertPGParserLineToPosition(stmt.LastLine()),
						})
					}
				}
			}
		case *ast.DropTriggerStmt:
			if n.Trigger == nil || n.Trigger.Table == nil {
				continue
			}
			schemaName := n.Trigger.Table.Schema
			if schemaName == "" {
				schemaName = defaultSchema
			}
			schemaMeta := dbMetadata.GetSchema(schemaName)
			if schemaMeta == nil {
				continue
			}
			owner := schemaMeta.GetOwner()
			if owner == pgDatabaseOwner {
				owner = dbMetadata.GetOwner()
			}
			if owner != currentRole {
				adviceList = append(adviceList, &storepb.Advice{
					Status:        level,
					Title:         title,
					Content:       fmt.Sprintf("Schema \"%s\" is owned by \"%s\", but the current role is \"%s\".", schemaName, owner, currentRole),
					Code:          advisor.StatementObjectOwnerCheck.Int32(),
					StartPosition: common.ConvertPGParserLineToPosition(stmt.LastLine()),
				})
			}
		case *ast.DropTypeStmt:
			for _, typeName := range n.TypeNameList {
				if typeName == nil {
					continue
				}
				schemaName := typeName.Schema
				if schemaName == "" {
					schemaName = defaultSchema
				}
				schemaMeta := dbMetadata.GetSchema(schemaName)
				if schemaMeta == nil {
					continue
				}
				owner := schemaMeta.GetOwner()
				if owner == pgDatabaseOwner {
					owner = dbMetadata.GetOwner()
				}
				if owner != currentRole {
					adviceList = append(adviceList, &storepb.Advice{
						Status:        level,
						Title:         title,
						Content:       fmt.Sprintf("Schema \"%s\" is owned by \"%s\", but the current role is \"%s\".", schemaName, owner, currentRole),
						Code:          advisor.StatementObjectOwnerCheck.Int32(),
						StartPosition: common.ConvertPGParserLineToPosition(stmt.LastLine()),
					})
				}
			}
		case *ast.RenameIndexStmt:
			schemaName := defaultSchema
			if n.Table != nil && n.Table.Schema != "" {
				schemaName = n.Table.Schema
			}
			schemaMeta := dbMetadata.GetSchema(schemaName)
			if schemaMeta == nil {
				continue
			}
			owner := schemaMeta.GetOwner()
			if owner == pgDatabaseOwner {
				owner = dbMetadata.GetOwner()
			}
			if owner != currentRole {
				adviceList = append(adviceList, &storepb.Advice{
					Status:        level,
					Title:         title,
					Content:       fmt.Sprintf("Schema \"%s\" is owned by \"%s\", but the current role is \"%s\".", schemaName, owner, currentRole),
					Code:          advisor.StatementObjectOwnerCheck.Int32(),
					StartPosition: common.ConvertPGParserLineToPosition(stmt.LastLine()),
				})
			}
		case *ast.RenameSchemaStmt:
			schemaMeta := dbMetadata.GetSchema(n.Schema)
			if schemaMeta == nil {
				continue
			}
			owner := schemaMeta.GetOwner()
			if owner == pgDatabaseOwner {
				owner = dbMetadata.GetOwner()
			}
			if owner != currentRole {
				adviceList = append(adviceList, &storepb.Advice{
					Status:        level,
					Title:         title,
					Content:       fmt.Sprintf("Schema \"%s\" is owned by \"%s\", but the current role is \"%s\".", n.Schema, owner, currentRole),
					Code:          advisor.StatementObjectOwnerCheck.Int32(),
					StartPosition: common.ConvertPGParserLineToPosition(stmt.LastLine()),
				})
			}
		}
	}

	return adviceList, nil
}

func getCurrentUser(ctx context.Context, driver *sql.DB) (string, error) {
	var user string
	err := driver.QueryRowContext(ctx, "SELECT CURRENT_USER").Scan(&user)
	if err != nil {
		return "", err
	}
	return user, nil
}
