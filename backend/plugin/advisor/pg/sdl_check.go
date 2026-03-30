package pg

import (
	"errors"
	"fmt"

	"github.com/bytebase/omni/pg/ast"

	"github.com/bytebase/bytebase/backend/plugin/advisor/code"

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
	if err := validateParenBalance(statement); err != nil {
		return nil, err
	}

	stmts, err := pgparser.ParsePg(statement)
	if err != nil {
		return nil, err
	}

	var adviceList []*storepb.Advice

	for _, stmt := range stmts {
		if stmt.AST == nil {
			continue
		}
		line := int32(stmt.Start.Line)

		switch n := stmt.AST.(type) {
		case *ast.CreateStmt:
			adviceList = checkCreateStmt(statement, n, line, adviceList)
		case *ast.IndexStmt:
			adviceList = checkIndexStmt(statement, n, line, adviceList)
		case *ast.ViewStmt:
			adviceList = checkViewStmt(n, line, adviceList)
		case *ast.CreateSeqStmt:
			adviceList = checkCreateSeqStmt(n, line, adviceList)
		case *ast.CreateFunctionStmt:
			adviceList = checkCreateFunctionStmt(n, line, adviceList)
		case *ast.AlterSeqStmt:
			adviceList = checkAlterSeqStmt(n, line, adviceList)
		default:
		}
	}

	return adviceList, nil
}

func nodeLineOrDefault(statement string, locStart int, defaultLine int32) int32 {
	if locStart >= 0 {
		pos := pgparser.ByteOffsetToRunePosition(statement, locStart)
		if pos.Line > 0 {
			return pos.Line
		}
	}
	return defaultLine
}

func checkCreateStmt(statement string, n *ast.CreateStmt, stmtLine int32, adviceList []*storepb.Advice) []*storepb.Advice {
	tableName := ""
	schemaName := ""
	if n.Relation != nil {
		tableName = n.Relation.Relname
		schemaName = n.Relation.Schemaname
	}

	// Check: Require explicit schema name
	if schemaName == "" {
		adviceList = append(adviceList, &storepb.Advice{
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
				Line:   stmtLine,
				Column: 0,
			},
		})
	}

	cols, tableConstraints := omniTableElements(n)

	// Check column-level constraints
	for _, col := range cols {
		columnName := col.Colname
		constraints := omniColumnConstraints(col)

		var disallowedTypes []string
		var firstConstraintLine int32

		for _, c := range constraints {
			typeName := constraintTypeName(c.Contype)
			if typeName == "" {
				continue
			}
			if disallowedTypes == nil {
				firstConstraintLine = nodeLineOrDefault(statement, c.Loc.Start, stmtLine)
			}
			disallowedTypes = append(disallowedTypes, typeName)

			// Also check FK reference schema for column-level FK
			if c.Contype == ast.CONSTR_FOREIGN && c.Pktable != nil && c.Pktable.Schemaname == "" {
				refTableName := c.Pktable.Relname
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
				adviceList = append(adviceList, &storepb.Advice{
					Status:  storepb.Advice_ERROR,
					Code:    code.SDLRequireSchemaName.Int32(),
					Title:   "sdl.require-schema-name",
					Content: content,
					StartPosition: &storepb.Position{
						Line:   nodeLineOrDefault(statement, c.Loc.Start, stmtLine),
						Column: 0,
					},
				})
			}
		}

		if len(disallowedTypes) > 0 {
			exampleConstraint := disallowedTypes[0]
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
				columnName, tableName, disallowedTypes,
				tableName, columnName, exampleConstraint,
				tableName, columnName, exampleConstraint, columnName,
			)

			adviceList = append(adviceList, &storepb.Advice{
				Status:  storepb.Advice_ERROR,
				Code:    code.SDLDisallowColumnConstraint.Int32(),
				Title:   "sdl.disallow-column-constraint",
				Content: content,
				StartPosition: &storepb.Position{
					Line:   firstConstraintLine,
					Column: 0,
				},
			})
		}
	}

	// Check table-level constraints
	for _, c := range tableConstraints {
		// Check unnamed constraint
		if c.Conname == "" {
			constraintType := tableConstraintTypeName(c.Contype)
			constraintExample := tableConstraintExample(c.Contype)

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
				tableName, constraintExample,
				tableName, constraintExample,
			)

			adviceList = append(adviceList, &storepb.Advice{
				Status:  storepb.Advice_ERROR,
				Code:    code.SDLRequireConstraintName.Int32(),
				Title:   "sdl.require-constraint-name",
				Content: content,
				StartPosition: &storepb.Position{
					Line:   nodeLineOrDefault(statement, c.Loc.Start, stmtLine),
					Column: 0,
				},
			})
		}

		// Check FK reference schema
		if c.Contype == ast.CONSTR_FOREIGN && c.Pktable != nil && c.Pktable.Schemaname == "" {
			refTableName := c.Pktable.Relname
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

			adviceList = append(adviceList, &storepb.Advice{
				Status:  storepb.Advice_ERROR,
				Code:    code.SDLRequireSchemaName.Int32(),
				Title:   "sdl.require-schema-name",
				Content: content,
				StartPosition: &storepb.Position{
					Line:   nodeLineOrDefault(statement, c.Loc.Start, stmtLine),
					Column: 0,
				},
			})
		}
	}

	return adviceList
}

