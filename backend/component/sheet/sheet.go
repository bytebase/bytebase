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
	"github.com/pkg/errors"
	"github.com/zeebo/xxh3"

	tidbparser "github.com/pingcap/tidb/pkg/parser"
	tidbast "github.com/pingcap/tidb/pkg/parser/ast"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	crparser "github.com/bytebase/bytebase/backend/plugin/parser/cockroachdb"
	mysqlparser "github.com/bytebase/bytebase/backend/plugin/parser/mysql"
	partiqlparser "github.com/bytebase/bytebase/backend/plugin/parser/partiql"
	pgrawparser "github.com/bytebase/bytebase/backend/plugin/parser/pg/legacy"
	"github.com/bytebase/bytebase/backend/plugin/parser/pg/legacy/ast"
	plsqlparser "github.com/bytebase/bytebase/backend/plugin/parser/plsql"
	redshiftparser "github.com/bytebase/bytebase/backend/plugin/parser/redshift"
	snowsqlparser "github.com/bytebase/bytebase/backend/plugin/parser/snowflake"
	tidbbbparser "github.com/bytebase/bytebase/backend/plugin/parser/tidb"
	tsqlparser "github.com/bytebase/bytebase/backend/plugin/parser/tsql"
	tsqlbatch "github.com/bytebase/bytebase/backend/plugin/parser/tsql/batch"
	"github.com/bytebase/bytebase/backend/store"
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
	astCache *lru.LRU[astHashKey, *Result]
}

type astHashKey struct {
	hash   uint64
	engine storepb.Engine
}

// NewManager creates a new sheet manager.
func NewManager(store *store.Store) *Manager {
	return &Manager{
		store:    store,
		astCache: lru.NewLRU[astHashKey, *Result](8, nil, 3*time.Minute),
	}
}

func (sm *Manager) CreateSheet(ctx context.Context, sheet *store.SheetMessage) (*store.SheetMessage, error) {
	if sheet.Payload == nil {
		sheet.Payload = &storepb.SheetPayload{}
	}
	sheet.Payload.Commands = getSheetCommands(sheet.Payload.Engine, sheet.Statement)

	return sm.store.CreateSheet(ctx, sheet)
}

func (sm *Manager) BatchCreateSheets(ctx context.Context, sheets []*store.SheetMessage, projectID string, creatorUID int) ([]*store.SheetMessage, error) {
	for _, sheet := range sheets {
		if sheet.Payload == nil {
			sheet.Payload = &storepb.SheetPayload{}
		}
		sheet.Payload.Commands = getSheetCommands(sheet.Payload.Engine, sheet.Statement)
	}

	sheets, err := sm.store.BatchCreateSheet(ctx, projectID, sheets, creatorUID)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to batch create sheets")
	}
	return sheets, nil
}

