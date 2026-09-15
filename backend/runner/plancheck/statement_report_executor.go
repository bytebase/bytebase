package plancheck

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/log"
	"github.com/bytebase/bytebase/backend/component/dbfactory"
	"github.com/bytebase/bytebase/backend/component/sheet"
	"github.com/bytebase/bytebase/backend/plugin/advisor/code"
	"github.com/bytebase/bytebase/backend/plugin/db"
	cockroachdbdriver "github.com/bytebase/bytebase/backend/plugin/db/cockroachdb"
	mssqldriver "github.com/bytebase/bytebase/backend/plugin/db/mssql"
	mysqldriver "github.com/bytebase/bytebase/backend/plugin/db/mysql"
	oracledriver "github.com/bytebase/bytebase/backend/plugin/db/oracle"
	pgdriver "github.com/bytebase/bytebase/backend/plugin/db/pg"
	redshiftdriver "github.com/bytebase/bytebase/backend/plugin/db/redshift"
	tidbdriver "github.com/bytebase/bytebase/backend/plugin/db/tidb"
	parserbase "github.com/bytebase/bytebase/backend/plugin/parser/base"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/store"
)

// NewStatementReportExecutor creates a statement report executor.
func NewStatementReportExecutor(store *store.Store, sheetManager *sheet.Manager, dbFactory *dbfactory.DBFactory) Executor {
	return &StatementReportExecutor{
		store:        store,
		sheetManager: sheetManager,
		dbFactory:    dbFactory,
	}
}

// StatementReportExecutor is the statement report executor.
type StatementReportExecutor struct {
	store        *store.Store
	sheetManager *sheet.Manager
	dbFactory    *dbfactory.DBFactory
}

// RunForTarget runs the statement report check for a single target.
func (e *StatementReportExecutor) RunForTarget(ctx context.Context, target *CheckTarget) ([]*storepb.PlanCheckRunResult_Result, error) {
	fullSheet, err := e.store.GetSheetFull(ctx, target.SheetSha256)
	if err != nil {
		return nil, err
	}
	if fullSheet == nil {
		return nil, errors.Errorf("sheet full %s not found", target.SheetSha256)
	}
	if fullSheet.Size > common.MaxSheetCheckSize {
		return []*storepb.PlanCheckRunResult_Result{
			{
				Status:  storepb.Advice_WARNING,
				Code:    common.SizeExceeded.Int32(),
				Title:   "Report for large SQL is not supported",
				Content: "",
			},
		}, nil
	}

	instance, database, err := resolveDatabaseTarget(ctx, e.store, target.Target)
	if err != nil {
		return nil, err
	}
	if !common.EngineSupportStatementReport(instance.Metadata.GetEngine()) {
		return []*storepb.PlanCheckRunResult_Result{
			{
				Status:  storepb.Advice_SUCCESS,
				Code:    common.Ok.Int32(),
				Title:   fmt.Sprintf("Statement report is not supported for %s", instance.Metadata.GetEngine()),
				Content: "",
			},
		}, nil
	}

	// Check statement syntax error.
	_, syntaxAdvices := e.sheetManager.GetStatementsForChecks(instance.Metadata.GetEngine(), fullSheet.Statement)
	if len(syntaxAdvices) > 0 {
		advice := syntaxAdvices[0]
		return []*storepb.PlanCheckRunResult_Result{
			{
				Status:  storepb.Advice_ERROR,
				Title:   advice.Title,
				Content: advice.Content,
				Code:    advice.Code,
				Report: &storepb.PlanCheckRunResult_Result_SqlReviewReport_{
					SqlReviewReport: &storepb.PlanCheckRunResult_Result_SqlReviewReport{
						StartPosition: advice.StartPosition,
						EndPosition:   advice.EndPosition,
					},
				},
			},
		}, nil
	}

	planCheckRunResult := &storepb.PlanCheckRunResult_Result{
		Status: storepb.Advice_SUCCESS,
		Code:   common.Ok.Int32(),
		Title:  "OK",
	}
	summaryReport, estimateWarning, err := GetSQLSummaryReport(ctx, e.store, e.sheetManager, e.dbFactory, database, fullSheet.Statement)
	if err != nil {
		return nil, err
	}
	if summaryReport != nil {
		planCheckRunResult.Report = &storepb.PlanCheckRunResult_Result_SqlSummaryReport_{
			SqlSummaryReport: summaryReport,
		}
	}
	if estimateWarning != "" {
		planCheckRunResult.Status = storepb.Advice_WARNING
		planCheckRunResult.Code = code.StatementExplainQueryFailed.Int32()
		planCheckRunResult.Title = AffectedRowsEstimateIncompleteTitle
		planCheckRunResult.Content = estimateWarning
	}
	return []*storepb.PlanCheckRunResult_Result{planCheckRunResult}, nil
}

