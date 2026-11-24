// Package util provides the library for database drivers.
//
//nolint:revive
package util

import (
	"fmt"
	"strings"
	"time"
	"unicode"

	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	omnidriver "github.com/bytebase/omni/driver"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/plugin/db"
	"github.com/bytebase/bytebase/backend/plugin/db/transaction"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

// QueryResultOptions customizes how OmniResultToQueryResultWith converts an
// omni/driver.Result into an API QueryResult.
type QueryResultOptions struct {
	// ClearExplainPlan clears QueryPlan so the API layer stamps the requested
	// format for explain requests.
	ClearExplainPlan bool
	// ClearExplainPlanOnError also clears QueryPlan when result.Err != nil.
	ClearExplainPlanOnError bool
	// TypedExplainPlan builds QueryPlan using db.TypedExplainPlan instead of
	// exact uppercase enum lookup.
	TypedExplainPlan bool
	// IncludePlanContent preserves result.Plan.Content when TypedExplainPlan is true.
	IncludePlanContent bool
	// SkipPlan omits QueryPlan completely.
	SkipPlan bool
	// SkipMessages omits result.Messages.
	SkipMessages bool
	// UppercaseColumnTypes uppercases each ColumnTypeName.
	UppercaseColumnTypes bool
	// SkipColumnTypes leaves ColumnTypeNames empty.
	SkipColumnTypes bool
	// TruncatedWins checks result.Truncated before result.Err when setting Error.
	TruncatedWins bool
	// FillZeroAffectedRows populates one "Affected Rows: 0" row for a
	// columnless, error-free result whose AffectedRows is nil.
	FillZeroAffectedRows bool
	// Limit, when > 0, caps the number of converted rows.
	Limit int
	// FormatOutOfRangeTimestamp, when non-nil, converts Timestamp and
	// TimestampTZ cells outside google.protobuf.Timestamp's valid [0001..9999]
	// range into StringValue cells instead of invalid protobuf timestamps.
	FormatOutOfRangeTimestamp func(t time.Time, withTZ bool) string
	// ConvertValue optionally overrides cell conversion; when nil, OmniRowValue is used.
	ConvertValue func(omnidriver.Value) *v1pb.RowValue
}

// OmniResultsToQueryResultsWith converts a slice of omni driver results into API
// QueryResults according to opts.
func OmniResultsToQueryResultsWith(results []*omnidriver.Result, maxResultBytes int64, opts QueryResultOptions) []*v1pb.QueryResult {
	if len(results) == 0 {
		return nil
	}
	converted := make([]*v1pb.QueryResult, 0, len(results))
	for _, result := range results {
		converted = append(converted, OmniResultToQueryResultWith(result, maxResultBytes, opts))
	}
	return converted
}

// fillZeroAffectedRows populates one "Affected Rows: 0" row when the contract
// returned a columnless, error-free result with nil AffectedRows.
func fillZeroAffectedRows(result *omnidriver.Result, queryResult *v1pb.QueryResult) {
	if result == nil || queryResult == nil {
		return
	}
	if len(result.Columns) == 0 && result.AffectedRows == nil && result.Err == nil && !result.Truncated {
		affected := buildAffectedRowsResult(0, queryResult.Messages)
		queryResult.ColumnNames = affected.ColumnNames
		queryResult.ColumnTypeNames = affected.ColumnTypeNames
		queryResult.Rows = affected.Rows
		queryResult.RowsCount = 1
	}
}

// OmniExplainFormat maps an API ExplainFormat enum onto the contract's
// uppercase format name when it is in supported, or returns fallback.
func OmniExplainFormat(format v1pb.QueryOption_ExplainFormat, fallback string, supported ...v1pb.QueryOption_ExplainFormat) string {
	for _, s := range supported {
		if format == s {
			return format.String()
		}
	}
	return fallback
}

// omniInstanceToMetadata converts an omni/driver.Instance into db.InstanceMetadata.
// When withInstanceMetadata is true, Metadata is populated with LowerCaseTableNames
// and Roles.
func omniInstanceToMetadata(instance *omnidriver.Instance, withInstanceMetadata bool) *db.InstanceMetadata {
	if instance == nil {
		return nil
	}
	meta := &db.InstanceMetadata{
		Version:   instance.Version,
		Databases: instance.Databases,
	}
	if !withInstanceMetadata {
		return meta
	}
	var roles []*storepb.InstanceRole
	if len(instance.Roles) > 0 {
		roles = make([]*storepb.InstanceRole, 0, len(instance.Roles))
		for _, role := range instance.Roles {
			roles = append(roles, &storepb.InstanceRole{
				Name:            role.Name,
				ConnectionLimit: role.ConnectionLimit,
				ValidUntil:      role.ValidUntil,
				Attribute:       role.Attribute,
			})
		}
	}
	meta.Metadata = &storepb.Instance{
		MysqlLowerCaseTableNames: instance.LowerCaseTableNames,
		Roles:                    roles,
	}
	return meta
}

// parseTransactionMode strips any leading transaction directive from script and
// returns the resolved omni transaction mode, isolation level, and cleaned script.
func parseTransactionMode(script string) (omnidriver.TransactionMode, omnidriver.IsolationLevel, string) {
	config, cleaned := base.ParseTransactionConfig(script)
	mode := config.Mode
	if mode == transaction.ModeUnspecified {
		mode = transaction.DefaultMode()
	}
	txMode := omnidriver.Transactional
	if mode == transaction.ModeOff {
		txMode = omnidriver.AutoCommit
	}
	return txMode, omnidriver.IsolationLevel(config.Isolation), cleaned
}

// ExecuteLoggerOption configures an ExecuteLogger.
type ExecuteLoggerOption func(*ExecuteLogger)

// WithCommandRange customizes how a command's byte range is mapped into a
// TaskRunLog Range.
func WithCommandRange(fn func(omnidriver.Command) *storepb.Range) ExecuteLoggerOption {
	return func(l *ExecuteLogger) {
		l.commandRange = fn
	}
}

// WithOmitPerResultAffectedRows omits per-statement affected-row counts from
// CommandResponse log entries while keeping total affectedRows.
func WithOmitPerResultAffectedRows() ExecuteLoggerOption {
	return func(l *ExecuteLogger) {
		l.omitPerResultAffectedRows = true
	}
}

// WithSkipCommandLogs suppresses CommandExecute and CommandResponse log entries
// so only transaction control and retry events are recorded.
func WithSkipCommandLogs() ExecuteLoggerOption {
	return func(l *ExecuteLogger) {
		l.skipCommandLogs = true
	}
}

// ExecuteLogger bridges omni/driver.ExecuteLogger onto Bytebase's task run log.
type ExecuteLogger struct {
	opts                      db.ExecuteOptions
	commandRange              func(omnidriver.Command) *storepb.Range
	omitPerResultAffectedRows bool
	skipCommandLogs           bool
}

// NewExecuteLogger creates an ExecuteLogger that reports execution steps to opts.
func NewExecuteLogger(opts db.ExecuteOptions, options ...ExecuteLoggerOption) *ExecuteLogger {
	l := &ExecuteLogger{opts: opts}
	for _, option := range options {
		option(l)
	}
	return l
}

func (l *ExecuteLogger) CommandExecute(command omnidriver.Command) {
	if l.skipCommandLogs {
		return
	}
	var commandRange *storepb.Range
	if l.commandRange != nil {
		commandRange = l.commandRange(command)
	} else {
		commandRange = &storepb.Range{
			Start: int32(command.ByteStart),
			End:   int32(command.ByteEnd),
		}
	}
	l.opts.LogCommandExecute(commandRange, command.Text)
}

func (l *ExecuteLogger) CommandResponse(affectedRows int64, perResult []int64, err error) {
	if l.skipCommandLogs {
		return
	}
	if l.omitPerResultAffectedRows {
		perResult = nil
	}
	l.opts.LogCommandResponse(affectedRows, perResult, errorString(err))
}

func (l *ExecuteLogger) TransactionControl(op omnidriver.TransactionOp, err error) {
	var control storepb.TaskRunLog_TransactionControl_Type
	switch op {
	case omnidriver.Begin:
		control = storepb.TaskRunLog_TransactionControl_BEGIN
	case omnidriver.Commit:
		control = storepb.TaskRunLog_TransactionControl_COMMIT
	case omnidriver.Rollback:
		control = storepb.TaskRunLog_TransactionControl_ROLLBACK
	default:
		return
	}
	l.opts.LogTransactionControl(control, errorString(err))
}

func (l *ExecuteLogger) Retry(err error, attempt, _ int) {
	l.opts.LogRetryInfo(err, attempt)
}

func errorString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

// DirectiveShift maps the byte range a driver reports on cleaned back to the
// caller's script, dropping directive lines removed by ParseTransactionConfig.
func DirectiveShift(script, cleaned string) func(int, int) (int, int) {
	identity := func(start, end int) (int, int) { return start, end }
	if cleaned == script {
		return identity
	}
	scriptLines := strings.Split(script, "\n")
	scriptStarts := make([]int, len(scriptLines))
	cursor := 0
	for i, line := range scriptLines {
		scriptStarts[i] = cursor
		cursor += len(line) + 1
	}
	cleanedLines := strings.Split(cleaned, "\n")
	cleanedStarts := make([]int, len(cleanedLines))
	sourceStarts := make([]int, len(cleanedLines))
	cursor = 0
	for i, line := range cleanedLines {
		cleanedStarts[i] = cursor
		cursor += len(line) + 1
	}
	kept := make([]bool, len(scriptLines))
	next := 0
	for i, line := range cleanedLines {
		first, last := i == 0, i == len(cleanedLines)-1
		for next < len(scriptLines) && !sameLine(scriptLines[next], line, first, last) {
			next++
		}
		if next == len(scriptLines) {
			return identity
		}
		kept[next] = true
		sourceStarts[i] = scriptStarts[next]
		if i == 0 {
			sourceStarts[i] += len(scriptLines[next]) - len(strings.TrimLeftFunc(scriptLines[next], unicode.IsSpace))
		}
		next++
	}
	var removed [][2]int
	for i, line := range scriptLines {
		if !kept[i] {
			removed = append(removed, [2]int{scriptStarts[i], scriptStarts[i] + len(line) + 1})
		}
	}
	sourceOffset := func(at int) int {
		if at < 0 || at > len(cleaned) {
			return at
		}
		i := 0
		for i+1 < len(cleanedStarts) && cleanedStarts[i+1] <= at {
			i++
		}
		return sourceStarts[i] + (at - cleanedStarts[i])
	}
	return func(start, end int) (int, int) {
		sourceStart, sourceEnd := sourceOffset(start), sourceOffset(end)
		for _, span := range removed {
			if sourceStart < span[1] && sourceEnd > span[0] && span[1] > sourceStart {
				sourceStart = span[1]
			}
		}
		if sourceStart > sourceEnd {
			sourceStart = sourceEnd
		}
		return sourceStart, sourceEnd
	}
}

// sameLine reports whether a cleaned line matches the caller's line.
func sameLine(scriptLine, cleanedLine string, first, last bool) bool {
	candidate := scriptLine
	if first {
		candidate = strings.TrimLeftFunc(candidate, unicode.IsSpace)
	}
	if last {
		candidate = strings.TrimRightFunc(candidate, unicode.IsSpace)
	}
	return candidate == cleanedLine
}

// OmniResultToQueryResult converts one result of the driver contract
// (github.com/bytebase/omni/driver) into the API's QueryResult. Every adapter
// over an omni driver returns its results through this, so the conversion lives
// here rather than in each engine's package.
//
// maxResultBytes is the cap the caller passed as QueryOptions.MaxResultBytes;
// the contract stops reading at it and sets Truncated, and the message the API
// shows for that cap is the caller's to format.
func OmniResultToQueryResult(result *omnidriver.Result, maxResultBytes int64) *v1pb.QueryResult {
	return OmniResultToQueryResultWith(result, maxResultBytes, QueryResultOptions{})
}

// OmniResultToQueryResultWith converts one omni driver result into an API
// QueryResult according to opts.
func OmniResultToQueryResultWith(result *omnidriver.Result, maxResultBytes int64, opts QueryResultOptions) *v1pb.QueryResult {
	if result == nil {
		return &v1pb.QueryResult{}
	}
	out := &v1pb.QueryResult{
		Statement: result.Statement,
		Latency:   durationpb.New(result.Latency),
	}
	if len(result.Columns) > 0 {
		out.ColumnNames = make([]string, len(result.Columns))
		if !opts.SkipColumnTypes {
			out.ColumnTypeNames = make([]string, len(result.Columns))
		}
		for i, column := range result.Columns {
			out.ColumnNames[i] = column.Name
			if !opts.SkipColumnTypes {
				typeName := column.TypeName
				if opts.UppercaseColumnTypes {
					typeName = strings.ToUpper(typeName)
				}
				out.ColumnTypeNames[i] = typeName
			}
		}
	}
	rows := result.Rows
	if opts.Limit > 0 && len(rows) > opts.Limit {
		rows = rows[:opts.Limit]
	}
	convertValue := opts.ConvertValue
	if convertValue == nil {
		convertValue = OmniRowValue
	}
	for _, row := range rows {
		values := make([]*v1pb.RowValue, len(row))
		for i, value := range row {
			if opts.FormatOutOfRangeTimestamp != nil &&
				(value.Kind == omnidriver.Timestamp || value.Kind == omnidriver.TimestampTZ) &&
				timestamppb.New(value.Time).CheckValid() != nil {
				values[i] = &v1pb.RowValue{
					Kind: &v1pb.RowValue_StringValue{
						StringValue: opts.FormatOutOfRangeTimestamp(value.Time, value.Kind == omnidriver.TimestampTZ),
					},
				}
				continue
			}
			values[i] = convertValue(value)
		}
		out.Rows = append(out.Rows, &v1pb.QueryRow{Values: values})
	}
	// A statement that returns no columns reports a count instead, which the API
	// shows as one "Affected Rows" row. The contract leaves the count nil for a
	// statement the engine states none for, which is not the same as zero.
	if len(result.Columns) == 0 && result.AffectedRows != nil {
		affected := buildAffectedRowsResult(*result.AffectedRows, nil)
		affected.Statement = out.Statement
		affected.Latency = out.Latency
		out = affected
	}
	if !opts.SkipMessages {
		for _, message := range result.Messages {
			out.Messages = append(out.Messages, &v1pb.QueryResult_Message{
				Level:   v1pb.QueryResult_Message_Level(v1pb.QueryResult_Message_Level_value[message.Level]),
				Content: message.Content,
			})
		}
	}
	clearPlan := opts.ClearExplainPlan || (opts.ClearExplainPlanOnError && result.Err != nil)
	if !opts.SkipPlan && !clearPlan && result.Plan != nil {
		if opts.TypedExplainPlan {
			out.QueryPlan = db.TypedExplainPlan(result.Plan.Format, result.Plan.Executed, true)
			if opts.IncludePlanContent {
				out.QueryPlan.Content = result.Plan.Content
			}
		} else {
			out.QueryPlan = &v1pb.QueryResult_QueryPlan{
				Format:   v1pb.QueryOption_ExplainFormat(v1pb.QueryOption_ExplainFormat_value[result.Plan.Format]),
				Executed: result.Plan.Executed,
				Content:  result.Plan.Content,
			}
		}
	}
	if opts.TruncatedWins {
		if result.Truncated {
			out.Error = FormatMaximumSQLResultSizeMessage(maxResultBytes)
		} else if result.Err != nil {
			out.Error = result.Err.Error()
		}
	} else {
		if result.Err != nil {
			out.Error = result.Err.Error()
		} else if result.Truncated {
			out.Error = FormatMaximumSQLResultSizeMessage(maxResultBytes)
		}
	}
	out.RowsCount = int64(len(out.Rows))
	if opts.FillZeroAffectedRows {
		fillZeroAffectedRows(result, out)
	}
	return out
}

// FormatDefaultTimestamp formats an out-of-range timestamp as RFC3339Nano when
// withTZ is true, or UTC "2006-01-02 15:04:05.999999999" otherwise.
func FormatDefaultTimestamp(t time.Time, withTZ bool) string {
	if withTZ {
		return t.Format(time.RFC3339Nano)
	}
	return t.UTC().Format("2006-01-02 15:04:05.999999999")
}

// OmniQueryOptions maps db.QueryContext onto omnidriver.QueryOptions.
// Set includeSchema to forward queryContext.Schema, and explainFormat for
// engines that accept an explicit explain format name.
func OmniQueryOptions(queryContext db.QueryContext, includeSchema bool, explainFormat string) omnidriver.QueryOptions {
	opts := omnidriver.QueryOptions{
		Limit:          queryContext.Limit,
		Explain:        queryContext.Explain,
		ExplainFormat:  explainFormat,
		MaxResultBytes: queryContext.MaximumSQLResultSize,
		MaskingEnabled: queryContext.MaskingEnabled,
	}
	if includeSchema {
		opts.Schema = queryContext.Schema
	}
	return opts
}

// MySQLForkQueryOptions maps a db.QueryContext onto omnidriver.QueryOptions for
// MySQL-protocol analytical forks (Doris and StarRocks).
func MySQLForkQueryOptions(queryContext db.QueryContext) omnidriver.QueryOptions {
	return OmniQueryOptions(queryContext, false, "")
}

// MySQLForkQueryResultOptions returns QueryResultOptions for Doris and
// StarRocks adapters.
func MySQLForkQueryResultOptions() QueryResultOptions {
	return QueryResultOptions{
		SkipPlan:     true,
		SkipMessages: true,
		ConvertValue: MySQLForkRowValue,
	}
}

// MySQLForkRowValue converts a driver cell for Doris and StarRocks: empty byte
// slices become NULL, Float32 widens to DoubleValue, Int32 widens to
// Int64Value, and unsupported kinds become NULL.
func MySQLForkRowValue(value omnidriver.Value) *v1pb.RowValue {
	switch value.Kind {
	case omnidriver.Bytes:
		if len(value.Bytes) == 0 {
			return NullRowValue
		}
		return &v1pb.RowValue{Kind: &v1pb.RowValue_BytesValue{BytesValue: value.Bytes}}
	case omnidriver.Float32:
		return &v1pb.RowValue{Kind: &v1pb.RowValue_DoubleValue{DoubleValue: value.Float64}}
	case omnidriver.Int32:
		return &v1pb.RowValue{Kind: &v1pb.RowValue_Int64Value{Int64Value: value.Int64}}
	case omnidriver.Bool, omnidriver.Float64, omnidriver.Int64, omnidriver.String, omnidriver.Timestamp:
		return OmniRowValue(value)
	default:
		return NullRowValue
	}
}

// OmniRowValue converts one cell of a contract result into the API's RowValue.
func OmniRowValue(value omnidriver.Value) *v1pb.RowValue {
	switch value.Kind {
	case omnidriver.Bool:
		return &v1pb.RowValue{Kind: &v1pb.RowValue_BoolValue{BoolValue: value.Bool}}
	case omnidriver.Bytes:
		return &v1pb.RowValue{Kind: &v1pb.RowValue_BytesValue{BytesValue: value.Bytes}}
	case omnidriver.Float32:
		return &v1pb.RowValue{Kind: &v1pb.RowValue_FloatValue{FloatValue: float32(value.Float64)}}
	case omnidriver.Float64:
		return &v1pb.RowValue{Kind: &v1pb.RowValue_DoubleValue{DoubleValue: value.Float64}}
	case omnidriver.Int32:
		return &v1pb.RowValue{Kind: &v1pb.RowValue_Int32Value{Int32Value: int32(value.Int64)}}
	case omnidriver.Int64:
		return &v1pb.RowValue{Kind: &v1pb.RowValue_Int64Value{Int64Value: value.Int64}}
	case omnidriver.Uint32:
		return &v1pb.RowValue{Kind: &v1pb.RowValue_Uint32Value{Uint32Value: uint32(value.Uint64)}}
	case omnidriver.Uint64:
		return &v1pb.RowValue{Kind: &v1pb.RowValue_Uint64Value{Uint64Value: value.Uint64}}
	case omnidriver.String:
		return &v1pb.RowValue{Kind: &v1pb.RowValue_StringValue{StringValue: value.String}}
	case omnidriver.Structured:
		structured, err := structpb.NewValue(value.Structured)
		if err != nil {
			// A value the JSON model cannot hold is reported rather than dropped.
			return &v1pb.RowValue{Kind: &v1pb.RowValue_StringValue{StringValue: fmt.Sprintf("%v", value.Structured)}}
		}
		return &v1pb.RowValue{Kind: &v1pb.RowValue_ValueValue{ValueValue: structured}}
	case omnidriver.Timestamp:
		return &v1pb.RowValue{Kind: &v1pb.RowValue_TimestampValue{TimestampValue: &v1pb.RowValue_Timestamp{
			GoogleTimestamp: timestamppb.New(value.Time),
			Accuracy:        value.Precision,
		}}}
	case omnidriver.TimestampTZ:
		zone, offset := value.Time.Zone()
		return &v1pb.RowValue{Kind: &v1pb.RowValue_TimestampTzValue{TimestampTzValue: &v1pb.RowValue_TimestampTZ{
			GoogleTimestamp: timestamppb.New(value.Time),
			Zone:            zone,
			Offset:          int32(offset),
			Accuracy:        value.Precision,
		}}}
	default:
		return NullRowValue
	}
}

func buildAffectedRowsResult(affectedRows int64, messages []*v1pb.QueryResult_Message) *v1pb.QueryResult {
	return &v1pb.QueryResult{
		ColumnNames:     []string{"Affected Rows"},
		ColumnTypeNames: []string{"INT"},
		Rows: []*v1pb.QueryRow{
			{
				Values: []*v1pb.RowValue{
					{
						Kind: &v1pb.RowValue_Int64Value{
							Int64Value: affectedRows,
						},
					},
				},
			},
		},
		Messages: messages,
	}
}

var NullRowValue = &v1pb.RowValue{
	Kind: &v1pb.RowValue_NullValue{
		NullValue: structpb.NullValue_NULL_VALUE,
	},
}

func FormatMaximumSQLResultSizeMessage(limit int64) string {
	return fmt.Sprintf("Output of query exceeds max allowed output size of %dMB", limit/1024/1024)
}
