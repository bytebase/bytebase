package pg

import (
	"context"
	"fmt"
	"strings"

	pgparser "github.com/bytebase/parser/postgresql"

	"github.com/antlr4-go/antlr/v4"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	parsererror "github.com/bytebase/bytebase/backend/plugin/parser/errors"
	"github.com/pkg/errors"
)

// Add this field to querySpanExtractor struct (defined in query_span_extractor.go):
// resultTableSource base.TableSource  // Stores the extracted table source from EnterStmtmulti

// getQuerySpanNew is the ANTLR-based implementation of query span extraction.
func (q *querySpanExtractor) getQuerySpanNew(ctx context.Context, stmt string) (*base.QuerySpan, error) {
	q.ctx = ctx

	// Parse the statement using ANTLR
	parseResult, err := ParsePostgreSQL(stmt)
	if err != nil {
		return nil, err
	}

	// Walk the tree to extract access tables
	antlr.ParseTreeWalkerDefault.Walk(q, parseResult.Tree)
	if q.err != nil {
		return nil, errors.Wrapf(q.err, "failed to extract query span from statement: %s", stmt)
	}

	// Build access map
	accessesMap := make(base.SourceColumnSet)
	for _, resource := range q.accessTables {
		accessesMap[resource] = true
	}

	// Check for mixed system/user tables
	allSystems, mixed := isMixedQuery(accessesMap)
	if mixed {
		return nil, base.MixUserSystemTablesError
	}

	// Get query type using ANTLR-based detection
	queryTypeListener := &queryTypeListener{
		result:     base.QueryTypeUnknown,
		allSystems: allSystems,
	}
	antlr.ParseTreeWalkerDefault.Walk(queryTypeListener, parseResult.Tree)
	if queryTypeListener.err != nil {
		return nil, errors.Wrapf(queryTypeListener.err, "failed to get query type from statement: %s", stmt)
	}

	queryType, isExplainAnalyze := queryTypeListener.result, queryTypeListener.isExplainAnalyze

	// For non-SELECT queries, return early
	if queryType != base.Select {
		return &base.QuerySpan{
			Type:          queryType,
			SourceColumns: base.SourceColumnSet{},
			Results:       []base.QuerySpanResult{},
		}, nil
	}

	// For EXPLAIN ANALYZE SELECT
	if isExplainAnalyze {
		return &base.QuerySpan{
			Type:          queryType,
			SourceColumns: accessesMap,
			Results:       []base.QuerySpanResult{},
		}, nil
	}

	// Extract table source using ANTLR (this was done in EnterStmtmulti)
	// The resultTableSource should have been populated by EnterStmtmulti
	if q.resultTableSource == nil {
		// If not populated, try to extract it now
		root := parseResult.Tree.(*pgparser.RootContext)
		if root.Stmtblock() != nil && root.Stmtblock().Stmtmulti() != nil {
			stmtmulti := root.Stmtblock().Stmtmulti()
			if len(stmtmulti.AllStmt()) == 1 {
				tableSource, err := q.extractTableSourceFromStmt(stmtmulti.AllStmt()[0])
				if err != nil {
					var functionNotSupported *parsererror.FunctionNotSupportedError
					if errors.As(err, &functionNotSupported) {
						if len(accessesMap) == 0 {
							accessesMap[base.ColumnResource{
								Database: q.defaultDatabase,
							}] = true
						}
						return &base.QuerySpan{
							Type:          base.Select,
							SourceColumns: accessesMap,
							Results:       []base.QuerySpanResult{},
							NotFoundError: functionNotSupported,
						}, nil
					}
					var resourceNotFound *parsererror.ResourceNotFoundError
					if errors.As(err, &resourceNotFound) {
						if len(accessesMap) == 0 {
							accessesMap[base.ColumnResource{
								Database: q.defaultDatabase,
							}] = true
						}
						return &base.QuerySpan{
							Type:          base.Select,
							SourceColumns: accessesMap,
							Results:       []base.QuerySpanResult{},
							NotFoundError: resourceNotFound,
						}, nil
					}
					return nil, err
				}
				q.resultTableSource = tableSource
			}
		}
	}

	// Merge the source columns in functions to the access tables
	for source := range q.sourceColumnsInFunction {
		accessesMap[source] = true
	}

	// Build the final query span result
	var results []base.QuerySpanResult
	if q.resultTableSource != nil {
		results = q.resultTableSource.GetQuerySpanResult()
	}

	return &base.QuerySpan{
		Type:          base.Select,
		SourceColumns: accessesMap,
		Results:       results,
	}, nil
}

func (q *querySpanExtractor) EnterStmtmulti(sm *pgparser.StmtmultiContext) {
	if len(sm.AllStmt()) != 1 {
		q.err = errors.Errorf("malformed query, expected 1 query but got %d", len(sm.AllStmt()))
		return
	}

	// Process the statement and store the result
	stmt := sm.AllStmt()[0]
	tableSource, err := q.extractTableSourceFromStmt(stmt)
	if err != nil {
		q.err = err
		return
	}

	// Store the extracted table source for later retrieval
	q.resultTableSource = tableSource
}

func (q *querySpanExtractor) extractTableSourceFromStmt(stmt pgparser.IStmtContext) (base.TableSource, error) {
	switch {
	case stmt.Explainstmt() != nil:
		// Handle EXPLAIN statement
		explainStmt := stmt.Explainstmt()
		if explainStmt.Explainablestmt() != nil {
			// Extract from the explained statement
			if explainableStmt := explainStmt.Explainablestmt(); explainableStmt != nil {
				if explainableStmt.Selectstmt() != nil {
					return q.extractTableSourceFromSelectstmt(explainableStmt.Selectstmt())
				}
				if explainableStmt.Insertstmt() != nil ||
					explainableStmt.Updatestmt() != nil ||
					explainableStmt.Deletestmt() != nil {
					// For DML in EXPLAIN, return empty pseudo table
					return &base.PseudoTable{
						Name:    "",
						Columns: []base.QuerySpanResult{},
					}, nil
				}
			}
		}
		return &base.PseudoTable{
			Name:    "",
			Columns: []base.QuerySpanResult{},
		}, nil
	case stmt.Selectstmt() != nil:
		return q.extractTableSourceFromSelectstmt(stmt.Selectstmt())
	default:
		// For non-SELECT statements, return empty pseudo table
		return &base.PseudoTable{
			Name:    "",
			Columns: []base.QuerySpanResult{},
		}, nil
	}
}

func (q *querySpanExtractor) extractTableSourceFromSelectstmt(selectstmt pgparser.ISelectstmtContext) (base.TableSource, error) {
	selectNoParens := getSelectNoParensFromSelectStmt(selectstmt)
	return q.extractTableSourceFromSelectNoParens(selectNoParens)
}

func (q *querySpanExtractor) extractTableSourceFromSelectNoParens(selectNoParens pgparser.ISelect_no_parensContext) (base.TableSource, error) {
	// Reset the table sources from the FROM clause after exit the SELECT statement.
	previousTableSourcesFromLength := len(q.tableSourcesFrom)
	defer func() {
		q.tableSourcesFrom = q.tableSourcesFrom[:previousTableSourcesFromLength]
	}()

	// Analyze the CTE first.
	withClause := selectNoParens.With_clause()
	if withClause != nil {
		cteLength, err := q.recordCTEs(withClause)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to record CTEs")
		}
		defer func() {
			q.ctes = q.ctes[:cteLength]
		}()
	}

	// Process the SELECT clause
	selectClause := selectNoParens.Select_clause()
	if selectClause == nil {
		return nil, errors.New("select_no_parens has no select_clause")
	}

	// Handle UNION/INTERSECT/EXCEPT at the top level
	allSimpleSelects := selectClause.AllSimple_select_intersect()
	if len(allSimpleSelects) == 0 {
		return nil, errors.New("select_clause has no simple_select_intersect")
	}

	// If there's only one simple_select_intersect, process it directly
	if len(allSimpleSelects) == 1 {
		return q.extractTableSourceFromSimpleSelectIntersect(allSimpleSelects[0])
	}

	// Multiple simple_select_intersect with operators between them
	// Extract operators from the select_clause
	operators, err := q.extractOperatorsFromSelectClause(selectClause)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to extract operators from select_clause")
	}

	return q.mergeSimpleSelectIntersectsWithOperators(allSimpleSelects, operators)
}

func (q *querySpanExtractor) recordCTEs(with pgparser.IWith_clauseContext) (restoreCTELength int, err error) {
	previousCTELength := len(q.ctes)
	recursive := with.RECURSIVE() != nil
	for _, cte := range with.Cte_list().AllCommon_table_expr() {
		cteName := normalizePostgreSQLName(cte.Name())
		var colAliasNames []string
		if cte.Opt_name_list() != nil {
			for _, name := range cte.Opt_name_list().Name_list().AllName() {
				colAliasNames = append(colAliasNames, normalizePostgreSQLName(name))
			}
		}

		var cteTableSource *base.PseudoTable
		if !recursive {
			// Non-recursive WITH clause - all CTEs are non-recursive
			cteTableSource, err = q.extractTableSourceFromNonRecursiveCTE(cte, cteName, colAliasNames)
			if err != nil {
				return previousCTELength, err
			}
		} else {
			// WITH RECURSIVE clause - try to extract as recursive
			// If it's not actually recursive, extractTableSourceFromRecursiveCTE will fall back to non-recursive
			cteTableSource, err = q.extractTableSourceFromRecursiveCTE(cte, cteName, colAliasNames)
			if err != nil {
				return previousCTELength, err
			}
		}

		q.ctes = append(q.ctes, cteTableSource)
	}

	return previousCTELength, nil
}

// extractTableSourceFromRecursiveCTE processes a recursive CTE by separating it into base and recursive parts,
// then performs iterative closure computation to resolve all column dependencies.
// A recursive CTE has the structure: base_case UNION [ALL] recursive_case
// We use a simple strategy: the last part after UNION is the recursive part.
func (q *querySpanExtractor) extractTableSourceFromRecursiveCTE(cte pgparser.ICommon_table_exprContext, cteName string, colAliasNames []string) (*base.PseudoTable, error) {
	selectStmt := cte.Preparablestmt().Selectstmt()
	if selectStmt == nil {
		return nil, errors.Errorf("malformed recursive CTE, expected SELECT statement in CTE but got others")
	}
	selectNoParens := getSelectNoParensFromSelectStmt(selectStmt)

	// Split the CTE into base and recursive parts using the simple strategy
	baseParts, baseOperators, recursivePart, recursiveOperator, err := splitRecursiveCTEParts(selectNoParens)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to split recursive CTE %q", cteName)
	}

	if len(baseParts) == 0 || recursivePart == nil || recursiveOperator == nil {
		// Not a valid recursive structure, treat as non-recursive
		return q.extractTableSourceFromNonRecursiveCTE(cte, cteName, colAliasNames)
	}

	// Process the base parts (can be multiple parts combined with various operators)
	var baseTableSource base.TableSource
	if len(baseParts) == 1 {
		// Single base part
		baseTableSource, err = q.extractTableSourceFromSimpleSelectIntersect(baseParts[0])
		if err != nil {
			return nil, errors.Wrapf(err, "failed to extract base part for recursive CTE %q", cteName)
		}
	} else {
		// Multiple base parts - need to merge them with their respective operators
		baseTableSource, err = q.mergeSimpleSelectIntersectsWithOperators(baseParts, baseOperators)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to merge base parts for recursive CTE %q", cteName)
		}
	}

	// Apply column aliases
	querySpanResult := baseTableSource.GetQuerySpanResult()
	if len(colAliasNames) > 0 {
		if len(colAliasNames) != len(querySpanResult) {
			return nil, errors.Errorf("CTE %q has %d columns but alias has %d columns", cteName, len(querySpanResult), len(colAliasNames))
		}
		for i, colAlias := range colAliasNames {
			querySpanResult[i].Name = colAlias
		}
	}

	cteTableSource := &base.PseudoTable{
		Name:    cteName,
		Columns: querySpanResult,
	}

	// Add to CTEs for recursive part to reference
	q.ctes = append(q.ctes, cteTableSource)
	defer func() {
		q.ctes = q.ctes[:len(q.ctes)-1]
	}()

	// Iterative closure computation
	for {
		recursiveTableSource, err := q.extractTableSourceFromSimpleSelectIntersect(recursivePart)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to extract recursive part for CTE %q", cteName)
		}

		recursiveQuerySpanResult := recursiveTableSource.GetQuerySpanResult()
		if len(recursiveQuerySpanResult) != len(querySpanResult) {
			return nil, errors.Errorf("CTE %q base has %d columns but recursive has %d", cteName, len(querySpanResult), len(recursiveQuerySpanResult))
		}

		changed := false
		for i, spanResult := range recursiveQuerySpanResult {
			newSourceColumns, hasDiff := base.MergeSourceColumnSet(querySpanResult[i].SourceColumns, spanResult.SourceColumns)
			if hasDiff {
				changed = true
				querySpanResult[i].SourceColumns = newSourceColumns
			}
		}

		if !changed {
			break
		}
		q.ctes[len(q.ctes)-1].Columns = querySpanResult
	}

	return cteTableSource, nil
}

