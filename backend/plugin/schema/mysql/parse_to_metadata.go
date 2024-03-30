package mysql

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/antlr4-go/antlr/v4"
	"github.com/pkg/errors"

	mysql "github.com/bytebase/mysql-parser"

	mysqlparser "github.com/bytebase/bytebase/backend/plugin/parser/mysql"
	"github.com/bytebase/bytebase/backend/plugin/schema"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

func init() {
	schema.RegisterParseToMetadatas(storepb.Engine_MYSQL, ParseToMetadata)
}

// ParseToMetadata converts a schema string to database metadata.
func ParseToMetadata(_, schema string) (*storepb.DatabaseSchemaMetadata, error) {
	list, err := mysqlparser.ParseMySQL(schema)
	if err != nil {
		return nil, err
	}

	listener := &mysqlTransformer{
		state: newDatabaseState(),
	}
	listener.state.schemas[""] = newSchemaState()

	for _, stmt := range list {
		antlr.ParseTreeWalkerDefault.Walk(listener, stmt.Tree)
	}

	return listener.state.convertToDatabaseMetadata(), listener.err
}

type mysqlTransformer struct {
	*mysql.BaseMySQLParserListener

	state        *databaseState
	currentTable string
	err          error
}

// EnterCreateTable is called when production createTable is entered.
func (t *mysqlTransformer) EnterCreateTable(ctx *mysql.CreateTableContext) {
	if t.err != nil {
		return
	}
	databaseName, tableName := mysqlparser.NormalizeMySQLTableName(ctx.TableName())
	if databaseName != "" {
		if t.state.name == "" {
			t.state.name = databaseName
		} else if t.state.name != databaseName {
			t.err = errors.New("multiple database names found: " + t.state.name + ", " + databaseName)
			return
		}
	}

	schema := t.state.schemas[""]
	if _, ok := schema.tables[tableName]; ok {
		t.err = errors.New("multiple table names found: " + tableName)
		return
	}

	schema.tables[tableName] = newTableState(len(schema.tables), tableName)
	t.currentTable = tableName
}

// ExitCreateTable is called when production createTable is exited.
func (t *mysqlTransformer) ExitCreateTable(_ *mysql.CreateTableContext) {
	t.currentTable = ""
}

// EnterCreateTableOption is called when production createTableOption is entered.
func (t *mysqlTransformer) EnterCreateTableOption(ctx *mysql.CreateTableOptionContext) {
	if t.err != nil || t.currentTable == "" {
		return
	}

	if ctx.ENGINE_SYMBOL() != nil {
		engineString := ctx.EngineRef().TextOrIdentifier().GetParser().GetTokenStream().GetTextFromRuleContext(ctx.EngineRef().TextOrIdentifier())
		schema := t.state.schemas[""]
		table, ok := schema.tables[t.currentTable]
		if !ok {
			// This should never happen.
			return
		}
		table.engine = engineString
	}

	if defaultCollation := ctx.DefaultCollation(); defaultCollation != nil {
		collationString := defaultCollation.CollationName().GetParser().GetTokenStream().GetTextFromRuleContext(defaultCollation.CollationName())
		schema := t.state.schemas[""]
		table, ok := schema.tables[t.currentTable]
		if !ok {
			// This should never happen.
			return
		}
		table.collation = collationString
	}

	if ctx.COMMENT_SYMBOL() != nil {
		commentString := ctx.TextStringLiteral().GetText()
		if len(commentString) > 2 {
			quotes := commentString[0]
			escape := fmt.Sprintf("%c%c", quotes, quotes)
			commentString = strings.ReplaceAll(commentString[1:len(commentString)-1], escape, string(quotes))
		}

		schema := t.state.schemas[""]
		table, ok := schema.tables[t.currentTable]
		if !ok {
			// This should never happen.
			return
		}
		table.comment = commentString
	}
}