func getSheetCommands(engine storepb.Engine, statement string) []*storepb.SheetCommand {
	// Burnout for large SQL.
	if len(statement) > common.MaxSheetCheckSize {
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
	// HACK(p0ny): always split for pg
	if len(singleSQLs) > common.MaximumCommands && engine != storepb.Engine_POSTGRES {
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

	batch := tsqlbatch.NewBatcher(statement)
	for {
		command, err := batch.Next()
		if err == io.EOF {
			b := batch.Batch()
			sheetCommands = append(sheetCommands, &storepb.SheetCommand{
				Start: int32(b.Start),
				End:   int32(b.End),
			})
			batch.Reset(nil)
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
			b := batch.Batch()
			sheetCommands = append(sheetCommands, &storepb.SheetCommand{
				Start: int32(b.Start),
				End:   int32(b.End),
			})
			batch.Reset(nil)
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

// GetASTsForChecks gets the ASTs of statement with caching, and it should only be used
// for plan checks because it involves some truncating.
func (sm *Manager) GetASTsForChecks(dbType storepb.Engine, statement string) (any, []*storepb.Advice) {
	var result *Result
	h := xxh3.HashString(statement)
	key := astHashKey{hash: h, engine: dbType}
	sm.Lock()
	if v, ok := sm.astCache.Get(key); ok {
		result = v
	} else {
		result = &Result{}
		sm.astCache.Add(key, result)
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
	case storepb.Engine_REDSHIFT:
		return redshiftSyntaxCheck(statement)
	case storepb.Engine_ORACLE:
		return oracleSyntaxCheck(statement)
	case storepb.Engine_SNOWFLAKE:
		return snowflakeSyntaxCheck(statement)
	case storepb.Engine_MSSQL:
		return mssqlSyntaxCheck(statement)
	case storepb.Engine_DYNAMODB:
		return partiqlSyntaxCheck(statement)
	case storepb.Engine_COCKROACHDB:
		return cockroachdbSyntaxCheck(statement)
	default:
		// Return default advice for unsupported database types
	}
	return nil, []*storepb.Advice{
		{
			Status:        storepb.Advice_ERROR,
			Code:          InternalErrorCode,
			Title:         "Unsupported database type",
			Content:       fmt.Sprintf("Unsupported database type %s", dbType),
			StartPosition: common.FirstLinePosition,
		},
	}
}

func cockroachdbSyntaxCheck(statement string) (any, []*storepb.Advice) {
	result, err := crparser.ParseCockroachDBSQL(statement)
	if err != nil {
		return nil, []*storepb.Advice{
			{
				Status:        storepb.Advice_WARNING,
				Code:          InternalErrorCode,
				Title:         "Parse error",
				Content:       err.Error(),
				StartPosition: common.FirstLinePosition,
			},
		}
	}

	if result == nil {
		return nil, nil
	}
	return result.Stmts, nil
}

func partiqlSyntaxCheck(statement string) (any, []*storepb.Advice) {
	result, err := partiqlparser.ParsePartiQL(statement)
	if err != nil {
		if syntaxErr, ok := err.(*base.SyntaxError); ok {
			return nil, []*storepb.Advice{
				{
					Status:        storepb.Advice_WARNING,
					Code:          StatementSyntaxErrorCode,
					Title:         SyntaxErrorTitle,
					Content:       syntaxErr.Message,
					StartPosition: syntaxErr.Position,
				},
			}
		}
		return nil, []*storepb.Advice{
			{
				Status:        storepb.Advice_WARNING,
				Code:          InternalErrorCode,
				Title:         "Parse error",
				Content:       err.Error(),
				StartPosition: common.FirstLinePosition,
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
					Status:        storepb.Advice_WARNING,
					Code:          StatementSyntaxErrorCode,
					Title:         SyntaxErrorTitle,
					Content:       syntaxErr.Message,
					StartPosition: syntaxErr.Position,
				},
			}
		}
		return nil, []*storepb.Advice{
			{
				Status:        storepb.Advice_WARNING,
				Code:          InternalErrorCode,
				Title:         "Parse error",
				Content:       err.Error(),
				StartPosition: common.FirstLinePosition,
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
					Status:        storepb.Advice_WARNING,
					Code:          StatementSyntaxErrorCode,
					Title:         SyntaxErrorTitle,
					Content:       syntaxErr.Message,
					StartPosition: syntaxErr.Position,
				},
			}
		}
		return nil, []*storepb.Advice{
			{
				Status:        storepb.Advice_WARNING,
				Code:          InternalErrorCode,
				Title:         "Parse error",
				Content:       err.Error(),
				StartPosition: common.FirstLinePosition,
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
					Status:        storepb.Advice_WARNING,
					Code:          StatementSyntaxErrorCode,
					Title:         SyntaxErrorTitle,
					Content:       syntaxErr.Message,
					StartPosition: syntaxErr.Position,
				},
			}
		}
		return nil, []*storepb.Advice{
			{
				Status:        storepb.Advice_WARNING,
				Code:          InternalErrorCode,
				Title:         "Parse error",
				Content:       err.Error(),
				StartPosition: common.FirstLinePosition,
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

func redshiftSyntaxCheck(statement string) (any, []*storepb.Advice) {
	// Parse using redshift parser to get ANTLR tree
	result, err := redshiftparser.ParseRedshift(statement)
	if err != nil {
		if syntaxErr, ok := err.(*base.SyntaxError); ok {
			return nil, []*storepb.Advice{
				{
					Status:        storepb.Advice_ERROR,
					Code:          StatementSyntaxErrorCode,
					Title:         SyntaxErrorTitle,
					Content:       syntaxErr.Message,
					StartPosition: syntaxErr.Position,
				},
			}
		}
		return nil, []*storepb.Advice{
			{
				Status:        storepb.Advice_ERROR,
				Code:          InternalErrorCode,
				Title:         "Parse error",
				Content:       err.Error(),
				StartPosition: common.FirstLinePosition,
			},
		}
	}
	if result == nil {
		return nil, nil
	}
	return result.Tree, nil
}

func calculatePostgresErrorLine(statement string) int {
	singleSQLs, err := base.SplitMultiSQL(storepb.Engine_POSTGRES, statement)
	if err != nil {
		// nolint:nilerr
		return 1
	}

	for _, singleSQL := range singleSQLs {
		if _, err := pgrawparser.Parse(pgrawparser.ParseContext{}, singleSQL.Text); err != nil {
			return int(singleSQL.End.GetLine()) + 1
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
					Status:        storepb.Advice_ERROR,
					Code:          StatementSyntaxErrorCode,
					Title:         SyntaxErrorTitle,
					Content:       syntaxErr.Message,
					StartPosition: syntaxErr.Position,
				},
			}
		}
		return nil, []*storepb.Advice{
			{
				Status:        storepb.Advice_ERROR,
				Code:          InternalErrorCode,
				Title:         "Parse error",
				Content:       err.Error(),
				StartPosition: common.FirstLinePosition,
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
				Status:        storepb.Advice_WARNING,
				Code:          InternalErrorCode,
				Title:         "Syntax error",
				Content:       err.Error(),
				StartPosition: common.FirstLinePosition,
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
						Line: int32(baseLine),
					},
				},
			}
		}

		if len(nodes) != 1 {
			continue
		}

		node := nodes[0]
		node.SetText(nil, singleSQL.Text)
		node.SetOriginTextPosition(int(singleSQL.End.GetLine()))
		if n, ok := node.(*tidbast.CreateTableStmt); ok {
			if err := tidbbbparser.SetLineForMySQLCreateTableStmt(n); err != nil {
				return nil, append(adviceList, &storepb.Advice{
					Status:  storepb.Advice_ERROR,
					Code:    InternalErrorCode,
					Title:   "Set line error",
					Content: err.Error(),
					StartPosition: &storepb.Position{
						Line: singleSQL.End.GetLine(),
					},
				})
			}
		}
		returnNodes = append(returnNodes, node)
		baseLine = int(singleSQL.End.GetLine())
	}

	return returnNodes, adviceList
}