// AffectedRowsEstimateIncompleteTitle titles the warning for DML statements whose affected
// rows could not be estimated.
const AffectedRowsEstimateIncompleteTitle = "Affected rows estimate is incomplete"

// GetSQLSummaryReport gets the SQL summary report for the given statement and database. The
// returned warning is non-empty when some DML statements' affected rows could not be estimated.
func GetSQLSummaryReport(ctx context.Context, stores *store.Store, sheetManager *sheet.Manager, dbFactory *dbfactory.DBFactory, database *store.DatabaseMessage, statement string) (*storepb.PlanCheckRunResult_Result_SqlSummaryReport, string, error) {
	instance, err := stores.GetInstanceByResourceID(ctx, database.InstanceID)
	if err != nil {
		return nil, "", err
	}
	if instance == nil {
		return nil, "", errors.Errorf("instance not found: %s", database.InstanceID)
	}
	databaseSchema, err := stores.GetDBSchema(ctx, &store.FindDBSchemaMessage{
		Workspace:    instance.Workspace,
		InstanceID:   database.InstanceID,
		DatabaseName: database.DatabaseName,
	})
	if err != nil {
		return nil, "", err
	}
	if databaseSchema == nil {
		return nil, "", errors.Errorf("database schema %s not found", database.String())
	}
	if databaseSchema.GetProto() == nil {
		return nil, "", errors.Errorf("database schema metadata %s not found", database.String())
	}

	stmts, syntaxAdvices := sheetManager.GetStatementsForChecks(instance.Metadata.GetEngine(), statement)
	if len(syntaxAdvices) > 0 {
		// Return nil as it should already be checked before running this function.
		return nil, "", nil
	}
	asts := parserbase.ExtractASTs(stmts)

	var explainCalculator getAffectedRowsFromExplain
	var defaultSchema string
	project, err := stores.GetProject(ctx, &store.FindProjectMessage{Workspace: instance.Workspace, ResourceID: &database.ProjectID})
	if err != nil {
		return nil, "", err
	}
	driver, err := dbFactory.GetAdminDatabaseDriver(ctx, instance, database, db.ConnectionContext{
		TenantMode: project.Setting.GetPostgresDatabaseTenantMode(),
	})
	if err != nil {
		return nil, "", err
	}
	defer driver.Close(ctx)

	switch instance.Metadata.GetEngine() {
	case storepb.Engine_POSTGRES:
		pd, ok := driver.(*pgdriver.Driver)
		if !ok {
			return nil, "", errors.Errorf("invalid pg driver type")
		}
		explainCalculator = pd.CountAffectedRows
		// Empty so pg's extractChangedResources falls back to the database's actual
		// default search_path (Change-1 makes a non-empty value force [currentSchema]).
		defaultSchema = ""
	case storepb.Engine_REDSHIFT:
		rd, ok := driver.(*redshiftdriver.Driver)
		if !ok {
			return nil, "", errors.Errorf("invalid redshift driver type")
		}
		explainCalculator = rd.CountAffectedRows
		defaultSchema = "public"
	case storepb.Engine_COCKROACHDB:
		cd, ok := driver.(*cockroachdbdriver.Driver)
		if !ok {
			return nil, "", errors.Errorf("invalid cockroachdb driver type")
		}
		explainCalculator = cd.CountAffectedRows
		// Empty so the extractor resolves names with the database's synced search_path, as for PostgreSQL.
		defaultSchema = ""
	case storepb.Engine_MYSQL, storepb.Engine_MARIADB, storepb.Engine_OCEANBASE:
		md, ok := driver.(*mysqldriver.Driver)
		if !ok {
			return nil, "", errors.Errorf("invalid mysql driver type")
		}
		explainCalculator = md.CountAffectedRows
		defaultSchema = ""
	case storepb.Engine_TIDB:
		md, ok := driver.(*tidbdriver.Driver)
		if !ok {
			return nil, "", errors.Errorf("invalid tidb driver type")
		}
		explainCalculator = md.CountAffectedRows
		defaultSchema = ""
	case storepb.Engine_ORACLE:
		od, ok := driver.(*oracledriver.Driver)
		if !ok {
			return nil, "", errors.Errorf("invalid oracle driver type")
		}
		explainCalculator = od.CountAffectedRows
		defaultSchema = database.DatabaseName
	case storepb.Engine_MSSQL:
		md, ok := driver.(*mssqldriver.Driver)
		if !ok {
			return nil, "", errors.Errorf("invalid mssql driver type")
		}
		explainCalculator = md.CountAffectedRows
		defaultSchema = "dbo"
	default:
		// Already checked in the Run().
		return nil, "", nil
	}

	sqlTypes, err := SummaryStatementTypes(instance.Metadata.GetEngine(), asts)
	if err != nil {
		return nil, "", err
	}

	// Database secrets feature has been removed
	// To avoid leaking the rendered statement, the error message should use the original statement and not the rendered statement.
	// Database secrets feature removed - using original statement directly
	changeSummary, err := parserbase.ExtractChangedResources(instance.Metadata.GetEngine(), database.DatabaseName, defaultSchema, databaseSchema, asts, statement)
	if err != nil {
		return nil, "", errors.Wrapf(err, "failed to extract changed resources")
	}
	totalAffectedRows, estimateWarning := calculateAffectedRows(ctx, instance.Metadata.GetEngine(), changeSummary, explainCalculator)

	return &storepb.PlanCheckRunResult_Result_SqlSummaryReport{
		StatementTypes:   sqlTypes,
		AffectedRows:     totalAffectedRows,
		ChangedResources: changeSummary.ChangedResources.Build(),
	}, estimateWarning, nil
}

