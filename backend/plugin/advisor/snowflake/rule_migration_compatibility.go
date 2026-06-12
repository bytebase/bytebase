// Package snowflake is the advisor for snowflake database.
package snowflake

import (
	"context"
	"fmt"
	"strings"

	omniast "github.com/bytebase/omni/snowflake/ast"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	snowsqlparser "github.com/bytebase/bytebase/backend/plugin/parser/snowflake"
)

var (
	_ advisor.Advisor = (*MigrationCompatibilityAdvisor)(nil)
)

func init() {
	advisor.Register(storepb.Engine_SNOWFLAKE, storepb.SQLReviewRule_SCHEMA_BACKWARD_COMPATIBILITY, &MigrationCompatibilityAdvisor{})
}

// MigrationCompatibilityAdvisor is the advisor checking for migration compatibility.
type MigrationCompatibilityAdvisor struct {
}

// Check checks for migration compatibility.
func (*MigrationCompatibilityAdvisor) Check(_ context.Context, checkCtx advisor.Context) ([]*storepb.Advice, error) {
	level, err := advisor.NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return nil, err
	}

	checker := &migrationCompatibilityChecker{
		level:                              level,
		title:                              checkCtx.Rule.Type.String(),
		currentDatabase:                    checkCtx.CurrentDatabase,
		normalizedNewCreateTableNameMap:    make(map[string]bool),
		normalizedNewCreateSchemaNameMap:   make(map[string]bool),
		normalizedNewCreateDatabaseNameMap: make(map[string]bool),
	}

	for _, stmt := range checkCtx.ParsedStatements {
		node, ok := snowsqlparser.GetOmniNode(stmt.AST)
		if !ok {
			continue
		}
		checker.checkStmt(node, stmt.Text, stmt.BaseLine())
	}

	return checker.adviceList, nil
}

// migrationCompatibilityChecker checks for migration compatibility issues.
type migrationCompatibilityChecker struct {
	adviceList []*storepb.Advice
	level      storepb.Advice_Status
	title      string

	// normalizedNewCreateTableNameMap contain the new created table name in normalized format, e.g. "SNOWFLAKE.PUBLIC.TABLE", If there are IF NOT EXISTS, the value will be false.
	normalizedNewCreateTableNameMap map[string]bool
	// normalizedNewCreateSchemaNameMap contain the new created schema name in normalized format, e.g. "SNOWFLAKE.PUBLIC", If there are IF NOT EXISTS, the value will be false.
	normalizedNewCreateSchemaNameMap map[string]bool
	// normalizedNewCreateDatabaseNameMap contain the new created database name in normalized format, e.g. "SNOWFLAKE", If there are IF NOT EXISTS, the value will be false.
	normalizedNewCreateDatabaseNameMap map[string]bool

	// currentDatabase is the current database name.
	currentDatabase string
}

