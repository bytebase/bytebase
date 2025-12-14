package redshift

import (
	"strings"

	"github.com/antlr4-go/antlr/v4"
	parser "github.com/bytebase/parser/redshift"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/store/model"
)

func init() {
	base.RegisterExtractChangedResourcesFunc(storepb.Engine_REDSHIFT, extractChangedResources)
}

func extractChangedResources(database string, _ string, dbMetadata *model.DatabaseMetadata, asts []base.AST, _ string) (*base.ChangeSummary, error) {
	changedResources := model.NewChangedResources(dbMetadata)
	listener := &resourceChangeListener{
		currentDatabase:  database,
		changedResources: changedResources,
		dbMetadata:       dbMetadata,
		searchPath:       []string{"public"}, // Default search path for Redshift
	}

	for _, ast := range asts {
		antlrAST, ok := base.GetANTLRAST(ast)
		if !ok {
			return nil, errors.New("expected ANTLR AST for Redshift")
		}
		antlr.ParseTreeWalkerDefault.Walk(listener, antlrAST.Tree)
	}

	return &base.ChangeSummary{
		ChangedResources: changedResources,
		SampleDMLS:       listener.sampleDMLs,
		DMLCount:         listener.dmlCount,
		InsertCount:      listener.insertCount,
	}, nil
}

type resourceChangeListener struct {
	*parser.BaseRedshiftParserListener

	currentDatabase  string
	changedResources *model.ChangedResources
	dbMetadata       *model.DatabaseMetadata
	searchPath       []string
	sampleDMLs       []string
	dmlCount         int
	insertCount      int
}

// EnterCreatestmt is called when entering a createstmt rule (CREATE TABLE).
func (l *resourceChangeListener) EnterCreatestmt(ctx *parser.CreatestmtContext) {
	if ctx.Table_name() == nil {
		return
	}

	tableName := extractTableName(ctx.Table_name())
	schemaName := extractSchemaFromTableName(ctx.Table_name(), l.searchPath[0])

	l.changedResources.AddTable(
		l.currentDatabase,
		schemaName,
		&storepb.ChangedResourceTable{
			Name: tableName,
		},
		false, // affectedTable
	)
}

// EnterDropstmt is called when entering a dropstmt rule.
func (l *resourceChangeListener) EnterDropstmt(ctx *parser.DropstmtContext) {
	if ctx.DROP() == nil {
		return
	}

	// Handle DROP TABLE
	if ctx.Object_type_any_name() != nil && ctx.Object_type_any_name().TABLE() != nil {
		if ctx.Any_name_list() != nil {
			for _, anyName := range ctx.Any_name_list().AllAny_name() {
				tableName := extractAnyName(anyName)
				schemaName := extractSchemaFromAnyName(anyName, l.searchPath[0])
				// For DROP TABLE, we mark it as removed
				l.changedResources.AddTable(
					l.currentDatabase,
					schemaName,
					&storepb.ChangedResourceTable{
						Name: tableName,
					},
					true, // affectedTable - true means it's being dropped
				)
			}
		}
	}

	// Handle DROP INDEX
	// Note: Drop_type_name is used for different drop types, not indexes
	// TODO: Find correct way to detect DROP INDEX in Redshift parser and implement index tracking
}

// EnterAltertablestmt is called when entering an altertablestmt rule.
func (l *resourceChangeListener) EnterAltertablestmt(ctx *parser.AltertablestmtContext) {
	if ctx.ALTER() == nil || ctx.TABLE() == nil || ctx.Qualified_name() == nil {
		return
	}

	tableName := extractQualifiedName(ctx.Qualified_name())
	schemaName := extractSchemaName(ctx.Qualified_name(), l.searchPath[0])

	// For ALTER TABLE, we add it as an affected table
	l.changedResources.AddTable(
		l.currentDatabase,
		schemaName,
		&storepb.ChangedResourceTable{
			Name: tableName,
		},
		true, // affectedTable
	)
}

// EnterIndexstmt is called when entering an indexstmt rule (CREATE INDEX).
func (*resourceChangeListener) EnterIndexstmt(ctx *parser.IndexstmtContext) {
	if ctx.CREATE() == nil || ctx.INDEX() == nil {
		return
	}

	// For CREATE INDEX, we need to track which table it affects
	// TODO: Extract table name from index statement context
	// For now, we skip index tracking
}

// EnterVariablesetstmt is called when entering a variablesetstmt rule.
func (l *resourceChangeListener) EnterVariablesetstmt(ctx *parser.VariablesetstmtContext) {
	if ctx.SET() == nil {
		return
	}

	// Check for SET search_path
	if ctx.Set_rest() != nil && ctx.Set_rest().GetText() != "" {
		text := strings.ToLower(ctx.Set_rest().GetText())
		if strings.Contains(text, "search_path") {
			// Extract the new search path value
			// This is a simplified implementation - in production you'd want more robust parsing
			if strings.Contains(text, "=") {
				parts := strings.Split(text, "=")
				if len(parts) > 1 {
					pathValue := strings.TrimSpace(parts[1])
					pathValue = strings.Trim(pathValue, "'\"")
					schemas := strings.Split(pathValue, ",")
					l.searchPath = make([]string, 0, len(schemas))
					for _, schema := range schemas {
						schema = strings.TrimSpace(schema)
						if schema != "" {
							l.searchPath = append(l.searchPath, schema)
						}
					}
					if len(l.searchPath) == 0 {
						l.searchPath = []string{"public"}
					}
				}
			}
		}
	}
}

