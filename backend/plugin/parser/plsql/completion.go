package plsql

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"unicode"

	"github.com/antlr4-go/antlr/v4"
	"github.com/bytebase/parser/plsql"

	"github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/store/model"
)

var (
	// globalFollowSetsByState is a map from state to follow sets.
	// It is shared by all PlSQL completers.
	// The FollowSetsByState is the thread-safe struct.
	globalFollowSetsByState = base.NewFollowSetsByState()
)

func init() {
	base.RegisterCompleteFunc(store.Engine_ORACLE, Completion)
}

func Completion(ctx context.Context, cCtx base.CompletionContext, statement string, caretLine int, caretOffset int) ([]base.Candidate, error) {
	completer := NewStandardCompleter(ctx, cCtx, statement, caretLine, caretOffset)
	result, err := completer.completion()
	if err != nil {
		return nil, err
	}
	if len(result) > 0 {
		return result, nil
	}

	trickyCompleter := NewTrickyCompleter(ctx, cCtx, statement, caretLine, caretOffset)
	return trickyCompleter.completion()
}

func newIgnoredTokens() map[int]bool {
	return map[int]bool{
		plsql.PlSqlParserEOF:                      true,
		plsql.PlSqlLexerEQUALS_OP:                 true,
		plsql.PlSqlLexerPERCENT:                   true,
		plsql.PlSqlLexerAMPERSAND:                 true,
		plsql.PlSqlLexerLEFT_PAREN:                true,
		plsql.PlSqlLexerRIGHT_PAREN:               true,
		plsql.PlSqlLexerDOUBLE_ASTERISK:           true,
		plsql.PlSqlLexerASTERISK:                  true,
		plsql.PlSqlLexerPLUS_SIGN:                 true,
		plsql.PlSqlLexerMINUS_SIGN:                true,
		plsql.PlSqlLexerCOMMA:                     true,
		plsql.PlSqlLexerSOLIDUS:                   true,
		plsql.PlSqlLexerAT_SIGN:                   true,
		plsql.PlSqlLexerASSIGN_OP:                 true,
		plsql.PlSqlLexerHASH_OP:                   true,
		plsql.PlSqlLexerSQ:                        true,
		plsql.PlSqlLexerNOT_EQUAL_OP:              true,
		plsql.PlSqlLexerCARRET_OPERATOR_PART:      true,
		plsql.PlSqlLexerTILDE_OPERATOR_PART:       true,
		plsql.PlSqlLexerEXCLAMATION_OPERATOR_PART: true,
		plsql.PlSqlLexerGREATER_THAN_OP:           true,
		plsql.PlSqlLexerLESS_THAN_OP:              true,
		plsql.PlSqlLexerCOLON:                     true,
		plsql.PlSqlLexerSEMICOLON:                 true,
		plsql.PlSqlLexerBAR:                       true,
		plsql.PlSqlLexerLEFT_BRACKET:              true,
		plsql.PlSqlLexerRIGHT_BRACKET:             true,
		plsql.PlSqlLexerINTRODUCER:                true,
		plsql.PlSqlLexerBINDVAR:                   true,
		plsql.PlSqlLexerNULL_:                     true,
		plsql.PlSqlLexerNATIONAL_CHAR_STRING_LIT:  true,
		plsql.PlSqlLexerBIT_STRING_LIT:            true,
		plsql.PlSqlLexerHEX_STRING_LIT:            true,
		plsql.PlSqlLexerDOUBLE_PERIOD:             true,
		plsql.PlSqlLexerPERIOD:                    true,
		plsql.PlSqlLexerUNSIGNED_INTEGER:          true,
		plsql.PlSqlLexerAPPROXIMATE_NUM_LIT:       true,
		plsql.PlSqlLexerCHAR_STRING:               true,
		plsql.PlSqlLexerDELIMITED_ID:              true,
		plsql.PlSqlLexerREGULAR_ID:                true,
	}
}

func newPreferredRules() map[int]bool {
	return map[int]bool{
		plsql.PlSqlParserRULE_general_element:          true,
		plsql.PlSqlParserRULE_tableview_name:           true,
		plsql.PlSqlParserRULE_column_name:              true,
		plsql.PlSqlParserRULE_identifier:               true,
		plsql.PlSqlParserRULE_id_expression:            true,
		plsql.PlSqlParserRULE_regular_id:               true,
		plsql.PlSqlParserRULE_xml_column_name:          true,
		plsql.PlSqlParserRULE_cost_class_name:          true,
		plsql.PlSqlParserRULE_attribute_name:           true,
		plsql.PlSqlParserRULE_savepoint_name:           true,
		plsql.PlSqlParserRULE_rollback_segment_name:    true,
		plsql.PlSqlParserRULE_schema_name:              true,
		plsql.PlSqlParserRULE_package_name:             true,
		plsql.PlSqlParserRULE_implementation_type_name: true,
		plsql.PlSqlParserRULE_parameter_name:           true,
		plsql.PlSqlParserRULE_reference_model_name:     true,
		plsql.PlSqlParserRULE_main_model_name:          true,
		plsql.PlSqlParserRULE_container_tableview_name: true,
		plsql.PlSqlParserRULE_aggregate_function_name:  true,
		plsql.PlSqlParserRULE_grantee_name:             true,
		plsql.PlSqlParserRULE_role_name:                true,
		plsql.PlSqlParserRULE_constraint_name:          true,
		plsql.PlSqlParserRULE_label_name:               true,
		plsql.PlSqlParserRULE_type_name:                true,
		plsql.PlSqlParserRULE_sequence_name:            true,
		plsql.PlSqlParserRULE_exception_name:           true,
		plsql.PlSqlParserRULE_function_name:            true,
		plsql.PlSqlParserRULE_procedure_name:           true,
		plsql.PlSqlParserRULE_trigger_name:             true,
		plsql.PlSqlParserRULE_variable_name:            true,
		plsql.PlSqlParserRULE_index_name:               true,
		plsql.PlSqlParserRULE_record_name:              true,
		plsql.PlSqlParserRULE_collection_name:          true,
		plsql.PlSqlParserRULE_link_name:                true,
		plsql.PlSqlParserRULE_char_set_name:            true,
		plsql.PlSqlParserRULE_synonym_name:             true,
		plsql.PlSqlParserRULE_dir_object_name:          true,
	}
}