func (q *querySpanExtractor) extractTableSourceFromNonRecursiveCTE(cte pgparser.ICommon_table_exprContext, cteName string, colAliasNames []string) (*base.PseudoTable, error) {
	selectStmt := cte.Preparablestmt().Selectstmt()
	if selectStmt == nil {
		return nil, errors.Errorf("malformed CTE, expected SELECT statement in CTE but got others")
	}
	// Non-recursive CTE.
	tableSource, err := q.extractTableSourceFromSelectstmt(selectStmt)
	if err != nil {
		return nil, err
	}
	querySpanResult := tableSource.GetQuerySpanResult()
	if len(colAliasNames) > 0 && len(colAliasNames) != len(querySpanResult) {
		return nil, errors.Errorf("the column alias number %d does not match the actual column number %d in CTE %q", len(colAliasNames), len(querySpanResult), cteName)
	}
	for i, colAlias := range colAliasNames {
		querySpanResult[i].Name = colAlias
	}
	cteTableSource := &base.PseudoTable{
		Name:    cteName,
		Columns: querySpanResult,
	}
	return cteTableSource, nil
}

// extractOperatorsFromSelectClause extracts set operators from a select_clause
func (q *querySpanExtractor) extractOperatorsFromSelectClause(selectClause pgparser.ISelect_clauseContext) ([]SetOperator, error) {
	if selectClause == nil {
		return nil, errors.New("nil select_clause")
	}

	allSimpleSelects := selectClause.AllSimple_select_intersect()
	if len(allSimpleSelects) <= 1 {
		return nil, nil // No operators needed for 0 or 1 select
	}

	var operators []SetOperator
	children := selectClause.GetChildren()

	// Parse children to find operators between simple_select_intersect nodes
	for i := 0; i < len(children); i++ {
		if token, ok := children[i].(antlr.TerminalNode); ok {
			tokenType := token.GetSymbol().GetTokenType()
			var op SetOperator

			switch tokenType {
			case pgparser.PostgreSQLParserUNION:
				op.Type = "UNION"
				op.IsDistinct = true // Default to DISTINCT
				// Check for ALL/DISTINCT modifier
				if i+1 < len(children) {
					if nextToken, ok := children[i+1].(antlr.TerminalNode); ok {
						if nextToken.GetSymbol().GetTokenType() == pgparser.PostgreSQLParserALL {
							op.IsDistinct = false
							i++ // Skip ALL token
						} else if nextToken.GetSymbol().GetTokenType() == pgparser.PostgreSQLParserDISTINCT {
							op.IsDistinct = true
							i++ // Skip DISTINCT token
						}
					}
				}
				operators = append(operators, op)

			case pgparser.PostgreSQLParserEXCEPT:
				op.Type = "EXCEPT"
				op.IsDistinct = true // Default to DISTINCT
				// Check for ALL/DISTINCT modifier
				if i+1 < len(children) {
					if nextToken, ok := children[i+1].(antlr.TerminalNode); ok {
						if nextToken.GetSymbol().GetTokenType() == pgparser.PostgreSQLParserALL {
							op.IsDistinct = false
							i++ // Skip ALL token
						} else if nextToken.GetSymbol().GetTokenType() == pgparser.PostgreSQLParserDISTINCT {
							op.IsDistinct = true
							i++ // Skip DISTINCT token
						}
					}
				}
				operators = append(operators, op)

			case pgparser.PostgreSQLParserINTERSECT:
				op.Type = "INTERSECT"
				op.IsDistinct = true // Default to DISTINCT
				// Check for ALL/DISTINCT modifier
				if i+1 < len(children) {
					if nextToken, ok := children[i+1].(antlr.TerminalNode); ok {
						if nextToken.GetSymbol().GetTokenType() == pgparser.PostgreSQLParserALL {
							op.IsDistinct = false
							i++ // Skip ALL token
						} else if nextToken.GetSymbol().GetTokenType() == pgparser.PostgreSQLParserDISTINCT {
							op.IsDistinct = true
							i++ // Skip DISTINCT token
						}
					}
				}
				operators = append(operators, op)
			}
		}
	}

	if len(operators) != len(allSimpleSelects)-1 {
		return nil, errors.Errorf("expected %d operators but found %d", len(allSimpleSelects)-1, len(operators))
	}

	return operators, nil
}

// SetOperator represents a set operation between SELECT statements
type SetOperator struct {
	Type       string // "UNION", "EXCEPT", "INTERSECT"
	IsDistinct bool   // false means ALL, true means DISTINCT (or implicit distinct)
}

// splitRecursiveCTEParts analyzes a recursive CTE's SELECT structure and separates it into:
// - baseParts: Everything before the last UNION/UNION ALL (the non-recursive anchor queries)
// - baseOperators: The operators between base parts (UNION, EXCEPT, etc.)
// - recursivePart: The part after the last UNION/UNION ALL (the recursive query)
// - recursiveOperator: The operator connecting the base to the recursive part (must be UNION [ALL])
// Note: The recursive part must be connected with UNION [ALL], not EXCEPT or INTERSECT.
func splitRecursiveCTEParts(selectNoParens pgparser.ISelect_no_parensContext) (
	baseParts []pgparser.ISimple_select_intersectContext,
	baseOperators []SetOperator,
	recursivePart pgparser.ISimple_select_intersectContext,
	recursiveOperator *SetOperator,
	err error) {
	if selectNoParens == nil || selectNoParens.Select_clause() == nil {
		return nil, nil, nil, nil, errors.New("invalid select_no_parens")
	}

	selectClause := selectNoParens.Select_clause()
	allParts := selectClause.AllSimple_select_intersect()

	if len(allParts) < 2 {
		// No UNION, cannot be a recursive CTE
		return nil, nil, nil, nil, nil
	}

	// Track all operators between parts
	var allOperators []SetOperator
	lastUnionIndex := -1

	// Parse the children to extract operators
	// Pattern: part0 [OPERATOR [ALL|DISTINCT]] part1 [OPERATOR [ALL|DISTINCT]] part2 ...
	children := selectClause.GetChildren()
	partIndex := 0
	i := 0

	for i < len(children) {
		child := children[i]
		if token, ok := child.(antlr.TerminalNode); ok {
			tokenType := token.GetSymbol().GetTokenType()
			var op SetOperator

			switch tokenType {
			case pgparser.PostgreSQLParserUNION:
				op.Type = "UNION"
				// Check if next token is ALL or DISTINCT
				op.IsDistinct = true // Default to DISTINCT
				if i+1 < len(children) {
					if nextToken, ok := children[i+1].(antlr.TerminalNode); ok {
						nextType := nextToken.GetSymbol().GetTokenType()
						switch nextType {
						case pgparser.PostgreSQLParserALL:
							op.IsDistinct = false
							i++
						case pgparser.PostgreSQLParserDISTINCT:
							op.IsDistinct = true
							i++
						}
					}
				}
				allOperators = append(allOperators, op)
				lastUnionIndex = partIndex
				partIndex++

			case pgparser.PostgreSQLParserEXCEPT:
				op.Type = "EXCEPT"
				// Check if next token is ALL or DISTINCT
				op.IsDistinct = true // Default to DISTINCT
				if i+1 < len(children) {
					if nextToken, ok := children[i+1].(antlr.TerminalNode); ok {
						nextType := nextToken.GetSymbol().GetTokenType()
						switch nextType {
						case pgparser.PostgreSQLParserALL:
							op.IsDistinct = false
							i++
						case pgparser.PostgreSQLParserDISTINCT:
							op.IsDistinct = true
							i++
						}
					}
				}
				allOperators = append(allOperators, op)
				partIndex++

			case pgparser.PostgreSQLParserINTERSECT:
				op.Type = "INTERSECT"
				// Check if next token is ALL or DISTINCT
				op.IsDistinct = true // Default to DISTINCT
				if i+1 < len(children) {
					if nextToken, ok := children[i+1].(antlr.TerminalNode); ok {
						nextType := nextToken.GetSymbol().GetTokenType()
						switch nextType {
						case pgparser.PostgreSQLParserALL:
							op.IsDistinct = false
							i++
						case pgparser.PostgreSQLParserDISTINCT:
							op.IsDistinct = true
							i++
						}
					}
				}
				allOperators = append(allOperators, op)
				partIndex++
			}
		}
		i++
	}

	if lastUnionIndex < 0 {
		// No UNION found, not a recursive CTE
		return nil, nil, nil, nil, nil
	}

	// Split at the last UNION position
	// Everything before the last UNION is the base part(s)
	// The part after the last UNION is the recursive part
	baseParts = allParts[:lastUnionIndex+1]
	recursivePart = allParts[lastUnionIndex+1]

	// Split operators too
	if lastUnionIndex > 0 {
		baseOperators = allOperators[:lastUnionIndex]
	}
	recursiveOperator = &allOperators[lastUnionIndex]

	return baseParts, baseOperators, recursivePart, recursiveOperator, nil
}

// extractTableSourceFromSimpleSelectIntersect processes a simple_select_intersect node and returns its table source.
// This handles the INTERSECT operations between multiple simple_select_pramary nodes.
func (q *querySpanExtractor) extractTableSourceFromSimpleSelectIntersect(simpleSelect pgparser.ISimple_select_intersectContext) (base.TableSource, error) {
	if simpleSelect == nil {
		return nil, errors.New("nil simple_select_intersect")
	}

	allPrimaries := simpleSelect.AllSimple_select_pramary()
	if len(allPrimaries) == 0 {
		return nil, errors.New("no primary selects in simple_select_intersect")
	}

	// If there's only one primary, just process it directly
	if len(allPrimaries) == 1 {
		return q.extractTableSourceFromSimpleSelectPrimary(allPrimaries[0])
	}

	// Multiple primaries - need to handle INTERSECT operations
	// Start with the first primary
	result, err := q.extractTableSourceFromSimpleSelectPrimary(allPrimaries[0])
	if err != nil {
		return nil, errors.Wrap(err, "failed to extract first primary")
	}

	// Process remaining primaries with INTERSECT
	for i := 1; i < len(allPrimaries); i++ {
		nextResult, err := q.extractTableSourceFromSimpleSelectPrimary(allPrimaries[i])
		if err != nil {
			return nil, errors.Wrapf(err, "failed to extract primary %d", i)
		}

		// Merge with INTERSECT semantics
		leftQuerySpanResult := result.GetQuerySpanResult()
		rightQuerySpanResult := nextResult.GetQuerySpanResult()

		if len(leftQuerySpanResult) != len(rightQuerySpanResult) {
			return nil, errors.Errorf("INTERSECT requires same number of columns: %d vs %d",
				len(leftQuerySpanResult), len(rightQuerySpanResult))
		}

		var mergedColumns []base.QuerySpanResult
		for j, leftCol := range leftQuerySpanResult {
			rightCol := rightQuerySpanResult[j]
			// For INTERSECT, we take columns that exist in both
			mergedSourceColumns, _ := base.MergeSourceColumnSet(leftCol.SourceColumns, rightCol.SourceColumns)
			mergedColumns = append(mergedColumns, base.QuerySpanResult{
				Name:          leftCol.Name,
				SourceColumns: mergedSourceColumns,
			})
		}

		result = &base.PseudoTable{
			Name:    "",
			Columns: mergedColumns,
		}
	}

	return result, nil
}

