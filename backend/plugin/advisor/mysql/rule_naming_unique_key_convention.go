package mysql

import (
	"context"
	"fmt"
	"strings"

	"github.com/bytebase/omni/mysql/ast"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	"github.com/bytebase/bytebase/backend/store/model"
)

var (
	_ advisor.Advisor = (*NamingUKConventionAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_MYSQL, storepb.SQLReviewRule_NAMING_INDEX_UK, &NamingUKConventionAdvisor{})
	advisor.Register(storepb.Engine_MARIADB, storepb.SQLReviewRule_NAMING_INDEX_UK, &NamingUKConventionAdvisor{})
	advisor.Register(storepb.Engine_OCEANBASE, storepb.SQLReviewRule_NAMING_INDEX_UK, &NamingUKConventionAdvisor{})
}

// NamingUKConventionAdvisor is the advisor checking for unique key naming convention.
type NamingUKConventionAdvisor struct {
}

// Check checks for index naming convention.
func (*NamingUKConventionAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	namingPayload := checkCtx.Rule.GetNamingPayload()
	if namingPayload == nil {
		return nil, errors.New("naming_payload is required for this rule")
	}

	format := namingPayload.Format
	templateList, _ := advisor.ParseTemplateTokens(format)

	for _, key := range templateList {
		if _, ok := advisor.TemplateNamingTokens[checkCtx.Rule.Type][key]; !ok {
			return nil, errors.Errorf("invalid template %s for rule %s", key, checkCtx.Rule.Type)
		}
	}

	maxLength := int(namingPayload.MaxLength)
	if maxLength == 0 {
		maxLength = advisor.DefaultNameLengthLimit
	}

	rule := &namingUKOmniRule{
		OmniBaseRule: OmniBaseRule{
			Level: level,
			Title: checkCtx.Rule.Type.String(),
		},
		format:           format,
		maxLength:        maxLength,
		templateList:     templateList,
		originalMetadata: checkCtx.OriginalMetadata,
	}

	return RunOmniRules(checkCtx.ParsedStatements, []OmniRule{rule}), nil
}

// ukIndexMetaData is the metadata for unique key.
type ukIndexMetaData struct {
	indexName string
	tableName string
	metaData  map[string]string
	line      int
}

type namingUKOmniRule struct {
	OmniBaseRule
	format           string
	maxLength        int
	templateList     []string
	originalMetadata *model.DatabaseMetadata
}

func (*namingUKOmniRule) Name() string {
	return "NamingUKConventionRule"
}

func (r *namingUKOmniRule) OnStatement(node ast.Node) {
	switch n := node.(type) {
	case *ast.CreateTableStmt:
		r.checkCreateTable(n)
	case *ast.AlterTableStmt:
		r.checkAlterTable(n)
	case *ast.CreateIndexStmt:
		r.checkCreateIndex(n)
	default:
	}
}

func (r *namingUKOmniRule) checkCreateTable(n *ast.CreateTableStmt) {
	if n.Table == nil {
		return
	}
	tableName := n.Table.Name
	var indexDataList []*ukIndexMetaData
	for _, constraint := range n.Constraints {
		if constraint == nil {
			continue
		}
		if metaData := r.handleConstraint(tableName, constraint, r.BaseLine+int(r.LocToLine(constraint.Loc))); metaData != nil {
			indexDataList = append(indexDataList, metaData)
		}
	}
	r.handleIndexList(indexDataList)
}

