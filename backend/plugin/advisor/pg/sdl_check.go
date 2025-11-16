package pg

import (
	"fmt"

	"github.com/bytebase/bytebase/backend/plugin/advisor/code"

	"github.com/antlr4-go/antlr/v4"
	parser "github.com/bytebase/parser/postgresql"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	pgparser "github.com/bytebase/bytebase/backend/plugin/parser/pg"
)

// CheckSDLStyle performs SDL style/convention checks on PostgreSQL declarative schema files.
//
// Style Checks:
// 1. Disallowed column-level constraints (PRIMARY KEY, UNIQUE, CHECK, FOREIGN KEY must be table-level; NOT NULL, DEFAULT, GENERATED are allowed)
// 2. Table constraints without explicit names
// 3. Objects without explicit schema names (tables, indexes, views, sequences, functions)
// 4. Foreign key references without explicit schema names
// 5. Indexes without explicit names (unnamed indexes are not allowed)
//
// For integrity checks (foreign key validation, duplicate detection, etc.), use CheckSDLIntegrity from sdl_integrity_check.go.
func CheckSDLStyle(statement string) ([]*storepb.Advice, error) {
	tree, err := pgparser.ParsePostgreSQL(statement)
	if err != nil {
		return nil, err
	}

	// Run style/convention checks
	styleChecker := &sdlChecker{
		BasePostgreSQLParserListener: &parser.BasePostgreSQLParserListener{},
	}
	antlr.ParseTreeWalkerDefault.Walk(styleChecker, tree.Tree)

	return styleChecker.adviceList, nil
}

type sdlChecker struct {
	*parser.BasePostgreSQLParserListener
	adviceList []*storepb.Advice
}

// EnterCreatestmt handles CREATE TABLE statements.
func (c *sdlChecker) EnterCreatestmt(ctx *parser.CreatestmtContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}

	var tableName, schemaName string
	allQualifiedNames := ctx.AllQualified_name()
	if len(allQualifiedNames) > 0 {
		schemaName = extractSchemaName(allQualifiedNames[0])
		tableName = extractTableName(allQualifiedNames[0])
	}

	// Check 3: Require explicit schema name
	if schemaName == "" {
		c.adviceList = append(c.adviceList, &storepb.Advice{
			Status: storepb.Advice_ERROR,
			Code:   code.SDLRequireSchemaName.Int32(),
			Title:  "sdl.require-schema-name",
			Content: fmt.Sprintf(
				"Table '%s' is created without explicit schema name.\n\n"+
					"SDL requires all objects to specify their schema explicitly.\n\n"+
					"Example - Instead of:\n"+
					"  CREATE TABLE %s (...);\n\n"+
					"Use:\n"+
					"  CREATE TABLE schema_name.%s (...);",
				tableName, tableName, tableName,
			),
			StartPosition: &storepb.Position{
				Line:   int32(ctx.GetStart().GetLine()),
				Column: 0,
			},
		})
	}

	// Check all column definitions and table constraints
	if ctx.Opttableelementlist() != nil && ctx.Opttableelementlist().Tableelementlist() != nil {
		allElements := ctx.Opttableelementlist().Tableelementlist().AllTableelement()
		for _, elem := range allElements {
			// Check 1: Disallow column-level constraints
			if elem.ColumnDef() != nil {
				c.checkColumnConstraints(elem.ColumnDef(), tableName)
			}
			// Check 2: Require constraint names for table constraints
			if elem.Tableconstraint() != nil {
				c.checkTableConstraintName(elem.Tableconstraint(), tableName)
				// Check 3: Check foreign key reference tables have schema names
				c.checkForeignKeyReferenceSchema(elem.Tableconstraint(), tableName)
			}
		}
	}
}

