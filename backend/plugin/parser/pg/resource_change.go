package pg

import (
	"slices"
	"strings"

	"github.com/bytebase/omni/pg/ast"
	"github.com/pkg/errors"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/store/model"
)

func init() {
	base.RegisterExtractChangedResourcesFunc(storepb.Engine_POSTGRES, extractChangedResources)
}

func extractChangedResources(database string, currentSchema string, dbMetadata *model.DatabaseMetadata, asts []base.AST, _ string) (*base.ChangeSummary, error) {
	changedResources := model.NewChangedResources(dbMetadata)
	searchPath := dbMetadata.GetSearchPath()
	if currentSchema != "" {
		searchPath = searchPathForSelectedSchema(currentSchema, searchPath)
	}
	if len(searchPath) == 0 {
		searchPath = []string{"public"}
	}

	if len(asts) == 0 {
		return &base.ChangeSummary{
			ChangedResources: changedResources,
			DMLCount:         0,
			DMLStatements:    []string{},
			InsertCount:      0,
		}, nil
	}

	var dmlCount, insertCount int
	var dmlStatements []string
	initialSearchPath := searchPath
	// sessionSearchPath leaves out SET LOCAL, which lasts until the transaction ends, and
	// transactionSearchPath is the session search path that ROLLBACK restores.
	sessionSearchPath, transactionSearchPath := searchPath, searchPath
	inTransaction := false
	sampleDML := func(text string) {
		dmlCount++
		text = strings.TrimSpace(text)
		if !strings.HasSuffix(text, ";") {
			text += ";"
		}
		// The EXPLAIN connection starts with the initial search path, so a changed one is replayed.
		if !slices.Equal(searchPath, initialSearchPath) {
			text = base.WithSearchPath(text, searchPath)
		}
		dmlStatements = append(dmlStatements, text)
	}
	addTarget := func(rv *ast.RangeVar, affectedTable bool) {
		db, schema, table := extractExistingRangeVarNames(rv, database, searchPath, dbMetadata)
		changedResources.AddTable(db, schema, &storepb.ChangedResourceTable{Name: table}, affectedTable)
	}
	// addDataModifyingCTETargets reports whether the WITH clause modifies data.
	addDataModifyingCTETargets := func(with *ast.WithClause) bool {
		targets := getDataModifyingCTETargets(with)
		for _, rv := range targets {
			addTarget(rv, false)
		}
		return len(targets) > 0
	}

	for _, unifiedAST := range asts {
		omniAST, ok := unifiedAST.(*OmniAST)
		if !ok {
			return nil, errors.New("expected OmniAST for PostgreSQL")
		}
		if omniAST.Node == nil {
			continue
		}
		node, text := UnwrapExplainAnalyze(omniAST.Node, omniAST.Text)

		switch n := node.(type) {
		case *ast.TransactionStmt:
			switch n.Kind {
			case ast.TRANS_STMT_BEGIN, ast.TRANS_STMT_START:
				// A BEGIN inside a transaction only warns.
				if !inTransaction {
					transactionSearchPath, inTransaction = sessionSearchPath, true
				}
			case ast.TRANS_STMT_COMMIT, ast.TRANS_STMT_ROLLBACK:
				// A ROLLBACK outside a transaction only warns.
				if n.Kind == ast.TRANS_STMT_ROLLBACK && inTransaction {
					sessionSearchPath = transactionSearchPath
				}
				// COMMIT AND CHAIN and ROLLBACK AND CHAIN start the next transaction from here.
				searchPath, transactionSearchPath, inTransaction = sessionSearchPath, sessionSearchPath, n.Chain
			default:
			}

		case *ast.DiscardStmt:
			if n.Target == ast.DISCARD_ALL {
				searchPath, sessionSearchPath = initialSearchPath, initialSearchPath
			}

		case *ast.VariableSetStmt:
			var newSearchPath []string
			if n.Kind == ast.VAR_RESET_ALL || (strings.EqualFold(n.Name, "search_path") && (n.Kind == ast.VAR_RESET || n.Kind == ast.VAR_SET_DEFAULT)) {
				newSearchPath = initialSearchPath
			} else if strings.EqualFold(n.Name, "search_path") && n.Kind == ast.VAR_SET_CURRENT {
				// SET ... FROM CURRENT makes the current search path, which a SET LOCAL may have set, the session's.
				newSearchPath = searchPath
			} else if strings.EqualFold(n.Name, "search_path") && n.Args != nil {
				for _, arg := range n.Args.Items {
					if ac, ok := arg.(*ast.A_Const); ok {
						if s, ok := ac.Val.(*ast.String); ok {
							newSearchPath = append(newSearchPath, s.Str)
						}
					}
				}
			}
			if len(newSearchPath) > 0 {
				searchPath = newSearchPath
				if !n.IsLocal {
					sessionSearchPath = newSearchPath
				}
			}

		case *ast.CreateStmt:
			if n.Relation != nil {
				db, schema, table := extractNewRangeVarNames(n.Relation, database, searchPath, dbMetadata)
				changedResources.AddTable(db, schema, &storepb.ChangedResourceTable{Name: table}, false)
			}

		case *ast.DropStmt:
			objType := ast.ObjectType(n.RemoveType)
			switch objType {
			case ast.OBJECT_INDEX:
				handleDropIndexOmni(n, database, searchPath, dbMetadata, changedResources)
			case ast.OBJECT_TABLE, ast.OBJECT_MATVIEW:
				handleDropTableOmni(n, database, searchPath, dbMetadata, changedResources)
			default:
			}

		case *ast.AlterTableStmt:
			if ast.ObjectType(n.ObjType) == ast.OBJECT_VIEW {
				continue
			}
			if n.Relation != nil {
				db, schema, table := extractExistingRangeVarNames(n.Relation, database, searchPath, dbMetadata)
				changedResources.AddTable(db, schema, &storepb.ChangedResourceTable{Name: table}, true)
			}

		case *ast.RenameStmt:
			if n.Relation == nil {
				continue
			}
			// Newname names a table only when the table itself is renamed.
			switch n.RenameType {
			case ast.OBJECT_TABLE, ast.OBJECT_MATVIEW:
				db, schema, oldTableName := extractExistingRangeVarNames(n.Relation, database, searchPath, dbMetadata)
				changedResources.AddTable(db, schema, &storepb.ChangedResourceTable{Name: oldTableName}, true)
				if n.Newname != "" {
					changedResources.AddTable(db, schema, &storepb.ChangedResourceTable{Name: n.Newname}, false)
				}
			case ast.OBJECT_COLUMN, ast.OBJECT_TABCONSTRAINT:
				if n.RelationType == ast.OBJECT_TABLE || n.RelationType == ast.OBJECT_MATVIEW {
					addTarget(n.Relation, true)
				}
			case ast.OBJECT_TRIGGER, ast.OBJECT_RULE, ast.OBJECT_POLICY:
				addTarget(n.Relation, true)
			default:
			}

		case *ast.IndexStmt:
			if n.Relation != nil {
				db, schema, table := extractExistingRangeVarNames(n.Relation, database, searchPath, dbMetadata)
				changedResources.AddTable(db, schema, &storepb.ChangedResourceTable{Name: table}, false)
			}

		case *ast.InsertStmt:
			hasDataModifyingCTE := addDataModifyingCTETargets(n.WithClause)
			if n.Relation != nil {
				addTarget(n.Relation, false)
			}
			// A data-modifying CTE makes the whole statement a sample, whose estimate already
			// includes the VALUES rows.
			if rows, ok := getInsertValuesRowCount(n); ok && !hasDataModifyingCTE {
				insertCount += rows
			} else {
				sampleDML(text)
			}

		case *ast.UpdateStmt:
			addDataModifyingCTETargets(n.WithClause)
			if n.Relation != nil {
				addTarget(n.Relation, false)
			}
			sampleDML(text)

		case *ast.DeleteStmt:
			addDataModifyingCTETargets(n.WithClause)
			if n.Relation != nil {
				addTarget(n.Relation, false)
			}
			sampleDML(text)

		case *ast.TruncateStmt:
			if n.Relations != nil {
				for _, item := range n.Relations.Items {
					rv, ok := item.(*ast.RangeVar)
					if !ok {
						continue
					}
					db, schema, table := extractExistingRangeVarNames(rv, database, searchPath, dbMetadata)
					changedResources.AddTable(db, schema, &storepb.ChangedResourceTable{Name: table}, true)
				}
			}

		case *ast.MergeStmt:
			addDataModifyingCTETargets(n.WithClause)
			if n.Relation != nil {
				addTarget(n.Relation, false)
			}
			sampleDML(text)

		case *ast.CreateTableAsStmt:
			if n.Into != nil && n.Into.Rel != nil {
				db, schema, table := extractNewRangeVarNames(n.Into.Rel, database, searchPath, dbMetadata)
				changedResources.AddTable(db, schema, &storepb.ChangedResourceTable{Name: table}, false)
			}
			if query, ok := n.Query.(*ast.SelectStmt); ok && addDataModifyingCTETargets(query.WithClause) {
				sampleDML(text)
			}

		case *ast.SelectStmt:
			// SELECT ... INTO target_table is a write (classified DDL via the INTO clause,
			// which may sit on the first arm of a set operation).
			if into := omniIntoClause(n); into != nil && into.Rel != nil {
				db, schema, table := extractNewRangeVarNames(into.Rel, database, searchPath, dbMetadata)
				changedResources.AddTable(db, schema, &storepb.ChangedResourceTable{Name: table}, false)
			}
			if addDataModifyingCTETargets(n.WithClause) {
				sampleDML(text)
			}

		default:
		}
	}

	return &base.ChangeSummary{
		ChangedResources: changedResources,
		DMLCount:         dmlCount,
		DMLStatements:    dmlStatements,
		InsertCount:      insertCount,
	}, nil
}