func checkIndexStmt(_ string, n *ast.IndexStmt, stmtLine int32, adviceList []*storepb.Advice) []*storepb.Advice {
	indexName := n.Idxname

	// Check: Disallow unnamed indexes
	if indexName == "" {
		adviceList = append(adviceList, &storepb.Advice{
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
				Line:   stmtLine,
				Column: 0,
			},
		})
	}

	// Check: Require schema name on table
	if n.Relation != nil {
		schemaName := n.Relation.Schemaname
		tableName := n.Relation.Relname

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
				content = fmt.Sprintf(
					"Index is created on table '%s' without explicit schema name.\n\n"+
						"SDL requires all objects to specify their schema explicitly.\n\n"+
						"Example - Use:\n"+
						"  CREATE INDEX idx_name ON schema_name.%s (...);",
					tableName,
					tableName,
				)
			}

			adviceList = append(adviceList, &storepb.Advice{
				Status:  storepb.Advice_ERROR,
				Code:    code.SDLRequireSchemaName.Int32(),
				Title:   "sdl.require-schema-name",
				Content: content,
				StartPosition: &storepb.Position{
					Line:   stmtLine,
					Column: 0,
				},
			})
		}
	}

	return adviceList
}

func checkViewStmt(n *ast.ViewStmt, stmtLine int32, adviceList []*storepb.Advice) []*storepb.Advice {
	if n.View == nil {
		return adviceList
	}
	schemaName := n.View.Schemaname
	viewName := n.View.Relname
	if schemaName == "" {
		adviceList = append(adviceList, &storepb.Advice{
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
				Line:   stmtLine,
				Column: 0,
			},
		})
	}
	return adviceList
}

func checkCreateSeqStmt(n *ast.CreateSeqStmt, stmtLine int32, adviceList []*storepb.Advice) []*storepb.Advice {
	if n.Sequence == nil {
		return adviceList
	}
	schemaName := n.Sequence.Schemaname
	sequenceName := n.Sequence.Relname
	if schemaName == "" {
		adviceList = append(adviceList, &storepb.Advice{
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
				Line:   stmtLine,
				Column: 0,
			},
		})
	}
	return adviceList
}

func checkCreateFunctionStmt(n *ast.CreateFunctionStmt, stmtLine int32, adviceList []*storepb.Advice) []*storepb.Advice {
	schemaName := omniExtractSchemaFromFuncname(n.Funcname)
	parts := omniListStrings(n.Funcname)
	functionName := ""
	if len(parts) > 0 {
		functionName = parts[len(parts)-1]
	}
	if schemaName == "" {
		adviceList = append(adviceList, &storepb.Advice{
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
				Line:   stmtLine,
				Column: 0,
			},
		})
	}
	return adviceList
}

