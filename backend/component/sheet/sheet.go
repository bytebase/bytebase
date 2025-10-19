package sheet

import (
	"context"
	"io"
	"log/slog"
	"strings"
	"sync"
	"time"

	lru "github.com/hashicorp/golang-lru/v2/expirable"
	"github.com/pkg/errors"
	"github.com/zeebo/xxh3"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	tsqlbatch "github.com/bytebase/bytebase/backend/plugin/parser/tsql/batch"
	"github.com/bytebase/bytebase/backend/store"

	// Import parsers to register their parse functions.
	_ "github.com/bytebase/bytebase/backend/plugin/parser/cockroachdb"
	_ "github.com/bytebase/bytebase/backend/plugin/parser/doris"
	_ "github.com/bytebase/bytebase/backend/plugin/parser/mysql"
	_ "github.com/bytebase/bytebase/backend/plugin/parser/partiql"
	_ "github.com/bytebase/bytebase/backend/plugin/parser/plsql"
	_ "github.com/bytebase/bytebase/backend/plugin/parser/redshift"
	_ "github.com/bytebase/bytebase/backend/plugin/parser/snowflake"
	_ "github.com/bytebase/bytebase/backend/plugin/parser/tidb"
	_ "github.com/bytebase/bytebase/backend/plugin/parser/tsql"
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

func getSheetCommands(engine storepb.Engine, statement string) []*storepb.Range {
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

func getSheetCommandsGeneral(engine storepb.Engine, statement string) []*storepb.Range {
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

	var sheetCommands []*storepb.Range
	p := 0
	for _, s := range singleSQLs {
		np := p + len(s.Text)
		sheetCommands = append(sheetCommands, &storepb.Range{
			Start: int32(p),
			End:   int32(np),
		})
		p = np
	}
	return sheetCommands
}

func getSheetCommandsFromByteOffset(engine storepb.Engine, statement string) []*storepb.Range {
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

	var sheetCommands []*storepb.Range
	for _, s := range singleSQLs {
		sheetCommands = append(sheetCommands, &storepb.Range{
			Start: int32(s.ByteOffsetStart),
			End:   int32(s.ByteOffsetEnd),
		})
	}
	return sheetCommands
}

func getSheetCommandsForMSSQL(statement string) []*storepb.Range {
	var sheetCommands []*storepb.Range

	batch := tsqlbatch.NewBatcher(statement)
	for {
		command, err := batch.Next()
		if err == io.EOF {
			b := batch.Batch()
			sheetCommands = append(sheetCommands, &storepb.Range{
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
			sheetCommands = append(sheetCommands, &storepb.Range{
				Start: int32(b.Start),
				End:   int32(b.End),
			})
			batch.Reset(nil)
		default:
		}
		// No command count limit for MSSQL to ensure consistency between sheet payload
		// and actual execution in mssql.go which splits and executes all batches
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
	ast, err := base.Parse(dbType, statement)
	if err != nil {
		result.advices = convertErrorToAdvice(err)
	} else {
		result.ast = ast
	}
	return result.ast, result.advices
}

func convertErrorToAdvice(err error) []*storepb.Advice {
	if syntaxErr, ok := err.(*base.SyntaxError); ok {
		return []*storepb.Advice{
			{
				Status:        storepb.Advice_ERROR,
				Code:          StatementSyntaxErrorCode,
				Title:         SyntaxErrorTitle,
				Content:       syntaxErr.Message,
				StartPosition: syntaxErr.Position,
			},
		}
	}
	return []*storepb.Advice{
		{
			Status:        storepb.Advice_ERROR,
			Code:          InternalErrorCode,
			Title:         SyntaxErrorTitle,
			Content:       err.Error(),
			StartPosition: nil,
		},
	}
}
