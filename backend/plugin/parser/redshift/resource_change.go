package redshift

import (
	"strings"

	omniredshift "github.com/bytebase/omni/redshift"
	redshiftast "github.com/bytebase/omni/redshift/ast"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/store/model"
)

func init() {
	base.RegisterExtractChangedResourcesFunc(storepb.Engine_REDSHIFT, extractChangedResources)
}

func extractChangedResources(database string, schema string, dbMetadata *model.DatabaseMetadata, _ []base.AST, statement string) (*base.ChangeSummary, error) {
	if schema == "" {
		schema = "public"
	}
	omniSummary, err := omniredshift.ExtractChangedResources(statement, database, schema)
	if err != nil {
		return nil, err
	}
	stmts, err := ParseRedshift(statement)
	if err != nil {
		return nil, err
	}
	ddlAffected, err := ddlAffectedTables(stmts, database, schema)
	if err != nil {
		return nil, err
	}

	summary := &base.ChangeSummary{ChangedResources: model.NewChangedResources(dbMetadata)}
	for _, table := range omniSummary.Tables {
		summary.ChangedResources.AddTable(
			table.Database,
			table.Schema,
			&storepb.ChangedResourceTable{Name: table.Name},
			ddlAffected[tableKey{database: table.Database, schema: table.Schema, name: table.Name}],
		)
	}

	searchPath := schema
	// replaySearchPath is the search path set after the starting one, which the EXPLAIN connection
	// does not have.
	var replaySearchPath []string
	sampleDML := func(text string) {
		summary.DMLCount++
		text = strings.TrimSpace(text)
		if len(replaySearchPath) > 0 {
			text = base.WithSearchPath(text, replaySearchPath)
		}
		summary.DMLStatements = append(summary.DMLStatements, text)
	}
	addTable := func(rv *redshiftast.RangeVar, affectedTable bool) {
		tableDatabase, tableSchema := rv.Catalogname, rv.Schemaname
		if tableDatabase == "" {
			tableDatabase = database
		}
		if tableSchema == "" {
			tableSchema = searchPath
		}
		summary.ChangedResources.AddTable(tableDatabase, tableSchema, &storepb.ChangedResourceTable{Name: rv.Relname}, affectedTable)
	}
	for _, stmt := range stmts {
		switch n := stmt.AST.(type) {
		case *redshiftast.InsertStmt:
			if rows, ok := insertValuesRowCount(n); ok {
				summary.InsertCount += rows
			} else {
				sampleDML(stmt.Text)
			}
		case *redshiftast.UpdateStmt, *redshiftast.DeleteStmt:
			sampleDML(stmt.Text)
		case *redshiftast.MergeStmt:
			// Redshift documents EXPLAIN only for SELECT, CREATE TABLE AS, INSERT, UPDATE, and DELETE.
			summary.DMLCount++
			// omni records no MERGE target.
			if n.Relation != nil {
				addTable(n.Relation, false)
			}
		case *redshiftast.TruncateStmt:
			if n.Relations == nil {
				continue
			}
			for _, item := range n.Relations.Items {
				if rv, ok := item.(*redshiftast.RangeVar); ok {
					addTable(rv, true)
				}
			}
		case *redshiftast.VariableSetStmt:
			if schemas, ok := searchPathFromSet(n); ok {
				searchPath, replaySearchPath = schema, schemas
				if len(schemas) > 0 {
					searchPath = schemas[0]
				}
			}
		default:
		}
	}
	return summary, nil
}

type tableKey struct {
	database string
	schema   string
	name     string
}

// ddlAffectedTables returns the tables whose existing rows the script's DDL affects.
// omni also flags DML targets and merges flags across statements, so the script it
// reads here leaves out the DML.
func ddlAffectedTables(stmts []omniredshift.Statement, database, schema string) (map[tableKey]bool, error) {
	var texts []string
	for _, stmt := range stmts {
		switch stmt.AST.(type) {
		case *redshiftast.InsertStmt, *redshiftast.UpdateStmt, *redshiftast.DeleteStmt, *redshiftast.MergeStmt:
		default:
			texts = append(texts, stmt.Text)
		}
	}
	// A statement's text lacks its semicolon when a comment precedes the semicolon.
	summary, err := omniredshift.ExtractChangedResources(strings.Join(texts, ";\n"), database, schema)
	if err != nil {
		return nil, err
	}
	affected := make(map[tableKey]bool)
	for _, table := range summary.Tables {
		if table.Affected {
			affected[tableKey{database: table.Database, schema: table.Schema, name: table.Name}] = true
		}
	}
	return affected, nil
}

// insertValuesRowCount returns how many rows INSERT ... VALUES or INSERT ... DEFAULT
// VALUES adds, and false for an INSERT whose rows come from a query.
func insertValuesRowCount(n *redshiftast.InsertStmt) (int, bool) {
	if n.SelectStmt == nil {
		return 1, true
	}
	if s, ok := n.SelectStmt.(*redshiftast.SelectStmt); ok && s.ValuesLists != nil {
		return len(s.ValuesLists.Items), true
	}
	return 0, false
}

// searchPathFromSet returns the schemas a SET search_path names, skipping "$user", or none for RESET
// or SET ... TO DEFAULT, which restore the starting search path. ok is false for any other statement.
// Like omni's ExtractChangedResources, unqualified names resolve to the first schema.
func searchPathFromSet(n *redshiftast.VariableSetStmt) ([]string, bool) {
	if !strings.EqualFold(n.Name, "search_path") {
		return nil, false
	}
	if n.Kind == redshiftast.VAR_SET_DEFAULT || n.Kind == redshiftast.VAR_RESET {
		return nil, true
	}
	if n.Args == nil {
		return nil, false
	}
	var schemas []string
	for _, arg := range n.Args.Items {
		c, ok := arg.(*redshiftast.A_Const)
		if !ok {
			continue
		}
		s, ok := c.Val.(*redshiftast.String)
		if !ok || s.Str == "" || s.Str == "$user" {
			continue
		}
		schemas = append(schemas, s.Str)
	}
	return schemas, len(schemas) > 0
}
