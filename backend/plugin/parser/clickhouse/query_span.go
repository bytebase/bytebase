package clickhouse

import (
	"context"
	"maps"
	"strings"

	chanalysis "github.com/bytebase/omni/clickhouse/analysis"
	chparser "github.com/bytebase/omni/clickhouse/parser"
	"github.com/pkg/errors"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

// GetQuerySpan extracts table-level source spans and query classification for ClickHouse statements.
func GetQuerySpan(
	_ context.Context,
	_ base.GetQuerySpanContext,
	stmt base.Statement,
	database, _ string,
	ignoreCaseSensitive bool,
) (*base.QuerySpan, error) {
	queryType, err := chanalysis.Classify(stmt.Text)
	if err != nil {
		return nil, convertParseError(stmt.Text, err, stmt.Start)
	}

	stmts, err := chparser.Parse(stmt.Text)
	if err != nil {
		return nil, convertParseError(stmt.Text, err, stmt.Start)
	}
	qt := base.OmniQueryType(queryType)
	if len(stmts) == 0 {
		return &base.QuerySpan{
			Type:             qt,
			SourceColumns:    base.SourceColumnSet{},
			PredicateColumns: base.SourceColumnSet{},
			Results:          []base.QuerySpanResult{},
		}, nil
	}

	c := &spanCollector{
		defaultDatabase:     database,
		ignoreCaseSensitive: ignoreCaseSensitive,
		sources:             make(base.SourceColumnSet),
	}
	c.walkStatement(stmts[0], nil)

	if qt == base.Select || qt == base.SelectInfoSchema {
		if c.hasUserTable && c.hasSystemTable {
			return nil, base.MixUserSystemTablesError
		}
		if c.hasSystemTable && !c.hasUserTable {
			qt = base.SelectInfoSchema
		}
	}

	return &base.QuerySpan{
		Type:             qt,
		SourceColumns:    c.sources,
		PredicateColumns: base.SourceColumnSet{},
		Results:          []base.QuerySpanResult{},
	}, nil
}

func convertParseError(statement string, err error, basePos *storepb.Position) error {
	var parseErr *chparser.ParseError
	if errors.As(err, &parseErr) {
		syntaxErr := base.SyntaxErrorAtOffset(statement, parseErr.Position, parseErr.Message)
		if basePos != nil && syntaxErr.Position != nil {
			if syntaxErr.Position.Line == 1 && basePos.Column > 0 {
				syntaxErr.Position.Column += basePos.Column - 1
			}
			if basePos.Line > 0 {
				syntaxErr.Position.Line += basePos.Line - 1
			}
		}
		return syntaxErr
	}
	return err
}

type spanCollector struct {
	defaultDatabase     string
	ignoreCaseSensitive bool
	sources             base.SourceColumnSet
	hasUserTable        bool
	hasSystemTable      bool
}

func (c *spanCollector) normalizeName(name string) string {
	if c.ignoreCaseSensitive {
		return strings.ToLower(name)
	}
	return name
}

func isClickHouseSystemDatabase(db string) bool {
	return strings.EqualFold(db, "system") || strings.EqualFold(db, "information_schema")
}

func (c *spanCollector) recordTable(db, table string) {
	if table == "" {
		return
	}
	if db == "" {
		db = c.defaultDatabase
	}
	if isClickHouseSystemDatabase(db) {
		c.hasSystemTable = true
	} else {
		c.hasUserTable = true
	}
	c.sources[base.ColumnResource{
		Database: db,
		Table:    table,
	}] = true
}

func (c *spanCollector) walkStatement(stmt chparser.Statement, ctes map[string]bool) {
	switch s := stmt.(type) {
	case *chparser.SelectUnionStmt:
		c.walkSelectUnion(s, ctes)
	case *chparser.SelectStmt:
		c.walkSelect(s, ctes)
	case *chparser.InsertStmt:
		c.recordTable(s.Database, s.Table)
		if s.Select != nil {
			c.walkSelectUnion(s.Select, ctes)
		}
	case *chparser.DeleteStmt:
		db, tbl := splitQualifiedTable(s.Table)
		c.recordTable(db, tbl)
		c.walkExpr(s.Where, ctes)
	case *chparser.UpdateStmt:
		db, tbl := splitQualifiedTable(s.Table)
		c.recordTable(db, tbl)
		c.walkExpr(s.Where, ctes)
	}
}

func splitQualifiedTable(qualified string) (string, string) {
	if db, tbl, ok := strings.Cut(qualified, "."); ok {
		return db, tbl
	}
	return "", qualified
}

func (c *spanCollector) walkSelectUnion(u *chparser.SelectUnionStmt, ctes map[string]bool) {
	if u == nil {
		return
	}
	for _, item := range u.Items {
		c.walkSelect(item, ctes)
	}
}

func (c *spanCollector) walkSelect(s *chparser.SelectStmt, parentCTEs map[string]bool) {
	if s == nil {
		return
	}
	ctes := make(map[string]bool, len(parentCTEs)+len(s.With))
	maps.Copy(ctes, parentCTEs)
	for _, w := range s.With {
		if w == nil {
			continue
		}
		if w.Subquery != nil {
			c.walkSelectUnion(w.Subquery, ctes)
		}
		if w.Expr != nil {
			c.walkExpr(w.Expr, ctes)
		}
		if w.Alias != "" {
			ctes[c.normalizeName(w.Alias)] = true
		}
	}

	if s.Nested != nil {
		c.walkSelectUnion(s.Nested, ctes)
	}
	if s.From != nil {
		c.walkJoin(s.From, ctes)
	}
	if s.ArrayJoin != nil {
		for _, expr := range s.ArrayJoin.Exprs {
			c.walkExpr(expr, ctes)
		}
	}
	for _, col := range s.Columns {
		c.walkExpr(col, ctes)
	}
	c.walkExpr(s.Prewhere, ctes)
	c.walkExpr(s.Where, ctes)
	if s.GroupBy != nil {
		for _, expr := range s.GroupBy.Exprs {
			c.walkExpr(expr, ctes)
		}
	}
	c.walkExpr(s.Having, ctes)
	c.walkExpr(s.Qualify, ctes)
	for _, ob := range s.OrderBy {
		if ob != nil {
			c.walkExpr(ob.Expr, ctes)
		}
	}
}

func (c *spanCollector) walkJoin(j *chparser.JoinExpr, ctes map[string]bool) {
	if j == nil {
		return
	}
	if j.Left != nil {
		c.walkJoin(j.Left, ctes)
	}
	if j.Right != nil {
		c.walkJoin(j.Right, ctes)
	}
	if j.Table != nil {
		t := j.Table
		if t.Subquery != nil {
			c.walkSelectUnion(t.Subquery, ctes)
		} else if t.Func != nil {
			c.walkExpr(t.Func, ctes)
		} else if t.Table != "" {
			if t.Database == "" && ctes[c.normalizeName(t.Table)] {
				return
			}
			c.recordTable(t.Database, t.Table)
		}
	}
	for _, on := range j.On {
		c.walkExpr(on, ctes)
	}
}

func (c *spanCollector) walkExpr(expr chparser.Expr, ctes map[string]bool) {
	if expr == nil {
		return
	}
	switch e := expr.(type) {
	case *chparser.SubqueryExpr:
		c.walkSelectUnion(e.Query, ctes)
	case *chparser.AliasExpr:
		c.walkExpr(e.Expr, ctes)
	case *chparser.ParenExpr:
		c.walkExpr(e.Expr, ctes)
	case *chparser.UnaryExpr:
		c.walkExpr(e.Expr, ctes)
	case *chparser.BinaryExpr:
		c.walkExpr(e.Left, ctes)
		c.walkExpr(e.Right, ctes)
	case *chparser.TernaryExpr:
		c.walkExpr(e.Cond, ctes)
		c.walkExpr(e.Then, ctes)
		c.walkExpr(e.Else, ctes)
	case *chparser.InExpr:
		c.walkExpr(e.Left, ctes)
		c.walkExpr(e.Right, ctes)
	case *chparser.BetweenExpr:
		c.walkExpr(e.Expr, ctes)
		c.walkExpr(e.Low, ctes)
		c.walkExpr(e.High, ctes)
	case *chparser.IsNullExpr:
		c.walkExpr(e.Expr, ctes)
	case *chparser.LikeExpr:
		c.walkExpr(e.Left, ctes)
		c.walkExpr(e.Right, ctes)
	case *chparser.CastExpr:
		c.walkExpr(e.Expr, ctes)
	case *chparser.CaseExpr:
		c.walkExpr(e.Operand, ctes)
		for _, w := range e.Whens {
			c.walkExpr(w.Cond, ctes)
			c.walkExpr(w.Then, ctes)
		}
		c.walkExpr(e.Else, ctes)
	case *chparser.FuncCallExpr:
		for _, p := range e.Params {
			c.walkExpr(p, ctes)
		}
		for _, a := range e.Args {
			c.walkExpr(a, ctes)
		}
	case *chparser.ArrayExpr:
		for _, item := range e.Items {
			c.walkExpr(item, ctes)
		}
	case *chparser.TupleExpr:
		for _, item := range e.Items {
			c.walkExpr(item, ctes)
		}
	case *chparser.TupleAccessExpr:
		c.walkExpr(e.Expr, ctes)
	case *chparser.IndexExpr:
		c.walkExpr(e.Expr, ctes)
		c.walkExpr(e.Index, ctes)
	}
}
