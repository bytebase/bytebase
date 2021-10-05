// Advisor defines the interface for analyzing sql statements.
// The advisor could be syntax checker, index suggestion etc.
package advisor

import (
	"fmt"
	"sync"

	"github.com/bytebase/bytebase/common"
	"github.com/bytebase/bytebase/plugin/db"
	"go.uber.org/zap"
)

type Status string

const (
	Success Status = "SUCCESS"
	Warn    Status = "WARN"
	Error   Status = "ERROR"
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

type AdvisorType string

const (
	Fake                        AdvisorType = "bb.plugin.advisor.fake"
	MySQLSyntax                 AdvisorType = "bb.plugin.advisor.mysql.syntax"
	MySQLMigrationCompatibility AdvisorType = "bb.plugin.advisor.mysql.migration-compatibility"
)

type Advice struct {
	Status  Status
	Code    common.Code
	Title   string
	Content string
}

type AdvisorContext struct {
	Logger    *zap.Logger
	Charset   string
	Collation string
}

type Advisor interface {
	Check(ctx AdvisorContext, statement string) ([]Advice, error)
}

var (
	advisorMu sync.RWMutex
	advisors  = make(map[db.Type]map[AdvisorType]Advisor)
)

// Register makes a advisor available by the provided id.
// If Register is called twice with the same name or if advisor is nil,
// it panics.
func Register(dbType db.Type, advType AdvisorType, f Advisor) {
	advisorMu.Lock()
	defer advisorMu.Unlock()
	if f == nil {
		panic("advisor: Register advisor is nil")
	}
	dbAdvisors, ok := advisors[dbType]
	if !ok {
		advisors[dbType] = map[AdvisorType]Advisor{
			advType: f,
		}
	} else {
		if _, dup := dbAdvisors[advType]; dup {
			panic(fmt.Sprintf("advisor: Register called twice for advisor %v for %v", advType, dbType))
		}
		dbAdvisors[advType] = f
	}
}

func Check(dbType db.Type, advType AdvisorType, ctx AdvisorContext, statement string) ([]Advice, error) {
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