// SummaryStatementTypes classifies a summary report's statements through the
// parser registry, the single classifier per engine that the approval
// evaluator's parser fallback also uses. An engine without a registered
// classifier is an error rather than an empty list, because an empty list
// silently drops every statement.sql_type approval rule; classifier errors
// propagate for the same reason.
func SummaryStatementTypes(engine storepb.Engine, asts []parserbase.AST) ([]storepb.StatementType, error) {
	sqlTypes, err := parserbase.GetStatementTypes(engine, asts)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get statement types for %s", engine)
	}
	return sqlTypes, nil
}

type getAffectedRowsFromExplain func(context.Context, string) (int64, error)

// calculateAffectedRows adds the estimated rows of the DML statements, the inserted VALUES rows,
// and the rows of tables changed by DDL. The returned warning describes DML statements whose rows
// could not be estimated.
func calculateAffectedRows(ctx context.Context, engine storepb.Engine, changeSummary *parserbase.ChangeSummary, explainCalculator getAffectedRowsFromExplain) (int64, string) {
	mysqlFamily := engine == storepb.Engine_MYSQL || engine == storepb.Engine_MARIADB || engine == storepb.Engine_TIDB || engine == storepb.Engine_OCEANBASE
	shapes := groupStatementsByShape(changeSummary.DMLStatements, mysqlFamily)
	sampled := 0
	var failures []error
	if explainCalculator != nil {
		// Each round estimates the next statement of every shape, so the samples cover as many
		// shapes as they can before sampling a shape twice.
		for round := 0; sampled < common.MaximumLintExplainSize; round++ {
			sampledInRound := false
			for _, shape := range shapes {
				if round >= len(shape.statements) || sampled >= common.MaximumLintExplainSize {
					continue
				}
				sampledInRound = true
				sampled++
				count, err := explainCalculator(ctx, shape.statements[round])
				if err != nil {
					slog.Error("failed to calculate affected rows", log.BBError(err))
					failures = append(failures, err)
					shape.failed++
					continue
				}
				shape.rows = common.AddRows(shape.rows, count)
				shape.estimated++
			}
			if !sampledInRound {
				break
			}
		}
	}

	var dmlRows, estimatedRows int64
	estimated := 0
	// The unestimated statements of a sampled shape count as that shape's average estimate; the
	// statements of other shapes, and DML statements without text, count as the overall average and
	// are reported as not estimated.
	unestimated := max(changeSummary.DMLCount-len(changeSummary.DMLStatements), 0)
	for _, shape := range shapes {
		estimatedRows = common.AddRows(estimatedRows, shape.rows)
		estimated += shape.estimated
		switch {
		case shape.estimated == len(shape.statements):
			dmlRows = common.AddRows(dmlRows, shape.rows)
		case shape.estimated > 0:
			dmlRows = common.AddRows(dmlRows, common.RoundRows(float64(shape.rows)/float64(shape.estimated)*float64(len(shape.statements))))
		default:
			unestimated += len(shape.statements)
		}
	}
	if estimated > 0 && unestimated > 0 {
		dmlRows = common.AddRows(dmlRows, common.RoundRows(float64(estimatedRows)/float64(estimated)*float64(unestimated)))
	}
	totalAffectedRows := common.AddRows(dmlRows, int64(changeSummary.InsertCount))
	totalAffectedRows = common.AddRows(totalAffectedRows, changeSummary.ChangedResources.CountAffectedTableRows())

	// A failed sample of a shape with other estimates counts as that shape's average, but is still
	// not estimated.
	notEstimated := unestimated
	for _, shape := range shapes {
		if shape.estimated > 0 {
			notEstimated += shape.failed
		}
	}
	var warning string
	switch {
	case len(failures) > 0:
		warning = fmt.Sprintf("Affected rows could not be estimated for %d of %d DML statements: %v", notEstimated, changeSummary.DMLCount, failures[0])
	case notEstimated > 0:
		warning = fmt.Sprintf("Affected rows could not be estimated for %d of %d DML statements.", notEstimated, changeSummary.DMLCount)
	default:
	}
	return totalAffectedRows, warning
}

