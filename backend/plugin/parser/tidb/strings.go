package tidb

import (
	"bufio"
	"regexp"
	"strings"

	"github.com/antlr4-go/antlr/v4"
	parser "github.com/bytebase/tidb-parser"
	tidbast "github.com/pingcap/tidb/pkg/parser/ast"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

type StringsManipulator struct {
	s      string
	l      *parser.TiDBLexer
	stream antlr.TokenStream
	p      *parser.TiDBParser
	tree   antlr.Tree
}

func NewStringsManipulator(s string) *StringsManipulator {
	l := parser.NewTiDBLexer(antlr.NewInputStream(""))
	stream := antlr.NewCommonTokenStream(l, antlr.TokenDefaultChannel)
	return &StringsManipulator{
		s:      s,
		l:      l,
		stream: stream,
		p:      parser.NewTiDBParser(stream),
	}
}

type StringsManipulatorActionType int

const (
	StringsManipulatorActionTypeNone StringsManipulatorActionType = iota
	StringsManipulatorActionTypeDropTable
	StringsManipulatorActionTypeDropColumn
	StringsManipulatorActionTypeModifyColumnType
	StringsManipulatorActionTypeDropColumnOption
	StringsManipulatorActionTypeAddColumnOption
	StringsManipulatorActionTypeModifyColumnOption
	StringsManipulatorActionTypeDropTableConstraint
	StringsManipulatorActionTypeModifyTableConstraint
)

type StringsManipulatorAction interface {
	GetType() StringsManipulatorActionType
	GetTopLevelNaming() string
	GetSecondLevelNaming() string
}

type StringsManipulatorActionBase struct {
	Type StringsManipulatorActionType
}

func (s *StringsManipulatorActionBase) GetType() StringsManipulatorActionType {
	return s.Type
}

type StringsManipulatorActionDropTable struct {
	StringsManipulatorActionBase
	Table string
}

func (s *StringsManipulatorActionDropTable) GetTopLevelNaming() string {
	return s.Table
}

func (s *StringsManipulatorActionDropTable) GetSecondLevelNaming() string {
	return ""
}

func NewDropTableAction(tableName string) *StringsManipulatorActionDropTable {
	return &StringsManipulatorActionDropTable{
		StringsManipulatorActionBase: StringsManipulatorActionBase{
			Type: StringsManipulatorActionTypeDropTable,
		},
		Table: tableName,
	}
}

type StringsManipulatorActionDropColumn struct {
	StringsManipulatorActionBase
	Table  string
	Column string
}

func (s *StringsManipulatorActionDropColumn) GetTopLevelNaming() string {
	return s.Table
}

func (s *StringsManipulatorActionDropColumn) GetSecondLevelNaming() string {
	return s.Column
}

func NewDropColumnAction(tableName string, columnName string) *StringsManipulatorActionDropColumn {
	return &StringsManipulatorActionDropColumn{
		StringsManipulatorActionBase: StringsManipulatorActionBase{
			Type: StringsManipulatorActionTypeDropColumn,
		},
		Table:  tableName,
		Column: columnName,
	}
}

type StringsManipulatorActionModifyColumnType struct {
	StringsManipulatorActionBase
	Table  string
	Column string
	Type   string
}

func (s *StringsManipulatorActionModifyColumnType) GetTopLevelNaming() string {
	return s.Table
}

func (s *StringsManipulatorActionModifyColumnType) GetSecondLevelNaming() string {
	return s.Column
}

func NewModifyColumnTypeAction(tableName string, columnName string, columnType string) *StringsManipulatorActionModifyColumnType {
	return &StringsManipulatorActionModifyColumnType{
		StringsManipulatorActionBase: StringsManipulatorActionBase{
			Type: StringsManipulatorActionTypeModifyColumnType,
		},
		Table:  tableName,
		Column: columnName,
		Type:   columnType,
	}
}

type StringsManipulatorActionDropColumnOption struct {
	StringsManipulatorActionBase
	Table  string
	Column string
	Option tidbast.ColumnOptionType
}

func (s *StringsManipulatorActionDropColumnOption) GetTopLevelNaming() string {
	return s.Table
}

func (s *StringsManipulatorActionDropColumnOption) GetSecondLevelNaming() string {
	return s.Column
}

func NewDropColumnOptionAction(tableName string, columnName string, option tidbast.ColumnOptionType) *StringsManipulatorActionDropColumnOption {
	return &StringsManipulatorActionDropColumnOption{
		StringsManipulatorActionBase: StringsManipulatorActionBase{
			Type: StringsManipulatorActionTypeDropColumnOption,
		},
		Table:  tableName,
		Column: columnName,
		Option: option,
	}
}

type StringsManipulatorActionModifyColumnOption struct {
	StringsManipulatorActionBase
	Table           string
	Column          string
	OldOption       tidbast.ColumnOptionType
	NewOptionDefine string
}

func (s *StringsManipulatorActionModifyColumnOption) GetTopLevelNaming() string {
	return s.Table
}

func (s *StringsManipulatorActionModifyColumnOption) GetSecondLevelNaming() string {
	return s.Column
}

func NewModifyColumnOptionAction(tableName string, columnName string, oldOption tidbast.ColumnOptionType, newOptionDefine string) *StringsManipulatorActionModifyColumnOption {
	return &StringsManipulatorActionModifyColumnOption{
		StringsManipulatorActionBase: StringsManipulatorActionBase{
			Type: StringsManipulatorActionTypeModifyColumnOption,
		},
		Table:           tableName,
		Column:          columnName,
		OldOption:       oldOption,
		NewOptionDefine: newOptionDefine,
	}
}

type StringsManipulatorActionAddColumnOption struct {
	StringsManipulatorActionBase
	Table           string
	Column          string
	OptionType      tidbast.ColumnOptionType
	NewOptionDefine string
}

func (s *StringsManipulatorActionAddColumnOption) GetTopLevelNaming() string {
	return s.Table
}

func (s *StringsManipulatorActionAddColumnOption) GetSecondLevelNaming() string {
	return s.Column
}

func NewAddColumnOptionAction(tableName string, columnName string, optionType tidbast.ColumnOptionType, newOptionDefine string) *StringsManipulatorActionAddColumnOption {
	return &StringsManipulatorActionAddColumnOption{
		StringsManipulatorActionBase: StringsManipulatorActionBase{
			Type: StringsManipulatorActionTypeAddColumnOption,
		},
		Table:           tableName,
		Column:          columnName,
		OptionType:      optionType,
		NewOptionDefine: newOptionDefine,
	}
}

type StringsManipulatorActionDropTableConstraint struct {
	StringsManipulatorActionBase
	Table          string
	Constraint     tidbast.ConstraintType
	ConstraintName string
}

func (s *StringsManipulatorActionDropTableConstraint) GetTopLevelNaming() string {
	return s.Table
}

func (s *StringsManipulatorActionDropTableConstraint) GetSecondLevelNaming() string {
	return s.ConstraintName
}

func NewDropTableConstraintAction(tableName string, constraintName string) *StringsManipulatorActionDropTableConstraint {
	return &StringsManipulatorActionDropTableConstraint{
		StringsManipulatorActionBase: StringsManipulatorActionBase{
			Type: StringsManipulatorActionTypeDropTableConstraint,
		},
		Table:          tableName,
		ConstraintName: constraintName,
	}
}

type StringsManipulatorActionModifyTableConstraint struct {
	StringsManipulatorActionBase
	Table               string
	OldConstraint       tidbast.ConstraintType
	OldConstraintName   string
	NewConstraintDefine string
}

func (s *StringsManipulatorActionModifyTableConstraint) GetTopLevelNaming() string {
	return s.Table
}

func (s *StringsManipulatorActionModifyTableConstraint) GetSecondLevelNaming() string {
	return s.OldConstraintName
}

func NewModifyTableConstraintAction(tableName string, oldConstraint tidbast.ConstraintType, oldConstraintName string, newConstraintDefine string) *StringsManipulatorActionModifyTableConstraint {
	return &StringsManipulatorActionModifyTableConstraint{
		StringsManipulatorActionBase: StringsManipulatorActionBase{
			Type: StringsManipulatorActionTypeModifyTableConstraint,
		},
		Table:               tableName,
		OldConstraint:       oldConstraint,
		OldConstraintName:   oldConstraintName,
		NewConstraintDefine: newConstraintDefine,
	}
}

var (
	regexpColumn = regexp.MustCompile("^  `([^`]+)`")
)

func (s *StringsManipulator) Manipulate(actions ...StringsManipulatorAction) (string, error) {
	tableActions := make(map[string][]StringsManipulatorAction)

	for _, action := range actions {
		tableName := action.GetTopLevelNaming()
		// do copy
		action := action
		tableActions[tableName] = append(tableActions[tableName], action)
	}

	stmts, err := SplitSQL(s.s)
	if err != nil {
		return "", errors.Wrap(err, "failed to split sql")
	}

	var results []string

	for _, stmt := range stmts {
		if stmt.Empty {
			results = append(results, stmt.Text)
			continue
		}
		isCreateTable, tableName := extractTableNameForCreateTable(stmt.Text)
		if !isCreateTable {
			results = append(results, stmt.Text)
			continue
		}
		actions, ok := tableActions[tableName]
		if !ok || len(actions) == 0 {
			results = append(results, stmt.Text)
			continue
		}

		var tableDefinition strings.Builder
		var tableActions []StringsManipulatorAction
		actionsMap := make(map[string][]StringsManipulatorAction)
		for _, action := range actions {
			// do copy
			action := action
			secondName := action.GetSecondLevelNaming()
			if secondName == "" {
				tableActions = append(tableActions, action)
			} else {
				actionsMap[secondName] = append(actionsMap[secondName], action)
			}
		}

		hasDropTable := false
		for _, action := range tableActions {
			if _, ok := action.(*StringsManipulatorActionDropTable); ok {
				hasDropTable = true
				continue
			}
		}
		if hasDropTable {
			continue
		}

		scanner := bufio.NewScanner(strings.NewReader(stmt.Text))
		for scanner.Scan() {
			line := scanner.Text()

			columnMatch := regexpColumn.FindStringSubmatch(line)
			if len(columnMatch) > 1 {
				// is column definition
				columnName := columnMatch[1]
				actions, ok := actionsMap[columnName]
				if !ok || len(actions) == 0 {
					if _, err := tableDefinition.WriteString(line); err != nil {
						return "", errors.Wrap(err, "failed to write string")
					}
					continue
				}

				s.Load(line)
				if err := s.ParseColumnDef(); err != nil {
					return "", errors.Wrapf(err, "failed to parse column def: %s", line)
				}

				columnActionMap := make(map[StringsManipulatorActionType][]StringsManipulatorAction)
				for _, action := range actions {
					columnActionMap[action.GetType()] = append(columnActionMap[action.GetType()], action)
				}
				result, err := s.RewriteColumnDef(columnActionMap)
				if err != nil {
					return "", errors.Wrapf(err, "failed to rewrite column def: %s", line)
				}
				if len(result) > 0 {
					results = append(results, result)
				}
				continue
			}

			results = append(results, line)
		}
		if err := scanner.Err(); err != nil {
			return "", errors.Wrap(err, "failed to scan create table statement")
		}
	}

	return strings.Join(results, "\n"), nil
}

func (s *StringsManipulator) RewriteColumnDef(columnActionMap map[StringsManipulatorActionType][]StringsManipulatorAction) (string, error) {
	if columnActionMap == nil {
		return s.stream.GetAllText(), nil
	}
	if dropTable, exists := columnActionMap[StringsManipulatorActionTypeDropColumn]; exists && len(dropTable) > 0 {
		return "", nil
	}
	listener := &rewriter{
		rewriter: antlr.NewTokenStreamRewriter(s.stream),
		actions:  columnActionMap,
	}
	antlr.ParseTreeWalkerDefault.Walk(listener, s.tree)
	if listener.err != nil {
		return "", errors.Wrap(listener.err, "failed to rewrite column def for column")
	}
	return listener.rewriter.GetTextDefault(), nil
}

type rewriter struct {
	*parser.BaseTiDBParserListener

	rewriter *antlr.TokenStreamRewriter
	actions  map[StringsManipulatorActionType][]StringsManipulatorAction
	err      error
}

func (r *rewriter) EnterColumnDef(ctx *parser.ColumnDefContext) {
	if modifyType, exists := r.actions[StringsManipulatorActionTypeModifyColumnType]; exists && len(modifyType) > 0 {
		if len(modifyType) > 1 {
			r.err = errors.New("multiple modify column type actions")
			return
		}
		modifyType, ok := (modifyType[0]).(*StringsManipulatorActionModifyColumnType)
		if !ok {
			r.err = errors.New("invalid modify column type action")
			return
		}
		r.rewriter.ReplaceTokenDefault(ctx.DataType().GetStart(), ctx.DataType().GetStop(), modifyType.Type)
	}

	modifyOptionMap := make(map[tidbast.ColumnOptionType]*StringsManipulatorActionModifyColumnOption)
	for _, action := range r.actions[StringsManipulatorActionTypeModifyColumnOption] {
		action, ok := action.(*StringsManipulatorActionModifyColumnOption)
		if !ok {
			r.err = errors.New("invalid modify column option action")
			return
		}
		modifyOptionMap[action.OldOption] = action
	}
	dropOptionMap := make(map[tidbast.ColumnOptionType]*StringsManipulatorActionDropColumnOption)
	for _, action := range r.actions[StringsManipulatorActionTypeDropColumnOption] {
		action, ok := action.(*StringsManipulatorActionDropColumnOption)
		if !ok {
			r.err = errors.New("invalid drop column option action")
			return
		}
		dropOptionMap[action.Option] = action
	}
	if ctx.ColumnOptionList() != nil {
		for _, option := range ctx.ColumnOptionList().AllColumnOption() {
			optionType := convertColumnOptionType(option)
			if _, exists := dropOptionMap[optionType]; exists {
				r.rewriter.DeleteTokenDefault(option.GetStart(), option.GetStop())
				continue
			}
			if action, exists := modifyOptionMap[optionType]; exists {
				r.rewriter.ReplaceTokenDefault(option.GetStart(), option.GetStop(), action.NewOptionDefine)
			}
		}
	}
	newOptionBuf := strings.Builder{}
	for _, action := range r.actions[StringsManipulatorActionTypeAddColumnOption] {
		action, ok := action.(*StringsManipulatorActionAddColumnOption)
		if !ok {
			r.err = errors.New("invalid add column option action")
			return
		}
		if _, err := newOptionBuf.WriteString(" "); err != nil {
			r.err = errors.Wrap(err, "failed to write string")
			return
		}
		if _, err := newOptionBuf.WriteString(action.NewOptionDefine); err != nil {
			r.err = errors.Wrap(err, "failed to write string")
			return
		}
	}
	r.rewriter.InsertAfterDefault(ctx.GetStop().GetTokenIndex(), newOptionBuf.String())
}

func convertColumnOptionType(ctx parser.IColumnOptionContext) tidbast.ColumnOptionType {
	if ctx == nil {
		return tidbast.ColumnOptionNoOption
	}

	switch {
	case ctx.PRIMARY_SYMBOL() != nil:
		return tidbast.ColumnOptionPrimaryKey
	case ctx.NOT_SYMBOL() != nil && ctx.NULL_SYMBOL() != nil:
		return tidbast.ColumnOptionNotNull
	case ctx.AUTO_INCREMENT_SYMBOL() != nil:
		return tidbast.ColumnOptionAutoIncrement
	case ctx.DEFAULT_SYMBOL() != nil && ctx.SERIAL_SYMBOL() == nil:
		return tidbast.ColumnOptionDefaultValue
	case ctx.UNIQUE_SYMBOL() != nil:
		return tidbast.ColumnOptionUniqKey
	case ctx.NOT_SYMBOL() == nil && ctx.NULL_SYMBOL() != nil:
		return tidbast.ColumnOptionNull
	case ctx.ON_SYMBOL() != nil && ctx.UPDATE_SYMBOL() != nil:
		return tidbast.ColumnOptionOnUpdate
	case ctx.COMMENT_SYMBOL() != nil:
		return tidbast.ColumnOptionComment
	case ctx.AS_SYMBOL() != nil:
		return tidbast.ColumnOptionGenerated
	case ctx.References() != nil:
		return tidbast.ColumnOptionReference
	case ctx.COLLATE_SYMBOL() != nil:
		return tidbast.ColumnOptionCollate
	case ctx.CHECK_SYMBOL() != nil:
		return tidbast.ColumnOptionCheck
	case ctx.COLUMN_FORMAT_SYMBOL() != nil:
		return tidbast.ColumnOptionColumnFormat
	case ctx.STORAGE_SYMBOL() != nil:
		return tidbast.ColumnOptionStorage
	case ctx.AUTO_RANDOM_SYMBOL() != nil:
		return tidbast.ColumnOptionAutoRandom
	}

	return tidbast.ColumnOptionNoOption
}

func (s *StringsManipulator) Load(text string) {
	s.l.SetInputStream(antlr.NewInputStream(text))
	s.stream = antlr.NewCommonTokenStream(s.l, antlr.TokenDefaultChannel)
	s.p.SetInputStream(s.stream)
}

func (s *StringsManipulator) ParseColumnDef() error {
	lexerErrorListener := &base.ParseErrorListener{}
	s.p.SetErrorHandler(antlr.NewBailErrorStrategy())
	s.l.RemoveErrorListeners()
	s.l.AddErrorListener(lexerErrorListener)

	parserErrorListener := &base.ParseErrorListener{}
	s.p.RemoveErrorListeners()
	s.p.AddErrorListener(parserErrorListener)

	s.p.BuildParseTrees = true

	s.tree = s.p.SingleColumnDef()

	if lexerErrorListener.Err != nil {
		return lexerErrorListener.Err
	}

	if parserErrorListener.Err != nil {
		return parserErrorListener.Err
	}

	return nil
}

var (
	regexpPattern = regexp.MustCompile("(?m)^-- Table structure for `([^`]+)`")
)

func extractTableNameForCreateTable(s string) (bool, string) {
	matches := regexpPattern.FindStringSubmatch(s)
	if len(matches) > 1 {
		return true, matches[1]
	}
	return false, ""
}
