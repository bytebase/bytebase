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
	"github.com/bytebase/bytebase/backend/plugin/advisor/catalog"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
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
	}
	return storepb.Advice_STATUS_UNSPECIFIED, errors.Errorf("unexpected rule level type: %s", level)
}

// SyntaxMode is the type of syntax mode.
type SyntaxMode int

const (
	// SyntaxModeNormal is the normal syntax mode.
	SyntaxModeNormal SyntaxMode = iota
	// SyntaxModeSDL is the SDL syntax mode.
	SyntaxModeSDL
)

// Context is the context for advisor.
type Context struct {
	Charset               string
	Collation             string
	DBSchema              *storepb.DatabaseSchemaMetadata
	SyntaxMode            SyntaxMode
	ChangeType            storepb.PlanCheckRunConfig_ChangeDatabaseType
	PreUpdateBackupDetail *storepb.PreUpdateBackupDetail
	ClassificationConfig  *storepb.DataClassificationSetting_DataClassificationConfig
	ListDatabaseNamesFunc base.ListDatabaseNamesFunc
	InstanceID            string

	// SQL review rule special fields.
	AST     any
	Rule    *storepb.SQLReviewRule
	Catalog *catalog.Finder
	Driver  *sql.DB
	Context context.Context

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
	Check(ctx Context, statement string) ([]*storepb.Advice, error)
}

var (
	advisorMu sync.RWMutex
	advisors  = make(map[storepb.Engine]map[Type]Advisor)
)

// Register makes a advisor available by the provided id.
// If Register is called twice with the same name or if advisor is nil,
// it panics.
func Register(dbType storepb.Engine, advType Type, f Advisor) {
	advisorMu.Lock()
	defer advisorMu.Unlock()
	if f == nil {
		panic("advisor: Register advisor is nil")
	}
	dbAdvisors, ok := advisors[dbType]
	if !ok {
		advisors[dbType] = map[Type]Advisor{
			advType: f,
		}
	} else {
		if _, dup := dbAdvisors[advType]; dup {
			panic(fmt.Sprintf("advisor: Register called twice for advisor %v for %v", advType, dbType))
		}
		dbAdvisors[advType] = f
	}
}

// Check runs the advisor and returns the advices.
func Check(dbType storepb.Engine, advType Type, ctx Context, statement string) (adviceList []*storepb.Advice, err error) {
	defer func() {
		if panicErr := recover(); panicErr != nil {
			panicErr, ok := panicErr.(error)
			if !ok {
				panicErr = errors.Errorf("%v", panicErr)
			}
			err = errors.Errorf("advisor check PANIC RECOVER, type: %v, err: %v", advType, panicErr)

			slog.Error("advisor check PANIC RECOVER", log.BBError(panicErr), log.BBStack("panic-stack"))
		}
	}()

	advisorMu.RLock()
	dbAdvisors, ok := advisors[dbType]
	defer advisorMu.RUnlock()
	if !ok {
		return nil, errors.Errorf("advisor: unknown db advisor type %v", dbType)
	}

	f, ok := dbAdvisors[advType]
	if !ok {
		return nil, errors.Errorf("advisor: unknown advisor %v for %v", advType, dbType)
	}

	return f.Check(ctx, statement)
}
