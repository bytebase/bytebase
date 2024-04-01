package mysql

import (
	"github.com/antlr4-go/antlr/v4"
	mysql "github.com/bytebase/mysql-parser"

	mysqlparser "github.com/bytebase/bytebase/backend/plugin/parser/mysql"
)

type stmtType int

const (
	currentStmtTypeUnknown stmtType = iota
	currentStmtTypeCreateFunction
	currentStmtTypeCreateProcedure
	currentStmtTypeSet
)

type stmtInfo struct {
	tp                   stmtType
	normalizedObjectName string
}

// stmtInfoGetter is a listener to get the statement type and normalized object name.
// Currently, it only support CREATE FUNCTION, CREATE PROCEDURE and SET statement,
// and it will regard other statements as unknown type.
type stmtInfoGetter struct {
	*mysql.BaseMySQLParserListener

	tp                   stmtType
	normalizedObjectName string

	result []*stmtInfo
}

func (g *stmtInfoGetter) ExitQuery(*mysql.QueryContext) {
	g.result = append(g.result, &stmtInfo{
		tp:                   g.tp,
		normalizedObjectName: g.normalizedObjectName,
	})
	g.tp = currentStmtTypeUnknown
	g.normalizedObjectName = ""
}

func (g *stmtInfoGetter) EnterSetStatement(ctx *mysql.SetStatementContext) {
	p := ctx.GetParent()
	if _, ok := p.(*mysql.SimpleStatementContext); !ok {
		return
	}
	pp := p.GetParent()
	if _, ok := pp.(*mysql.QueryContext); !ok {
		return
	}
	g.tp = currentStmtTypeSet
}

func (g *stmtInfoGetter) EnterCreateProcedure(ctx *mysql.CreateProcedureContext) {
	p := ctx.GetParent()
	if _, ok := p.(*mysql.CreateStatementContext); !ok {
		return
	}
	pp := p.GetParent()
	if _, ok := pp.(*mysql.SimpleStatementContext); !ok {
		return
	}
	ppp := pp.GetParent()
	if _, ok := ppp.(*mysql.QueryContext); !ok {
		return
	}
	g.tp = currentStmtTypeCreateProcedure
	_, g.normalizedObjectName = mysqlparser.NormalizeMySQLProcedureName(ctx.ProcedureName())
}

func (g *stmtInfoGetter) EnterCreateFunction(ctx *mysql.CreateFunctionContext) {
	p := ctx.GetParent()
	if _, ok := p.(*mysql.CreateStatementContext); !ok {
		return
	}
	pp := p.GetParent()
	if _, ok := pp.(*mysql.SimpleStatementContext); !ok {
		return
	}
	ppp := pp.GetParent()
	if _, ok := ppp.(*mysql.QueryContext); !ok {
		return
	}
	g.tp = currentStmtTypeCreateFunction
	_, g.normalizedObjectName = mysqlparser.NormalizeMySQLFunctionName(ctx.FunctionName())
}

type groupType int

const (
	groupTypeUnknown         groupType = iota
	groupTypeCreateFunction            // CREATE FUNCTION
	groupTypeCreateProcedure           // CREATE PROCEDURE
)

type stmtGroup struct {
	tp         groupType
	beginIdx   int
	endIdx     int
	objectName string
}

func groupStatement(parseResults []*mysqlparser.ParseResult) ([]*stmtGroup, error) {
	var groups []*stmtGroup
	infoGetter := &stmtInfoGetter{}
	for _, parseResult := range parseResults {
		antlr.ParseTreeWalkerDefault.Walk(infoGetter, parseResult.Tree)
	}

	infos := infoGetter.result
	for i := 0; i < len(infos); i++ {
		nextIdx := i
		if infos[i].tp == currentStmtTypeSet {
			// Find the next SET/CREATE FUNCTION/CREATE PROCEDURE statement.
			j := i + 1
			for ; j < len(infos); j++ {
				if infos[j].tp == currentStmtTypeSet {
					continue
				}
				if infos[j].tp == currentStmtTypeCreateFunction {
					nextIdx = j + 1
					groups = append(groups, &stmtGroup{
						tp:         groupTypeCreateFunction,
						beginIdx:   i,
						endIdx:     j,
						objectName: infos[j].normalizedObjectName,
					})
					break
				}
				if infos[j].tp == currentStmtTypeCreateProcedure {
					nextIdx = j + 1
					groups = append(groups, &stmtGroup{
						tp:         groupTypeCreateProcedure,
						beginIdx:   i,
						endIdx:     j,
						objectName: infos[j].normalizedObjectName,
					})
					break
				}
				break
			}
		}
		if nextIdx > i {
			i = nextIdx - 1
		}
		if infos[i].tp == currentStmtTypeCreateFunction {
			groups = append(groups, &stmtGroup{
				tp:         groupTypeCreateFunction,
				beginIdx:   i,
				endIdx:     i,
				objectName: infos[i].normalizedObjectName,
			})
			continue
		}
		if infos[i].tp == currentStmtTypeCreateProcedure {
			groups = append(groups, &stmtGroup{
				tp:         groupTypeCreateProcedure,
				beginIdx:   i,
				endIdx:     i,
				objectName: infos[i].normalizedObjectName,
			})
			continue
		}
		groups = append(groups, &stmtGroup{
			tp:       groupTypeUnknown,
			beginIdx: i,
			endIdx:   i,
		})
	}

	return groups, nil
}
