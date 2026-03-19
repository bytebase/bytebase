package pg

import (
	"context"
	"fmt"
	"strings"

	"github.com/pkg/errors"

	"github.com/bytebase/omni/pg/ast"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
)

var (
	_ advisor.Advisor = (*NamingFKConventionAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_POSTGRES, storepb.SQLReviewRule_NAMING_INDEX_FK, &NamingFKConventionAdvisor{})
}

// NamingFKConventionAdvisor is the advisor checking for foreign key naming convention.
type NamingFKConventionAdvisor struct {
}

// Check checks for foreign key naming convention.
func (*NamingFKConventionAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
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

	rule := &namingFKConventionRule{
		OmniBaseRule: OmniBaseRule{
			Level: level,
			Title: checkCtx.Rule.Type.String(),
		},
		format:       format,
		maxLength:    maxLength,
		templateList: templateList,
	}

	return RunOmniRules(checkCtx.ParsedStatements, []OmniRule{rule}), nil
}

type namingFKConventionRule struct {
	OmniBaseRule

	format       string
	maxLength    int
	templateList []string
}

type fkMetaData struct {
	indexName string
	tableName string
	metaData  map[string]string
}

func (*namingFKConventionRule) Name() string {
	return "naming.table.fk"
}

func (r *namingFKConventionRule) OnStatement(node ast.Node) {
	switch n := node.(type) {
	case *ast.CreateStmt:
		r.handleCreateStmt(n)
	case *ast.AlterTableStmt:
		r.handleAlterTableStmt(n)
	default:
	}
}

func (r *namingFKConventionRule) handleCreateStmt(n *ast.CreateStmt) {
	tableName := omniTableName(n.Relation)
	cols, constraints := omniTableElements(n)

	// Check table-level FK constraints
	for _, c := range constraints {
		if md := r.extractFKMetaData(c, tableName); md != nil {
			r.checkFKMetadata(md)
		}
	}

	// Check column-level FK constraints
	for _, col := range cols {
		for _, c := range omniColumnConstraints(col) {
			if md := r.extractColumnFKMetaData(c, tableName, col.Colname); md != nil {
				r.checkFKMetadata(md)
			}
		}
	}
}

func (r *namingFKConventionRule) handleAlterTableStmt(n *ast.AlterTableStmt) {
	tableName := omniTableName(n.Relation)

	for _, cmd := range omniAlterTableCmds(n) {
		if ast.AlterTableType(cmd.Subtype) == ast.AT_AddConstraint {
			c, ok := cmd.Def.(*ast.Constraint)
			if !ok {
				continue
			}
			if md := r.extractFKMetaData(c, tableName); md != nil {
				r.checkFKMetadata(md)
			}
		}

		// ADD COLUMN with inline FK constraint
		if ast.AlterTableType(cmd.Subtype) == ast.AT_AddColumn {
			colDef, ok := cmd.Def.(*ast.ColumnDef)
			if !ok {
				continue
			}
			for _, c := range omniColumnConstraints(colDef) {
				if md := r.extractColumnFKMetaData(c, tableName, colDef.Colname); md != nil {
					r.checkFKMetadata(md)
				}
			}
		}
	}
}

func (*namingFKConventionRule) extractFKMetaData(c *ast.Constraint, tableName string) *fkMetaData {
	if c.Contype != ast.CONSTR_FOREIGN {
		return nil
	}

	constraintName := c.Conname

	// Extract referencing columns from FkAttrs
	var referencingColumns []string
	if c.FkAttrs != nil {
		for _, item := range c.FkAttrs.Items {
			if s, ok := item.(*ast.String); ok {
				referencingColumns = append(referencingColumns, s.Str)
			}
		}
	}

	// Extract referenced table
	referencedTable := omniTableName(c.Pktable)

	// Extract referenced columns from PkAttrs
	var referencedColumns []string
	if c.PkAttrs != nil {
		for _, item := range c.PkAttrs.Items {
			if s, ok := item.(*ast.String); ok {
				referencedColumns = append(referencedColumns, s.Str)
			}
		}
	}

	metaData := map[string]string{
		advisor.ReferencingTableNameTemplateToken:  tableName,
		advisor.ReferencingColumnNameTemplateToken: strings.Join(referencingColumns, "_"),
		advisor.ReferencedTableNameTemplateToken:   referencedTable,
		advisor.ReferencedColumnNameTemplateToken:  strings.Join(referencedColumns, "_"),
	}

	return &fkMetaData{
		indexName: constraintName,
		tableName: tableName,
		metaData:  metaData,
	}
}

func (*namingFKConventionRule) extractColumnFKMetaData(c *ast.Constraint, tableName, colName string) *fkMetaData {
	if c.Contype != ast.CONSTR_FOREIGN {
		return nil
	}

	constraintName := c.Conname
	referencedTable := omniTableName(c.Pktable)

	// Extract referenced columns from PkAttrs
	var referencedColumns []string
	if c.PkAttrs != nil {
		for _, item := range c.PkAttrs.Items {
			if s, ok := item.(*ast.String); ok {
				referencedColumns = append(referencedColumns, s.Str)
			}
		}
	}

	metaData := map[string]string{
		advisor.ReferencingTableNameTemplateToken:  tableName,
		advisor.ReferencingColumnNameTemplateToken: colName,
		advisor.ReferencedTableNameTemplateToken:   referencedTable,
		advisor.ReferencedColumnNameTemplateToken:  strings.Join(referencedColumns, "_"),
	}

	return &fkMetaData{
		indexName: constraintName,
		tableName: tableName,
		metaData:  metaData,
	}
}

func (r *namingFKConventionRule) checkFKMetadata(fkData *fkMetaData) {
	line := r.FindLineByName(fkData.indexName)

	regex, err := getTemplateRegexp(r.format, r.templateList, fkData.metaData)
	if err != nil {
		r.AddAdvice(&storepb.Advice{
			Status:  r.Level,
			Code:    code.Internal.Int32(),
			Title:   "Internal error for foreign key naming convention rule",
			Content: fmt.Sprintf("Failed to compile regex: %s", err.Error()),
			StartPosition: &storepb.Position{
				Line:   line,
				Column: 0,
			},
		})
		return
	}

	if !regex.MatchString(fkData.indexName) {
		r.AddAdvice(&storepb.Advice{
			Status:  r.Level,
			Code:    code.NamingFKConventionMismatch.Int32(),
			Title:   r.Title,
			Content: fmt.Sprintf(`Foreign key in table "%s" mismatches the naming convention, expect %q but found "%s"`, fkData.tableName, regex, fkData.indexName),
			StartPosition: &storepb.Position{
				Line:   line,
				Column: 0,
			},
		})
	}

	if r.maxLength > 0 && len(fkData.indexName) > r.maxLength {
		r.AddAdvice(&storepb.Advice{
			Status:  r.Level,
			Code:    code.NamingFKConventionMismatch.Int32(),
			Title:   r.Title,
			Content: fmt.Sprintf(`Foreign key "%s" in table "%s" mismatches the naming convention, its length should be within %d characters`, fkData.indexName, fkData.tableName, r.maxLength),
			StartPosition: &storepb.Position{
				Line:   line,
				Column: 0,
			},
		})
	}
}
