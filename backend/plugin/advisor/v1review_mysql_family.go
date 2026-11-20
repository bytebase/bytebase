package advisor

import (
	"cmp"
	"context"
	"fmt"
	"slices"
	"strings"

	metadatapb "github.com/bytebase/omni/metadata"
	"github.com/bytebase/omni/v1review"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/store/model"
)

// MySQLFamilyOptions configures a MySQL-family rule builder for one engine.
type MySQLFamilyOptions[N any] struct {
	Extract        func(Context) ([]V1Stmt[N], error)
	ByteOffsetLine func(sql string, byteOffset int) int32
	// TiDBQuirks preserves V1 position, text-trimming, and message differences
	// between the TiDB adapter lineage and the MySQL/MariaDB/OceanBase lineage.
	TiDBQuirks bool

	// Optional AST node predicates for V1 panic parity and position selection.
	DefinesColumn       func(N) bool
	ReadsColumnType     func(N) bool
	DropsColumn         func(N) bool
	IsRenameIndex       func(N) bool
	IsDropDatabase      func(N) bool
	IsCreateTable       func(N) bool
	IsAlterTable        func(N) bool
	CreateTableLocStart func(N) (int, bool)
}

// MySQLFamily builds Advisor implementations for MySQL-family v1review rules.
type MySQLFamily[N any] struct {
	r   v1Runner[N]
	opt MySQLFamilyOptions[N]
}

// NewMySQLFamily returns a rule builder for the given engine options.
func NewMySQLFamily[N any](opt MySQLFamilyOptions[N]) *MySQLFamily[N] {
	return &MySQLFamily[N]{
		r: v1Runner[N]{
			extract:             opt.Extract,
			byteOffsetLine:      opt.ByteOffsetLine,
			tidbQuirks:          opt.TiDBQuirks,
			isAlterTable:        opt.IsAlterTable,
			createTableLocStart: opt.CreateTableLocStart,
		},
		opt: opt,
	}
}

type preparedStmts[N any] struct {
	stmts  []V1Stmt[N]
	nodes  []N
	status storepb.Advice_Status
	title  string
}

func (m *MySQLFamily[N]) prepare(checkCtx Context, withNodes bool) (preparedStmts[N], error) {
	var stmts []V1Stmt[N]
	var nodes []N
	var err error
	if withNodes {
		stmts, nodes, err = m.r.extractNodes(checkCtx)
	} else {
		stmts, err = m.r.extract(checkCtx)
	}
	if err != nil {
		return preparedStmts[N]{}, err
	}
	level, err := NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
	if err != nil {
		return preparedStmts[N]{}, err
	}
	return preparedStmts[N]{stmts: stmts, nodes: nodes, status: level, title: checkCtx.Rule.Type.String()}, nil
}

func newAdvice(status storepb.Advice_Status, c code.Code, title, content string, pos *storepb.Position) *storepb.Advice {
	return &storepb.Advice{Status: status, Code: c.Int32(), Title: title, Content: content, StartPosition: pos}
}

func (m *MySQLFamily[N]) perStmt(
	fn func([]N) []v1review.Finding,
	build func(V1Stmt[N], v1review.Finding, storepb.Advice_Status, string) *storepb.Advice,
) Func {
	return func(_ context.Context, checkCtx Context) ([]*storepb.Advice, error) {
		p, err := m.prepare(checkCtx, false)
		if err != nil {
			return nil, err
		}
		var adviceList []*storepb.Advice
		for _, stmt := range p.stmts {
			for _, f := range fn([]N{stmt.Node}) {
				if a := build(stmt, f, p.status, p.title); a != nil {
					adviceList = append(adviceList, a)
				}
			}
		}
		return adviceList, nil
	}
}

func (m *MySQLFamily[N]) simpleStmt(
	fn func([]N) []v1review.Finding,
	adviceCode code.Code,
	posFn func(V1Stmt[N], v1review.Finding) *storepb.Position,
	contentFn func(V1Stmt[N], v1review.Finding) string,
) Func {
	return m.perStmt(fn, func(stmt V1Stmt[N], f v1review.Finding, status storepb.Advice_Status, title string) *storepb.Advice {
		return newAdvice(status, adviceCode, title, contentFn(stmt, f), posFn(stmt, f))
	})
}

func (m *MySQLFamily[N]) batch(
	fn func([]N) []v1review.Finding,
	build func([]V1Stmt[N], v1review.Finding, storepb.Advice_Status, string) *storepb.Advice,
) Func {
	return func(_ context.Context, checkCtx Context) ([]*storepb.Advice, error) {
		p, err := m.prepare(checkCtx, true)
		if err != nil {
			return nil, err
		}
		var adviceList []*storepb.Advice
		for _, f := range fn(p.nodes) {
			if a := build(p.stmts, f, p.status, p.title); a != nil {
				adviceList = append(adviceList, a)
			}
		}
		return adviceList, nil
	}
}

func (m *MySQLFamily[N]) numberStmtRule(
	fn func([]N, int) []v1review.Finding,
	missingMsg string,
	positiveOnly bool,
	build func(V1Stmt[N], v1review.Finding, int, storepb.Advice_Status, string) *storepb.Advice,
) Func {
	return func(_ context.Context, checkCtx Context) ([]*storepb.Advice, error) {
		p, err := m.prepare(checkCtx, false)
		if err != nil {
			return nil, err
		}
		limit, err := requireNumberPayload(checkCtx.Rule, missingMsg)
		if err != nil {
			return nil, err
		}
		if positiveOnly && limit <= 0 {
			return nil, nil
		}
		var adviceList []*storepb.Advice
		for _, stmt := range p.stmts {
			for _, f := range fn([]N{stmt.Node}, limit) {
				adviceList = append(adviceList, build(stmt, f, limit, p.status, p.title))
			}
		}
		return adviceList, nil
	}
}

func (m *MySQLFamily[N]) stringArrayStmtRule(
	fn func([]N, []string) []v1review.Finding,
	adviceCode code.Code,
	posFn func(V1Stmt[N], v1review.Finding) *storepb.Position,
	contentFn func(V1Stmt[N], v1review.Finding) string,
) Func {
	return func(_ context.Context, checkCtx Context) ([]*storepb.Advice, error) {
		p, err := m.prepare(checkCtx, false)
		if err != nil {
			return nil, err
		}
		list := checkCtx.Rule.GetStringArrayPayload().List
		var adviceList []*storepb.Advice
		for _, stmt := range p.stmts {
			for _, f := range fn([]N{stmt.Node}, list) {
				adviceList = append(adviceList, newAdvice(p.status, adviceCode, p.title, contentFn(stmt, f), posFn(stmt, f)))
			}
		}
		return adviceList, nil
	}
}

func (m *MySQLFamily[N]) disallowCreateRule(fn func([]N) []v1review.Finding, c code.Code, object string) Func {
	return m.simpleStmt(fn, c, m.r.posBaseLineTwice, func(stmt V1Stmt[N], _ v1review.Finding) string {
		return fmt.Sprintf("%s is forbidden, but \"%s\" creates", object, strings.TrimSpace(stmt.Text))
	})
}

func (m *MySQLFamily[N]) indexSchemaBatchRule(
	fn func([]N, v1review.Schema) ([]v1review.Finding, error),
	adviceCode code.Code,
	contentFn func(v1review.Finding) string,
) Func {
	return func(_ context.Context, checkCtx Context) ([]*storepb.Advice, error) {
		p, err := m.prepare(checkCtx, true)
		if err != nil {
			return nil, err
		}
		findings, err := fn(p.nodes, v1SchemaOf(checkCtx.OriginalMetadata))
		if err != nil {
			readMissingSchema(checkCtx.OriginalMetadata, err)
			return nil, err
		}
		var adviceList []*storepb.Advice
		for _, f := range findings {
			adviceList = append(adviceList, newAdvice(p.status, adviceCode, p.title, contentFn(f), m.r.posFinding(p.stmts[f.Statement], f)))
		}
		return adviceList, nil
	}
}

// MaxStatementsInTransaction validates the rule level and number payload and
// reports no advice, preserving the MySQL no-op rule behavior.
func (*MySQLFamily[N]) MaxStatementsInTransaction() Func {
	return func(_ context.Context, checkCtx Context) ([]*storepb.Advice, error) {
		if _, err := NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level); err != nil {
			return nil, err
		}
		if _, err := requireNumberPayload(checkCtx.Rule, "number_payload is required for this rule"); err != nil {
			return nil, err
		}
		return nil, nil
	}
}