func newNoSeparatorRequired() map[int]bool {
	return map[int]bool{
		plsql.PlSqlLexerEQUALS_OP:                 true,
		plsql.PlSqlLexerPERCENT:                   true,
		plsql.PlSqlLexerAMPERSAND:                 true,
		plsql.PlSqlLexerLEFT_PAREN:                true,
		plsql.PlSqlLexerRIGHT_PAREN:               true,
		plsql.PlSqlLexerDOUBLE_ASTERISK:           true,
		plsql.PlSqlLexerASTERISK:                  true,
		plsql.PlSqlLexerPLUS_SIGN:                 true,
		plsql.PlSqlLexerMINUS_SIGN:                true,
		plsql.PlSqlLexerCOMMA:                     true,
		plsql.PlSqlLexerSOLIDUS:                   true,
		plsql.PlSqlLexerAT_SIGN:                   true,
		plsql.PlSqlLexerASSIGN_OP:                 true,
		plsql.PlSqlLexerHASH_OP:                   true,
		plsql.PlSqlLexerSQ:                        true,
		plsql.PlSqlLexerNOT_EQUAL_OP:              true,
		plsql.PlSqlLexerCARRET_OPERATOR_PART:      true,
		plsql.PlSqlLexerTILDE_OPERATOR_PART:       true,
		plsql.PlSqlLexerEXCLAMATION_OPERATOR_PART: true,
		plsql.PlSqlLexerGREATER_THAN_OP:           true,
		plsql.PlSqlLexerLESS_THAN_OP:              true,
		plsql.PlSqlLexerCOLON:                     true,
		plsql.PlSqlLexerSEMICOLON:                 true,
		plsql.PlSqlLexerDOUBLE_PERIOD:             true,
		plsql.PlSqlLexerPERIOD:                    true,
	}
}

type Completer struct {
	ctx                 context.Context
	core                *base.CodeCompletionCore
	scene               base.SceneType
	parser              *plsql.PlSqlParser
	lexer               *plsql.PlSqlLexer
	scanner             *base.Scanner
	instanceID          string
	getMetadata         base.GetDatabaseMetadataFunc
	listDatabaseNames   base.ListDatabaseNamesFunc
	defaultDatabase     string
	metadataCache       map[string]*model.DatabaseMetadata
	noSeparatorRequired map[int]bool
	// referencesStack is a hierarchical stack of table references.
	// We'll update the stack when we encounter a new FROM clauses.
	referencesStack [][]base.TableReference
	// references is the flattened table references.
	// It's helpful to look up the table reference.
	references         []base.TableReference
	cteCache           map[int][]*base.VirtualTableReference
	cteTables          []*base.VirtualTableReference
	caretTokenIsQuoted bool
}

func NewTrickyCompleter(ctx context.Context, cCtx base.CompletionContext, statement string, caretLine int, caretOffset int) *Completer {
	parser, lexer, scanner := prepareTrickyParserAndScanner(statement, caretLine, caretOffset)
	core := base.NewCodeCompletionCore(
		ctx,
		parser,
		newIgnoredTokens(),
		newPreferredRules(),
		&globalFollowSetsByState,
		plsql.PlSqlParserRULE_select_statement,
		plsql.PlSqlParserRULE_query_block,
		plsql.PlSqlParserRULE_column_alias,
		plsql.PlSqlParserRULE_subquery_factoring_clause,
	)
	return &Completer{
		ctx:                 ctx,
		core:                core,
		scene:               cCtx.Scene,
		parser:              parser,
		lexer:               lexer,
		scanner:             scanner,
		instanceID:          cCtx.InstanceID,
		getMetadata:         cCtx.Metadata,
		listDatabaseNames:   cCtx.ListDatabaseNames,
		defaultDatabase:     cCtx.DefaultDatabase,
		metadataCache:       make(map[string]*model.DatabaseMetadata),
		noSeparatorRequired: newNoSeparatorRequired(),
		cteCache:            make(map[int][]*base.VirtualTableReference),
	}
}

func NewStandardCompleter(ctx context.Context, cCtx base.CompletionContext, statement string, caretLine int, caretOffset int) *Completer {
	parser, lexer, scanner := prepareParserAndScanner(statement, caretLine, caretOffset)
	core := base.NewCodeCompletionCore(
		ctx,
		parser,
		newIgnoredTokens(),
		newPreferredRules(),
		&globalFollowSetsByState,
		plsql.PlSqlParserRULE_select_statement,
		plsql.PlSqlParserRULE_query_block,
		plsql.PlSqlParserRULE_column_alias,
		plsql.PlSqlParserRULE_subquery_factoring_clause,
	)
	return &Completer{
		ctx:                 ctx,
		core:                core,
		scene:               cCtx.Scene,
		parser:              parser,
		lexer:               lexer,
		scanner:             scanner,
		instanceID:          cCtx.InstanceID,
		getMetadata:         cCtx.Metadata,
		listDatabaseNames:   cCtx.ListDatabaseNames,
		defaultDatabase:     cCtx.DefaultDatabase,
		metadataCache:       make(map[string]*model.DatabaseMetadata),
		noSeparatorRequired: newNoSeparatorRequired(),
		cteCache:            make(map[int][]*base.VirtualTableReference),
	}
}