func checkAlterSeqStmt(n *ast.AlterSeqStmt, stmtLine int32, adviceList []*storepb.Advice) []*storepb.Advice {
	if n.Sequence == nil {
		return adviceList
	}
	schemaName := n.Sequence.Schemaname
	sequenceName := n.Sequence.Relname
	if schemaName == "" {
		adviceList = append(adviceList, &storepb.Advice{
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
				Line:   stmtLine,
				Column: 0,
			},
		})
	}
	return adviceList
}

// constraintTypeName returns the display name for disallowed column-level constraints.
// Returns empty string for allowed constraints (NOT NULL, DEFAULT, GENERATED, NULL).
func constraintTypeName(contype ast.ConstrType) string {
	switch contype {
	case ast.CONSTR_PRIMARY:
		return "PRIMARY KEY"
	case ast.CONSTR_UNIQUE:
		return "UNIQUE"
	case ast.CONSTR_CHECK:
		return "CHECK"
	case ast.CONSTR_FOREIGN:
		return "FOREIGN KEY"
	default:
		return ""
	}
}

// tableConstraintTypeName returns the display name for a table-level constraint type.
func tableConstraintTypeName(contype ast.ConstrType) string {
	switch contype {
	case ast.CONSTR_PRIMARY:
		return "PRIMARY KEY"
	case ast.CONSTR_UNIQUE:
		return "UNIQUE"
	case ast.CONSTR_CHECK:
		return "CHECK"
	case ast.CONSTR_FOREIGN:
		return "FOREIGN KEY"
	default:
		return "CONSTRAINT"
	}
}

// tableConstraintExample returns an example SQL snippet for a given constraint type.
func tableConstraintExample(contype ast.ConstrType) string {
	switch contype {
	case ast.CONSTR_PRIMARY:
		return "PRIMARY KEY (id)"
	case ast.CONSTR_UNIQUE:
		return "UNIQUE (column)"
	case ast.CONSTR_CHECK:
		return "CHECK (condition)"
	case ast.CONSTR_FOREIGN:
		return "FOREIGN KEY (column) REFERENCES other_table(id)"
	default:
		return "..."
	}
}

// validateParenBalance checks for unmatched parentheses in SQL, skipping string
// literals, dollar-quoted strings, and comments. This catches syntax errors that
// the omni parser may recover from silently.
func validateParenBalance(sql string) error {
	depth := 0
	i := 0
	for i < len(sql) {
		ch := sql[i]
		switch {
		case ch == '-' && i+1 < len(sql) && sql[i+1] == '-':
			// Line comment: skip to end of line
			for i < len(sql) && sql[i] != '\n' {
				i++
			}
		case ch == '/' && i+1 < len(sql) && sql[i+1] == '*':
			// Block comment: skip to */
			i += 2
			for i+1 < len(sql) && (sql[i] != '*' || sql[i+1] != '/') {
				i++
			}
			if i+1 < len(sql) {
				i += 2
			}
		case ch == '\'':
			// Single-quoted string: skip to closing quote (handle escaped quotes)
			i++
			for i < len(sql) {
				if sql[i] == '\'' {
					if i+1 < len(sql) && sql[i+1] == '\'' {
						i += 2
						continue
					}
					break
				}
				i++
			}
			i++
		case ch == '$':
			// Dollar-quoted string: find tag and skip to closing tag
			j := i + 1
			for j < len(sql) && (sql[j] == '_' || (sql[j] >= 'a' && sql[j] <= 'z') || (sql[j] >= 'A' && sql[j] <= 'Z') || (sql[j] >= '0' && sql[j] <= '9')) {
				j++
			}
			if j < len(sql) && sql[j] == '$' {
				tag := sql[i : j+1]
				i = j + 1
				// Find closing tag
				for i+len(tag) <= len(sql) {
					if sql[i] == '$' && sql[i:i+len(tag)] == tag {
						i += len(tag)
						break
					}
					i++
				}
			} else {
				i++
			}
		case ch == '(':
			depth++
			i++
		case ch == ')':
			depth--
			if depth < 0 {
				return errors.New("syntax error: unmatched closing parenthesis")
			}
			i++
		default:
			i++
		}
	}
	if depth != 0 {
		return errors.New("syntax error: unmatched opening parenthesis")
	}
	return nil
}
