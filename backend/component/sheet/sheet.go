package sheet

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	lru "github.com/hashicorp/golang-lru/v2/expirable"
	"github.com/zeebo/xxh3"

	tidbparser "github.com/pingcap/tidb/pkg/parser"
	tidbast "github.com/pingcap/tidb/pkg/parser/ast"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/plugin/db/mssql"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	mysqlparser "github.com/bytebase/bytebase/backend/plugin/parser/mysql"
	partiqlparser "github.com/bytebase/bytebase/backend/plugin/parser/partiql"
	plsqlparser "github.com/bytebase/bytebase/backend/plugin/parser/plsql"
	snowsqlparser "github.com/bytebase/bytebase/backend/plugin/parser/snowflake"
	"github.com/bytebase/bytebase/backend/plugin/parser/sql/ast"
	pgrawparser "github.com/bytebase/bytebase/backend/plugin/parser/sql/engine/pg"
	tidbbbparser "github.com/bytebase/bytebase/backend/plugin/parser/tidb"
	tsqlparser "github.com/bytebase/bytebase/backend/plugin/parser/tsql"
	tsqlbatch "github.com/bytebase/bytebase/backend/plugin/parser/tsql/batch"
	"github.com/bytebase/bytebase/backend/store"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

const (
	// SyntaxErrorTitle is the error title for syntax error.
	SyntaxErrorTitle         string = "Syntax error"
	StatementSyntaxErrorCode int32  = 201
	InternalErrorCode        int32  = 1
)

// Manager is the coordinator for all sheets and SQL statements.
type Manager struct {
	sync.Mutex

	store    *store.Store
	astCache *lru.LRU[uint64, *Result]
}

// NewManager creates a new sheet manager.
func NewManager(store *store.Store) *Manager {
	return &Manager{
		store:    store,
		astCache: lru.NewLRU[uint64, *Result](8, nil, 3*time.Minute),
	}
}

func (sm *Manager) CreateSheet(ctx context.Context, sheet *store.SheetMessage) (*store.SheetMessage, error) {
	if sheet.Payload == nil {
		sheet.Payload = &storepb.SheetPayload{}
	}
	sheet.Payload.Commands = getSheetCommands(sheet.Payload.Engine, sheet.Statement)

	return sm.store.CreateSheet(ctx, sheet)
}

func getSheetCommands(engine storepb.Engine, statement string) []*storepb.SheetCommand {
	switch engine {
	case
		storepb.Engine_MYSQL,
		storepb.Engine_TIDB:
		if len(statement) > common.MaxSheetCheckSize {
			return nil
		}
	case storepb.Engine_POSTGRES:
	case storepb.Engine_ORACLE:
	case storepb.Engine_MSSQL:
	case storepb.Engine_DYNAMODB:
	default:
		return nil
	}

	switch engine {
	case
		storepb.Engine_TIDB,
		storepb.Engine_ORACLE:
		return getSheetCommandsFromByteOffset(engine, statement)
	case storepb.Engine_MSSQL:
		return getSheetCommandsForMSSQL(statement)
	default:
		return getSheetCommandsGeneral(engine, statement)
	}
}

func getSheetCommandsGeneral(engine storepb.Engine, statement string) []*storepb.SheetCommand {
	singleSQLs, err := base.SplitMultiSQL(engine, statement)
	if err != nil {
		if !strings.Contains(err.Error(), "not supported") {
			slog.Warn("failed to split multi sql", "engine", engine.String(), "statement", statement)
		}
		return nil
	}
	if len(singleSQLs) > common.MaximumCommands {
		return nil
	}

	var sheetCommands []*storepb.SheetCommand
	p := 0
	for _, s := range singleSQLs {
		np := p + len(s.Text)
		sheetCommands = append(sheetCommands, &storepb.SheetCommand{
			Start: int32(p),
			End:   int32(np),
		})
		p = np
	}
	return sheetCommands
}

