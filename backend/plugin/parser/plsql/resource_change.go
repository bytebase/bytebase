package plsql

import (
	"strings"

	oracleast "github.com/bytebase/omni/oracle/ast"

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
	extractor := &changedResourceExtractor{
		currentSchema:    currentDatabase,
		dbMetadata:       dbMetadata,
		changedResources: changedResources,
		statement:        statement,
	}

	for _, ast := range asts {
		wrapped, ok := ast.(*AST)
		if !ok {
			continue
		}
		extractor.extract(wrapped)
	}

	return &base.ChangeSummary{
		ChangedResources: changedResources,
		DMLStatements:    extractor.dmlStatements,
		DMLCount:         extractor.dmlCount,
		InsertCount:      extractor.insertCount,
	}, nil
}

type changedResourceExtractor struct {
	currentSchema    string
	dbMetadata       *model.DatabaseMetadata
	changedResources *model.ChangedResources
	statement        string
	dmlStatements    []string
	dmlCount         int
	insertCount      int
}

func (e *changedResourceExtractor) extract(a *AST) {
	if a == nil || a.Node == nil {
		return
	}

	switch n := a.Node.(type) {
	case *oracleast.CreateTableStmt:
		e.addTable(n.Name, false)
	case *oracleast.DropStmt:
		if n.ObjectType == oracleast.OBJECT_TABLE {
			for _, name := range objectNameList(n.Names) {
				e.addTable(name, true)
			}
		}
		if n.ObjectType == oracleast.OBJECT_INDEX {
			for _, name := range objectNameList(n.Names) {
				e.addIndexTable(name)
			}
		}
		switch n.ObjectType {
		case oracleast.OBJECT_VIEW, oracleast.OBJECT_SEQUENCE, oracleast.OBJECT_SYNONYM,
			oracleast.OBJECT_PROCEDURE, oracleast.OBJECT_FUNCTION, oracleast.OBJECT_TRIGGER,
			oracleast.OBJECT_PACKAGE, oracleast.OBJECT_PACKAGE_BODY, oracleast.OBJECT_TYPE,
			oracleast.OBJECT_TYPE_BODY, oracleast.OBJECT_MATERIALIZED_VIEW, oracleast.OBJECT_CLUSTER:
			for _, name := range objectNameList(n.Names) {
				e.addObjectDatabase(name)
			}
		default:
		}
	case *oracleast.AlterTableStmt:
		affected := true
		for _, cmd := range alterTableCmdList(n.Actions) {
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
	case *oracleast.MergeStmt:
		e.addTable(n.Target, false)
		e.trackDML(a.Text)
	case *oracleast.TruncateStmt:
		e.addTable(n.Table, true)
	case *oracleast.CreateViewStmt:
		e.addObjectDatabase(n.Name)
	case *oracleast.CreateProcedureStmt:
		e.addObjectDatabase(n.Name)
	case *oracleast.CreateFunctionStmt:
		e.addObjectDatabase(n.Name)
	case *oracleast.CreateTriggerStmt:
		e.addObjectDatabase(n.Name)
	case *oracleast.CreateSequenceStmt:
		e.addObjectDatabase(n.Name)
	case *oracleast.CreateSynonymStmt:
		e.addObjectDatabase(n.Name)
	case *oracleast.AlterViewStmt:
		e.addObjectDatabase(n.Name)
	case *oracleast.AlterSequenceStmt:
		e.addObjectDatabase(n.Name)
	case *oracleast.AlterProcedureStmt:
		e.addObjectDatabase(n.Name)
	case *oracleast.AlterFunctionStmt:
		e.addObjectDatabase(n.Name)
	case *oracleast.AlterTriggerStmt:
		e.addObjectDatabase(n.Name)
	case *oracleast.AlterSynonymStmt:
		e.addObjectDatabase(n.Name)
	case *oracleast.CreatePackageStmt:
		e.addObjectDatabase(n.Name)
	case *oracleast.AlterPackageStmt:
		e.addObjectDatabase(n.Name)
	case *oracleast.CreateTypeStmt:
		e.addObjectDatabase(n.Name)
	case *oracleast.AlterTypeStmt:
		e.addObjectDatabase(n.Name)
	case *oracleast.CreateClusterStmt:
		e.addObjectDatabase(n.Name)
	case *oracleast.AlterMaterializedViewStmt:
		e.addObjectDatabase(n.Name)
	default:
	}
}

func (e *changedResourceExtractor) extractInsert(stmt *oracleast.InsertStmt, text string) {
	if stmt.Table != nil {
		e.addTable(stmt.Table, false)
	}
	for _, item := range insertIntoList(stmt.MultiTable) {
		e.addTable(item.Table, false)
	}

	if stmt.Table != nil && stmt.Values != nil {
		e.insertCount++
		return
	}
	e.trackDML(text)
}

// addObjectDatabase records a database-only write target for a non-table object DDL (view,
// procedure, function, trigger, sequence, synonym) ONLY when its name carries an explicit schema
// qualifier (Oracle's schema maps to a database). An unqualified name records nothing, keeping
// the request-database fallback. See SUP-222 / BYT-9698.
func (e *changedResourceExtractor) addObjectDatabase(name *oracleast.ObjectName) {
	if name != nil && name.Schema != "" {
		e.changedResources.AddDatabase(name.Schema)
	}
}

func (e *changedResourceExtractor) addTable(name *oracleast.ObjectName, affected bool) {
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

func (e *changedResourceExtractor) addIndexTable(name *oracleast.ObjectName) {
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

func (e *changedResourceExtractor) trackDML(text string) {
	e.dmlCount++
	e.dmlStatements = append(e.dmlStatements, statementText(text))
}

func statementText(text string) string {
	text = strings.TrimSpace(text)
	if strings.HasSuffix(text, ";") {
		return text
	}
	return text + ";"
}

func objectNameList(list *oracleast.List) []*oracleast.ObjectName {
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

func alterTableCmdList(list *oracleast.List) []*oracleast.AlterTableCmd {
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

func insertIntoList(list *oracleast.List) []*oracleast.InsertIntoClause {
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