func (c *Completer) completion() ([]base.Candidate, error) {
	// Check the caret token is quoted or not.
	// This check should be done before checking the caret token is a separator or not.
	if c.scanner.IsTokenType(plsql.PlSqlLexerDELIMITED_ID) {
		c.caretTokenIsQuoted = true
	}

	caretIndex := c.scanner.GetIndex()
	if caretIndex > 0 && !c.noSeparatorRequired[c.scanner.GetPreviousTokenType(false /* skipHidden */)] {
		caretIndex--
	}
	c.referencesStack = append([][]base.TableReference{{}}, c.referencesStack...)
	c.parser.Reset()
	var context antlr.ParserRuleContext
	if c.scene == base.SceneTypeQuery {
		context = c.parser.Select_statement()
	} else {
		context = c.parser.Sql_script()
	}

	candidates := c.core.CollectCandidates(caretIndex, context)

	for ruleName := range candidates.Rules {
		if ruleName == plsql.PlSqlParserRULE_general_element {
			c.collectLeadingTableReferences(caretIndex)
			c.takeReferencesSnapshot()
			c.collectRemainingTableReferences()
			c.takeReferencesSnapshot()
		}
	}

	return c.convertCandidates(candidates)
}

type CompletionMap map[string]base.Candidate

func (m CompletionMap) toSlice() []base.Candidate {
	var result []base.Candidate
	for _, candidate := range m {
		result = append(result, candidate)
	}
	slices.SortFunc(result, func(i, j base.Candidate) int {
		if i.Type != j.Type {
			if i.Type < j.Type {
				return -1
			}
			return 1
		}
		if i.Text < j.Text {
			return -1
		}
		if i.Text > j.Text {
			return 1
		}
		return 0
	})
	return result
}

func (m CompletionMap) Insert(entry base.Candidate) {
	m[entry.String()] = entry
}

func (m CompletionMap) insertDatabases(c *Completer) {
	for _, name := range c.listAllDatabases() {
		m.Insert(base.Candidate{
			Type: base.CandidateTypeSchema, // For oracle, schema is the same as database modified by bytebase.
			Text: name,
		})
	}
}

func (m CompletionMap) insertViews(c *Completer, schemas map[string]bool) {
	for schema := range schemas {
		for _, view := range c.listViews(schema) {
			m.Insert(base.Candidate{
				Type: base.CandidateTypeView,
				Text: c.quotedIdentifierIfNeeded(view),
			})
		}
	}
}

func (m CompletionMap) insertSequences(c *Completer, schemas map[string]bool) {
	for schema := range schemas {
		for _, seq := range c.listSequences(schema) {
			m.Insert(base.Candidate{
				Type: base.CandidateTypeSequence,
				Text: c.quotedIdentifierIfNeeded(seq),
			})
		}
	}
}

func (m CompletionMap) insertTables(c *Completer, schemas map[string]bool) {
	for schema := range schemas {
		if len(schema) == 0 {
			// User didn't specify the schema, so we need to append cte tables.
			for _, table := range c.cteTables {
				m.Insert(base.Candidate{
					Type: base.CandidateTypeTable,
					Text: c.quotedIdentifierIfNeeded(table.Table),
				})
			}
			continue
		}
		for _, table := range c.listTables(schema) {
			m.Insert(base.Candidate{
				Type: base.CandidateTypeTable,
				Text: c.quotedIdentifierIfNeeded(table),
			})
		}
	}
}

func (m CompletionMap) insertAllColumns(c *Completer) {
	for _, schema := range c.listAllDatabases() {
		if _, exists := c.metadataCache[schema]; !exists {
			_, metadata, err := c.getMetadata(c.ctx, c.instanceID, schema)
			if err != nil || metadata == nil {
				continue
			}
			c.metadataCache[schema] = metadata
		}
		schemaMeta := c.metadataCache[schema].GetSchemaMetadata("")
		for _, table := range schemaMeta.ListTableNames() {
			tableMeta := schemaMeta.GetTable(table)
			if tableMeta == nil {
				continue
			}
			for _, column := range tableMeta.GetProto().GetColumns() {
				definition := fmt.Sprintf("%s.%s | %s", schema, table, column.Type)
				if !column.Nullable {
					definition += ", NOT NULL"
				}
				m.Insert(base.Candidate{
					Type:       base.CandidateTypeColumn,
					Text:       c.quotedIdentifierIfNeeded(column.Name),
					Definition: definition,
					Comment:    column.Comment,
				})
			}
		}
	}
}

func (m CompletionMap) insertColumns(c *Completer, schemas, tables map[string]bool) {
	for schema := range schemas {
		if len(schema) == 0 {
			// User didn't specify the schema, so we need to append cte tables.
			for _, table := range c.cteTables {
				if tables[table.Table] {
					for _, column := range table.Columns {
						m.Insert(base.Candidate{
							Type: base.CandidateTypeColumn,
							Text: c.quotedIdentifierIfNeeded(column),
						})
					}
				}
			}
			continue
		}
		if _, exists := c.metadataCache[schema]; !exists {
			_, metadata, err := c.getMetadata(c.ctx, c.instanceID, schema)
			if err != nil || metadata == nil {
				continue
			}
			c.metadataCache[schema] = metadata
		}

		for table := range tables {
			tableMeta := c.metadataCache[schema].GetSchemaMetadata("").GetTable(table)
			if tableMeta == nil {
				continue
			}
			for _, column := range tableMeta.GetProto().GetColumns() {
				definition := fmt.Sprintf("%s.%s | %s", schema, table, column.Type)
				if !column.Nullable {
					definition += ", NOT NULL"
				}
				m.Insert(base.Candidate{
					Type:       base.CandidateTypeColumn,
					Text:       c.quotedIdentifierIfNeeded(column.Name),
					Definition: definition,
					Comment:    column.Comment,
				})
			}
		}
	}
}

