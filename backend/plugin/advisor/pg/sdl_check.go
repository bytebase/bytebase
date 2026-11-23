package pg

import (
	"errors"
	"fmt"
	"strings"

	pganalysis "github.com/bytebase/omni/pg/analysis"

	"github.com/bytebase/bytebase/backend/plugin/advisor/code"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

// CheckSDLStyle performs SDL style/convention checks on PostgreSQL declarative schema files.
// The judgment is omni's pganalysis.CheckSDLStyle; this renders its findings.
//
// For integrity checks (foreign key validation, duplicate detection, etc.), use CheckSDLIntegrity from sdl_integrity_check.go.
func CheckSDLStyle(statement string) ([]*storepb.Advice, error) {
	findings := pganalysis.CheckSDLStyle(statement)
	if len(findings) == 1 && findings[0].Code == pganalysis.SDLSyntaxError {
		return nil, errors.New(findings[0].Detail)
	}

	var adviceList []*storepb.Advice
	for _, f := range findings {
		adviceList = append(adviceList, sdlStyleAdvice(f, sdlStyleLine(statement, f)))
	}
	return adviceList, nil
}

// sdlStyleLine is the line of the finding's node, or of its statement when
// it has none. A statement's line counts only LF line breaks, while a node's
// also counts CR, as the checker always has.
func sdlStyleLine(statement string, f pganalysis.SDLFinding) int32 {
	if f.Start >= 0 {
		return base.ByteOffsetToRunePosition(statement, f.Start).Line
	}
	return int32(strings.Count(statement[:f.StatementStart], "\n") + 1)
}

func sdlStyleAdvice(f pganalysis.SDLFinding, line int32) *storepb.Advice {
	advice := &storepb.Advice{
		Status:        storepb.Advice_ERROR,
		Code:          code.SDLRequireSchemaName.Int32(),
		Title:         "sdl.require-schema-name",
		StartPosition: &storepb.Position{Line: line, Column: 0},
	}
	switch f.Code {
	case pganalysis.SDLTableWithoutSchema:
		advice.Content = sdlObjectWithoutSchema("Table", f.Table, "created", "CREATE TABLE", " (...);")
	case pganalysis.SDLColumnForeignKeyWithoutSchema:
		advice.Content = fmt.Sprintf(
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
			f.Column, f.Table, f.RefTable,
			f.Table, f.Column, f.RefTable,
			f.Table, f.Column, f.Column, f.RefTable,
		)
	case pganalysis.SDLColumnConstraint:
		exampleConstraint := f.ConstraintTypes[0]
		advice.Code = code.SDLDisallowColumnConstraint.Int32()
		advice.Title = "sdl.disallow-column-constraint"
		advice.Content = fmt.Sprintf(
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
			f.Column, f.Table, f.ConstraintTypes,
			f.Table, f.Column, exampleConstraint,
			f.Table, f.Column, exampleConstraint, f.Column,
		)
	case pganalysis.SDLUnnamedConstraint:
		constraintExample := tableConstraintExample(f.ConstraintType)
		advice.Code = code.SDLRequireConstraintName.Int32()
		advice.Title = "sdl.require-constraint-name"
		advice.Content = fmt.Sprintf(
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
			f.Table, f.ConstraintType,
			f.Table, constraintExample,
			f.Table, constraintExample,
		)
	case pganalysis.SDLForeignKeyWithoutSchema:
		advice.Content = fmt.Sprintf(
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
			f.Table, f.RefTable,
			f.Table, f.RefTable,
			f.Table, f.RefTable,
		)
	case pganalysis.SDLUnnamedIndex:
		advice.Code = code.SDLRequireIndexName.Int32()
		advice.Title = "sdl.require-index-name"
		advice.Content = "Index is created without an explicit name.\n\n" +
			"SDL requires all indexes to have explicit names.\n\n" +
			"Example - Instead of:\n" +
			"  CREATE INDEX ON table_name (column);\n\n" +
			"Use:\n" +
			"  CREATE INDEX idx_table_column ON table_name (column);"
	case pganalysis.SDLIndexWithoutSchema:
		if f.Name != "" {
			advice.Content = fmt.Sprintf(
				"Index '%s' is created on table '%s' without explicit schema name.\n\n"+
					"SDL requires all objects to specify their schema explicitly.\n\n"+
					"Example - Instead of:\n"+
					"  CREATE INDEX %s ON %s (...);\n\n"+
					"Use:\n"+
					"  CREATE INDEX %s ON schema_name.%s (...);",
				f.Name, f.Table,
				f.Name, f.Table,
				f.Name, f.Table,
			)
		} else {
			advice.Content = fmt.Sprintf(
				"Index is created on table '%s' without explicit schema name.\n\n"+
					"SDL requires all objects to specify their schema explicitly.\n\n"+
					"Example - Use:\n"+
					"  CREATE INDEX idx_name ON schema_name.%s (...);",
				f.Table,
				f.Table,
			)
		}
	case pganalysis.SDLViewWithoutSchema:
		advice.Content = sdlObjectWithoutSchema("View", f.Name, "created", "CREATE VIEW", " AS ...;")
	case pganalysis.SDLSequenceWithoutSchema:
		advice.Content = sdlObjectWithoutSchema("Sequence", f.Name, "created", "CREATE SEQUENCE", " ...;")
	case pganalysis.SDLFunctionWithoutSchema:
		advice.Content = sdlObjectWithoutSchema("Function", f.Name, "created", "CREATE FUNCTION", "(...) ...;")
	case pganalysis.SDLAlterSequenceWithoutSchema:
		advice.Content = sdlObjectWithoutSchema("Sequence", f.Name, "altered", "ALTER SEQUENCE", " ...;")
	default:
	}
	return advice
}

func sdlObjectWithoutSchema(kind, name, action, stmtPrefix, stmtSuffix string) string {
	return fmt.Sprintf(
		"%s '%s' is %s without explicit schema name.\n\n"+
			"SDL requires all objects to specify their schema explicitly.\n\n"+
			"Example - Instead of:\n"+
			"  %s %s%s\n\n"+
			"Use:\n"+
			"  %s schema_name.%s%s",
		kind, name, action,
		stmtPrefix, name, stmtSuffix,
		stmtPrefix, name, stmtSuffix,
	)
}

// tableConstraintExample returns an example SQL snippet for a constraint type name.
func tableConstraintExample(constraintType string) string {
	switch constraintType {
	case "PRIMARY KEY":
		return "PRIMARY KEY (id)"
	case "UNIQUE":
		return "UNIQUE (column)"
	case "CHECK":
		return "CHECK (condition)"
	case "FOREIGN KEY":
		return "FOREIGN KEY (column) REFERENCES other_table(id)"
	default:
		return "..."
	}
}
