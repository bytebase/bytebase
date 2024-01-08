package tidb

import (
	"fmt"
	"sort"
	"strings"

	tidbast "github.com/pingcap/tidb/pkg/parser/ast"
	"github.com/pkg/errors"

	tidbparser "github.com/bytebase/bytebase/backend/plugin/parser/tidb"
	"github.com/bytebase/bytebase/backend/plugin/schema"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

func init() {
	schema.RegisterGetDesignSchema(storepb.Engine_TIDB, GetDesignSchema)
}

func GetDesignSchema(baselineSchema string, to *storepb.DatabaseSchemaMetadata) (string, error) {
	toState := convertToDatabaseState(to)
	stmts, err := tidbparser.ParseTiDB(baselineSchema, "", "")
	if err != nil {
		return "", err
	}

	generator := &tidbDesignSchemaGenerator{
		to: toState,
	}

	for _, stmt := range stmts {
		switch stmt.(type) {
		case *tidbast.CreateTableStmt:
			stmt.Accept(generator)
		default:
			if _, err := generator.result.WriteString(stmt.OriginalText() + "\n"); err != nil {
				return "", err
			}
		}
	}
	if generator.err != nil {
		return "", generator.err
	}

	firstTable := true
	for _, schema := range to.Schemas {
		schemaState, ok := toState.schemas[schema.Name]
		if !ok {
			continue
		}
		for _, table := range schema.Tables {
			table, ok := schemaState.tables[table.Name]
			if !ok {
				continue
			}
			if firstTable {
				firstTable = false
				if _, err := generator.result.WriteString("\n"); err != nil {
					return "", err
				}
			}
			if err := table.toString(&generator.result); err != nil {
				return "", err
			}
		}
	}

	return generator.result.String(), nil
}

type tidbDesignSchemaGenerator struct {
	tidbast.Node

	to                  *databaseState
	result              strings.Builder
	currentTable        *tableState
	firstElementInTable bool
	columnDefine        strings.Builder
	tableConstraints    strings.Builder
	err                 error
}

func (g *tidbDesignSchemaGenerator) Enter(in tidbast.Node) (tidbast.Node, bool) {
	if g.err != nil {
		return in, true
	}

	if node, ok := in.(*tidbast.CreateTableStmt); ok {
		dbInfo := node.Table.DBInfo
		databaseName := ""
		if dbInfo != nil {
			databaseName = dbInfo.Name.String()
		}
		if databaseName != "" && g.to.name != "" && databaseName != g.to.name {
			g.err = errors.New("multiple database names found: " + g.to.name + ", " + databaseName)
			return in, true
		}

		schema, ok := g.to.schemas[""]
		if !ok || schema == nil {
			return in, true
		}

		tableName := node.Table.Name.String()
		table, ok := schema.tables[tableName]
		if !ok {
			return in, true
		}
		g.currentTable = table
		g.firstElementInTable = true
		g.columnDefine.Reset()
		g.tableConstraints.Reset()

		delete(schema.tables, tableName)

		// Start constructing sql.
		// Temporary keyword.
		var temporaryKeyword string
		switch node.TemporaryKeyword {
		case tidbast.TemporaryNone:
			temporaryKeyword = "CREATE TABLE "
		case tidbast.TemporaryGlobal:
			temporaryKeyword = "CREATE GLOBAL TEMPORARY TABLE "
		case tidbast.TemporaryLocal:
			temporaryKeyword = "CREATE TEMPORARY TABLE "
		}
		if _, err := g.result.WriteString(temporaryKeyword); err != nil {
			g.err = err
			return in, true
		}

		// if not exists
		if node.IfNotExists {
			if _, err := g.result.WriteString("IF NOT EXISTS "); err != nil {
				g.err = err
				return in, true
			}
		}

		if tableNameStr, err := tidbRestoreNodeDefault(tidbast.Node(node.Table)); err == nil {
			if _, err := g.result.WriteString(tableNameStr + " "); err != nil {
				g.err = err
				return in, true
			}
		}

		if node.ReferTable != nil {
			if _, err := g.result.WriteString(" LIKE "); err != nil {
				g.err = err
				return in, true
			}
			if referTableStr, err := tidbRestoreNodeDefault(tidbast.Node(node.ReferTable)); err == nil {
				if _, err := g.result.WriteString(referTableStr + " "); err != nil {
					g.err = err
					return in, true
				}
			}
		}

		if _, err := g.result.WriteString("(\n  "); err != nil {
			g.err = err
			return in, true
		}

		// Column definition.
		for _, column := range node.Cols {
			columnName := column.Name.Name.String()
			stateColumn, ok := g.currentTable.columns[columnName]
			if !ok {
				continue
			}

			delete(g.currentTable.columns, columnName)

			if g.firstElementInTable {
				g.firstElementInTable = false
			} else {
				if _, err := g.columnDefine.WriteString(",\n  "); err != nil {
					g.err = err
					return in, true
				}
			}

			// Column name.
			if columnNameStr, err := tidbRestoreNodeDefault(tidbast.Node(column.Name)); err == nil {
				if _, err := g.columnDefine.WriteString(columnNameStr + " "); err != nil {
					g.err = err
					return in, true
				}
			}

			// Compare column types.
			dataType := columnTypeStr(column.Tp)
			if !strings.EqualFold(dataType, stateColumn.tp) {
				if _, err := g.columnDefine.WriteString(stateColumn.tp); err != nil {
					g.err = err
					return in, true
				}
			} else {
				if typeStr, err := tidbRestoreFieldType(column.Tp); err == nil {
					if _, err := g.columnDefine.WriteString(typeStr); err != nil {
						g.err = err
						return in, true
					}
				}
			}

			// Column attributes.
			// todo(zp): refactor column auto_increment.
			skipSchemaAutoIncrement := false
			skipAutoRand := false
			// Default value, auto increment and auto random are mutually exclusive.
			for _, option := range column.Options {
				switch option.Tp {
				case tidbast.ColumnOptionDefaultValue:
					skipAutoRand = stateColumn.hasDefault
					skipSchemaAutoIncrement = stateColumn.hasDefault
				case tidbast.ColumnOptionAutoIncrement:
					skipSchemaAutoIncrement = stateColumn.hasDefault
				case tidbast.ColumnOptionAutoRandom:
					skipAutoRand = stateColumn.hasDefault
				}
			}
			newAttr := tidbExtractNewAttrs(stateColumn, column.Options)
			for _, option := range column.Options {
				attrOrder := tidbGetAttrOrder(option)
				for ; len(newAttr) > 0 && newAttr[0].order < attrOrder; newAttr = newAttr[1:] {
					if _, err := g.columnDefine.WriteString(" " + newAttr[0].text); err != nil {
						g.err = err
						return in, true
					}
				}

				switch option.Tp {
				case tidbast.ColumnOptionNull, tidbast.ColumnOptionNotNull:
					sameNullable := option.Tp == tidbast.ColumnOptionNull && stateColumn.nullable
					sameNullable = sameNullable || (option.Tp == tidbast.ColumnOptionNotNull && !stateColumn.nullable)

					if sameNullable {
						if optionStr, err := tidbRestoreNodeDefault(tidbast.Node(option)); err == nil {
							if _, err := g.columnDefine.WriteString(" " + optionStr); err != nil {
								g.err = err
								return in, true
							}
						}
					} else {
						if stateColumn.nullable {
							if _, err := g.columnDefine.WriteString(" NULL"); err != nil {
								g.err = err
								return in, true
							}
						} else {
							if _, err := g.columnDefine.WriteString(" NOT NULL"); err != nil {
								g.err = err
								return in, true
							}
						}
					}
				case tidbast.ColumnOptionDefaultValue:
					defaultValueText, err := columnDefaultValue(column)
					if err != nil {
						g.err = err
						return in, true
					}
					var defaultValue defaultValue
					switch {
					case strings.EqualFold(*defaultValueText, "NULL"):
						defaultValue = &defaultValueNull{}
					case strings.HasPrefix(*defaultValueText, "'") && strings.HasSuffix(*defaultValueText, "'"):
						defaultValue = &defaultValueString{value: strings.ReplaceAll((*defaultValueText)[1:len(*defaultValueText)-1], "''", "'")}
					default:
						defaultValue = &defaultValueExpression{value: *defaultValueText}
					}
					if stateColumn.hasDefault && stateColumn.defaultValue.toString() == defaultValue.toString() {
						if defaultStr, err := tidbRestoreNodeDefault(option); err == nil {
							if _, err := g.columnDefine.WriteString(" " + defaultStr); err != nil {
								g.err = err
								return in, true
							}
						}
					} else if stateColumn.hasDefault {
						if strings.EqualFold(stateColumn.defaultValue.toString(), autoIncrementSymbol) {
							if _, err := g.columnDefine.WriteString(" " + stateColumn.defaultValue.toString()); err != nil {
								g.err = err
								return in, true
							}
						} else if strings.Contains(strings.ToUpper(stateColumn.defaultValue.toString()), autoRandSymbol) {
							if _, err := g.columnDefine.WriteString(fmt.Sprintf(" /*T![auto_rand] %s */" + stateColumn.defaultValue.toString())); err != nil {
								g.err = err
								return in, true
							}
						} else {
							if _, err := g.columnDefine.WriteString(" DEFAULT"); err != nil {
								g.err = err
								return in, true
							}
							if _, err := g.columnDefine.WriteString(" " + stateColumn.defaultValue.toString()); err != nil {
								g.err = err
								return in, true
							}
						}
					}
				case tidbast.ColumnOptionComment:
					commentValue, err := columnComment(column)
					if err != nil {
						g.err = err
						return in, true
					}
					if stateColumn.comment == commentValue {
						if commentStr, err := tidbRestoreNodeDefault(option); err == nil {
							if _, err := g.columnDefine.WriteString(" " + commentStr); err != nil {
								g.err = err
								return in, true
							}
						}
					} else if stateColumn.comment != "" {
						if _, err := g.columnDefine.WriteString(" COMMENT"); err != nil {
							g.err = err
							return in, true
						}
						if _, err := g.columnDefine.WriteString(fmt.Sprintf(" '%s'", stateColumn.comment)); err != nil {
							g.err = err
							return in, true
						}
					}
				default:
					if skipSchemaAutoIncrement && option.Tp == tidbast.ColumnOptionAutoIncrement {
						continue
					}
					if skipAutoRand && option.Tp == tidbast.ColumnOptionAutoRandom {
						continue
					}
					if optionStr, err := tidbRestoreNodeDefault(option); err == nil {
						if _, err := g.columnDefine.WriteString(" " + optionStr); err != nil {
							g.err = err
							return in, true
						}
					}
				}
			}

			for _, attr := range newAttr {
				if _, err := g.columnDefine.WriteString(" " + attr.text); err != nil {
					g.err = err
					return in, true
				}
			}
		}

		// Table Constraint.
		for _, constraint := range node.Constraints {
			switch constraint.Tp {
			case tidbast.ConstraintPrimaryKey:
				if g.currentTable.indexes["PRIMARY"] != nil {
					if g.firstElementInTable {
						g.firstElementInTable = false
					} else {
						if _, err := g.tableConstraints.WriteString(",\n  "); err != nil {
							g.err = err
							return in, true
						}
					}
					var keys []string
					for _, key := range constraint.Keys {
						keys = append(keys, key.Column.Name.String())
					}
					if equalKeys(keys, g.currentTable.indexes["PRIMARY"].keys) {
						if constraintStr, err := tidbRestoreNodeDefault(constraint); err == nil {
							if _, err := g.tableConstraints.WriteString(constraintStr); err != nil {
								g.err = err
								return in, true
							}
						}
					} else {
						if err := g.currentTable.indexes["PRIMARY"].toString(&g.tableConstraints); err != nil {
							g.err = err
							return in, true
						}
					}
					delete(g.currentTable.indexes, "PRIMARY")
				}
			case tidbast.ConstraintForeignKey:
				fkName := constraint.Name
				if fkName == "" {
					g.err = errors.New("empty foreign key name")
					return in, true
				}
				if g.currentTable.foreignKeys[fkName] != nil {
					if g.firstElementInTable {
						g.firstElementInTable = false
					} else {
						if _, err := g.tableConstraints.WriteString(",\n  "); err != nil {
							g.err = err
							return in, true
						}
					}

					fk := g.currentTable.foreignKeys[fkName]

					var columns []string
					for _, key := range constraint.Keys {
						columns = append(columns, key.Column.Name.String())
					}

					var referencedColumnList []string
					for _, spec := range constraint.Refer.IndexPartSpecifications {
						referencedColumnList = append(referencedColumnList, spec.Column.Name.String())
					}
					referencedTable := constraint.Refer.Table.Name.String()
					if equalKeys(columns, fk.columns) && referencedTable == fk.referencedTable && equalKeys(referencedColumnList, fk.referencedColumns) {
						if constraintStr, err := tidbRestoreNodeDefault(constraint); err == nil {
							if _, err := g.tableConstraints.WriteString(constraintStr); err != nil {
								g.err = err
								return in, true
							}
						}
					} else {
						if err := fk.toString(&g.tableConstraints); err != nil {
							g.err = err
							return in, true
						}
					}
					delete(g.currentTable.foreignKeys, fkName)
				}
			default:
				if g.firstElementInTable {
					g.firstElementInTable = false
				} else {
					if _, err := g.tableConstraints.WriteString(",\n  "); err != nil {
						g.err = err
						return in, true
					}
				}
				if constraintStr, err := tidbRestoreNodeDefault(constraint); err == nil {
					if _, err := g.tableConstraints.WriteString(constraintStr); err != nil {
						g.err = err
						return in, true
					}
				}
			}
		}
	}

	return in, false
}

func (g *tidbDesignSchemaGenerator) Leave(in tidbast.Node) (tidbast.Node, bool) {
	if g.err != nil || g.currentTable == nil {
		return in, true
	}
	if node, ok := in.(*tidbast.CreateTableStmt); ok {
		// Column definition.
		var columnList []*columnState
		for _, column := range g.currentTable.columns {
			columnList = append(columnList, column)
		}
		sort.Slice(columnList, func(i, j int) bool {
			return columnList[i].id < columnList[j].id
		})
		for _, column := range columnList {
			if g.firstElementInTable {
				g.firstElementInTable = false
			} else {
				if _, err := g.columnDefine.WriteString(",\n  "); err != nil {
					g.err = err
					return in, true
				}
			}
			if err := column.toString(&g.columnDefine); err != nil {
				g.err = err
				return in, true
			}
		}

		// Primary key definition.
		if g.currentTable.indexes["PRIMARY"] != nil {
			if g.firstElementInTable {
				g.firstElementInTable = false
			} else {
				if _, err := g.columnDefine.WriteString(",\n  "); err != nil {
					g.err = err
					return in, true
				}
			}
			if err := g.currentTable.indexes["PRIMARY"].toString(&g.tableConstraints); err != nil {
				return in, true
			}
		}

		// Foreign key definition.
		var fks []*foreignKeyState
		for _, fk := range g.currentTable.foreignKeys {
			fks = append(fks, fk)
		}
		sort.Slice(fks, func(i, j int) bool {
			return fks[i].id < fks[j].id
		})
		for _, fk := range fks {
			if g.firstElementInTable {
				g.firstElementInTable = false
			} else {
				if _, err := g.columnDefine.WriteString(",\n "); err != nil {
					g.err = err
					return in, true
				}
			}
			if err := fk.toString(&g.tableConstraints); err != nil {
				return in, true
			}
		}

		if _, err := g.result.WriteString(g.columnDefine.String()); err != nil {
			g.err = err
			return in, true
		}
		if _, err := g.result.WriteString(g.tableConstraints.String()); err != nil {
			g.err = err
			return in, true
		}
		if _, err := g.result.WriteString("\n)"); err != nil {
			g.err = err
			return in, true
		}

		// Table option.
		hasTableComment := false
		for _, option := range node.Options {
			if option.Tp == tidbast.TableOptionComment {
				commentValue := tableComment(option)
				if g.currentTable.comment == commentValue && g.currentTable.comment != "" {
					if _, err := g.result.WriteString(" COMMENT"); err != nil {
						g.err = err
						return in, true
					}
					if _, err := g.result.WriteString(fmt.Sprintf(" '%s'", g.currentTable.comment)); err != nil {
						g.err = err
						return in, true
					}
				}
				hasTableComment = true
			} else {
				if optionStr, err := tidbRestoreTableOption(option); err == nil {
					if _, err := g.result.WriteString(" " + optionStr); err != nil {
						g.err = err
						return in, true
					}
				}
			}
		}
		if !hasTableComment && g.currentTable.comment != "" {
			if _, err := g.result.WriteString(" COMMENT"); err != nil {
				g.err = err
				return in, true
			}
			if _, err := g.result.WriteString(fmt.Sprintf(" '%s'", g.currentTable.comment)); err != nil {
				g.err = err
				return in, true
			}
		}

		// Table partition.
		if node.Partition != nil {
			if partitionStr, err := tidbRestoreNodeDefault(node.Partition); err == nil {
				if _, err := g.result.WriteString(" " + partitionStr); err != nil {
					g.err = err
					return in, true
				}
			}
		}

		// Table select.
		if node.Select != nil {
			duplicateStr := ""
			switch node.OnDuplicate {
			case tidbast.OnDuplicateKeyHandlingError:
				duplicateStr = " AS "
			case tidbast.OnDuplicateKeyHandlingIgnore:
				duplicateStr = " IGNORE AS "
			case tidbast.OnDuplicateKeyHandlingReplace:
				duplicateStr = " REPLACE AS "
			}

			if selectStr, err := tidbRestoreNodeDefault(node.Select); err == nil {
				if _, err := g.result.WriteString(duplicateStr + selectStr); err != nil {
					g.err = err
					return in, true
				}
			}
		}

		if node.TemporaryKeyword == tidbast.TemporaryGlobal {
			if node.OnCommitDelete {
				if _, err := g.result.WriteString(" ON COMMIT DELETE ROWS"); err != nil {
					g.err = err
					return in, true
				}
			} else {
				if _, err := g.result.WriteString(" ON COMMIT PRESERVE ROWS"); err != nil {
					g.err = err
					return in, true
				}
			}
		}
		if _, err := g.result.WriteString(";\n"); err != nil {
			g.err = err
			return in, true
		}

		g.currentTable = nil
		g.firstElementInTable = false
	}
	return in, true
}

func tidbExtractNewAttrs(column *columnState, options []*tidbast.ColumnOption) []columnAttr {
	var result []columnAttr
	nullExists := false
	defaultExists := false
	commentExists := false

	for _, option := range options {
		switch option.Tp {
		case tidbast.ColumnOptionNull, tidbast.ColumnOptionNotNull:
			nullExists = true
		case tidbast.ColumnOptionDefaultValue:
			defaultExists = true
		case tidbast.ColumnOptionComment:
			commentExists = true
		}
	}

	if !nullExists && !column.nullable {
		result = append(result, columnAttr{
			text:  "NOT NULL",
			order: columnAttrOrder["NULL"],
		})
	}
	if !defaultExists && column.hasDefault {
		// todo(zp): refactor column attribute.
		if strings.EqualFold(column.defaultValue.toString(), autoIncrementSymbol) {
			result = append(result, columnAttr{
				text:  column.defaultValue.toString(),
				order: columnAttrOrder["DEFAULT"],
			})
		} else if strings.Contains(strings.ToUpper(column.defaultValue.toString()), autoRandSymbol) {
			result = append(result, columnAttr{
				text:  fmt.Sprintf("/*T![auto_rand] %s */", column.defaultValue.toString()),
				order: columnAttrOrder["DEFAULT"],
			})
		} else {
			result = append(result, columnAttr{
				text:  "DEFAULT " + column.defaultValue.toString(),
				order: columnAttrOrder["DEFAULT"],
			})
		}
	}
	if !commentExists && column.comment != "" {
		result = append(result, columnAttr{
			text:  "COMMENT '" + column.comment + "'",
			order: columnAttrOrder["COMMENT"],
		})
	}
	return result
}

func tidbGetAttrOrder(option *tidbast.ColumnOption) int {
	switch option.Tp {
	case tidbast.ColumnOptionDefaultValue:
		return columnAttrOrder["DEFAULT"]
	case tidbast.ColumnOptionNull, tidbast.ColumnOptionNotNull:
		return columnAttrOrder["NULL"]
	case tidbast.ColumnOptionUniqKey:
		return columnAttrOrder["UNIQUE"]
	case tidbast.ColumnOptionColumnFormat:
		return columnAttrOrder["COLUMN_FORMAT"]
	case tidbast.ColumnOptionAutoIncrement:
		return columnAttrOrder["AUTO_INCREMENT"]
	case tidbast.ColumnOptionAutoRandom:
		return columnAttrOrder["AUTO_RANDOM"]
	case tidbast.ColumnOptionComment:
		return columnAttrOrder["COMMENT"]
	case tidbast.ColumnOptionCollate:
		return columnAttrOrder["COLLATE"]
	case tidbast.ColumnOptionStorage:
		return columnAttrOrder["STORAGE"]
	case tidbast.ColumnOptionCheck:
		return columnAttrOrder["CHECK"]
	}
	if option.Enforced {
		return columnAttrOrder["ENFORCED"]
	}
	return len(columnAttrOrder) + 1
}