func (c *Completer) listAllDatabases() []string {
	var result []string
	if c.defaultDatabase != "" {
		result = append(result, c.defaultDatabase)
	}
	list, err := c.listDatabaseNames(c.ctx, c.instanceID)
	if err != nil {
		return result
	}
	for _, name := range list {
		if name != c.defaultDatabase {
			result = append(result, name)
		}
	}
	return result
}

func (c *Completer) listTables(schema string) []string {
	if _, exists := c.metadataCache[schema]; !exists {
		_, metadata, err := c.getMetadata(c.ctx, c.instanceID, schema)
		if err != nil || metadata == nil {
			return nil
		}
		c.metadataCache[schema] = metadata
	}

	return c.metadataCache[schema].GetSchemaMetadata("").ListTableNames()
}

func (c *Completer) listViews(schema string) []string {
	if _, exists := c.metadataCache[schema]; !exists {
		_, metadata, err := c.getMetadata(c.ctx, c.instanceID, schema)
		if err != nil || metadata == nil {
			return nil
		}
		c.metadataCache[schema] = metadata
	}

	return c.metadataCache[schema].GetSchemaMetadata("").ListViewNames()
}

func (c *Completer) listSequences(schema string) []string {
	if _, exists := c.metadataCache[schema]; !exists {
		_, metadata, err := c.getMetadata(c.ctx, c.instanceID, schema)
		if err != nil || metadata == nil {
			return nil
		}
		c.metadataCache[schema] = metadata
	}

	return c.metadataCache[schema].GetSchemaMetadata("").ListSequenceNames()
}

func (c *Completer) convertCandidates(candidates *base.CandidatesCollection) ([]base.Candidate, error) {
	keywordEntries := make(CompletionMap)
	runtimeFunctionEntries := make(CompletionMap)
	schemaEntries := make(CompletionMap)
	tableEntries := make(CompletionMap)
	columnEntries := make(CompletionMap)
	viewEntries := make(CompletionMap)
	sequenceEntries := make(CompletionMap)

	for token, value := range candidates.Tokens {
		entry := c.parser.SymbolicNames[token]
		entry = unquote(entry)

		list := 0
		if len(value) > 0 {
			// For function call:
			if value[0] == plsql.PlSqlLexerLEFT_PAREN {
				list = 1
			} else {
				for _, item := range value {
					subEntry := c.parser.SymbolicNames[item]
					subEntry = unquote(subEntry)
					entry += " " + subEntry
				}
			}
		}

		switch list {
		case 1:
			runtimeFunctionEntries.Insert(base.Candidate{
				Type: base.CandidateTypeFunction,
				Text: strings.ToUpper(entry) + "()",
			})
		default:
			keywordEntries.Insert(base.Candidate{
				Type: base.CandidateTypeKeyword,
				Text: strings.ToUpper(entry),
			})
		}
	}

	for candidate := range candidates.Rules {
		c.scanner.PopAndRestore()
		c.scanner.Push()

		c.fetchCommonTableExpression(candidates.Rules[candidate])

		switch candidate {
		case plsql.PlSqlParserRULE_tableview_name:
			schema, flags := c.determineTableViewName()

			if flags&ObjectFlagsShowFirst != 0 {
				schemaEntries.insertDatabases(c)
			}

			if flags&ObjectFlagsShowSecond != 0 {
				schemas := make(map[string]bool)
				if len(schema) == 0 {
					schemas[c.defaultDatabase] = true
					schemas[""] = true
				} else {
					schemas[schema] = true
				}

				tableEntries.insertTables(c, schemas)
				viewEntries.insertViews(c, schemas)
				sequenceEntries.insertSequences(c, schemas)
			}
		case plsql.PlSqlParserRULE_general_element:
			schema, table, flags := c.determineGeneralElementPartCandidates()
			if flags&ObjectFlagsShowSchemas != 0 {
				schemaEntries.insertDatabases(c)
			}

			schemas := make(map[string]bool)
			if len(schema) > 0 {
				schemas[schema] = true
			} else if len(c.references) > 0 {
				for _, reference := range c.references {
					if physicalTable, ok := reference.(*base.PhysicalTableReference); ok {
						if len(physicalTable.Schema) > 0 {
							schemas[physicalTable.Schema] = true
						}
					}
				}
			}

			if len(schemas) == 0 {
				schemas[c.defaultDatabase] = true
				// User didn't specify the schema, so we need to append cte tables.
				schemas[""] = true
			}

			if flags&ObjectFlagsShowTables != 0 {
				tableEntries.insertTables(c, schemas)
				viewEntries.insertViews(c, schemas)
				sequenceEntries.insertSequences(c, schemas)

				for _, reference := range c.references {
					switch reference := reference.(type) {
					case *base.PhysicalTableReference:
						if (len(schema) == 0 && len(reference.Schema) == 0) || schemas[reference.Schema] {
							if len(reference.Alias) == 0 {
								tableEntries.Insert(base.Candidate{
									Type: base.CandidateTypeTable,
									Text: c.quotedIdentifierIfNeeded(reference.Table),
								})
							} else {
								tableEntries.Insert(base.Candidate{
									Type: base.CandidateTypeTable,
									Text: c.quotedIdentifierIfNeeded(reference.Alias),
								})
							}
						}
					case *base.VirtualTableReference:
						if len(schema) > 0 {
							continue
						}
						tableEntries.Insert(base.Candidate{
							Type: base.CandidateTypeTable,
							Text: c.quotedIdentifierIfNeeded(reference.Table),
						})
					}
				}
			}

			if flags&ObjectFlagsShowColumns != 0 {
				if schema == table { // Schema and table are equal if it's not clear if we see a schema or table qualifier.
					schemas[c.defaultDatabase] = true
					// User didn't specify the schema, so we need to append cte tables.
					schemas[""] = true
				}

				tables := make(map[string]bool)
				if len(table) != 0 {
					tables[table] = true

					for _, reference := range c.references {
						switch reference := reference.(type) {
						case *base.PhysicalTableReference:
							// Could be an alias.
							if strings.EqualFold(reference.Alias, table) {
								tables[reference.Table] = true
								schemas[reference.Schema] = true
							}
						case *base.VirtualTableReference:
							// Could be a virtual table.
							if strings.EqualFold(reference.Table, table) {
								for _, column := range reference.Columns {
									columnEntries.Insert(base.Candidate{
										Type: base.CandidateTypeColumn,
										Text: c.quotedIdentifierIfNeeded(column),
									})
								}
							}
						}
					}
				} else if len(c.references) > 0 {
					list := c.fetchSelectItemAliases(candidates.Rules[candidate])
					for _, alias := range list {
						columnEntries.Insert(base.Candidate{
							Type: base.CandidateTypeColumn,
							Text: c.quotedIdentifierIfNeeded(alias),
						})
					}
					for _, reference := range c.references {
						switch reference := reference.(type) {
						case *base.PhysicalTableReference:
							schemas[reference.Schema] = true
							tables[reference.Table] = true
						case *base.VirtualTableReference:
							for _, column := range reference.Columns {
								columnEntries.Insert(base.Candidate{
									Type: base.CandidateTypeColumn,
									Text: c.quotedIdentifierIfNeeded(column),
								})
							}
						}
					}
				} else {
					// User didn't specify the table, so we return all columns from all tables.
					columnEntries.insertAllColumns(c)
				}

				if len(tables) > 0 {
					columnEntries.insertColumns(c, schemas, tables)
				}
			}
		default:
			// Handle other candidates
		}
	}

	c.scanner.PopAndRestore()
	var result []base.Candidate
	result = append(result, keywordEntries.toSlice()...)
	result = append(result, runtimeFunctionEntries.toSlice()...)
	result = append(result, schemaEntries.toSlice()...)
	result = append(result, tableEntries.toSlice()...)
	result = append(result, columnEntries.toSlice()...)
	result = append(result, viewEntries.toSlice()...)
	result = append(result, sequenceEntries.toSlice()...)
	return result, nil
}

