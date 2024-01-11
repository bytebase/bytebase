package tidb

import (
	"strings"

	"github.com/antlr4-go/antlr/v4"
	parser "github.com/bytebase/tidb-parser"
	tidbast "github.com/pingcap/tidb/pkg/parser/ast"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

const (
	PrimarySymbol = "PRIMARY"
)

type StringsManipulator struct {
	s      string
	l      *parser.TiDBLexer
	stream antlr.TokenStream
	p      *parser.TiDBParser
	tree   parser.ISingleCreateTableContext
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
	StringsManipulatorActionTypeAddColumn
	StringsManipulatorActionTypeModifyColumnType
	StringsManipulatorActionTypeDropColumnOption
	StringsManipulatorActionTypeAddColumnOption
	StringsManipulatorActionTypeModifyColumnOption
	StringsManipulatorActionTypeDropTableConstraint
	StringsManipulatorActionTypeModifyTableConstraint
	StringsManipulatorActionTypeAddTableConstraint
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

func (*StringsManipulatorActionDropTable) GetSecondLevelNaming() string {
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

type StringsManipulatorActionAddColumn struct {
	StringsManipulatorActionBase
	Table            string
	ColumnDefinition string
}

func (s *StringsManipulatorActionAddColumn) GetTopLevelNaming() string {
	return s.Table
}

func (*StringsManipulatorActionAddColumn) GetSecondLevelNaming() string {
	return ""
}

func NewAddColumnAction(tableName string, columnDefinition string) *StringsManipulatorActionAddColumn {
	return &StringsManipulatorActionAddColumn{
		StringsManipulatorActionBase: StringsManipulatorActionBase{
			Type: StringsManipulatorActionTypeAddColumn,
		},
		Table:            tableName,
		ColumnDefinition: columnDefinition,
	}
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

type StringsManipulatorActionAddTableConstraint struct {
	StringsManipulatorActionBase
	Table               string
	Type                tidbast.ConstraintType
	NewConstraintDefine string
}

func (s *StringsManipulatorActionAddTableConstraint) GetTopLevelNaming() string {
	return s.Table
}

func (*StringsManipulatorActionAddTableConstraint) GetSecondLevelNaming() string {
	return ""
}

func NewAddTableConstraintAction(tableName string, constraintType tidbast.ConstraintType, newConstraintDefine string) *StringsManipulatorActionAddTableConstraint {
	return &StringsManipulatorActionAddTableConstraint{
		StringsManipulatorActionBase: StringsManipulatorActionBase{
			Type: StringsManipulatorActionTypeAddTableConstraint,
		},
		Table:               tableName,
		Type:                constraintType,
		NewConstraintDefine: newConstraintDefine,
	}
}

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
		isCreateTable, tableName := s.extractTableNameForCreateTable(stmt.Text)
		if !isCreateTable {
			results = append(results, stmt.Text)
			continue
		}
		actions, ok := tableActions[tableName]
		if !ok || len(actions) == 0 {
			results = append(results, stmt.Text)
			continue
		}

		hasDropTable := false
		actionsMap := make(map[string][]StringsManipulatorAction)
		for _, action := range actions {
			// do copy
			action := action
			secondName := action.GetSecondLevelNaming()
			actionsMap[secondName] = append(actionsMap[secondName], action)
			if action.GetType() == StringsManipulatorActionTypeDropTable {
				hasDropTable = true
			}
		}

		if hasDropTable {
			continue
		}

		result, err := s.RewriteCreateTable(actionsMap)
		if err != nil {
			return "", errors.Wrapf(err, "failed to rewrite create table: %s", stmt.Text)
		}
		if len(result) > 0 {
			results = append(results, result)
		}
	}

	return strings.Join(results, "\n"), nil
}

func (s *StringsManipulator) RewriteCreateTable(actionsMap map[string][]StringsManipulatorAction) (string, error) {
	listener := &rewriter{
		actions: actionsMap,
	}
	antlr.ParseTreeWalkerDefault.Walk(listener, s.tree)
	if listener.err != nil {
		return "", errors.Wrap(listener.err, "failed to rewrite create table")
	}
	return listener.generateStatement()
}

type rewriter struct {
	*parser.BaseTiDBParserListener

	actions map[string][]StringsManipulatorAction
	err     error

	prefixString     string
	columnDefines    []string
	tableConstraints []string
	suffixString     string
}

func (r *rewriter) generateStatement() (string, error) {
	buf := strings.Builder{}
	if _, err := buf.WriteString(r.prefixString); err != nil {
		return "", errors.Wrap(err, "failed to write string")
	}
	if len(r.columnDefines) > 0 {
		if _, err := buf.WriteString("\n  "); err != nil {
			return "", errors.Wrap(err, "failed to write string")
		}
		if _, err := buf.WriteString(strings.Join(r.columnDefines, ",\n  ")); err != nil {
			return "", errors.Wrap(err, "failed to write string")
		}
	}
	if len(r.tableConstraints) > 0 {
		if len(r.columnDefines) > 0 {
			if _, err := buf.WriteString(",\n  "); err != nil {
				return "", errors.Wrap(err, "failed to write string")
			}
		}
		if _, err := buf.WriteString(strings.Join(r.tableConstraints, ",\n  ")); err != nil {
			return "", errors.Wrap(err, "failed to write string")
		}
	}
	if err := buf.WriteByte('\n'); err != nil {
		return "", errors.Wrap(err, "failed to write byte")
	}
	if _, err := buf.WriteString(r.suffixString); err != nil {
		return "", errors.Wrap(err, "failed to write string")
	}
	return buf.String(), nil
}

func (r *rewriter) EnterCreateTable(ctx *parser.CreateTableContext) {
	if r.err != nil {
		return
	}

	if ctx.OPEN_PAR_SYMBOL() == nil {
		r.err = errors.New("invalid create table statement: no open parenthesis")
		return
	}
	if ctx.CLOSE_PAR_SYMBOL() == nil {
		r.err = errors.New("invalid create table statement: no close parenthesis")
		return
	}
	r.prefixString = ctx.GetParser().GetTokenStream().GetTextFromInterval(
		antlr.NewInterval(
			0, // We need to include the non-default channel tokens
			ctx.OPEN_PAR_SYMBOL().GetSourceInterval().Stop,
		),
	)
	r.suffixString = ctx.GetParser().GetTokenStream().GetTextFromInterval(
		antlr.NewInterval(
			ctx.CLOSE_PAR_SYMBOL().GetSourceInterval().Start,
			ctx.GetParser().GetTokenStream().Size()-1, // We need to include the non-default channel tokens
		),
	)
}

func (r *rewriter) ExitCreateTable(_ *parser.CreateTableContext) {
	if r.err != nil {
		return
	}
	actions, exists := r.actions[""]
	if exists && len(actions) > 0 {
		for _, action := range actions {
			switch action := action.(type) {
			case *StringsManipulatorActionAddColumn:
				r.columnDefines = append(r.columnDefines, action.ColumnDefinition)
			case *StringsManipulatorActionAddTableConstraint:
				r.tableConstraints = append(r.tableConstraints, action.NewConstraintDefine)
			default:
				r.err = errors.Errorf("invalid table action in ExitCreateTable: %T", action)
				return
			}
		}
	}
}

func (r *rewriter) EnterTableConstraintDef(ctx *parser.TableConstraintDefContext) {
	if r.err != nil {
		return
	}

	secondName := extractTableConstraintName(ctx)
	if secondName == "" {
		r.err = errors.New("invalid table constraint name")
		return
	}

	actions, exists := r.actions[secondName]
	if !exists {
		// no action for this table constraint
		return
	}
	// column name and constraint name are in the different namespace for tidb
	for _, action := range actions {
		switch action := action.(type) {
		case *StringsManipulatorActionDropTableConstraint:
			return
		case *StringsManipulatorActionModifyTableConstraint:
			r.tableConstraints = append(r.tableConstraints, action.NewConstraintDefine)
		}
	}
}

func extractTableConstraintName(ctx *parser.TableConstraintDefContext) string {
	if ctx.PRIMARY_SYMBOL() != nil {
		return PrimarySymbol
	}
	if ctx.ConstraintName() != nil {
		return NormalizeConstraintName(ctx.ConstraintName())
	}
	if ctx.IndexName() != nil {
		return NormalizeIndexName(ctx.IndexName())
	}
	if ctx.IndexNameAndType() != nil {
		return NormalizeIndexName(ctx.IndexNameAndType().IndexName())
	}
	return ""
}

func (r *rewriter) EnterColumnDef(ctx *parser.ColumnDefContext) {
	if r.err != nil {
		return
	}

	_, _, columnName := NormalizeTiDBColumnName(ctx.ColumnName())
	if columnName == "" {
		r.err = errors.New("invalid column name")
		return
	}

	actions, exists := r.actions[columnName]
	if !exists {
		r.columnDefines = append(r.columnDefines, ctx.GetParser().GetTokenStream().GetTextFromRuleContext(ctx))
		return
	}
	actionsMap := make(map[StringsManipulatorActionType][]StringsManipulatorAction)
	for _, action := range actions {
		if action.GetType() == StringsManipulatorActionTypeDropColumn {
			// drop column action is special, we need to handle it first
			return
		}
		actionsMap[action.GetType()] = append(actionsMap[action.GetType()], action)
	}

	buf := strings.Builder{}
	if _, err := buf.WriteString(ctx.GetParser().GetTokenStream().GetTextFromRuleContext(ctx.ColumnName())); err != nil {
		r.err = errors.Wrap(err, "failed to write string")
		return
	}
	if err := buf.WriteByte(' '); err != nil {
		r.err = errors.Wrap(err, "failed to write byte")
		return
	}

	if modifyType, exists := actionsMap[StringsManipulatorActionTypeModifyColumnType]; exists && len(modifyType) > 0 {
		if len(modifyType) > 1 {
			r.err = errors.New("multiple modify column type actions")
			return
		}
		modifyType, ok := (modifyType[0]).(*StringsManipulatorActionModifyColumnType)
		if !ok {
			r.err = errors.New("invalid modify column type action")
			return
		}
		if _, err := buf.WriteString(modifyType.Type); err != nil {
			r.err = errors.Wrap(err, "failed to write string")
			return
		}
	} else {
		if _, err := buf.WriteString(ctx.GetParser().GetTokenStream().GetTextFromRuleContext(ctx.DataType())); err != nil {
			r.err = errors.Wrap(err, "failed to write string")
			return
		}
	}

	modifyOptionMap := make(map[tidbast.ColumnOptionType]*StringsManipulatorActionModifyColumnOption)
	for _, action := range actionsMap[StringsManipulatorActionTypeModifyColumnOption] {
		action, ok := action.(*StringsManipulatorActionModifyColumnOption)
		if !ok {
			r.err = errors.New("invalid modify column option action")
			return
		}
		modifyOptionMap[action.OldOption] = action
	}
	dropOptionMap := make(map[tidbast.ColumnOptionType]*StringsManipulatorActionDropColumnOption)
	for _, action := range actionsMap[StringsManipulatorActionTypeDropColumnOption] {
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
				// Drop column option
				continue
			}
			if err := buf.WriteByte(' '); err != nil {
				r.err = errors.Wrap(err, "failed to write byte")
				return
			}
			// Modify column option
			if action, exists := modifyOptionMap[optionType]; exists {
				if _, err := buf.WriteString(action.NewOptionDefine); err != nil {
					r.err = errors.Wrap(err, "failed to write string")
					return
				}
				continue
			}
			// Original column option
			if _, err := buf.WriteString(ctx.GetParser().GetTokenStream().GetTextFromRuleContext(option)); err != nil {
				r.err = errors.Wrap(err, "failed to write string")
				return
			}
		}
	}
	for _, action := range actionsMap[StringsManipulatorActionTypeAddColumnOption] {
		action, ok := action.(*StringsManipulatorActionAddColumnOption)
		if !ok {
			r.err = errors.New("invalid add column option action")
			return
		}
		if _, err := buf.WriteString(" "); err != nil {
			r.err = errors.Wrap(err, "failed to write string")
			return
		}
		if _, err := buf.WriteString(action.NewOptionDefine); err != nil {
			r.err = errors.Wrap(err, "failed to write string")
			return
		}
	}
	r.columnDefines = append(r.columnDefines, buf.String())
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

func (s *StringsManipulator) ParseCreateTable() error {
	lexerErrorListener := &base.ParseErrorListener{}
	s.p.SetErrorHandler(antlr.NewBailErrorStrategy())
	s.l.RemoveErrorListeners()
	s.l.AddErrorListener(lexerErrorListener)

	parserErrorListener := &base.ParseErrorListener{}
	s.p.RemoveErrorListeners()
	s.p.AddErrorListener(parserErrorListener)

	s.p.BuildParseTrees = true

	s.tree = s.p.SingleCreateTable()

	if lexerErrorListener.Err != nil {
		return lexerErrorListener.Err
	}

	if parserErrorListener.Err != nil {
		return parserErrorListener.Err
	}

	return nil
}

func (s *StringsManipulator) extractTableNameForCreateTable(text string) (bool, string) {
	s.Load(text)
	if err := s.ParseCreateTable(); err != nil {
		return false, ""
	}
	if s.tree == nil || s.tree.CreateTable() == nil || s.tree.CreateTable().TableName() == nil {
		return false, ""
	}
	_, tableName := NormalizeTiDBTableName(s.tree.CreateTable().TableName())
	if tableName == "" {
		return false, ""
	}
	return true, tableName
}
