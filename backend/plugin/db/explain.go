package db

import (
	"fmt"
	"sync"

	"github.com/pkg/errors"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
)

// Explain is how an engine answers an explain request. A driver whose engine
// can explain registers one with RegisterExplain.
type Explain struct {
	// Formats lists the plan formats a request may name.
	Formats []v1pb.QueryOption_ExplainFormat
	// DefaultFormat is the plan format when a request names none, or
	// EXPLAIN_FORMAT_UNSPECIFIED when the session decides.
	DefaultFormat v1pb.QueryOption_ExplainFormat
	// Statement returns the statement whose result is statement's plan in
	// format, or an error when statement cannot be explained without running
	// it. It is nil when the driver plans through the engine's own API, which
	// never runs the statement.
	Statement func(statement string, format v1pb.QueryOption_ExplainFormat) (string, error)
}

// PlanFormat returns the format of the plans a request naming format gets.
func (e Explain) PlanFormat(format v1pb.QueryOption_ExplainFormat) v1pb.QueryOption_ExplainFormat {
	if format == v1pb.QueryOption_EXPLAIN_FORMAT_UNSPECIFIED {
		return e.DefaultFormat
	}
	return format
}

var (
	explainsMu sync.RWMutex
	explains   = make(map[storepb.Engine]Explain)
)

// RegisterExplain makes engine explainable.
func RegisterExplain(engine storepb.Engine, explain Explain) {
	explainsMu.Lock()
	defer explainsMu.Unlock()
	if _, dup := explains[engine]; dup {
		panic(fmt.Sprintf("db: RegisterExplain called twice for %s", engine))
	}
	explains[engine] = explain
}

// GetExplain returns how engine explains, and false when it cannot.
func GetExplain(engine storepb.Engine) (Explain, bool) {
	explainsMu.RLock()
	defer explainsMu.RUnlock()
	explain, ok := explains[engine]
	return explain, ok
}

// ExplainStatement returns the statement a driver runs for statement's plan.
// The Query handler validates the same statement before the driver runs it.
func ExplainStatement(engine storepb.Engine, statement string, format v1pb.QueryOption_ExplainFormat) (string, error) {
	explain, ok := GetExplain(engine)
	if !ok || explain.Statement == nil {
		return "", errors.Errorf("%s does not explain by running a statement", engine)
	}
	return explain.Statement(statement, format)
}