func (c *Completer) fetchCommonTableExpression(ruleStack []*base.RuleContext) {
	c.cteTables = nil
	for _, rule := range ruleStack {
		if rule.ID == plsql.PlSqlParserRULE_query_block {
			for _, pos := range rule.CTEList {
				c.cteTables = append(c.cteTables, c.extractCTETables(pos)...)
			}
		}
	}
}

func (c *Completer) extractCTETables(pos int) []*base.VirtualTableReference {
	if metadata, exists := c.cteCache[pos]; exists {
		return metadata
	}
	followingText := c.scanner.GetFollowingTextAfter(pos)
	if len(followingText) == 0 {
		return nil
	}

	input := antlr.NewInputStream(followingText)
	lexer := plsql.NewPlSqlLexer(input)
	tokens := antlr.NewCommonTokenStream(lexer, antlr.TokenDefaultChannel)
	parser := plsql.NewPlSqlParser(tokens)

	parser.BuildParseTrees = true
	parser.RemoveErrorListeners()
	tree := parser.Subquery_factoring_clause()

	listener := &CTETableListener{context: c}
	antlr.ParseTreeWalkerDefault.Walk(listener, tree)

	c.cteCache[pos] = listener.result
	return listener.result
}

type CTETableListener struct {
	*plsql.BasePlSqlParserListener

	context *Completer
	result  []*base.VirtualTableReference
}

func (l *CTETableListener) EnterFactoring_element(ctx *plsql.Factoring_elementContext) {
	table := &base.VirtualTableReference{
		Table: NormalizeIdentifierContext(ctx.Query_name().Identifier()),
	}
	if ctx.Paren_column_list() != nil {
		for _, column := range ctx.Paren_column_list().Column_list().AllColumn_name() {
			_, _, columnName := NormalizeColumnName(column)
			table.Columns = append(table.Columns, columnName)
		}
	} else {
		// User didn't specify the column list, so we need to fetch the column list from the query.
		if span, err := GetQuerySpan(
			l.context.ctx,
			base.GetQuerySpanContext{
				InstanceID:              l.context.instanceID,
				GetDatabaseMetadataFunc: l.context.getMetadata,
				ListDatabaseNamesFunc:   l.context.listDatabaseNames,
			},
			fmt.Sprintf("SELECT * FROM (%s)", ctx.GetParser().GetTokenStream().GetTextFromRuleContext(ctx.Subquery())),
			l.context.defaultDatabase,
			"",
			false,
		); err == nil && span.NotFoundError == nil {
			for _, column := range span.Results {
				table.Columns = append(table.Columns, column.Name)
			}
		}
	}

	l.result = append(l.result, table)
}

