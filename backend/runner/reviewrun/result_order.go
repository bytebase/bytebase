package reviewrun

import (
	"cmp"
	"math"
	"slices"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/store"
)

// sortReviewResults puts a run's results in reading order: by the plan's spec
// order, then by the line each result is anchored on, the highest priority
// first on one line. A result without an anchor on one of the specs goes
// last. The sort is stable, so results tied on all three keep the order the
// reviewer gave. A run's results post in this order, and the activity
// timeline shows them in the order they posted.
func sortReviewResults(specs []*storepb.PlanConfig_Spec, results []*store.IssueCommentMessage) {
	specIndex := make(map[string]int, len(specs))
	for i, spec := range specs {
		specIndex[spec.GetId()] = i
	}
	type place struct{ spec, line, priority int }
	placeOf := func(result *store.IssueCommentMessage) place {
		p := place{spec: len(specs), priority: priorityRank(result.Payload.GetReviewMetadata().GetPriority())}
		anchor := result.Payload.GetStatementAnchor()
		if i, ok := specIndex[anchor.GetSpecId()]; ok {
			p.spec = i
			p.line = int(anchor.GetStartPosition().GetLine())
		}
		return p
	}
	slices.SortStableFunc(results, func(x, y *store.IssueCommentMessage) int {
		a, b := placeOf(x), placeOf(y)
		return cmp.Or(cmp.Compare(a.spec, b.spec), cmp.Compare(a.line, b.line), cmp.Compare(a.priority, b.priority))
	})
}

// priorityRank orders P0 first and a result with no priority last. The enum
// numbers P0, P1, and P2 in that order.
func priorityRank(priority storepb.IssueCommentPayload_ReviewMetadata_Priority) int {
	if priority == storepb.IssueCommentPayload_ReviewMetadata_PRIORITY_UNSPECIFIED {
		return math.MaxInt
	}
	return int(priority)
}
