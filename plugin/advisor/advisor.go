// Package advisor defines the interface for analyzing sql statements.
// The advisor could be syntax checker, index suggestion etc.
package advisor

import (
	"fmt"
	"sync"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/common"
	"github.com/bytebase/bytebase/plugin/db"
	"go.uber.org/zap"
)

// Status is the advisor result status.
type Status string

const (
	// Success is the advisor status for successes.
	Success Status = "SUCCESS"
	// Warn is the advisor status for warnings.
	Warn Status = "WARN"
	// Error is the advisor status for errors.
	Error Status = "ERROR"
)

func (e Status) String() string {
	switch e {
	case Success:
		return "INFO"
	case Warn:
		return "WARN"
	case Error:
		return "ERROR"
	}
	return "UNKNOWN"
}

// NewStatusBySchemaReviewRuleLevel returns status by SchemaReviewRuleLevel.
func NewStatusBySchemaReviewRuleLevel(level api.SchemaReviewRuleLevel) (Status, error) {
	switch level {
	case api.SchemaRuleLevelError:
		return Error, nil
	case api.SchemaRuleLevelWarning:
		return Warn, nil
	}
	return "", fmt.Errorf("unexpected rule level type: %s", level)
}

// Type is the type of advisor.
type Type string

const (
	// Fake is a fake advisor type for testing.
	Fake Type = "bb.plugin.advisor.fake"
	// MySQLSyntax is an advisor type for MySQL syntax.
	MySQLSyntax Type = "bb.plugin.advisor.mysql.syntax"
	// MySQLMigrationCompatibility is an advisor type for MySQL migration compatibility.
	MySQLMigrationCompatibility Type = "bb.plugin.advisor.mysql.migration-compatibility"
	// MySQLWhereRequirement is an advisor type for MySQL WHERE clause requirement.
	MySQLWhereRequirement Type = "bb.plugin.advisor.mysql.where.require"
	// MySQLTableNamingConvention is an advisor type for MySQL table naming convention.
	MySQLTableNamingConvention Type = "bb.plugin.advisor.mysql.naming.table"
)

// Advice is the result of an advisor.
type Advice struct {
	Status  Status
	Code    common.Code
	Title   string
	Content string
}

// Context is the context for advisor.
type Context struct {
	Logger    *zap.Logger
	Charset   string
	Collation string

	// Schema review rule special fields.
	Rule *api.SchemaReviewRule
}

// Advisor is the interface for advisor.
type Advisor interface {
	Check(ctx Context, statement string) ([]Advice, error)
}

var (
	advisorMu sync.RWMutex
	advisors  = make(map[db.Type]map[Type]Advisor)
)

// Register makes a advisor available by the provided id.
// If Register is called twice with the same name or if advisor is nil,
// it panics.
func Register(dbType db.Type, advType Type, f Advisor) {
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
func Check(dbType db.Type, advType Type, ctx Context, statement string) ([]Advice, error) {
	advisorMu.RLock()
	dbAdvisors, ok := advisors[dbType]
	defer advisorMu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("advisor: unknown advisor %v for %v", advType, dbType)
	}

	f, ok := dbAdvisors[advType]
	if !ok {
		return nil, fmt.Errorf("advisor: unknown advisor %v for %v", advType, dbType)
	}

	return f.Check(ctx, statement)
}