func (c *Completer) fetchSelectItemAliases(ruleStack []*base.RuleContext) []string {
	canUseAliases := false
	for i := len(ruleStack) - 1; i >= 0; i-- {
		switch ruleStack[i].ID {
		case plsql.PlSqlParserRULE_group_by_clause, plsql.PlSqlParserRULE_order_by_clause:
			canUseAliases = true
		case plsql.PlSqlParserRULE_query_block, plsql.PlSqlParserRULE_select_statement:
			if !canUseAliases {
				return nil
			}
			aliasMap := make(map[string]bool)
			for pos := range ruleStack[i].SelectItemAliases {
				if aliasText := c.extractAliasText(pos); len(aliasText) > 0 {
					aliasMap[aliasText] = true
				}
			}

			var result []string
			for alias := range aliasMap {
				result = append(result, alias)
			}
			slices.Sort(result)
			return result
		default:
			// Other cases
		}
	}
	return nil
}

func (c *Completer) extractAliasText(pos int) string {
	followingText := c.scanner.GetFollowingTextAfter(pos)
	if len(followingText) == 0 {
		return ""
	}

	input := antlr.NewInputStream(followingText)
	lexer := plsql.NewPlSqlLexer(input)
	tokens := antlr.NewCommonTokenStream(lexer, antlr.TokenDefaultChannel)
	parser := plsql.NewPlSqlParser(tokens)

	parser.BuildParseTrees = true
	parser.RemoveErrorListeners()
	lexer.RemoveErrorListeners()
	tree := parser.Column_alias()

	listener := &AliasListener{}
	antlr.ParseTreeWalkerDefault.Walk(listener, tree)
	return listener.result
}

type AliasListener struct {
	*plsql.BasePlSqlParserListener

	result string
}

func (l *AliasListener) EnterColumn_alias(ctx *plsql.Column_aliasContext) {
	l.result = normalizeColumnAlias(ctx)
}

type ObjectFlags int

const (
	ObjectFlagsShowSchemas ObjectFlags = 1 << iota
	ObjectFlagsShowTables
	ObjectFlagsShowColumns
	ObjectFlagsShowFirst
	ObjectFlagsShowSecond
)

func (c *Completer) determineTableViewName() (string, ObjectFlags) {
	position := c.scanner.GetIndex()
	if c.scanner.GetTokenChannel() != 0 {
		c.scanner.Forward(true /* skipHidden */) // Skip whitespace.
	}

	if !c.scanner.IsTokenType(plsql.PlSqlLexerPERIOD) && !c.lexer.IsIdentifier(c.scanner.GetTokenType()) {
		c.scanner.Backward(true /* skipHidden */)
	}

	if position > 0 {
		if c.lexer.IsIdentifier(c.scanner.GetTokenType()) && c.scanner.GetPreviousTokenType(false /* skipHidden */) == plsql.PlSqlLexerPERIOD {
			c.scanner.Backward(true /* skipHidden */)
		}
		if c.scanner.IsTokenType(plsql.PlSqlLexerPERIOD) && c.lexer.IsIdentifier(c.scanner.GetPreviousTokenType(false /* skipHidden */)) {
			c.scanner.Backward(true /* skipHidden */)
		}
	}

	schema := ""
	temp := ""
	if c.lexer.IsIdentifier(c.scanner.GetTokenType()) {
		temp = unquote(c.scanner.GetTokenText())
		c.scanner.Forward(true /* skipHidden */)
	}

	if !c.scanner.IsTokenType(plsql.PlSqlLexerPERIOD) || position <= c.scanner.GetIndex() {
		return schema, ObjectFlagsShowFirst | ObjectFlagsShowSecond
	}

	schema = temp
	return schema, ObjectFlagsShowSecond
}

func (c *Completer) determineGeneralElementPartCandidates() (schema, table string, flags ObjectFlags) {
	position := c.scanner.GetIndex()
	if c.scanner.GetTokenChannel() != 0 {
		c.scanner.Forward(true /* skipHidden */) // Skip whitespace.
	}

	tokenType := c.scanner.GetTokenType()
	if tokenType != plsql.PlSqlLexerPERIOD && !c.lexer.IsIdentifier(c.scanner.GetTokenType()) {
		c.scanner.Backward(true /* skipHidden */)
	}

	if position > 0 {
		if c.lexer.IsIdentifier(c.scanner.GetTokenType()) && c.scanner.GetPreviousTokenType(false /* skipHidden */) == plsql.PlSqlLexerPERIOD {
			c.scanner.Backward(true /* skipHidden */)
		}
		if c.scanner.IsTokenType(plsql.PlSqlLexerPERIOD) && c.lexer.IsIdentifier(c.scanner.GetPreviousTokenType(false /* skipHidden */)) {
			c.scanner.Backward(true /* skipHidden */)

			if c.scanner.GetPreviousTokenType(false /* skipHidden */) == plsql.PlSqlLexerPERIOD {
				c.scanner.Backward(true /* skipHidden */)
				if c.lexer.IsIdentifier(c.scanner.GetPreviousTokenType(false /* skipHidden */)) {
					c.scanner.Backward(true /* skipHidden */)
				}
			}
		}
	}

	schema = ""
	table = ""
	temp := ""
	if c.lexer.IsIdentifier(c.scanner.GetTokenType()) {
		temp = unquote(c.scanner.GetTokenText())
		c.scanner.Forward(true /* skipHidden */)
	}

	if !c.scanner.IsTokenType(plsql.PlSqlLexerPERIOD) || position <= c.scanner.GetIndex() {
		return schema, table, ObjectFlagsShowSchemas | ObjectFlagsShowTables | ObjectFlagsShowColumns
	}

	c.scanner.Forward(true /* skipHidden */) // skip dot
	table = temp
	schema = temp
	if c.lexer.IsIdentifier(c.scanner.GetTokenType()) {
		temp = unquote(c.scanner.GetTokenText())
		c.scanner.Forward(true /* skipHidden */)

		if !c.scanner.IsTokenType(plsql.PlSqlLexerPERIOD) || position <= c.scanner.GetIndex() {
			return schema, table, ObjectFlagsShowTables | ObjectFlagsShowColumns
		}

		table = temp
		return schema, table, ObjectFlagsShowColumns
	}

	return schema, table, ObjectFlagsShowTables | ObjectFlagsShowColumns
}