// statementShape is the DML statements that share a shape, with the estimates of those sampled.
type statementShape struct {
	statements []string
	estimated  int
	failed     int
	rows       int64
}

// groupStatementsByShape groups statements that differ only in literal values or spacing, in the
// order each shape first appears.
func groupStatementsByShape(statements []string, mysqlFamily bool) []*statementShape {
	var shapes []*statementShape
	byKey := map[string]*statementShape{}
	for _, statement := range statements {
		key := shapeKey(statement, mysqlFamily)
		shape, ok := byKey[key]
		if !ok {
			shape = &statementShape{}
			byKey[key] = shape
			shapes = append(shapes, shape)
		}
		shape.statements = append(shape.statements, statement)
	}
	return shapes
}

// shapeKey replaces the string and numeric literals of a statement with ? and drops comments and
// whitespace that does not separate words. Everything else keeps its text, including identifiers,
// their case, double-quoted text, which MySQL's ANSI_QUOTES makes an identifier, and the hints and
// executable comments that isKeptComment lists. A statement with text the scan does not follow, such
// as an unclosed quote, an Oracle q'{...}' string, a dollar-quoted string, a nested comment, or a
// quote in brackets, is its own key, because a quote in that text could make the scan read the
// statement's clauses as a literal. mysqlFamily applies the MySQL, MariaDB, TiDB, and OceanBase rules
// that a backslash escapes the next character of a quoted string, that # starts a line comment, and
// that -- starts one only before whitespace or a control character.
func shapeKey(statement string, mysqlFamily bool) string {
	var b strings.Builder
	space := false
	// Whitespace is kept between words, where ? counts as a word, and in - - so that it never reads
	// as a comment marker.
	isWord := func(c byte) bool { return c == '?' || isWordByte(c) }
	write := func(c byte) {
		if space && b.Len() > 0 {
			if last := b.String()[b.Len()-1]; (isWord(last) && isWord(c)) || (last == '-' && c == '-') {
				b.WriteByte(' ')
			}
		}
		space = false
		b.WriteByte(c)
	}
	for i := 0; i < len(statement); {
		c := statement[i]
		switch {
		case c == '\'':
			word := wordBefore(statement, i)
			if strings.EqualFold(word, "q") || strings.EqualFold(word, "nq") {
				return statement
			}
			// PostgreSQL E'...' and CockroachDB b'...' strings escape with backslashes too.
			end, closed := quotedEnd(statement, i, mysqlFamily || strings.EqualFold(word, "e") || strings.EqualFold(word, "b"))
			if !closed {
				return statement
			}
			i = end
			write('?')
		case c == '$' && !mysqlFamily && (i == 0 || !isWordByte(statement[i-1])) && isDollarQuoteStart(statement[i:]):
			return statement
		case c == '[':
			// A SQL Server bracketed identifier, where ]] continues it, or an array subscript, keeps its text.
			end := i + 1
			for end < len(statement) && (statement[end] != ']' || (end+1 < len(statement) && statement[end+1] == ']')) {
				if statement[end] == ']' {
					end++
				}
				end++
			}
			if end >= len(statement) || strings.ContainsAny(statement[i+1:end], `'"`) {
				return statement
			}
			write(c)
			b.WriteString(statement[i+1 : end+1])
			i = end + 1
		case c == '"' || c == '`':
			end, closed := quotedEnd(statement, i, mysqlFamily && c == '"')
			if !closed {
				return statement
			}
			write(c)
			b.WriteString(statement[i+1 : end])
			i = end
		case (strings.HasPrefix(statement[i:], "--") && (!mysqlFamily || i+2 == len(statement) || statement[i+2] <= ' ')) ||
			strings.HasPrefix(statement[i:], "/*") || (mysqlFamily && c == '#'):
			end := commentEnd(statement, i)
			if c == '/' && strings.Contains(statement[i+2:end], "/*") {
				return statement
			}
			if isKeptComment(statement[i:end]) {
				write(c)
				b.WriteString(statement[i+1 : end])
			} else {
				space = true
			}
			i = end
		case c >= '0' && c <= '9' && (i == 0 || !isWordByte(statement[i-1])):
			if end := numberEnd(statement, i, mysqlFamily); end >= 0 {
				i = end
				write('?')
				break
			}
			// A word that starts with digits but is not a number, such as [2024_orders], is an identifier.
			for i < len(statement) && isWordByte(statement[i]) {
				write(statement[i])
				i++
			}
		case c == ' ' || c == '\t' || c == '\n' || c == '\r' || c == '\f' || c == '\v':
			space = true
			i++
		default:
			write(c)
			i++
		}
	}
	return b.String()
}