// extractTableSourceFromSimpleSelectPrimary processes a simple_select_pramary node and returns its table source.
// This is the basic building block of SELECT statements, containing the SELECT list, FROM clause, etc.
func (q *querySpanExtractor) extractTableSourceFromSimpleSelectPrimary(primary pgparser.ISimple_select_pramaryContext) (base.TableSource, error) {
	if primary == nil {
		return nil, errors.New("nil simple_select_pramary")
	}

	// Handle VALUES clause
	if v := primary.Values_clause(); v != nil {
		columnNumber := len(v.AllExpr_list()[0].AllA_expr())
		// Check other rows have the same number of columns
		for _, exprList := range v.AllExpr_list()[1:] {
			if len(exprList.AllA_expr()) != columnNumber {
				return nil, errors.New("VALUES clause rows have different column counts")
			}
		}
		columnSourceSets := make([]base.SourceColumnSet, columnNumber)
		for i, exprList := range v.AllExpr_list() {
			for j, expr := range exprList.AllA_expr() {
				cols, err := q.extractSourceColumnSetFromAExpr(expr)
				if err != nil {
					return nil, errors.Wrapf(err, "failed to extract columns from VALUES clause row %d, column %d", i, j)
				}
				columnSourceSets[j], _ = base.MergeSourceColumnSet(columnSourceSets[j], cols)
			}
		}
		tableSource := &base.PseudoTable{
			Name: "",
			Columns: func() []base.QuerySpanResult {
				var results []base.QuerySpanResult
				for i, colSet := range columnSourceSets {
					results = append(results, base.QuerySpanResult{
						Name:          fmt.Sprintf("column%d", i+1),
						SourceColumns: colSet,
					})
				}
				return results
			}(),
		}

		return tableSource, nil
	}

	// Handle TABLE clause (TABLE tablename)
	// TODO: Implement TABLE clause handling when needed

	// Handle regular SELECT
	if primary.SELECT() != nil {
		// Save current FROM table sources
		previousTableSourcesFromLength := len(q.tableSourcesFrom)
		defer func() {
			q.tableSourcesFrom = q.tableSourcesFrom[:previousTableSourcesFromLength]
		}()

		// Process FROM clause first
		fromFieldList, err := q.extractTableSourceFromFromClause(primary.From_clause())
		if err != nil {
			return nil, errors.Wrapf(err, "failed to extract table sources from FROM clause")
		}

		// Process target list (SELECT items)
		var targetListContext pgparser.ITarget_listContext
		if primary.Opt_target_list() != nil {
			targetListContext = primary.Opt_target_list().Target_list()
		} else if primary.Target_list() != nil {
			targetListContext = primary.Target_list()
		}

		// PostgreSQL allow SELECT without target list (e.g., SELECT FROM table), it returns no columns.
		if targetListContext == nil {
			return &base.PseudoTable{
				Name:    "",
				Columns: []base.QuerySpanResult{},
			}, nil
		}

		columns, err := q.extractColumnsFromTargetList(targetListContext, fromFieldList)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to extract columns from target list")
		}

		result := &base.PseudoTable{
			Name:    "",
			Columns: columns,
		}

		return result, nil
	}

	// Handle parenthesized SELECT
	if s := primary.Select_with_parens(); s != nil {
		selectNoParens := getSelectNoParensFromSelectWithParens(s)
		return q.extractTableSourceFromSelectNoParens(selectNoParens)
	}

	// Default case
	return &base.PseudoTable{
		Name:    "",
		Columns: []base.QuerySpanResult{},
	}, nil
}

// extractTableSourceFromSelectWithParens handles SELECT statements wrapped in parentheses
func (q *querySpanExtractor) extractTableSourceFromSelectWithParens(selectWithParens pgparser.ISelect_with_parensContext) (base.TableSource, error) {
	if selectWithParens == nil {
		return nil, errors.New("nil select_with_parens")
	}

	if selectWithParens.Select_no_parens() != nil {
		return q.extractTableSourceFromSelectNoParens(selectWithParens.Select_no_parens())
	}

	if selectWithParens.Select_with_parens() != nil {
		// Nested parentheses
		return q.extractTableSourceFromSelectWithParens(selectWithParens.Select_with_parens())
	}

	return nil, errors.New("select_with_parens has no valid content")
}

// mergeSimpleSelectIntersectsWithOperators combines multiple simple_select_intersect nodes with their respective operators.
// This properly handles UNION, EXCEPT, INTERSECT with ALL/DISTINCT modifiers.
func (q *querySpanExtractor) mergeSimpleSelectIntersectsWithOperators(parts []pgparser.ISimple_select_intersectContext, operators []SetOperator) (base.TableSource, error) {
	if len(parts) == 0 {
		return nil, errors.New("no parts to merge")
	}

	if len(parts) == 1 {
		// Single part, no operators needed
		return q.extractTableSourceFromSimpleSelectIntersect(parts[0])
	}

	if len(operators) != len(parts)-1 {
		return nil, errors.Errorf("invalid operators count: %d operators for %d parts", len(operators), len(parts))
	}

	// Start with the first part
	result, err := q.extractTableSourceFromSimpleSelectIntersect(parts[0])
	if err != nil {
		return nil, errors.Wrap(err, "failed to extract first simple_select_intersect")
	}

	// Apply each operator with the next part
	for i, op := range operators {
		nextResult, err := q.extractTableSourceFromSimpleSelectIntersect(parts[i+1])
		if err != nil {
			return nil, errors.Wrapf(err, "failed to extract simple_select_intersect %d", i+1)
		}

		// Apply the specific operator semantics
		result, err = q.applySetOperator(result, nextResult, op)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to apply %s at position %d", op.Type, i)
		}
	}

	return result, nil
}

// applySetOperator applies a set operation (UNION, EXCEPT, INTERSECT) between two table sources
func (q *querySpanExtractor) applySetOperator(left, right base.TableSource, op SetOperator) (base.TableSource, error) {
	leftQuerySpanResult := left.GetQuerySpanResult()
	rightQuerySpanResult := right.GetQuerySpanResult()

	if len(leftQuerySpanResult) != len(rightQuerySpanResult) {
		return nil, errors.Errorf("%s requires same number of columns: %d vs %d",
			op.Type, len(leftQuerySpanResult), len(rightQuerySpanResult))
	}

	var mergedColumns []base.QuerySpanResult

	switch op.Type {
	case "UNION":
		// UNION combines rows from both queries
		for j, leftCol := range leftQuerySpanResult {
			rightCol := rightQuerySpanResult[j]
			mergedSourceColumns, _ := base.MergeSourceColumnSet(leftCol.SourceColumns, rightCol.SourceColumns)
			mergedColumns = append(mergedColumns, base.QuerySpanResult{
				Name:          leftCol.Name, // Use the name from the first query
				SourceColumns: mergedSourceColumns,
			})
		}

	case "EXCEPT":
		// EXCEPT returns rows from left that don't appear in right
		// For column tracking, we only track columns from the left side
		for _, leftCol := range leftQuerySpanResult {
			mergedColumns = append(mergedColumns, base.QuerySpanResult{
				Name:          leftCol.Name,
				SourceColumns: leftCol.SourceColumns, // Only left side columns matter
			})
		}

	case "INTERSECT":
		// INTERSECT returns rows that appear in both
		// We need to track columns from both sides
		for j, leftCol := range leftQuerySpanResult {
			rightCol := rightQuerySpanResult[j]
			mergedSourceColumns, _ := base.MergeSourceColumnSet(leftCol.SourceColumns, rightCol.SourceColumns)
			mergedColumns = append(mergedColumns, base.QuerySpanResult{
				Name:          leftCol.Name,
				SourceColumns: mergedSourceColumns,
			})
		}

	default:
		return nil, errors.Errorf("unsupported set operator: %s", op.Type)
	}

	return &base.PseudoTable{
		Name:    "",
		Columns: mergedColumns,
	}, nil
}

// extractTableSourceFromFromClause processes the FROM clause and returns all table sources
func (q *querySpanExtractor) extractTableSourceFromFromClause(fromClause pgparser.IFrom_clauseContext) ([]base.TableSource, error) {
	if fromClause == nil || fromClause.From_list() == nil {
		return nil, nil
	}

	var fromFieldList []base.TableSource
	for _, fromItem := range fromClause.From_list().AllTable_ref() {
		tableSource, err := q.extractTableSourceFromTableRef(fromItem)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to extract table source from FROM item")
		}
		q.tableSourcesFrom = append(q.tableSourcesFrom, tableSource)
		fromFieldList = append(fromFieldList, tableSource)
	}

	return fromFieldList, nil
}


// extractTableSourceFromTableRef extracts table source from a table reference in FROM clause
func (q *querySpanExtractor) extractTableSourceFromTableRef(tableRef pgparser.ITable_refContext) (base.TableSource, error) {
	if tableRef == nil {
		return nil, errors.New("nil table_ref")
	}

	var anchor base.TableSource
	var err error

	// Handle simple table reference (e.g., schema.table)
	if tableRef.Relation_expr() != nil {
		anchor, err = q.extractTableSourceFromRelationExpr(tableRef.Relation_expr(), tableRef.Opt_alias_clause())
		if err != nil {
			return nil, errors.Wrapf(err, "failed to extract table source from relation_expr")
		}
		if tableRef.Opt_alias_clause() != nil {
			anchor, err = applyOptAliasClauseToTableSource(anchor, tableRef.Opt_alias_clause())
			if err != nil {
				return nil, errors.Wrapf(err, "failed to apply alias to relation_expr")
			}
		}
	}

	// Handle function tables (e.g., generate_series, unnest)
	if tableRef.Func_table() != nil {
		anchor, err = q.extractTableSourceFromFuncTable(tableRef.Func_table(), tableRef.Func_alias_clause())
		if err != nil {
			return nil, errors.Wrapf(err, "failed to extract table source from func_table")
		}
		if tableRef.Func_alias_clause() != nil {
			anchor, err = applyFuncAliasClauseToTableSource(anchor, tableRef.Func_alias_clause())
			if err != nil {
				return nil, errors.Wrapf(err, "failed to apply alias to func_table")
			}
		}
	}

	// TODO(zp): Handle xmltable candidates.

	// Handle subquery (SELECT in parentheses)
	if tableRef.Select_with_parens() != nil {
		anchor, err = q.extractTableSourceFromSelectWithParens(tableRef.Select_with_parens())
		if err != nil {
			return nil, errors.Wrapf(err, "failed to extract subquery")
		}

		// Apply alias if present
		if tableRef.Opt_alias_clause() != nil {
			anchor, err = applyOptAliasClauseToTableSource(anchor, tableRef.Opt_alias_clause())
			if err != nil {
				return nil, errors.Wrapf(err, "failed to apply alias to subquery")
			}
		}
	}

	if tableRef.Table_ref() != nil {
		anchor, err = q.extractTableSourceFromTableRef(tableRef.Table_ref())
		if err != nil {
			return nil, errors.Wrapf(err, "failed to extract nested table_ref")
		}
		if len(tableRef.AllJoined_table()) == 0 && tableRef.Opt_alias_clause() != nil {
			anchor, err = applyOptAliasClauseToTableSource(anchor, tableRef.Opt_alias_clause())
			if err != nil {
				return nil, errors.Wrapf(err, "failed to apply alias to nested table_ref")
			}
		}
	}

	q.tableSourcesFrom = append(q.tableSourcesFrom, anchor)

	// Handle JOIN expressions
	if len(tableRef.AllJoined_table()) != 0 {
		joinedTables := tableRef.AllJoined_table()
		for i, join := range joinedTables {
			tableSource, err := q.extractTableSourceFromTableRef(join.Table_ref())
			if err != nil {
				return nil, errors.Wrapf(err, "failed to extract joined table in JOIN")
			}
			// According to the grammar:
			// OPEN_PAREN table_ref joined_table? CLOSE_PAREN opt_alias_clause?
			// ) joined_table*
			// The alias appears before the second joined_table.
			// Determine join type and conditions
			naturalJoin := join.NATURAL() != nil
			var usingColumns []string
			if join.Join_qual() != nil && join.Join_qual().USING() != nil {
				usingColumns = normalizePostgreSQLNameList(join.Join_qual().Name_list())
			}
			anchor, err = q.joinTableSources(anchor, tableSource, naturalJoin, usingColumns)
			if err != nil {
				return nil, errors.Wrapf(err, "failed to join tables")
			}
			if i == 0 && tableRef.Opt_alias_clause() != nil && tableRef.Table_ref() != nil {
				anchor, err = applyOptAliasClauseToTableSource(anchor, tableRef.Opt_alias_clause())
				if err != nil {
					return nil, errors.Wrapf(err, "failed to apply alias to table source")
				}
			}
		}
	}

	return anchor, nil
}

