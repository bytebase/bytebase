package tidb

import (
	"regexp"
	"strings"

	"github.com/antlr4-go/antlr/v4"
	parser "github.com/bytebase/tidb-parser"
	tidbast "github.com/pingcap/tidb/pkg/parser/ast"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

const (
	PrimarySymbol                       = "PRIMARY"
	restoreUniqueAndForeignKeyCheckStmt = "SET FOREIGN_KEY_CHECKS=@OLD_FOREIGN_KEY_CHECKS;"
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
	StringsManipulatorActionTypeAddTable
	StringsManipulatorActionTypeDropColumn
	StringsManipulatorActionTypeAddColumn
	StringsManipulatorActionTypeModifyColumnType
	StringsManipulatorActionTypeDropColumnOption
	StringsManipulatorActionTypeAddColumnOption
	StringsManipulatorActionTypeModifyColumnOption
	StringsManipulatorActionTypeDropTableConstraint
	StringsManipulatorActionTypeModifyTableConstraint
	StringsManipulatorActionTypeAddTableConstraint
	StringsManipulatorActionTypeDropTableOption
	StringsManipulatorActionTypeModifyTableOption
	StringsManipulatorActionTypeAddTableOption
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

type StringsManipulatorActionAddTable struct {
	StringsManipulatorActionBase
	TableDefinition string
}

func (*StringsManipulatorActionAddTable) GetTopLevelNaming() string {
	return ""
}

func (*StringsManipulatorActionAddTable) GetSecondLevelNaming() string {
	return ""
}

func NewAddTableAction(tableDefinition string) *StringsManipulatorActionAddTable {
	return &StringsManipulatorActionAddTable{
		StringsManipulatorActionBase: StringsManipulatorActionBase{
			Type: StringsManipulatorActionTypeAddTable,
		},
		TableDefinition: tableDefinition,
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

type StringsManipulatorActionColumnOptionBase struct {
	StringsManipulatorActionBase
	Type tidbast.ColumnOptionType
}

func (s *StringsManipulatorActionColumnOptionBase) GetOptionType() tidbast.ColumnOptionType {
	return s.Type
}

type StringsManipulatorActionDropColumnOption struct {
	StringsManipulatorActionColumnOptionBase
	Table  string
	Column string
}

func (s *StringsManipulatorActionDropColumnOption) GetTopLevelNaming() string {
	return s.Table
}

func (s *StringsManipulatorActionDropColumnOption) GetSecondLevelNaming() string {
	return s.Column
}

func NewDropColumnOptionAction(tableName string, columnName string, option tidbast.ColumnOptionType) *StringsManipulatorActionDropColumnOption {
	return &StringsManipulatorActionDropColumnOption{
		StringsManipulatorActionColumnOptionBase: StringsManipulatorActionColumnOptionBase{
			StringsManipulatorActionBase: StringsManipulatorActionBase{
				Type: StringsManipulatorActionTypeDropColumnOption,
			},
			Type: option,
		},
		Table:  tableName,
		Column: columnName,
	}
}

type StringsManipulatorActionModifyColumnOption struct {
	StringsManipulatorActionColumnOptionBase
	Table           string
	Column          string
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
		StringsManipulatorActionColumnOptionBase: StringsManipulatorActionColumnOptionBase{
			StringsManipulatorActionBase: StringsManipulatorActionBase{
				Type: StringsManipulatorActionTypeModifyColumnOption,
			},
			Type: oldOption,
		},
		Table:           tableName,
		Column:          columnName,
		NewOptionDefine: newOptionDefine,
	}
}

type StringsManipulatorActionAddColumnOption struct {
	StringsManipulatorActionColumnOptionBase
	Table           string
	Column          string
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
		StringsManipulatorActionColumnOptionBase: StringsManipulatorActionColumnOptionBase{
			StringsManipulatorActionBase: StringsManipulatorActionBase{
				Type: StringsManipulatorActionTypeAddColumnOption,
			},
			Type: optionType,
		},
		Table:           tableName,
		Column:          columnName,
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

type StringsManipulatorActionDropTableOption struct {
	StringsManipulatorActionBase
	Table     string
	OldOption tidbast.TableOptionType
}

func (s *StringsManipulatorActionDropTableOption) GetTopLevelNaming() string {
	return s.Table
}

func (*StringsManipulatorActionDropTableOption) GetSecondLevelNaming() string {
	return ""
}

func NewDropTableOptionAction(tableName string, oldOption tidbast.TableOptionType) *StringsManipulatorActionDropTableOption {
	return &StringsManipulatorActionDropTableOption{
		StringsManipulatorActionBase: StringsManipulatorActionBase{
			Type: StringsManipulatorActionTypeDropTableOption,
		},
		Table:     tableName,
		OldOption: oldOption,
	}
}

type StringsManipulatorActionModifyTableOption struct {
	StringsManipulatorActionBase
	Table          string
	OldOption      tidbast.TableOptionType
	NewOptionValue string
}

func (s *StringsManipulatorActionModifyTableOption) GetTopLevelNaming() string {
	return s.Table
}

func (*StringsManipulatorActionModifyTableOption) GetSecondLevelNaming() string {
	return ""
}

func NewModifyTableOptionAction(tableName string, oldOption tidbast.TableOptionType, newOptionValue string) *StringsManipulatorActionModifyTableOption {
	return &StringsManipulatorActionModifyTableOption{
		StringsManipulatorActionBase: StringsManipulatorActionBase{
			Type: StringsManipulatorActionTypeModifyTableOption,
		},
		Table:          tableName,
		OldOption:      oldOption,
		NewOptionValue: newOptionValue,
	}
}

type StringsManipulatorActionAddTableOption struct {
	StringsManipulatorActionBase
	Table          string
	NewOptionValue string
}

func (s *StringsManipulatorActionAddTableOption) GetTopLevelNaming() string {
	return s.Table
}

func (*StringsManipulatorActionAddTableOption) GetSecondLevelNaming() string {
	return ""
}

func NewAddTableOptionAction(tableName string, newOptionValue string) *StringsManipulatorActionAddTableOption {
	return &StringsManipulatorActionAddTableOption{
		StringsManipulatorActionBase: StringsManipulatorActionBase{
			Type: StringsManipulatorActionTypeAddTableOption,
		},
		Table:          tableName,
		NewOptionValue: newOptionValue,
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

	stmts, err := SplitSQLKeepEmptyBlocks(s.s)
	if err != nil {
		return "", errors.Wrap(err, "failed to split sql")
	}

	var results []string
	doneAddTableActions := false

	for _, stmt := range stmts {
		if stmt.Empty {
			results = append(results, stmt.Text)
			continue
		}
		isCreateTable, tableName := s.extractTableNameForCreateTable(stmt.Text)
		if !isCreateTable {
			if strings.Contains(stmt.Text, restoreUniqueAndForeignKeyCheckStmt) {
				// Do add table actions
				doneAddTableActions = true
				addTableActions := tableActions[""]
				for _, action := range addTableActions {
					if action.GetType() == StringsManipulatorActionTypeAddTable {
						results = append(results, action.(*StringsManipulatorActionAddTable).TableDefinition)
					}
				}
			}
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

	if !doneAddTableActions {
		addTableActions := tableActions[""]
		for _, action := range addTableActions {
			if action.GetType() == StringsManipulatorActionTypeAddTable {
				results = append(results, action.(*StringsManipulatorActionAddTable).TableDefinition)
			}
		}
	}

	// Add a empty line at the end of the file
	results = append(results, "")

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

	closeParIndex int
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
	r.closeParIndex = ctx.CLOSE_PAR_SYMBOL().GetSourceInterval().Start
}

func (r *rewriter) ExitCreateTable(ctx *parser.CreateTableContext) {
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
			case *StringsManipulatorActionAddTableOption:
				if ctx.CreateTableOptions() != nil {
					r.suffixString = strings.Join([]string{
						ctx.GetParser().GetTokenStream().GetTextFromInterval(antlr.Interval{
							Start: r.closeParIndex,
							Stop:  ctx.CreateTableOptions().GetStop().GetTokenIndex(),
						}),
						" ",
						action.NewOptionValue,
						ctx.GetParser().GetTokenStream().GetTextFromInterval(antlr.Interval{
							Start: ctx.CreateTableOptions().GetStop().GetTokenIndex() + 1,
							Stop:  ctx.GetParser().GetTokenStream().Size() - 1,
						}),
					}, "")
				} else {
					r.suffixString = strings.Join([]string{
						") ",
						action.NewOptionValue,
						" ",
						ctx.GetParser().GetTokenStream().GetTextFromInterval(antlr.Interval{
							Start: r.closeParIndex + 1,
							Stop:  ctx.GetParser().GetTokenStream().Size() - 1,
						}),
					}, "")
				}
			}
		}
	}
}

func getIndexUntilNewLine(start int, stream antlr.TokenStream) int {
	for i := start; i < stream.Size(); i++ {
		token := stream.Get(i)
		if token.GetChannel() == antlr.TokenDefaultChannel {
			return i - 1
		}
		switch token.GetTokenType() {
		case parser.TiDBLexerWHITESPACE:
			if token.GetText() == "\n" {
				return i - 1
			}
		case antlr.TokenEOF:
			return i - 1
		}
	}
	return stream.Size() - 1
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
		// We need to remain the original constraint definition with comments.
		stop := getIndexUntilNewLine(ctx.GetStop().GetTokenIndex()+1, ctx.GetParser().GetTokenStream())
		r.tableConstraints = append(r.tableConstraints, ctx.GetParser().GetTokenStream().GetTextFromInterval(
			antlr.NewInterval(
				ctx.GetStart().GetTokenIndex(),
				stop,
			),
		))
		return
	}
	// column name and constraint name are in the different namespace for tidb
	for _, action := range actions {
		switch action := action.(type) {
		case *StringsManipulatorActionDropTableConstraint:
			return
		case *StringsManipulatorActionModifyTableConstraint:
			var buf strings.Builder
			if _, err := buf.WriteString(action.NewConstraintDefine); err != nil {
				r.err = errors.Wrap(err, "failed to write string")
				return
			}
			// Add following comments.
			// TODO: we need to add comments for the constraint definition.
			if ctx.AllIndexOption() != nil {
				for _, option := range ctx.AllIndexOption() {
					if option.CommonIndexOption() != nil && option.CommonIndexOption().COMMENT_SYMBOL() != nil {
						if err := buf.WriteByte(' '); err != nil {
							r.err = errors.Wrap(err, "failed to write byte")
							return
						}
						if _, err := buf.WriteString(ctx.GetParser().GetTokenStream().GetTextFromRuleContext(option)); err != nil {
							r.err = errors.Wrap(err, "failed to write string")
							return
						}
					}
				}
			}
			r.tableConstraints = append(r.tableConstraints, buf.String())
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

func (r *rewriter) EnterCreateTableOption(ctx *parser.CreateTableOptionContext) {
	if r.err != nil {
		return
	}

	if ctx.GetOption() == nil {
		return
	}
	switch ctx.GetOption().GetTokenType() {
	case parser.TiDBParserCOMMENT_SYMBOL:
		for _, action := range r.actions[""] {
			switch action := action.(type) {
			case *StringsManipulatorActionDropTableOption:
				if action.OldOption == tidbast.TableOptionComment {
					r.suffixString = strings.Join([]string{
						ctx.GetParser().GetTokenStream().GetTextFromInterval(antlr.Interval{
							Start: r.closeParIndex,
							Stop:  ctx.GetStart().GetTokenIndex() - 1,
						}),
						ctx.GetParser().GetTokenStream().GetTextFromInterval(antlr.Interval{
							Start: ctx.GetStop().GetTokenIndex() + 1,
							Stop:  ctx.GetParser().GetTokenStream().Size() - 1,
						}),
					}, "")
				}
				return
			case *StringsManipulatorActionModifyTableOption:
				if action.OldOption == tidbast.TableOptionComment {
					r.suffixString = strings.Join([]string{
						ctx.GetParser().GetTokenStream().GetTextFromInterval(antlr.Interval{
							Start: r.closeParIndex,
							Stop:  ctx.GetStart().GetTokenIndex() - 1,
						}),
						action.NewOptionValue,
						ctx.GetParser().GetTokenStream().GetTextFromInterval(antlr.Interval{
							Start: ctx.GetStop().GetTokenIndex() + 1,
							Stop:  ctx.GetParser().GetTokenStream().Size() - 1,
						}),
					}, "")
					return
				}
			}
		}
	default:
		// We only support comment option for now
		return
	}
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
		// We need to remain the original column definition with comments.
		stop := getIndexUntilNewLine(ctx.GetStop().GetTokenIndex()+1, ctx.GetParser().GetTokenStream())
		r.columnDefines = append(r.columnDefines, ctx.GetParser().GetTokenStream().GetTextFromInterval(
			antlr.NewInterval(
				ctx.GetStart().GetTokenIndex(),
				stop,
			),
		))
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

	optionMap := make(map[tidbast.ColumnOptionType]parser.IColumnOptionContext)
	if ctx.ColumnOptionList() != nil {
		for _, option := range ctx.ColumnOptionList().AllColumnOption() {
			optionType := convertColumnOptionType(option)
			optionMap[optionType] = option
		}
	}
	stop := getIndexUntilNewLine(ctx.GetStop().GetTokenIndex()+1, ctx.GetParser().GetTokenStream())
	autoRand := extractAutoRand(ctx.GetParser().GetTokenStream().GetTextFromInterval(
		antlr.NewInterval(
			ctx.GetStop().GetTokenIndex()+1,
			stop,
		),
	))

	optionActionMap := make(map[tidbast.ColumnOptionType]StringsManipulatorAction)
	for _, action := range actionsMap[StringsManipulatorActionTypeDropColumnOption] {
		action, ok := action.(*StringsManipulatorActionDropColumnOption)
		if !ok {
			r.err = errors.New("invalid drop column option action")
			return
		}
		optionActionMap[action.GetOptionType()] = action
	}
	for _, action := range actionsMap[StringsManipulatorActionTypeModifyColumnOption] {
		action, ok := action.(*StringsManipulatorActionModifyColumnOption)
		if !ok {
			r.err = errors.New("invalid modify column option action")
			return
		}
		optionActionMap[action.GetOptionType()] = action
	}
	for _, action := range actionsMap[StringsManipulatorActionTypeAddColumnOption] {
		action, ok := action.(*StringsManipulatorActionAddColumnOption)
		if !ok {
			r.err = errors.New("invalid add column option action")
			return
		}
		optionActionMap[action.GetOptionType()] = action
	}

	// Generate column options.
	// TODO: we don't support generated column for now.
	if option, exists := optionMap[tidbast.ColumnOptionGenerated]; exists {
		if err := buf.WriteByte(' '); err != nil {
			r.err = errors.Wrap(err, "failed to write byte")
			return
		}
		if _, err := buf.WriteString(option.GetParser().GetTokenStream().GetTextFromRuleContext(option)); err != nil {
			r.err = errors.Wrap(err, "failed to write string")
			return
		}
	}

	// NULL and NOT NULL.
	if err := writeNullableOption(&buf, optionMap, optionActionMap); err != nil {
		r.err = err
		return
	}

	// DEFAULT, AUTO_INCREMENT and AUTO_RANDOM.
	if err := writeDefaultLikeOption(&buf, optionMap, optionActionMap, autoRand); err != nil {
		r.err = err
		return
	}

	// ON UPDATE.
	// TODO: we don't support ON UPDATE for now.
	if option, exists := optionMap[tidbast.ColumnOptionOnUpdate]; exists {
		if err := buf.WriteByte(' '); err != nil {
			r.err = errors.Wrap(err, "failed to write byte")
			return
		}
		if _, err := buf.WriteString(option.GetParser().GetTokenStream().GetTextFromRuleContext(option)); err != nil {
			r.err = errors.Wrap(err, "failed to write string")
			return
		}
	}

	// COMMENT.
	if err := writeColumnCommentOption(&buf, optionMap, optionActionMap); err != nil {
		r.err = err
		return
	}

	r.columnDefines = append(r.columnDefines, buf.String())
}

func writeColumnCommentOption(buf *strings.Builder, optionMap map[tidbast.ColumnOptionType]parser.IColumnOptionContext, optionActionMap map[tidbast.ColumnOptionType]StringsManipulatorAction) error {
	needOrigin := true
	if action, exists := optionActionMap[tidbast.ColumnOptionComment]; exists {
		switch action := action.(type) {
		case *StringsManipulatorActionDropColumnOption:
			// Drop column option
			needOrigin = false
		case *StringsManipulatorActionModifyColumnOption:
			needOrigin = false
			if err := buf.WriteByte(' '); err != nil {
				return errors.Wrap(err, "failed to write byte")
			}
			if _, err := buf.WriteString(action.NewOptionDefine); err != nil {
				return errors.Wrap(err, "failed to write string")
			}
		case *StringsManipulatorActionAddColumnOption:
			needOrigin = false
			if err := buf.WriteByte(' '); err != nil {
				return errors.Wrap(err, "failed to write byte")
			}
			if _, err := buf.WriteString(action.NewOptionDefine); err != nil {
				return errors.Wrap(err, "failed to write string")
			}
		}
	}
	if option, exists := optionMap[tidbast.ColumnOptionComment]; exists && needOrigin {
		if err := buf.WriteByte(' '); err != nil {
			return errors.Wrap(err, "failed to write byte")
		}
		if _, err := buf.WriteString(option.GetParser().GetTokenStream().GetTextFromRuleContext(option)); err != nil {
			return errors.Wrap(err, "failed to write string")
		}
	}
	return nil
}

func writeDefaultLikeOption(buf *strings.Builder, optionMap map[tidbast.ColumnOptionType]parser.IColumnOptionContext, optionActionMap map[tidbast.ColumnOptionType]StringsManipulatorAction, autoRand string) error {
	needOrigin := true
	if action, exists := optionActionMap[tidbast.ColumnOptionDefaultValue]; exists {
		switch action := action.(type) {
		case *StringsManipulatorActionDropColumnOption:
			// Drop column option
			needOrigin = false
		case *StringsManipulatorActionModifyColumnOption:
			if err := buf.WriteByte(' '); err != nil {
				return errors.Wrap(err, "failed to write byte")
			}
			if _, err := buf.WriteString(action.NewOptionDefine); err != nil {
				return errors.Wrap(err, "failed to write string")
			}
			needOrigin = false
		case *StringsManipulatorActionAddColumnOption:
			needOrigin = false
			if err := buf.WriteByte(' '); err != nil {
				return errors.Wrap(err, "failed to write byte")
			}
			if _, err := buf.WriteString(action.NewOptionDefine); err != nil {
				return errors.Wrap(err, "failed to write string")
			}
		}
	}
	if option, exists := optionMap[tidbast.ColumnOptionDefaultValue]; exists && needOrigin {
		if err := buf.WriteByte(' '); err != nil {
			return errors.Wrap(err, "failed to write byte")
		}
		if _, err := buf.WriteString(option.GetParser().GetTokenStream().GetTextFromRuleContext(option)); err != nil {
			return errors.Wrap(err, "failed to write string")
		}
		needOrigin = false
	}

	if action, exists := optionActionMap[tidbast.ColumnOptionAutoIncrement]; exists {
		switch action := action.(type) {
		case *StringsManipulatorActionDropColumnOption:
			// Drop column option
			needOrigin = false
		case *StringsManipulatorActionModifyColumnOption:
			return errors.New("invalid modify column option action: modify auto_increment")
		case *StringsManipulatorActionAddColumnOption:
			needOrigin = false
			if err := buf.WriteByte(' '); err != nil {
				return errors.Wrap(err, "failed to write byte")
			}
			if _, err := buf.WriteString(action.NewOptionDefine); err != nil {
				return errors.Wrap(err, "failed to write string")
			}
		}
	}
	if option, exists := optionMap[tidbast.ColumnOptionAutoIncrement]; exists && needOrigin {
		if err := buf.WriteByte(' '); err != nil {
			return errors.Wrap(err, "failed to write byte")
		}
		if _, err := buf.WriteString(option.GetParser().GetTokenStream().GetTextFromRuleContext(option)); err != nil {
			return errors.Wrap(err, "failed to write string")
		}
		needOrigin = false
	}

	if action, exists := optionActionMap[tidbast.ColumnOptionAutoRandom]; exists {
		switch action := action.(type) {
		case *StringsManipulatorActionDropColumnOption:
			// Drop column option
			needOrigin = false
		case *StringsManipulatorActionModifyColumnOption:
			return errors.New("invalid modify column option action: modify auto_random")
		case *StringsManipulatorActionAddColumnOption:
			needOrigin = false
			if err := buf.WriteByte(' '); err != nil {
				return errors.Wrap(err, "failed to write byte")
			}
			if _, err := buf.WriteString(action.NewOptionDefine); err != nil {
				return errors.Wrap(err, "failed to write string")
			}
		}
	}
	if needOrigin && autoRand != "" {
		if err := buf.WriteByte(' '); err != nil {
			return errors.Wrap(err, "failed to write byte")
		}
		if _, err := buf.WriteString(autoRand); err != nil {
			return errors.Wrap(err, "failed to write string")
		}
	}
	return nil
}

func writeNullableOption(buf *strings.Builder, optionMap map[tidbast.ColumnOptionType]parser.IColumnOptionContext, optionActionMap map[tidbast.ColumnOptionType]StringsManipulatorAction) error {
	needOrigin := true
	if action, exists := optionActionMap[tidbast.ColumnOptionNull]; exists {
		switch action.(type) {
		case *StringsManipulatorActionDropColumnOption:
			// Drop column option
			needOrigin = false
		case *StringsManipulatorActionModifyColumnOption:
			return errors.New("invalid modify column option action: modify null")
		case *StringsManipulatorActionAddColumnOption:
			needOrigin = false
			if _, err := buf.WriteString(" NULL"); err != nil {
				return errors.Wrap(err, "failed to write string")
			}
		}
	}
	if option, exists := optionMap[tidbast.ColumnOptionNull]; exists && needOrigin {
		if err := buf.WriteByte(' '); err != nil {
			return errors.Wrap(err, "failed to write byte")
		}
		if _, err := buf.WriteString(option.GetParser().GetTokenStream().GetTextFromRuleContext(option)); err != nil {
			return errors.Wrap(err, "failed to write string")
		}
		needOrigin = false
	}
	if action, exists := optionActionMap[tidbast.ColumnOptionNotNull]; exists {
		switch action.(type) {
		case *StringsManipulatorActionDropColumnOption:
			// Drop column option
			needOrigin = false
		case *StringsManipulatorActionModifyColumnOption:
			return errors.New("invalid modify column option action: modify not null")
		case *StringsManipulatorActionAddColumnOption:
			needOrigin = false
			if _, err := buf.WriteString(" NOT NULL"); err != nil {
				return errors.Wrap(err, "failed to write string")
			}
		}
	}
	if option, exists := optionMap[tidbast.ColumnOptionNotNull]; exists && needOrigin {
		if err := buf.WriteByte(' '); err != nil {
			return errors.Wrap(err, "failed to write byte")
		}
		if _, err := buf.WriteString(option.GetParser().GetTokenStream().GetTextFromRuleContext(option)); err != nil {
			return errors.Wrap(err, "failed to write string")
		}
	}
	return nil
}

var (
	autoRandPattern = regexp.MustCompile(`/\*T!\[auto_rand\] AUTO_RANDOM\((\d+(?:,\s*\d+)*)\) \*/`)
)

func extractAutoRand(s string) string {
	match := autoRandPattern.FindStringSubmatch(s)
	if len(match) > 0 {
		return match[0]
	}
	return ""
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