// OnlineMigration returns an Func for ADVICE_ONLINE_MIGRATION.
func (m *MySQLFamily[N]) OnlineMigration(engine storepb.Engine, fn func([]N, v1review.Schema, string, int64) []v1review.Finding) Func {
	return func(ctx context.Context, checkCtx Context) ([]*storepb.Advice, error) {
		n, err := requireNumberPayload(checkCtx.Rule, "number_payload is required for this rule")
		if err != nil {
			return nil, err
		}
		minRows := int64(n)
		level, err := NewStatusBySQLReviewRuleLevel(checkCtx.Rule.Level)
		if err != nil {
			return nil, err
		}
		title := checkCtx.Rule.Type.String()
		if checkCtx.EnableGhost {
			ghostDB := common.BackupDatabaseNameOfEngine(engine)
			if !DatabaseExists(ctx, checkCtx, ghostDB) {
				return []*storepb.Advice{newAdvice(level, code.DatabaseNotExists, title, fmt.Sprintf("Needs database %q to save temporary data for online migration but it does not exist", ghostDB), nil)}, nil
			}
		}
		stmts, nodes, err := m.r.extractNodes(checkCtx)
		if err != nil {
			return nil, err
		}
		schema := v1review.Schema{Metadata: checkCtx.DBSchema, ObjectCaseSensitive: checkCtx.IsObjectCaseSensitive}
		var adviceList []*storepb.Advice
		for _, f := range fn(nodes, schema, checkCtx.CurrentDatabase, minRows) {
			stmt := stmts[f.Statement]
			adviceList = append(adviceList, &storepb.Advice{
				Status:        level,
				Code:          code.AdviseOnlineMigrationForStatement.Int32(),
				Title:         title,
				Content:       fmt.Sprintf("Estimated table row count of %q is %s exceeding the set value %d. Consider using online migration for this statement", f.Table, f.Value, minRows),
				StartPosition: base.ConvertANTLRLineToPosition(stmt.BaseLine + int(m.r.findingLine(stmt.Text, f))),
				EndPosition:   base.ConvertANTLRLineToPosition(stmt.BaseLine + int(m.r.contentEndLine(stmt.Text))),
			})
		}
		if len(adviceList) > 1 || (len(adviceList) == 1 && len(checkCtx.ParsedStatements) > 1) {
			return adviceList, nil
		}
		if len(adviceList) == 1 && len(checkCtx.ParsedStatements) == 1 {
			if checkCtx.EnableGhost {
				return nil, nil
			}
			adviceList[0].Code = code.AdviseOnlineMigration.Int32()
			return adviceList, nil
		}
		if len(adviceList) == 0 && checkCtx.EnableGhost {
			return []*storepb.Advice{newAdvice(level, code.AdviseNoOnlineMigration, title, "Advise to disable online migration because found no statements that need online migration", nil)}, nil
		}
		return nil, nil
	}
}

// ColumnAutoIncrementInitialValue builds COLUMN_AUTO_INCREMENT_INITIAL_VALUE.
func (m *MySQLFamily[N]) ColumnAutoIncrementInitialValue(fn func([]N, int) []v1review.Finding) Func {
	return m.numberStmtRule(fn, "number_payload is required for column auto increment initial value rule", false,
		func(stmt V1Stmt[N], f v1review.Finding, value int, status storepb.Advice_Status, title string) *storepb.Advice {
			return newAdvice(status, code.AutoIncrementColumnInitialValueNotMatch, title, fmt.Sprintf("The initial auto-increment value in table `%s` is %s, which doesn't equal %v", f.Table, f.Value, value), m.r.posAbsOrFinding(stmt, f))
		})
}

// ColumnAutoIncrementMustInteger builds COLUMN_AUTO_INCREMENT_MUST_INTEGER.
func (m *MySQLFamily[N]) ColumnAutoIncrementMustInteger(fn func([]N) []v1review.Finding) Func {
	return m.simpleStmt(fn, code.AutoIncrementColumnNotInteger, m.r.posAbsOrFinding, func(_ V1Stmt[N], f v1review.Finding) string {
		return fmt.Sprintf("Auto-increment column `%s`.`%s` requires integer type", f.Table, f.Name)
	})
}

// ColumnAutoIncrementMustUnsigned builds COLUMN_AUTO_INCREMENT_MUST_UNSIGNED.
func (m *MySQLFamily[N]) ColumnAutoIncrementMustUnsigned(fn func([]N) []v1review.Finding) Func {
	return m.simpleStmt(fn, code.AutoIncrementColumnSigned, m.r.posAbsOrFinding, func(_ V1Stmt[N], f v1review.Finding) string {
		return fmt.Sprintf("Auto-increment column `%s`.`%s` is not UNSIGNED type", f.Table, f.Name)
	})
}

// ColumnComment builds COLUMN_COMMENT.
func (m *MySQLFamily[N]) ColumnComment(fn func([]N, bool, int) []v1review.Finding) Func {
	return func(_ context.Context, checkCtx Context) ([]*storepb.Advice, error) {
		p, err := m.prepare(checkCtx, false)
		if err != nil {
			return nil, err
		}
		payload := checkCtx.Rule.GetCommentConventionPayload()
		required, maxLength := true, 0
		if m.r.tidbQuirks {
			maxLength = -1
		}
		if payload != nil {
			required, maxLength = payload.Required, int(payload.MaxLength)
		}
		var adviceList []*storepb.Advice
		for _, stmt := range p.stmts {
			for _, f := range fn([]N{stmt.Node}, required, maxLength) {
				var adviceCode code.Code
				var content string
				switch f.Code {
				case v1review.ColumnCommentTooLong:
					if m.r.tidbQuirks && payload.MaxLength < 0 {
						continue
					}
					adviceCode = code.CommentTooLong
					content = fmt.Sprintf("The length of column `%s`.`%s` comment should be within %d characters", f.Table, f.Name, payload.MaxLength)
				case v1review.ColumnCommentMissing:
					adviceCode = code.CommentEmpty
					content = fmt.Sprintf("Column `%s`.`%s` requires comments", f.Table, f.Name)
				default:
					continue
				}
				adviceList = append(adviceList, newAdvice(p.status, adviceCode, p.title, content, m.r.posAlterFirstToken(stmt, f)))
			}
		}
		return adviceList, nil
	}
}

// ColumnCurrentTimeCountLimit builds COLUMN_CURRENT_TIME_COUNT_LIMIT.
func (m *MySQLFamily[N]) ColumnCurrentTimeCountLimit(fn func([]N) []v1review.Finding) Func {
	return m.batch(fn, func(stmts []V1Stmt[N], f v1review.Finding, status storepb.Advice_Status, title string) *storepb.Advice {
		switch f.Code {
		case v1review.ColumnDefaultCurrentTimeCount:
			return newAdvice(status, code.DefaultCurrentTimeColumnCountExceedsLimit, title, fmt.Sprintf("Table `%s` has %s DEFAULT CURRENT_TIMESTAMP() columns. The count greater than %d.", f.Table, f.Value, 2), m.r.posAbsOrFinding(stmts[f.Statement], f))
		case v1review.ColumnOnUpdateCurrentTimeCount:
			return newAdvice(status, code.OnUpdateCurrentTimeColumnCountExceedsLimit, title, fmt.Sprintf("Table `%s` has %s ON UPDATE CURRENT_TIMESTAMP() columns. The count greater than %d.", f.Table, f.Value, 1), m.r.posAbsOrFinding(stmts[f.Statement], f))
		default:
			return nil
		}
	})
}

// ColumnDisallowChange builds COLUMN_DISALLOW_CHANGE.
func (m *MySQLFamily[N]) ColumnDisallowChange(fn func([]N) []v1review.Finding) Func {
	return m.simpleStmt(fn, code.UseChangeColumnStatement, m.r.posStatementAbsOrStmt, func(stmt V1Stmt[N], _ v1review.Finding) string {
		return fmt.Sprintf("\"%s\" contains CHANGE COLUMN statement", m.r.stmtText(stmt, false))
	})
}

// ColumnDisallowChangingOrder builds COLUMN_DISALLOW_CHANGING_ORDER.
func (m *MySQLFamily[N]) ColumnDisallowChangingOrder(fn func([]N) []v1review.Finding) Func {
	return m.simpleStmt(fn, code.ChangeColumnOrder, m.r.posStatementAbsOrStmt, func(stmt V1Stmt[N], _ v1review.Finding) string {
		return fmt.Sprintf("\"%s\" changes column order", m.r.stmtText(stmt, false))
	})
}

// ColumnDisallowChangeType builds COLUMN_DISALLOW_CHANGE_TYPE.
func (m *MySQLFamily[N]) ColumnDisallowChangeType(fn func([]N, v1review.Schema) []v1review.Finding) Func {
	return func(_ context.Context, checkCtx Context) ([]*storepb.Advice, error) {
		p, err := m.prepare(checkCtx, true)
		if err != nil {
			return nil, err
		}
		if checkCtx.OriginalMetadata == nil && m.opt.ReadsColumnType != nil && slices.ContainsFunc(p.nodes, m.opt.ReadsColumnType) {
			checkCtx.OriginalMetadata.GetSchemaMetadata("")
		}
		schema := v1SchemaOf(checkCtx.OriginalMetadata)
		var adviceList []*storepb.Advice
		for _, stmt := range p.stmts {
			for _, f := range fn([]N{stmt.Node}, schema) {
				adviceList = append(adviceList, newAdvice(p.status, code.ChangeColumnType, p.title, fmt.Sprintf("\"%s\" changes column type", m.r.stmtText(stmt, false)), m.r.posAbsOrFinding(stmt, f)))
			}
		}
		return adviceList, nil
	}
}