// getInsertValuesRowCount returns the rows inserted by INSERT ... VALUES or INSERT ... DEFAULT
// VALUES; ok is false when a query supplies the rows.
func getInsertValuesRowCount(n *ast.InsertStmt) (rows int, ok bool) {
	if n.SelectStmt == nil {
		return 1, true
	}
	if sel, isSelect := n.SelectStmt.(*ast.SelectStmt); isSelect && sel.ValuesLists != nil {
		return len(sel.ValuesLists.Items), true
	}
	return 0, false
}

// getWithClause returns the WITH clause of a statement, the only one where PostgreSQL allows
// data-modifying statements.
func getWithClause(node ast.Node) *ast.WithClause {
	switch n := node.(type) {
	case *ast.SelectStmt:
		return n.WithClause
	case *ast.InsertStmt:
		return n.WithClause
	case *ast.UpdateStmt:
		return n.WithClause
	case *ast.DeleteStmt:
		return n.WithClause
	case *ast.MergeStmt:
		return n.WithClause
	case *ast.CreateTableAsStmt:
		return getWithClause(n.Query)
	default:
		return nil
	}
}

// getDataModifyingCTETargets returns the tables that INSERT, UPDATE, DELETE, and MERGE statements in
// a WITH clause modify. PostgreSQL only allows data-modifying statements in a top-level WITH.
func getDataModifyingCTETargets(with *ast.WithClause) []*ast.RangeVar {
	if with == nil || with.Ctes == nil {
		return nil
	}
	var targets []*ast.RangeVar
	for _, item := range with.Ctes.Items {
		cte, ok := item.(*ast.CommonTableExpr)
		if !ok {
			continue
		}
		var target *ast.RangeVar
		switch query := cte.Ctequery.(type) {
		case *ast.InsertStmt:
			target = query.Relation
		case *ast.UpdateStmt:
			target = query.Relation
		case *ast.DeleteStmt:
			target = query.Relation
		case *ast.MergeStmt:
			target = query.Relation
		default:
			continue
		}
		if target != nil {
			targets = append(targets, target)
		}
	}
	return targets
}