// DML statement handlers
func (l *resourceChangeListener) EnterInsertstmt(ctx *parser.InsertstmtContext) {
	if ctx.INSERT() == nil {
		return
	}
	l.dmlCount++
	l.insertCount++
	if len(l.sampleDMLs) < common.MaximumLintExplainSize {
		l.sampleDMLs = append(l.sampleDMLs, ctx.GetText())
	}
}

func (l *resourceChangeListener) EnterUpdatestmt(ctx *parser.UpdatestmtContext) {
	if ctx.UPDATE() == nil {
		return
	}
	l.dmlCount++
	if len(l.sampleDMLs) < common.MaximumLintExplainSize {
		l.sampleDMLs = append(l.sampleDMLs, ctx.GetText())
	}
}

func (l *resourceChangeListener) EnterDeletestmt(ctx *parser.DeletestmtContext) {
	if ctx.DELETE_P() == nil {
		return
	}
	l.dmlCount++
	if len(l.sampleDMLs) < common.MaximumLintExplainSize {
		l.sampleDMLs = append(l.sampleDMLs, ctx.GetText())
	}
}

// Helper functions to extract names from various contexts
func extractQualifiedName(ctx parser.IQualified_nameContext) string {
	if ctx == nil {
		return ""
	}

	parts := []string{}
	if ctx.Colid() != nil {
		parts = append(parts, normalizeRedshiftIdentifierText(ctx.Colid().GetText()))
	}

	if ctx.Indirection() != nil {
		for _, el := range ctx.Indirection().AllIndirection_el() {
			if el.Attr_name() != nil {
				parts = append(parts, normalizeRedshiftIdentifierText(el.Attr_name().GetText()))
			}
		}
	}

	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return ""
}

func extractSchemaName(ctx parser.IQualified_nameContext, defaultSchema string) string {
	if ctx == nil {
		return defaultSchema
	}

	parts := []string{}
	if ctx.Colid() != nil {
		parts = append(parts, normalizeRedshiftIdentifierText(ctx.Colid().GetText()))
	}

	if ctx.Indirection() != nil {
		for _, el := range ctx.Indirection().AllIndirection_el() {
			if el.Attr_name() != nil {
				parts = append(parts, normalizeRedshiftIdentifierText(el.Attr_name().GetText()))
			}
		}
	}

	// If we have more than one part, the first part is the schema
	if len(parts) > 1 {
		return parts[0]
	}
	return defaultSchema
}

func extractAnyName(ctx parser.IAny_nameContext) string {
	if ctx == nil {
		return ""
	}

	parts := []string{}
	if ctx.Colid() != nil {
		parts = append(parts, normalizeRedshiftIdentifierText(ctx.Colid().GetText()))
	}

	if ctx.Attrs() != nil {
		for _, attr := range ctx.Attrs().AllAttr_name() {
			parts = append(parts, normalizeRedshiftIdentifierText(attr.GetText()))
		}
	}

	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return ""
}

func extractSchemaFromAnyName(ctx parser.IAny_nameContext, defaultSchema string) string {
	if ctx == nil {
		return defaultSchema
	}

	parts := []string{}
	if ctx.Colid() != nil {
		parts = append(parts, normalizeRedshiftIdentifierText(ctx.Colid().GetText()))
	}

	if ctx.Attrs() != nil {
		for _, attr := range ctx.Attrs().AllAttr_name() {
			parts = append(parts, normalizeRedshiftIdentifierText(attr.GetText()))
		}
	}

	// If we have more than one part, the first part is the schema
	if len(parts) > 1 {
		return parts[0]
	}
	return defaultSchema
}

// Helper function to extract table name
func extractTableName(ctx parser.ITable_nameContext) string {
	if ctx == nil {
		return ""
	}

	// Check qualified name first
	if ctx.Qualified_name() != nil {
		return extractQualifiedName(ctx.Qualified_name())
	}

	// Check temporary table name
	if ctx.Temporary_table_name() != nil {
		// For temporary tables, just return the text
		return normalizeRedshiftIdentifierText(ctx.Temporary_table_name().GetText())
	}

	return ""
}

func extractSchemaFromTableName(ctx parser.ITable_nameContext, defaultSchema string) string {
	if ctx == nil {
		return defaultSchema
	}

	// Check qualified name first
	if ctx.Qualified_name() != nil {
		return extractSchemaName(ctx.Qualified_name(), defaultSchema)
	}

	// Temporary tables are typically in a special schema
	if ctx.Temporary_table_name() != nil {
		return "pg_temp"
	}

	return defaultSchema
}