// checkStmt dispatches one top-level statement node, mirroring the legacy
// listener's context cases: CREATE TABLE [AS SELECT] / CREATE SCHEMA /
// CREATE DATABASE record new objects; DROP TABLE / DROP SCHEMA /
// DROP DATABASE / ALTER TABLE ... DROP COLUMN produce compatibility advices.
func (c *migrationCompatibilityChecker) checkStmt(node omniast.Node, text string, baseLine int) {
	switch stmt := node.(type) {
	case *omniast.CreateTableStmt:
		// Covers both CREATE TABLE and CREATE TABLE ... AS SELECT (the legacy
		// grammar had two contexts; omni folds CTAS into CreateTableStmt).
		c.normalizedNewCreateTableNameMap[c.normalizeTableName(stmt.Name)] = !stmt.IfNotExists
	case *omniast.CreateSchemaStmt:
		c.normalizedNewCreateSchemaNameMap[c.normalizeSchemaName(stmt.Name)] = !stmt.IfNotExists
	case *omniast.CreateDatabaseStmt:
		c.normalizedNewCreateDatabaseNameMap[normalizeDatabaseName(stmt.Name)] = !stmt.IfNotExists
	case *omniast.DropStmt:
		if stmt.Kind != omniast.DropTable {
			return
		}
		normalizedFullDropTableName := c.normalizeTableName(stmt.Name)
		mustNewCreate, ok := c.normalizedNewCreateTableNameMap[normalizedFullDropTableName]
		if ok && mustNewCreate {
			return
		}
		level := c.level
		if ok && !mustNewCreate {
			level = storepb.Advice_WARNING
		}
		c.adviceList = append(c.adviceList, &storepb.Advice{
			Status:        level,
			Code:          code.CompatibilityDropTable.Int32(),
			Title:         c.title,
			Content:       fmt.Sprintf("Drop table %q may cause incompatibility with the existing data and code", normalizedFullDropTableName),
			StartPosition: common.ConvertANTLRLineToPosition(baseLine + statementLineForOffset(text, stmt.Loc.Start)),
		})
	case *omniast.DropSchemaStmt:
		normalizedFullDropSchemaName := c.normalizeSchemaName(stmt.Name)
		mustNewCreate, ok := c.normalizedNewCreateSchemaNameMap[normalizedFullDropSchemaName]
		if ok && mustNewCreate {
			return
		}
		level := c.level
		if ok && !mustNewCreate {
			level = storepb.Advice_WARNING
		}
		c.adviceList = append(c.adviceList, &storepb.Advice{
			Status:        level,
			Code:          code.CompatibilityDropSchema.Int32(),
			Title:         c.title,
			Content:       fmt.Sprintf("Drop schema %q may cause incompatibility with the existing data and code", normalizedFullDropSchemaName),
			StartPosition: common.ConvertANTLRLineToPosition(baseLine + statementLineForOffset(text, stmt.Loc.Start)),
		})
	case *omniast.DropDatabaseStmt:
		normalizedFullDropDatabaseName := normalizeDatabaseName(stmt.Name)
		mustNewCreate, ok := c.normalizedNewCreateDatabaseNameMap[normalizedFullDropDatabaseName]
		if ok && mustNewCreate {
			return
		}
		level := c.level
		if ok && !mustNewCreate {
			level = storepb.Advice_WARNING
		}
		c.adviceList = append(c.adviceList, &storepb.Advice{
			Status:        level,
			Code:          code.CompatibilityDropDatabase.Int32(),
			Title:         c.title,
			Content:       fmt.Sprintf("Drop database %q may cause incompatibility with the existing data and code", normalizedFullDropDatabaseName),
			StartPosition: common.ConvertANTLRLineToPosition(baseLine + statementLineForOffset(text, stmt.Loc.Start)),
		})
	case *omniast.AlterTableStmt:
		for _, action := range stmt.Actions {
			if action == nil || action.Kind != omniast.AlterTableDropColumn {
				continue
			}
			normalizedAllColumnNames := make([]string, 0, len(action.DropColumnNames))
			for _, columnName := range action.DropColumnNames {
				if columnName.IsEmpty() {
					continue
				}
				normalizedAllColumnNames = append(normalizedAllColumnNames, fmt.Sprintf("%q", columnName.Normalize()))
			}
			c.adviceList = append(c.adviceList, &storepb.Advice{
				Status:        c.level,
				Code:          code.CompatibilityDropColumn.Int32(),
				Title:         c.title,
				Content:       fmt.Sprintf("Drop column %s may cause incompatibility with the existing data and code", strings.Join(normalizedAllColumnNames, ",")),
				StartPosition: common.ConvertANTLRLineToPosition(baseLine + statementLineForOffset(text, stmt.Loc.Start)),
			})
		}
	default:
		// Other statement types
	}
}

// normalizeTableName mirrors the legacy NormalizeSnowSQLObjectName(name,
// currentDatabase, "PUBLIC") for omni object names: each present part is
// folded per Snowflake rules (unquoted uppercased, quoted verbatim), missing
// database/schema parts fall back to the current database and PUBLIC.
func (c *migrationCompatibilityChecker) normalizeTableName(name *omniast.ObjectName) string {
	database := c.currentDatabase
	schema := "PUBLIC"
	var table string
	if name != nil {
		if d := name.Database.Normalize(); d != "" {
			database = d
		}
		if s := name.Schema.Normalize(); s != "" {
			schema = s
		}
		table = name.Name.Normalize()
	}
	parts := []string{database, schema}
	if table != "" {
		parts = append(parts, table)
	}
	return strings.Join(parts, ".")
}

// normalizeSchemaName mirrors the legacy NormalizeSnowSQLSchemaName(name,
// currentDatabase) for omni object names. A 2-part schema name parses as
// Schema.Name (database.schema); a 1-part name parses as just Name (schema).
func (c *migrationCompatibilityChecker) normalizeSchemaName(name *omniast.ObjectName) string {
	database := c.currentDatabase
	var schema string
	if name != nil {
		if d := name.Schema.Normalize(); d != "" {
			database = d
		}
		schema = name.Name.Normalize()
	}
	return strings.Join([]string{database, schema}, ".")
}

// normalizeDatabaseName mirrors the legacy NormalizeSnowSQLObjectNamePart on
// the single identifier of CREATE/DROP DATABASE.
func normalizeDatabaseName(name *omniast.ObjectName) string {
	if name == nil {
		return ""
	}
	return name.Name.Normalize()
}
