// Package advisor defines the interface for analyzing sql statements.
// The advisor could be syntax checker, index suggestion etc.
package advisor

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	metadatapb "github.com/bytebase/omni/metadata"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common/log"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/db"
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
	DBSchema              *metadatapb.DatabaseSchemaMetadata
	EnablePriorBackup     bool
	EnableGhost           bool
	ListDatabaseNamesFunc base.ListDatabaseNamesFunc
	InstanceID            string
	IsObjectCaseSensitive bool

	// SQL review rule special fields.
	Rule             *storepb.SQLReviewRule
	OriginalMetadata *model.DatabaseMetadata
	FinalMetadata    *model.DatabaseMetadata
	// Prober runs the live probes the rules need. Nil means the review has no
	// live connection; a rule that needs one reports that it was not checked
	// (see UnsupportedError) rather than returning no advice.
	Prober db.Prober
	// ParsedStatements contains complete per-statement info including text.
	ParsedStatements []base.ParsedStatement

	// CurrentDatabase is the current database.
	CurrentDatabase string
	// StatementsTotalSize is the total size of all statements in bytes.
	// Used for size limit checks without needing the full statement text.
	StatementsTotalSize int
	// TenantMode indicates whether to use database owner role for PostgreSQL tenant mode.
	// The probe's role switch belongs to the driver, which reads its own connection
	// context, so a caller that opens the driver for a review must pass this same value to
	// it (db.ConnectionContext.TenantMode); this field only decides whether a rule probes
	// as the owner or trusts the synced schema's owner.
	TenantMode bool

	// SQL review level fields.
	DBType      storepb.Engine
	SessionUser string

	// Used for test only.
	NoAppendBuiltin bool

	// memo is a per-review key/value cache shared across rules in one
	// SQLReviewCheck invocation. Initialized once by SQLReviewCheck before the
	// rule loop; the map header propagates by value across rule invocations
	// (advisors run sequentially, no synchronization needed).
	//
	// Engine-specific helpers use this to amortize work that would otherwise
	// repeat per advisor, such as memoized AST extraction in V1Engine.
	//
	// Callers that bypass SQLReviewCheck and construct Context directly
	// without initializing memo will see Memo/SetMemo silently no-op (cache
	// disabled, work repeats). Test fixtures and ad-hoc callers should
	// initialize memo if they want caching behavior.
	memo map[string]any
}

// Memo returns the value previously stored under key.
// Returns (nil, false) on cache miss or uninitialized cache.
func (c Context) Memo(key string) (any, bool) {
	if c.memo == nil {
		return nil, false
	}
	v, ok := c.memo[key]
	return v, ok
}

// SetMemo stores a value under key for retrieval by Memo.
// No-op if the cache is uninitialized.
func (c Context) SetMemo(key string, value any) {
	if c.memo == nil {
		return
	}
	c.memo[key] = value
}

// InitMemo initializes the per-review memo cache if not already initialized.
// Called by SQLReviewCheck before its rule loop. Tests that bypass
// SQLReviewCheck and want caching behavior should call this on the Context
// they construct (Context is passed by value through the rule loop, so the
// init must happen on the original before any copies are made).
func (c *Context) InitMemo() {
	if c.memo == nil {
		c.memo = make(map[string]any)
	}
}

// Advisor is the interface for advisor.
type Advisor interface {
	Check(ctx context.Context, checkCtx Context) ([]*storepb.Advice, error)
}

// Func is an adapter to allow the use of ordinary functions as Advisors.
type Func func(ctx context.Context, checkCtx Context) ([]*storepb.Advice, error)

// Check calls f(ctx, checkCtx).
func (f Func) Check(ctx context.Context, checkCtx Context) ([]*storepb.Advice, error) {
	return f(ctx, checkCtx)
}

// RegisterRules registers every rule in rules for dbType.
func RegisterRules(dbType storepb.Engine, rules map[storepb.SQLReviewRule_Type]Advisor) {
	for ruleType, a := range rules {
		Register(dbType, ruleType, a)
	}
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

// IsRegistered reports whether an advisor is registered for dbType and ruleType.
func IsRegistered(dbType storepb.Engine, ruleType storepb.SQLReviewRule_Type) bool {
	advisorMu.RLock()
	defer advisorMu.RUnlock()
	_, ok := advisors[dbType][ruleType]
	return ok
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
