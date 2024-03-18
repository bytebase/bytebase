package plsql

import (
	"strings"

	"github.com/antlr4-go/antlr/v4"
	parser "github.com/bytebase/plsql-parser"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

type StringsManipulator struct {
	tree   antlr.Tree
	stream antlr.TokenStream
}

func NewStringsManipulator(tree antlr.Tree, stream antlr.TokenStream) *StringsManipulator {
	return &StringsManipulator{tree, stream}
}

type StringsManipulatorActionDropTable struct {
	base.StringsManipulatorActionBase
	Table string
}

func (s *StringsManipulatorActionDropTable) GetTopLevelNaming() string {
	return s.Table
}

func (*StringsManipulatorActionDropTable) GetSecondLevelNaming() string {
	return ""
}

func NewDropTableAction(schemaName, tableName string) *StringsManipulatorActionDropTable {
	return &StringsManipulatorActionDropTable{
		StringsManipulatorActionBase: base.StringsManipulatorActionBase{
			Type:       base.StringsManipulatorActionTypeDropTable,
			SchemaName: schemaName,
		},
		Table: tableName,
	}
}

type StringsManipulatorActionAddTable struct {
	base.StringsManipulatorActionBase
	Table           string
	TableDefinition string
}

func (s *StringsManipulatorActionAddTable) GetTopLevelNaming() string {
	return s.Table
}

func (*StringsManipulatorActionAddTable) GetSecondLevelNaming() string {
	return ""
}

func NewAddTableAction(schemaName, tableName, tableDefinition string) *StringsManipulatorActionAddTable {
	return &StringsManipulatorActionAddTable{
		StringsManipulatorActionBase: base.StringsManipulatorActionBase{
			Type:       base.StringsManipulatorActionTypeAddTable,
			SchemaName: schemaName,
		},
		Table:           tableName,
		TableDefinition: tableDefinition,
	}
}

type StringsManipulatorActionDropColumn struct {
	base.StringsManipulatorActionBase
	Table  string
	Column string
}

func (s *StringsManipulatorActionDropColumn) GetTopLevelNaming() string {
	return s.Table
}

func (s *StringsManipulatorActionDropColumn) GetSecondLevelNaming() string {
	return s.Column
}

func NewDropColumnAction(schemaName, tableName, columnName string) *StringsManipulatorActionDropColumn {
	return &StringsManipulatorActionDropColumn{
		StringsManipulatorActionBase: base.StringsManipulatorActionBase{
			Type:       base.StringsManipulatorActionTypeDropColumn,
			SchemaName: schemaName,
		},
		Table:  tableName,
		Column: columnName,
	}
}

type StringsManipulatorActionModifyColumnType struct {
	base.StringsManipulatorActionBase
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

func NewModifyColumnTypeAction(schemaName, tableName, columnName, columnType string) *StringsManipulatorActionModifyColumnType {
	return &StringsManipulatorActionModifyColumnType{
		StringsManipulatorActionBase: base.StringsManipulatorActionBase{
			Type:       base.StringsManipulatorActionTypeModifyColumnType,
			SchemaName: schemaName,
		},
		Table:  tableName,
		Column: columnName,
		Type:   columnType,
	}
}

type StringsManipulatorActionAddColumn struct {
	base.StringsManipulatorActionBase
	Table            string
	ColumnDefinition string
}

func (s *StringsManipulatorActionAddColumn) GetTopLevelNaming() string {
	return s.Table
}

func (*StringsManipulatorActionAddColumn) GetSecondLevelNaming() string {
	return ""
}

func NewAddColumnAction(schemaName, tableName, columnDefinition string) *StringsManipulatorActionAddColumn {
	return &StringsManipulatorActionAddColumn{
		StringsManipulatorActionBase: base.StringsManipulatorActionBase{
			Type:       base.StringsManipulatorActionTypeAddColumn,
			SchemaName: schemaName,
		},
		Table:            tableName,
		ColumnDefinition: columnDefinition,
	}
}

type StringsManipulatorActionColumnOptionBase struct {
	base.StringsManipulatorActionBase
	Type base.ColumnOptionType
}

func (s *StringsManipulatorActionColumnOptionBase) GetOptionType() base.ColumnOptionType {
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

func NewDropColumnOptionAction(schemaName, tableName string, columnName string, option base.ColumnOptionType) *StringsManipulatorActionDropColumnOption {
	return &StringsManipulatorActionDropColumnOption{
		StringsManipulatorActionColumnOptionBase: StringsManipulatorActionColumnOptionBase{
			StringsManipulatorActionBase: base.StringsManipulatorActionBase{
				Type:       base.StringsManipulatorActionTypeDropColumnOption,
				SchemaName: schemaName,
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

func NewModifyColumnOptionAction(schemaName, tableName string, columnName string, oldOption base.ColumnOptionType, newOptionDefine string) *StringsManipulatorActionModifyColumnOption {
	return &StringsManipulatorActionModifyColumnOption{
		StringsManipulatorActionColumnOptionBase: StringsManipulatorActionColumnOptionBase{
			StringsManipulatorActionBase: base.StringsManipulatorActionBase{
				Type:       base.StringsManipulatorActionTypeModifyColumnOption,
				SchemaName: schemaName,
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

func NewAddColumnOptionAction(schemaName, tableName string, columnName string, optionType base.ColumnOptionType, newOptionDefine string) *StringsManipulatorActionAddColumnOption {
	return &StringsManipulatorActionAddColumnOption{
		StringsManipulatorActionColumnOptionBase: StringsManipulatorActionColumnOptionBase{
			StringsManipulatorActionBase: base.StringsManipulatorActionBase{
				Type:       base.StringsManipulatorActionTypeAddColumnOption,
				SchemaName: schemaName,
			},
			Type: optionType,
		},
		Table:           tableName,
		Column:          columnName,
		NewOptionDefine: newOptionDefine,
	}
}

type StringsManipulatorActionDropTableConstraint struct {
	base.StringsManipulatorActionBase
	Table          string
	Constraint     base.TableConstraintType
	ConstraintName string
}

func (s *StringsManipulatorActionDropTableConstraint) GetTopLevelNaming() string {
	return s.Table
}

func (s *StringsManipulatorActionDropTableConstraint) GetSecondLevelNaming() string {
	return s.ConstraintName
}

func NewDropTableConstraintAction(schemaName, tableName string, constraintName string) *StringsManipulatorActionDropTableConstraint {
	return &StringsManipulatorActionDropTableConstraint{
		StringsManipulatorActionBase: base.StringsManipulatorActionBase{
			Type:       base.StringsManipulatorActionTypeDropTableConstraint,
			SchemaName: schemaName,
		},
		Table:          tableName,
		ConstraintName: constraintName,
	}
}

type StringsManipulatorActionModifyTableConstraint struct {
	base.StringsManipulatorActionBase
	Table               string
	OldConstraint       base.TableConstraintType
	OldConstraintName   string
	NewConstraintDefine string
}

func (s *StringsManipulatorActionModifyTableConstraint) GetTopLevelNaming() string {
	return s.Table
}

func (s *StringsManipulatorActionModifyTableConstraint) GetSecondLevelNaming() string {
	return s.OldConstraintName
}

func NewModifyTableConstraintAction(schemaName, tableName string, oldConstraint base.TableConstraintType, oldConstraintName string, newConstraintDefine string) *StringsManipulatorActionModifyTableConstraint {
	return &StringsManipulatorActionModifyTableConstraint{
		StringsManipulatorActionBase: base.StringsManipulatorActionBase{
			Type:       base.StringsManipulatorActionTypeModifyTableConstraint,
			SchemaName: schemaName,
		},
		Table:               tableName,
		OldConstraint:       oldConstraint,
		OldConstraintName:   oldConstraintName,
		NewConstraintDefine: newConstraintDefine,
	}
}

type StringsManipulatorActionAddTableConstraint struct {
	base.StringsManipulatorActionBase
	Table               string
	Type                base.TableConstraintType
	NewConstraintDefine string
}

func (s *StringsManipulatorActionAddTableConstraint) GetTopLevelNaming() string {
	return s.Table
}

func (*StringsManipulatorActionAddTableConstraint) GetSecondLevelNaming() string {
	return ""
}

func NewAddTableConstraintAction(schemaName, tableName string, constraintType base.TableConstraintType, newConstraintDefine string) *StringsManipulatorActionAddTableConstraint {
	return &StringsManipulatorActionAddTableConstraint{
		StringsManipulatorActionBase: base.StringsManipulatorActionBase{
			Type:       base.StringsManipulatorActionTypeAddTableConstraint,
			SchemaName: schemaName,
		},
		Table:               tableName,
		Type:                constraintType,
		NewConstraintDefine: newConstraintDefine,
	}
}

type StringsManipulatorActionDropIndex struct {
	base.StringsManipulatorActionBase
	Table string
	Index string
}

func (s *StringsManipulatorActionDropIndex) GetTopLevelNaming() string {
	return s.Table
}

func (s *StringsManipulatorActionDropIndex) GetSecondLevelNaming() string {
	return s.Index
}

func NewDropIndexAction(schemaName, tableName, indexName string) *StringsManipulatorActionDropIndex {
	return &StringsManipulatorActionDropIndex{
		StringsManipulatorActionBase: base.StringsManipulatorActionBase{
			Type:       base.StringsManipulatorActionTypeDropIndex,
			SchemaName: schemaName,
		},
		Table: tableName,
		Index: indexName,
	}
}

type StringsManipulatorActionAddIndex struct {
	base.StringsManipulatorActionBase
	Table       string
	IndexDefine string
}

func (s *StringsManipulatorActionAddIndex) GetTopLevelNaming() string {
	return s.Table
}

func (*StringsManipulatorActionAddIndex) GetSecondLevelNaming() string {
	return ""
}

func NewAddIndexAction(schemaName, tableName, indexDefine string) *StringsManipulatorActionAddIndex {
	return &StringsManipulatorActionAddIndex{
		StringsManipulatorActionBase: base.StringsManipulatorActionBase{
			Type:       base.StringsManipulatorActionTypeAddIndex,
			SchemaName: schemaName,
		},
		Table:       tableName,
		IndexDefine: indexDefine,
	}
}

type StringsManipulatorActionModifyIndex struct {
	base.StringsManipulatorActionBase
	Table          string
	IndexName      string
	NewIndexDefine string
}

func (s *StringsManipulatorActionModifyIndex) GetTopLevelNaming() string {
	return s.Table
}

func (s *StringsManipulatorActionModifyIndex) GetSecondLevelNaming() string {
	return s.IndexName
}

func NewModifyIndexAction(schemaName, tableName, indexName string, newIndexDefine string) *StringsManipulatorActionModifyIndex {
	return &StringsManipulatorActionModifyIndex{
		StringsManipulatorActionBase: base.StringsManipulatorActionBase{
			Type:       base.StringsManipulatorActionTypeModifyIndex,
			SchemaName: schemaName,
		},
		Table:          tableName,
		IndexName:      indexName,
		NewIndexDefine: newIndexDefine,
	}
}

type tableID struct {
	schema string
	table  string
}

type indexID struct {
	schema string
	table  string
	index  string
}

func (s *StringsManipulator) Manipulate(actions ...base.StringsManipulatorAction) (string, error) {
	tableActions := make(map[tableID][]base.StringsManipulatorAction)
	var addTables []base.StringsManipulatorAction

	indexActions := make(map[indexID][]base.StringsManipulatorAction)

	for _, action := range actions {
		switch action := action.(type) {
		case *StringsManipulatorActionAddTable:
			addTables = append(addTables, action)
		case *StringsManipulatorActionDropIndex:
			indexID := indexID{
				schema: action.GetSchemaName(),
				table:  action.GetTopLevelNaming(),
				index:  action.GetSecondLevelNaming(),
			}
			indexActions[indexID] = append(indexActions[indexID], action)
		case *StringsManipulatorActionModifyIndex:
			indexID := indexID{
				schema: action.GetSchemaName(),
				table:  action.GetTopLevelNaming(),
				index:  action.GetSecondLevelNaming(),
			}
			indexActions[indexID] = append(indexActions[indexID], action)
		default:
			table := tableID{
				schema: action.GetSchemaName(),
				table:  action.GetTopLevelNaming(),
			}
			tableActions[table] = append(tableActions[table], action)
		}
	}

	listener := &statementListener{
		actions:      tableActions,
		indexActions: indexActions,
		addTables:    addTables,
	}
	antlr.ParseTreeWalkerDefault.Walk(listener, s.tree)
	return strings.Join(listener.results, "\n"), listener.err
}

type statementListener struct {
	*parser.BasePlSqlParserListener

	actions      map[tableID][]base.StringsManipulatorAction
	indexActions map[indexID][]base.StringsManipulatorAction
	addTables    []base.StringsManipulatorAction
	results      []string
	err          error
}

func (l *statementListener) EnterSql_script(ctx *parser.Sql_scriptContext) {
	for _, statement := range ctx.AllUnit_statement() {
		switch {
		case statement.Create_table() != nil:
			rewriter := &rewriter{
				actions:      l.actions,
				state:        statementStateRemaining,
				buf:          &strings.Builder{},
				tableActions: make(map[string][]base.StringsManipulatorAction),
			}
			antlr.ParseTreeWalkerDefault.Walk(rewriter, statement.Create_table())
			if rewriter.err != nil {
				l.err = rewriter.err
				return
			}
			l.results = append(l.results, rewriter.buf.String())
		case statement.Create_index() != nil:
			rewriter := &rewriter{
				actions:      l.actions,
				indexActions: l.indexActions,
				state:        statementStateRemaining,
				buf:          &strings.Builder{},
			}
			antlr.ParseTreeWalkerDefault.Walk(rewriter, statement.Create_index())
			if rewriter.err != nil {
				l.err = rewriter.err
				return
			}
			l.results = append(l.results, rewriter.buf.String())
		default:
			l.results = append(l.results, statement.GetParser().GetTokenStream().GetTextFromRuleContext(statement)+";\n")
		}
	}
	for _, action := range l.addTables {
		l.results = append(l.results, action.(*StringsManipulatorActionAddTable).TableDefinition)
		tableID := tableID{
			schema: action.GetSchemaName(),
			table:  action.GetTopLevelNaming(),
		}
		actions, ok := l.actions[tableID]
		if !ok || len(actions) == 0 {
			continue
		}
		for _, action := range actions {
			switch action := action.(type) {
			case *StringsManipulatorActionAddIndex:
				// Add an empty line before the index definition.
				l.results = append(l.results, "")
				l.results = append(l.results, action.IndexDefine)
			default:
				// Do nothing.
			}
		}
	}
}

type statementState int

const (
	statementStateRemaining statementState = iota
	statementStateModify
	statementStateDelete
)

type rewriter struct {
	*parser.BasePlSqlParserListener

	actions      map[tableID][]base.StringsManipulatorAction
	indexActions map[indexID][]base.StringsManipulatorAction
	buf          *strings.Builder
	err          error

	// per-statement state
	currentTableID    *tableID
	tableActions      map[string][]base.StringsManipulatorAction
	state             statementState
	columnDefines     []string
	constraintDefines []string

	newIndexDefine string
}

func (r *rewriter) EnterCreate_index(ctx *parser.Create_indexContext) {
	if r.err != nil {
		return
	}

	if ctx.Table_index_clause() == nil {
		return
	}

	_, indexName := NormalizeIndexName(ctx.Index_name())

	indexDefinition := ctx.Table_index_clause()
	schema, table := NormalizeTableViewName("", indexDefinition.Tableview_name())
	indexID := indexID{
		schema: schema,
		table:  table,
		index:  indexName,
	}

	actions, ok := r.indexActions[indexID]
	if !ok || len(actions) == 0 {
		return
	}

	for _, action := range actions {
		switch action := action.(type) {
		case *StringsManipulatorActionDropIndex:
			r.state = statementStateDelete
		case *StringsManipulatorActionModifyIndex:
			r.state = statementStateModify
			r.newIndexDefine = action.NewIndexDefine
		}
	}
}

func (r *rewriter) ExitCreate_index(ctx *parser.Create_indexContext) {
	if r.err != nil {
		return
	}

	switch r.state {
	case statementStateDelete:
		return
	case statementStateRemaining:
		if _, err := r.buf.WriteString(ctx.GetParser().GetTokenStream().GetTextFromRuleContext(ctx)); err != nil {
			r.err = err
			return
		}
	case statementStateModify:
		if _, err := r.buf.WriteString(r.newIndexDefine); err != nil {
			r.err = err
			return
		}
	}
}

func (r *rewriter) EnterCreate_table(ctx *parser.Create_tableContext) {
	if r.err != nil {
		return
	}

	if ctx.Relational_table() == nil {
		return
	}

	schema := ""
	if ctx.Schema_name() != nil {
		schema = NormalizeSchemaName(ctx.Schema_name())
	}
	table := NormalizeTableName(ctx.Table_name())
	r.currentTableID = &tableID{
		schema: schema,
		table:  table,
	}
	actions, ok := r.actions[*r.currentTableID]
	if !ok || len(actions) == 0 {
		r.currentTableID = nil
		return
	}
	r.state = statementStateModify

	hasDropTable := false
	for _, action := range actions {
		// do copy
		action := action
		secondName := action.GetSecondLevelNaming()
		r.tableActions[secondName] = append(r.tableActions[secondName], action)
		if action.GetType() == base.StringsManipulatorActionTypeDropTable {
			hasDropTable = true
		}
	}

	if hasDropTable {
		r.state = statementStateDelete
		return
	}
}

func (r *rewriter) ExitCreate_table(ctx *parser.Create_tableContext) {
	if r.err != nil {
		return
	}

	addIndexFunc := func() error {
		for _, action := range r.tableActions[""] {
			if action.GetType() == base.StringsManipulatorActionTypeAddIndex {
				addIndex, ok := action.(*StringsManipulatorActionAddIndex)
				if !ok {
					return errors.New("unexpected action type")
				}
				if _, err := r.buf.WriteString("\n"); err != nil {
					return err
				}
				if _, err := r.buf.WriteString(addIndex.IndexDefine); err != nil {
					return err
				}
			}
		}
		return nil
	}

	switch r.state {
	case statementStateDelete:
		return
	case statementStateRemaining:
		if _, err := r.buf.WriteString(ctx.GetParser().GetTokenStream().GetTextFromRuleContext(ctx)); err != nil {
			r.err = err
			return
		}
		// Keep the original format with dump.
		if _, err := r.buf.WriteString("\n"); err != nil {
			r.err = err
			return
		}
		if err := addIndexFunc(); err != nil {
			r.err = err
		}
		return
	case statementStateModify:
		actions, exists := r.tableActions[""]
		if exists && len(actions) > 0 {
			for _, action := range actions {
				switch action := action.(type) {
				case *StringsManipulatorActionAddColumn:
					r.columnDefines = append(r.columnDefines, action.ColumnDefinition)
				case *StringsManipulatorActionAddTableConstraint:
					r.constraintDefines = append(r.constraintDefines, action.NewConstraintDefine)
				}
			}
		}

		if err := r.assembleCreateTable(ctx); err != nil {
			r.err = err
			return
		}

		if err := addIndexFunc(); err != nil {
			r.err = err
			return
		}
	}
}

func (r *rewriter) assembleCreateTable(ctx *parser.Create_tableContext) error {
	// Write the prefix part.
	if _, err := r.buf.WriteString(ctx.GetParser().GetTokenStream().GetTextFromInterval(antlr.Interval{
		Start: ctx.GetStart().GetTokenIndex(),
		Stop:  ctx.Relational_table().GetStart().GetTokenIndex() - 1,
	})); err != nil {
		return errors.Wrap(err, "failed to write string")
	}
	if _, err := r.buf.WriteString(" ("); err != nil {
		return errors.Wrap(err, "failed to write string")
	}
	if len(r.columnDefines) > 0 {
		if _, err := r.buf.WriteString("\n  "); err != nil {
			return errors.Wrap(err, "failed to write string")
		}
		if _, err := r.buf.WriteString(strings.Join(r.columnDefines, ",\n  ")); err != nil {
			return errors.Wrap(err, "failed to write string")
		}
	}
	if len(r.constraintDefines) > 0 {
		if len(r.columnDefines) > 0 {
			if _, err := r.buf.WriteString(",\n  "); err != nil {
				return errors.Wrap(err, "failed to write string")
			}
		}
		if _, err := r.buf.WriteString(strings.Join(r.constraintDefines, ",\n  ")); err != nil {
			return errors.Wrap(err, "failed to write string")
		}
	}
	if _, err := r.buf.WriteString("\n)"); err != nil {
		return errors.Wrap(err, "failed to write string")
	}
	// Write the suffix part.
	start := ctx.Relational_table().RIGHT_PAREN().GetSymbol().GetTokenIndex() + 1
	if _, err := r.buf.WriteString(ctx.GetParser().GetTokenStream().GetTextFromInterval(antlr.Interval{
		Start: start,
		Stop:  ctx.GetStop().GetTokenIndex(),
	})); err != nil {
		return errors.Wrap(err, "failed to write string")
	}
	// Keep the original format with dump.
	if _, err := r.buf.WriteString("\n"); err != nil {
		return errors.Wrap(err, "failed to write string")
	}
	return nil
}

func (r *rewriter) EnterOut_of_line_constraint(ctx *parser.Out_of_line_constraintContext) {
	if r.err != nil {
		return
	}

	if r.state == statementStateDelete || r.currentTableID == nil {
		return
	}

	if ctx.Constraint_name() == nil {
		// Remaining constraints are not supported.
		r.constraintDefines = append(r.constraintDefines, ctx.GetParser().GetTokenStream().GetTextFromRuleContext(ctx))
		return
	}

	switch {
	case ctx.UNIQUE() != nil, ctx.PRIMARY() != nil:
	default:
		// Remaining constraints are not supported.
		r.constraintDefines = append(r.constraintDefines, ctx.GetParser().GetTokenStream().GetTextFromRuleContext(ctx))
		return
	}

	_, constraintName := NormalizeConstraintName(ctx.Constraint_name())
	var constraintActions []base.StringsManipulatorAction
	if actions, exists := r.tableActions[constraintName]; exists {
		for _, action := range actions {
			switch action := action.(type) {
			case *StringsManipulatorActionDropTableConstraint, *StringsManipulatorActionModifyTableConstraint:
				constraintActions = append(constraintActions, action)
			}
		}
	}
	if len(constraintActions) == 0 {
		// We need to remain the original constraint definition if no action is found.
		r.constraintDefines = append(r.constraintDefines, ctx.GetParser().GetTokenStream().GetTextFromRuleContext(ctx))
		return
	}
	for _, action := range constraintActions {
		switch action := action.(type) {
		case *StringsManipulatorActionDropTableConstraint:
			// Drop constraint.
			return
		case *StringsManipulatorActionModifyTableConstraint:
			r.constraintDefines = append(r.constraintDefines, action.NewConstraintDefine)
		}
	}
}

func (r *rewriter) EnterColumn_definition(ctx *parser.Column_definitionContext) {
	if r.err != nil {
		return
	}

	if r.state == statementStateDelete || r.currentTableID == nil {
		return
	}

	_, _, columnName := NormalizeColumnName(ctx.Column_name())
	if columnName == "" {
		r.err = errors.New("invalid column name")
		return
	}

	var columnActions []base.StringsManipulatorAction
	if actions, exists := r.tableActions[columnName]; exists {
		for _, action := range actions {
			switch action := action.(type) {
			case *StringsManipulatorActionDropColumn,
				*StringsManipulatorActionModifyColumnType,
				*StringsManipulatorActionAddColumnOption,
				*StringsManipulatorActionDropColumnOption,
				*StringsManipulatorActionModifyColumnOption:
				columnActions = append(columnActions, action)
			}
		}
	}
	if len(columnActions) == 0 {
		// We need to remain the original column definition if no action is found.
		r.columnDefines = append(r.columnDefines, ctx.GetParser().GetTokenStream().GetTextFromRuleContext(ctx))
		return
	}
	actionsMap := make(map[base.StringsManipulatorActionType][]base.StringsManipulatorAction)
	for _, action := range columnActions {
		if action.GetType() == base.StringsManipulatorActionTypeDropColumn {
			return
		}
		actionsMap[action.GetType()] = append(actionsMap[action.GetType()], action)
	}

	buf := strings.Builder{}
	if _, err := buf.WriteString(ctx.GetParser().GetTokenStream().GetTextFromRuleContext(ctx.Column_name())); err != nil {
		r.err = errors.Wrap(err, "failed to write column name")
		return
	}
	if _, err := buf.WriteString(" "); err != nil {
		r.err = errors.Wrap(err, "failed to write string")
		return
	}

	if modifyType, exists := actionsMap[base.StringsManipulatorActionTypeModifyColumnType]; exists {
		if len(modifyType) != 1 {
			r.err = errors.New("multiple modify type actions found")
			return
		}
		modifyType, ok := modifyType[0].(*StringsManipulatorActionModifyColumnType)
		if !ok {
			r.err = errors.New("invalid modify type action")
			return
		}
		if _, err := buf.WriteString(modifyType.Type); err != nil {
			r.err = errors.Wrap(err, "failed to write string")
			return
		}
	} else {
		if ctx.Datatype() != nil {
			if _, err := buf.WriteString(ctx.GetParser().GetTokenStream().GetTextFromRuleContext(ctx.Datatype())); err != nil {
				r.err = errors.Wrap(err, "failed to write string")
				return
			}
		} else if ctx.Regular_id() != nil {
			if _, err := buf.WriteString(ctx.GetParser().GetTokenStream().GetTextFromRuleContext(ctx.Regular_id())); err != nil {
				r.err = errors.Wrap(err, "failed to write string")
				return
			}
		}

		if ctx.COLLATE() != nil {
			if err := buf.WriteByte(' '); err != nil {
				r.err = errors.Wrap(err, "failed to write string")
				return
			}
			if _, err := buf.WriteString(ctx.GetParser().GetTokenStream().GetTextFromInterval(antlr.Interval{
				Start: ctx.COLLATE().GetSymbol().GetTokenIndex(),
				Stop:  ctx.Column_collation_name().GetStop().GetTokenIndex(),
			})); err != nil {
				r.err = errors.Wrap(err, "failed to write string")
				return
			}
		}
	}

	if ctx.VISIBLE() != nil {
		if _, err := buf.WriteString(" VISIBLE"); err != nil {
			r.err = errors.Wrap(err, "failed to write string")
			return
		}
	} else if ctx.INVISIBLE() != nil {
		if _, err := buf.WriteString(" INVISIBLE"); err != nil {
			r.err = errors.Wrap(err, "failed to write string")
			return
		}
	}

	optionActionMap := make(map[base.ColumnOptionType]base.StringsManipulatorAction)
	for _, action := range actionsMap[base.StringsManipulatorActionTypeAddColumnOption] {
		action, ok := action.(*StringsManipulatorActionAddColumnOption)
		if !ok {
			r.err = errors.New("invalid add column option action")
			return
		}
		optionActionMap[action.GetOptionType()] = action
	}
	for _, action := range actionsMap[base.StringsManipulatorActionTypeDropColumnOption] {
		action, ok := action.(*StringsManipulatorActionDropColumnOption)
		if !ok {
			r.err = errors.New("invalid drop column option action")
			return
		}
		optionActionMap[action.GetOptionType()] = action
	}
	for _, action := range actionsMap[base.StringsManipulatorActionTypeModifyColumnOption] {
		action, ok := action.(*StringsManipulatorActionModifyColumnOption)
		if !ok {
			r.err = errors.New("invalid modify column option action")
			return
		}
		optionActionMap[action.GetOptionType()] = action
	}

	// Generate column default.
	if err := writeDefaultOption(&buf, ctx, optionActionMap); err != nil {
		r.err = err
		return
	}

	for _, constraint := range ctx.AllInline_constraint() {
		// Cause we only write NOT NULL in Oracle dump, so we don't need to handle NULL.
		if constraint.NULL_() != nil && constraint.NOT() != nil {
			if err := writeNotNullOption(&buf, ctx.GetParser().GetTokenStream().GetTextFromInterval(antlr.Interval{
				Start: constraint.NOT().GetSymbol().GetTokenIndex(),
				Stop:  constraint.NULL_().GetSymbol().GetTokenIndex(),
			}), optionActionMap); err != nil {
				r.err = err
				return
			}
		}
	}

	r.columnDefines = append(r.columnDefines, buf.String())
}

func writeNotNullOption(buf *strings.Builder, origin string, optionActionMap map[base.ColumnOptionType]base.StringsManipulatorAction) error {
	needOrigin := true
	if action, exists := optionActionMap[base.ColumnOptionTypeNotNull]; exists {
		switch action := action.(type) {
		case *StringsManipulatorActionDropColumnOption:
			// Drop not null.
			needOrigin = false
		case *StringsManipulatorActionModifyColumnOption:
			needOrigin = false
			if err := buf.WriteByte(' '); err != nil {
				return errors.Wrap(err, "failed to write string")
			}
			if _, err := buf.WriteString(action.NewOptionDefine); err != nil {
				return errors.Wrap(err, "failed to write string")
			}
		case *StringsManipulatorActionAddColumnOption:
			needOrigin = false
			if err := buf.WriteByte(' '); err != nil {
				return errors.Wrap(err, "failed to write string")
			}
			if _, err := buf.WriteString(action.NewOptionDefine); err != nil {
				return errors.Wrap(err, "failed to write string")
			}
		}
	}
	if needOrigin {
		if err := buf.WriteByte(' '); err != nil {
			return errors.Wrap(err, "failed to write string")
		}
		if _, err := buf.WriteString(origin); err != nil {
			return errors.Wrap(err, "failed to write string")
		}
	}
	return nil
}

func writeDefaultOption(buf *strings.Builder, ctx *parser.Column_definitionContext, optionActionMap map[base.ColumnOptionType]base.StringsManipulatorAction) error {
	needOrigin := true
	if action, exists := optionActionMap[base.ColumnOptionTypeDefault]; exists {
		switch action := action.(type) {
		case *StringsManipulatorActionDropColumnOption:
			// Drop default.
			needOrigin = false
		case *StringsManipulatorActionModifyColumnOption:
			needOrigin = false
			if err := buf.WriteByte(' '); err != nil {
				return errors.Wrap(err, "failed to write string")
			}
			if _, err := buf.WriteString(action.NewOptionDefine); err != nil {
				return errors.Wrap(err, "failed to write string")
			}
		case *StringsManipulatorActionAddColumnOption:
			needOrigin = false
			if err := buf.WriteByte(' '); err != nil {
				return errors.Wrap(err, "failed to write string")
			}
			if _, err := buf.WriteString(action.NewOptionDefine); err != nil {
				return errors.Wrap(err, "failed to write string")
			}
		}
	}
	if ctx.DEFAULT() != nil && needOrigin {
		if err := buf.WriteByte(' '); err != nil {
			return errors.Wrap(err, "failed to write string")
		}
		if ctx.Expression() != nil {
			if _, err := buf.WriteString(ctx.GetParser().GetTokenStream().GetTextFromInterval(antlr.Interval{
				Start: ctx.DEFAULT().GetSymbol().GetTokenIndex(),
				Stop:  ctx.Expression().GetStop().GetTokenIndex(),
			})); err != nil {
				return errors.Wrap(err, "failed to write string")
			}
		} else if ctx.Identity_clause() != nil {
			if _, err := buf.WriteString(ctx.GetParser().GetTokenStream().GetTextFromInterval(antlr.Interval{
				Start: ctx.DEFAULT().GetSymbol().GetTokenIndex(),
				Stop:  ctx.Identity_clause().GetStop().GetTokenIndex(),
			})); err != nil {
				return errors.Wrap(err, "failed to write string")
			}
		}
	}
	return nil
}
