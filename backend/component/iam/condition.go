package iam

import (
	"github.com/google/cel-go/cel"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
)

// conditionScopesResources reports whether a binding condition constrains
// anything beyond expiry -- a database, schema, table, or environment scope.
// The generic permission check binds request.time and lets the rest ride, so
// callers that must not honor a data-slice grant use this to drop the binding
// instead.
//
// The check reads the type-checker's reference map rather than walking the
// syntax tree, so it sees through macros and comprehensions, and it asks
// whether any bound variable other than request.time is referenced rather
// than naming today's resource attributes -- a new attribute is treated as
// scoping from the day it is declared.
func conditionScopesResources(expression string) (bool, error) {
	if expression == "" {
		return false, nil
	}
	e, err := cel.NewEnv(common.IAMPolicyConditionCELAttributes...)
	if err != nil {
		return false, errors.Wrapf(err, "failed to new cel env")
	}
	ast, issues := e.Compile(expression)
	if issues != nil && issues.Err() != nil {
		return false, errors.Wrapf(issues.Err(), "failed to compile expr %q", expression)
	}
	checked, err := cel.AstToCheckedExpr(ast)
	if err != nil {
		return false, errors.Wrapf(err, "failed to check expr %q", expression)
	}
	for _, reference := range checked.GetReferenceMap() {
		// Functions and their overloads are not attributes.
		if len(reference.GetOverloadId()) > 0 {
			continue
		}
		if reference.GetName() == common.CELAttributeRequestTime {
			continue
		}
		return true, nil
	}
	return false, nil
}