func (q *querySpanExtractor) joinTableSources(left, right base.TableSource, naturalJoin bool, usingColumn []string) (base.TableSource, error) {
	leftSpanResult, rightSpanResult := left.GetQuerySpanResult(), right.GetQuerySpanResult()

	result := new(base.PseudoTable)

	if naturalJoin {
		leftSpanResultIdx, rightSpanResultIdx := make(map[string]int, len(leftSpanResult)), make(map[string]int, len(rightSpanResult))
		for i, spanResult := range leftSpanResult {
			leftSpanResultIdx[spanResult.Name] = i
		}
		for i, spanResult := range rightSpanResult {
			rightSpanResultIdx[spanResult.Name] = i
		}

		// NaturalJoin will merge the same column name field.
		for idx, spanResult := range leftSpanResult {
			if _, ok := rightSpanResultIdx[spanResult.Name]; ok {
				spanResult.SourceColumns, _ = base.MergeSourceColumnSet(spanResult.SourceColumns, rightSpanResult[idx].SourceColumns)
			}
			result.Columns = append(result.Columns, spanResult)
		}
		for _, spanResult := range rightSpanResult {
			if _, ok := leftSpanResultIdx[spanResult.Name]; !ok {
				result.Columns = append(result.Columns, spanResult)
			}
		}
	} else {
		if len(usingColumn) > 0 {
			// ... JOIN ... USING (...) will merge the column in USING.
			usingMap := make(map[string]bool, len(usingColumn))
			for _, using := range usingColumn {
				usingMap[using] = true
			}

			result.Columns = append(result.Columns, leftSpanResult...)
			for _, spanResult := range rightSpanResult {
				if _, ok := usingMap[spanResult.Name]; !ok {
					result.Columns = append(result.Columns, spanResult)
				}
			}
		} else {
			result.Columns = append(result.Columns, leftSpanResult...)
			result.Columns = append(result.Columns, rightSpanResult...)
		}
	}

	return result, nil
}

// extractTableSourceFromRelationExpr extracts table source from a relation expression (simple table)
func (q *querySpanExtractor) extractTableSourceFromRelationExpr(relationExpr pgparser.IRelation_exprContext, aliasClause pgparser.IOpt_alias_clauseContext) (base.TableSource, error) {
	if relationExpr == nil {
		return nil, errors.New("nil relation_expr")
	}

	qualifiedName := relationExpr.Qualified_name()
	if qualifiedName == nil {
		return nil, errors.New("relation_expr has no qualified_name")
	}

	// Extract the table name parts
	nameParts := NormalizePostgreSQLQualifiedName(qualifiedName)

	var schemaName, tableName string
	switch len(nameParts) {
	case 1:
		tableName = nameParts[0]
		schemaName = "" // Will be resolved using search path
	case 2:
		schemaName = nameParts[0]
		tableName = nameParts[1]
	case 3:
		// Cross-database query: database.schema.table
		// For now, we ignore the database part and use schema.table
		schemaName = nameParts[1]
		tableName = nameParts[2]
	default:
		return nil, errors.Errorf("invalid qualified name with %d parts", len(nameParts))
	}

	// Extract alias and column names from alias clause
	aliasName := tableName
	var columnAliases []string
	if aliasClause != nil && aliasClause.Table_alias_clause() != nil {
		tableAlias := aliasClause.Table_alias_clause()
		if tableAlias.Table_alias() != nil {
			aliasName = normalizePostgreSQLTableAlias(tableAlias.Table_alias())
		}
		if tableAlias.Name_list() != nil {
			columnAliases = normalizePostgreSQLNameList(tableAlias.Name_list())
		}
	}

	// Find the actual table schema with columns
	tableSource, err := q.findTableSchema(schemaName, tableName)
	if err != nil {
		return nil, err
	}

	// Apply alias if specified
	if aliasName != tableName || len(columnAliases) > 0 {
		// If we have column aliases, use them; otherwise use the original columns
		var columns []base.QuerySpanResult
		if len(columnAliases) > 0 {
			for _, alias := range columnAliases {
				columns = append(columns, base.QuerySpanResult{Name: alias})
			}
		} else {
			columns = tableSource.GetQuerySpanResult()
		}
		return &base.PseudoTable{
			Name:    aliasName,
			Columns: columns,
		}, nil
	}

	return tableSource, nil
}

func (q *querySpanExtractor) extractTableSourceFromUDF2(funcExprWindowless pgparser.IFunc_expr_windowlessContext, funcAlias pgparser.IFunc_alias_clauseContext) (base.TableSource, error) {
	if funcExprWindowless == nil {
		return nil, errors.New("nil func_expr_windowless")
	}

	schemaName, funcName, args, err := q.extractFunctionElementFromFuncExprWindowless(funcExprWindowless)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to extract function element from func_expr_windowless")
	}

	tableSource, err := q.findFunctionDefine(schemaName, funcName, len(args))
	if err != nil {
		return nil, errors.Wrapf(err, "failed to find function definition for %s.%s with %d args", schemaName, funcName, len(args))
	}
	if funcAlias != nil {
		tableSource, err = applyFuncAliasClauseToTableSource(tableSource, funcAlias)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to apply function alias clause")
		}
	}
	return tableSource, nil
}

// extractTableSourceFromFuncTable extracts table source from a function table
func (q *querySpanExtractor) extractTableSourceFromFuncTable(funcTable pgparser.IFunc_tableContext, funcAlias pgparser.IFunc_alias_clauseContext) (base.TableSource, error) {
	result := &base.PseudoTable{
		Name:    "",
		Columns: []base.QuerySpanResult{},
	}

	if funcTable == nil {
		return result, nil
	}

	if funcTable.Func_expr_windowless() != nil {
		schemaName, funcName, args, err := q.extractFunctionElementFromFuncExprWindowless(funcTable.Func_expr_windowless())
		if err != nil {
			return nil, errors.Wrapf(err, "failed to extract function name from func_expr_windowless")
		}
		if funcName == "" {
			return nil, errors.Wrapf(err, "empty function name in func_expr_windowless")
		}
		if schemaName == "" && IsSystemFunction(funcName, "") {
			// If the schemaName is empty, we try to match the system function first.
			return q.extractTableSourceFromSystemFunctionNew(funcName, args, funcAlias)
		}
		return q.extractTableSourceFromUDF2(funcTable.Func_expr_windowless(), funcAlias)
	}

	// The actual column count and names will be determined by the alias
	// For now, just return an empty pseudo table
	// The applyAliasToTableSource function will handle the column names

	// Check if this is a ROWS FROM expression
	if funcTable.ROWS() != nil && funcTable.Rowsfrom_list() != nil {
		// Handle ROWS FROM (...)
		// Each function in ROWS FROM produces columns
		// ROWS FROM expands multiple functions in parallel, creating a single result set where the first row contains
		// the first result from each function, the second row contains the second result from each function, and so on.
		tableSource := &base.PseudoTable{
			Columns: []base.QuerySpanResult{},
		}
		for _, item := range funcTable.Rowsfrom_list().AllRowsfrom_item() {
			if item.Func_expr_windowless() != nil {
				schemaName, funcName, args, err := q.extractFunctionElementFromFuncExprWindowless(funcTable.Func_expr_windowless())
				if err != nil {
					return nil, errors.Wrapf(err, "failed to extract function name from func_expr_windowless")
				}
				if funcName == "" {
					return nil, errors.Wrapf(err, "empty function name in func_expr_windowless")
				}
				var t base.TableSource
				if schemaName == "" && IsSystemFunction(funcName, "") {
					// If the schemaName is empty, we try to match the system function first.
					t, err = q.extractTableSourceFromSystemFunctionNew(funcName, args, nil)
					if err != nil {
						return nil, errors.Wrapf(err, "failed to extract table source from system function %s", funcName)
					}
				} else {
					t, err = q.extractTableSourceFromUDF2(item.Func_expr_windowless(), nil)
					if err != nil {
						return nil, errors.Wrapf(err, "failed to extract table source from UDF %s", funcName)
					}
				}
				if t != nil {
					// Append columns from this function to the overall table source
					tableSource.Columns = append(tableSource.Columns, t.GetQuerySpanResult()...)
				}
			}
		}
		if funcAlias != nil {
			tableSource, err := applyFuncAliasClauseToTableSource(tableSource, funcAlias)
			if err != nil {
				return nil, errors.Wrapf(err, "failed to apply function alias clause")
			}
			return tableSource, nil
		}
		return tableSource, nil
	}

	// For function tables, the columns are typically determined by:
	// 1. The function's return type (which we don't have access to without metadata)
	// 2. The alias provided (which will be applied later)
	// So we return an empty pseudo table that will be populated by the alias

	return result, nil
}

func applyAliasToTableSource(source base.TableSource, tableAliasName string, colAliases []string) (base.TableSource, error) {
	// Create a pseudo table with the alias
	pseudoTable := &base.PseudoTable{
		Name:    source.GetTableName(),
		Columns: source.GetQuerySpanResult(),
	}
	if tableAliasName != "" {
		pseudoTable.Name = tableAliasName
	}

	// Apply column aliases if provided
	if len(colAliases) > 0 {
		// Special handling for function tables that return no columns initially
		if len(pseudoTable.Columns) == 0 {
			// Create columns based on aliases
			for _, alias := range colAliases {
				pseudoTable.Columns = append(pseudoTable.Columns, base.QuerySpanResult{
					Name:          alias,
					SourceColumns: base.SourceColumnSet{},
				})
			}
		} else if len(colAliases) != len(pseudoTable.Columns) {
			// For non-empty tables, the alias count must match
			return nil, errors.Errorf("alias has %d columns but table has %d",
				len(colAliases), len(pseudoTable.Columns))
		} else {
			// Apply aliases to existing columns
			for i, alias := range colAliases {
				pseudoTable.Columns[i].Name = alias
			}
		}
	}
	return pseudoTable, nil
}

func applyTableAliasClauseToTableSource(source base.TableSource, tableAlias pgparser.ITable_alias_clauseContext) (base.TableSource, error) {
	aliasName := ""
	var colAliases []string

	// Get table alias name
	if tableAlias.Table_alias() != nil {
		aliasName = normalizePostgreSQLTableAlias(tableAlias.Table_alias())
	}

	// Get column aliases if present
	if tableAlias.Name_list() != nil {
		for _, name := range tableAlias.Name_list().AllName() {
			colAliases = append(colAliases, normalizePostgreSQLName(name))
		}
	}

	return applyAliasToTableSource(source, aliasName, colAliases)
}

func applyAliasClauseToTableSource(source base.TableSource, aliasClause pgparser.IAlias_clauseContext) (base.TableSource, error) {
	if aliasClause == nil {
		return source, nil
	}

	aliasName := ""
	var colAliases []string

	// Get table alias name
	if aliasClause.Colid() != nil {
		aliasName = NormalizePostgreSQLColid(aliasClause.Colid())
	}

	// Get column aliases if present
	if aliasClause.Name_list() != nil {
		for _, name := range aliasClause.Name_list().AllName() {
			colAliases = append(colAliases, normalizePostgreSQLName(name))
		}
	}

	return applyAliasToTableSource(source, aliasName, colAliases)
}

// applyAliasToTableSource applies an alias to a table source
func applyOptAliasClauseToTableSource(source base.TableSource, aliasClause pgparser.IOpt_alias_clauseContext) (base.TableSource, error) {
	if aliasClause == nil || aliasClause.Table_alias_clause() == nil {
		return source, nil
	}

	return applyTableAliasClauseToTableSource(source, aliasClause.Table_alias_clause())
}

func applyFuncAliasClauseToTableSource(source base.TableSource, funcAlias pgparser.IFunc_alias_clauseContext) (base.TableSource, error) {
	if funcAlias == nil {
		return source, nil
	}

	if funcAlias.Alias_clause() != nil {
		return applyAliasClauseToTableSource(source, funcAlias.Alias_clause())
	}

	aliasName := ""
	if funcAlias.Colid() != nil {
		aliasName = NormalizePostgreSQLColid(funcAlias.Colid())
	}

	var colAliases []string
	if funcAlias.Tablefuncelementlist() != nil {
		for _, name := range funcAlias.Tablefuncelementlist().AllTablefuncelement() {
			if name.Colid() != nil {
				colAliases = append(colAliases, NormalizePostgreSQLColid(name.Colid()))
			}
		}
	}

	return applyAliasToTableSource(source, aliasName, colAliases)
}

// extractColumnsFromTargetList processes the SELECT target list and returns the result columns.
// It handles *, table.*, column references, expressions, and aliases.
func (q *querySpanExtractor) extractColumnsFromTargetList(targetList pgparser.ITarget_listContext, fromFieldList []base.TableSource) ([]base.QuerySpanResult, error) {
	if targetList == nil {
		return nil, nil
	}

	var result []base.QuerySpanResult

	for _, targetEl := range targetList.AllTarget_el() {
		// Check if this is a star expansion
		if _, ok := targetEl.(*pgparser.Target_starContext); ok {
			// Handle SELECT * or SELECT table.*
			starColumns, err := q.handleStarExpansion(fromFieldList)
			if err != nil {
				return nil, errors.Wrapf(err, "failed to expand star")
			}
			result = append(result, starColumns...)
			continue
		}

		// Check if this is a labeled target (expression with optional alias)
		if labelCtx, ok := targetEl.(*pgparser.Target_labelContext); ok {
			column, err := q.handleTargetLabel(labelCtx)
			if err != nil {
				return nil, errors.Wrapf(err, "failed to process target element")
			}
			result = append(result, column)
			continue
		}
	}

	return result, nil
}

