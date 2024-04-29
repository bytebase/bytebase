package pg

import (
	"database/sql"
	"fmt"
	"regexp"
	"strings"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/parser/sql/ast"
	"github.com/bytebase/bytebase/proto/generated-go/store"
)

func init() {
	advisor.Register(store.Engine_POSTGRES, advisor.PostgreSQLNamingFullyQualifiedObjectName, &FullyQualifiedObjectNameAdvisor{})
}

type FullyQualifiedObjectNameAdvisor struct{}

type FullyQualifiedObjectNameChecker struct {
	_          *sql.DB
	adviceList []advisor.Advice
	status     advisor.Status
	title      string
	line       int
}

// Visit implements ast.Visitor.
func (checker *FullyQualifiedObjectNameChecker) Visit(in ast.Node) ast.Visitor {
	switch node := in.(type) {
	// Create statement.
	case *ast.CreateTableStmt:
		if node.Name != nil {
			fullyQualifiedName := getFullyQualiufiedObjectName(node.Name)
			checker.appendAdviceByObjName(fullyQualifiedName)
		}
	case *ast.CreateSequenceStmt:
		if node.SequenceDef.SequenceName != nil {
			fullyQualifiedName := getFullyQualiufiedObjectName(node.SequenceDef.SequenceName)
			checker.appendAdviceByObjName(fullyQualifiedName)
		}
	case *ast.CreateTriggerStmt:
		if node.Trigger != nil && node.Trigger.Table != nil {
			fullyQualifiedName := getFullyQualiufiedObjectName(node.Trigger.Table)
			checker.appendAdviceByObjName(fullyQualifiedName)
		}

	// Drop statement.
	case *ast.DropSequenceStmt:
		if node.SequenceNameList != nil {
			for _, seqName := range node.SequenceNameList {
				fullyQualifiedName := getFullyQualiufiedObjectName(seqName)
				checker.appendAdviceByObjName(fullyQualifiedName)
			}
		}
	case *ast.DropTableStmt:
		if node.TableList != nil {
			for _, table := range node.TableList {
				fullyQualifiedName := getFullyQualiufiedObjectName(table)
				checker.appendAdviceByObjName(fullyQualifiedName)
			}
		}
	case *ast.DropIndexStmt:
		if node.IndexList != nil {
			for _, index := range node.IndexList {
				fullyQualifiedName := getFullyQualiufiedObjectName(index)
				checker.appendAdviceByObjName(fullyQualifiedName)
			}
		}
	case *ast.DropTriggerStmt:
		if node.Trigger != nil && node.Trigger.Table != nil {
			fullyQualifiedName := getFullyQualiufiedObjectName(node.Trigger.Table)
			checker.appendAdviceByObjName(fullyQualifiedName)
		}

	// Alter statement.
	case *ast.AlterTableStmt:
		if node.Table != nil {
			fullyQualifiedName := getFullyQualiufiedObjectName(node.Table)
			checker.appendAdviceByObjName(fullyQualifiedName)
		}
	case *ast.AlterSequenceStmt:
		if node.Name != nil {
			fullyQualifiedName := getFullyQualiufiedObjectName(node.Name)
			checker.appendAdviceByObjName(fullyQualifiedName)
		}

	// Insert statement.
	case *ast.InsertStmt:
		if node.Table != nil {
			fullyQualifiedName := getFullyQualiufiedObjectName(node.Table)
			checker.appendAdviceByObjName(fullyQualifiedName)
		}

	// Select statement.
	case *ast.SelectStmt:
		if node.FieldList != nil {
			for _, field := range node.FieldList {
				if fieldDef, ok := field.(*ast.ColumnNameDef); ok {
					if fieldDef.ColumnName == "*" {
						break
					}
					fullyQualifiedName := getFullyQualiufiedObjectName(fieldDef)
					checker.appendAdviceByObjName(fullyQualifiedName)
				}
			}
		}

	// Update statement.
	case *ast.UpdateStmt:
		if node.Table != nil {
			fullyQualifiedName := getFullyQualiufiedObjectName(node.Table)
			checker.appendAdviceByObjName(fullyQualifiedName)
		}

	// TODO(tommy): check whether this is needed.
	// Comment statement.
	case *ast.CommentStmt:
	default:
	}

	return checker
}

var (
	_ advisor.Advisor = (*FullyQualifiedObjectNameAdvisor)(nil)
	_ ast.Visitor     = (*FullyQualifiedObjectNameChecker)(nil)
)

func (*FullyQualifiedObjectNameAdvisor) Check(ctx advisor.Context, _ string) ([]advisor.Advice, error) {
	checker := &FullyQualifiedObjectNameChecker{}
	status, err := advisor.NewStatusBySQLReviewRuleLevel(ctx.Rule.Level)
	if err != nil {
		return nil, err
	}
	checker.status = status
	checker.title = ctx.Rule.Type

	nodes, ok := ctx.AST.([]ast.Node)
	if !ok {
		return nil, errors.Errorf("failed to convert to Node")
	}

	for _, node := range nodes {
		checker.line = node.LastLine()
		ast.Walk(checker, node)
	}

	if len(checker.adviceList) == 0 {
		checker.adviceList = append(checker.adviceList, advisor.Advice{
			Status:  advisor.Success,
			Code:    advisor.Ok,
			Title:   "OK",
			Content: "",
		})
	}

	return checker.adviceList, nil
}

func (checker *FullyQualifiedObjectNameChecker) appendAdviceByObjName(objName string) {
	if objName == "" {
		return
	}
	re := regexp.MustCompile(`.+\..+`)
	if !re.MatchString(objName) {
		checker.adviceList = append(checker.adviceList, advisor.Advice{
			Status:  checker.status,
			Code:    advisor.NamingNotFullyQualifiedName,
			Title:   checker.title,
			Content: fmt.Sprintf("unqualified object name: '%s'", objName),
			Line:    checker.line,
		})
	}
}

func getFullyQualiufiedObjectName(nodeDef ast.Node) string {
	sb := strings.Builder{}
	switch def := nodeDef.(type) {
	case *ast.TableDef:
		if def.Database != "" {
			_, _ = sb.WriteString(def.Database)
			_, _ = sb.WriteRune('.')
		}
		if def.Schema != "" {
			_, _ = sb.WriteString(def.Schema)
			_, _ = sb.WriteRune('.')
		}
		_, _ = sb.WriteString(def.Name)

	case *ast.IndexDef:
		if def.Table != nil && def.Table.Schema != "" {
			_, _ = sb.WriteString(def.Table.Name)
			_, _ = sb.WriteRune('.')
		}
		_, _ = sb.WriteString(def.Name)

	case *ast.SequenceNameDef:
		if def.Schema != "" {
			_, _ = sb.WriteString(def.Schema)
			_, _ = sb.WriteRune('.')
		}
		sb.WriteString(def.Name)

	case *ast.ColumnNameDef:
		if def.Table != nil && def.Table.Name != "" {
			_, _ = sb.WriteString(def.Table.Name)
			sb.WriteRune('.')
		}
		sb.WriteString(def.ColumnName)

	default:
	}
	return sb.String()
}