func getSheetCommandsFromByteOffset(engine storepb.Engine, statement string) []*storepb.SheetCommand {
	singleSQLs, err := base.SplitMultiSQL(engine, statement)
	if err != nil {
		if !strings.Contains(err.Error(), "not supported") {
			slog.Warn("failed to get sheet command from byte offset", "engine", engine.String(), "statement", statement)
		}
		return nil
	}
	if len(singleSQLs) > common.MaximumCommands {
		return nil
	}

	var sheetCommands []*storepb.SheetCommand
	for _, s := range singleSQLs {
		sheetCommands = append(sheetCommands, &storepb.SheetCommand{
			Start: int32(s.ByteOffsetStart),
			End:   int32(s.ByteOffsetEnd),
		})
	}
	return sheetCommands
}

func getSheetCommandsForMSSQL(statement string) []*storepb.SheetCommand {
	var sheetCommands []*storepb.SheetCommand
	p := 0

	batch := mssql.NewBatch(statement)
	for {
		command, err := batch.Next()
		if err == io.EOF {
			np := p + len(batch.String())
			sheetCommands = append(sheetCommands, &storepb.SheetCommand{
				Start: int32(p),
				End:   int32(np),
			})
			break
		}
		if err != nil {
			slog.Warn("failed to get sheet commands for mssql", "statement", statement)
			return nil
		}
		if command == nil {
			continue
		}
		switch command.(type) {
		case *tsqlbatch.GoCommand:
			np := p + len(batch.String())
			sheetCommands = append(sheetCommands, &storepb.SheetCommand{
				Start: int32(p),
				End:   int32(np),
			})
			p = np
		default:
		}
		if len(sheetCommands) > common.MaximumCommands {
			return nil
		}
	}
	return sheetCommands
}

type Result struct {
	sync.Mutex
	ast     any
	advices []*storepb.Advice
}

func (sm *Manager) GetAST(dbType storepb.Engine, statement string) (any, []*storepb.Advice) {
	var result *Result
	hashKey := xxh3.HashString(statement)
	sm.Lock()
	if v, ok := sm.astCache.Get(hashKey); ok {
		result = v
	} else {
		result = &Result{}
		sm.astCache.Add(hashKey, result)
	}
	sm.Unlock()

	result.Lock()
	defer result.Unlock()
	if result.ast != nil || result.advices != nil {
		return result.ast, result.advices
	}
	result.ast, result.advices = syntaxCheck(dbType, statement)
	return result.ast, result.advices
}

func syntaxCheck(dbType storepb.Engine, statement string) (any, []*storepb.Advice) {
	switch dbType {
	case storepb.Engine_TIDB:
		return tidbSyntaxCheck(statement)
	case storepb.Engine_MYSQL, storepb.Engine_MARIADB, storepb.Engine_OCEANBASE:
		return mysqlSyntaxCheck(statement)
	case storepb.Engine_POSTGRES:
		return postgresSyntaxCheck(statement)
	case storepb.Engine_ORACLE, storepb.Engine_OCEANBASE_ORACLE:
		return oracleSyntaxCheck(statement)
	case storepb.Engine_SNOWFLAKE:
		return snowflakeSyntaxCheck(statement)
	case storepb.Engine_MSSQL:
		return mssqlSyntaxCheck(statement)
	case storepb.Engine_DYNAMODB:
		return partiqlSyntaxCheck(statement)
	}
	return nil, []*storepb.Advice{
		{
			Status:  storepb.Advice_ERROR,
			Code:    InternalErrorCode,
			Title:   "Unsupported database type",
			Content: fmt.Sprintf("Unsupported database type %s", dbType),
			StartPosition: &storepb.Position{
				Line: 1,
			},
		},
	}
}

func partiqlSyntaxCheck(statement string) (any, []*storepb.Advice) {
	result, err := partiqlparser.ParsePartiQL(statement)
	if err != nil {
		if syntaxErr, ok := err.(*base.SyntaxError); ok {
			return nil, []*storepb.Advice{
				{
					Status:  storepb.Advice_WARNING,
					Code:    StatementSyntaxErrorCode,
					Title:   SyntaxErrorTitle,
					Content: syntaxErr.Message,
					StartPosition: &storepb.Position{
						Line:   int32(syntaxErr.Line),
						Column: int32(syntaxErr.Column),
					},
				},
			}
		}
		return nil, []*storepb.Advice{
			{
				Status:  storepb.Advice_WARNING,
				Code:    InternalErrorCode,
				Title:   "Parse error",
				Content: err.Error(),
				StartPosition: &storepb.Position{
					Line: 1,
				},
			},
		}
	}

	if result == nil {
		return nil, nil
	}

	return result.Tree, nil
}

