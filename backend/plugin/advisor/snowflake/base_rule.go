package snowflake

import (
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
)

// BaseRule provides the common advice-accumulation plumbing for the omni-AST
// review rules. Rules embed it to get SetBaseLine/AddAdvice/GetAdviceList.
// (It is the ANTLR-free survivor of the legacy GenericChecker walker, which
// was deleted with the ANTLR parser cutover.)
type BaseRule struct {
	level      storepb.Advice_Status
	title      string
	adviceList []*storepb.Advice
	baseLine   int
}

// SetBaseLine sets the base line for the rule.
func (r *BaseRule) SetBaseLine(baseLine int) {
	r.baseLine = baseLine
}

// GetAdviceList returns the accumulated advice.
func (r *BaseRule) GetAdviceList() []*storepb.Advice {
	return r.adviceList
}

// AddAdvice adds a new advice to the list.
func (r *BaseRule) AddAdvice(advice *storepb.Advice) {
	r.adviceList = append(r.adviceList, advice)
}