func unquote(s string) string {
	if len(s) < 2 {
		return strings.ToUpper(s)
	}

	if (s[0] == '`' || s[0] == '\'' || s[0] == '"') && s[0] == s[len(s)-1] {
		return s[1 : len(s)-1]
	}
	return strings.ToUpper(s)
}

func (c *Completer) collectRemainingTableReferences() {
	c.scanner.Push()

	level := 0
	for {
		found := c.scanner.GetTokenType() == plsql.PlSqlLexerFROM
		for !found {
			if !c.scanner.Forward(false /* skipHidden */) {
				break
			}

			switch c.scanner.GetTokenType() {
			case plsql.PlSqlLexerLEFT_PAREN:
				level++
			case plsql.PlSqlLexerRIGHT_PAREN:
				if level > 0 {
					level--
				}
			case plsql.PlSqlLexerFROM:
				if level == 0 {
					found = true
				}
			default:
				// Other tokens, continue scanning
			}
		}

		if !found {
			c.scanner.PopAndRestore()
			return
		}

		c.parseTableReferences(c.scanner.GetFollowingText())
		if c.scanner.GetTokenType() == plsql.PlSqlLexerFROM {
			c.scanner.Forward(false /* skipHidden */)
		}
	}
}

func (c *Completer) takeReferencesSnapshot() {
	for _, references := range c.referencesStack {
		c.references = append(c.references, references...)
	}
}

func (c *Completer) collectLeadingTableReferences(caretIndex int) {
	c.scanner.Push()

	c.scanner.SeekIndex(0)
	level := 0
	for {
		found := c.scanner.GetTokenType() == plsql.PlSqlLexerFROM
		for !found {
			if !c.scanner.Forward(false /* skipHidden */) || c.scanner.GetIndex() >= caretIndex {
				break
			}

			switch c.scanner.GetTokenType() {
			case plsql.PlSqlLexerLEFT_PAREN:
				level++
				c.referencesStack = append([][]base.TableReference{{}}, c.referencesStack...)
			case plsql.PlSqlLexerRIGHT_PAREN:
				if level == 0 {
					c.scanner.PopAndRestore()
					return // We cannot go above the initial nesting level.
				}

				level--
				c.referencesStack = c.referencesStack[1:]
			case plsql.PlSqlLexerFROM:
				found = true
			default:
				// Other tokens, continue scanning
			}
		}

		if !found {
			c.scanner.PopAndRestore()
			return // No FROM clause found.
		}

		c.parseTableReferences(c.scanner.GetFollowingText())
		if c.scanner.GetTokenType() == plsql.PlSqlLexerFROM {
			c.scanner.Forward(false /* skipHidden */)
		}
	}
}

func (c *Completer) parseTableReferences(fromClause string) {
	// We use a local parser just for the FROM clause to avoid messing up tokens on the autocompletion
	// parser (which would affect the processing of the found candidates)

	input := antlr.NewInputStream(fromClause)
	lexer := plsql.NewPlSqlLexer(input)
	tokens := antlr.NewCommonTokenStream(lexer, antlr.TokenDefaultChannel)
	parser := plsql.NewPlSqlParser(tokens)

	parser.BuildParseTrees = true
	parser.RemoveErrorListeners()
	tree := parser.From_clause()

	listener := &TableRefListener{
		context: c,
	}
	antlr.ParseTreeWalkerDefault.Walk(listener, tree)
}

type TableRefListener struct {
	*plsql.BasePlSqlParserListener

	context *Completer
	done    bool
	level   int
}

func (l *TableRefListener) ExitDml_table_expression_clause(ctx *plsql.Dml_table_expression_clauseContext) {
	if l.done {
		return
	}

	if ctx.Tableview_name() != nil && l.level == 0 {
		reference := &base.PhysicalTableReference{}
		_, reference.Schema, reference.Table = NormalizeTableViewName("", ctx.Tableview_name())
		l.context.referencesStack[0] = append(l.context.referencesStack[0], reference)
		return
	}

	if ctx.Select_statement() != nil && l.level == 0 {
		reference := &base.VirtualTableReference{}

		if span, err := GetQuerySpan(
			l.context.ctx,
			base.GetQuerySpanContext{
				InstanceID:              l.context.instanceID,
				GetDatabaseMetadataFunc: l.context.getMetadata,
				ListDatabaseNamesFunc:   l.context.listDatabaseNames,
			},
			ctx.GetParser().GetTokenStream().GetTextFromRuleContext(ctx.Select_statement()),
			l.context.defaultDatabase,
			"",
			false,
		); err == nil && span.NotFoundError == nil {
			for _, column := range span.Results {
				reference.Columns = append(reference.Columns, column.Name)
			}
		}

		l.context.referencesStack[0] = append(l.context.referencesStack[0], reference)
	}
}

func (l *TableRefListener) ExitTable_alias(ctx *plsql.Table_aliasContext) {
	if l.done {
		return
	}

	if l.level == 0 && len(l.context.referencesStack) > 0 && len(l.context.referencesStack[0]) > 0 {
		alias := NormalizeTableAlias(ctx)
		switch reference := l.context.referencesStack[0][len(l.context.referencesStack[0])-1].(type) {
		case *base.PhysicalTableReference:
			reference.Alias = alias
		case *base.VirtualTableReference:
			reference.Table = alias
		}
	}
}