// EnterIndexstmt handles CREATE INDEX
func (c *sdlChecker) EnterIndexstmt(ctx *parser.IndexstmtContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}

	// Get index name if available
	var indexName string
	if ctx.Name() != nil && ctx.Name().Colid() != nil {
		indexName = pgparser.NormalizePostgreSQLColid(ctx.Name().Colid())
	}

	// Check 1: Disallow unnamed indexes
	if indexName == "" {
		c.adviceList = append(c.adviceList, &storepb.Advice{
			Status: storepb.Advice_ERROR,
			Code:   code.SDLRequireIndexName.Int32(),
			Title:  "sdl.require-index-name",
			Content: "Index is created without an explicit name.\n\n" +
				"SDL requires all indexes to have explicit names.\n\n" +
				"Example - Instead of:\n" +
				"  CREATE INDEX ON table_name (column);\n\n" +
				"Use:\n" +
				"  CREATE INDEX idx_table_column ON table_name (column);",
			StartPosition: &storepb.Position{
				Line:   int32(ctx.GetStart().GetLine()),
				Column: 0,
			},
		})
	}

	// Check 2: Require schema name on table
	if ctx.Relation_expr() != nil && ctx.Relation_expr().Qualified_name() != nil {
		schemaName := extractSchemaName(ctx.Relation_expr().Qualified_name())
		tableName := extractTableName(ctx.Relation_expr().Qualified_name())

		if schemaName == "" {
			var content string
			if indexName != "" {
				content = fmt.Sprintf(
					"Index '%s' is created on table '%s' without explicit schema name.\n\n"+
						"SDL requires all objects to specify their schema explicitly.\n\n"+
						"Example - Instead of:\n"+
						"  CREATE INDEX %s ON %s (...);\n\n"+
						"Use:\n"+
						"  CREATE INDEX %s ON schema_name.%s (...);",
					indexName, tableName,
					indexName, tableName,
					indexName, tableName,
				)
			} else {
				// Unnamed index (already reported above, but still check table schema)
				content = fmt.Sprintf(
					"Index is created on table '%s' without explicit schema name.\n\n"+
						"SDL requires all objects to specify their schema explicitly.\n\n"+
						"Example - Use:\n"+
						"  CREATE INDEX idx_name ON schema_name.%s (...);",
					tableName,
					tableName,
				)
			}

			c.adviceList = append(c.adviceList, &storepb.Advice{
				Status:  storepb.Advice_ERROR,
				Code:    code.SDLRequireSchemaName.Int32(),
				Title:   "sdl.require-schema-name",
				Content: content,
				StartPosition: &storepb.Position{
					Line:   int32(ctx.GetStart().GetLine()),
					Column: 0,
				},
			})
		}
	}
}

// EnterViewstmt handles CREATE VIEW
func (c *sdlChecker) EnterViewstmt(ctx *parser.ViewstmtContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}

	if ctx.Qualified_name() != nil {
		schemaName := extractSchemaName(ctx.Qualified_name())
		viewName := extractTableName(ctx.Qualified_name())
		if schemaName == "" {
			c.adviceList = append(c.adviceList, &storepb.Advice{
				Status: storepb.Advice_ERROR,
				Code:   code.SDLRequireSchemaName.Int32(),
				Title:  "sdl.require-schema-name",
				Content: fmt.Sprintf(
					"View '%s' is created without explicit schema name.\n\n"+
						"SDL requires all objects to specify their schema explicitly.\n\n"+
						"Example - Instead of:\n"+
						"  CREATE VIEW %s AS ...;\n\n"+
						"Use:\n"+
						"  CREATE VIEW schema_name.%s AS ...;",
					viewName, viewName, viewName,
				),
				StartPosition: &storepb.Position{
					Line:   int32(ctx.GetStart().GetLine()),
					Column: 0,
				},
			})
		}
	}
}