// wordBefore returns the word that ends at offset end of statement.
func wordBefore(statement string, end int) string {
	start := end
	for start > 0 && isWordByte(statement[start-1]) {
		start--
	}
	return statement[start:end]
}

// isDollarQuoteStart reports whether text starts with a dollar-quote delimiter, such as $$ or $tag$.
func isDollarQuoteStart(text string) bool {
	end := 1
	for end < len(text) && text[end] != '$' && isWordByte(text[end]) && (end > 1 || text[end] < '0' || text[end] > '9') {
		end++
	}
	return end < len(text) && text[end] == '$'
}

// isKeptComment reports whether a comment changes how a statement runs: an optimizer hint, such as
// /*+ ... */ or Oracle's --+ ..., or a MySQL, MariaDB, or TiDB executable comment, such as
// /*!80000 ... */, /*M!100100 ... */, or /*T![feature] ... */.
func isKeptComment(comment string) bool {
	for _, prefix := range []string{"/*+", "--+", "/*!", "/*M!", "/*T!"} {
		if strings.HasPrefix(comment, prefix) {
			return true
		}
	}
	return false
}

// commentEnd returns the offset just past the comment that starts at start: past the newline that
// ends a -- or # comment, which keeps the next line out of the comment's copied text, or past the */
// that closes a /* comment, or the end of the statement when it does not close.
func commentEnd(statement string, start int) int {
	if statement[start] != '/' {
		if end := strings.IndexByte(statement[start:], '\n'); end >= 0 {
			return start + end + 1
		}
		return len(statement)
	}
	if end := strings.Index(statement[start+2:], "*/"); end >= 0 {
		return start + 2 + end + 2
	}
	return len(statement)
}

