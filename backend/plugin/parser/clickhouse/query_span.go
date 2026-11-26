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
	if err := c.walkStatement(stmts[0], nil); err != nil {
		return nil, err
	}

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

func isClickHouseSystemDatabase(db string, ignoreCaseSensitive bool) bool {
	if ignoreCaseSensitive {
		return strings.EqualFold(db, "system") || strings.EqualFold(db, "information_schema")
	}
	return db == "system" || db == "information_schema" || db == "INFORMATION_SCHEMA"
}

func (c *spanCollector) recordTable(db, table string) {
	if table == "" {
		return
	}
	if db == "" {
		db = c.defaultDatabase
	}
	if isClickHouseSystemDatabase(db, c.ignoreCaseSensitive) {
		c.hasSystemTable = true
	} else {
		c.hasUserTable = true
	}
	c.sources[base.ColumnResource{
		Database: db,
		Table:    table,
	}] = true
}

func (c *spanCollector) walkStatement(stmt chparser.Statement, ctes map[string]bool) error {
	switch s := stmt.(type) {
	case *chparser.SelectUnionStmt:
		return c.walkSelectUnion(s, ctes)
	case *chparser.SelectStmt:
		return c.walkSelect(s, ctes)
	case *chparser.InsertStmt:
		if s.Func != nil {
			return errors.Errorf("table function %q is not supported in ClickHouse query span", s.Func.Name)
		}
		c.recordTable(s.Database, s.Table)
		if s.Select != nil {
			return c.walkSelectUnion(s.Select, ctes)
		}
	case *chparser.DeleteStmt:
		db, tbl := splitQualifiedTable(s.Table)
		c.recordTable(db, tbl)
		return c.walkExpr(s.Where, ctes)
	case *chparser.UpdateStmt:
		db, tbl := splitQualifiedTable(s.Table)
		c.recordTable(db, tbl)
		return c.walkExpr(s.Where, ctes)
	}
	return nil
}

func splitQualifiedTable(qualified string) (string, string) {
	if db, tbl, ok := strings.Cut(qualified, "."); ok {
		return db, tbl
	}
	return "", qualified
}

func (c *spanCollector) walkSelectUnion(u *chparser.SelectUnionStmt, ctes map[string]bool) error {
	if u == nil {
		return nil
	}
	for _, item := range u.Items {
		if err := c.walkSelect(item, ctes); err != nil {
			return err
		}
	}
	return nil
}

func (c *spanCollector) walkSelect(s *chparser.SelectStmt, parentCTEs map[string]bool) error {
	if s == nil {
		return nil
	}
	ctes := make(map[string]bool, len(parentCTEs)+len(s.With))
	maps.Copy(ctes, parentCTEs)
	for _, w := range s.With {
		if w == nil {
			continue
		}
		if err := c.walkSelectUnion(w.Subquery, ctes); err != nil {
			return err
		}
		if err := c.walkExpr(w.Expr, ctes); err != nil {
			return err
		}
		if w.Alias != "" {
			ctes[c.normalizeName(w.Alias)] = true
		}
	}

	if err := c.walkSelectUnion(s.Nested, ctes); err != nil {
		return err
	}
	if err := c.walkJoin(s.From, ctes); err != nil {
		return err
	}
	if s.ArrayJoin != nil {
		for _, expr := range s.ArrayJoin.Exprs {
			if err := c.walkExpr(expr, ctes); err != nil {
				return err
			}
		}
	}
	for _, col := range s.Columns {
		if err := c.walkExpr(col, ctes); err != nil {
			return err
		}
	}
	if err := c.walkExpr(s.Prewhere, ctes); err != nil {
		return err
	}
	if err := c.walkExpr(s.Where, ctes); err != nil {
		return err
	}
	if s.GroupBy != nil {
		for _, expr := range s.GroupBy.Exprs {
			if err := c.walkExpr(expr, ctes); err != nil {
				return err
			}
		}
	}
	if err := c.walkExpr(s.Having, ctes); err != nil {
		return err
	}
	if err := c.walkExpr(s.Qualify, ctes); err != nil {
		return err
	}
	for _, ob := range s.OrderBy {
		if ob != nil {
			if err := c.walkExpr(ob.Expr, ctes); err != nil {
				return err
			}
		}
	}
	return nil
}