// EnterCreateseqstmt handles CREATE SEQUENCE
func (c *sdlChecker) EnterCreateseqstmt(ctx *parser.CreateseqstmtContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}

	if ctx.Qualified_name() != nil {
		schemaName := extractSchemaName(ctx.Qualified_name())
		sequenceName := extractTableName(ctx.Qualified_name())
		if schemaName == "" {
			c.adviceList = append(c.adviceList, &storepb.Advice{
				Status: storepb.Advice_ERROR,
				Code:   code.SDLRequireSchemaName.Int32(),
				Title:  "sdl.require-schema-name",
				Content: fmt.Sprintf(
					"Sequence '%s' is created without explicit schema name.\n\n"+
						"SDL requires all objects to specify their schema explicitly.\n\n"+
						"Example - Instead of:\n"+
						"  CREATE SEQUENCE %s ...;\n\n"+
						"Use:\n"+
						"  CREATE SEQUENCE schema_name.%s ...;",
					sequenceName, sequenceName, sequenceName,
				),
				StartPosition: &storepb.Position{
					Line:   int32(ctx.GetStart().GetLine()),
					Column: 0,
				},
			})
		}
	}
}

// EnterCreatefunctionstmt handles CREATE FUNCTION
func (c *sdlChecker) EnterCreatefunctionstmt(ctx *parser.CreatefunctionstmtContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}

	if ctx.Func_name() != nil {
		schemaName, functionName := c.extractSchemaAndNameFromFuncName(ctx.Func_name())
		if schemaName == "" {
			c.adviceList = append(c.adviceList, &storepb.Advice{
				Status: storepb.Advice_ERROR,
				Code:   code.SDLRequireSchemaName.Int32(),
				Title:  "sdl.require-schema-name",
				Content: fmt.Sprintf(
					"Function '%s' is created without explicit schema name.\n\n"+
						"SDL requires all objects to specify their schema explicitly.\n\n"+
						"Example - Instead of:\n"+
						"  CREATE FUNCTION %s(...) ...;\n\n"+
						"Use:\n"+
						"  CREATE FUNCTION schema_name.%s(...) ...;",
					functionName, functionName, functionName,
				),
				StartPosition: &storepb.Position{
					Line:   int32(ctx.GetStart().GetLine()),
					Column: 0,
				},
			})
		}
	}
}

// EnterAlterseqstmt handles ALTER SEQUENCE
func (c *sdlChecker) EnterAlterseqstmt(ctx *parser.AlterseqstmtContext) {
	if !isTopLevel(ctx.GetParent()) {
		return
	}

	if ctx.Qualified_name() != nil {
		schemaName := extractSchemaName(ctx.Qualified_name())
		sequenceName := extractTableName(ctx.Qualified_name())
		if schemaName == "" {
			c.adviceList = append(c.adviceList, &storepb.Advice{
				Status: storepb.Advice_ERROR,
				Code:   code.SDLRequireSchemaName.Int32(),
				Title:  "sdl.require-schema-name",
				Content: fmt.Sprintf(
					"Sequence '%s' is altered without explicit schema name.\n\n"+
						"SDL requires all objects to specify their schema explicitly.\n\n"+
						"Example - Instead of:\n"+
						"  ALTER SEQUENCE %s ...;\n\n"+
						"Use:\n"+
						"  ALTER SEQUENCE schema_name.%s ...;",
					sequenceName, sequenceName, sequenceName,
				),
				StartPosition: &storepb.Position{
					Line:   int32(ctx.GetStart().GetLine()),
					Column: 0,
				},
			})
		}
	}
}

