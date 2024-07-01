package plsql

import (
	"context"
	"strings"

	"github.com/antlr4-go/antlr/v4"
	plsql "github.com/bytebase/plsql-parser"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/store/model"
)

var (
	// globalFollowSetsByState is a map from state to follow sets.
	// It is shared by all PlSQL completers.
	// The FollowSetsByState is the thread-safe struct.
	globalFollowSetsByState = base.NewFollowSetsByState()
)

func Completion(ctx context.Context, cCtx base.CompletionContext, statement string, caretLine int, caretOffset int) ([]base.Candidate, error) {
	completer := NewStandardCompleter(ctx, cCtx, statement, caretLine, caretOffset)
	result, err := completer.completion()
	if err != nil {
		return nil, err
	}
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
		plsql.PlSqlParserRULE_general_element_part:     true,
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
		plsql.PlSqlParserRULE_table_var_name:           true,
		plsql.PlSqlParserRULE_schema_name:              true,
		plsql.PlSqlParserRULE_routine_name:             true,
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
		plsql.PlSqlParserRULE_cursor_name:              true,
		plsql.PlSqlParserRULE_record_name:              true,
		plsql.PlSqlParserRULE_collection_name:          true,
		plsql.PlSqlParserRULE_link_name:                true,
		plsql.PlSqlParserRULE_char_set_name:            true,
		plsql.PlSqlParserRULE_synonym_name:             true,
		plsql.PlSqlParserRULE_dir_object_name:          true,
		plsql.PlSqlParserRULE_user_object_name:         true,
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
	getMetadata         base.GetDatabaseMetadataFunc
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

func NewStandardCompleter(ctx context.Context, cCtx base.CompletionContext, statement string, caretLine int, caretOffset int) *Completer {
	parser, lexer, scanner := prepareParserAndScanner(statement, caretLine, caretOffset)
	core := base.NewCodeCompletionCore(
		parser,
		newIgnoredTokens(),
		newPreferredRules(),
		&globalFollowSetsByState,
		0, // todo
		0, // todo
		0, // todo
		0, // todo
	)
	return &Completer{
		ctx:                 ctx,
		core:                core,
		scene:               cCtx.Scene,
		parser:              parser,
		lexer:               lexer,
		scanner:             scanner,
		getMetadata:         cCtx.Metadata,
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
		if ruleName == plsql.PlSqlParserRULE_general_element_part {
			c.collectLeadingTableReferences(caretIndex)
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

func notEmptySQLCount(list []base.SingleSQL) int {
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
	list, err := SplitSQLWithoutModify(statement)
	if err != nil || notEmptySQLCount(list) <= 1 {
		return statement, caretLine, caretOffset
	}

	caretLine-- // Convert caretLine to 0-based.

	start := 0
	for i, sql := range list {
		if sql.LastLine > caretLine || (sql.LastLine == caretLine && sql.LastColumn >= caretOffset) {
			start = i
			if i == 0 {
				// The caret is in the first SQL statement, so we don't need to skip any SQL statement.
				continue
			}
			newCaretLine = caretLine - list[i-1].LastLine + 1 // Convert to 1-based.
			if caretLine == list[i-1].LastLine {
				// The caret is in the same line as the last SQL statement, so we don't need to skip any SQL statement.
				// We just need to adjust the caret offset.
				newCaretOffset = caretOffset - list[i-1].LastColumn - 1 // Convert to 0-based.
			}
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
