// Package advisor defines the interface for analyzing sql statements.
// The advisor could be syntax checker, index suggestion etc.
package advisor

import (
	"fmt"
	"sync"

	"github.com/bytebase/bytebase/common"
	"github.com/bytebase/bytebase/plugin/catalog"
	"github.com/bytebase/bytebase/plugin/db"
	"go.uber.org/zap/zapcore"
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

	// SyntaxErrorTitle is the error title for syntax error.
	SyntaxErrorTitle string = "Syntax error"
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
func NewStatusBySchemaReviewRuleLevel(level SchemaReviewRuleLevel) (Status, error) {
	switch level {
	case SchemaRuleLevelError:
		return Error, nil
	case SchemaRuleLevelWarning:
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

	// MySQLUseInnoDB is an advisor type for MySQL InnoDB Engine.
	MySQLUseInnoDB Type = "bb.plugin.advisor.mysql.use-innodb"

	// MySQLMigrationCompatibility is an advisor type for MySQL migration compatibility.
	MySQLMigrationCompatibility Type = "bb.plugin.advisor.mysql.migration-compatibility"

	// MySQLWhereRequirement is an advisor type for MySQL WHERE clause requirement.
	MySQLWhereRequirement Type = "bb.plugin.advisor.mysql.where.require"

	// MySQLNoLeadingWildcardLike is an advisor type for MySQL no leading wildcard LIKE.
	MySQLNoLeadingWildcardLike Type = "bb.plugin.advisor.mysql.where.no-leading-wildcard-like"

	// MySQLNamingTableConvention is an advisor type for MySQL table naming convention.
	MySQLNamingTableConvention Type = "bb.plugin.advisor.mysql.naming.table"

	// MySQLNamingIndexConvention is an advisor type for MySQL index key naming convention.
	MySQLNamingIndexConvention Type = "bb.plugin.advisor.mysql.naming.index"

	// MySQLNamingUKConvention is an advisor type for MySQL unique key naming convention.
	MySQLNamingUKConvention Type = "bb.plugin.advisor.mysql.naming.uk"

	// MySQLNamingFKConvention is an advisor type for MySQL foreign key naming convention.
	MySQLNamingFKConvention Type = "bb.plugin.advisor.mysql.naming.fk"

	// MySQLNamingColumnConvention is an advisor type for MySQL column naming convention.
	MySQLNamingColumnConvention Type = "bb.plugin.advisor.mysql.naming.column"

	// MySQLColumnRequirement is an advisor type for MySQL column requirement.
	MySQLColumnRequirement Type = "bb.plugin.advisor.mysql.column.require"

	// MySQLColumnNoNull is an advisor type for MySQL column no NULL value.
	MySQLColumnNoNull Type = "bb.plugin.advisor.mysql.column.no-null"

	// MySQLNoSelectAll is an advisor type for MySQL no select all.
	MySQLNoSelectAll Type = "bb.plugin.advisor.mysql.select.no-select-all"

	// MySQLTableRequirePK is an advisor type for MySQL table require primary key.
	MySQLTableRequirePK Type = "bb.plugin.advisor.mysql.table.require-pk"
)

// Advice is the result of an advisor.
type Advice struct {
	Status  Status      `json:"status"`
	Code    common.Code `json:"code"`
	Title   string      `json:"title"`
	Content string      `json:"content"`
}

// MarshalLogObject constructs a field that carries Advice.
func (a Advice) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	enc.AddString("status", a.Status.String())
	enc.AddInt("code", int(a.Code))
	enc.AddString("title", a.Title)
	enc.AddString("content", a.Content)
	return nil
}

// ZapAdviceArray is a helper to format zap.Array.
type ZapAdviceArray []Advice

// MarshalLogArray implements the zapcore.ArrayMarshaler interface.
func (array ZapAdviceArray) MarshalLogArray(enc zapcore.ArrayEncoder) error {
	for i := range array {
		if err := enc.AppendObject(array[i]); err != nil {
			return err
		}
	}
	return nil
}

// Context is the context for advisor.
type Context struct {
	Charset   string
	Collation string

	// Schema review rule special fields.
	Rule    *SchemaReviewRule
	Catalog catalog.Catalog
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