// checkColumnConstraints checks a column definition for disallowed constraints.
// SDL allows NOT NULL, DEFAULT, GENERATED at column level.
// SDL disallows PRIMARY KEY, UNIQUE, CHECK, FOREIGN KEY at column level.
func (c *sdlChecker) checkColumnConstraints(columnDef parser.IColumnDefContext, tableName string) {
	if columnDef == nil {
		return
	}

	var columnName string
	if columnDef.Colid() != nil {
		columnName = pgparser.NormalizePostgreSQLColid(columnDef.Colid())
	}

	// Check if column has any disallowed constraints
	if columnDef.Colquallist() != nil {
		allConstraints := columnDef.Colquallist().AllColconstraint()
		var disallowedConstraints []parser.IColconstraintContext
		var disallowedConstraintTypes []string

		for _, constraint := range allConstraints {
			if constraint.Colconstraintelem() != nil {
				elem := constraint.Colconstraintelem()
				// Check if this is a disallowed constraint type
				if c.isDisallowedColumnConstraint(elem) {
					disallowedConstraints = append(disallowedConstraints, constraint)
					disallowedConstraintTypes = append(disallowedConstraintTypes, c.getConstraintTypeName(elem))
				}
				// Also check FK reference schema even though FK is disallowed at column level
				// This provides a more specific error message
				c.checkColumnForeignKeyReferenceSchema(elem, columnName, tableName)
			}
		}

		if len(disallowedConstraints) > 0 {
			firstConstraint := disallowedConstraints[0]
			exampleConstraint := disallowedConstraintTypes[0]

			content := fmt.Sprintf(
				"Column '%s' in table '%s' has disallowed column-level constraint(s): %v.\n\n"+
					"SDL only allows NOT NULL, DEFAULT, and GENERATED constraints at the column level.\n"+
					"PRIMARY KEY, UNIQUE, CHECK, and FOREIGN KEY constraints must be defined at the table level.\n\n"+
					"Example - Instead of:\n"+
					"  CREATE TABLE %s (\n"+
					"    %s INTEGER %s\n"+
					"  );\n\n"+
					"Use table-level constraint:\n"+
					"  CREATE TABLE %s (\n"+
					"    %s INTEGER,\n"+
					"    CONSTRAINT constraint_name %s (%s)\n"+
					"  );",
				columnName, tableName, disallowedConstraintTypes,
				tableName, columnName, exampleConstraint,
				tableName, columnName, exampleConstraint, columnName,
			)

			c.adviceList = append(c.adviceList, &storepb.Advice{
				Status:  storepb.Advice_ERROR,
				Code:    code.SDLDisallowColumnConstraint.Int32(),
				Title:   "sdl.disallow-column-constraint",
				Content: content,
				StartPosition: &storepb.Position{
					Line:   int32(firstConstraint.GetStart().GetLine()),
					Column: 0,
				},
			})
		}
	}
}

// isDisallowedColumnConstraint returns true if the constraint is not allowed at column level in SDL.
// Allowed: NOT NULL, NULL, DEFAULT, GENERATED
// Disallowed: PRIMARY KEY, UNIQUE, CHECK, FOREIGN KEY
func (*sdlChecker) isDisallowedColumnConstraint(elem parser.IColconstraintelemContext) bool {
	if elem == nil {
		return false
	}
	// Disallowed constraints
	if elem.PRIMARY() != nil && elem.KEY() != nil {
		return true // PRIMARY KEY
	}
	if elem.UNIQUE() != nil {
		return true // UNIQUE
	}
	if elem.CHECK() != nil {
		return true // CHECK
	}
	if elem.REFERENCES() != nil {
		return true // FOREIGN KEY
	}
	// Allowed constraints: NOT NULL, NULL, DEFAULT, GENERATED
	return false
}