func (r *namingUKOmniRule) checkAlterTable(n *ast.AlterTableStmt) {
	tableName := ""
	if n.Table != nil {
		tableName = n.Table.Name
	}
	var indexDataList []*ukIndexMetaData
	for _, cmd := range n.Commands {
		if cmd == nil {
			continue
		}
		switch cmd.Type {
		case ast.ATAddConstraint, ast.ATAddIndex:
			if cmd.Constraint != nil {
				if metaData := r.handleConstraint(tableName, cmd.Constraint, r.BaseLine+int(r.LocToLine(n.Loc))); metaData != nil {
					indexDataList = append(indexDataList, metaData)
				}
			}
		case ast.ATRenameIndex:
			oldIndexName := cmd.Name
			newIndexName := cmd.NewName
			indexState := r.originalMetadata.GetSchemaMetadata("").GetTable(tableName).GetIndex(oldIndexName)
			if indexState == nil {
				continue
			}
			if !indexState.GetProto().GetUnique() {
				continue
			}
			columnList := indexState.GetProto().GetExpressions()
			metaData := map[string]string{
				advisor.ColumnListTemplateToken: strings.Join(columnList, "_"),
				advisor.TableNameTemplateToken:  tableName,
			}
			indexData := &ukIndexMetaData{
				indexName: newIndexName,
				tableName: tableName,
				metaData:  metaData,
				line:      r.BaseLine + int(r.LocToLine(n.Loc)),
			}
			indexDataList = append(indexDataList, indexData)
		default:
		}
	}
	r.handleIndexList(indexDataList)
}

func (r *namingUKOmniRule) checkCreateIndex(n *ast.CreateIndexStmt) {
	// Only focus on unique index.
	if !n.Unique {
		return
	}
	if n.Table == nil {
		return
	}
	tableName := n.Table.Name
	columnList := omniIndexColumns(n.Columns)
	metaData := map[string]string{
		advisor.ColumnListTemplateToken: strings.Join(columnList, "_"),
		advisor.TableNameTemplateToken:  tableName,
	}
	indexDataList := []*ukIndexMetaData{
		{
			indexName: n.IndexName,
			tableName: tableName,
			metaData:  metaData,
			line:      r.BaseLine + int(r.LocToLine(n.Loc)),
		},
	}
	r.handleIndexList(indexDataList)
}

func (r *namingUKOmniRule) handleIndexList(indexDataList []*ukIndexMetaData) {
	for _, indexData := range indexDataList {
		regex, err := getTemplateRegexp(r.format, r.templateList, indexData.metaData)
		if err != nil {
			r.AddAdviceAbsolute(&storepb.Advice{
				Status:  r.Level,
				Code:    code.Internal.Int32(),
				Title:   "Internal error for unique key naming convention rule",
				Content: fmt.Sprintf("%q meet internal error %q", r.TrimmedStmtText(), err.Error()),
			})
			continue
		}
		if !regex.MatchString(indexData.indexName) {
			r.AddAdviceAbsolute(&storepb.Advice{
				Status:        r.Level,
				Code:          code.NamingUKConventionMismatch.Int32(),
				Title:         r.Title,
				Content:       fmt.Sprintf("Unique key in table `%s` mismatches the naming convention, expect %q but found `%s`", indexData.tableName, regex, indexData.indexName),
				StartPosition: common.ConvertANTLRLineToPosition(indexData.line),
			})
		}
		if r.maxLength > 0 && len(indexData.indexName) > r.maxLength {
			r.AddAdviceAbsolute(&storepb.Advice{
				Status:        r.Level,
				Code:          code.NamingUKConventionMismatch.Int32(),
				Title:         r.Title,
				Content:       fmt.Sprintf("Unique key `%s` in table `%s` mismatches the naming convention, its length should be within %d characters", indexData.indexName, indexData.tableName, r.maxLength),
				StartPosition: common.ConvertANTLRLineToPosition(indexData.line),
			})
		}
	}
}

func (*namingUKOmniRule) handleConstraint(tableName string, constraint *ast.Constraint, line int) *ukIndexMetaData {
	// Focus on unique index.
	if constraint.Type != ast.ConstrUnique {
		return nil
	}
	indexName := constraint.Name
	columnList := constraint.Columns
	if len(columnList) == 0 {
		columnList = omniIndexColumns(constraint.IndexColumns)
	}
	metaData := map[string]string{
		advisor.ColumnListTemplateToken: strings.Join(columnList, "_"),
		advisor.TableNameTemplateToken:  tableName,
	}
	return &ukIndexMetaData{
		indexName: indexName,
		tableName: tableName,
		metaData:  metaData,
		line:      line,
	}
}