// ColumnDisallowDrop builds COLUMN_DISALLOW_DROP.
func (m *MySQLFamily[N]) ColumnDisallowDrop(fn func([]N) []v1review.Finding) Func {
	return m.simpleStmt(fn, code.DropColumn, m.r.posAbsOrFinding, func(_ V1Stmt[N], f v1review.Finding) string {
		return fmt.Sprintf("drops column \"%s\" of table \"%s\"", f.Name, f.Table)
	})
}

// ColumnDisallowDropInIndex builds COLUMN_DISALLOW_DROP_IN_INDEX.
func (m *MySQLFamily[N]) ColumnDisallowDropInIndex(fn func([]N, v1review.Schema) []v1review.Finding) Func {
	return func(_ context.Context, checkCtx Context) ([]*storepb.Advice, error) {
		p, err := m.prepare(checkCtx, true)
		if err != nil {
			return nil, err
		}
		if checkCtx.OriginalMetadata == nil && m.opt.DropsColumn != nil && slices.ContainsFunc(p.nodes, m.opt.DropsColumn) {
			checkCtx.OriginalMetadata.GetSchemaMetadata("")
		}
		var adviceList []*storepb.Advice
		for _, f := range fn(p.nodes, v1SchemaOf(checkCtx.OriginalMetadata)) {
			stmt := p.stmts[f.Statement]
			pos := m.r.posFinding(stmt, f)
			if m.r.tidbQuirks {
				pos = m.r.posStatement(stmt)
			}
			adviceList = append(adviceList, newAdvice(p.status, code.DropIndexColumn, p.title, fmt.Sprintf("`%s`.`%s` cannot drop index column", f.Table, f.Name), pos))
		}
		return adviceList, nil
	}
}

// ColumnDisallowSetCharset builds COLUMN_DISALLOW_SET_CHARSET.
func (m *MySQLFamily[N]) ColumnDisallowSetCharset(fn func([]N) []v1review.Finding) Func {
	return m.simpleStmt(fn, code.SetColumnCharset, func(stmt V1Stmt[N], _ v1review.Finding) *storepb.Position { return m.r.posStatement(stmt) },
		func(stmt V1Stmt[N], _ v1review.Finding) string {
			return fmt.Sprintf("Disallow set column charset but \"%s\" does", m.r.stmtText(stmt, false))
		})
}

// ColumnMaximumCharacterLength builds COLUMN_MAXIMUM_CHARACTER_LENGTH.
func (m *MySQLFamily[N]) ColumnMaximumCharacterLength(fn func([]N, int) []v1review.Finding) Func {
	return m.numberStmtRule(fn, "number_payload is required for column maximum character length rule", true,
		func(stmt V1Stmt[N], f v1review.Finding, maximum int, status storepb.Advice_Status, title string) *storepb.Advice {
			content := fmt.Sprintf("The length of the CHAR column `%s.%s` is bigger than %d, please use VARCHAR instead", f.Table, f.Name, maximum)
			if m.r.tidbQuirks {
				content = fmt.Sprintf("The length of the CHAR column `%s` is %s, bigger than %d, please use VARCHAR instead", f.Name, f.Value, maximum)
			}
			return newAdvice(status, code.CharLengthExceedsLimit, title, content, m.r.posAbsOrFinding(stmt, f))
		})
}

// ColumnMaximumVarcharLength builds COLUMN_MAXIMUM_VARCHAR_LENGTH.
func (m *MySQLFamily[N]) ColumnMaximumVarcharLength(fn func([]N, int) []v1review.Finding) Func {
	return m.numberStmtRule(fn, "number_payload is required for column maximum varchar length rule", true,
		func(stmt V1Stmt[N], f v1review.Finding, maximum int, status storepb.Advice_Status, title string) *storepb.Advice {
			return newAdvice(status, code.VarcharLengthExceedsLimit, title, fmt.Sprintf("The length of the VARCHAR column `%s.%s` is bigger than %d", f.Table, f.Name, maximum), m.r.posAbsOrFinding(stmt, f))
		})
}

// ColumnNoNull builds COLUMN_NO_NULL.
func (m *MySQLFamily[N]) ColumnNoNull(fn func([]N, v1review.Schema) []v1review.Finding) Func {
	return func(_ context.Context, checkCtx Context) ([]*storepb.Advice, error) {
		p, err := m.prepare(checkCtx, true)
		if err != nil {
			return nil, err
		}
		if checkCtx.FinalMetadata == nil && m.opt.DefinesColumn != nil && slices.ContainsFunc(p.nodes, m.opt.DefinesColumn) {
			checkCtx.FinalMetadata.GetSchemaMetadata("")
		}
		type placed struct {
			line   int
			column string
			table  string
		}
		var columns []placed
		for _, f := range fn(p.nodes, v1SchemaOf(checkCtx.FinalMetadata)) {
			stmt := p.stmts[f.Statement]
			line := stmt.BaseLine + int(m.r.findingLine(stmt.Text, f))
			if m.r.tidbQuirks {
				line = m.r.absLine(stmt, f.Start)
			}
			columns = append(columns, placed{line: line, column: f.Name, table: f.Table})
		}
		slices.SortStableFunc(columns, func(a, b placed) int {
			return cmp.Or(cmp.Compare(a.line, b.line), cmp.Compare(a.column, b.column))
		})
		var adviceList []*storepb.Advice
		for _, c := range columns {
			adviceList = append(adviceList, newAdvice(p.status, code.ColumnCannotNull, p.title, fmt.Sprintf("`%s`.`%s` cannot have NULL value", c.table, c.column), base.ConvertANTLRLineToPosition(c.line)))
		}
		return adviceList, nil
	}
}

// ColumnRequireCharset builds COLUMN_REQUIRE_CHARSET.
func (m *MySQLFamily[N]) ColumnRequireCharset(fn func([]N) []v1review.Finding) Func {
	return m.simpleStmt(fn, code.NoCharset, m.r.posAbsOrFinding, func(_ V1Stmt[N], f v1review.Finding) string {
		return fmt.Sprintf("Column %s does not have a character set specified", f.Name)
	})
}

// ColumnRequireCollation builds COLUMN_REQUIRE_COLLATION.
func (m *MySQLFamily[N]) ColumnRequireCollation(fn func([]N) []v1review.Finding) Func {
	return m.simpleStmt(fn, code.NoCollation, m.r.posAbsOrFinding, func(_ V1Stmt[N], f v1review.Finding) string {
		return fmt.Sprintf("Column %s does not have a collation specified", f.Name)
	})
}

// ColumnRequireDefault builds COLUMN_REQUIRE_DEFAULT.
func (m *MySQLFamily[N]) ColumnRequireDefault(fn func([]N) []v1review.Finding) Func {
	return m.simpleStmt(fn, code.NoDefault, m.r.posAbsOrFinding, func(_ V1Stmt[N], f v1review.Finding) string {
		return fmt.Sprintf("Column `%s`.`%s` doesn't have DEFAULT.", f.Table, f.Name)
	})
}

// ColumnRequired builds COLUMN_REQUIRED.
func (m *MySQLFamily[N]) ColumnRequired(fn func([]N, []string) []v1review.Finding) Func {
	return func(_ context.Context, checkCtx Context) ([]*storepb.Advice, error) {
		p, err := m.prepare(checkCtx, true)
		if err != nil {
			return nil, err
		}
		payload := checkCtx.Rule.GetStringArrayPayload()
		if payload == nil {
			return nil, errors.New("string_array_payload is required for column required rule")
		}
		findings := fn(p.nodes, payload.List)
		var adviceList []*storepb.Advice
		for i := 0; i < len(findings); {
			f := findings[i]
			var missing []string
			for ; i < len(findings) && findings[i].Table == f.Table; i++ {
				missing = append(missing, findings[i].Name)
			}
			stmt := p.stmts[f.Statement]
			pos := m.r.posStatement(stmt)
			if !m.r.tidbQuirks {
				pos = &storepb.Position{Line: int32(stmt.BaseLine + int(m.r.findingLine(stmt.Text, f)))}
			}
			adviceList = append(adviceList, newAdvice(p.status, code.NoRequiredColumn, p.title, fmt.Sprintf("Table `%s` requires columns: %s", f.Table, strings.Join(missing, ", ")), pos))
		}
		return adviceList, nil
	}
}