// checkTableConstraintName checks if a table constraint has an explicit name.
func (c *sdlChecker) checkTableConstraintName(constraint parser.ITableconstraintContext, tableName string) {
	if constraint == nil {
		return
	}

	// Check if constraint has a name using CONSTRAINT keyword
	hasConstraintName := false
	if constraint.CONSTRAINT() != nil {
		if constraint.Name() != nil && constraint.Name().Colid() != nil {
			hasConstraintName = true
		}
	}

	if !hasConstraintName {
		constraintType := c.getTableConstraintType(constraint)

		content := fmt.Sprintf(
			"Table '%s' has a %s constraint without an explicit name.\n\n"+
				"SDL requires all table constraints to have explicit names using the CONSTRAINT keyword.\n\n"+
				"Example - Instead of:\n"+
				"  CREATE TABLE %s (\n"+
				"    id INTEGER,\n"+
				"    %s\n"+
				"  );\n\n"+
				"Use named constraint:\n"+
				"  CREATE TABLE %s (\n"+
				"    id INTEGER,\n"+
				"    CONSTRAINT constraint_name %s\n"+
				"  );",
			tableName, constraintType,
			tableName, c.getConstraintExample(constraint, false),
			tableName, c.getConstraintExample(constraint, true),
		)

		c.adviceList = append(c.adviceList, &storepb.Advice{
			Status:  storepb.Advice_ERROR,
			Code:    code.SDLRequireConstraintName.Int32(),
			Title:   "sdl.require-constraint-name",
			Content: content,
			StartPosition: &storepb.Position{
				Line:   int32(constraint.GetStart().GetLine()),
				Column: 0,
			},
		})
	}
}

// checkForeignKeyReferenceSchema checks if a FOREIGN KEY constraint references a table with explicit schema name.
func (c *sdlChecker) checkForeignKeyReferenceSchema(constraint parser.ITableconstraintContext, tableName string) {
	if constraint == nil || constraint.Constraintelem() == nil {
		return
	}

	elem := constraint.Constraintelem()
	// Check if this is a FOREIGN KEY constraint
	if elem.FOREIGN() == nil || elem.KEY() == nil {
		return
	}

	// Check the referenced table (REFERENCES clause)
	if elem.Qualified_name() != nil {
		refSchemaName := extractSchemaName(elem.Qualified_name())
		refTableName := extractTableName(elem.Qualified_name())

		if refSchemaName == "" {
			content := fmt.Sprintf(
				"Foreign key constraint in table '%s' references table '%s' without explicit schema name.\n\n"+
					"SDL requires all table references in foreign keys to specify their schema explicitly.\n\n"+
					"Example - Instead of:\n"+
					"  CREATE TABLE %s (\n"+
					"    ...\n"+
					"    CONSTRAINT fk_name FOREIGN KEY (column) REFERENCES %s(id)\n"+
					"  );\n\n"+
					"Use:\n"+
					"  CREATE TABLE %s (\n"+
					"    ...\n"+
					"    CONSTRAINT fk_name FOREIGN KEY (column) REFERENCES schema_name.%s(id)\n"+
					"  );",
				tableName, refTableName,
				tableName, refTableName,
				tableName, refTableName,
			)

			c.adviceList = append(c.adviceList, &storepb.Advice{
				Status:  storepb.Advice_ERROR,
				Code:    code.SDLRequireSchemaName.Int32(),
				Title:   "sdl.require-schema-name",
				Content: content,
				StartPosition: &storepb.Position{
					Line:   int32(constraint.GetStart().GetLine()),
					Column: 0,
				},
			})
		}
	}
}

// checkColumnForeignKeyReferenceSchema checks if a column-level FOREIGN KEY references a table with explicit schema name.
func (c *sdlChecker) checkColumnForeignKeyReferenceSchema(elem parser.IColconstraintelemContext, columnName, tableName string) {
	if elem == nil {
		return
	}

	// Check if this is a FOREIGN KEY constraint (REFERENCES)
	if elem.REFERENCES() == nil {
		return
	}

	// Check the referenced table
	if elem.Qualified_name() != nil {
		refSchemaName := extractSchemaName(elem.Qualified_name())
		refTableName := extractTableName(elem.Qualified_name())

		if refSchemaName == "" {
			content := fmt.Sprintf(
				"Column '%s' in table '%s' has a foreign key reference to table '%s' without explicit schema name.\n\n"+
					"SDL requires all table references in foreign keys to specify their schema explicitly.\n"+
					"Note: Foreign keys should be defined at table level, not column level.\n\n"+
					"Example - Instead of:\n"+
					"  CREATE TABLE %s (\n"+
					"    %s INTEGER REFERENCES %s(id)\n"+
					"  );\n\n"+
					"Use table-level constraint with schema:\n"+
					"  CREATE TABLE %s (\n"+
					"    %s INTEGER,\n"+
					"    CONSTRAINT fk_name FOREIGN KEY (%s) REFERENCES schema_name.%s(id)\n"+
					"  );",
				columnName, tableName, refTableName,
				tableName, columnName, refTableName,
				tableName, columnName, columnName, refTableName,
			)

			c.adviceList = append(c.adviceList, &storepb.Advice{
				Status:  storepb.Advice_ERROR,
				Code:    code.SDLRequireSchemaName.Int32(),
				Title:   "sdl.require-schema-name",
				Content: content,
				StartPosition: &storepb.Position{
					Line:   int32(elem.GetStart().GetLine()),
					Column: 0,
				},
			})
		}
	}
}

