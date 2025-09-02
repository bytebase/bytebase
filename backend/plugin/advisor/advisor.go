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
	"github.com/bytebase/bytebase/backend/plugin/advisor/catalog"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
)

const (
	// SyntaxErrorTitle is the error title for syntax error.
	SyntaxErrorTitle string = "Syntax error"
)

// NewStatusBySQLReviewRuleLevel returns status by SQLReviewRuleLevel.
func NewStatusBySQLReviewRuleLevel(level storepb.SQLReviewRuleLevel) (storepb.Advice_Status, error) {
	switch level {
	case storepb.SQLReviewRuleLevel_ERROR:
		return storepb.Advice_ERROR, nil
	case storepb.SQLReviewRuleLevel_WARNING:
		return storepb.Advice_WARNING, nil
	default:
		return storepb.Advice_STATUS_UNSPECIFIED, errors.Errorf("unexpected rule level type: %s", level)
	}
}

// Context is the context for advisor.
type Context struct {
	DBSchema              *storepb.DatabaseSchemaMetadata
	ChangeType            storepb.PlanCheckRunConfig_ChangeDatabaseType
	EnablePriorBackup     bool
	ClassificationConfig  *storepb.DataClassificationSetting_DataClassificationConfig
	ListDatabaseNamesFunc base.ListDatabaseNamesFunc
	InstanceID            string
	IsObjectCaseSensitive bool

	// SQL review rule special fields.
	AST     any
	Rule    *storepb.SQLReviewRule
	Catalog *catalog.Finder
	Driver  *sql.DB

	// CurrentDatabase is the current database.
	CurrentDatabase string
	// Statement is the original statement of AST, it is used for some PostgreSQL
	// advisors which need to check the token stream.
	Statements string
	// UsePostgresDatabaseOwner is true if the advisor should use the database owner as default role.
	UsePostgresDatabaseOwner bool
}

// Advisor is the interface for advisor.
type Advisor interface {
	Check(ctx context.Context, checkCtx Context) ([]*storepb.Advice, error)
}

var (
	advisorMu sync.RWMutex
	advisors  = make(map[storepb.Engine]map[SQLReviewRuleType]Advisor)
)

// Register makes a advisor available by the provided id.
// If Register is called twice with the same name or if advisor is nil,
// it panics.
func Register(dbType storepb.Engine, ruleType SQLReviewRuleType, f Advisor) {
	advisorMu.Lock()
	defer advisorMu.Unlock()
	if f == nil {
		panic("advisor: Register advisor is nil")
	}
	dbAdvisors, ok := advisors[dbType]
	if !ok {
		advisors[dbType] = map[SQLReviewRuleType]Advisor{
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
func Check(ctx context.Context, dbType storepb.Engine, ruleType SQLReviewRuleType, checkCtx Context) (adviceList []*storepb.Advice, err error) {
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
