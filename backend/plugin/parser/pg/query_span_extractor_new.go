package pg

import (
	"context"
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
	// TODO: Implement VALUES clause handling when needed

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
	if primary.Select_with_parens() != nil {
		selectWithParens := primary.Select_with_parens()
		if selectWithParens.Select_no_parens() != nil {
			return q.extractTableSourceFromSelectNoParens(selectWithParens.Select_no_parens())
		}
		if selectWithParens.Select_with_parens() != nil {
			// Recursive parentheses
			return q.extractTableSourceFromSelectWithParens(selectWithParens.Select_with_parens())
		}
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

// mergeSimpleSelectIntersects combines multiple simple_select_intersect nodes with UNION operations.
// This is used to process the base or recursive parts of a CTE that may contain multiple queries.
// Deprecated: Use mergeSimpleSelectIntersectsWithOperators instead.
func (q *querySpanExtractor) mergeSimpleSelectIntersects(parts []pgparser.ISimple_select_intersectContext, isUnionAll bool) (base.TableSource, error) {
	if len(parts) == 0 {
		return nil, errors.New("no parts to merge")
	}

	// Start with the first part
	result, err := q.extractTableSourceFromSimpleSelectIntersect(parts[0])
	if err != nil {
		return nil, errors.Wrap(err, "failed to extract first simple_select_intersect")
	}

	// Merge remaining parts with UNION semantics
	for i := 1; i < len(parts); i++ {
		nextResult, err := q.extractTableSourceFromSimpleSelectIntersect(parts[i])
		if err != nil {
			return nil, errors.Wrapf(err, "failed to extract simple_select_intersect %d", i)
		}

		// Merge with UNION/UNION ALL semantics
		leftQuerySpanResult := result.GetQuerySpanResult()
		rightQuerySpanResult := nextResult.GetQuerySpanResult()

		if len(leftQuerySpanResult) != len(rightQuerySpanResult) {
			return nil, errors.Errorf("UNION requires same number of columns: %d vs %d at position %d",
				len(leftQuerySpanResult), len(rightQuerySpanResult), i)
		}

		var mergedColumns []base.QuerySpanResult
		for j, leftCol := range leftQuerySpanResult {
			rightCol := rightQuerySpanResult[j]
			// For UNION, we combine source columns from both sides
			mergedSourceColumns, _ := base.MergeSourceColumnSet(leftCol.SourceColumns, rightCol.SourceColumns)
			mergedColumns = append(mergedColumns, base.QuerySpanResult{
				Name:          leftCol.Name, // Use the name from the first query
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

	// Handle simple table reference (e.g., schema.table)
	if tableRef.Relation_expr() != nil {
		return q.extractTableSourceFromRelationExpr(tableRef.Relation_expr(), tableRef.Opt_alias_clause())
	}

	// Handle subquery (SELECT in parentheses)
	if tableRef.Select_with_parens() != nil {
		subquerySource, err := q.extractTableSourceFromSelectWithParens(tableRef.Select_with_parens())
		if err != nil {
			return nil, errors.Wrapf(err, "failed to extract subquery")
		}

		// Apply alias if present
		if tableRef.Opt_alias_clause() != nil {
			return q.applyAliasToTableSource(subquerySource, tableRef.Opt_alias_clause())
		}
		return subquerySource, nil
	}

	// Handle JOIN expressions
	if tableRef.Joined_table(0) != nil {
		return q.extractTableSourceFromJoinedTable(tableRef.Joined_table(0))
	}

	// Handle function tables (e.g., generate_series, unnest)
	if tableRef.Func_table() != nil {
		return q.extractTableSourceFromFuncTable(tableRef.Func_table())
	}

	// If we have nested table_ref (for complex expressions)
	allTableRefs := tableRef.AllTable_ref()
	if len(allTableRefs) > 0 {
		// Process the first table_ref recursively
		return q.extractTableSourceFromTableRef(allTableRefs[0])
	}

	return nil, errors.New("unsupported table_ref type")
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

// extractTableSourceFromJoinedTable extracts table source from a joined table
func (q *querySpanExtractor) extractTableSourceFromJoinedTable(joinedTable pgparser.IJoined_tableContext) (base.TableSource, error) {
	if joinedTable == nil {
		return nil, errors.New("nil joined_table")
	}

	// TODO: Implement proper JOIN handling
	// The actual implementation would need to:
	// 1. Extract left and right table references
	// 2. Handle different join types (INNER, LEFT, RIGHT, FULL, CROSS)
	// 3. Process JOIN conditions (ON clause, USING clause, NATURAL)
	// 4. Merge columns appropriately based on join type

	// For now, return a placeholder
	return &base.PseudoTable{
		Name:    "",
		Columns: []base.QuerySpanResult{},
	}, nil
}

// extractTableSourceFromFuncTable extracts table source from a function table
func (q *querySpanExtractor) extractTableSourceFromFuncTable(funcTable pgparser.IFunc_tableContext) (base.TableSource, error) {
	// Handle function tables like unnest, generate_series, etc.
	// For now, return a pseudo table that will be populated with alias columns if present
	result := &base.PseudoTable{
		Name:    "",
		Columns: []base.QuerySpanResult{},
	}

	if funcTable == nil {
		return result, nil
	}

	// The actual column count and names will be determined by the alias
	// For now, just return an empty pseudo table
	// The applyAliasToTableSource function will handle the column names

	// Check if this is a ROWS FROM expression
	if funcTable.ROWS() != nil && funcTable.Rowsfrom_list() != nil {
		// Handle ROWS FROM (...)
		// Each function in ROWS FROM produces columns
		for _, item := range funcTable.Rowsfrom_list().AllRowsfrom_item() {
			if item.Func_expr_windowless() != nil {
				// Without analyzing the function, we can't determine the columns
				// The alias will provide the actual column names
			}
		}
	}

	// For function tables, the columns are typically determined by:
	// 1. The function's return type (which we don't have access to without metadata)
	// 2. The alias provided (which will be applied later)
	// So we return an empty pseudo table that will be populated by the alias

	return result, nil
}

// applyAliasToTableSource applies an alias to a table source
func (q *querySpanExtractor) applyAliasToTableSource(source base.TableSource, aliasClause pgparser.IOpt_alias_clauseContext) (base.TableSource, error) {
	if aliasClause == nil || aliasClause.Table_alias_clause() == nil {
		return source, nil
	}

	// Use the existing normalization function
	tableAlias := aliasClause.Table_alias_clause()
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

	// Create a pseudo table with the alias
	pseudoTable := &base.PseudoTable{
		Name:    aliasName,
		Columns: source.GetQuerySpanResult(),
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
	} else if len(pseudoTable.Columns) == 0 && aliasName != "" {
		// For function tables with just a table alias and no columns, create a single column
		pseudoTable.Columns = append(pseudoTable.Columns, base.QuerySpanResult{
			Name:          aliasName,
			SourceColumns: base.SourceColumnSet{},
		})
	}

	return pseudoTable, nil
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
			columnName = "?column?"
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
		return "?column?", nil
	}

	// Try to find a column reference in the expression
	columnName := q.findColumnNameInExpression(expr)
	if columnName != "" {
		return columnName, nil
	}

	// Default to unknown column name for other expression types
	return "?column?", nil
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
	case pgparser.ICase_exprContext:
		// Handle CASE expression - process all branches
		if ctx.Case_arg() != nil && ctx.Case_arg().A_expr() != nil {
			argSources, err := q.extractSourceColumnSetFromNode(ctx.Case_arg().A_expr())
			if err != nil {
				return nil, err
			}
			result, _ = base.MergeSourceColumnSet(result, argSources)
		}

		// Process WHEN clauses
		if ctx.When_clause_list() != nil {
			whenSources, err := q.extractSourceColumnSetFromNode(ctx.When_clause_list())
			if err != nil {
				return nil, err
			}
			result, _ = base.MergeSourceColumnSet(result, whenSources)
		}

		// Process ELSE clause
		if ctx.Case_default() != nil && ctx.Case_default().A_expr() != nil {
			elseSources, err := q.extractSourceColumnSetFromNode(ctx.Case_default().A_expr())
			if err != nil {
				return nil, err
			}
			result, _ = base.MergeSourceColumnSet(result, elseSources)
		}
		return result, nil

	case pgparser.IWhen_clause_listContext:
		// Process all WHEN clauses
		for _, whenClause := range ctx.AllWhen_clause() {
			// Each when clause has two a_expr: condition and result
			allExprs := whenClause.AllA_expr()
			for _, expr := range allExprs {
				exprSources, err := q.extractSourceColumnSetFromNode(expr)
				if err != nil {
					return nil, err
				}
				result, _ = base.MergeSourceColumnSet(result, exprSources)
			}
		}
		return result, nil

	case pgparser.IA_exprContext:
		// For a_expr nodes, recursively process all children
		// a_expr can contain various expression types, so we use generic traversal
		result := make(base.SourceColumnSet)
		for i := 0; i < ctx.GetChildCount(); i++ {
			if child, ok := ctx.GetChild(i).(antlr.ParseTree); ok {
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
		return result, nil

	case pgparser.IC_exprContext:
		// c_expr can contain various expression types
		// Use generic traversal to find column references and other expressions
		result := make(base.SourceColumnSet)
		for i := 0; i < ctx.GetChildCount(); i++ {
			if child, ok := ctx.GetChild(i).(antlr.ParseTree); ok {
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
		return result, nil

	case pgparser.IArray_exprContext:
		// Process array elements
		if ctx.Expr_list() != nil {
			return q.extractSourceColumnSetFromNode(ctx.Expr_list())
		}
		if ctx.Array_expr_list() != nil {
			return q.extractSourceColumnSetFromNode(ctx.Array_expr_list())
		}

	case pgparser.IRowContext:
		// Process row elements
		if ctx.Expr_list() != nil {
			return q.extractSourceColumnSetFromNode(ctx.Expr_list())
		}

	case pgparser.IExpr_listContext:
		// Process expression list
		for _, expr := range ctx.AllA_expr() {
			exprSources, err := q.extractSourceColumnSetFromNode(expr)
			if err != nil {
				return nil, err
			}
			result, _ = base.MergeSourceColumnSet(result, exprSources)
		}
		return result, nil

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

// getFieldColumnSourceNew looks up the source columns for a specific field.
func (q *querySpanExtractor) getFieldColumnSourceNew(schemaName, tableName, columnName string) (base.SourceColumnSet, bool) {
	// Helper function to find column in a table source
	findInTableSource := func(tableSource base.TableSource) (base.SourceColumnSet, bool) {
		if tableSource == nil {
			return nil, false
		}
		if schemaName != "" && schemaName != tableSource.GetSchemaName() {
			return nil, false
		}
		if tableName != "" && tableName != tableSource.GetTableName() {
			return nil, false
		}

		querySpanResult := tableSource.GetQuerySpanResult()
		for _, field := range querySpanResult {
			if field.Name == columnName {
				return field.SourceColumns, true
			}
		}
		return nil, false
	}

	// Search in FROM clause tables
	for _, tableSource := range q.tableSourcesFrom {
		if sources, ok := findInTableSource(tableSource); ok {
			return sources, true
		}
	}

	// Search in CTEs
	if schemaName == "" && tableName == "" {
		for i := len(q.ctes) - 1; i >= 0; i-- {
			for _, column := range q.ctes[i].Columns {
				if column.Name == columnName {
					return column.SourceColumns, true
				}
			}
		}
	} else if schemaName == "" {
		for i := len(q.ctes) - 1; i >= 0; i-- {
			if q.ctes[i].Name == tableName {
				for _, column := range q.ctes[i].Columns {
					if column.Name == columnName {
						return column.SourceColumns, true
					}
				}
				break
			}
		}
	}

	// Search in outer tables (for correlated subqueries)
	for _, tableSource := range q.outerTableSources {
		if sources, ok := findInTableSource(tableSource); ok {
			return sources, true
		}
	}

	return nil, false
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
		subexpr := funcExpr.Func_expr_common_subexpr()
		// Extract from expression list
		if subexpr.Expr_list() != nil {
			for _, expr := range subexpr.Expr_list().AllA_expr() {
				exprSources, err := q.extractSourceColumnSetFromAExpr(expr)
				if err != nil {
					return nil, err
				}
				result, _ = base.MergeSourceColumnSet(result, exprSources)
			}
		}
		// Handle XML functions that may have additional expressions
		// TODO: Add more specific handling for XML functions if needed
	}

	// TODO: Handle window functions if needed

	return result, nil
}

// extractSourceColumnSetFromCaseExpr extracts source columns from a CASE expression
func (q *querySpanExtractor) extractSourceColumnSetFromCaseExpr(caseExpr pgparser.ICase_exprContext) (base.SourceColumnSet, error) {
	result := make(base.SourceColumnSet)

	// Handle the CASE expression's test expression (CASE expr WHEN ...)
	if caseExpr.Case_arg() != nil && caseExpr.Case_arg().A_expr() != nil {
		argSources, err := q.extractSourceColumnSetFromAExpr(caseExpr.Case_arg().A_expr())
		if err != nil {
			return nil, err
		}
		result, _ = base.MergeSourceColumnSet(result, argSources)
	}

	// Handle WHEN clauses
	if caseExpr.When_clause_list() != nil {
		// TODO: Extract source columns from when clauses
		// Need to understand the structure better
	}

	// Handle ELSE clause
	if caseExpr.Case_default() != nil && caseExpr.Case_default().A_expr() != nil {
		elseSources, err := q.extractSourceColumnSetFromAExpr(caseExpr.Case_default().A_expr())
		if err != nil {
			return nil, err
		}
		result, _ = base.MergeSourceColumnSet(result, elseSources)
	}

	return result, nil
}

// extractSourceColumnSetFromSubquery extracts source columns from a subquery
func (q *querySpanExtractor) extractSourceColumnSetFromSubquery(subquery pgparser.ISelect_with_parensContext) (base.SourceColumnSet, error) {
	// Create a new extractor for the subquery with current FROM tables as outer context
	subqueryExtractor := &querySpanExtractor{
		defaultDatabase:   q.defaultDatabase,
		searchPath:        q.searchPath,
		ctes:              q.ctes,
		outerTableSources: append(q.outerTableSources, q.tableSourcesFrom...),
		tableSourcesFrom:  []base.TableSource{},
	}

	// Extract table source from the subquery
	var tableSource base.TableSource
	var err error

	if subquery.Select_no_parens() != nil {
		tableSource, err = subqueryExtractor.extractTableSourceFromSelectNoParensNew(subquery.Select_no_parens())
	} else if subquery.Select_with_parens() != nil {
		tableSource, err = subqueryExtractor.extractTableSourceFromSelectWithParensNew(subquery.Select_with_parens())
	}

	if err != nil {
		return nil, errors.Wrapf(err, "failed to extract table source from subquery")
	}

	// Collect all source columns from the subquery result
	result := make(base.SourceColumnSet)
	for _, column := range tableSource.GetQuerySpanResult() {
		result, _ = base.MergeSourceColumnSet(result, column.SourceColumns)
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

// extractTableSourceFromSelectWithParensNew handles SELECT with parentheses.
func (q *querySpanExtractor) extractTableSourceFromSelectWithParensNew(ctx pgparser.ISelect_with_parensContext) (base.TableSource, error) {
	if ctx == nil {
		return nil, errors.New("select_with_parens context is nil")
	}

	// Handle direct select_no_parens
	if ctx.Select_no_parens() != nil {
		return q.extractTableSourceFromSelectNoParensNew(ctx.Select_no_parens())
	}

	// Handle nested select_with_parens
	if ctx.Select_with_parens() != nil {
		return q.extractTableSourceFromSelectWithParensNew(ctx.Select_with_parens())
	}

	return nil, errors.New("no select clause found in select_with_parens")
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

// extractTableSourceFromRecursiveCTENew handles recursive CTEs
func (q *querySpanExtractor) extractTableSourceFromRecursiveCTENew(cte pgparser.ICommon_table_exprContext) (base.TableSource, error) {
	// For recursive CTEs, we follow the simplified strategy:
	// The last UNION/EXCEPT part is considered recursive
	// TODO: Implement proper recursive CTE handling
	return q.extractTableSourceFromNonRecursiveCTENew(cte)
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