// Helper functions

func (*sdlChecker) getConstraintTypeName(elem parser.IColconstraintelemContext) string {
	if elem == nil {
		return "UNKNOWN"
	}
	if elem.PRIMARY() != nil && elem.KEY() != nil {
		return "PRIMARY KEY"
	}
	if elem.UNIQUE() != nil {
		return "UNIQUE"
	}
	if elem.NOT() != nil && elem.NULL_P() != nil {
		return "NOT NULL"
	}
	if elem.NULL_P() != nil {
		return "NULL"
	}
	if elem.CHECK() != nil {
		return "CHECK"
	}
	if elem.DEFAULT() != nil {
		return "DEFAULT"
	}
	if elem.REFERENCES() != nil {
		return "FOREIGN KEY"
	}
	if elem.GENERATED() != nil {
		return "GENERATED"
	}
	return "CONSTRAINT"
}

func (*sdlChecker) getTableConstraintType(constraint parser.ITableconstraintContext) string {
	if constraint.Constraintelem() == nil {
		return "CONSTRAINT"
	}
	elem := constraint.Constraintelem()
	if elem.PRIMARY() != nil && elem.KEY() != nil {
		return "PRIMARY KEY"
	}
	if elem.UNIQUE() != nil {
		return "UNIQUE"
	}
	if elem.CHECK() != nil {
		return "CHECK"
	}
	if elem.FOREIGN() != nil && elem.KEY() != nil {
		return "FOREIGN KEY"
	}
	if elem.EXCLUDE() != nil {
		return "EXCLUDE"
	}
	return "CONSTRAINT"
}

func (*sdlChecker) getConstraintExample(constraint parser.ITableconstraintContext, _ bool) string {
	if constraint.Constraintelem() == nil {
		return "..."
	}
	elem := constraint.Constraintelem()
	if elem.PRIMARY() != nil && elem.KEY() != nil {
		return "PRIMARY KEY (id)"
	}
	if elem.UNIQUE() != nil {
		return "UNIQUE (column)"
	}
	if elem.CHECK() != nil {
		return "CHECK (condition)"
	}
	if elem.FOREIGN() != nil && elem.KEY() != nil {
		return "FOREIGN KEY (column) REFERENCES other_table(id)"
	}
	return "..."
}

func (*sdlChecker) extractSchemaAndNameFromFuncName(funcName parser.IFunc_nameContext) (string, string) {
	if funcName.Type_function_name() != nil {
		return "", funcName.Type_function_name().GetText()
	}
	if funcName.Indirection() != nil {
		parts := []string{}
		if funcName.Colid() != nil {
			parts = append(parts, funcName.Colid().GetText())
		}
		for _, attr := range funcName.Indirection().AllIndirection_el() {
			if attr.Attr_name() != nil {
				parts = append(parts, attr.Attr_name().GetText())
			}
		}
		if len(parts) == 1 {
			return "", parts[0]
		}
		if len(parts) >= 2 {
			return parts[0], parts[1]
		}
	}
	return "", ""
}