func mssqlSyntaxCheck(statement string) (any, []*storepb.Advice) {
	result, err := tsqlparser.ParseTSQL(statement)
	if err != nil {
		if syntaxErr, ok := err.(*base.SyntaxError); ok {
			return nil, []*storepb.Advice{
				{
					Status:  storepb.Advice_WARNING,
					Code:    StatementSyntaxErrorCode,
					Title:   SyntaxErrorTitle,
					Content: syntaxErr.Message,
					StartPosition: &storepb.Position{
						Line:   int32(syntaxErr.Line),
						Column: int32(syntaxErr.Column),
					},
				},
			}
		}
		return nil, []*storepb.Advice{
			{
				Status:  storepb.Advice_WARNING,
				Code:    InternalErrorCode,
				Title:   "Parse error",
				Content: err.Error(),
				StartPosition: &storepb.Position{
					Line: 1,
				},
			},
		}
	}

	if result == nil {
		return nil, nil
	}

	return result.Tree, nil
}

func snowflakeSyntaxCheck(statement string) (any, []*storepb.Advice) {
	result, err := snowsqlparser.ParseSnowSQL(statement + ";")
	if err != nil {
		if syntaxErr, ok := err.(*base.SyntaxError); ok {
			return nil, []*storepb.Advice{
				{
					Status:  storepb.Advice_WARNING,
					Code:    StatementSyntaxErrorCode,
					Title:   SyntaxErrorTitle,
					Content: syntaxErr.Message,
					StartPosition: &storepb.Position{
						Line:   int32(syntaxErr.Line),
						Column: int32(syntaxErr.Column),
					},
				},
			}
		}
		return nil, []*storepb.Advice{
			{
				Status:  storepb.Advice_WARNING,
				Code:    InternalErrorCode,
				Title:   "Parse error",
				Content: err.Error(),
				StartPosition: &storepb.Position{
					Line: 1,
				},
			},
		}
	}
	if result == nil {
		return nil, nil
	}

	return result.Tree, nil
}

func oracleSyntaxCheck(statement string) (any, []*storepb.Advice) {
	tree, _, err := plsqlparser.ParsePLSQL(statement + ";")
	if err != nil {
		if syntaxErr, ok := err.(*base.SyntaxError); ok {
			return nil, []*storepb.Advice{
				{
					Status:  storepb.Advice_WARNING,
					Code:    StatementSyntaxErrorCode,
					Title:   SyntaxErrorTitle,
					Content: syntaxErr.Message,
					StartPosition: &storepb.Position{
						Line:   int32(syntaxErr.Line),
						Column: int32(syntaxErr.Column),
					},
				},
			}
		}
		return nil, []*storepb.Advice{
			{
				Status:  storepb.Advice_WARNING,
				Code:    InternalErrorCode,
				Title:   "Parse error",
				Content: err.Error(),
				StartPosition: &storepb.Position{
					Line: 1,
				},
			},
		}
	}

	return tree, nil
}

func postgresSyntaxCheck(statement string) (any, []*storepb.Advice) {
	nodes, err := pgrawparser.Parse(pgrawparser.ParseContext{}, statement)
	if err != nil {
		if _, ok := err.(*pgrawparser.ConvertError); ok {
			return nil, []*storepb.Advice{
				{
					Status:  storepb.Advice_ERROR,
					Code:    InternalErrorCode,
					Title:   "Parser conversion error",
					Content: err.Error(),
					StartPosition: &storepb.Position{
						Line: int32(calculatePostgresErrorLine(statement)),
					},
				},
			}
		}
		return nil, []*storepb.Advice{
			{
				Status:  storepb.Advice_ERROR,
				Code:    StatementSyntaxErrorCode,
				Title:   SyntaxErrorTitle,
				Content: err.Error(),
				StartPosition: &storepb.Position{
					Line: int32(calculatePostgresErrorLine(statement)),
				},
			},
		}
	}
	var res []ast.Node
	for _, node := range nodes {
		if node != nil {
			res = append(res, node)
		}
	}
	return res, nil
}