// extractRangeVarNames extracts database, schema, table from a RangeVar with defaults.
func extractRangeVarNames(rv *ast.RangeVar, defaultDB string, searchPath []string) (string, string, string) {
	db := rv.Catalogname
	schema := rv.Schemaname
	table := rv.Relname
	if db == "" {
		db = defaultDB
	}
	if schema == "" && len(searchPath) > 0 {
		schema = searchPath[0]
	}
	return db, schema, table
}

func extractExistingRangeVarNames(rv *ast.RangeVar, defaultDB string, searchPath []string, dbMetadata *model.DatabaseMetadata) (string, string, string) {
	db := rv.Catalogname
	schema := rv.Schemaname
	table := rv.Relname
	if db == "" {
		db = defaultDB
	}
	if schema == "" {
		schemaName := searchRelationObject(dbMetadata, searchPath, table)
		if schemaName != "" {
			schema = schemaName
		} else if len(searchPath) > 0 {
			schema = searchPath[0]
		}
	}
	return db, schema, table
}

func extractNewRangeVarNames(rv *ast.RangeVar, defaultDB string, searchPath []string, dbMetadata *model.DatabaseMetadata) (string, string, string) {
	db, schema, table := extractRangeVarNames(rv, defaultDB, searchPath)
	if rv.Schemaname == "" && len(searchPath) > 1 && dbMetadata.GetSchemaMetadata(schema) == nil {
		schema = ""
	}
	return db, schema, table
}