// EnterColumnDefinition is called when production columnDefinition is entered.
func (t *mysqlTransformer) EnterColumnDefinition(ctx *mysql.ColumnDefinitionContext) {
	if t.err != nil || t.currentTable == "" {
		return
	}

	_, _, columnName := mysqlparser.NormalizeMySQLColumnName(ctx.ColumnName())
	dataType := getDataTypePlainText(ctx.FieldDefinition().DataType())
	table := t.state.schemas[""].tables[t.currentTable]
	if _, ok := table.columns[columnName]; ok {
		t.err = errors.New("multiple column names found: " + columnName + " in table " + t.currentTable)
		return
	}
	columnState := &columnState{
		id:           len(table.columns),
		name:         columnName,
		tp:           dataType,
		defaultValue: nil,
		comment:      "",
		nullable:     true,
	}

	for _, attribute := range ctx.FieldDefinition().AllColumnAttribute() {
		switch {
		case attribute.NullLiteral() != nil && attribute.NOT_SYMBOL() != nil:
			columnState.nullable = false
		case attribute.DEFAULT_SYMBOL() != nil && attribute.SERIAL_SYMBOL() == nil:
			defaultValueStart := nextDefaultChannelTokenIndex(ctx.GetParser().GetTokenStream(), attribute.DEFAULT_SYMBOL().GetSymbol().GetTokenIndex())
			defaultValue := attribute.GetParser().GetTokenStream().GetTextFromInterval(antlr.Interval{
				Start: defaultValueStart,
				Stop:  attribute.GetStop().GetTokenIndex(),
			})
			switch {
			case strings.EqualFold(defaultValue, "NULL"):
				columnState.defaultValue = &defaultValueNull{}
			case strings.HasPrefix(defaultValue, "'") && strings.HasSuffix(defaultValue, "'"):
				columnState.defaultValue = &defaultValueString{value: strings.ReplaceAll(defaultValue[1:len(defaultValue)-1], "''", "'")}
			case strings.HasPrefix(defaultValue, "\"") && strings.HasSuffix(defaultValue, "\""):
				columnState.defaultValue = &defaultValueString{value: strings.ReplaceAll(defaultValue[1:len(defaultValue)-1], "\"\"", "\"")}
			default:
				columnState.defaultValue = &defaultValueExpression{value: defaultValue}
			}
		case attribute.COMMENT_SYMBOL() != nil:
			commentStart := nextDefaultChannelTokenIndex(ctx.GetParser().GetTokenStream(), attribute.COMMENT_SYMBOL().GetSymbol().GetTokenIndex())
			comment := attribute.GetParser().GetTokenStream().GetTextFromInterval(antlr.Interval{
				Start: commentStart,
				Stop:  attribute.GetStop().GetTokenIndex(),
			})
			if comment != `''` && len(comment) > 2 {
				columnState.comment = comment[1 : len(comment)-1]
			}
		// todo(zp): refactor column attribute.
		case attribute.AUTO_INCREMENT_SYMBOL() != nil:
			defaultValue := autoIncrementSymbol
			columnState.defaultValue = &defaultValueExpression{value: defaultValue}
		case attribute.ON_SYMBOL() != nil && attribute.UPDATE_SYMBOL() != nil:
			onUpdateValue := ""
			if attribute.TimeFunctionParameters() != nil && attribute.TimeFunctionParameters().FractionalPrecision() != nil {
				onUpdateValue = "CURRENT_TIMESTAMP(" + attribute.TimeFunctionParameters().FractionalPrecision().GetText() + ")"
			} else {
				onUpdateValue = "CURRENT_TIMESTAMP"
			}
			columnState.onUpdate = onUpdateValue
		}
	}

	if columnState.defaultValue == nil && columnState.nullable {
		columnState.defaultValue = &defaultValueNull{}
	}

	table.columns[columnName] = columnState
}

