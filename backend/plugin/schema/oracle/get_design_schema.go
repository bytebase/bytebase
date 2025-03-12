package oracle

import (
	"strings"

	"github.com/antlr4-go/antlr/v4"

	"github.com/bytebase/bytebase/backend/plugin/parser/plsql"
	"github.com/bytebase/bytebase/backend/plugin/schema"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

func init() {
	schema.RegisterGetDesignSchema(storepb.Engine_ORACLE, GetDesignSchema)
}

func GetDesignSchema(to *storepb.DatabaseSchemaMetadata) (string, error) {
	baselineSchema := ""
	toState := convertToDatabaseState(to)
	tree, tokens, err := plsql.ParsePLSQL(baselineSchema)
	if err != nil {
		return "", err
	}

	generator := &designSchemaGenerator{
		to: toState,
	}
	antlr.ParseTreeWalkerDefault.Walk(generator, tree)
	if generator.err != nil {
		return "", generator.err
	}

	for _, schema := range to.Schemas {
		schemaState, ok := toState.schemas[schema.Name]
		if !ok {
			continue
		}
		for _, table := range schema.Tables {
			tableState, ok := schemaState.tables[table.Name]
			if !ok {
				continue
			}
			if tableState.deleted {
				// Add indexes.
				for _, index := range table.Indexes {
					if index.Primary || index.Unique {
						continue
					}
					if indexState := tableState.indexes[index.Name]; indexState != nil {
						var buf strings.Builder
						if err := indexState.toOutlineString(schemaState.name, tableState.name, &buf); err != nil {
							return "", err
						}
						generator.actions = append(generator.actions, plsql.NewAddIndexAction(schema.Name, table.Name, buf.String()))
					}
				}
				continue
			}
			buf := &strings.Builder{}
			if err := tableState.toString(schema.Name, buf); err != nil {
				return "", err
			}
			generator.actions = append(generator.actions, plsql.NewAddTableAction(schema.Name, table.Name, buf.String()))
			for _, index := range table.Indexes {
				indexState := tableState.indexes[index.Name]
				if indexState == nil {
					continue
				}
				if index.Primary || index.Unique {
					continue
				}
				buf := &strings.Builder{}
				if err := indexState.toOutlineString(schemaState.name, tableState.name, buf); err != nil {
					return "", err
				}
				generator.actions = append(generator.actions, plsql.NewAddIndexAction(schema.Name, table.Name, buf.String()))
			}
		}
	}
	manipulator := plsql.NewStringsManipulator(tree, tokens)
	return manipulator.Manipulate(generator.actions...)
}