func searchRelationObject(dbMetadata *model.DatabaseMetadata, searchPath []string, name string) string {
	for _, schemaName := range searchPath {
		schema := dbMetadata.GetSchemaMetadata(schemaName)
		if schema == nil {
			continue
		}
		if schema.GetTable(name) != nil || schema.GetView(name) != nil || schema.GetMaterializedView(name) != nil || schema.GetExternalTable(name) != nil {
			return schema.GetProto().GetName()
		}
	}
	return ""
}

// handleDropTableOmni handles DROP TABLE/MATERIALIZED VIEW.
func handleDropTableOmni(n *ast.DropStmt, database string, searchPath []string, dbMetadata *model.DatabaseMetadata, changedResources *model.ChangedResources) {
	if n.Objects == nil {
		return
	}
	for _, item := range n.Objects.Items {
		nameList, ok := item.(*ast.List)
		if !ok {
			continue
		}
		db, schema, name := extractNameListParts(nameList, database)
		if schema == "" {
			schemaName := searchRelationObject(dbMetadata, searchPath, name)
			if schemaName == "" {
				if len(searchPath) > 0 {
					schema = searchPath[0]
				}
			} else {
				schema = schemaName
			}
		}
		changedResources.AddTable(db, schema, &storepb.ChangedResourceTable{Name: name}, true)
	}
}

// handleDropIndexOmni handles DROP INDEX.
func handleDropIndexOmni(n *ast.DropStmt, database string, searchPath []string, dbMetadata *model.DatabaseMetadata, changedResources *model.ChangedResources) {
	if n.Objects == nil {
		return
	}
	for _, item := range n.Objects.Items {
		nameList, ok := item.(*ast.List)
		if !ok {
			continue
		}
		db, schema, indexName := extractNameListParts(nameList, database)

		lookupPath := searchPath
		if schema != "" {
			lookupPath = []string{schema}
		}
		schemaName, indexMetadata := dbMetadata.SearchIndex(lookupPath, indexName)
		if indexMetadata != nil && schemaName != "" {
			tableProto := indexMetadata.GetTableProto()
			if tableProto != nil {
				changedResources.AddTable(db, schemaName, &storepb.ChangedResourceTable{Name: tableProto.GetName()}, false)
			}
		}
	}
}

// extractNameListParts extracts db, schema, name from a list of String nodes.
func extractNameListParts(nameList *ast.List, defaultDB string) (string, string, string) {
	var parts []string
	for _, item := range nameList.Items {
		if s, ok := item.(*ast.String); ok {
			parts = append(parts, s.Str)
		}
	}
	switch len(parts) {
	case 1:
		return defaultDB, "", parts[0]
	case 2:
		return defaultDB, parts[0], parts[1]
	case 3:
		return parts[0], parts[1], parts[2]
	default:
		return defaultDB, "", ""
	}
}