// numberEnd returns the offset just past the number that starts at start, or -1 when the word there
// is not a number: a decimal, or a 0x hexadecimal or 0b binary integer. Outside mysqlFamily, where
// words such as 0X1F, 0o17, and 1_000 are identifiers, it also reads uppercase prefixes, PostgreSQL's
// 0o octal integers, and _ between digits.
func numberEnd(statement string, start int, mysqlFamily bool) int {
	isDigit := func(i int, digits string) bool {
		return i < len(statement) && (strings.IndexByte(digits, statement[i]) >= 0 || (!mysqlFamily && statement[i] == '_'))
	}
	prefixDigits := ""
	if start+1 < len(statement) && statement[start] == '0' {
		switch c := statement[start+1]; {
		case c == 'x' || (c == 'X' && !mysqlFamily):
			prefixDigits = "0123456789abcdefABCDEF"
		case c == 'b' || (c == 'B' && !mysqlFamily):
			prefixDigits = "01"
		case (c == 'o' || c == 'O') && !mysqlFamily:
			prefixDigits = "01234567"
		default:
		}
	}
	i := start
	if prefixDigits != "" {
		i += 2
		for isDigit(i, prefixDigits) {
			i++
		}
		if i == start+2 {
			return -1
		}
	} else {
		for isDigit(i, "0123456789.") {
			i++
		}
		if i < len(statement) && (statement[i] == 'e' || statement[i] == 'E') {
			exponent := i + 1
			if exponent < len(statement) && (statement[exponent] == '+' || statement[exponent] == '-') {
				exponent++
			}
			if isDigit(exponent, "0123456789") {
				i = exponent
				for isDigit(i, "0123456789") {
					i++
				}
			}
		}
	}
	if i < len(statement) && isWordByte(statement[i]) {
		return -1
	}
	return i
}

// quotedEnd returns the offset just past the quoted text that starts at start, where a doubled
// quote continues the text, and so does an escaped quote with backslashEscapes. It reports false
// when the text never closes.
func quotedEnd(statement string, start int, backslashEscapes bool) (int, bool) {
	quote := statement[start]
	for i := start + 1; i < len(statement); i++ {
		switch {
		case backslashEscapes && statement[i] == '\\':
			i++
		case statement[i] != quote:
		case i+1 < len(statement) && statement[i+1] == quote:
			i++
		default:
			return i + 1, true
		}
	}
	return len(statement), false
}

func isWordByte(c byte) bool {
	return c == '_' || c == '$' || c >= 0x80 || ('0' <= c && c <= '9') || ('a' <= c && c <= 'z') || ('A' <= c && c <= 'Z')
}
