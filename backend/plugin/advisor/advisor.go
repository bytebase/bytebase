// Package advisor defines the interface for analyzing sql statements.
// The advisor could be syntax checker, index suggestion etc.
package advisor

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"sync"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common/log"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/store/model"
)

const (
	// SyntaxErrorTitle is the error title for syntax error.
	SyntaxErrorTitle string = "Syntax error"
)

// NewStatusBySQLReviewRuleLevel returns status by SQLReviewRuleLevel.
func NewStatusBySQLReviewRuleLevel(level storepb.SQLReviewRule_Level) (storepb.Advice_Status, error) {
	switch level {
	case storepb.SQLReviewRule_ERROR:
		return storepb.Advice_ERROR, nil
	case storepb.SQLReviewRule_WARNING:
		return storepb.Advice_WARNING, nil
	default:
		return storepb.Advice_STATUS_UNSPECIFIED, errors.Errorf("unexpected rule level type: %s", level)
	}
}

// Context is the context for advisor.
type Context struct {
	DBSchema              *storepb.DatabaseSchemaMetadata
	EnablePriorBackup     bool
	EnableGhost           bool
	ListDatabaseNamesFunc base.ListDatabaseNamesFunc
	InstanceID            string
	IsObjectCaseSensitive bool

	// SQL review rule special fields.
	Rule             *storepb.SQLReviewRule
	OriginalMetadata *model.DatabaseMetadata
	FinalMetadata    *model.DatabaseMetadata
	Driver           *sql.DB
	// ParsedStatements contains complete per-statement info including text.
	ParsedStatements []base.ParsedStatement

	// CurrentDatabase is the current database.
	CurrentDatabase string
	// StatementsTotalSize is the total size of all statements in bytes.
	// Used for size limit checks without needing the full statement text.
	StatementsTotalSize int
	// TenantMode indicates whether to use database owner role for PostgreSQL tenant mode.
	TenantMode bool

	// SQL review level fields.
	DBType storepb.Engine

	// Snowflake specific fields (duplicates CurrentDatabase, kept for compatibility).
	// CurrentDatabase string

	// Used for test only.
	NoAppendBuiltin bool
}

// Advisor is the interface for advisor.
type Advisor interface {
	Check(ctx context.Context, checkCtx Context) ([]*storepb.Advice, error)
}

var (
	advisorMu sync.RWMutex
	advisors  = make(map[storepb.Engine]map[storepb.SQLReviewRule_Type]Advisor)
)

// Register makes a advisor available by the provided id.
// If Register is called twice with the same name or if advisor is nil,
// it panics.
func Register(dbType storepb.Engine, ruleType storepb.SQLReviewRule_Type, f Advisor) {
	advisorMu.Lock()
	defer advisorMu.Unlock()
	if f == nil {
		panic("advisor: Register advisor is nil")
	}
	dbAdvisors, ok := advisors[dbType]
	if !ok {
		advisors[dbType] = map[storepb.SQLReviewRule_Type]Advisor{
			ruleType: f,
		}
	} else {
		if _, dup := dbAdvisors[ruleType]; dup {
			panic(fmt.Sprintf("advisor: Register called twice for advisor %v for %v", ruleType, dbType))
		}
		dbAdvisors[ruleType] = f
	}
}

// Check runs the advisor and returns the advices.
func Check(ctx context.Context, dbType storepb.Engine, ruleType storepb.SQLReviewRule_Type, checkCtx Context) (adviceList []*storepb.Advice, err error) {
	defer func() {
		if panicErr := recover(); panicErr != nil {
			panicErr, ok := panicErr.(error)
			if !ok {
				panicErr = errors.Errorf("%v", panicErr)
			}
			err = errors.Errorf("advisor check PANIC RECOVER, type: %v, err: %v", ruleType, panicErr)

			slog.Error("advisor check PANIC RECOVER", log.BBError(panicErr), log.BBStack("panic-stack"))
		}
	}()

	advisorMu.RLock()
	dbAdvisors, ok := advisors[dbType]
	defer advisorMu.RUnlock()
	if !ok {
		return nil, errors.Errorf("advisor: unknown db advisor type %v", dbType)
	}

	f, ok := dbAdvisors[ruleType]
	if !ok {
		return nil, errors.Errorf("advisor: unknown advisor %v for %v", ruleType, dbType)
	}

	return f.Check(ctx, checkCtx)
}