// handleStarExpansion handles SELECT * expansion
func (q *querySpanExtractor) handleStarExpansion(fromFieldList []base.TableSource) ([]base.QuerySpanResult, error) {
	// Simple * - expand all columns from all tables in FROM clause
	var result []base.QuerySpanResult
	for _, tableSource := range fromFieldList {
		result = append(result, tableSource.GetQuerySpanResult()...)
	}
	return result, nil
}

// handleTargetLabel handles a labeled target element (expression with optional alias)
func (q *querySpanExtractor) handleTargetLabel(labelCtx *pgparser.Target_labelContext) (base.QuerySpanResult, error) {
	// Get the expression
	expr := labelCtx.A_expr()
	if expr == nil {
		return base.QuerySpanResult{}, errors.New("target_label without expression")
	}

	// Extract source columns from the expression
	sourceColumns, err := q.extractSourceColumnSetFromAExpr(expr)
	if err != nil {
		return base.QuerySpanResult{}, errors.Wrapf(err, "failed to extract source columns from expression")
	}

	// Get the column name or alias
	columnName := ""
	if labelCtx.Target_alias() != nil {
		// Has an explicit alias
		targetAlias := labelCtx.Target_alias()
		if targetAlias.Collabel() != nil {
			columnName = normalizePostgreSQLCollabel(targetAlias.Collabel())
		}
	}

	if columnName == "" {
		// No alias, try to extract name from expression
		columnName, err = q.extractFieldNameFromAExpr(expr)
		if err != nil || columnName == "" {
			// Use default unknown column name
			columnName = pgUnknownFieldName
		}
	}

	return base.QuerySpanResult{
		Name:          columnName,
		SourceColumns: sourceColumns,
	}, nil
}

// extractFieldNameFromAExpr attempts to extract a meaningful column name from an expression.
// Returns "?column?" if no meaningful name can be derived.
func (q *querySpanExtractor) extractFieldNameFromAExpr(expr pgparser.IA_exprContext) (string, error) {
	if expr == nil {
		return pgUnknownFieldName, nil
	}

	// Try to find a column reference in the expression
	columnName := q.findColumnNameInExpression(expr)
	if columnName != "" {
		return columnName, nil
	}

	// Default to unknown column name for other expression types
	return "?column?pgUnknownFieldName", nil
}

// findColumnNameInExpression recursively searches for a column name in an expression
func (q *querySpanExtractor) findColumnNameInExpression(node antlr.ParseTree) string {
	if node == nil {
		return ""
	}

	// Check if this is a column reference
	if columnRef, ok := node.(pgparser.IColumnrefContext); ok {
		// Get the column name from the columnref
		if columnRef.Colid() != nil {
			colName := NormalizePostgreSQLColid(columnRef.Colid())

			// Check if there's indirection (qualified name)
			if columnRef.Indirection() != nil {
				parts := normalizePostgreSQLIndirection(columnRef.Indirection())
				if len(parts) > 0 && parts[len(parts)-1] != "*" {
					// Return the last part as the column name
					return parts[len(parts)-1]
				}
			}
			// Simple column name
			return colName
		}
	}

	switch ctx := node.(type) {
	case pgparser.IColumnrefContext:
		if ctx.Colid() != nil {
			return NormalizePostgreSQLColid(ctx.Colid())
		}
	case pgparser.IFunc_exprContext:
		// return the function name as the column name
		_, funcName, _, err := q.extractFunctionElementFromFuncExpr(ctx)
		if err == nil && funcName != "" {
			return funcName
		}
		return pgUnknownFieldName
	case pgparser.IFunc_expr_windowlessContext:
		_, funcName, _, err := q.extractFunctionElementFromFuncExprWindowless(ctx)
		if err == nil && funcName != "" {
			return funcName
		}
		return pgUnknownFieldName
	}

	// Recursively search children
	for i := 0; i < node.GetChildCount(); i++ {
		if child, ok := node.GetChild(i).(antlr.ParseTree); ok {
			// Skip terminal nodes
			if _, isTerminal := child.(antlr.TerminalNode); isTerminal {
				continue
			}
			if name := q.findColumnNameInExpression(child); name != "" {
				return name
			}
		}
	}

	return ""
}

// extractSourceColumnSetFromAExpr extracts source columns from an a_expr node
func (q *querySpanExtractor) extractSourceColumnSetFromAExpr(expr pgparser.IA_exprContext) (base.SourceColumnSet, error) {
	if expr == nil {
		return base.SourceColumnSet{}, nil
	}

	// Use the generic node traversal
	return q.extractSourceColumnSetFromNode(expr)
}

// extractSourceColumnSetFromNode recursively extracts source columns from any parse tree node.
// This follows the pg_query_go pattern but adapted for ANTLR's tree structure.
func (q *querySpanExtractor) extractSourceColumnSetFromNode(node antlr.ParseTree) (base.SourceColumnSet, error) {
	if node == nil {
		return base.SourceColumnSet{}, nil
	}

	result := make(base.SourceColumnSet)

	// Handle specific node types (similar to pg_query_go's switch statement)
	switch ctx := node.(type) {
	case pgparser.IColumnrefContext:
		// Handle column reference directly
		return q.processColumnRef(ctx)
	case pgparser.IFunc_exprContext:
		// Handle function calls
		// Process function arguments recursively
		argsResult, err := q.extractSourceColumnSetFromFuncExpr(ctx)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to extract from function expression")
		}
		if udfResult, err := q.extractSourceColumnSetFromUDF2(ctx); err == nil {
			argsResult, _ = base.MergeSourceColumnSet(argsResult, udfResult)
		}
		return argsResult, nil
	case pgparser.ISelect_with_parensContext:
		sourceColumnSet := make(base.SourceColumnSet)
		// Subquery in SELECT fields is special.
		// It can be the non-associated or associated subquery.
		// For associated subquery, we should set the fromFieldList as the outerSchemaInfo.
		// So that the subquery can access the outer schema.
		// The reason for new extractor is that we still need the current fromFieldList, overriding it is not expected.
		subqueryExtractor := &querySpanExtractor{
			ctx:                     q.ctx,
			defaultDatabase:         q.defaultDatabase,
			searchPath:              q.searchPath,
			metaCache:               q.metaCache,
			gCtx:                    q.gCtx,
			ctes:                    q.ctes,
			outerTableSources:       append(q.outerTableSources, q.tableSourcesFrom...),
			tableSourcesFrom:        []base.TableSource{},
			sourceColumnsInFunction: make(base.SourceColumnSet),
		}
		tableSource, err := subqueryExtractor.extractTableSourceFromSelectWithParens(ctx)
		if err != nil {
			return base.SourceColumnSet{}, errors.Wrapf(err, "failed to extract span result from subquery %s", ctx.GetText())
		}
		spanResult := tableSource.GetQuerySpanResult()

		for _, field := range spanResult {
			sourceColumnSet, _ = base.MergeSourceColumnSet(sourceColumnSet, field.SourceColumns)
		}
		return sourceColumnSet, nil
	// Add more specific cases as we discover we need them...

	default:
		// For any unhandled node type, recursively visit all children
		// This is the generic fallback that ensures we don't miss any column references
		for i := 0; i < node.GetChildCount(); i++ {
			if child, ok := node.GetChild(i).(antlr.ParseTree); ok {
				childSources, err := q.extractSourceColumnSetFromNode(child)
				if err != nil {
					// For now, propagate errors. Could also log and continue.
					return nil, err
				}
				result, _ = base.MergeSourceColumnSet(result, childSources)
			}
		}
	}

	return result, nil
}

// processColumnRef extracts source columns from a column reference.
func (q *querySpanExtractor) processColumnRef(ctx pgparser.IColumnrefContext) (base.SourceColumnSet, error) {
	if ctx == nil {
		return base.SourceColumnSet{}, nil
	}

	// Extract schema, table, column from the columnref
	var schemaName, tableName, columnName string

	// Get the base column name
	if ctx.Colid() != nil {
		columnName = NormalizePostgreSQLColid(ctx.Colid())
	}

	// Handle indirection (qualified names like table.column or schema.table.column)
	if ctx.Indirection() != nil {
		parts := normalizePostgreSQLIndirection(ctx.Indirection())

		// Check for star expansion (should have been handled elsewhere, but check anyway)
		if len(parts) > 0 && parts[len(parts)-1] == "*" {
			// This is table.* or schema.table.* - should not happen in regular column refs
			// Return empty set as this should be handled by star expansion logic
			return base.SourceColumnSet{}, nil
		}

		switch len(parts) {
		case 1:
			// table.column case
			tableName = columnName
			columnName = parts[0]
		case 2:
			// schema.table.column case
			schemaName = columnName
			tableName = parts[0]
			columnName = parts[1]
		default:
			// More complex indirection, just take the last part as column
			if len(parts) > 0 {
				columnName = parts[len(parts)-1]
			}
		}
	}

	// Special handling for record/composite types
	// If we have a table name but no column name, it might be a whole-row reference
	if tableName != "" && columnName == "" {
		// This is a whole-row reference like "table" in row_to_json(table)
		// We need to return all columns from that table
		tableSource := q.findTableInFromNew(tableName)
		if tableSource != nil {
			result := make(base.SourceColumnSet)
			for _, column := range tableSource.GetQuerySpanResult() {
				result, _ = base.MergeSourceColumnSet(result, column.SourceColumns)
			}
			return result, nil
		}
	}

	// Look up the column source
	sources, columnSourceOk := q.getFieldColumnSource(schemaName, tableName, columnName)
	// The column ref in function call can be record type, such as row_to_json.
	if !columnSourceOk {
		if schemaName == "" {
			tableSource := q.findTableInFrom(tableName, columnName)
			if tableSource == nil {
				return base.SourceColumnSet{}, &parsererror.ResourceNotFoundError{
					Err:      errors.New("cannot find the column ref"),
					Database: &q.defaultDatabase,
					Schema:   &schemaName,
					Table:    &tableName,
					Column:   &columnName,
				}
			}
			querySpanResult := tableSource.GetQuerySpanResult()
			result := make(base.SourceColumnSet)
			for _, span := range querySpanResult {
				result, _ = base.MergeSourceColumnSet(result, span.SourceColumns)
			}
			return result, nil
		}
		return base.SourceColumnSet{}, &parsererror.ResourceNotFoundError{
			Err:      errors.New("cannot find the column ref"),
			Database: &q.defaultDatabase,
			Schema:   &schemaName,
			Table:    &tableName,
			Column:   &columnName,
		}
	}
	return sources, nil
}

// findTableInFromNew finds a table source by name in the FROM clause or CTEs.
func (q *querySpanExtractor) findTableInFromNew(tableName string) base.TableSource {
	// Search in FROM clause tables
	for _, tableSource := range q.tableSourcesFrom {
		if tableSource.GetTableName() == tableName {
			return tableSource
		}
	}

	// Search in CTEs
	for i := len(q.ctes) - 1; i >= 0; i-- {
		if q.ctes[i].Name == tableName {
			return q.ctes[i]
		}
	}

	// Search in outer tables (for correlated subqueries)
	for _, tableSource := range q.outerTableSources {
		if tableSource.GetTableName() == tableName {
			return tableSource
		}
	}

	return nil
}