// ColumnSetDefaultForNotNull builds COLUMN_SET_DEFAULT_FOR_NOT_NULL.
func (m *MySQLFamily[N]) ColumnSetDefaultForNotNull(fn func([]N) []v1review.Finding) Func {
	return m.simpleStmt(fn, code.NotNullColumnWithNoDefault, m.r.posAlterFirstToken, func(_ V1Stmt[N], f v1review.Finding) string {
		return fmt.Sprintf("Column `%s`.`%s` is NOT NULL but doesn't have DEFAULT", f.Table, f.Name)
	})
}

// ColumnTypeDisallowList builds COLUMN_TYPE_DISALLOW_LIST.
func (m *MySQLFamily[N]) ColumnTypeDisallowList(fn func([]N, []string) []v1review.Finding) Func {
	return m.stringArrayStmtRule(fn, code.DisabledColumnType, m.r.posAbsOrFinding, func(_ V1Stmt[N], f v1review.Finding) string {
		return fmt.Sprintf("Disallow column type %s but column `%s`.`%s` is", f.Value, f.Table, f.Name)
	})
}

// DatabaseDropEmptyDatabase builds DATABASE_DROP_EMPTY_DATABASE.
func (m *MySQLFamily[N]) DatabaseDropEmptyDatabase(fn func([]N, v1review.Schema) []v1review.Finding) Func {
	return func(_ context.Context, checkCtx Context) ([]*storepb.Advice, error) {
		p, err := m.prepare(checkCtx, false)
		if err != nil {
			return nil, err
		}
		var adviceList []*storepb.Advice
		for _, stmt := range p.stmts {
			var schema v1review.Schema
			if m.opt.IsDropDatabase != nil && m.opt.IsDropDatabase(stmt.Node) {
				schema = v1review.Schema{
					Metadata:            checkCtx.OriginalMetadata.GetProto(),
					ObjectCaseSensitive: checkCtx.OriginalMetadata.GetIsObjectCaseSensitive(),
				}
			}
			for _, f := range fn([]N{stmt.Node}, schema) {
				switch f.Code {
				case v1review.DatabaseNotCurrent:
					adviceList = append(adviceList, newAdvice(p.status, code.NotCurrentDatabase, p.title, fmt.Sprintf("Database `%s` that is trying to be deleted is not the current database `%s`", f.Name, checkCtx.OriginalMetadata.DatabaseName()), m.r.posBaseLineTwice(stmt, f)))
				case v1review.DatabaseNotEmpty:
					adviceList = append(adviceList, newAdvice(p.status, code.DatabaseNotEmpty, p.title, fmt.Sprintf("Database `%s` is not allowed to drop if not empty", f.Name), m.r.posBaseLineTwice(stmt, f)))
				default:
				}
			}
		}
		return adviceList, nil
	}
}

// EngineMySQLUseInnoDB builds ENGINE_MYSQL_USE_INNODB.
func (m *MySQLFamily[N]) EngineMySQLUseInnoDB(fn func([]N) []v1review.Finding) Func {
	return m.simpleStmt(fn, code.NotInnoDBEngine, m.r.posSearchOrStart, func(stmt V1Stmt[N], _ v1review.Finding) string {
		return fmt.Sprintf("\"%s;\" doesn't use InnoDB engine", trimmedStmtText(stmt.Text))
	})
}

// IndexKeyNumberLimit builds INDEX_KEY_NUMBER_LIMIT.
func (m *MySQLFamily[N]) IndexKeyNumberLimit(fn func([]N, int) []v1review.Finding) Func {
	return m.numberStmtRule(fn, "number_payload is required for this rule", false,
		func(stmt V1Stmt[N], f v1review.Finding, limit int, status storepb.Advice_Status, title string) *storepb.Advice {
			return newAdvice(status, code.IndexKeyNumberExceedsLimit, title, fmt.Sprintf("The number of index `%s` in table `%s` should be not greater than %d", keyName(stmt.Text, f, m.r.tidbQuirks), f.Table, limit), m.r.posFinding(stmt, f))
		})
}

// IndexNoDuplicateColumn builds INDEX_NO_DUPLICATE_COLUMN.
func (m *MySQLFamily[N]) IndexNoDuplicateColumn(fn func([]N) []v1review.Finding) Func {
	return m.simpleStmt(fn, code.DuplicateColumnInIndex, m.r.posFinding, func(stmt V1Stmt[N], f v1review.Finding) string {
		return fmt.Sprintf("%s `%s` has duplicate column `%s`.`%s`", f.Value, keyName(stmt.Text, f, m.r.tidbQuirks), f.Table, f.Other)
	})
}

// IndexPKTypeLimit builds INDEX_PK_TYPE_LIMIT.
func (m *MySQLFamily[N]) IndexPKTypeLimit(fn func([]N, v1review.Schema) ([]v1review.Finding, error)) Func {
	return m.indexSchemaBatchRule(fn, code.IndexPKType, func(f v1review.Finding) string {
		return fmt.Sprintf("Columns in primary key must be INT/BIGINT but `%s`.`%s` is %s", f.Table, f.Name, f.Value)
	})
}

// IndexTypeNoBlob builds INDEX_TYPE_NO_BLOB.
func (m *MySQLFamily[N]) IndexTypeNoBlob(fn func([]N, v1review.Schema) ([]v1review.Finding, error)) Func {
	return m.indexSchemaBatchRule(fn, code.IndexTypeNoBlob, func(f v1review.Finding) string {
		return fmt.Sprintf("Columns in index must not be BLOB but `%s`.`%s` is %s", f.Table, f.Name, f.Value)
	})
}

// IndexPrimaryKeyTypeAllowlist builds INDEX_PRIMARY_KEY_TYPE_ALLOWLIST.
func (m *MySQLFamily[N]) IndexPrimaryKeyTypeAllowlist(fn func([]N, []string, v1review.Schema) ([]v1review.Finding, error)) Func {
	return func(_ context.Context, checkCtx Context) ([]*storepb.Advice, error) {
		p, err := m.prepare(checkCtx, true)
		if err != nil {
			return nil, err
		}
		findings, err := fn(p.nodes, checkCtx.Rule.GetStringArrayPayload().List, v1SchemaOf(checkCtx.OriginalMetadata))
		if err != nil {
			readMissingSchema(checkCtx.OriginalMetadata, err)
			return nil, err
		}
		var adviceList []*storepb.Advice
		for _, f := range findings {
			adviceList = append(adviceList, newAdvice(p.status, code.IndexPKType, p.title, fmt.Sprintf("The column `%s` in table `%s` is one of the primary key, but its type \"%s\" is not in allowlist", f.Name, f.Table, f.Value), m.r.posFinding(p.stmts[f.Statement], f)))
		}
		return adviceList, nil
	}
}

// IndexTotalNumberLimit builds INDEX_TOTAL_NUMBER_LIMIT.
func (m *MySQLFamily[N]) IndexTotalNumberLimit(fn func([]N, int, v1review.Schema) ([]v1review.Finding, error)) Func {
	return func(_ context.Context, checkCtx Context) ([]*storepb.Advice, error) {
		p, err := m.prepare(checkCtx, true)
		if err != nil {
			return nil, err
		}
		limit, err := requireNumberPayload(checkCtx.Rule, "number_payload is required for this rule")
		if err != nil {
			return nil, err
		}
		findings, err := fn(p.nodes, limit, v1SchemaOf(checkCtx.FinalMetadata))
		if err != nil {
			readMissingSchema(checkCtx.FinalMetadata, err)
			return nil, err
		}
		var adviceList []*storepb.Advice
		for _, f := range findings {
			adviceList = append(adviceList, newAdvice(p.status, code.IndexCountExceedsLimit, p.title, fmt.Sprintf("The count of index in table `%s` should be no more than %d, but found %s", f.Table, limit, f.Value), m.r.posFinding(p.stmts[f.Statement], f)))
		}
		return adviceList, nil
	}
}

// IndexTypeAllowList builds INDEX_TYPE_ALLOW_LIST.
func (m *MySQLFamily[N]) IndexTypeAllowList(fn func([]N, []string) []v1review.Finding) Func {
	return m.stringArrayStmtRule(fn, code.IndexTypeNotAllowed, m.r.posFinding, func(_ V1Stmt[N], f v1review.Finding) string {
		return fmt.Sprintf("Index type `%s` is not allowed", f.Value)
	})
}