func (c *spanCollector) walkJoin(j *chparser.JoinExpr, ctes map[string]bool) error {
	if j == nil {
		return nil
	}
	if err := c.walkJoin(j.Left, ctes); err != nil {
		return err
	}
	if err := c.walkJoin(j.Right, ctes); err != nil {
		return err
	}
	if j.Table != nil {
		t := j.Table
		if t.Subquery != nil {
			if err := c.walkSelectUnion(t.Subquery, ctes); err != nil {
				return err
			}
		} else if t.Func != nil {
			// Table functions such as remote() or cluster() can read arbitrary external
			// or cross-database tables; fail closed rather than returning an empty span.
			return errors.Errorf("table function %q is not supported in ClickHouse query span", t.Func.Name)
		} else if t.Table != "" {
			if t.Database == "" && ctes[c.normalizeName(t.Table)] {
				return nil
			}
			c.recordTable(t.Database, t.Table)
		}
	}
	for _, on := range j.On {
		if err := c.walkExpr(on, ctes); err != nil {
			return err
		}
	}
	return nil
}

func (c *spanCollector) walkExpr(expr chparser.Expr, ctes map[string]bool) error {
	if expr == nil {
		return nil
	}
	switch e := expr.(type) {
	case *chparser.SubqueryExpr:
		return c.walkSelectUnion(e.Query, ctes)
	case *chparser.AliasExpr:
		return c.walkExpr(e.Expr, ctes)
	case *chparser.ParenExpr:
		return c.walkExpr(e.Expr, ctes)
	case *chparser.UnaryExpr:
		return c.walkExpr(e.Expr, ctes)
	case *chparser.BinaryExpr:
		if err := c.walkExpr(e.Left, ctes); err != nil {
			return err
		}
		return c.walkExpr(e.Right, ctes)
	case *chparser.TernaryExpr:
		if err := c.walkExpr(e.Cond, ctes); err != nil {
			return err
		}
		if err := c.walkExpr(e.Then, ctes); err != nil {
			return err
		}
		return c.walkExpr(e.Else, ctes)
	case *chparser.InExpr:
		if err := c.walkExpr(e.Left, ctes); err != nil {
			return err
		}
		return c.walkExpr(e.Right, ctes)
	case *chparser.BetweenExpr:
		if err := c.walkExpr(e.Expr, ctes); err != nil {
			return err
		}
		if err := c.walkExpr(e.Low, ctes); err != nil {
			return err
		}
		return c.walkExpr(e.High, ctes)
	case *chparser.IsNullExpr:
		return c.walkExpr(e.Expr, ctes)
	case *chparser.LikeExpr:
		if err := c.walkExpr(e.Left, ctes); err != nil {
			return err
		}
		return c.walkExpr(e.Right, ctes)
	case *chparser.CastExpr:
		return c.walkExpr(e.Expr, ctes)
	case *chparser.CaseExpr:
		if err := c.walkExpr(e.Operand, ctes); err != nil {
			return err
		}
		for _, w := range e.Whens {
			if err := c.walkExpr(w.Cond, ctes); err != nil {
				return err
			}
			if err := c.walkExpr(w.Then, ctes); err != nil {
				return err
			}
		}
		return c.walkExpr(e.Else, ctes)
	case *chparser.FuncCallExpr:
		for _, p := range e.Params {
			if err := c.walkExpr(p, ctes); err != nil {
				return err
			}
		}
		for _, a := range e.Args {
			if err := c.walkExpr(a, ctes); err != nil {
				return err
			}
		}
	case *chparser.ArrayExpr:
		for _, item := range e.Items {
			if err := c.walkExpr(item, ctes); err != nil {
				return err
			}
		}
	case *chparser.TupleExpr:
		for _, item := range e.Items {
			if err := c.walkExpr(item, ctes); err != nil {
				return err
			}
		}
	case *chparser.TupleAccessExpr:
		return c.walkExpr(e.Expr, ctes)
	case *chparser.IndexExpr:
		if err := c.walkExpr(e.Expr, ctes); err != nil {
			return err
		}
		return c.walkExpr(e.Index, ctes)
	}
	return nil
}