// EnterTableConstraintDef is called when production tableConstraintDef is entered.
func (t *mysqlTransformer) EnterTableConstraintDef(ctx *mysql.TableConstraintDefContext) {
	if t.err != nil || t.currentTable == "" {
		return
	}

	if ctx.GetType_() != nil {
		symbol := strings.ToUpper(ctx.GetType_().GetText())
		switch symbol {
		case "PRIMARY":
			keys, keyLengths := extractKeyListVariants(ctx.KeyListVariants())
			table := t.state.schemas[""].tables[t.currentTable]
			table.indexes["PRIMARY"] = &indexState{
				id:      len(table.indexes),
				name:    "PRIMARY",
				keys:    keys,
				lengths: keyLengths,
				primary: true,
				unique:  true,
			}
		case "FOREIGN":
			var name string
			if ctx.ConstraintName() != nil && ctx.ConstraintName().Identifier() != nil {
				name = mysqlparser.NormalizeMySQLIdentifier(ctx.ConstraintName().Identifier())
			} else if ctx.IndexName() != nil {
				name = mysqlparser.NormalizeMySQLIdentifier(ctx.IndexName().Identifier())
			}
			keys, _ := extractKeyList(ctx.KeyList())
			table := t.state.schemas[""].tables[t.currentTable]
			if table.foreignKeys[name] != nil {
				t.err = errors.New("multiple foreign keys found: " + name)
				return
			}
			referencedTable, referencedColumns := extractReference(ctx.References())
			fk := &foreignKeyState{
				id:                len(table.foreignKeys),
				name:              name,
				columns:           keys,
				referencedTable:   referencedTable,
				referencedColumns: referencedColumns,
			}
			table.foreignKeys[name] = fk
		case "FULLTEXT":
			var name string
			if ctx.IndexName() != nil {
				name = mysqlparser.NormalizeMySQLIdentifier(ctx.IndexName().Identifier())
			}
			keys, keyLengths := extractKeyListVariants(ctx.KeyListVariants())
			table := t.state.schemas[""].tables[t.currentTable]
			if table.indexes[name] != nil {
				t.err = errors.New("multiple indexes found: " + name)
				return
			}
			idx := &indexState{
				id:      len(table.indexes),
				name:    name,
				keys:    keys,
				lengths: keyLengths,
				primary: false,
				unique:  false,
				tp:      symbol,
			}
			table.indexes[name] = idx
		case "SPATIAL":
			var name string
			if ctx.IndexName() != nil {
				name = mysqlparser.NormalizeMySQLIdentifier(ctx.IndexName().Identifier())
			}
			keys, keyLengths := extractKeyListVariants(ctx.KeyListVariants())
			table := t.state.schemas[""].tables[t.currentTable]
			if table.indexes[name] != nil {
				t.err = errors.New("multiple indexes found: " + name)
				return
			}
			idx := &indexState{
				id:      len(table.indexes),
				name:    name,
				keys:    keys,
				lengths: keyLengths,
				primary: false,
				unique:  false,
				tp:      symbol,
			}
			table.indexes[name] = idx
		case "KEY", "INDEX", "UNIQUE":
			var name string
			if v := ctx.IndexNameAndType(); v != nil {
				name = mysqlparser.NormalizeMySQLIdentifier(v.IndexName().Identifier())
			} else {
				t.err = errors.New("index name not found")
			}
			keys, keyLengths := extractKeyListVariants(ctx.KeyListVariants())
			table := t.state.schemas[""].tables[t.currentTable]
			if table.indexes[name] != nil {
				t.err = errors.New("multiple indexes found: " + name)
				return
			}
			tp := "BTREE"
			if v := ctx.IndexNameAndType(); v != nil && v.IndexType() != nil {
				tp = strings.ToUpper(v.IndexType().GetText())
			}
			idx := &indexState{
				id:      len(table.indexes),
				name:    name,
				keys:    keys,
				lengths: keyLengths,
				primary: false,
				unique:  symbol == "UNIQUE",
				tp:      tp,
			}
			table.indexes[name] = idx
		}
	}
}

