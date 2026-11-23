package pg

import (
	"fmt"
	"strings"

	pganalysis "github.com/bytebase/omni/pg/analysis"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

// CheckSDLIntegrity performs comprehensive integrity checks across SDL files.
// The judgment is omni's pganalysis.CheckSDLIntegrity; this renders its findings.
//
// Parameters:
//   - files: map[filePath]sqlContent - All SDL files to check together.
//     For single-file checking, use a map with one entry.
//
// Returns:
//   - map[filePath][]*storepb.Advice - Per-file advice list
//   - error - Parse or system errors
func CheckSDLIntegrity(files map[string]string) (map[string][]*storepb.Advice, error) {
	sdlFiles := make([]pganalysis.SDLFile, 0, len(files))
	for path, text := range files {
		sdlFiles = append(sdlFiles, pganalysis.SDLFile{Path: path, Text: text})
	}

	results := make(map[string][]*storepb.Advice)
	for path, findings := range pganalysis.CheckSDLIntegrity(sdlFiles) {
		advices := make([]*storepb.Advice, 0, len(findings))
		for _, f := range findings {
			advices = append(advices, sdlIntegrityAdvice(files, path, f))
		}
		results[path] = advices
	}
	return results, nil
}

// checkSingleStatement is a helper for backward compatibility with single-statement checks.
// It wraps the statement in a single-file map and extracts the results.
func checkSingleStatement(statement string) ([]*storepb.Advice, error) {
	const defaultFileName = "statement.sql"
	results, err := CheckSDLIntegrity(map[string]string{
		defaultFileName: statement,
	})
	if err != nil {
		return nil, err
	}
	return results[defaultFileName], nil
}

// sdlFileLine is the line of byte offset start in file, or 1 when unknown.
func sdlFileLine(files map[string]string, path string, start int) int {
	if start < 0 {
		return 1
	}
	return int(base.ByteOffsetToRunePosition(files[path], start).Line)
}

func sdlIntegrityAdvice(files map[string]string, path string, f pganalysis.SDLFinding) *storepb.Advice {
	line := sdlFileLine(files, path, f.Start)
	firstLine := 0
	if f.First != nil {
		firstPath := f.First.Path
		if firstPath == "" {
			firstPath = path
		}
		firstLine = sdlFileLine(files, firstPath, f.First.Start)
	}
	advice := &storepb.Advice{
		Status:        storepb.Advice_ERROR,
		StartPosition: &storepb.Position{Line: int32(line)},
	}
	switch f.Code {
	case pganalysis.SDLSyntaxError:
		advice.Code = code.StatementSyntaxError.Int32()
		advice.Title = "SQL syntax error"
		advice.Content = fmt.Sprintf("Failed to parse SQL in file '%s': %v", path, f.Detail)
		advice.StartPosition = nil
	case pganalysis.SDLDuplicateTable:
		advice.Code = code.SDLDuplicateTableName.Int32()
		advice.Title = "Duplicate table name"
		advice.Content = fmt.Sprintf(
			"Table '%s.%s' is defined multiple times in the SDL.\n\n"+
				"First definition at line %d\n"+
				"Duplicate definition at line %d\n\n"+
				"Each table can only be defined once per schema.",
			f.Schema, f.Table, firstLine, line,
		)
	case pganalysis.SDLDuplicateColumn:
		advice.Code = code.SDLDuplicateColumnName.Int32()
		advice.Title = "Duplicate column name"
		advice.Content = fmt.Sprintf(
			"Column '%s' is defined multiple times in table '%s.%s'.\n\n"+
				"Each column can only be defined once per table.",
			f.Column, f.Schema, f.Table,
		)
	case pganalysis.SDLDuplicateConstraint:
		scope := "table"
		scopeDetail := "PostgreSQL requires CHECK and FOREIGN KEY constraint names to be unique within a table."
		if f.ConstraintType == "PRIMARY KEY" || f.ConstraintType == "UNIQUE" {
			scope = "schema"
			scopeDetail = "PostgreSQL requires PRIMARY KEY and UNIQUE constraint names to be unique within a schema."
		}
		advice.Code = code.SDLDuplicateConstraintName.Int32()
		advice.Title = "Duplicate constraint name"
		advice.Content = fmt.Sprintf(
			"Constraint '%s' is defined multiple times in %s '%s'.\n\n"+
				"First definition: %s on table '%s' (line %d)\n"+
				"Duplicate definition: %s on table '%s' (line %d)\n\n"+
				"%s\n"+
				"Use different names or let PostgreSQL generate unique names automatically.",
			f.Name, scope, f.Schema,
			f.First.ConstraintType, f.First.Table, firstLine,
			f.ConstraintType, f.Table, line,
			scopeDetail,
		)
	case pganalysis.SDLDuplicateIndex:
		advice.Code = code.SDLDuplicateIndexName.Int32()
		advice.Title = "Duplicate index name"
		advice.Content = fmt.Sprintf(
			"Index '%s.%s' is defined multiple times in the SDL.\n\n"+
				"First definition at line %d\n"+
				"Duplicate definition at line %d\n\n"+
				"Each index name must be unique within a schema.",
			f.Schema, f.Name, firstLine, line,
		)
	case pganalysis.SDLDuplicateTableAcrossFiles:
		advice.Code = code.SDLDuplicateTableName.Int32()
		advice.Title, advice.Content = sdlDuplicateAcrossFiles("Table", "table", f.Schema, f.Table, f.First.Path, firstLine, path, line)
	case pganalysis.SDLDuplicateIndexAcrossFiles:
		advice.Code = code.SDLDuplicateIndexName.Int32()
		advice.Title, advice.Content = sdlDuplicateAcrossFiles("Index", "index", f.Schema, f.Name, f.First.Path, firstLine, path, line)
	case pganalysis.SDLDuplicateViewAcrossFiles:
		advice.Code = code.SDLDuplicateTableName.Int32()
		advice.Title, advice.Content = sdlDuplicateAcrossFiles("View", "view", f.Schema, f.Name, f.First.Path, firstLine, path, line)
	case pganalysis.SDLDuplicateConstraintAcrossFiles:
		advice.Code = code.SDLDuplicateConstraintName.Int32()
		advice.Title = "Duplicate constraint name across files"
		advice.Content = fmt.Sprintf(
			"Constraint '%s' in schema '%s' is defined in multiple SDL files.\n\n"+
				"First definition: %s on table '%s' in %s (line %d)\n"+
				"Duplicate definition: %s on table '%s' in %s (line %d)\n\n"+
				"PostgreSQL requires PRIMARY KEY and UNIQUE constraint names to be unique within a schema.\n"+
				"Each constraint must be defined in exactly one file in the SDL project.",
			f.Name, f.Schema,
			f.First.ConstraintType, f.First.Table, f.First.Path, firstLine,
			f.ConstraintType, f.Table, path, line,
		)
	case pganalysis.SDLMultiplePrimaryKeys:
		var pkDescriptions []string
		for i, pk := range f.PrimaryKeys {
			pkDescriptions = append(pkDescriptions, fmt.Sprintf("  %d. %s at line %d", i+1, sdlConstraintName(pk.Name), sdlFileLine(files, path, pk.Start)))
		}
		advice.Code = code.SDLMultiplePrimaryKey.Int32()
		advice.Title = "Multiple primary keys defined"
		advice.Content = fmt.Sprintf(
			"Table '%s.%s' has multiple PRIMARY KEY constraints defined.\n\n"+
				"Found %d primary key constraints:\n%s\n\n"+
				"A table can only have one PRIMARY KEY constraint.\n"+
				"If you need to enforce uniqueness on multiple column combinations, use UNIQUE constraints instead.",
			f.Schema, f.Table,
			len(f.PrimaryKeys),
			strings.Join(pkDescriptions, "\n"),
		)
	case pganalysis.SDLForeignKeyTableNotFound:
		advice.Code = code.SDLForeignKeyTableNotFound.Int32()
		advice.Title = "Foreign key references non-existent table"
		advice.Content = fmt.Sprintf(
			"Foreign key constraint '%s' on table '%s.%s' references table '%s.%s' which does not exist in any SDL file.\n\n"+
				"Foreign key columns: %s\n\n"+
				"Make sure the referenced table is defined in one of the SDL files.",
			sdlConstraintName(f.Name),
			f.Schema, f.Table,
			f.RefSchema, f.RefTable,
			sdlJoinOr(f.Columns, "<none>"),
		)
	case pganalysis.SDLForeignKeyColumnNotFound:
		advice.Code = code.SDLForeignKeyColumnNotFound.Int32()
		advice.Title = "Foreign key references non-existent column"
		advice.Content = fmt.Sprintf(
			"Foreign key constraint '%s' on table '%s.%s' references column '%s' in table '%s.%s', but this column does not exist.\n\n"+
				"Foreign key definition:\n"+
				"  Source column: %s.%s.%s\n"+
				"  Referenced column: %s.%s.%s (DOES NOT EXIST)\n\n"+
				"Available columns in '%s.%s': %s",
			sdlConstraintName(f.Name),
			f.Schema, f.Table,
			f.RefColumn,
			f.RefSchema, f.RefTable,
			f.Schema, f.Table, f.Column,
			f.RefSchema, f.RefTable, f.RefColumn,
			f.RefSchema, f.RefTable,
			sdlJoinOr(f.Columns, "<no columns>"),
		)
	case pganalysis.SDLViewDependencyNotFound:
		objectType := "table or view"
		advice.Code = code.SDLViewDependencyNotFound.Int32()
		advice.Title = "View references non-existent table or view"
		advice.Content = fmt.Sprintf(
			"View '%s.%s' (line %d) references %s '%s.%s' which does not exist in any SDL file.\n\n"+
				"Views must reference tables or views that are defined in the SDL project.\n\n"+
				"Fix: Define %s '%s.%s' in one of the SDL files, or remove the view if the object is external.",
			f.Schema, f.Name, line,
			objectType, f.RefSchema, f.RefTable,
			objectType, f.RefSchema, f.RefTable,
		)
	default:
	}
	return advice
}

func sdlDuplicateAcrossFiles(titleKind, lowerKind, schema, name, firstPath string, firstLine int, path string, line int) (string, string) {
	title := fmt.Sprintf("Duplicate %s name across files", lowerKind)
	content := fmt.Sprintf(
		"%s '%s.%s' is defined in multiple SDL files.\n\n"+
			"First definition: %s (line %d)\n"+
			"Duplicate definition: %s (line %d)\n\n"+
			"Each %s must be defined in exactly one file in the SDL project.",
		titleKind, schema, name,
		firstPath, firstLine,
		path, line,
		lowerKind,
	)
	return title, content
}

func sdlConstraintName(name string) string {
	if name != "" {
		return name
	}
	return "<unnamed>"
}

func sdlJoinOr(items []string, empty string) string {
	if len(items) == 0 {
		return empty
	}
	return strings.Join(items, ", ")
}
