package starrocks

import (
	"strings"

	"github.com/bytebase/omni/starrocks/ast"
	"github.com/pkg/errors"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/store/model"
)

func init() {
	base.RegisterExtractChangedResourcesFunc(storepb.Engine_STARROCKS, extractChangedResources)
}

func extractChangedResources(currentDatabase string, _ string, dbMetadata *model.DatabaseMetadata, asts []base.AST, _ string) (*base.ChangeSummary, error) {
	changedResources := model.NewChangedResources(dbMetadata)

	var dmlCount, insertCount int
	var dmlStatements []string

	addTable := func(name *ast.ObjectName, affectsExistingRows bool) {
		db, table := objectName(name, currentDatabase)
		if table == "" {
			return
		}
		changedResources.AddTable(db, "", &storepb.ChangedResourceTable{Name: table}, affectsExistingRows)
	}

	// addObjectDatabase records a database-only write target for a non-table object DDL
	// (view, materialized view, routine) ONLY when the object's own name carries an
	// explicit database qualifier. An unqualified name records nothing, leaving the
	// request database to stand in. SUP-222.
	addObjectDatabase := func(name *ast.ObjectName) {
		if db, _ := objectName(name, ""); db != "" {
			changedResources.AddDatabase(db)
		}
	}

	for _, unifiedAST := range asts {
		parsed, ok := unifiedAST.(*omniAST)
		if !ok {
			return nil, errors.New("expected omniAST for StarRocks")
		}
		if parsed.node == nil {
			continue
		}

		switch n := parsed.node.(type) {
		case *ast.CreateTableStmt:
			addTable(n.Name, false)

		case *ast.DropTableStmt:
			addTable(n.Name, true)

		case *ast.TruncateTableStmt:
			addTable(n.Target, true)

		case *ast.AlterTableStmt:
			addTable(n.Name, true)

		case *ast.CreateIndexStmt:
			addTable(n.Table, false)

		case *ast.DropIndexStmt:
			addTable(n.Table, false)

		case *ast.LoadDataStmt:
			// One LOAD LABEL carries a data description per target table.
			for _, desc := range n.DataDescs {
				if desc != nil {
					addTable(desc.Target, false)
				}
			}

		case *ast.InsertStmt:
			// INSERT INTO FILES(...) writes to external storage and leaves Target nil,
			// so it records no table while still counting as a DML statement.
			addTable(n.Target, false)
			if len(n.Values) > 0 {
				insertCount += len(n.Values)
				continue
			}
			dmlCount++
			dmlStatements = append(dmlStatements, statementText(parsed))

		case *ast.UpdateStmt:
			// StarRocks updates a single target; FROM names read sources, not write targets.
			addTable(n.Target, false)
			dmlCount++
			dmlStatements = append(dmlStatements, statementText(parsed))

		case *ast.DeleteStmt:
			// USING names read sources, as FROM does for UPDATE.
			addTable(n.Target, false)
			dmlCount++
			dmlStatements = append(dmlStatements, statementText(parsed))

		case *ast.MergeStmt:
			addTable(n.Target, false)
			dmlCount++
			dmlStatements = append(dmlStatements, statementText(parsed))

		// Non-table object DDL: gate by the object's own explicit database qualifier only.
		// CREATE FUNCTION is absent because its AST keeps the name inside RawSignature
		// rather than as an ObjectName, so there is nothing to qualify by.
		case *ast.CreateViewStmt:
			addObjectDatabase(n.Name)
		case *ast.AlterViewStmt:
			addObjectDatabase(n.Name)
		case *ast.DropViewStmt:
			addObjectDatabase(n.Name)
		case *ast.CreateMTMVStmt:
			addObjectDatabase(n.Name)
		case *ast.AlterMTMVStmt:
			addObjectDatabase(n.Name)
		case *ast.DropMTMVStmt:
			addObjectDatabase(n.Name)
		case *ast.DropFunctionStmt:
			addObjectDatabase(n.Name)
		case *ast.CreateProcedureStmt:
			addObjectDatabase(n.Name)
		case *ast.DropProcedureStmt:
			addObjectDatabase(n.Name)

		default:
		}
	}

	return &base.ChangeSummary{
		ChangedResources: changedResources,
		DMLStatements:    dmlStatements,
		DMLCount:         dmlCount,
		InsertCount:      insertCount,
	}, nil
}

// objectName splits an ObjectName into its database and object name. StarRocks writes a
// name as table, db.table or catalog.db.table, so the object is the last part and the
// database the one before it; a catalog qualifier is dropped, since changed resources
// are tracked per database.
func objectName(name *ast.ObjectName, defaultDatabase string) (string, string) {
	if name == nil || len(name.Parts) == 0 {
		return defaultDatabase, ""
	}
	parts := name.Parts
	database := defaultDatabase
	if len(parts) >= 2 {
		database = parts[len(parts)-2]
	}
	return database, parts[len(parts)-1]
}

// statementText returns the text of a statement, ensuring it ends with a semicolon.
func statementText(a *omniAST) string {
	text := strings.TrimSpace(a.text)
	if !strings.HasSuffix(text, ";") {
		text += ";"
	}
	return text
}
