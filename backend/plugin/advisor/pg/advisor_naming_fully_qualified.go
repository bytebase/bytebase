package pg

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	pgquery "github.com/pganalyze/pg_query_go/v6"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/plugin/parser/pg/legacy/ast"
)

func init() {
	advisor.Register(storepb.Engine_POSTGRES, advisor.PostgreSQLNamingFullyQualifiedObjectName, &FullyQualifiedObjectNameAdvisor{})
}

type FullyQualifiedObjectNameAdvisor struct{}

type FullyQualifiedObjectNameChecker struct {
	adviceList   []*storepb.Advice
	status       storepb.Advice_Status
	title        string
	line         int
	isSelectStmt bool
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
	case *ast.CreateIndexStmt:
		if node.Index != nil {
			fullyQualifiedName := getFullyQualiufiedObjectName(node.Index)
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
		checker.isSelectStmt = true

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

func (*FullyQualifiedObjectNameAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	checker := &FullyQualifiedObjectNameChecker{}
	status, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}
	checker.status = status
	checker.title = checkCtx.Rule.Type

	nodes, ok := checkCtx.AST.([]ast.Node)
	if !ok {
		return nil, errors.Errorf("failed to convert to Node")
	}

	for _, node := range nodes {
		checker.line = node.LastLine()
		ast.Walk(checker, node)
		// Dive in again for the object names in the subquery if it's a select statement.
		if checker.isSelectStmt {
			if checkCtx.DBSchema == nil {
				continue
			}
			for _, tableName := range findAllTables(node.Text(), checkCtx.DBSchema) {
				checker.appendAdviceByObjName(tableName.String())
			}
		}
	}

	return checker.adviceList, nil
}

func (checker *FullyQualifiedObjectNameChecker) appendAdviceByObjName(objName string) {
	if objName == "" {
		return
	}
	re := regexp.MustCompile(`.+\..+`)
	if !re.MatchString(objName) {
		checker.adviceList = append(checker.adviceList, &storepb.Advice{
			Status:        checker.status,
			Code:          advisor.NamingNotFullyQualifiedName,
			Title:         checker.title,
			Content:       fmt.Sprintf("unqualified object name: '%s'", objName),
			StartPosition: common.ConvertPGParserLineToPosition(checker.line),
		})
	}
}

func getFullyQualiufiedObjectName(nodeDef ast.Node) string {
	sb := strings.Builder{}
	switch def := nodeDef.(type) {
	case *ast.TableDef:
		if def.Database != "" {
			if _, err := sb.WriteString(def.Database); err != nil {
				return ""
			}
			if _, err := sb.WriteRune('.'); err != nil {
				return ""
			}
		}
		if def.Schema != "" {
			if _, err := sb.WriteString(def.Schema); err != nil {
				return ""
			}
			if _, err := sb.WriteRune('.'); err != nil {
				return ""
			}
		}
		if _, err := sb.WriteString(def.Name); err != nil {
			return ""
		}

	case *ast.IndexDef:
		if def.Table != nil && def.Table.Schema != "" {
			if _, err := sb.WriteString(def.Table.Name); err != nil {
				return ""
			}
			if _, err := sb.WriteRune('.'); err != nil {
				return ""
			}
		}
		if _, err := sb.WriteString(def.Name); err != nil {
			return ""
		}

	case *ast.SequenceNameDef:
		if def.Schema != "" {
			if _, err := sb.WriteString(def.Schema); err != nil {
				return ""
			}
			if _, err := sb.WriteRune('.'); err != nil {
				return ""
			}
		}
		if _, err := sb.WriteString(def.Name); err != nil {
			return ""
		}

	case *ast.ColumnNameDef:
		if def.Table != nil && def.Table.Name != "" {
			_, _ = sb.WriteString(def.Table.Name)
			if _, err := sb.WriteRune('.'); err != nil {
				return ""
			}
		}
		if _, err := sb.WriteString(def.ColumnName); err != nil {
			return ""
		}

	default:
	}
	return sb.String()
}

// Used for select statement.
func findAllTables(statement string, schemaMetadata *storepb.DatabaseSchemaMetadata) []base.ColumnResource {
	jsonText, err := pgquery.ParseToJSON(statement)
	if err != nil {
		return nil
	}

	var jsonData map[string]any
	if err := json.Unmarshal([]byte(jsonText), &jsonData); err != nil {
		return nil
	}

	schemaNameMap := getSchemaNameMapFromPublic(schemaMetadata)
	if schemaNameMap == nil {
		return []base.ColumnResource{}
	}

	resourceArray, err := getRangeVarsFromJSONRecursive(jsonData, &schemaNameMap)
	if err != nil {
		return nil
	}

	return resourceArray
}

func getSchemaNameMapFromPublic(schemaMetadata *storepb.DatabaseSchemaMetadata) map[string]bool {
	if schemaMetadata.Schemas == nil {
		return nil
	}
	filterMap := map[string]bool{}
	for _, schema := range schemaMetadata.Schemas {
		// Tables.
		for _, tbl := range schema.Tables {
			filterMap[tbl.Name] = true
		}
		// External Tables.
		for _, tbl := range schema.ExternalTables {
			filterMap[tbl.Name] = true
		}
	}
	return filterMap
}

// get table names from Json.
func getRangeVarsFromJSONRecursive(jsonData map[string]any, filterMap *map[string]bool) ([]base.ColumnResource, error) {
	var result []base.ColumnResource
	if jsonData["RangeVar"] != nil {
		resource := base.ColumnResource{}

		rangeVar, ok := jsonData["RangeVar"].(map[string]any)
		if !ok {
			return nil, errors.Errorf("failed to convert range var")
		}
		if rangeVar["schemaname"] != nil {
			schema, ok := rangeVar["schemaname"].(string)
			if !ok {
				return nil, errors.Errorf("failed to convert schemaname")
			}
			resource.Schema = schema
		}
		if rangeVar["relname"] != nil {
			table, ok := rangeVar["relname"].(string)
			if !ok {
				return nil, errors.Errorf("failed to convert relname")
			}
			resource.Table = table
		}

		if _, ok := (*filterMap)[resource.Table]; ok {
			result = append(result, resource)
		}
	}

	for _, value := range jsonData {
		switch v := value.(type) {
		case map[string]any:
			resources, err := getRangeVarsFromJSONRecursive(v, filterMap)
			if err != nil {
				return nil, err
			}
			result = append(result, resources...)
		case []any:
			for _, item := range v {
				if m, ok := item.(map[string]any); ok {
					resources, err := getRangeVarsFromJSONRecursive(m, filterMap)
					if err != nil {
						return nil, err
					}
					result = append(result, resources...)
				}
			}
		}
	}

	return result, nil
}