func calculatePostgresErrorLine(statement string) int {
	singleSQLs, err := base.SplitMultiSQL(storepb.Engine_POSTGRES, statement)
	if err != nil {
		// nolint:nilerr
		return 1
	}

	for _, singleSQL := range singleSQLs {
		if _, err := pgrawparser.Parse(pgrawparser.ParseContext{}, singleSQL.Text); err != nil {
			return singleSQL.LastLine
		}
	}

	return 0
}

func newTiDBParser() *tidbparser.Parser {
	p := tidbparser.New()

	// To support MySQL8 window function syntax.
	// See https://github.com/bytebase/bytebase/issues/175.
	p.EnableWindowFunc(true)

	return p
}

func mysqlSyntaxCheck(statement string) (any, []*storepb.Advice) {
	res, err := mysqlparser.ParseMySQL(statement)
	if err != nil {
		if syntaxErr, ok := err.(*base.SyntaxError); ok {
			return nil, []*storepb.Advice{
				{
					Status:  storepb.Advice_ERROR,
					Code:    StatementSyntaxErrorCode,
					Title:   SyntaxErrorTitle,
					Content: syntaxErr.Message,
					StartPosition: &storepb.Position{
						Line:   int32(syntaxErr.Line),
						Column: int32(syntaxErr.Column),
					},
				},
			}
		}
		return nil, []*storepb.Advice{
			{
				Status:  storepb.Advice_ERROR,
				Code:    InternalErrorCode,
				Title:   "Parse error",
				Content: err.Error(),
				StartPosition: &storepb.Position{
					Line: 1,
				},
			},
		}
	}

	return res, nil
}

func relocationTiDBErrorLine(errorMessage string, baseLine int) string {
	re := regexp.MustCompile(`line\s+(\d+)`)
	modified := re.ReplaceAllStringFunc(errorMessage, func(match string) string {
		matchSlice := re.FindStringSubmatch(match)
		if len(matchSlice) != 2 {
			return match
		}
		lineStr := matchSlice[1]
		lineNum, err := strconv.Atoi(lineStr)
		if err != nil {
			return match
		}
		newLine := lineNum + baseLine
		return fmt.Sprintf("line %d", newLine)
	})
	return modified
}

func tidbSyntaxCheck(statement string) (any, []*storepb.Advice) {
	singleSQLs, err := base.SplitMultiSQL(storepb.Engine_TIDB, statement)
	if err != nil {
		return nil, []*storepb.Advice{
			{
				Status:  storepb.Advice_WARNING,
				Code:    InternalErrorCode,
				Title:   "Syntax error",
				Content: err.Error(),
				StartPosition: &storepb.Position{
					Line: 1,
				},
			},
		}
	}

	p := newTiDBParser()
	var returnNodes []tidbast.StmtNode
	var adviceList []*storepb.Advice
	baseLine := 0
	for _, singleSQL := range singleSQLs {
		nodes, _, err := p.Parse(singleSQL.Text, "", "")
		if err != nil {
			return nil, []*storepb.Advice{
				{
					Status:  storepb.Advice_WARNING,
					Code:    InternalErrorCode,
					Title:   "Parse error",
					Content: relocationTiDBErrorLine(err.Error(), baseLine),
					StartPosition: &storepb.Position{
						Line: int32(baseLine + 1),
					},
				},
			}
		}

		if len(nodes) != 1 {
			continue
		}

		node := nodes[0]
		node.SetText(nil, singleSQL.Text)
		node.SetOriginTextPosition(singleSQL.LastLine)
		if n, ok := node.(*tidbast.CreateTableStmt); ok {
			if err := tidbbbparser.SetLineForMySQLCreateTableStmt(n); err != nil {
				return nil, append(adviceList, &storepb.Advice{
					Status:  storepb.Advice_ERROR,
					Code:    InternalErrorCode,
					Title:   "Set line error",
					Content: err.Error(),
					StartPosition: &storepb.Position{
						Line: int32(singleSQL.LastLine),
					},
				})
			}
		}
		returnNodes = append(returnNodes, node)
		baseLine = singleSQL.LastLine
	}

	return returnNodes, adviceList
}