// extractSourceColumnSetFromFuncExpr extracts source columns from a function expression
func (q *querySpanExtractor) extractSourceColumnSetFromFuncExpr(funcExpr pgparser.IFunc_exprContext) (base.SourceColumnSet, error) {
	result := make(base.SourceColumnSet)

	// Handle function application
	if funcExpr.Func_application() != nil {
		funcApp := funcExpr.Func_application()
		// Process function arguments
		if funcApp.Func_arg_list() != nil {
			for _, arg := range funcApp.Func_arg_list().AllFunc_arg_expr() {
				if arg.A_expr() != nil {
					argSources, err := q.extractSourceColumnSetFromAExpr(arg.A_expr())
					if err != nil {
						return nil, err
					}
					result, _ = base.MergeSourceColumnSet(result, argSources)
				}
			}
		}
	}

	// Handle common subexpressions (COALESCE, GREATEST, LEAST, etc.)
	if funcExpr.Func_expr_common_subexpr() != nil {
		// Recursive extract all nodes in common subexpr
		for i := 0; i < funcExpr.Func_expr_common_subexpr().GetChildCount(); i++ {
			if child, ok := funcExpr.Func_expr_common_subexpr().GetChild(i).(antlr.ParseTree); ok {
				// Skip terminal nodes
				if _, isTerminal := child.(antlr.TerminalNode); isTerminal {
					continue
				}
				childSources, err := q.extractSourceColumnSetFromNode(child)
				if err != nil {
					return nil, err
				}
				result, _ = base.MergeSourceColumnSet(result, childSources)
			}
		}
		// Handle XML functions that may have additional expressions
		// TODO: Add more specific handling for XML functions if needed
	}

	// TODO: Handle window functions if needed

	filterClause := funcExpr.Filter_clause()
	if filterClause != nil {
		r, err := q.extractSourceColumnSetFromNode(filterClause)
		if err != nil {
			return nil, err
		}
		result, _ = base.MergeSourceColumnSet(result, r)
	}
	return result, nil
}

func getSelectNoParensFromSelectStmt(selectstmt pgparser.ISelectstmtContext) pgparser.ISelect_no_parensContext {
	if selectstmt == nil {
		return nil
	}

	// Direct check
	if selectstmt.Select_no_parens() != nil {
		return selectstmt.Select_no_parens()
	}

	// Check nested select_with_parens
	if selectstmt.Select_with_parens() != nil {
		return getSelectNoParensFromSelectWithParens(selectstmt.Select_with_parens())
	}

	return nil
}

func getSelectNoParensFromSelectWithParens(selectWithParens pgparser.ISelect_with_parensContext) pgparser.ISelect_no_parensContext {
	if selectWithParens == nil {
		return nil
	}

	// Direct check
	if selectWithParens.Select_no_parens() != nil {
		return selectWithParens.Select_no_parens()
	}

	// Recurse for nested parentheses
	if selectWithParens.Select_with_parens() != nil {
		return getSelectNoParensFromSelectWithParens(selectWithParens.Select_with_parens())
	}

	return nil
}

// extractTableSourceFromSelectNoParensNew handles SELECT without parentheses.
func (q *querySpanExtractor) extractTableSourceFromSelectNoParensNew(ctx pgparser.ISelect_no_parensContext) (base.TableSource, error) {
	if ctx == nil {
		return nil, errors.New("select_no_parens context is nil")
	}

	// Handle WITH clause if present
	if ctx.With_clause() != nil {
		if err := q.handleWithClause(ctx.With_clause()); err != nil {
			return nil, errors.Wrapf(err, "failed to handle WITH clause")
		}
	}

	// Handle SELECT clause
	if ctx.Select_clause() != nil {
		return q.extractTableSourceFromSelectClause(ctx.Select_clause())
	}

	return nil, errors.New("no select clause found")
}

// extractTableSourceFromSelectClause handles the main SELECT clause.
func (q *querySpanExtractor) extractTableSourceFromSelectClause(ctx pgparser.ISelect_clauseContext) (base.TableSource, error) {
	if ctx == nil {
		return nil, errors.New("select_clause context is nil")
	}

	// Create result table
	result := &base.PseudoTable{
		Name:    "",
		Columns: []base.QuerySpanResult{},
	}

	// Handle simple SELECT
	for _, simpleSelect := range ctx.AllSimple_select_intersect() {
		for _, primary := range simpleSelect.AllSimple_select_pramary() {
			// Handle FROM clause
			if primary.From_clause() != nil && primary.From_clause().From_list() != nil {
				for _, fromItem := range primary.From_clause().From_list().AllTable_ref() {
					tableSource, err := q.extractTableSourceFromTableRef(fromItem)
					if err != nil {
						return nil, err
					}
					q.tableSourcesFrom = append(q.tableSourcesFrom, tableSource)
				}
			}

			// Handle target list
			if primary.Opt_target_list() != nil && primary.Opt_target_list().Target_list() != nil {
				for _, target := range primary.Opt_target_list().Target_list().AllTarget_el() {
					if err := q.handleTargetElement(target, result); err != nil {
						return nil, err
					}
				}
			}
		}
	}

	return result, nil
}

// handleTargetElement processes a single target element in SELECT list
func (q *querySpanExtractor) handleTargetElement(target pgparser.ITarget_elContext, result *base.PseudoTable) error {
	// Check if this is a star expansion
	if _, ok := target.(*pgparser.Target_starContext); ok {
		// Handle SELECT * or SELECT table.*
		starColumns, err := q.handleStarExpansion(q.tableSourcesFrom)
		if err != nil {
			return errors.Wrapf(err, "failed to expand star")
		}
		result.Columns = append(result.Columns, starColumns...)
		return nil
	}

	// Check if this is a labeled target (expression with optional alias)
	if labelCtx, ok := target.(*pgparser.Target_labelContext); ok {
		column, err := q.handleTargetLabel(labelCtx)
		if err != nil {
			return errors.Wrapf(err, "failed to process target element")
		}
		result.Columns = append(result.Columns, column)
		return nil
	}

	return nil
}

// handleWithClause processes WITH clause (CTEs)
func (q *querySpanExtractor) handleWithClause(withClause pgparser.IWith_clauseContext) error {
	if withClause == nil || withClause.Cte_list() == nil {
		return nil
	}

	for _, cte := range withClause.Cte_list().AllCommon_table_expr() {
		if err := q.handleCommonTableExpr(cte); err != nil {
			return err
		}
	}

	return nil
}

// handleCommonTableExpr processes a single CTE
func (q *querySpanExtractor) handleCommonTableExpr(cte pgparser.ICommon_table_exprContext) error {
	if cte == nil {
		return nil
	}

	// Extract CTE name
	cteName := ""
	if cte.Name() != nil {
		cteName = normalizePostgreSQLName(cte.Name())
	}

	// Extract table source from CTE query
	var tableSource base.TableSource
	var err error

	// CTEs contain preparablestmt which can be SELECT, INSERT, UPDATE, DELETE
	// For now, we only handle SELECT
	if cte.Preparablestmt() != nil {
		// TODO: Check for RECURSIVE keyword - for now treat all as non-recursive
		tableSource, err = q.extractTableSourceFromNonRecursiveCTENew(cte)
	}

	if err != nil {
		return errors.Wrapf(err, "failed to extract CTE %s", cteName)
	}

	// Apply column aliases if present
	if cte.Opt_name_list() != nil && cte.Opt_name_list().Name_list() != nil {
		columnAliases := normalizePostgreSQLNameList(cte.Opt_name_list().Name_list())
		querySpanResults := tableSource.GetQuerySpanResult()

		if len(columnAliases) > 0 && len(columnAliases) != len(querySpanResults) {
			return errors.Errorf("CTE %s has %d columns but alias has %d columns",
				cteName, len(querySpanResults), len(columnAliases))
		}

		for i, alias := range columnAliases {
			querySpanResults[i].Name = alias
		}
	}

	// Add CTE to the list
	pseudoTable := &base.PseudoTable{
		Name:    cteName,
		Columns: tableSource.GetQuerySpanResult(),
	}
	q.ctes = append(q.ctes, pseudoTable)

	return nil
}

// extractTableSourceFromNonRecursiveCTENew handles non-recursive CTEs
func (q *querySpanExtractor) extractTableSourceFromNonRecursiveCTENew(cte pgparser.ICommon_table_exprContext) (base.TableSource, error) {
	if cte.Preparablestmt() == nil {
		return nil, errors.New("CTE without preparable statement")
	}

	// Try to get the SELECT statement from preparablestmt
	preparable := cte.Preparablestmt()
	if preparable.Selectstmt() != nil {
		selectNoParens := getSelectNoParensFromSelectStmt(preparable.Selectstmt())
		if selectNoParens != nil {
			return q.extractTableSourceFromSelectNoParensNew(selectNoParens)
		}
	}

	// TODO: Handle INSERT, UPDATE, DELETE in CTEs if needed

	return nil, errors.New("failed to extract select from CTE")
}

func (q *querySpanExtractor) extractSourceColumnSetFromUDF2(funcExpr pgparser.IFunc_exprContext) (base.SourceColumnSet, error) {
	schemaName, funcName, err := q.extractFunctionNameFromFuncExpr(funcExpr)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to extract function name")
	}
	if schemaName == "" && IsSystemFunction(funcName, "") {
		return base.SourceColumnSet{}, nil
	}
	nArgs, err := q.extractNFunctionArgsFromFuncExpr(funcExpr)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to extract function args")
	}

	result := make(base.SourceColumnSet)
	tableSource, err := q.findFunctionDefine(schemaName, funcName, nArgs)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to find function definition for %s.%s/%d", schemaName, funcName, nArgs)
	}
	for _, span := range tableSource.GetQuerySpanResult() {
		result, _ = base.MergeSourceColumnSet(result, span.SourceColumns)
	}
	return result, nil
}

func (q *querySpanExtractor) extractFunctionElementFromFuncExpr(funcExpr pgparser.IFunc_exprContext) (string, string, []antlr.ParseTree, error) {
	args := getArgumentsFromFunctionExpr(funcExpr)
	switch {
	case funcExpr.Func_application() != nil:
		funcName := funcExpr.Func_application().Func_name()
		names := NormalizePostgreSQLFuncName(funcName)
		switch len(names) {
		case 2:
			return names[0], names[1], args, nil
		case 1:
			return "", names[0], args, nil
		default:
			return "", "", nil, errors.New("invalid function name")
		}
	case funcExpr.Json_aggregate_func() != nil:
		if funcExpr.Json_aggregate_func().JSON_OBJECTAGG() != nil {
			return "", "json_objectagg", args, nil
		} else if funcExpr.Json_aggregate_func().JSON_ARRAYAGG() != nil {
			return "", "json_arrayagg", args, nil
		}
	case funcExpr.Func_expr_common_subexpr() != nil:
		// Builtin function, return the first token text as function name
		if funcExpr.Func_expr_common_subexpr().GetStart() != nil {
			return "", strings.ToLower(funcExpr.Func_expr_common_subexpr().GetStart().GetText()), args, nil
		}
	}
	return "", "", nil, errors.New("unable to extract function name")
}

func (q *querySpanExtractor) extractFunctionElementFromFuncExprWindowless(funcExprWindowless pgparser.IFunc_expr_windowlessContext) (string, string, []antlr.ParseTree, error) {
	args := getArgumentsFromFunctionExprWindowless(funcExprWindowless)
	switch {
	case funcExprWindowless.Func_application() != nil:
		funcName := funcExprWindowless.Func_application().Func_name()
		names := NormalizePostgreSQLFuncName(funcName)
		switch len(names) {
		case 2:
			return names[0], names[1], args, nil
		case 1:
			return "", names[0], args, nil
		default:
			return "", "", nil, errors.New("invalid function name")
		}
	case funcExprWindowless.Json_aggregate_func() != nil:
		if funcExprWindowless.Json_aggregate_func().JSON_OBJECTAGG() != nil {
			return "", "json_objectagg", args, nil
		} else if funcExprWindowless.Json_aggregate_func().JSON_ARRAYAGG() != nil {
			return "", "json_arrayagg", args, nil
		}
	case funcExprWindowless.Func_expr_common_subexpr() != nil:
		// Builtin function, return the first token text as function name
		if funcExprWindowless.Func_expr_common_subexpr().GetStart() != nil {
			return "", strings.ToLower(funcExprWindowless.Func_expr_common_subexpr().GetStart().GetText()), args, nil
		}
	}
	return "", "", nil, errors.New("unable to extract function name")
}

func (q *querySpanExtractor) extractFunctionNameFromFuncExpr(funcExpr pgparser.IFunc_exprContext) (string, string, error) {
	switch {
	case funcExpr.Func_application() != nil:
		funcName := funcExpr.Func_application().Func_name()
		if funcName.Colid() != nil {
			names := NormalizePostgreSQLFuncName(funcName)
			switch len(names) {
			case 2:
				return names[0], names[1], nil
			case 1:
				return "", names[0], nil
			default:
				return "", "", errors.New("invalid function name")
			}
		} else if t := funcName.Type_function_name(); t != nil {
			// No indirection, just a simple function name
			s := t.GetText()
			if t.Identifier() != nil {
				s = normalizePostgreSQLIdentifier(t.Identifier())
			}
			return "", s, nil
		} else {
			// Builtin function without schema
			return "", strings.ToLower(funcName.GetText()), nil
		}
	case funcExpr.Json_aggregate_func() != nil:
		if funcExpr.Json_aggregate_func().JSON_OBJECTAGG() != nil {
			return "", "json_objectagg", nil
		} else if funcExpr.Json_aggregate_func().JSON_ARRAYAGG() != nil {
			return "", "json_arrayagg", nil
		}
	case funcExpr.Func_expr_common_subexpr() != nil:
		// Builtin function, return the first token text as function name
		if funcExpr.Func_expr_common_subexpr().GetStart() != nil {
			return "", strings.ToLower(funcExpr.Func_expr_common_subexpr().GetStart().GetText()), nil
		}
	}
	return "", "", errors.New("unable to extract function name")
}

