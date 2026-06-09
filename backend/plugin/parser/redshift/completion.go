package redshift

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"

	redshiftcatalog "github.com/bytebase/omni/redshift/catalog"
	redshiftcompletion "github.com/bytebase/omni/redshift/completion"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

func init() {
	base.RegisterCompleteFunc(storepb.Engine_REDSHIFT, Completion)
}

// Completion returns Redshift auto-completion candidates backed by omni's
// parser-native completer. Bytebase metadata is copied into an omni Redshift
// catalog with minimal DDL so the completer can resolve schemas, relations,
// columns, sequences, and Redshift-specific grammar slots.
func Completion(ctx context.Context, cCtx base.CompletionContext, statement string, caretLine int, caretOffset int) ([]base.Candidate, error) {
	cat := buildCompletionCatalog(ctx, cCtx)

	sql, line, offset := completionStatementAtCaret(statement, caretLine, caretOffset)
	byteOffset := caretToByteOffset(sql, line, offset)
	omniCandidates := redshiftcompletion.Complete(sql, byteOffset, cat)

	result := make([]base.Candidate, 0, len(omniCandidates))
	seen := make(map[string]bool, len(omniCandidates))
	for _, c := range omniCandidates {
		candidate := base.Candidate{
			Text:       completionCandidateText(c),
			Type:       convertCandidateType(c.Type),
			Definition: c.Definition,
			Comment:    c.Comment,
		}
		key := candidate.String()
		if seen[key] {
			continue
		}
		seen[key] = true
		result = append(result, candidate)
	}
	slices.SortFunc(result, func(a, b base.Candidate) int {
		return strings.Compare(a.String(), b.String())
	})
	return result, nil
}