func (m *MySQLFamily[N]) namingRule(
	fn func([]N, v1review.NamingConfig) ([]v1review.Finding, error),
	adviceCode code.Code,
	missingMsg string,
	useFindingPos bool,
	mismatchFmt func(v1review.Finding, string) string,
	tooLongFmt func(v1review.Finding, int) string,
) Func {
	return func(_ context.Context, checkCtx Context) ([]*storepb.Advice, error) {
		p, err := m.prepare(checkCtx, false)
		if err != nil {
			return nil, err
		}
		namingPayload, maxLength, err := requireNamingPayload(checkCtx.Rule, missingMsg)
		if err != nil {
			return nil, err
		}
		cfg := v1review.NamingConfig{Pattern: namingPayload.Format, MaxLength: maxLength}
		if _, err := cfg.Compile(); err != nil {
			return nil, errors.Wrapf(err, "failed to compile regex format %q", namingPayload.Format)
		}
		var adviceList []*storepb.Advice
		for _, stmt := range p.stmts {
			findings, err := fn([]N{stmt.Node}, cfg)
			if err != nil {
				return nil, err
			}
			for _, f := range findings {
				var content string
				switch f.Code {
				case v1review.NameMismatch:
					content = mismatchFmt(f, namingPayload.Format)
				case v1review.NameTooLong:
					if tooLongFmt == nil {
						continue
					}
					content = tooLongFmt(f, maxLength)
				default:
					continue
				}
				pos := m.r.posAbsOrFinding(stmt, f)
				if useFindingPos {
					pos = m.r.posFinding(stmt, f)
				}
				adviceList = append(adviceList, newAdvice(p.status, adviceCode, p.title, content, pos))
			}
		}
		return adviceList, nil
	}
}

// NamingTable builds NAMING_TABLE.
func (m *MySQLFamily[N]) NamingTable(fn func([]N, v1review.NamingConfig) ([]v1review.Finding, error)) Func {
	return m.namingRule(fn, code.NamingTableConventionMismatch, "naming_payload is required for naming table rule", true,
		func(f v1review.Finding, format string) string {
			return fmt.Sprintf("`%s` mismatches table naming convention, naming format should be %q", f.Name, format)
		},
		func(f v1review.Finding, maxLength int) string {
			return fmt.Sprintf("`%s` mismatches table naming convention, its length should be within %d characters", f.Name, maxLength)
		},
	)
}

// NamingColumn builds NAMING_COLUMN.
func (m *MySQLFamily[N]) NamingColumn(fn func([]N, v1review.NamingConfig) ([]v1review.Finding, error)) Func {
	return m.namingRule(fn, code.NamingColumnConventionMismatch, "naming_payload is required for naming column rule", false,
		func(f v1review.Finding, format string) string {
			return fmt.Sprintf("`%s`.`%s` mismatches column naming convention, naming format should be %q", f.Table, f.Name, format)
		},
		func(f v1review.Finding, maxLength int) string {
			return fmt.Sprintf("`%s`.`%s` mismatches column naming convention, its length should be within %d characters", f.Table, f.Name, maxLength)
		},
	)
}

// NamingAutoIncrementColumn builds NAMING_COLUMN_AUTO_INCREMENT.
func (m *MySQLFamily[N]) NamingAutoIncrementColumn(fn func([]N, v1review.NamingConfig) ([]v1review.Finding, error)) Func {
	return m.namingRule(fn, code.NamingAutoIncrementColumnConventionMismatch, "naming_payload is required for this rule", false,
		func(f v1review.Finding, format string) string {
			return fmt.Sprintf("`%s`.`%s` mismatches auto_increment column naming convention, naming format should be %q", f.Table, f.Name, format)
		},
		func(f v1review.Finding, maxLength int) string {
			return fmt.Sprintf("`%s`.`%s` mismatches auto_increment column naming convention, its length should be within %d characters", f.Table, f.Name, maxLength)
		},
	)
}

// TableDropNaming builds TABLE_DROP_NAMING_CONVENTION.
func (m *MySQLFamily[N]) TableDropNaming(fn func([]N, v1review.NamingConfig) ([]v1review.Finding, error)) Func {
	return m.namingRule(fn, code.TableDropNamingConventionMismatch, "naming_payload is required for table drop naming convention rule", false,
		func(f v1review.Finding, format string) string {
			return fmt.Sprintf("`%s` mismatches drop table naming convention, naming format should be %q", f.Name, format)
		},
		nil,
	)
}

func (m *MySQLFamily[N]) templateNamingRule(
	adviceCode code.Code,
	internalTitle string,
	kindLabel string,
	eval func(V1Stmt[N], v1review.TemplateConfig, *model.DatabaseMetadata) ([]v1review.Finding, error),
) Func {
	return func(_ context.Context, checkCtx Context) ([]*storepb.Advice, error) {
		p, err := m.prepare(checkCtx, false)
		if err != nil {
			return nil, err
		}
		namingPayload, maxLength, err := requireNamingPayload(checkCtx.Rule, "naming_payload is required for this rule")
		if err != nil {
			return nil, err
		}
		templateList, _ := ParseTemplateTokens(namingPayload.Format)
		for _, key := range templateList {
			if _, ok := TemplateNamingTokens[checkCtx.Rule.Type][key]; !ok {
				return nil, errors.Errorf("invalid template %s for rule %s", key, checkCtx.Rule.Type)
			}
		}
		cfg := v1review.TemplateConfig{Format: namingPayload.Format, MaxLength: maxLength}
		var adviceList []*storepb.Advice
		for _, stmt := range p.stmts {
			findings, err := eval(stmt, cfg, checkCtx.OriginalMetadata)
			if err != nil {
				return nil, err
			}
			for _, f := range findings {
				switch f.Code {
				case v1review.PatternInvalid:
					adviceList = append(adviceList, newAdvice(p.status, code.Internal, internalTitle, fmt.Sprintf("%q meet internal error %q", trimmedStmtText(stmt.Text), f.Detail), nil))
				case v1review.NameMismatch:
					adviceList = append(adviceList, newAdvice(p.status, adviceCode, p.title, fmt.Sprintf("%s in table `%s` mismatches the naming convention, expect %q but found `%s`", kindLabel, f.Table, f.Pattern, f.Name), m.r.posFinding(stmt, f)))
				case v1review.NameTooLong:
					adviceList = append(adviceList, newAdvice(p.status, adviceCode, p.title, fmt.Sprintf("%s `%s` in table `%s` mismatches the naming convention, its length should be within %d characters", kindLabel, f.Name, f.Table, maxLength), m.r.posFinding(stmt, f)))
				default:
				}
			}
		}
		return adviceList, nil
	}
}

func (m *MySQLFamily[N]) renameIndexSchema(node N, metadata *model.DatabaseMetadata) v1review.Schema {
	if m.opt.IsRenameIndex != nil && m.opt.IsRenameIndex(node) {
		return v1review.Schema{Metadata: metadata.GetProto(), ObjectCaseSensitive: metadata.GetIsObjectCaseSensitive()}
	}
	return v1review.Schema{}
}

// NamingForeignKey builds NAMING_INDEX_FK.
func (m *MySQLFamily[N]) NamingForeignKey(fn func([]N, v1review.TemplateConfig) ([]v1review.Finding, error)) Func {
	return m.templateNamingRule(code.NamingFKConventionMismatch, "Internal error for foreign key naming convention rule", "Foreign key",
		func(stmt V1Stmt[N], tc v1review.TemplateConfig, _ *model.DatabaseMetadata) ([]v1review.Finding, error) {
			return fn([]N{stmt.Node}, tc)
		},
	)
}

// NamingIndex builds NAMING_INDEX_IDX.
func (m *MySQLFamily[N]) NamingIndex(fn func([]N, v1review.TemplateConfig, v1review.Schema) []v1review.Finding) Func {
	errTitle := "Internal error for unique key naming convention rule"
	if m.r.tidbQuirks {
		errTitle = "Internal error for index naming convention rule"
	}
	return m.templateNamingRule(code.NamingIndexConventionMismatch, errTitle, "Index",
		func(stmt V1Stmt[N], tc v1review.TemplateConfig, orig *model.DatabaseMetadata) ([]v1review.Finding, error) {
			return fn([]N{stmt.Node}, tc, m.renameIndexSchema(stmt.Node, orig)), nil
		},
	)
}

// NamingUniqueKey builds NAMING_INDEX_UK.
func (m *MySQLFamily[N]) NamingUniqueKey(fn func([]N, v1review.TemplateConfig, v1review.Schema) []v1review.Finding) Func {
	return m.templateNamingRule(code.NamingUKConventionMismatch, "Internal error for unique key naming convention rule", "Unique key",
		func(stmt V1Stmt[N], tc v1review.TemplateConfig, orig *model.DatabaseMetadata) ([]v1review.Finding, error) {
			return fn([]N{stmt.Node}, tc, m.renameIndexSchema(stmt.Node, orig)), nil
		},
	)
}

// NamingIdentifierNoKeyword builds NAMING_IDENTIFIER_NO_KEYWORD.
func (m *MySQLFamily[N]) NamingIdentifierNoKeyword(fn func([]N) []v1review.Finding) Func {
	return m.simpleStmt(fn, code.NameIsKeywordIdentifier, m.r.posSearchOrFinding, func(_ V1Stmt[N], f v1review.Finding) string {
		return fmt.Sprintf("Identifier %q is a keyword and should be avoided", f.Name)
	})
}