func prepareParserAndScanner(statement string, caretLine int, caretOffset int) (*plsql.PlSqlParser, *plsql.PlSqlLexer, *base.Scanner) {
	statement, caretLine, caretOffset = skipHeadingSQLs(statement, caretLine, caretOffset)
	input := antlr.NewInputStream(statement)
	lexer := plsql.NewPlSqlLexer(input)
	stream := antlr.NewCommonTokenStream(lexer, antlr.TokenDefaultChannel)
	parser := plsql.NewPlSqlParser(stream)
	parser.RemoveErrorListeners()
	lexer.RemoveErrorListeners()
	scanner := base.NewScanner(stream, true /* fillInput */)
	scanner.SeekPosition(caretLine, caretOffset)
	scanner.Push()
	return parser, lexer, scanner
}

func prepareTrickyParserAndScanner(statement string, caretLine int, caretOffset int) (*plsql.PlSqlParser, *plsql.PlSqlLexer, *base.Scanner) {
	statement, caretLine, caretOffset = skipHeadingSQLs(statement, caretLine, caretOffset)
	statement, caretLine, caretOffset = skipHeadingSQLWithoutSemicolon(statement, caretLine, caretOffset)
	input := antlr.NewInputStream(statement)
	lexer := plsql.NewPlSqlLexer(input)
	stream := antlr.NewCommonTokenStream(lexer, antlr.TokenDefaultChannel)
	parser := plsql.NewPlSqlParser(stream)
	parser.RemoveErrorListeners()
	lexer.RemoveErrorListeners()
	scanner := base.NewScanner(stream, true /* fillInput */)
	scanner.SeekPosition(caretLine, caretOffset)
	scanner.Push()
	return parser, lexer, scanner
}

func notEmptySQLCount(list []base.Statement) int {
	count := 0
	for _, sql := range list {
		if !sql.Empty {
			count++
		}
	}
	return count
}

// caretLine is 1-based and caretOffset is 0-based.
func skipHeadingSQLs(statement string, caretLine int, caretOffset int) (string, int, int) {
	newCaretLine, newCaretOffset := caretLine, caretOffset
	list, err := SplitSQLForCompletion(statement)
	if err != nil || notEmptySQLCount(list) <= 1 {
		return statement, caretLine, caretOffset
	}

	caretLine-- // Convert caretLine to 0-based.

	start := 0
	for i, sql := range list {
		// End.Line is 1-based per proto spec, convert to 0-based for comparison with caretLine
		sqlEndLine := int(sql.End.GetLine()) - 1
		sqlEndColumn := int(sql.End.GetColumn())
		if sqlEndLine > caretLine || (sqlEndLine == caretLine && sqlEndColumn >= caretOffset) {
			start = i
			if i == 0 {
				// The caret is in the first SQL statement, so we don't need to skip any SQL statements.
				break
			}
			// End.Line is 1-based per proto spec, convert to 0-based
			previousSQLEndLine := int(list[i-1].End.GetLine()) - 1
			previousSQLEndColumn := int(list[i-1].End.GetColumn())
			newCaretLine = caretLine - previousSQLEndLine + 1 // Convert to 1-based.
			if caretLine == previousSQLEndLine {
				// The caret is in the same line as the last line of the previous SQL statement.
				// End.Column is 1-based exclusive, so (End.Column - 1) gives 0-based start of next statement.
				// newCaretOffset = caretOffset - (previousSQLEndColumn - 1)
				newCaretOffset = caretOffset - previousSQLEndColumn + 1
			}
			break
		}
	}

	var buf strings.Builder
	for i := start; i < len(list); i++ {
		if _, err := buf.WriteString(list[i].Text); err != nil {
			return statement, caretLine, caretOffset
		}
	}

	return buf.String(), newCaretLine, newCaretOffset
}

// caretLine is 1-based and caretOffset is 0-based.
func skipHeadingSQLWithoutSemicolon(statement string, caretLine int, caretOffset int) (string, int, int) {
	input := antlr.NewInputStream(statement)
	lexer := plsql.NewPlSqlLexer(input)
	stream := antlr.NewCommonTokenStream(lexer, antlr.TokenDefaultChannel)
	lexer.RemoveErrorListeners()
	lexerErrorListener := &base.ParseErrorListener{
		Statement: statement,
	}
	lexer.AddErrorListener(lexerErrorListener)

	stream.Fill()
	tokens := stream.GetAllTokens()
	latestSelect := 0
	newCaretLine, newCaretOffset := caretLine, caretOffset
	for _, token := range tokens {
		if token.GetLine() > caretLine || (token.GetLine() == caretLine && token.GetColumn() >= caretOffset) {
			break
		}
		if token.GetTokenType() == plsql.PlSqlLexerSELECT && token.GetColumn() == 0 {
			latestSelect = token.GetTokenIndex()
			newCaretLine = caretLine - token.GetLine() + 1 // Convert to 1-based.
			newCaretOffset = caretOffset
		}
	}

	if latestSelect == 0 {
		return statement, caretLine, caretOffset
	}
	return stream.GetTextFromInterval(antlr.Interval{Start: latestSelect, Stop: stream.Size()}), newCaretLine, newCaretOffset
}

func (c *Completer) quotedIdentifierIfNeeded(s string) string {
	if c.caretTokenIsQuoted {
		return s
	}

	if c.lexer.IsReservedKeywords(s) {
		return fmt.Sprintf(`"%s"`, s)
	}
	if s != strings.ToUpper(s) {
		return fmt.Sprintf(`"%s"`, s)
	}
	for i, r := range s {
		if i == 0 && !unicode.IsLetter(r) {
			return fmt.Sprintf(`"%s"`, s)
		}
		if i > 0 && !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '_' {
			return fmt.Sprintf(`"%s"`, s)
		}
	}
	return s
}
