package plsql

import (
	"strings"

	oracleast "github.com/bytebase/omni/oracle/ast"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/store/model"
)

func init() {
	base.RegisterExtractChangedResourcesFunc(storepb.Engine_ORACLE, extractChangedResources)
}

func extractChangedResources(currentDatabase string, _ string, dbMetadata *model.DatabaseMetadata, asts []base.AST, statement string) (*base.ChangeSummary, error) {
	// currentDatabase is the same as currentSchema for Oracle.
	changedResources := model.NewChangedResources(dbMetadata)
	extractor := &omniChangedResourceExtractor{
		currentSchema:    currentDatabase,
		dbMetadata:       dbMetadata,
		changedResources: changedResources,
		statement:        statement,
	}

	for _, ast := range asts {
		omniAST, ok := ast.(*OmniAST)
		if !ok {
			continue
		}
		extractor.extract(omniAST)
	}

	return &base.ChangeSummary{
		ChangedResources: changedResources,
		SampleDMLS:       extractor.sampleDMLs,
		DMLCount:         extractor.dmlCount,
		InsertCount:      extractor.insertCount,
	}, nil
}

type omniChangedResourceExtractor struct {
	currentSchema    string
	dbMetadata       *model.DatabaseMetadata
	changedResources *model.ChangedResources
	statement        string
	sampleDMLs       []string
	dmlCount         int
	insertCount      int
}

func (e *omniChangedResourceExtractor) extract(a *OmniAST) {
	if a == nil || a.Node == nil {
		return
	}

	switch n := a.Node.(type) {
	case *oracleast.CreateTableStmt:
		e.addTable(n.Name, false)
	case *oracleast.DropStmt:
		if n.ObjectType == oracleast.OBJECT_TABLE {
			for _, name := range omniObjectNameList(n.Names) {
				e.addTable(name, true)
			}
		}
		if n.ObjectType == oracleast.OBJECT_INDEX {
			for _, name := range omniObjectNameList(n.Names) {
				e.addIndexTable(name)
			}
		}
	case *oracleast.AlterTableStmt:
		affected := true
		for _, cmd := range omniAlterTableCmdList(n.Actions) {
			if cmd.Action == oracleast.AT_RENAME {
				affected = false
				break
			}
		}
		e.addTable(n.Name, affected)
	case *oracleast.CreateIndexStmt:
		e.addTable(n.Table, false)
	case *oracleast.InsertStmt:
		e.extractInsert(n, a.Text)
	case *oracleast.UpdateStmt:
		e.addTable(n.Table, false)
		e.trackDML(a.Text)
	case *oracleast.DeleteStmt:
		e.addTable(n.Table, false)
		e.trackDML(a.Text)
	default:
	}
}

func (e *omniChangedResourceExtractor) extractInsert(stmt *oracleast.InsertStmt, text string) {
	if stmt.Table != nil {
		e.addTable(stmt.Table, false)
	}
	for _, item := range omniInsertIntoList(stmt.MultiTable) {
		e.addTable(item.Table, false)
	}

	if stmt.Table != nil && stmt.Values != nil {
		e.insertCount++
		return
	}
	e.trackDML(text)
}

func (e *omniChangedResourceExtractor) addTable(name *oracleast.ObjectName, affected bool) {
	if name == nil || name.Name == "" {
		return
	}
	schema := name.Schema
	if schema == "" {
		schema = e.currentSchema
	}
	e.changedResources.AddTable(
		schema,
		"",
		&storepb.ChangedResourceTable{
			Name: name.Name,
		},
		affected,
	)
}

func (e *omniChangedResourceExtractor) addIndexTable(name *oracleast.ObjectName) {
	if e.dbMetadata == nil || name == nil || name.Name == "" {
		return
	}
	schema := name.Schema
	if schema == "" {
		schema = e.currentSchema
	}
	foundSchema := e.dbMetadata.GetSchemaMetadata(schema)
	if foundSchema == nil {
		return
	}
	foundIndex := foundSchema.GetIndex(name.Name)
	if foundIndex == nil {
		return
	}
	e.changedResources.AddTable(
		schema,
		"",
		&storepb.ChangedResourceTable{
			Name: foundIndex.GetTableProto().GetName(),
		},
		false,
	)
}

func (e *omniChangedResourceExtractor) trackDML(text string) {
	e.dmlCount++
	if len(e.sampleDMLs) < common.MaximumLintExplainSize {
		e.sampleDMLs = append(e.sampleDMLs, omniStatementText(text))
	}
}

func omniStatementText(text string) string {
	text = strings.TrimSpace(text)
	if strings.HasSuffix(text, ";") {
		return text
	}
	return text + ";"
}

func omniObjectNameList(list *oracleast.List) []*oracleast.ObjectName {
	if list == nil {
		return nil
	}
	names := make([]*oracleast.ObjectName, 0, len(list.Items))
	for _, item := range list.Items {
		if name, ok := item.(*oracleast.ObjectName); ok {
			names = append(names, name)
		}
	}
	return names
}

func omniAlterTableCmdList(list *oracleast.List) []*oracleast.AlterTableCmd {
	if list == nil {
		return nil
	}
	cmds := make([]*oracleast.AlterTableCmd, 0, len(list.Items))
	for _, item := range list.Items {
		if cmd, ok := item.(*oracleast.AlterTableCmd); ok {
			cmds = append(cmds, cmd)
		}
	}
	return cmds
}

func omniInsertIntoList(list *oracleast.List) []*oracleast.InsertIntoClause {
	if list == nil {
		return nil
	}
	clauses := make([]*oracleast.InsertIntoClause, 0, len(list.Items))
	for _, item := range list.Items {
		if clause, ok := item.(*oracleast.InsertIntoClause); ok {
			clauses = append(clauses, clause)
		}
	}
	return clauses
}