var (
	compatibilityCodes = map[v1review.Code]code.Code{
		v1review.IncompatibleDropDatabase:  code.CompatibilityDropDatabase,
		v1review.IncompatibleRenameTable:   code.CompatibilityRenameTable,
		v1review.IncompatibleDropTable:     code.CompatibilityDropTable,
		v1review.IncompatibleRenameColumn:  code.CompatibilityRenameColumn,
		v1review.IncompatibleDropColumn:    code.CompatibilityDropColumn,
		v1review.IncompatibleAddPrimaryKey: code.CompatibilityAddPrimaryKey,
		v1review.IncompatibleAddUniqueKey:  code.CompatibilityAddUniqueKey,
		v1review.IncompatibleAddForeignKey: code.CompatibilityAddForeignKey,
		v1review.IncompatibleAddCheck:      code.CompatibilityAddCheck,
		v1review.IncompatibleAlterCheck:    code.CompatibilityAlterCheck,
		v1review.IncompatibleAlterColumn:   code.CompatibilityAlterColumn,
	}
	dmlLimitCodes   = map[string]code.Code{"INSERT": code.InsertUseLimit, "UPDATE": code.UpdateUseLimit, "DELETE": code.DeleteUseLimit}
	dmlOrderByCodes = map[string]code.Code{"UPDATE": code.UpdateUseOrderBy, "DELETE": code.DeleteUseOrderBy}
)

// SchemaBackwardCompatibility builds SCHEMA_BACKWARD_COMPATIBILITY.
func (m *MySQLFamily[N]) SchemaBackwardCompatibility(fn func([]N) []v1review.Finding) Func {
	return m.batch(fn, func(stmts []V1Stmt[N], f v1review.Finding, status storepb.Advice_Status, title string) *storepb.Advice {
		adviceCode, ok := compatibilityCodes[f.Code]
		if !ok {
			return nil
		}
		stmt := stmts[f.Statement]
		return newAdvice(status, adviceCode, title, fmt.Sprintf("\"%s\" may cause incompatibility with the existing data and code", m.r.stmtText(stmt, true)), m.r.posBaseLineTwice(stmt, f))
	})
}

// StatementAddColumnWithoutPosition builds STATEMENT_ADD_COLUMN_WITHOUT_POSITION.
func (m *MySQLFamily[N]) StatementAddColumnWithoutPosition(fn func([]N) []v1review.Finding) Func {
	return m.simpleStmt(fn, code.StatementAddColumnWithPosition, m.r.posFinding, func(_ V1Stmt[N], f v1review.Finding) string {
		return fmt.Sprintf("add column with position \"%s\"", f.Name)
	})
}

// StatementDisallowCommit builds STATEMENT_DISALLOW_COMMIT.
func (m *MySQLFamily[N]) StatementDisallowCommit(fn func([]N) []v1review.Finding) Func {
	return m.simpleStmt(fn, code.StatementDisallowCommit, m.r.posFindingOrTiDBFirstToken, func(stmt V1Stmt[N], _ v1review.Finding) string {
		return fmt.Sprintf("Commit is not allowed, related statement: \"%s\"", m.r.stmtText(stmt, false))
	})
}

// StatementDisallowLimit builds STATEMENT_DISALLOW_LIMIT.
func (m *MySQLFamily[N]) StatementDisallowLimit(fn func([]N) []v1review.Finding) Func {
	return m.perStmt(fn, func(stmt V1Stmt[N], f v1review.Finding, status storepb.Advice_Status, title string) *storepb.Advice {
		return newAdvice(status, dmlLimitCodes[f.Name], title, fmt.Sprintf("LIMIT clause is forbidden in INSERT, UPDATE and DELETE statement, but \"%s\" uses", m.r.stmtText(stmt, false)), m.r.posFindingOrTiDBFirstToken(stmt, f))
	})
}

// StatementDisallowOrderBy builds STATEMENT_DISALLOW_ORDER_BY.
func (m *MySQLFamily[N]) StatementDisallowOrderBy(fn func([]N) []v1review.Finding) Func {
	return m.perStmt(fn, func(stmt V1Stmt[N], f v1review.Finding, status storepb.Advice_Status, title string) *storepb.Advice {
		return newAdvice(status, dmlOrderByCodes[f.Name], title, fmt.Sprintf("ORDER BY clause is forbidden in DELETE and UPDATE statements, but \"%s\" uses", m.r.stmtText(stmt, false)), m.r.posFindingOrTiDBFirstToken(stmt, f))
	})
}

// StatementDisallowTruncate builds STATEMENT_DISALLOW_TRUNCATE.
func (m *MySQLFamily[N]) StatementDisallowTruncate(fn func([]N) []v1review.Finding) Func {
	return m.perStmt(fn, func(stmt V1Stmt[N], f v1review.Finding, status storepb.Advice_Status, title string) *storepb.Advice {
		var content string
		switch f.Code {
		case v1review.Truncate:
			content = fmt.Sprintf(`TRUNCATE TABLE %q is not allowed: TRUNCATE auto-commits (breaking surrounding transactional work), bypasses triggers, and resets AUTO_INCREMENT. Prior-backup treats this as DDL and does not produce row-level snapshots.`, f.Table)
		case v1review.TruncatePartition:
			content = fmt.Sprintf(`ALTER TABLE %q TRUNCATE PARTITION %q is not allowed: partition truncate shares the auto-commit and prior-backup gaps of TRUNCATE TABLE on MySQL.`, f.Table, f.Name)
		default:
			return nil
		}
		return newAdvice(status, code.StatementDisallowTruncate, title, content, m.r.posFinding(stmt, f))
	})
}

// StatementInsertDisallowOrderByRand builds STATEMENT_INSERT_DISALLOW_ORDER_BY_RAND.
func (m *MySQLFamily[N]) StatementInsertDisallowOrderByRand(fn func([]N) []v1review.Finding) Func {
	return m.simpleStmt(fn, code.InsertUseOrderByRand, m.r.posFindingOrTiDBFirstToken, func(stmt V1Stmt[N], _ v1review.Finding) string {
		return fmt.Sprintf("\"%s\" uses ORDER BY RAND in the INSERT statement", m.r.stmtText(stmt, false))
	})
}

// StatementInsertMustSpecifyColumn builds STATEMENT_INSERT_MUST_SPECIFY_COLUMN.
func (m *MySQLFamily[N]) StatementInsertMustSpecifyColumn(fn func([]N) []v1review.Finding) Func {
	return m.simpleStmt(fn, code.InsertNotSpecifyColumn, m.r.posAbsOrFinding, func(stmt V1Stmt[N], _ v1review.Finding) string {
		return fmt.Sprintf("The INSERT statement must specify columns but \"%s\" does not", m.r.stmtText(stmt, true))
	})
}

// StatementMaxExecutionTime builds STATEMENT_MAX_EXECUTION_TIME.
func (m *MySQLFamily[N]) StatementMaxExecutionTime(fn func([]N) []v1review.Finding) Func {
	return m.batch(fn, func(_ []V1Stmt[N], f v1review.Finding, status storepb.Advice_Status, title string) *storepb.Advice {
		return newAdvice(status, code.StatementNoMaxExecutionTime, title, fmt.Sprintf("The %s is not set", f.Name), nil)
	})
}

// StatementMergeAlterTable builds STATEMENT_MERGE_ALTER_TABLE.
func (m *MySQLFamily[N]) StatementMergeAlterTable(fn func([]N) []v1review.Finding) Func {
	return m.batch(fn, func(stmts []V1Stmt[N], f v1review.Finding, status storepb.Advice_Status, title string) *storepb.Advice {
		return newAdvice(status, code.StatementRedundantAlterTable, title, fmt.Sprintf("There are %s statements to modify table `%s`", f.Value, f.Table), m.r.posFindingOrTiDBFirstToken(stmts[f.Statement], f))
	})
}

// StatementRequireAlgorithmOption builds STATEMENT_REQUIRE_ALGORITHM_OPTION.
func (m *MySQLFamily[N]) StatementRequireAlgorithmOption(fn func([]N) []v1review.Finding) Func {
	return m.simpleStmt(fn, code.StatementNoAlgorithmOption, m.r.posFinding, func(_ V1Stmt[N], f v1review.Finding) string {
		return "ALTER TABLE statement should include " + f.Name + " option"
	})
}

// StatementRequireLockOption builds STATEMENT_REQUIRE_LOCK_OPTION.
func (m *MySQLFamily[N]) StatementRequireLockOption(fn func([]N) []v1review.Finding) Func {
	return m.simpleStmt(fn, code.StatementNoLockOption, m.r.posFinding, func(_ V1Stmt[N], f v1review.Finding) string {
		return "ALTER TABLE statement should include " + f.Name + " option"
	})
}