func (t *mysqlTransformer) EnterPartitionClause(ctx *mysql.PartitionClauseContext) {
	if t.err != nil {
		return
	}
	if _, parentIsCreateTable := ctx.GetParent().(*mysql.CreateTableContext); !parentIsCreateTable {
		return
	}
	if t.currentTable == "" {
		return
	}
	table := t.state.schemas[""].tables[t.currentTable]
	if table == nil {
		return
	}

	parititonInfo := partitionInfo{}

	iTypeDefCtx := ctx.PartitionTypeDef()
	if iTypeDefCtx != nil {
		switch typeDefCtx := iTypeDefCtx.(type) {
		case *mysql.PartitionDefKeyContext:
			parititonInfo.tp = storepb.TablePartitionMetadata_KEY
			if typeDefCtx.LINEAR_SYMBOL() != nil {
				parititonInfo.tp = storepb.TablePartitionMetadata_LINEAR_KEY
			}
			// TODO(zp): handle the key algorithm
			if typeDefCtx.IdentifierList() != nil {
				identifiers := extractIdentifierList(typeDefCtx.IdentifierList())
				for i, identifier := range identifiers {
					identifier := strings.TrimSpace(identifier)
					if !strings.HasPrefix(identifier, "`") || !strings.HasSuffix(identifier, "`") {
						identifiers[i] = fmt.Sprintf("`%s`", identifier)
					}
				}
				parititonInfo.expr = strings.Join(identifiers, ",")
			}
		case *mysql.PartitionDefHashContext:
			parititonInfo.tp = storepb.TablePartitionMetadata_HASH
			if typeDefCtx.LINEAR_SYMBOL() != nil {
				parititonInfo.tp = storepb.TablePartitionMetadata_LINEAR_HASH
			}
			bitExprText := typeDefCtx.GetParser().GetTokenStream().GetTextFromRuleContext(typeDefCtx.BitExpr())
			bitExprFields := strings.Split(bitExprText, ",")
			for i, bitExprField := range bitExprFields {
				bitExprField := strings.TrimSpace(bitExprField)
				if !strings.HasPrefix(bitExprField, "`") || !strings.HasSuffix(bitExprField, "`") {
					bitExprFields[i] = fmt.Sprintf("`%s`", bitExprField)
				}
			}
			parititonInfo.expr = strings.Join(bitExprFields, ",")
		case *mysql.PartitionDefRangeListContext:
			if typeDefCtx.RANGE_SYMBOL() != nil {
				parititonInfo.tp = storepb.TablePartitionMetadata_RANGE
			} else {
				parititonInfo.tp = storepb.TablePartitionMetadata_LIST
			}
			if typeDefCtx.COLUMNS_SYMBOL() != nil {
				if parititonInfo.tp == storepb.TablePartitionMetadata_RANGE {
					parititonInfo.tp = storepb.TablePartitionMetadata_RANGE_COLUMNS
				} else {
					parititonInfo.tp = storepb.TablePartitionMetadata_LIST_COLUMNS
				}

				identifierList := extractIdentifierList(typeDefCtx.IdentifierList())
				for i, identifier := range identifierList {
					identifier := strings.TrimSpace(identifier)
					if !strings.HasPrefix(identifier, "`") || !strings.HasSuffix(identifier, "`") {
						identifierList[i] = fmt.Sprintf("`%s`", identifier)
					}
				}
				parititonInfo.expr = strings.Join(identifierList, ",")
			} else {
				bitExprText := typeDefCtx.GetParser().GetTokenStream().GetTextFromRuleContext(typeDefCtx.BitExpr())
				bitExprFields := strings.Split(bitExprText, ",")
				for i, bitExprField := range bitExprFields {
					bitExprField := strings.TrimSpace(bitExprField)
					if !strings.HasPrefix(bitExprField, "`") || !strings.HasSuffix(bitExprField, "`") {
						bitExprFields[i] = fmt.Sprintf("`%s`", bitExprField)
					}
				}
				parititonInfo.expr = strings.Join(bitExprFields, ",")
			}
		default:
			t.err = errors.New("unknown partition type")
			return
		}
	}

	if n := ctx.Real_ulong_number(); n != nil {
		number, err := strconv.ParseInt(n.GetText(), 10, 64)
		if err != nil {
			t.err = errors.Wrap(err, "failed to parse partition number")
			return
		}
		parititonInfo.useDefault = int(number)
	}

	var subInfo *partitionInfo
	if subPartitionCtx := ctx.SubPartitions(); subPartitionCtx != nil {
		subInfo = new(partitionInfo)
		if subPartitionCtx.HASH_SYMBOL() != nil {
			subInfo.tp = storepb.TablePartitionMetadata_HASH
			if subPartitionCtx.LINEAR_SYMBOL() != nil {
				subInfo.tp = storepb.TablePartitionMetadata_LINEAR_HASH
			}
			if bitExprCtx := subPartitionCtx.BitExpr(); bitExprCtx != nil {
				bitExprText := bitExprCtx.GetParser().GetTokenStream().GetTextFromRuleContext(bitExprCtx)
				bitExprFields := strings.Split(bitExprText, ",")
				for i, bitExprField := range bitExprFields {
					bitExprField := strings.TrimSpace(bitExprField)
					if !strings.HasPrefix(bitExprField, "`") || !strings.HasSuffix(bitExprField, "`") {
						bitExprFields[i] = fmt.Sprintf("`%s`", bitExprField)
					}
				}
				subInfo.expr = strings.Join(bitExprFields, ",")
			}
		} else if subPartitionCtx.KEY_SYMBOL() != nil {
			subInfo.tp = storepb.TablePartitionMetadata_KEY
			if subPartitionCtx.LINEAR_SYMBOL() != nil {
				subInfo.tp = storepb.TablePartitionMetadata_LINEAR_KEY
			}
			if identifierListParensCtx := subPartitionCtx.IdentifierListWithParentheses(); identifierListParensCtx != nil {
				identifiers := extractIdentifierList(identifierListParensCtx.IdentifierList())
				for i, identifier := range identifiers {
					identifier := strings.TrimSpace(identifier)
					if !strings.HasPrefix(identifier, "`") || !strings.HasSuffix(identifier, "`") {
						identifiers[i] = fmt.Sprintf("`%s`", identifier)
					}
				}
				subInfo.expr = strings.Join(identifiers, ",")
			}
		}

		if n := subPartitionCtx.Real_ulong_number(); n != nil {
			number, err := strconv.ParseInt(n.GetText(), 10, 64)
			if err != nil {
				t.err = errors.Wrap(err, "failed to parse sub partition number")
				return
			}
			subInfo.useDefault = int(number)
		}
	}

	partitionDefinitions := make(map[string]*partitionDefinition)
	allPartDefs := ctx.PartitionDefinitions().AllPartitionDefinition()
	for i, partDef := range allPartDefs {
		pd := &partitionDefinition{
			id:   i + 1,
			name: mysqlparser.NormalizeMySQLIdentifier(partDef.Identifier()),
		}
		switch parititonInfo.tp {
		case storepb.TablePartitionMetadata_RANGE_COLUMNS, storepb.TablePartitionMetadata_RANGE:
			if partDef.LESS_SYMBOL() == nil {
				t.err = errors.New("RANGE partition but no LESS THAN clause")
				return
			}
			if partDef.PartitionValueItemListParen() != nil {
				itemsText := partDef.PartitionValueItemListParen().GetParser().GetTokenStream().GetTextFromInterval(
					antlr.NewInterval(
						partDef.PartitionValueItemListParen().OPEN_PAR_SYMBOL().GetSymbol().GetTokenIndex()+1,
						partDef.PartitionValueItemListParen().CLOSE_PAR_SYMBOL().GetSymbol().GetTokenIndex()-1,
					),
				)
				itemsTextFields := strings.Split(itemsText, ",")
				for i, itemsTextField := range itemsTextFields {
					itemsTextField := strings.TrimSpace(itemsTextField)
					if strings.HasPrefix(itemsTextField, "`") && strings.HasSuffix(itemsTextField, "`") {
						itemsTextField = itemsTextField[1 : len(itemsTextField)-1]
					}
					itemsTextFields[i] = itemsTextField
				}
				pd.value = strings.Join(itemsTextFields, ",")
			} else {
				pd.value = "MAXVALUE"
			}
		case storepb.TablePartitionMetadata_LIST_COLUMNS, storepb.TablePartitionMetadata_LIST:
			if partDef.PartitionValuesIn() == nil {
				t.err = errors.New("COLUMNS partition but no partition value item in IN clause")
				return
			}
			var itemsText string
			if partDef.PartitionValuesIn().OPEN_PAR_SYMBOL() != nil {
				itemsText = partDef.PartitionValuesIn().GetParser().GetTokenStream().GetTextFromInterval(
					antlr.NewInterval(
						partDef.PartitionValuesIn().OPEN_PAR_SYMBOL().GetSymbol().GetTokenIndex()+1,
						partDef.PartitionValuesIn().CLOSE_PAR_SYMBOL().GetSymbol().GetTokenIndex()-1,
					),
				)
			} else {
				itemsText = partDef.PartitionValuesIn().GetParser().GetTokenStream().GetTextFromRuleContext(partDef.PartitionValuesIn().PartitionValueItemListParen(0))
			}

			itemsTextFields := strings.Split(itemsText, ",")
			for i, itemsTextField := range itemsTextFields {
				itemsTextField := strings.TrimSpace(itemsTextField)
				if strings.HasPrefix(itemsTextField, "`") && strings.HasSuffix(itemsTextField, "`") {
					itemsTextField = itemsTextField[1 : len(itemsTextField)-1]
				}
				itemsTextFields[i] = itemsTextField
			}
			pd.value = strings.Join(itemsTextFields, ",")
		case storepb.TablePartitionMetadata_HASH, storepb.TablePartitionMetadata_LINEAR_HASH, storepb.TablePartitionMetadata_KEY, storepb.TablePartitionMetadata_LINEAR_KEY:
		default:
			t.err = errors.New("unknown partition type")
			return
		}

		if subInfo != nil {
			allSubpartitions := partDef.AllSubpartitionDefinition()
			if subInfo.tp == storepb.TablePartitionMetadata_TYPE_UNSPECIFIED && len(allSubpartitions) > 0 {
				t.err = errors.New("specify subpartition definition but no subpartition type specified")
				return
			}
			subPartitionDefinitions := make(map[string]*partitionDefinition)
			for i, sub := range allSubpartitions {
				subpd := &partitionDefinition{
					id:   i + 1,
					name: mysqlparser.NormalizeMySQLTextOrIdentifier(sub.TextOrIdentifier()),
				}
				subPartitionDefinitions[subpd.name] = subpd
			}
			pd.subpartitions = subPartitionDefinitions
		}

		partitionDefinitions[pd.name] = pd
	}

	table.partition = &partitionState{
		info:       parititonInfo,
		subInfo:    subInfo,
		partitions: partitionDefinitions,
	}
}

func (t *mysqlTransformer) EnterCreateView(ctx *mysql.CreateViewContext) {
	if t.err != nil {
		return
	}

	databaseName, viewName := mysqlparser.NormalizeMySQLViewName(ctx.ViewName())
	if databaseName != "" && t.state.name != "" && databaseName != t.state.name {
		t.err = errors.New("multiple database names found: " + t.state.name + ", " + databaseName)
		return
	}

	schema, ok := t.state.schemas[""]
	if !ok || schema == nil {
		t.state.schemas[""] = newSchemaState()
	}

	definition := ctx.GetParser().GetTokenStream().GetTextFromInterval(antlr.Interval{
		Start: ctx.ViewTail().ViewSelect().GetStart().GetTokenIndex(),
		Stop:  ctx.ViewTail().ViewSelect().GetStop().GetTokenIndex(),
	})
	schema.views[viewName] = &viewState{
		id:         len(schema.views),
		name:       viewName,
		definition: definition,
	}
}