func (q *querySpanExtractor) extractNFunctionArgsFromFuncExpr(funcExpr pgparser.IFunc_exprContext) (int, error) {
	if funcExpr == nil {
		return 0, nil
	}

	// This handles generic function calls like my_func(a, b).
	if fa := funcExpr.Func_application(); fa != nil {
		if fa.Func_arg_list() != nil {
			return len(fa.Func_arg_list().AllFunc_arg_expr()), nil
		}
		// This handles cases like my_func(a) without a list rule.
		if fa.Func_arg_expr() != nil {
			return 1, nil
		}
		return 0, nil
	}

	// This handles JSON aggregate functions like JSON_AGG(expr).
	if f := funcExpr.Json_aggregate_func(); f != nil {
		// Based on Postgres grammar, these functions (JSON_AGG, JSONB_AGG, JSON_OBJECT_AGG, JSONB_OBJECT_AGG)
		// typically take one or two main arguments. Returning 1 is a reasonable assumption for the primary expression argument.
		return 1, nil
	}

	// This handles a large set of built-in functions with special syntax.
	if f := funcExpr.Func_expr_common_subexpr(); f != nil {
		// Functions with no parentheses have zero arguments (e.g., CURRENT_DATE, CURRENT_USER).
		if f.OPEN_PAREN() == nil {
			return 0, nil
		}

		// --- List-based Functions ---
		// These functions take a variable number of arguments in a simple list.
		if f.COALESCE() != nil || f.GREATEST() != nil || f.LEAST() != nil || f.XMLCONCAT() != nil {
			if list := f.Expr_list(); list != nil {
				return len(list.AllA_expr()), nil
			}
			return 0, nil
		}

		// --- Functions with Specific Keyword-based Syntax ---

		if f.COLLATION() != nil { // COLLATION FOR (a_expr)
			return 1, nil
		}

		if f.CURRENT_TIME() != nil || f.CURRENT_TIMESTAMP() != nil || f.LOCALTIME() != nil || f.LOCALTIMESTAMP() != nil {
			// e.g., CURRENT_TIME(precision)
			if f.Iconst() != nil {
				return 1, nil
			}
			return 0, nil
		}

		if f.CAST() != nil || f.TREAT() != nil { // CAST(expr AS type), TREAT(expr AS type)
			return 2, nil
		}

		if f.EXTRACT() != nil { // EXTRACT(field FROM source)
			if f.Extract_list() != nil {
				// extract_list contains 'extract_arg' and 'a_expr'.
				return 2, nil
			}
			return 0, nil
		}

		if f.NORMALIZE() != nil { // NORMALIZE(string [, form])
			count := 1 // 'string' argument is mandatory
			if f.Unicode_normal_form() != nil {
				count++
			}
			return count, nil
		}

		if f.OVERLAY() != nil { // OVERLAY(string PLACING new_substring FROM start [FOR count])
			if list := f.Overlay_list(); list != nil {
				return len(list.AllA_expr()), nil // Counts all expressions
			}
			return 0, nil
		}

		if f.POSITION() != nil { // POSITION(substring IN string)
			if f.Position_list() != nil {
				return 2, nil
			}
			return 0, nil
		}

		if f.SUBSTRING() != nil { // SUBSTRING(string [FROM start] [FOR count])
			if list := f.Substr_list(); list != nil {
				return len(list.AllA_expr()), nil
			}
			return 0, nil
		}

		if f.TRIM() != nil { // TRIM([LEADING|TRAILING|BOTH] [characters] FROM string)
			if list := f.Trim_list(); list != nil {
				count := 0
				if list.A_expr() != nil { // The optional 'characters' argument
					count++
				}
				if list.Expr_list() != nil { // The mandatory 'string' argument
					count += len(list.Expr_list().AllA_expr())
				}
				return count, nil
			}
			return 0, nil
		}

		if f.NULLIF() != nil { // NULLIF(value1, value2)
			return len(f.AllA_expr()), nil // Expects 2
		}

		// --- XML Functions ---
		if f.XMLELEMENT() != nil {
			count := 0
			if f.Collabel() != nil {
				count++
			} // The element name
			if f.Xml_attributes() != nil {
				count += len(f.Xml_attributes().Xml_attribute_list().AllXml_attribute_el())
			}
			if f.Expr_list() != nil {
				count += len(f.Expr_list().AllA_expr())
			}
			return count, nil
		}
		if f.XMLEXISTS() != nil {
			return 2, nil
		} // XMLEXISTS(xpath PASSING xml)
		if f.XMLFOREST() != nil {
			if list := f.Xml_attribute_list(); list != nil {
				return len(list.AllXml_attribute_el()), nil
			}
			return 0, nil
		}
		if f.XMLPARSE() != nil { // XMLPARSE(DOCUMENT|CONTENT string_value [WHITESPACE option])
			count := 1 // string_value
			if f.Xml_whitespace_option() != nil {
				count++
			}
			return count, nil
		}
		if f.XMLPI() != nil { // XMLPI(NAME target [, content])
			count := 1 // target
			if len(f.AllA_expr()) > 0 {
				count++
			}
			return count, nil
		}
		if f.XMLROOT() != nil {
			count := 2 // xml and version
			if f.Opt_xml_root_standalone() != nil {
				count++
			}
			return count, nil
		}
		if f.XMLSERIALIZE() != nil {
			return 2, nil
		} // XMLSERIALIZE(CONTENT value AS type)

		// --- JSON Functions ---
		if f.JSON_OBJECT() != nil {
			if list := f.Func_arg_list(); list != nil {
				return len(list.AllFunc_arg_expr()), nil
			}
			if list := f.Json_name_and_value_list(); list != nil {
				return len(list.AllJson_name_and_value()), nil
			}
			return 0, nil
		}
		if f.JSON_ARRAY() != nil {
			if list := f.Json_value_expr_list(); list != nil {
				return len(list.AllJson_value_expr()), nil
			}
			if f.Select_no_parens() != nil {
				return 1, nil
			} // subquery
			return 0, nil
		}
		if f.JSON() != nil || f.JSON_SCALAR() != nil || f.JSON_SERIALIZE() != nil {
			return 1, nil
		}
		if f.MERGE_ACTION() != nil {
			return 0, nil
		}
		if f.JSON_QUERY() != nil || f.JSON_EXISTS() != nil || f.JSON_VALUE() != nil {
			count := 2 // json_value_expr, a_expr
			return count, nil
		}
	}

	return 0, nil
}

func getArgumentsFromFunctionExpr(funcExpr pgparser.IFunc_exprContext) []antlr.ParseTree {
	var result []antlr.ParseTree
	if funcExpr == nil {
		return result
	}
	if fa := funcExpr.Func_application(); fa != nil {
		if fa.Func_arg_list() != nil {
			result = append(result, getArgumentsFromFuncArgList(fa.Func_arg_list())...)
		}
		if fa.Func_arg_expr() != nil {
			result = append(result, fa.Func_arg_expr())
		}
		// TODO(zp): Handle *, ALL, DISTINCT if needed
		return result
	}
	if f := funcExpr.Func_expr_common_subexpr(); f != nil {
		if f.COLLATION() != nil {
			result = append(result, f.A_expr(0))
		} else if f.CURRENT_DATE() != nil {
			// No arguments
		} else if f.CURRENT_TIME() != nil || f.CURRENT_TIMESTAMP() != nil || f.LOCALTIME() != nil || f.LOCALTIMESTAMP() != nil {
			if f.Iconst() != nil {
				result = append(result, f.Iconst())
			}
		} else if f.CURRENT_ROLE() != nil || f.CURRENT_USER() != nil || f.SESSION_USER() != nil || f.USER() != nil || f.CURRENT_CATALOG() != nil || f.CURRENT_SCHEMA() != nil {
			// No arguments
		} else if f.CAST() != nil || f.TREAT() != nil {
			result = append(result, f.A_expr(0))
		} else if f.EXTRACT() != nil {
			if f.Extract_list() != nil {
				result = append(result, f.Extract_list().A_expr())
			}
		} else if f.NORMALIZE() != nil {
			result = append(result, f.A_expr(0))
		} else if f.OVERLAY() != nil {
			if list := f.Overlay_list(); list != nil {
				for _, expr := range list.AllA_expr() {
					result = append(result, expr)
				}
			}
		} else if f.POSITION() != nil {
			if list := f.Position_list(); list != nil {
				for _, child := range list.GetChildren() {
					if expr, ok := child.(pgparser.IA_exprContext); ok {
						result = append(result, expr)
					}
				}
			}
		} else if f.SUBSTRING() != nil {
			if list := f.Substr_list(); list != nil {
				for _, expr := range list.AllA_expr() {
					result = append(result, expr)
				}
			}
		} else if f.TRIM() != nil {
			if list := f.Trim_list(); list != nil {
				if list.A_expr() != nil {
					result = append(result, list.A_expr())
				}
				if list.Expr_list() != nil {
					for _, expr := range list.Expr_list().AllA_expr() {
						result = append(result, expr)
					}
				}
			}
		} else if f.NULLIF() != nil {
			for _, expr := range f.AllA_expr() {
				result = append(result, expr)
			}
		} else if f.COALESCE() != nil || f.GREATEST() != nil || f.LEAST() != nil {
			if f.Expr_list() != nil {
				for _, expr := range f.Expr_list().AllA_expr() {
					result = append(result, expr)
				}
			}
		} else if f.XMLELEMENT() != nil {
			if f.Expr_list() != nil {
				for _, expr := range f.Expr_list().AllA_expr() {
					result = append(result, expr)
				}
			}
		} else if f.XMLEXISTS() != nil {
			for _, expr := range f.AllA_expr() {
				result = append(result, expr)
			}
		} else if f.XMLFOREST() != nil {
			if list := f.Xml_attribute_list(); list != nil {
				for _, attr := range list.AllXml_attribute_el() {
					result = append(result, attr.A_expr())
				}
			}
		} else if f.XMLPARSE() != nil {
			result = append(result, f.A_expr(0))
		} else if f.XMLPI() != nil {
			if len(f.AllA_expr()) > 0 {
				result = append(result, f.A_expr(0))
			}
		} else if f.XMLROOT() != nil {
			result = append(result, f.A_expr(0))
		} else if f.XMLSERIALIZE() != nil {
			result = append(result, f.A_expr(0))
		} else if f.JSON_OBJECT() != nil {
			if list := f.Func_arg_list(); list != nil {
				result = append(result, getArgumentsFromFuncArgList(list)...)
			}
			if list := f.Json_name_and_value_list(); list != nil {
				result = append(result, getArgumentsFromJsonNameAndValueList(list)...)
			}
			if list := f.Json_value_expr_list(); list != nil {
				result = append(result, getArgumentsFromJsonValueExprList(list)...)
			}
		} else if f.JSON_ARRAY() != nil {
			if list := f.Json_value_expr_list(); list != nil {
				result = append(result, getArgumentsFromJsonValueExprList(list)...)
			}
			if list := f.Select_no_parens(); list != nil {
				result = append(result, list)
			}
		} else if f.JSON() != nil {
			result = append(result, f.Json_value_expr())
		} else if f.JSON_SCALAR() != nil {
			result = append(result, f.A_expr(0))
		} else if f.JSON_SERIALIZE() != nil {
			result = append(result, f.Json_value_expr())
		} else if f.JSON_QUERY() != nil {
			result = append(result, f.Json_value_expr())
			result = append(result, f.A_expr(0))
		} else if f.JSON_EXISTS() != nil {
			result = append(result, f.Json_value_expr())
			result = append(result, f.A_expr(0))
		} else if f.JSON_VALUE() != nil {
			result = append(result, f.Json_value_expr())
			result = append(result, f.A_expr(0))
		}
	}
	return result
}