// StatementSelectNoSelectAll builds STATEMENT_SELECT_NO_SELECT_ALL.
func (m *MySQLFamily[N]) StatementSelectNoSelectAll(fn func([]N) []v1review.Finding) Func {
	return m.simpleStmt(fn, code.StatementSelectAll, m.r.posFindingOrTiDBFirstToken, func(stmt V1Stmt[N], _ v1review.Finding) string {
		return fmt.Sprintf("\"%s\" uses SELECT all", m.r.stmtText(stmt, false))
	})
}

// StatementWhereNoLeadingWildcardLike builds STATEMENT_WHERE_NO_LEADING_WILDCARD_LIKE.
func (m *MySQLFamily[N]) StatementWhereNoLeadingWildcardLike(fn func([]N) []v1review.Finding) Func {
	return m.simpleStmt(fn, code.StatementLeadingWildcardLike, m.r.posFindingOrTiDBFirstToken, func(stmt V1Stmt[N], _ v1review.Finding) string {
		return fmt.Sprintf("\"%s\" uses leading wildcard LIKE", m.r.stmtText(stmt, false))
	})
}

// StatementWhereRequireSelect builds STATEMENT_WHERE_REQUIRE_SELECT.
func (m *MySQLFamily[N]) StatementWhereRequireSelect(fn func([]N) []v1review.Finding) Func {
	return m.simpleStmt(fn, code.StatementNoWhere, m.r.posFindingOrTiDBFirstToken, func(stmt V1Stmt[N], _ v1review.Finding) string {
		return fmt.Sprintf("\"%s\" requires WHERE clause", m.r.stmtText(stmt, false))
	})
}

// StatementWhereRequireUpdateDelete builds STATEMENT_WHERE_REQUIRE_UPDATE_DELETE.
func (m *MySQLFamily[N]) StatementWhereRequireUpdateDelete(fn func([]N) []v1review.Finding) Func {
	return m.StatementWhereRequireSelect(fn)
}

// StatementWhereNoEqualNull builds STATEMENT_WHERE_NO_EQUAL_NULL.
func (m *MySQLFamily[N]) StatementWhereNoEqualNull(fn func([]N) []v1review.Finding) Func {
	return m.simpleStmt(fn, code.StatementWhereNoEqualNull, m.r.posFinding, func(stmt V1Stmt[N], _ v1review.Finding) string {
		return fmt.Sprintf("WHERE clause contains equal null: %s", m.r.stmtText(stmt, false))
	})
}

// StatementWhereDisallowFunctionsAndCalc builds STATEMENT_WHERE_DISALLOW_FUNCTIONS_AND_CALCULATIONS.
func (m *MySQLFamily[N]) StatementWhereDisallowFunctionsAndCalc(fn func([]N, v1review.Schema) []v1review.Finding) Func {
	return func(_ context.Context, checkCtx Context) ([]*storepb.Advice, error) {
		p, err := m.prepare(checkCtx, false)
		if err != nil {
			return nil, err
		}
		var schema v1review.Schema
		if checkCtx.DBSchema != nil {
			schema = v1review.Schema{Metadata: checkCtx.DBSchema, ObjectCaseSensitive: checkCtx.IsObjectCaseSensitive}
		}
		var adviceList []*storepb.Advice
		for _, stmt := range p.stmts {
			for _, f := range fn([]N{stmt.Node}, schema) {
				var content string
				switch f.Code {
				case v1review.FunctionOnIndexedColumn:
					content = fmt.Sprintf("Function %q is applied to indexed column %q in the WHERE clause, which prevents index usage", f.Value, f.Name)
				case v1review.CalculationOnIndexedColumn:
					content = fmt.Sprintf("Calculation is applied to indexed column %q in the WHERE clause, which prevents index usage", f.Name)
				default:
					continue
				}
				adviceList = append(adviceList, newAdvice(p.status, code.StatementDisallowFunctionsAndCalculations, p.title, content, m.r.posFinding(stmt, f)))
			}
		}
		return adviceList, nil
	}
}

// StatementWhereMaximumLogicalOperatorCount builds STATEMENT_WHERE_MAXIMUM_LOGICAL_OPERATOR_COUNT.
func (m *MySQLFamily[N]) StatementWhereMaximumLogicalOperatorCount(fn func([]N, int) []v1review.Finding) Func {
	return m.numberStmtRule(fn, "number_payload is required for this rule", false,
		func(stmt V1Stmt[N], f v1review.Finding, maximum int, status storepb.Advice_Status, title string) *storepb.Advice {
			op := "the OR predicate"
			if f.Name == "IN" {
				op = "IN predicate"
			}
			content := fmt.Sprintf("Number of tokens (%s) in %s operation exceeds limit (%d) in statement %q.", f.Value, op, maximum, strings.TrimSpace(stmt.Text))
			return newAdvice(status, code.StatementWhereMaximumLogicalOperatorCount, title, content, m.r.posFinding(stmt, f))
		})
}

// SystemCharsetAllowlist builds SYSTEM_CHARSET_ALLOWLIST.
func (m *MySQLFamily[N]) SystemCharsetAllowlist(fn func([]N, []string) []v1review.Finding) Func {
	return m.stringArrayStmtRule(fn, code.DisabledCharset, m.r.posAllowlist, func(stmt V1Stmt[N], f v1review.Finding) string {
		return fmt.Sprintf("\"%s\" used disabled charset '%s'", strings.TrimSpace(stmt.Text), f.Name)
	})
}

// SystemCollationAllowlist builds SYSTEM_COLLATION_ALLOWLIST.
func (m *MySQLFamily[N]) SystemCollationAllowlist(fn func([]N, []string) []v1review.Finding) Func {
	return m.stringArrayStmtRule(fn, code.DisabledCollation, m.r.posAllowlist, func(stmt V1Stmt[N], f v1review.Finding) string {
		return fmt.Sprintf("\"%s\" used disabled collation '%s'", strings.TrimSpace(stmt.Text), f.Name)
	})
}

// SystemFunctionDisallowedList builds SYSTEM_FUNCTION_DISALLOWED_LIST.
func (m *MySQLFamily[N]) SystemFunctionDisallowedList(fn func([]N, []string) []v1review.Finding) Func {
	return m.stringArrayStmtRule(fn, code.DisabledFunction, m.r.posBaseLineTwice, func(stmt V1Stmt[N], f v1review.Finding) string {
		return fmt.Sprintf("Function \"%s\" is disallowed, but \"%s\" uses", f.Name, strings.TrimSpace(stmt.Text))
	})
}

// TableDisallowDDL builds TABLE_DISALLOW_DDL.
func (m *MySQLFamily[N]) TableDisallowDDL(fn func([]N, []string) []v1review.Finding) Func {
	return m.stringArrayStmtRule(fn, code.TableDisallowDDL, m.r.posFinding, func(_ V1Stmt[N], f v1review.Finding) string {
		return fmt.Sprintf("DDL is disallowed on table %s.", f.Name)
	})
}

// TableDisallowDML builds TABLE_DISALLOW_DML.
func (m *MySQLFamily[N]) TableDisallowDML(fn func([]N, []string) []v1review.Finding) Func {
	return m.stringArrayStmtRule(fn, code.TableDisallowDML, m.r.posFinding, func(_ V1Stmt[N], f v1review.Finding) string {
		return fmt.Sprintf("DML is disallowed on table %s.", f.Name)
	})
}

// SystemProcedureDisallowCreate builds SYSTEM_PROCEDURE_DISALLOW_CREATE.
func (m *MySQLFamily[N]) SystemProcedureDisallowCreate(fn func([]N) []v1review.Finding) Func {
	return m.disallowCreateRule(fn, code.DisallowCreateProcedure, "Procedure")
}

// SystemEventDisallowCreate builds SYSTEM_EVENT_DISALLOW_CREATE.
func (m *MySQLFamily[N]) SystemEventDisallowCreate(fn func([]N) []v1review.Finding) Func {
	return m.disallowCreateRule(fn, code.DisallowCreateEvent, "Event")
}

// SystemFunctionDisallowCreate builds SYSTEM_FUNCTION_DISALLOW_CREATE.
func (m *MySQLFamily[N]) SystemFunctionDisallowCreate(fn func([]N) []v1review.Finding) Func {
	return m.disallowCreateRule(fn, code.DisallowCreateFunction, "Function")
}

// SystemViewDisallowCreate builds SYSTEM_VIEW_DISALLOW_CREATE.
func (m *MySQLFamily[N]) SystemViewDisallowCreate(fn func([]N) []v1review.Finding) Func {
	return m.disallowCreateRule(fn, code.DisallowCreateView, "View")
}