// completionStatementAtCaret drops statements before the caret so table refs
// from earlier statements in the same editor buffer cannot leak into completion.
func completionStatementAtCaret(statement string, caretLine int, caretOffset int) (string, int, int) {
	list, err := SplitSQL(statement)
	if err != nil || len(base.FilterEmptyStatements(list)) <= 1 {
		return statement, caretLine, caretOffset
	}

	caretLineZeroBased := caretLine - 1
	start := 0
	newCaretLine, newCaretOffset := caretLine, caretOffset
	for i, sql := range list {
		sqlEndLine := int(sql.End.GetLine()) - 1
		sqlEndColumn := int(sql.End.GetColumn())
		if sqlEndLine > caretLineZeroBased || (sqlEndLine == caretLineZeroBased && sqlEndColumn >= caretOffset) {
			start = i
			if i == 0 {
				break
			}
			previousEndLine := int(list[i-1].End.GetLine()) - 1
			previousEndColumn := int(list[i-1].End.GetColumn())
			newCaretLine = caretLineZeroBased - previousEndLine + 1
			if caretLineZeroBased == previousEndLine {
				newCaretOffset = caretOffset - previousEndColumn + 1
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

func buildCompletionCatalog(ctx context.Context, cCtx base.CompletionContext) *redshiftcatalog.Catalog {
	if cCtx.Metadata == nil || cCtx.DefaultDatabase == "" {
		return nil
	}

	_, metadata, err := cCtx.Metadata(ctx, cCtx.InstanceID, cCtx.DefaultDatabase)
	if err != nil || metadata == nil {
		return nil
	}

	cat := redshiftcatalog.New()
	for _, schemaName := range metadata.ListSchemaNames() {
		schemaMeta := metadata.GetSchemaMetadata(schemaName)
		if schemaMeta == nil {
			continue
		}
		execCompletionDDL(cat, fmt.Sprintf("CREATE SCHEMA IF NOT EXISTS %s;", quoteIdent(schemaName)))

		for _, tableName := range schemaMeta.ListTableNames() {
			tableMeta := schemaMeta.GetTable(tableName)
			if tableMeta == nil {
				continue
			}
			execCompletionDDL(cat, createTableDDL(schemaName, tableName, tableMeta.GetProto().GetColumns()))
		}
		for _, tableName := range schemaMeta.ListForeignTableNames() {
			tableMeta := schemaMeta.GetExternalTable(tableName)
			if tableMeta == nil {
				continue
			}
			execCompletionDDL(cat, createTableDDL(schemaName, tableName, tableMeta.GetProto().GetColumns()))
		}
		for _, viewName := range schemaMeta.ListViewNames() {
			viewMeta := schemaMeta.GetView(viewName)
			if viewMeta == nil {
				continue
			}
			execCompletionDDL(cat, createViewDDL("VIEW", schemaName, viewName, viewMeta.GetColumns()))
		}
		for _, viewName := range schemaMeta.ListMaterializedViewNames() {
			viewMeta := schemaMeta.GetMaterializedView(viewName)
			if viewMeta == nil {
				continue
			}
			cat.SetSearchPath([]string{schemaName, "public"})
			if viewMeta.GetDefinition() == "" || !execCompletionDDL(cat, createMaterializedViewDDL(schemaName, viewName, viewMeta.GetDefinition())) {
				execCompletionDDL(cat, createViewDDL("MATERIALIZED VIEW", schemaName, viewName, nil))
			}
		}
		for _, sequenceName := range schemaMeta.ListSequenceNames() {
			execCompletionDDL(cat, fmt.Sprintf("CREATE SEQUENCE %s.%s;", quoteIdent(schemaName), quoteIdent(sequenceName)))
		}
	}

	searchPath := []string{"public"}
	if cCtx.DefaultSchema != "" {
		searchPath = []string{cCtx.DefaultSchema, "public"}
	}
	cat.SetSearchPath(searchPath)
	return cat
}

func execCompletionDDL(cat *redshiftcatalog.Catalog, sql string) bool {
	_, err := cat.Exec(sql, nil)
	return err == nil
}

func createTableDDL(schemaName, tableName string, columns []*storepb.ColumnMetadata) string {
	return fmt.Sprintf(
		"CREATE TABLE %s.%s (%s);",
		quoteIdent(schemaName),
		quoteIdent(tableName),
		columnListDDL(columns),
	)
}

func createViewDDL(kind, schemaName, viewName string, columns []*storepb.ColumnMetadata) string {
	selectItems := make([]string, 0, len(columns))
	for _, column := range columns {
		if column == nil || column.GetName() == "" {
			continue
		}
		selectItems = append(selectItems, fmt.Sprintf(
			"CAST(NULL AS %s) AS %s",
			normalizeCompletionType(column.GetType()),
			quoteIdent(column.GetName()),
		))
	}
	if len(selectItems) == 0 {
		selectItems = append(selectItems, "1 AS __bytebase_completion_placeholder")
	}
	return fmt.Sprintf(
		"CREATE %s %s.%s AS SELECT %s;",
		kind,
		quoteIdent(schemaName),
		quoteIdent(viewName),
		strings.Join(selectItems, ", "),
	)
}

func createMaterializedViewDDL(schemaName, viewName, definition string) string {
	definition = strings.TrimSpace(definition)
	definition = strings.TrimSuffix(definition, ";")
	if strings.HasPrefix(strings.ToUpper(definition), "CREATE ") {
		return definition + ";"
	}
	return fmt.Sprintf(
		"CREATE MATERIALIZED VIEW %s.%s AS %s;",
		quoteIdent(schemaName),
		quoteIdent(viewName),
		definition,
	)
}

func columnListDDL(columns []*storepb.ColumnMetadata) string {
	columnDefs := make([]string, 0, len(columns))
	for _, column := range columns {
		if column == nil || column.GetName() == "" {
			continue
		}
		columnDefs = append(columnDefs, fmt.Sprintf("%s %s", quoteIdent(column.GetName()), normalizeCompletionType(column.GetType())))
	}
	if len(columnDefs) == 0 {
		columnDefs = append(columnDefs, "__bytebase_completion_placeholder text")
	}
	return strings.Join(columnDefs, ", ")
}

func normalizeCompletionType(typ string) string {
	lower := strings.ToLower(strings.TrimSpace(typ))
	switch {
	case lower == "":
		return "text"
	case strings.Contains(lower, "bigint"):
		return "bigint"
	case strings.Contains(lower, "int"):
		return "integer"
	case strings.Contains(lower, "bool"):
		return "boolean"
	case strings.Contains(lower, "char"), strings.Contains(lower, "text"), strings.Contains(lower, "string"):
		return "text"
	case strings.Contains(lower, "numeric"), strings.Contains(lower, "decimal"):
		return "numeric"
	case strings.Contains(lower, "double"):
		return "double precision"
	case strings.Contains(lower, "float"), strings.Contains(lower, "real"):
		return "real"
	case strings.Contains(lower, "date"):
		return "date"
	case strings.Contains(lower, "time"):
		return "timestamp"
	default:
		return "text"
	}
}

func quoteIdent(name string) string {
	return `"` + strings.ReplaceAll(name, `"`, `""`) + `"`
}

func completionCandidateText(candidate redshiftcompletion.Candidate) string {
	switch candidate.Type {
	case redshiftcompletion.CandidateKeyword, redshiftcompletion.CandidateFunction:
		return candidate.Text
	default:
		return quotedIdentifierIfNeeded(candidate.Text)
	}
}

func quotedIdentifierIfNeeded(s string) string {
	if strings.ToLower(s) != s || !isValidUnquotedIdentifier(s) {
		return quoteIdent(s)
	}
	return s
}

// isValidUnquotedIdentifier checks if the identifier can be used without quotes in Redshift.
func isValidUnquotedIdentifier(s string) bool {
	if len(s) == 0 {
		return false
	}

	first := rune(s[0])
	if !unicode.IsLetter(first) && first != '_' {
		return false
	}
	for _, ch := range s[1:] {
		if !unicode.IsLetter(ch) && !unicode.IsDigit(ch) && ch != '_' {
			return false
		}
	}
	return true
}

func convertCandidateType(t redshiftcompletion.CandidateType) base.CandidateType {
	switch t {
	case redshiftcompletion.CandidateKeyword:
		return base.CandidateTypeKeyword
	case redshiftcompletion.CandidateSchema:
		return base.CandidateTypeSchema
	case redshiftcompletion.CandidateTable:
		return base.CandidateTypeTable
	case redshiftcompletion.CandidateView:
		return base.CandidateTypeView
	case redshiftcompletion.CandidateMaterializedView:
		return base.CandidateTypeMaterializedView
	case redshiftcompletion.CandidateColumn:
		return base.CandidateTypeColumn
	case redshiftcompletion.CandidateFunction:
		return base.CandidateTypeFunction
	case redshiftcompletion.CandidateSequence:
		return base.CandidateTypeSequence
	case redshiftcompletion.CandidateIndex:
		return base.CandidateTypeIndex
	case redshiftcompletion.CandidateTrigger:
		return base.CandidateTypeTrigger
	default:
		return base.CandidateTypeNone
	}
}

// caretToByteOffset converts a (1-based line, 0-based column) caret position
// into a byte offset into statement. The column is interpreted as a UTF-16
// code-unit offset, matching Monaco/LSP completion positions.
func caretToByteOffset(statement string, caretLine, caretColumn int) int {
	line := 1
	col := 0
	for i := 0; i < len(statement); {
		if line == caretLine && col >= caretColumn {
			return i
		}
		r, size := utf8.DecodeRuneInString(statement[i:])
		if r == '\n' {
			if line == caretLine {
				return i
			}
			line++
			col = 0
		} else if r <= 0xFFFF {
			col++
		} else {
			col += 2
		}
		i += size
	}
	return len(statement)
}
