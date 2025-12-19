package sheet

import (
	"sync"
	"time"

	lru "github.com/hashicorp/golang-lru/v2/expirable"
	"github.com/zeebo/xxh3"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"

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

	statementCache *lru.LRU[astHashKey, *StatementResult]
}

type astHashKey struct {
	hash   uint64
	engine storepb.Engine
}

// NewManager creates a new sheet manager.
func NewManager() *Manager {
	return &Manager{
		statementCache: lru.NewLRU[astHashKey, *StatementResult](8, nil, 3*time.Minute),
	}
}

// StatementResult holds the cached parsing results with the unified ParsedStatement type.
type StatementResult struct {
	sync.Mutex
	statements []base.ParsedStatement
	advices    []*storepb.Advice
}

// GetStatementsForChecks gets the unified Statements (with both text and AST) with caching.
// This is the new unified API that returns complete ParsedStatement objects.
// Use this for new code that needs both text and AST information.
func (sm *Manager) GetStatementsForChecks(dbType storepb.Engine, statement string) ([]base.ParsedStatement, []*storepb.Advice) {
	var result *StatementResult
	h := xxh3.HashString(statement)
	key := astHashKey{hash: h, engine: dbType}
	sm.Lock()
	if v, ok := sm.statementCache.Get(key); ok {
		result = v
	} else {
		result = &StatementResult{}
		sm.statementCache.Add(key, result)
	}
	sm.Unlock()

	result.Lock()
	defer result.Unlock()
	if result.statements != nil || result.advices != nil {
		return result.statements, result.advices
	}
	statements, err := base.ParseStatements(dbType, statement)
	if err != nil {
		result.advices = convertErrorToAdvice(err)
	} else {
		result.statements = statements
	}
	return result.statements, result.advices
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
