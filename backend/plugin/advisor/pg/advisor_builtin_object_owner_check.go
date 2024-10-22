package pg

import (
	"fmt"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/parser/sql/ast"
	"github.com/bytebase/bytebase/backend/store/model"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

const (
	pgDatabaseOwner = "pg_database_owner"
)

type BuiltinObjectOwnerCheckAdvisor struct {
}

func (*BuiltinObjectOwnerCheckAdvisor) Check(ctx advisor.Context, _ string) ([]*storepb.Advice, error) {
	var adviceList []*storepb.Advice
	stmtList, ok := ctx.AST.([]ast.Node)
	if !ok {
		return nil, errors.Errorf("failed to convert to Node")
	}

	level, err := advisor.NewStatusBySQLReviewRuleLevel(ctx.Rule.Level)
	if err != nil {
		return nil, err
	}
	title := string(ctx.Rule.Type)

	dbMetadata := model.NewDatabaseMetadata(ctx.DBSchema)
	currentRole := ctx.DBSchema.Owner
	defaultSchema := "public"

	for _, stmt := range stmtList {
		switch n := stmt.(type) {
		case *ast.VariableSetStmt:
			if n.Name == "role" {
				currentRole = n.GetRoleName()
			}
		case *ast.AlterSequenceStmt:
			// todo: use sequence owner instead of schema owner
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
					Status:  level,
					Title:   title,
					Content: fmt.Sprintf("Sequence \"%s\" is owned by \"%s\", but the current role is \"%s\".", n.Name.Name, owner, currentRole),
					Code:    advisor.BuiltinObjectOwnerCheck.Int32(),
					StartPosition: &storepb.Position{
						Line: int32(stmt.LastLine()),
					},
				})
			}
		case *ast.AlterTableStmt:
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
					Status:  level,
					Title:   title,
					Content: fmt.Sprintf("Table \"%s\" is owned by \"%s\", but the current role is \"%s\".", n.Table.Name, owner, currentRole),
					Code:    advisor.BuiltinObjectOwnerCheck.Int32(),
					StartPosition: &storepb.Position{
						Line: int32(stmt.LastLine()),
					},
				})
			}
		case *ast.AlterTypeStmt:
			// todo: use type owner instead of schema owner
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
					Status:  level,
					Title:   title,
					Content: fmt.Sprintf("Type \"%s\" is owned by \"%s\", but the current role is \"%s\".", n.Type.Name, owner, currentRole),
					Code:    advisor.BuiltinObjectOwnerCheck.Int32(),
					StartPosition: &storepb.Position{
						Line: int32(stmt.LastLine()),
					},
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
					Status:  level,
					Title:   title,
					Content: fmt.Sprintf("Schema \"%s\" is owned by \"%s\", but the current role is \"%s\".", schemaName, owner, currentRole),
					Code:    advisor.BuiltinObjectOwnerCheck.Int32(),
					StartPosition: &storepb.Position{
						Line: int32(stmt.LastLine()),
					},
				})
			}
		case *ast.CreateFunctionStmt:
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
					Status:  level,
					Title:   title,
					Content: fmt.Sprintf("Schema \"%s\" is owned by \"%s\", but the current role is \"%s\".", schemaName, owner, currentRole),
					Code:    advisor.BuiltinObjectOwnerCheck.Int32(),
					StartPosition: &storepb.Position{
						Line: int32(stmt.LastLine()),
					},
				})
			}
		case *ast.CreateIndexStmt:
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
					Status:  level,
					Title:   title,
					Content: fmt.Sprintf("Schema \"%s\" is owned by \"%s\", but the current role is \"%s\".", schemaName, owner, currentRole),
					Code:    advisor.BuiltinObjectOwnerCheck.Int32(),
					StartPosition: &storepb.Position{
						Line: int32(stmt.LastLine()),
					},
				})
			}
		case *ast.CreateSchemaStmt:
			owner := dbMetadata.GetOwner()
			if owner != currentRole {
				adviceList = append(adviceList, &storepb.Advice{
					Status:  level,
					Title:   title,
					Content: fmt.Sprintf("Database \"%s\" is owned by \"%s\", but the current role is \"%s\".", dbMetadata.GetName(), owner, currentRole),
					Code:    advisor.BuiltinObjectOwnerCheck.Int32(),
					StartPosition: &storepb.Position{
						Line: int32(stmt.LastLine()),
					},
				})
			}
		case *ast.CreateSequenceStmt:
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
					Status:  level,
					Title:   title,
					Content: fmt.Sprintf("Schema \"%s\" is owned by \"%s\", but the current role is \"%s\".", schemaName, owner, currentRole),
					Code:    advisor.BuiltinObjectOwnerCheck.Int32(),
					StartPosition: &storepb.Position{
						Line: int32(stmt.LastLine()),
					},
				})
			}
		case *ast.CreateTableStmt:
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
					Status:  level,
					Title:   title,
					Content: fmt.Sprintf("Schema \"%s\" is owned by \"%s\", but the current role is \"%s\".", schemaName, owner, currentRole),
					Code:    advisor.BuiltinObjectOwnerCheck.Int32(),
					StartPosition: &storepb.Position{
						Line: int32(stmt.LastLine()),
					},
				})
			}
		case *ast.CreateTriggerStmt:
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
					Status:  level,
					Title:   title,
					Content: fmt.Sprintf("Schema \"%s\" is owned by \"%s\", but the current role is \"%s\".", schemaName, owner, currentRole),
					Code:    advisor.BuiltinObjectOwnerCheck.Int32(),
					StartPosition: &storepb.Position{
						Line: int32(stmt.LastLine()),
					},
				})
			}
		case *ast.CreateTypeStmt:
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
					Status:  level,
					Title:   title,
					Content: fmt.Sprintf("Schema \"%s\" is owned by \"%s\", but the current role is \"%s\".", schemaName, owner, currentRole),
					Code:    advisor.BuiltinObjectOwnerCheck.Int32(),
					StartPosition: &storepb.Position{
						Line: int32(stmt.LastLine()),
					},
				})
			}
		case *ast.CreateViewStmt:
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
					Status:  level,
					Title:   title,
					Content: fmt.Sprintf("Schema \"%s\" is owned by \"%s\", but the current role is \"%s\".", schemaName, owner, currentRole),
					Code:    advisor.BuiltinObjectOwnerCheck.Int32(),
					StartPosition: &storepb.Position{
						Line: int32(stmt.LastLine()),
					},
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
						Status:  level,
						Title:   title,
						Content: fmt.Sprintf("Schema \"%s\" is owned by \"%s\", but the current role is \"%s\".", schemaName, owner, currentRole),
						Code:    advisor.BuiltinObjectOwnerCheck.Int32(),
						StartPosition: &storepb.Position{
							Line: int32(stmt.LastLine()),
						},
					})
				}
			}
		case *ast.DropIndexStmt:
			for _, indexDef := range n.IndexList {
				if indexDef == nil {
					continue
				}
				schemaName := indexDef.Table.Schema
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
						Status:  level,
						Title:   title,
						Content: fmt.Sprintf("Schema \"%s\" is owned by \"%s\", but the current role is \"%s\".", schemaName, owner, currentRole),
						Code:    advisor.BuiltinObjectOwnerCheck.Int32(),
						StartPosition: &storepb.Position{
							Line: int32(stmt.LastLine()),
						},
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
						Status:  level,
						Title:   title,
						Content: fmt.Sprintf("Schema \"%s\" is owned by \"%s\", but the current role is \"%s\".", schemaName, owner, currentRole),
						Code:    advisor.BuiltinObjectOwnerCheck.Int32(),
						StartPosition: &storepb.Position{
							Line: int32(stmt.LastLine()),
						},
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
						Status:  level,
						Title:   title,
						Content: fmt.Sprintf("Schema \"%s\" is owned by \"%s\", but the current role is \"%s\".", schemaName, owner, currentRole),
						Code:    advisor.BuiltinObjectOwnerCheck.Int32(),
						StartPosition: &storepb.Position{
							Line: int32(stmt.LastLine()),
						},
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
				tableMeta := schemaMeta.GetTable(table.Name)
				owner := tableMeta.GetOwner()
				if owner == pgDatabaseOwner {
					owner = dbMetadata.GetOwner()
				}
				if owner != currentRole {
					adviceList = append(adviceList, &storepb.Advice{
						Status:  level,
						Title:   title,
						Content: fmt.Sprintf("Table \"%s\" is owned by \"%s\", but the current role is \"%s\".", table.Name, owner, currentRole),
						Code:    advisor.BuiltinObjectOwnerCheck.Int32(),
						StartPosition: &storepb.Position{
							Line: int32(stmt.LastLine()),
						},
					})
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
					Status:  level,
					Title:   title,
					Content: fmt.Sprintf("Schema \"%s\" is owned by \"%s\", but the current role is \"%s\".", schemaName, owner, currentRole),
					Code:    advisor.BuiltinObjectOwnerCheck.Int32(),
					StartPosition: &storepb.Position{
						Line: int32(stmt.LastLine()),
					},
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
						Status:  level,
						Title:   title,
						Content: fmt.Sprintf("Schema \"%s\" is owned by \"%s\", but the current role is \"%s\".", schemaName, owner, currentRole),
						Code:    advisor.BuiltinObjectOwnerCheck.Int32(),
						StartPosition: &storepb.Position{
							Line: int32(stmt.LastLine()),
						},
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
					Status:  level,
					Title:   title,
					Content: fmt.Sprintf("Schema \"%s\" is owned by \"%s\", but the current role is \"%s\".", schemaName, owner, currentRole),
					Code:    advisor.BuiltinObjectOwnerCheck.Int32(),
					StartPosition: &storepb.Position{
						Line: int32(stmt.LastLine()),
					},
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
					Status:  level,
					Title:   title,
					Content: fmt.Sprintf("Schema \"%s\" is owned by \"%s\", but the current role is \"%s\".", schemaName, owner, currentRole),
					Code:    advisor.BuiltinObjectOwnerCheck.Int32(),
					StartPosition: &storepb.Position{
						Line: int32(stmt.LastLine()),
					},
				})
			}
		}
	}
}