func getArgumentsFromFunctionExprWindowless(funcExprWindowless pgparser.IFunc_expr_windowlessContext) []antlr.ParseTree {
	var result []antlr.ParseTree
	if funcExprWindowless == nil {
		return result
	}
	if fa := funcExprWindowless.Func_application(); fa != nil {
		if fa.Func_arg_list() != nil {
			result = append(result, getArgumentsFromFuncArgList(fa.Func_arg_list())...)
		}
		if fa.Func_arg_expr() != nil {
			result = append(result, fa.Func_arg_expr())
		}
		// TODO(zp): Handle *, ALL, DISTINCT if needed
		return result
	}
	if f := funcExprWindowless.Func_expr_common_subexpr(); f != nil {
		if f.COLLATION() != nil {
			result = append(result, f.A_expr(0))
		} else if f.CURRENT_DATE() != nil {
			// No arguments
		} else if f.CURRENT_TIME() != nil || f.CURRENT_TIMESTAMP() != nil || f.LOCALTIME() != nil || f.LOCALTIMESTAMP() != nil {
			if f.Iconst() != nil {
				result = append(result, f.Iconst())
			}
		} else if f.CURRENT_ROLE() != nil || f.CURRENT_USER() != nil || f.SESSION_USER() != nil || f.USER() != nil || f.CURRENT_CATALOG() != nil || f.CURRENT_SCHEMA() != nil {
			// No arguments
		} else if f.CAST() != nil || f.TREAT() != nil {
			result = append(result, f.A_expr(0))
		} else if f.EXTRACT() != nil {
			if f.Extract_list() != nil {
				result = append(result, f.Extract_list().A_expr())
			}
		} else if f.NORMALIZE() != nil {
			result = append(result, f.A_expr(0))
		} else if f.OVERLAY() != nil {
			if list := f.Overlay_list(); list != nil {
				for _, expr := range list.AllA_expr() {
					result = append(result, expr)
				}
			}
		} else if f.POSITION() != nil {
			if list := f.Position_list(); list != nil {
				for _, child := range list.GetChildren() {
					if expr, ok := child.(pgparser.IA_exprContext); ok {
						result = append(result, expr)
					}
				}
			}
		} else if f.SUBSTRING() != nil {
			if list := f.Substr_list(); list != nil {
				for _, expr := range list.AllA_expr() {
					result = append(result, expr)
				}
			}
		} else if f.TRIM() != nil {
			if list := f.Trim_list(); list != nil {
				if list.A_expr() != nil {
					result = append(result, list.A_expr())
				}
				if list.Expr_list() != nil {
					for _, expr := range list.Expr_list().AllA_expr() {
						result = append(result, expr)
					}
				}
			}
		} else if f.NULLIF() != nil {
			for _, expr := range f.AllA_expr() {
				result = append(result, expr)
			}
		} else if f.COALESCE() != nil || f.GREATEST() != nil || f.LEAST() != nil {
			if f.Expr_list() != nil {
				for _, expr := range f.Expr_list().AllA_expr() {
					result = append(result, expr)
				}
			}
		} else if f.XMLELEMENT() != nil {
			if f.Expr_list() != nil {
				for _, expr := range f.Expr_list().AllA_expr() {
					result = append(result, expr)
				}
			}
		} else if f.XMLEXISTS() != nil {
			for _, expr := range f.AllA_expr() {
				result = append(result, expr)
			}
		} else if f.XMLFOREST() != nil {
			if list := f.Xml_attribute_list(); list != nil {
				for _, attr := range list.AllXml_attribute_el() {
					result = append(result, attr.A_expr())
				}
			}
		} else if f.XMLPARSE() != nil {
			result = append(result, f.A_expr(0))
		} else if f.XMLPI() != nil {
			if len(f.AllA_expr()) > 0 {
				result = append(result, f.A_expr(0))
			}
		} else if f.XMLROOT() != nil {
			result = append(result, f.A_expr(0))
		} else if f.XMLSERIALIZE() != nil {
			result = append(result, f.A_expr(0))
		} else if f.JSON_OBJECT() != nil {
			if list := f.Func_arg_list(); list != nil {
				result = append(result, getArgumentsFromFuncArgList(list)...)
			}
			if list := f.Json_name_and_value_list(); list != nil {
				result = append(result, getArgumentsFromJsonNameAndValueList(list)...)
			}
			if list := f.Json_value_expr_list(); list != nil {
				result = append(result, getArgumentsFromJsonValueExprList(list)...)
			}
		} else if f.JSON_ARRAY() != nil {
			if list := f.Json_value_expr_list(); list != nil {
				result = append(result, getArgumentsFromJsonValueExprList(list)...)
			}
			if list := f.Select_no_parens(); list != nil {
				result = append(result, list)
			}
		} else if f.JSON() != nil {
			result = append(result, f.Json_value_expr())
		} else if f.JSON_SCALAR() != nil {
			result = append(result, f.A_expr(0))
		} else if f.JSON_SERIALIZE() != nil {
			result = append(result, f.Json_value_expr())
		} else if f.JSON_QUERY() != nil {
			result = append(result, f.Json_value_expr())
			result = append(result, f.A_expr(0))
		} else if f.JSON_EXISTS() != nil {
			result = append(result, f.Json_value_expr())
			result = append(result, f.A_expr(0))
		} else if f.JSON_VALUE() != nil {
			result = append(result, f.Json_value_expr())
			result = append(result, f.A_expr(0))
		}
	}
	return result
}

func getArgumentsFromFuncArgList(funcArgList pgparser.IFunc_arg_listContext) []antlr.ParseTree {
	var result []antlr.ParseTree
	if funcArgList == nil {
		return result
	}
	for _, argExpr := range funcArgList.AllFunc_arg_expr() {
		result = append(result, argExpr)
	}
	return result
}

func getArgumentsFromJsonNameAndValueList(jsonNameAndValueList pgparser.IJson_name_and_value_listContext) []antlr.ParseTree {
	var result []antlr.ParseTree
	if jsonNameAndValueList == nil {
		return result
	}
	for _, nameAndValue := range jsonNameAndValueList.AllJson_name_and_value() {
		result = append(result, nameAndValue)
	}
	return result
}

func getArgumentsFromJsonValueExprList(jsonValueExprList pgparser.IJson_value_expr_listContext) []antlr.ParseTree {
	var result []antlr.ParseTree
	if jsonValueExprList == nil {
		return result
	}
	for _, valueExpr := range jsonValueExprList.AllJson_value_expr() {
		result = append(result, valueExpr)
	}
	return result
}

func (q *querySpanExtractor) extractTableSourceFromSystemFunctionNew(funcName string, args []antlr.ParseTree, alias pgparser.IFunc_alias_clauseContext) (base.TableSource, error) {
	switch strings.ToLower(funcName) {
	case generateSeries:
		// https://neon.com/postgresql/postgresql-tutorial/postgresql-generate_series
		tableSource := &base.PseudoTable{
			Name: generateSeries,
			Columns: []base.QuerySpanResult{
				{
					Name:          generateSeries,
					SourceColumns: make(base.SourceColumnSet, 0),
				},
			},
		}
		return applyFuncAliasClauseToTableSource(tableSource, alias)
	case generateSubscripts:
		tableSource := &base.PseudoTable{
			Name: generateSubscripts,
			Columns: []base.QuerySpanResult{
				{
					Name:          generateSubscripts,
					SourceColumns: make(base.SourceColumnSet, 0),
				},
			},
		}
		return applyFuncAliasClauseToTableSource(tableSource, alias)
	case unnest:
		tableSource := &base.PseudoTable{
			Name:    unnest,
			Columns: []base.QuerySpanResult{},
		}
		if alias == nil || (alias.Alias_clause() != nil && alias.Alias_clause().Name_list() == nil) {
			for range args {
				tableSource.Columns = append(tableSource.Columns, base.QuerySpanResult{
					Name:          unnest,
					SourceColumns: make(base.SourceColumnSet, 0),
				})
			}
		}
		return applyFuncAliasClauseToTableSource(tableSource, alias)
	case jsonbEach, jsonEach, jsonbEachText, jsonEachText:
		// Should be only called while jsonb_each act as table source.
		// SELECT * FROM json_test, jsonb_each(jb) AS hh(key, value) WHERE id = 1;
		var tableSource base.TableSource = &base.PseudoTable{
			Name: "",
			Columns: []base.QuerySpanResult{
				{
					Name:          "key",
					SourceColumns: make(base.SourceColumnSet, 0),
				},
				{
					Name:          "value",
					SourceColumns: make(base.SourceColumnSet, 0),
				},
			},
		}
		tableSource, err := applyFuncAliasClauseToTableSource(tableSource, alias)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to apply alias clause to %s", funcName)
		}

		if len(args) == 0 {
			return nil, errors.Errorf("unexpected empty args for function %s", funcName)
		}
		fieldArg := args[0]
		set, err := q.extractSourceColumnSetFromNode(fieldArg)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to extract source columns from argument of %s", funcName)
		}
		for i := range tableSource.GetQuerySpanResult() {
			tableSource.GetQuerySpanResult()[i].SourceColumns, _ = base.MergeSourceColumnSet(tableSource.GetQuerySpanResult()[i].SourceColumns, set)
		}
		return tableSource, nil
	case jsonToRecordset, jsonbToRecordset:
		if alias == nil {
			return nil, &parsererror.TypeNotSupportedError{
				Extra: fmt.Sprintf("function %s must explicitly define the structure of the record with an AS clause", funcName),
				Type:  "function",
				Name:  funcName,
			}
		}

		sourceColumn := make(base.SourceColumnSet)
		for _, arg := range args {
			argSources, err := q.extractSourceColumnSetFromNode(arg)
			if err != nil {
				return nil, err
			}
			sourceColumn, _ = base.MergeSourceColumnSet(sourceColumn, argSources)
		}

		tableSource := &base.PseudoTable{
			Name:    "",
			Columns: []base.QuerySpanResult{},
		}

		tableName := ""
		var columnNames []string
		if alias.Colid() != nil {
			tableName = NormalizePostgreSQLColid(alias.Colid())
			if alias.Tablefuncelementlist() != nil {
				for _, name := range alias.Tablefuncelementlist().AllTablefuncelement() {
					if name.Colid() != nil {
						columnNames = append(columnNames, NormalizePostgreSQLColid(name.Colid()))
					}
				}
			}
		} else if a := alias.Alias_clause(); a != nil {
			// Get table alias name
			if a.Colid() != nil {
				tableName = NormalizePostgreSQLColid(a.Colid())
			}

			// Get column aliases if present
			if a.Name_list() != nil {
				for _, name := range a.Name_list().AllName() {
					columnNames = append(columnNames, normalizePostgreSQLName(name))
				}
			}
		}

		for _, colName := range columnNames {
			tableSource.Columns = append(tableSource.Columns, base.QuerySpanResult{
				Name:          colName,
				SourceColumns: sourceColumn,
			})
		}
		tableSource.Name = tableName
		return tableSource, nil
	case jsonPopulateRecord, jsonbPopulateRecord, jsonPopulateRecordset, jsonbPopulateRecordset,
		jsonToRecord, jsonbToRecord:
		return nil, &parsererror.TypeNotSupportedError{
			Extra: fmt.Sprintf("Unsupport function %s", funcName),
			Type:  "function",
			Name:  funcName,
		}
	default:
		// For unknown functions, continue with the generic handling below
		if alias == nil {
			return nil, &parsererror.TypeNotSupportedError{
				Extra: "Use system function result as the table source must have the alias clause to specify table and columns name",
				Type:  "function",
				Name:  funcName,
			}
		}

		tableName := ""
		var columnNames []string
		if alias.Colid() != nil {
			tableName = NormalizePostgreSQLColid(alias.Colid())
			if alias.Tablefuncelementlist() != nil {
				for _, name := range alias.Tablefuncelementlist().AllTablefuncelement() {
					if name.Colid() != nil {
						columnNames = append(columnNames, NormalizePostgreSQLColid(name.Colid()))
					}
				}
			}
		} else if a := alias.Alias_clause(); a != nil {
			// Get table alias name
			if a.Colid() != nil {
				tableName = NormalizePostgreSQLColid(a.Colid())
			}

			// Get column aliases if present
			if a.Name_list() != nil {
				for _, name := range a.Name_list().AllName() {
					columnNames = append(columnNames, normalizePostgreSQLName(name))
				}
			}
		}
		tableSource := &base.PseudoTable{
			Name:    tableName,
			Columns: []base.QuerySpanResult{},
		}
		for _, colName := range columnNames {
			tableSource.Columns = append(tableSource.Columns, base.QuerySpanResult{
				Name:          colName,
				SourceColumns: make(base.SourceColumnSet, 0),
			})
		}
		return tableSource, nil
	}
}