// TableComment builds TABLE_COMMENT.
func (m *MySQLFamily[N]) TableComment(fn func([]N, *v1review.TableCommentConfig) []v1review.Finding) Func {
	return func(_ context.Context, checkCtx Context) ([]*storepb.Advice, error) {
		p, err := m.prepare(checkCtx, false)
		if err != nil {
			return nil, err
		}
		var tableCfg *v1review.TableCommentConfig
		if payload := checkCtx.Rule.GetCommentConventionPayload(); payload != nil {
			tableCfg = &v1review.TableCommentConfig{Required: payload.Required, MaxLength: int(payload.MaxLength)}
		}
		var adviceList []*storepb.Advice
		for _, stmt := range p.stmts {
			pos := m.r.posStatement(stmt)
			for _, f := range fn([]N{stmt.Node}, tableCfg) {
				switch f.Code {
				case v1review.TableCommentMissing:
					adviceList = append(adviceList, newAdvice(p.status, code.CommentEmpty, p.title, fmt.Sprintf("Table `%s` requires comments", f.Name), pos))
				case v1review.TableCommentTooLong:
					adviceList = append(adviceList, newAdvice(p.status, code.CommentTooLong, p.title, fmt.Sprintf("The length of table `%s` comment should be within %d characters", f.Name, tableCfg.MaxLength), pos))
				default:
				}
			}
		}
		return adviceList, nil
	}
}

// TableDisallowPartition builds TABLE_DISALLOW_PARTITION.
func (m *MySQLFamily[N]) TableDisallowPartition(fn func([]N) []v1review.Finding) Func {
	return m.perStmt(fn, func(stmt V1Stmt[N], f v1review.Finding, status storepb.Advice_Status, title string) *storepb.Advice {
		if f.Code != "" && f.Code != v1review.TablePartitioned {
			return nil
		}
		return newAdvice(status, code.CreateTablePartition, title, fmt.Sprintf("Table partition is forbidden, but \"%s\" creates", m.r.stmtText(stmt, false)), m.r.posStatement(stmt))
	})
}

// TableDisallowSetCharset builds TABLE_DISALLOW_SET_CHARSET.
func (m *MySQLFamily[N]) TableDisallowSetCharset(fn func([]N) []v1review.Finding) Func {
	return m.simpleStmt(fn, code.DisallowSetCharset, func(stmt V1Stmt[N], _ v1review.Finding) *storepb.Position { return m.r.posStatement(stmt) },
		func(stmt V1Stmt[N], _ v1review.Finding) string {
			return fmt.Sprintf("Set charset on tables is disallowed, but \"%s\" uses", strings.TrimSpace(stmt.Text))
		})
}

// TableDisallowTrigger builds TABLE_DISALLOW_TRIGGER.
func (m *MySQLFamily[N]) TableDisallowTrigger(fn func([]N) []v1review.Finding) Func {
	return m.disallowCreateRule(fn, code.CreateTableTrigger, "Trigger")
}

// TableLimitSize builds TABLE_LIMIT_SIZE.
func (m *MySQLFamily[N]) TableLimitSize(fn func([]N, *metadatapb.DatabaseSchemaMetadata, int) []v1review.Finding) Func {
	return func(_ context.Context, checkCtx Context) ([]*storepb.Advice, error) {
		p, err := m.prepare(checkCtx, true)
		if err != nil {
			return nil, err
		}
		limit, err := requireNumberPayload(checkCtx.Rule, "number_payload is required for this rule")
		if err != nil {
			return nil, err
		}
		var adviceList []*storepb.Advice
		for _, f := range fn(p.nodes, checkCtx.DBSchema, limit) {
			if f.Code == v1review.TableTooLarge {
				adviceList = append(adviceList, newAdvice(p.status, code.TableExceedLimitSize, p.title, fmt.Sprintf("Apply DDL on large table '%s' ( %s rows ) will lock table for a long time", f.Name, f.Value), m.r.posStatement(p.stmts[f.Statement])))
			}
		}
		return adviceList, nil
	}
}

// TableNoDuplicateIndex builds TABLE_NO_DUPLICATE_INDEX.
func (m *MySQLFamily[N]) TableNoDuplicateIndex(fn func([]N) []v1review.Finding) Func {
	return m.batch(fn, func(stmts []V1Stmt[N], f v1review.Finding, status storepb.Advice_Status, title string) *storepb.Advice {
		return newAdvice(status, code.DuplicateIndexInTable, title, fmt.Sprintf("`%s` has duplicate index `%s`", f.Table, f.Name), m.r.posFinding(stmts[f.Statement], f))
	})
}

// TableNoForeignKey builds TABLE_NO_FOREIGN_KEY.
func (m *MySQLFamily[N]) TableNoForeignKey(fn func([]N) []v1review.Finding) Func {
	return m.perStmt(fn, func(stmt V1Stmt[N], f v1review.Finding, status storepb.Advice_Status, title string) *storepb.Advice {
		if f.Code != "" && f.Code != v1review.TableForeignKey {
			return nil
		}
		table := f.Table
		if table == "" {
			table = f.Name
		}
		return newAdvice(status, code.TableHasFK, title, fmt.Sprintf("Foreign key is not allowed in the table `%s`", table), m.r.posBaseLineTwice(stmt, f))
	})
}

// TableRequireCharset builds TABLE_REQUIRE_CHARSET.
func (m *MySQLFamily[N]) TableRequireCharset(fn func([]N) []v1review.Finding) Func {
	return m.simpleStmt(fn, code.NoCharset, func(stmt V1Stmt[N], _ v1review.Finding) *storepb.Position { return m.r.posStatement(stmt) },
		func(_ V1Stmt[N], f v1review.Finding) string {
			return fmt.Sprintf("Table %s does not have a character set specified", f.Name)
		})
}

// TableRequireCollation builds TABLE_REQUIRE_COLLATION.
func (m *MySQLFamily[N]) TableRequireCollation(fn func([]N) []v1review.Finding) Func {
	return m.simpleStmt(fn, code.NoCollation, func(stmt V1Stmt[N], _ v1review.Finding) *storepb.Position { return m.r.posStatement(stmt) },
		func(_ V1Stmt[N], f v1review.Finding) string {
			return fmt.Sprintf("Table %s does not have a collation specified", f.Name)
		})
}

// TableRequirePK builds TABLE_REQUIRE_PK.
func (m *MySQLFamily[N]) TableRequirePK(fn func([]N, *v1review.Schema) []v1review.Finding) Func {
	return func(_ context.Context, checkCtx Context) ([]*storepb.Advice, error) {
		p, err := m.prepare(checkCtx, true)
		if err != nil {
			return nil, err
		}
		var adviceList []*storepb.Advice
		for _, f := range fn(p.nodes, v1SchemaPtrOf(checkCtx.OriginalMetadata)) {
			if f.Code != v1review.TablePrimaryKeyMissing {
				continue
			}
			line := 0
			if f.Statement >= 0 {
				stmt := p.stmts[f.Statement]
				if m.r.tidbQuirks {
					line = m.r.absLine(stmt, f.Start)
				} else {
					line = stmt.BaseLine + int(m.r.findingLine(stmt.Text, f))
				}
			}
			adviceList = append(adviceList, newAdvice(p.status, code.TableNoPK, p.title, fmt.Sprintf("Table `%s` requires PRIMARY KEY", f.Name), base.ConvertANTLRLineToPosition(line)))
		}
		return adviceList, nil
	}
}

// TableTextFieldsTotalLength builds TABLE_TEXT_FIELDS_TOTAL_LENGTH.
func (m *MySQLFamily[N]) TableTextFieldsTotalLength(fn func([]N, *v1review.Schema, int) []v1review.Finding) Func {
	return func(_ context.Context, checkCtx Context) ([]*storepb.Advice, error) {
		p, err := m.prepare(checkCtx, false)
		if err != nil {
			return nil, err
		}
		maximum, err := requireNumberPayload(checkCtx.Rule, "number_payload is required for this rule")
		if err != nil {
			return nil, err
		}
		final := v1SchemaPtrOf(checkCtx.FinalMetadata)
		var adviceList []*storepb.Advice
		for _, stmt := range p.stmts {
			adviceCode := code.TotalTextLengthExceedsLimit
			if m.opt.IsCreateTable != nil && m.opt.IsCreateTable(stmt.Node) {
				adviceCode = code.IndexCountExceedsLimit
			}
			for _, f := range fn([]N{stmt.Node}, final, maximum) {
				if f.Code == v1review.TableTextTooLong {
					adviceList = append(adviceList, newAdvice(p.status, adviceCode, p.title, fmt.Sprintf("Table %q total text column length (%s) exceeds the limit (%d).", f.Name, f.Value, maximum), &storepb.Position{Line: int32(stmt.BaseLine) + m.r.findingLine(stmt.Text, f)}))
				}
			}
		}
		return adviceList, nil
	}
}
