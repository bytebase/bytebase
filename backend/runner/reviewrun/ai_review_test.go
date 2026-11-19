package reviewrun

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	metadatapb "github.com/bytebase/omni/metadata"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	"golang.org/x/sync/semaphore"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/aireview"
	_ "github.com/bytebase/bytebase/backend/plugin/parser/pg"
	"github.com/bytebase/bytebase/backend/runner/plancheck"
	"github.com/bytebase/bytebase/backend/store"
)

func TestAIReviewTarget(t *testing.T) {
	t.Parallel()

	instance := &store.InstanceMessage{Metadata: &storepb.Instance{Engine: storepb.Engine_POSTGRES, Version: "16.2"}}
	prod := "prod"
	schema := &metadatapb.DatabaseSchemaMetadata{Schemas: []*metadatapb.SchemaMetadata{
		{
			Name:   "public",
			Tables: []*metadatapb.TableMetadata{{Name: "orders"}, {Name: "users"}},
			Views:  []*metadatapb.ViewMetadata{{Name: "orders_summary"}},
		},
		{Name: "audit", Sequences: []*metadatapb.SequenceMetadata{{Name: "audit_id_seq"}}},
	}}
	require.Equal(t, aireview.Target{
		Engine:      "POSTGRES",
		Version:     "16.2",
		Environment: "prod",
		Schemas:     []aireview.SchemaSummary{{Name: "public", ObjectCount: 3}, {Name: "audit", ObjectCount: 1}},
	}, aiReviewTarget(instance, &store.DatabaseMessage{EffectiveEnvironmentID: &prod}, schema))

	require.Equal(t, aireview.Target{Engine: "POSTGRES", Version: "16.2"}, aiReviewTarget(instance, &store.DatabaseMessage{}, &metadatapb.DatabaseSchemaMetadata{}))

	defaults := &metadatapb.DatabaseSchemaMetadata{CharacterSet: "utf8mb4", Collation: "utf8mb4_bin", SearchPath: "app, public", Owner: "app_owner"}
	require.Equal(t, aireview.Target{Engine: "POSTGRES", Version: "16.2", CharacterSet: "utf8mb4", Collation: "utf8mb4_bin", SearchPath: "app, public", Owner: "app_owner"}, aiReviewTarget(instance, &store.DatabaseMessage{}, defaults))
}

func TestAIReviewComments(t *testing.T) {
	t.Parallel()

	unit := func(target string) *aiReviewUnit {
		return &aiReviewUnit{Check: &plancheck.CheckTarget{SpecID: "spec-1", SheetSha256: "sha", Target: target}}
	}
	drop := aireview.Finding{
		Title:    "Keep legacy_status while orders_summary reads it",
		Severity: aireview.SeverityP1,
		Line:     4,
		Rule:     "No column is dropped while a view reads it.",
		Evidence: "orders_summary selects legacy_status.",
		Fix:      "Drop the view first.",
	}
	dropOnTheSecondDatabase := drop
	dropOnTheSecondDatabase.Title = "Drop orders_summary and orders_report before legacy_status"
	dropOnTheSecondDatabase.Severity = aireview.SeverityP0
	dropOnTheSecondDatabase.Evidence = "orders_summary and orders_report select legacy_status."
	dropOnTheSecondDatabase.Fix = "Drop both views first."
	dropOnTheThirdDatabase := drop
	dropOnTheThirdDatabase.Line = 5
	dropOnTheThirdDatabase.Evidence = "orders_summary on staging selects legacy_status."
	index := aireview.Finding{
		Title:    "Create the index CONCURRENTLY",
		Severity: aireview.SeverityP2,
		Line:     2,
		Rule:     "An index on a table over 1 million rows is created CONCURRENTLY.",
		Evidence: "orders holds 52,000,000 rows.",
		Fix:      "CREATE INDEX CONCURRENTLY idx_orders_created_at ON orders (created_at);",
	}
	reviewed := []string{"instances/prod-1/databases/db", "instances/prod-2/databases/db", "instances/staging/databases/app", "instances/staging/databases/billing"}
	dropProblem := func() *aiReviewProblem {
		return &aiReviewProblem{
			Findings: []*aiReviewFinding{
				{Unit: unit("instances/prod-2/databases/db"), Finding: drop},
				{Unit: unit("instances/staging/databases/app"), Finding: dropOnTheThirdDatabase},
				{Unit: unit("instances/prod-1/databases/db"), Finding: dropOnTheSecondDatabase},
			},
			Reviewed: reviewed,
		}
	}
	comments := aiReviewComments("p", 7, []*aiReviewProblem{
		dropProblem(),
		{Findings: []*aiReviewFinding{{Unit: unit("instances/prod-1/databases/db"), Finding: index}}, Reviewed: reviewed},
	})
	require.Len(t, comments, 2)

	merged := comments[0]
	require.Equal(t, "p", merged.ProjectID)
	require.Equal(t, int64(7), merged.IssueUID)
	require.Equal(t, "prod-1/db: orders_summary and orders_report select legacy_status.\n"+
		"\n- app, prod-1/db, prod-2/db"+
		"\n- Not flagged on billing"+
		"\n\nRule: No column is dropped while a view reads it."+
		"\n\nSuggested fix: Drop both views first.", merged.Payload.Comment, "without a summary, the P0 finding leads")
	anchor := merged.Payload.StatementAnchor
	require.Equal(t, "spec-1", anchor.SpecId)
	require.Equal(t, "sha", anchor.SheetSha256)
	require.Equal(t, int32(4), anchor.StartPosition.Line, "the line most of the findings start on")
	require.Equal(t, int32(4), anchor.EndPosition.Line)
	metadata := merged.Payload.ReviewMetadata
	require.Equal(t, storepb.ReviewRuleType_REVIEW_RULE_TYPE_UNSPECIFIED, metadata.RuleType, "a result of the AI review names no rule")
	require.Equal(t, storepb.IssueCommentPayload_ReviewMetadata_P0, metadata.Priority)
	require.Equal(t, []string{"instances/prod-1/databases/db", "instances/prod-2/databases/db", "instances/staging/databases/app"}, metadata.Targets)
	require.Equal(t, "Drop orders_summary and orders_report before legacy_status", metadata.Title)

	single := comments[1]
	require.Equal(t, "Create the index CONCURRENTLY", single.Payload.ReviewMetadata.Title)
	require.Equal(t, "orders holds 52,000,000 rows.\n\nRule: An index on a table over 1 million rows is created CONCURRENTLY.\n\nSuggested fix: CREATE INDEX CONCURRENTLY idx_orders_created_at ON orders (created_at);", single.Payload.Comment)
	require.Equal(t, int32(2), single.Payload.StatementAnchor.StartPosition.Line)
	require.Equal(t, storepb.IssueCommentPayload_ReviewMetadata_P2, single.Payload.ReviewMetadata.Priority)
	require.Equal(t, []string{"instances/prod-1/databases/db"}, single.Payload.ReviewMetadata.Targets)

	require.Empty(t, aiReviewComments("p", 7, nil))

	summarized := dropProblem()
	summarized.Summary = &aireview.Summary{
		Title:    "Drop the views that read legacy_status first",
		Evidence: "Views select legacy_status, so the DROP fails.",
		Groups: []aireview.SummaryGroup{
			{Databases: []string{"instances/prod-2/databases/db", "instances/staging/databases/app"}, Facts: "orders_summary reads it."},
			{Databases: []string{"instances/prod-1/databases/db"}, Facts: "orders_summary and orders_report read it."},
		},
		Fix: "Drop each view that reads legacy_status first.",
	}
	comment := aiReviewComments("p", 7, []*aiReviewProblem{summarized})[0]
	require.Equal(t, "Drop the views that read legacy_status first", comment.Payload.ReviewMetadata.Title)
	require.Equal(t, "Views select legacy_status, so the DROP fails.\n"+
		"\n- prod-1/db: orders_summary and orders_report read it."+
		"\n- app, prod-2/db: orders_summary reads it."+
		"\n- Not flagged on billing"+
		"\n\nRule: No column is dropped while a view reads it."+
		"\n\nSuggested fix: Drop each view that reads legacy_status first.", comment.Payload.Comment, "the most severe group comes first")
	require.Equal(t, storepb.IssueCommentPayload_ReviewMetadata_P0, comment.Payload.ReviewMetadata.Priority)
	require.Equal(t, int32(4), comment.Payload.StatementAnchor.StartPosition.Line)

	summarized.Summary = &aireview.Summary{
		Title:    "t",
		Evidence: "e",
		Groups:   []aireview.SummaryGroup{{Databases: []string{"instances/prod-1/databases/db", "instances/prod-2/databases/db", "instances/staging/databases/app"}}},
		Fix:      "f",
	}
	summarized.Reviewed = reviewed[:3]
	comment = aiReviewComments("p", 7, []*aiReviewProblem{summarized})[0]
	require.Equal(t, "e\n\n- app, prod-1/db, prod-2/db\n\nRule: No column is dropped while a view reads it.\n\nSuggested fix: f", comment.Payload.Comment, "every database flagged the problem")

	// billing reports a different problem on the drop's statement.
	other := drop
	other.Title = "Archive legacy_status first"
	withOther := func(blockIsStatement bool) string {
		problem := dropProblem()
		problem.BlockIsStatement = blockIsStatement
		otherProblem := &aiReviewProblem{Findings: []*aiReviewFinding{{Unit: unit("instances/staging/databases/billing"), Finding: other}}, Reviewed: reviewed, BlockIsStatement: blockIsStatement}
		return aiReviewComments("p", 7, []*aiReviewProblem{problem, otherProblem})[0].Payload.Comment
	}
	require.Contains(t, withOther(true), "\n- app, prod-1/db, prod-2/db\n- Flagged differently on billing\n\nRule:")
	require.NotContains(t, withOther(true), "Not flagged")
	require.Contains(t, withOther(false), "\n- app, prod-1/db, prod-2/db\n\nRule:", "on a sheet that could not be split, the other finding can be on any statement")

	problem := dropProblem()
	problem.BlockIsStatement = true
	held := &aiReviewProblem{Findings: []*aiReviewFinding{{Unit: unit("instances/staging/databases/billing"), Finding: other}}, Reviewed: reviewed, BlockIsStatement: true, Held: true}
	comments = aiReviewComments("p", 7, []*aiReviewProblem{problem, held})
	require.Len(t, comments, 1, "a held problem posts no comment")
	require.Contains(t, comments[0].Payload.Comment, "\n- Flagged differently on billing\n", "a database whose finding is held still flagged the statement")
}

func TestDatabaseList(t *testing.T) {
	t.Parallel()

	var targets []string
	labels := make(map[string]string)
	for i := 12; i >= 1; i-- {
		target := fmt.Sprintf("instances/i/databases/t%02d", i)
		targets = append(targets, target)
		labels[target] = fmt.Sprintf("t%02d", i)
	}
	require.Equal(t, "t01, t02, t03, t04, t05, t06, t07, t08, t09, t10, and 2 more", databaseList(targets, labels))
	require.Equal(t, "t11, t12", databaseList(targets[:2], labels))
}

func TestAIReviewCommentsBreakTies(t *testing.T) {
	t.Parallel()

	finding := func(target string, line int) *aiReviewFinding {
		return &aiReviewFinding{
			Unit:    &aiReviewUnit{Check: &plancheck.CheckTarget{SpecID: "spec-1", Target: target}},
			Finding: aireview.Finding{Title: target, Severity: aireview.SeverityP1, Line: line, Evidence: "e", Rule: "r", Fix: "f"},
		}
	}
	comments := aiReviewComments("p", 7, []*aiReviewProblem{{Findings: []*aiReviewFinding{
		finding("instances/i/databases/c", 7),
		finding("instances/i/databases/a", 9),
		finding("instances/i/databases/b", 7),
		finding("instances/i/databases/d", 9),
	}}})
	require.Len(t, comments, 1)
	require.Equal(t, int32(7), comments[0].Payload.StatementAnchor.StartPosition.Line, "the first line on a tie")
	require.Equal(t, "instances/i/databases/a", comments[0].Payload.ReviewMetadata.Title, "the first database leads on a tie")
	require.True(t, strings.HasPrefix(comments[0].Payload.Comment, "a: e\n\n- a, b, c, d\n"))
}

// mergeStatement has two statements: a drop on lines 1 and 2, and an index
// on line 3.
const mergeStatement = "ALTER TABLE orders\n  DROP COLUMN legacy_status;\nCREATE INDEX idx ON orders (created_at);\n"

func mergeResult(target string, engine storepb.Engine, findings ...aireview.Finding) *aiReviewResult {
	return &aiReviewResult{
		Unit: &aiReviewUnit{
			Check:     &plancheck.CheckTarget{SpecID: "spec-1", SheetSha256: "sha", Target: target},
			Statement: mergeStatement,
			Engine:    engine,
		},
		Findings: findings,
	}
}

func titles(problems []*aiReviewProblem) [][]string {
	var result [][]string
	for _, problem := range problems {
		var problemTitles []string
		for _, finding := range problem.Findings {
			problemTitles = append(problemTitles, finding.Finding.Title)
		}
		result = append(result, problemTitles)
	}
	return result
}

// recordingMerge records the merge requests and answers with groups, or
// with err.
type recordingMerge struct {
	requests []*aireview.MergeRequest
	groups   [][]int
	same     []int
	err      error
}

func (m *recordingMerge) merge(_ context.Context, request *aireview.MergeRequest) (*aireview.MergeResult, error) {
	m.requests = append(m.requests, request)
	if m.err != nil {
		return nil, m.err
	}
	return &aireview.MergeResult{Groups: m.groups, Same: m.same}, nil
}

// dropRule is the standard rules' result on the drop of mergeStatement, which
// starts on line 1 and names the column on line 2. It applies to db1.
func dropRule() *ruleResult {
	return &ruleResult{specID: "spec-1", sheetSha256: "sha", members: []*ruleFinding{{
		rule:        storepb.ReviewRuleType_DISALLOW_DROP_OBJECT,
		targets:     []string{"instances/prod/databases/db1"},
		codes:       []string{"DO101"},
		message:     "Drops column `legacy_status`.",
		fingerprint: "drop",
		anchor: &storepb.IssueCommentPayload_StatementAnchor{
			SpecId:        "spec-1",
			SheetSha256:   "sha",
			StartPosition: &storepb.Position{Line: 2, Column: 3},
			EndPosition:   &storepb.Position{Line: 2, Column: 28},
		},
	}}}
}

const (
	// dropRuleSentence is how a finding cites the rule of dropRule.
	dropRuleSentence = "DISALLOW_DROP_OBJECT: Dropping an object, such as a table, column, view, index, sequence, schema, function, user, or role, loses the data, definition, or access it held."
	breakageSentence = "A view, function, trigger, or foreign key depends on what it changes."
)

func TestMergeAIReviewFindingsPutsProblemsOnRuleFindings(t *testing.T) {
	t.Parallel()

	finding := func(title string, severity aireview.Severity, line int, rule string) aireview.Finding {
		return aireview.Finding{Title: title, Severity: severity, Line: line, Rule: rule}
	}
	view := finding("view", aireview.SeverityP0, 1, breakageSentence)
	index := finding("index", aireview.SeverityP1, 3, "r")
	// One database, so only a rule finding gives the merge a second source.
	results := func(findings ...aireview.Finding) []*aiReviewResult {
		return []*aiReviewResult{mergeResult("instances/prod/databases/db1", storepb.Engine_POSTGRES, findings...)}
	}
	standard := aireview.MergeCandidate{Block: 0, Standard: true, Finding: aireview.Finding{
		Title: "DISALLOW_DROP_OBJECT (DO101)", Severity: aireview.SeverityP1, Line: 2, Evidence: "Drops column `legacy_status`.",
	}}

	t.Run("puts a problem with the rule finding the merge groups it with", func(t *testing.T) {
		t.Parallel()

		rule := dropRule()
		merge := &recordingMerge{groups: [][]int{{0, 1}}}
		problems, err := mergeAIReviewFindings(context.Background(), merge.merge, results(view, index), nil, []*ruleResult{rule})
		require.NoError(t, err)
		require.Equal(t, [][]string{{"view"}, {"index"}}, titles(problems))
		require.Equal(t, &ruleMember{result: rule, finding: rule.members[0]}, problems[0].Rule)
		require.Nil(t, problems[1].Rule)
		require.Len(t, merge.requests, 1)
		require.Equal(t, []aireview.MergeCandidate{
			{Block: 0, Database: "instances/prod/databases/db1", Finding: view},
			standard,
		}, merge.requests[0].Candidates, "the statement of the index has one finding and no rule finding, so it is not offered")
		require.Empty(t, merge.requests[0].Threads)
	})

	t.Run("leaves a problem the merge keeps apart", func(t *testing.T) {
		t.Parallel()

		merge := &recordingMerge{groups: [][]int{{0}}}
		problems, err := mergeAIReviewFindings(context.Background(), merge.merge, results(view), nil, []*ruleResult{dropRule()})
		require.NoError(t, err)
		require.Nil(t, problems[0].Rule)
	})

	t.Run("leaves a problem of a database the rule finding is not on", func(t *testing.T) {
		t.Parallel()

		// db2 reports the view too, and the rule finding is on db1 alone.
		both := append(results(view), mergeResult("instances/prod/databases/db2", storepb.Engine_POSTGRES, view))
		merge := &recordingMerge{groups: [][]int{{0, 1, 2}}}
		problems, err := mergeAIReviewFindings(context.Background(), merge.merge, both, nil, []*ruleResult{dropRule()})
		require.NoError(t, err)
		require.Len(t, problems, 1)
		require.Nil(t, problems[0].Rule, "the result names db1 alone, so the problem posts on its own")

		everywhere := dropRule()
		everywhere.members[0].targets = []string{"instances/prod/databases/db1", "instances/prod/databases/db2"}
		problems, err = mergeAIReviewFindings(context.Background(), merge.merge, both, nil, []*ruleResult{everywhere})
		require.NoError(t, err)
		require.NotNil(t, problems[0].Rule)
	})

	t.Run("offers no rule finding alone on its statement", func(t *testing.T) {
		t.Parallel()

		merge := &recordingMerge{}
		other := dropRule()
		other.specID = "spec-2"
		whole := dropRule()
		whole.members[0].anchor = nil
		problems, err := mergeAIReviewFindings(context.Background(), merge.merge, results(index), nil, []*ruleResult{dropRule(), other, whole})
		require.NoError(t, err)
		require.Equal(t, [][]string{{"index"}}, titles(problems))
		require.Empty(t, merge.requests, "nor one on another spec, or on the whole change")
	})

	t.Run("a problem on a rule finding keeps no earlier result", func(t *testing.T) {
		t.Parallel()

		segments := sheetStatements(context.Background(), storepb.Engine_POSTGRES, "spec-1", mergeStatement).Segments
		earlier := map[aiReviewSheet][]*aiReviewEarlier{{specID: "spec-1", sheetSha256: "sha"}: {{
			Fingerprint: segments[0] + "/posted", Priority: storepb.IssueCommentPayload_ReviewMetadata_P0, Title: "view", Sheet: mergeStatement, SheetSha256: "sha", Line: 1,
		}}}
		merge := &recordingMerge{groups: [][]int{{0, 1, 2}}}
		problems, err := mergeAIReviewFindings(context.Background(), merge.merge, results(view), earlier, []*ruleResult{dropRule()})
		require.NoError(t, err)
		require.NotNil(t, problems[0].Rule)
		require.Nil(t, problems[0].Earlier, "its text joins the rule result, so the earlier thread is superseded")

		merge = &recordingMerge{groups: [][]int{{0, 1}}}
		problems, err = mergeAIReviewFindings(context.Background(), merge.merge, results(view), earlier, []*ruleResult{dropRule()})
		require.NoError(t, err)
		require.Nil(t, problems[0].Rule)
		require.NotNil(t, problems[0].Earlier)
	})

	t.Run("posts a problem on its own when the merge fails or groups two rule findings", func(t *testing.T) {
		t.Parallel()

		merge := &recordingMerge{err: errors.New("provider down")}
		problems, err := mergeAIReviewFindings(context.Background(), merge.merge, results(view), nil, []*ruleResult{dropRule()})
		require.NoError(t, err)
		require.Equal(t, [][]string{{"view"}}, titles(problems))
		require.Nil(t, problems[0].Rule)

		merge = &recordingMerge{groups: [][]int{{0, 1, 2}}}
		problems, err = mergeAIReviewFindings(context.Background(), merge.merge, results(view), nil, []*ruleResult{dropRule(), dropRule()})
		require.NoError(t, err)
		require.Nil(t, problems[0].Rule)
	})

	t.Run("keeps the comment an earlier run left when the merge says it is the same", func(t *testing.T) {
		t.Parallel()

		p0, p1 := storepb.IssueCommentPayload_ReviewMetadata_P0, storepb.IssueCommentPayload_ReviewMetadata_P1
		unchanged := func(earlier *ruleThread, same []int, findings ...aireview.Finding) (bool, *recordingMerge) {
			t.Helper()
			rule := dropRule()
			rule.earlier = earlier
			merge := &recordingMerge{groups: [][]int{{0, 1}}, same: same}
			problems, err := mergeAIReviewFindings(context.Background(), merge.merge, results(findings...), nil, []*ruleResult{rule})
			require.NoError(t, err)
			require.NotNil(t, problems[0].Rule)
			return problems[0].Unchanged, merge
		}
		const added = "Drops column `legacy_status`.\n\nv_orders reads it.\n\nSuggested fix: Drop v_orders first."

		kept, merge := unchanged(&ruleThread{Comment: added, Priority: p0}, []int{0}, view)
		require.True(t, kept)
		require.Equal(t, []aireview.MergeThread{{Comment: added, Candidates: []int{1}}}, merge.requests[0].Threads)

		kept, _ = unchanged(&ruleThread{Comment: added, Priority: p0}, nil, view)
		require.False(t, kept, "the merge did not say so")

		kept, merge = unchanged(&ruleThread{Comment: "Drops column `legacy_status`.", Priority: p1}, nil, view)
		require.False(t, kept)
		require.Empty(t, merge.requests[0].Threads, "a comment that holds the rules' text alone has nothing to keep")

		kept, merge = unchanged(nil, nil, view)
		require.False(t, kept)
		require.Empty(t, merge.requests[0].Threads)

		kept, _ = unchanged(&ruleThread{Comment: added, Priority: p1}, []int{0}, view)
		require.False(t, kept, "nobody read the comment at P0, so it is written again")

		restated := finding("drop", aireview.SeverityP0, 2, dropRuleSentence)
		kept, _ = unchanged(&ruleThread{Comment: added, Priority: p1}, []int{0}, restated)
		require.True(t, kept, "a problem that restates the rule's condition does not raise the rule's priority")

		const reworded = "Drops the column `legacy_status`.\n\nv_orders reads it.\n\nSuggested fix: Drop v_orders first."
		kept, merge = unchanged(&ruleThread{Comment: reworded, Priority: p0}, []int{0}, view)
		require.False(t, kept, "the comment lacks the rule's text as it is now, so it is written again")
		require.Empty(t, merge.requests[0].Threads)

		// The rule finding is on db1 and db2, and this run's problem on db1
		// alone: the comment was written for both, so it cannot stay.
		everywhere := dropRule()
		everywhere.members[0].targets = []string{"instances/prod/databases/db1", "instances/prod/databases/db2"}
		everywhere.earlier = &ruleThread{Comment: added, Priority: p0}
		partial := &recordingMerge{groups: [][]int{{0, 1}}, same: []int{0}}
		problems, err := mergeAIReviewFindings(context.Background(), partial.merge, results(view), nil, []*ruleResult{everywhere})
		require.NoError(t, err)
		require.NotNil(t, problems[0].Rule)
		require.False(t, problems[0].Unchanged, "the problem is on part of the rule finding's databases now")

		// A comment of facts that names no database states them for every
		// database, so it can stay whichever ones the problem is on.
		const facts = "Drops column `legacy_status`.\n\nv_orders reads it."
		low := finding("view", aireview.SeverityP1, 1, breakageSentence)
		everywhere.earlier, everywhere.keepsEarlier = &ruleThread{Comment: facts, Priority: p1}, false
		stated := &recordingMerge{groups: [][]int{{0, 1}}, same: []int{0}}
		problems, err = mergeAIReviewFindings(context.Background(), stated.merge, results(low), nil, []*ruleResult{everywhere})
		require.NoError(t, err)
		require.Equal(t, []aireview.MergeThread{{Comment: facts, Candidates: []int{1}}}, stated.requests[0].Threads)
		require.True(t, problems[0].Unchanged, "the facts already state the problem, and it is no higher than the rule")

		// Facts that name a database are about that database, and the merge
		// cannot tell whether they state a problem on another one.
		for _, scoped := range []string{"db2: v_orders reads it.", "db1, db2: v_orders reads it.", "Views differ.\n\n- db2: v_orders reads it.\n- db1"} {
			everywhere.earlier, everywhere.keepsEarlier = &ruleThread{Comment: "Drops column `legacy_status`.\n\n" + scoped, Priority: p1}, false
			named := &recordingMerge{groups: [][]int{{0, 1}}, same: []int{0}}
			problems, err = mergeAIReviewFindings(context.Background(), named.merge, results(low), nil, []*ruleResult{everywhere})
			require.NoError(t, err)
			require.Empty(t, named.requests[0].Threads, scoped)
			require.False(t, problems[0].Unchanged, scoped)
		}

		kept, _ = unchanged(&ruleThread{Comment: facts, Priority: p1}, nil, low)
		require.False(t, kept, "the merge did not say the facts state it")
		kept, _ = unchanged(&ruleThread{Comment: facts, Priority: p1}, []int{0}, view)
		require.False(t, kept, "nobody read the facts at P0, so the problem is written")
	})
}

func TestCanKeepEarlier(t *testing.T) {
	t.Parallel()

	const rules = "Drops column `legacy_status`."
	const added = rules + "\n\nv_orders reads it.\n\n- db1, db2: orders holds 5k to 9k rows.\n\nSuggested fix: Drop v_orders first."
	keeps := func(comment string) bool {
		rule := dropRule()
		rule.members[0].targets = []string{"instances/prod/databases/db1", "instances/prod/databases/db2"}
		rule.earlier = &ruleThread{Comment: comment}
		return canKeepEarlier(rule)
	}
	require.True(t, keeps(added), "a comment for every database of the result, whatever facts it lists them with")
	require.False(t, keeps(rules), "the rules' text alone holds nothing to keep")
	require.False(t, keeps(strings.Replace(added, "Drops column", "Drops the column", 1)), "the rules' text changed")
	require.False(t, keeps(rules+"\n\ndb2: v_orders reads it.\n\nSuggested fix: Drop v_orders first."), "written for one database")
	require.False(t, keeps(rules+"\n\nprod/db2: v_orders reads it."), "one database, named with its instance")
	require.False(t, keeps(rules+"\n\nv_orders reads it.\n\n- db1\n- Not flagged on db2\n\nSuggested fix: Drop v_orders first."), "written for part of the databases")
	require.False(t, keeps(rules+"\n\nv_orders reads it.\n\n- db1\n- Flagged differently on db2"))
	require.True(t, keeps(rules+"\n\n  v_orders reads it."))
	require.False(t, keeps(rules+"\n\n  v_orders reads it.\n\n  - db1\n  - Not flagged on db2"), "indented under the item of a merged result too")

	none := dropRule()
	require.False(t, canKeepEarlier(none), "no earlier thread")
}

func TestHoldsLines(t *testing.T) {
	t.Parallel()

	const rules = "3 statements, on lines 1 to 3.\n\n- Line 1: Drops table `a`.\n- Line 2: Drops table `b`.\n- Line 3: Drops table `c`."
	const posted = "3 statements, on lines 1 to 3.\n\n- Line 1: Drops table `a`.\n- Line 2: Drops table `b`.\n\n  v reads b.\n\n  Suggested fix: Drop v first.\n- Line 3: Drops table `c`."
	require.True(t, holdsLines(posted, rules), "what the AI review added sits between the rules' lines")
	require.True(t, holdsLines(rules, rules))
	moved := strings.NewReplacer("lines 1 to 3", "lines 2 to 4", "Line 1", "Line 2", "Line 2", "Line 3", "Line 3", "Line 4").Replace(rules)
	require.False(t, holdsLines(posted, moved), "the statements moved, so the posted line numbers are stale")
	require.False(t, holdsLines("- Line 2: Drops table `b`.\n- Line 1: Drops table `a`.", "- Line 1: Drops table `a`.\n- Line 2: Drops table `b`."), "in order")
}

func TestFoldAIReviewProblems(t *testing.T) {
	t.Parallel()

	p0, p1 := storepb.IssueCommentPayload_ReviewMetadata_P0, storepb.IssueCommentPayload_ReviewMetadata_P1
	// problemOn is a problem db1 reports on a rule finding, in a review under
	// the policies.
	problemOn := func(member *ruleMember, finding aireview.Finding, policies ...string) *aiReviewProblem {
		unit := &aiReviewUnit{Check: &plancheck.CheckTarget{SpecID: "spec-1", SheetSha256: "sha", Target: "instances/prod/databases/db1"}, Statement: mergeStatement, Policies: policies}
		return &aiReviewProblem{Findings: []*aiReviewFinding{{Unit: unit, Finding: finding}}, Block: aireview.LineRange{Start: 1, End: 2}, BlockIsStatement: true, Rule: member, Segment: "segment"}
	}
	restated := aireview.Finding{Title: "Dropping legacy_status loses its data", Severity: aireview.SeverityP0, Line: 2, Rule: dropRuleSentence, Evidence: "orders.legacy_status holds 12,000 values.", Fix: "Archive the column first."}
	view := aireview.Finding{Title: "DROP COLUMN legacy_status fails: v_orders reads it", Severity: aireview.SeverityP0, Line: 2, Rule: breakageSentence, Evidence: "v_orders reads legacy_status.", Fix: "Drop v_orders first."}
	index := aireview.Finding{Title: "index", Severity: aireview.SeverityP1, Line: 3, Rule: "r", Evidence: "e", Fix: "f"}

	t.Run("adds a problem that restates the rule as its evidence and fix", func(t *testing.T) {
		t.Parallel()

		rule := dropRule()
		problems := []*aiReviewProblem{problemOn(&ruleMember{result: rule, finding: rule.members[0]}, restated), problemOn(nil, index)}
		foldAIReviewProblems(problems)
		payload := rule.payload()
		require.Equal(t, "Drops column `legacy_status`.\n\norders.legacy_status holds 12,000 values.\n\nSuggested fix: Archive the column first.", payload.Comment)
		require.Equal(t, p1, payload.ReviewMetadata.Priority, "the rule sets the priority of its own condition")
		require.Equal(t, storepb.ReviewRuleType_DISALLOW_DROP_OBJECT, payload.ReviewMetadata.RuleType)
		require.Equal(t, []string{"DO101"}, payload.ReviewMetadata.Codes)
		require.Equal(t, "drop", payload.ReviewMetadata.Fingerprint)

		comments := aiReviewComments("p", 1, problems)
		require.Len(t, comments, 1, "a problem on a rule finding posts no comment of its own")
		require.Equal(t, "index", comments[0].Payload.ReviewMetadata.Title)
	})

	t.Run("adds any other problem as plain text too, and takes its priority when higher", func(t *testing.T) {
		t.Parallel()

		rule := dropRule()
		member := &ruleMember{result: rule, finding: rule.members[0]}
		foldAIReviewProblems([]*aiReviewProblem{problemOn(member, restated), problemOn(member, view)})
		payload := rule.payload()
		require.Equal(t, "Drops column `legacy_status`.\n\n"+
			"orders.legacy_status holds 12,000 values.\n\nSuggested fix: Archive the column first.\n\n"+
			"v_orders reads legacy_status.\n\nSuggested fix: Drop v_orders first.", payload.Comment, "no title, and no sentence of the review's own prompt")
		require.Equal(t, p0, payload.ReviewMetadata.Priority)

		lower := dropRule()
		advisory := view
		advisory.Severity = aireview.SeverityP2
		foldAIReviewProblems([]*aiReviewProblem{problemOn(&ruleMember{result: lower, finding: lower.members[0]}, advisory)})
		require.Equal(t, p1, lower.payload().ReviewMetadata.Priority, "a lower priority leaves the rule's")
	})

	t.Run("cites the rule of a problem only when the review's prompt does not hold it", func(t *testing.T) {
		t.Parallel()

		// folded returns what a problem that cites rule adds to a rule finding,
		// in a review with the policies.
		folded := func(rule string, policies ...string) string {
			result := dropRule()
			finding := view
			finding.Rule = rule
			foldAIReviewProblems([]*aiReviewProblem{problemOn(&ruleMember{result: result, finding: result.members[0]}, finding, policies...)})
			return strings.TrimPrefix(result.payload().Comment, "Drops column `legacy_status`.\n\n")
		}
		const evidence, fix = "v_orders reads legacy_status.", "\n\nSuggested fix: Drop v_orders first."
		const policy = "No column is dropped while a view reads it."
		// The standard rules section of the prompt gives this sentence as an
		// example of a policy sentence.
		const example = "Every UPDATE needs a WHERE clause."

		require.Equal(t, evidence+fix, folded(breakageSentence))
		require.Equal(t, evidence+fix, folded(dropRuleSentence))
		require.Equal(t, evidence+"\n\nRule: "+policy+fix, folded(policy, "", policy), "a policy sentence says why the problem is one")
		require.Equal(t, evidence+"\n\nRule: "+policy+fix, folded(policy), "a sentence the review does not know stays too")
		require.Equal(t, evidence+fix, folded(example))
		require.Equal(t, evidence+"\n\nRule: "+example+fix, folded(example, example, ""), "the policy decides over the prompt")
	})

	t.Run("takes the priority of a policy sentence that names the rule", func(t *testing.T) {
		t.Parallel()

		const policy = "A change that DISALLOW_DROP_OBJECT flags is P0 in this project."
		priorityOf := func(rule string, policies ...string) storepb.IssueCommentPayload_ReviewMetadata_Priority {
			result := dropRule()
			finding := restated
			finding.Rule = rule
			foldAIReviewProblems([]*aiReviewProblem{problemOn(&ruleMember{result: result, finding: result.members[0]}, finding, policies...)})
			return result.payload().ReviewMetadata.Priority
		}
		require.Equal(t, p1, priorityOf(dropRuleSentence, policy), "the rule itself, whatever the policy says elsewhere")
		require.Equal(t, p0, priorityOf(policy, policy), "the policy's sentence is not a restatement of the rule")
		require.Equal(t, p1, priorityOf(policy), "the same words with no such policy name the rule alone")
	})

	t.Run("takes the priority and the rule of the finding that quotes the policy, whichever leads", func(t *testing.T) {
		t.Parallel()

		const policy = "No column that holds values is dropped."
		result := dropRule()
		result.members[0].targets = []string{"instances/prod/databases/db1", "instances/prod/databases/db2"}
		// db1 leads by name and cites the rule itself, and db2 quotes the policy.
		problem := problemOn(&ruleMember{result: result, finding: result.members[0]}, restated, policy)
		quoting := restated
		quoting.Rule = policy
		problem.Findings = append(problem.Findings, &aiReviewFinding{
			Unit:    &aiReviewUnit{Check: &plancheck.CheckTarget{SpecID: "spec-1", SheetSha256: "sha", Target: "instances/prod/databases/db2"}, Statement: mergeStatement, Policies: []string{policy}},
			Finding: quoting,
		})
		foldAIReviewProblems([]*aiReviewProblem{problem})
		payload := result.payload()
		require.Equal(t, p0, payload.ReviewMetadata.Priority)
		require.Contains(t, payload.Comment, "\n\nRule: "+policy+"\n\nSuggested fix: ")
	})

	t.Run("names the databases of a problem only when they are not the rule finding's", func(t *testing.T) {
		t.Parallel()

		db1, db2, db3 := "instances/prod/databases/db1", "instances/prod/databases/db2", "instances/prod/databases/db3"
		// folded returns what a problem that targets report adds to a rule
		// finding on ruleTargets, after three databases reviewed the sheet.
		folded := func(ruleTargets []string, targets ...string) string {
			rule := dropRule()
			rule.members[0].targets = ruleTargets
			problem := &aiReviewProblem{Block: aireview.LineRange{Start: 1, End: 2}, BlockIsStatement: true, Rule: &ruleMember{result: rule, finding: rule.members[0]}, Reviewed: []string{db1, db2, db3}}
			for _, target := range targets {
				unit := &aiReviewUnit{Check: &plancheck.CheckTarget{SpecID: "spec-1", SheetSha256: "sha", Target: target}, Statement: mergeStatement}
				problem.Findings = append(problem.Findings, &aiReviewFinding{Unit: unit, Finding: restated})
			}
			foldAIReviewProblems([]*aiReviewProblem{problem})
			return strings.TrimPrefix(rule.payload().Comment, "Drops column `legacy_status`.\n\n")
		}
		const fix = "\n\nSuggested fix: Archive the column first."

		require.Equal(t, "db1: orders.legacy_status holds 12,000 values."+fix, folded([]string{db1, db2, db3}, db1, db2, db3), "the thread already names the rule finding's databases")
		require.Equal(t, "db1: orders.legacy_status holds 12,000 values.\n\n- db1, db2\n- Not flagged on db3"+fix, folded([]string{db1, db2, db3}, db1, db2))
		require.Equal(t, "db2: orders.legacy_status holds 12,000 values."+fix, folded([]string{db1, db2, db3}, db2), "one database of several is named")
		require.Equal(t, "orders.legacy_status holds 12,000 values."+fix, folded([]string{db2}, db2))
	})

	t.Run("keeps the earlier comment of a result whose problems are unchanged", func(t *testing.T) {
		t.Parallel()

		rule := dropRule()
		rule.earlier = &ruleThread{Comment: "the comment an earlier run left", Priority: p0}
		problem := problemOn(&ruleMember{result: rule, finding: rule.members[0]}, view)
		problem.Unchanged = true
		foldAIReviewProblems([]*aiReviewProblem{problem})
		payload := rule.payload()
		require.Equal(t, "the comment an earlier run left", payload.Comment)
		require.Equal(t, p0, payload.ReviewMetadata.Priority)
		require.Equal(t, "drop", payload.ReviewMetadata.Fingerprint)
	})
}

func TestMergeAIReviewFindings(t *testing.T) {
	t.Parallel()

	finding := func(title string, line int) aireview.Finding {
		return aireview.Finding{Title: title, Severity: aireview.SeverityP1, Line: line}
	}
	results := func(engine storepb.Engine) []*aiReviewResult {
		return []*aiReviewResult{
			mergeResult("instances/prod/databases/db1", engine, finding("drop on db1", 1), finding("index on db1", 3)),
			nil,
			mergeResult("instances/prod/databases/db2", engine, finding("drop on db2", 2)),
			mergeResult("instances/prod/databases/db3", engine, finding("view on db3", 1)),
			mergeResult("instances/prod/databases/db0", engine),
		}
	}

	t.Run("merges what the model groups", func(t *testing.T) {
		t.Parallel()

		merge := &recordingMerge{groups: [][]int{{0, 1}, {2}}}
		problems, err := mergeAIReviewFindings(context.Background(), merge.merge, results(storepb.Engine_POSTGRES), nil, nil)
		require.NoError(t, err)
		require.Equal(t, [][]string{{"drop on db1", "drop on db2"}, {"index on db1"}, {"view on db3"}}, titles(problems))
		require.Equal(t, aireview.LineRange{Start: 1, End: 2}, problems[0].Block)
		require.Equal(t, aireview.LineRange{Start: 3, End: 3}, problems[1].Block)
		require.Equal(t, aireview.LineRange{Start: 1, End: 2}, problems[2].Block)
		for _, problem := range problems {
			require.True(t, problem.BlockIsStatement)
			require.Equal(t, []string{"instances/prod/databases/db0", "instances/prod/databases/db1", "instances/prod/databases/db2", "instances/prod/databases/db3"}, problem.Reviewed, "a database that reported nothing was reviewed too")
		}

		require.Len(t, merge.requests, 1)
		request := merge.requests[0]
		require.Equal(t, mergeStatement, request.Statement)
		require.Equal(t, []aireview.LineRange{{Start: 1, End: 2}, {Start: 3, End: 3}}, request.Blocks)
		require.Equal(t, []aireview.MergeCandidate{
			{Block: 0, Database: "instances/prod/databases/db1", Finding: finding("drop on db1", 1)},
			{Block: 0, Database: "instances/prod/databases/db2", Finding: finding("drop on db2", 2)},
			{Block: 0, Database: "instances/prod/databases/db3", Finding: finding("view on db3", 1)},
		}, request.Candidates, "a statement that one database reported on is not offered")
	})

	t.Run("posts each finding on its own when the merge fails", func(t *testing.T) {
		t.Parallel()

		merge := &recordingMerge{err: errors.New("provider down")}
		problems, err := mergeAIReviewFindings(context.Background(), merge.merge, results(storepb.Engine_POSTGRES), nil, nil)
		require.NoError(t, err)
		require.Equal(t, [][]string{{"drop on db1"}, {"index on db1"}, {"drop on db2"}, {"view on db3"}}, titles(problems))
	})

	t.Run("stops when the run ends", func(t *testing.T) {
		t.Parallel()

		ctx, cancel := context.WithCancel(context.Background())
		merge := func(context.Context, *aireview.MergeRequest) (*aireview.MergeResult, error) {
			cancel()
			return nil, context.Canceled
		}
		_, err := mergeAIReviewFindings(ctx, merge, results(storepb.Engine_POSTGRES), nil, nil)
		require.ErrorIs(t, err, context.Canceled)
	})

	t.Run("treats the sheet as one block when the engines differ", func(t *testing.T) {
		t.Parallel()

		mixed := results(storepb.Engine_POSTGRES)
		mixed[3].Unit.Engine = storepb.Engine_MYSQL
		merge := &recordingMerge{groups: [][]int{{0, 2}, {1}, {3}}}
		problems, err := mergeAIReviewFindings(context.Background(), merge.merge, mixed, nil, nil)
		require.NoError(t, err)
		require.Equal(t, [][]string{{"drop on db1", "drop on db2"}, {"index on db1"}, {"view on db3"}}, titles(problems))
		require.Equal(t, []aireview.LineRange{{Start: 1, End: 4}}, merge.requests[0].Blocks)
		require.Len(t, merge.requests[0].Candidates, 4)
		for _, problem := range problems {
			require.False(t, problem.BlockIsStatement)
		}
	})

	t.Run("calls no model for one database", func(t *testing.T) {
		t.Parallel()

		merge := &recordingMerge{}
		problems, err := mergeAIReviewFindings(context.Background(), merge.merge, results(storepb.Engine_POSTGRES)[:1], nil, nil)
		require.NoError(t, err)
		require.Equal(t, [][]string{{"drop on db1"}, {"index on db1"}}, titles(problems))
		require.Empty(t, merge.requests)
	})
}

func TestMergeAIReviewFindingsKeepsEarlierResults(t *testing.T) {
	t.Parallel()

	finding := func(title string, severity aireview.Severity, line int) aireview.Finding {
		return aireview.Finding{Title: title, Severity: severity, Line: line}
	}
	// One database, so only the earlier results give the merge a second
	// source.
	results := func() []*aiReviewResult {
		return []*aiReviewResult{mergeResult("instances/prod/databases/db1", storepb.Engine_POSTGRES,
			finding("drop", aireview.SeverityP1, 1),
			finding("index", aireview.SeverityP1, 3),
		)}
	}
	sheet := aiReviewSheet{specID: "spec-1", sheetSha256: "sha"}
	segments := sheetStatements(context.Background(), storepb.Engine_POSTGRES, "spec-1", mergeStatement).Segments
	// earlier is a result posted on statement, of segments, at line of the
	// sheet under review.
	earlier := func(name string, statement, line int, resolved bool, priority storepb.IssueCommentPayload_ReviewMetadata_Priority) *aiReviewEarlier {
		return &aiReviewEarlier{
			Fingerprint: segments[statement] + "/" + name,
			Resolved:    resolved,
			Priority:    priority,
			Title:       name,
			Comment:     "evidence on db1\n\nRule: r\n\nSuggested fix: f",
			Sheet:       mergeStatement,
			SheetSha256: "sha",
			Line:        line,
		}
	}
	p1, p2 := storepb.IssueCommentPayload_ReviewMetadata_P1, storepb.IssueCommentPayload_ReviewMetadata_P2
	kept := func(problems []*aiReviewProblem) []string {
		var names []string
		for _, problem := range problems {
			if problem.Earlier == nil {
				names = append(names, "")
				continue
			}
			_, name, _ := strings.Cut(problem.Earlier.Fingerprint, "/")
			names = append(names, name)
		}
		return names
	}

	t.Run("keeps the earlier result the merge groups with a problem", func(t *testing.T) {
		t.Parallel()

		changed := earlier("changed", 0, 2, false, p1)
		changed.Fingerprint = sheetStatements(context.Background(), storepb.Engine_POSTGRES, "spec-1", "ALTER TABLE orders\n  DROP COLUMN status;\n").Segments[0] + "/changed"
		merge := &recordingMerge{groups: [][]int{{0, 2}, {1}, {3}}}
		problems, err := mergeAIReviewFindings(context.Background(), merge.merge, results(), map[aiReviewSheet][]*aiReviewEarlier{
			sheet: {earlier("open", 0, 2, false, p1), earlier("gone", 1, 3, true, p1), changed},
		}, nil)
		require.NoError(t, err)
		require.Equal(t, [][]string{{"drop"}, {"index"}}, titles(problems))
		require.Equal(t, []string{"open", ""}, kept(problems), "an earlier result left alone keeps nothing")
		require.Equal(t, []string{segments[0], segments[1]}, []string{problems[0].Segment, problems[1].Segment})
		require.Len(t, merge.requests, 1)
		require.Equal(t, []aireview.MergeCandidate{
			{Block: 0, Database: "instances/prod/databases/db1", Finding: finding("drop", aireview.SeverityP1, 1)},
			{Block: 1, Database: "instances/prod/databases/db1", Finding: finding("index", aireview.SeverityP1, 3)},
			{Block: 0, Earlier: true, Finding: aireview.Finding{Title: "open", Severity: aireview.SeverityP1, Line: 2, Rule: "r", Fix: "f"}},
			{Block: 1, Earlier: true, Finding: aireview.Finding{Title: "gone", Severity: aireview.SeverityP1, Line: 3, Rule: "r", Fix: "f"}},
		}, merge.requests[0].Candidates, "a result on a statement that changed is not offered, and none shows its evidence")
	})

	t.Run("moves an earlier result with its statement", func(t *testing.T) {
		t.Parallel()

		// The result was posted on line 3 of a sheet that opened with a
		// comment and a blank line.
		moved := earlier("moved", 0, 3, false, p1)
		moved.Sheet, moved.SheetSha256 = "-- archive\n\n"+mergeStatement, "before"
		merge := &recordingMerge{groups: [][]int{{0, 1}}}
		_, err := mergeAIReviewFindings(context.Background(), merge.merge, results(), map[aiReviewSheet][]*aiReviewEarlier{sheet: {moved}}, nil)
		require.NoError(t, err)
		require.Equal(t, 1, merge.requests[0].Candidates[1].Finding.Line, "the statement's first line, where its body starts now")
	})

	t.Run("keeps a result on a line outside every statement", func(t *testing.T) {
		t.Parallel()

		const gap = "SELECT 1;\n\nSELECT 2;\n"
		result := mergeResult("instances/prod/databases/db1", storepb.Engine_POSTGRES, finding("between", aireview.SeverityP2, 2))
		result.Unit.Statement = gap
		first, err := mergeAIReviewFindings(context.Background(), (&recordingMerge{}).merge, []*aiReviewResult{result}, nil, nil)
		require.NoError(t, err)
		posted := &aiReviewEarlier{Fingerprint: first[0].Segment + "/between", Priority: p2, Title: "between", Sheet: gap, SheetSha256: "sha", Line: 2}
		merge := &recordingMerge{groups: [][]int{{0, 1}}}
		problems, err := mergeAIReviewFindings(context.Background(), merge.merge, []*aiReviewResult{result}, map[aiReviewSheet][]*aiReviewEarlier{sheet: {posted}}, nil)
		require.NoError(t, err)
		require.Equal(t, []string{"between"}, kept(problems))
	})

	t.Run("tells equal lines outside every statement apart", func(t *testing.T) {
		t.Parallel()

		const gaps = "SELECT 1;\n\nSELECT 2;\n\nSELECT 3;\n"
		run := func(earlierResults []*aiReviewEarlier, merge *recordingMerge, findings ...aireview.Finding) []*aiReviewProblem {
			t.Helper()
			result := mergeResult("instances/prod/databases/db1", storepb.Engine_POSTGRES, findings...)
			result.Unit.Statement = gaps
			problems, err := mergeAIReviewFindings(context.Background(), merge.merge, []*aiReviewResult{result}, map[aiReviewSheet][]*aiReviewEarlier{sheet: earlierResults}, nil)
			require.NoError(t, err)
			return problems
		}
		both := run(nil, &recordingMerge{}, finding("second line", aireview.SeverityP2, 2), finding("fourth line", aireview.SeverityP2, 4))
		require.NotEqual(t, both[0].Segment, both[1].Segment)
		alone := run(nil, &recordingMerge{}, finding("fourth line", aireview.SeverityP2, 4))
		require.Equal(t, both[1].Segment, alone[0].Segment, "a line's segment does not depend on which other lines have findings")

		posted := func(problem *aiReviewProblem, name string, line int) *aiReviewEarlier {
			return &aiReviewEarlier{Fingerprint: problem.Segment + "/" + name, Priority: p2, Title: name, Sheet: gaps, SheetSha256: "sha", Line: line}
		}
		merge := &recordingMerge{groups: [][]int{{0, 3}, {1, 2}}}
		problems := run([]*aiReviewEarlier{posted(both[0], "second", 2), posted(both[1], "fourth", 4)}, merge,
			finding("fourth line", aireview.SeverityP2, 4), finding("second line", aireview.SeverityP2, 2))
		require.Equal(t, []int{4, 2, 2, 4}, []int{
			merge.requests[0].Candidates[0].Finding.Line, merge.requests[0].Candidates[1].Finding.Line,
			merge.requests[0].Candidates[2].Finding.Line, merge.requests[0].Candidates[3].Finding.Line,
		}, "each earlier result stands on its own line")
		require.Equal(t, merge.requests[0].Candidates[1].Block, merge.requests[0].Candidates[2].Block)
		require.Equal(t, []string{"fourth", "second"}, kept(problems))
	})

	t.Run("keeps a resolved result only for a problem no more severe", func(t *testing.T) {
		t.Parallel()

		merge := &recordingMerge{groups: [][]int{{0, 2}, {1, 3}}}
		problems, err := mergeAIReviewFindings(context.Background(), merge.merge, results(), map[aiReviewSheet][]*aiReviewEarlier{
			sheet: {earlier("accepted as P2", 0, 1, true, p2), earlier("open as P2", 1, 3, false, p2)},
		}, nil)
		require.NoError(t, err)
		require.Equal(t, []string{"", "open as P2"}, kept(problems), "a P1 problem does not keep a P2 a person resolved, but keeps an open one")

		merge = &recordingMerge{groups: [][]int{{0, 1}}}
		problems, err = mergeAIReviewFindings(context.Background(), merge.merge, results(), map[aiReviewSheet][]*aiReviewEarlier{
			sheet: {earlier("accepted as P1", 0, 1, true, p1)},
		}, nil)
		require.NoError(t, err)
		require.Equal(t, []string{"accepted as P1", ""}, kept(problems))
	})

	t.Run("keeps nothing when the merge fails or groups two earlier results", func(t *testing.T) {
		t.Parallel()

		earlierResults := map[aiReviewSheet][]*aiReviewEarlier{sheet: {earlier("a", 0, 1, false, p1), earlier("b", 0, 2, false, p1)}}
		merge := &recordingMerge{err: errors.New("provider down")}
		problems, err := mergeAIReviewFindings(context.Background(), merge.merge, results(), earlierResults, nil)
		require.NoError(t, err)
		require.Equal(t, []string{"", ""}, kept(problems))

		merge = &recordingMerge{groups: [][]int{{0, 1, 2}}}
		problems, err = mergeAIReviewFindings(context.Background(), merge.merge, results(), earlierResults, nil)
		require.NoError(t, err)
		require.Equal(t, []string{"", ""}, kept(problems))
	})
}

func TestSheetStatements(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	sheet := "-- archive old orders\nUPDATE orders SET status = 'archived';\n\nSELECT 1;\nSELECT 1;\n"
	statements := sheetStatements(ctx, storepb.Engine_POSTGRES, "spec-1", sheet)
	require.True(t, statements.Split)
	require.Equal(t, []aireview.LineRange{{Start: 1, End: 2}, {Start: 4, End: 4}, {Start: 5, End: 5}}, statements.Blocks)
	require.Equal(t, []int{2, 4, 5}, statements.BodyStart, "a body leaves out the comment before its statement")
	require.NotEqual(t, statements.Segments[1], statements.Segments[2], "equal statements are told apart by their order")

	same := sheetStatements(ctx, storepb.Engine_POSTGRES, "spec-1", "-- archive orders before 2024\n\nUPDATE orders SET status = 'archived';\nSELECT 1;\nSELECT 1;")
	require.Equal(t, statements.Segments, same.Segments, "the comments and blank lines around a statement are not part of it")
	edited := sheetStatements(ctx, storepb.Engine_POSTGRES, "spec-1", strings.Replace(sheet, "'archived'", "'closed'", 1))
	require.NotEqual(t, statements.Segments[0], edited.Segments[0])
	require.Equal(t, statements.Segments[1:], edited.Segments[1:], "an edit to one statement leaves the others")
	require.NotEqual(t, statements.Segments, sheetStatements(ctx, storepb.Engine_POSTGRES, "spec-2", sheet).Segments, "a segment belongs to its spec")

	whole := sheetStatements(ctx, storepb.Engine_ENGINE_UNSPECIFIED, "spec-1", sheet)
	require.False(t, whole.Split)
	require.Len(t, whole.Segments, 1)
	require.Equal(t, statementSegment("spec-1", strings.TrimSpace(sheet), 1), whole.Segments[0], "a sheet that cannot be split is one statement")
}

func TestAIReviewCommentParts(t *testing.T) {
	t.Parallel()

	unit := func(target string) *aiReviewUnit {
		return &aiReviewUnit{Check: &plancheck.CheckTarget{SpecID: "spec-1", SheetSha256: "sha", Target: target}}
	}
	finding := aireview.Finding{Title: "Drop the view first", Severity: aireview.SeverityP1, Line: 1, Rule: "No column is dropped while a view reads it.", Evidence: "v reads c.\n\nRule: not the rule", Fix: "DROP VIEW v;\n\nThen drop c."}
	for _, targets := range [][]string{{"instances/i/databases/a"}, {"instances/i/databases/a", "instances/i/databases/b"}} {
		problem := &aiReviewProblem{Reviewed: targets}
		for _, target := range targets {
			problem.Findings = append(problem.Findings, &aiReviewFinding{Unit: unit(target), Finding: finding})
		}
		rule, fix := aiReviewCommentParts(aiReviewComments("p", 7, []*aiReviewProblem{problem})[0].Payload.Comment)
		require.Equal(t, []string{finding.Rule, finding.Fix}, []string{rule, fix})
	}
	rule, fix := aiReviewCommentParts("only evidence")
	require.Equal(t, []string{"", ""}, []string{rule, fix})
}

func TestAIReviewCommentsKeepEarlierResults(t *testing.T) {
	t.Parallel()

	problem := func(earlier *aiReviewEarlier) *aiReviewProblem {
		return &aiReviewProblem{
			Findings: []*aiReviewFinding{{
				Unit:    &aiReviewUnit{Check: &plancheck.CheckTarget{SpecID: "spec-1", SheetSha256: "new", Target: "instances/i/databases/a"}},
				Finding: aireview.Finding{Title: "new words", Severity: aireview.SeverityP2, Line: 4, Rule: "r", Evidence: "e", Fix: "f"},
			}},
			Segment: "segment",
			Earlier: earlier,
		}
	}
	comments := aiReviewComments("p", 7, []*aiReviewProblem{
		problem(&aiReviewEarlier{Fingerprint: "kept", Title: "first title", Comment: "first words", Priority: storepb.IssueCommentPayload_ReviewMetadata_P1}),
		problem(nil),
		problem(nil),
	})
	require.Equal(t, "first words", comments[0].Payload.Comment, "a kept result keeps its text")
	require.Equal(t, "first title", comments[0].Payload.ReviewMetadata.Title, "and its title")
	require.Equal(t, "kept", comments[0].Payload.ReviewMetadata.Fingerprint)
	require.Equal(t, storepb.IssueCommentPayload_ReviewMetadata_P2, comments[0].Payload.ReviewMetadata.Priority, "and takes this run's priority")
	require.Equal(t, "new", comments[0].Payload.StatementAnchor.SheetSha256)
	require.Equal(t, int32(4), comments[0].Payload.StatementAnchor.StartPosition.Line)
	require.Equal(t, "new words", comments[1].Payload.ReviewMetadata.Title)
	require.True(t, strings.HasPrefix(comments[1].Payload.ReviewMetadata.Fingerprint, "segment/"), "a new fingerprint starts with its statement's segment")
	require.NotEqual(t, comments[1].Payload.ReviewMetadata.Fingerprint, comments[2].Payload.ReviewMetadata.Fingerprint)
}

func TestLimitAIReviewProblems(t *testing.T) {
	t.Parallel()

	specs := []*storepb.PlanConfig_Spec{{Id: "spec-1"}, {Id: "spec-2"}}
	problem := func(title string, severity aireview.Severity, specID string, line int) *aiReviewProblem {
		return &aiReviewProblem{Findings: []*aiReviewFinding{{
			Unit:    &aiReviewUnit{Check: &plancheck.CheckTarget{SpecID: specID, Target: "instances/i/databases/a"}},
			Finding: aireview.Finding{Title: title, Severity: severity, Line: line},
		}}}
	}
	kept := func(problem *aiReviewProblem, resolved bool) *aiReviewProblem {
		problem.Earlier = &aiReviewEarlier{Fingerprint: problem.Findings[0].Finding.Title, Resolved: resolved}
		return problem
	}
	// onLines returns a P1 problem on each line of spec-1 from first to last.
	onLines := func(first, last int) []*aiReviewProblem {
		var problems []*aiReviewProblem
		for line := first; line <= last; line++ {
			problems = append(problems, problem(fmt.Sprintf("line %d", line), aireview.SeverityP1, "spec-1", line))
		}
		return problems
	}
	names := func(problems []*aiReviewProblem) []string {
		var result []string
		for _, problem := range problems {
			result = append(result, problem.Findings[0].Finding.Title)
		}
		return result
	}
	posted := func(problems []*aiReviewProblem) []string {
		return names(slices.DeleteFunc(slices.Clone(problems), func(problem *aiReviewProblem) bool { return problem.Held }))
	}

	t.Run("posts the most important problems", func(t *testing.T) {
		t.Parallel()

		problems := append([]*aiReviewProblem{
			problem("advisory", aireview.SeverityP2, "spec-1", 1),
			problem("second spec", aireview.SeverityP1, "spec-2", 1),
		}, onLines(2, 10)...)
		problems = append(problems, problem("blocking", aireview.SeverityP0, "spec-2", 5))
		require.Equal(t, 2, limitAIReviewProblems(specs, problems))
		require.Equal(t, append(names(onLines(2, 10)), "blocking"), posted(problems), "P0 first, then P1 in reading order")

		few := onLines(1, 3)
		require.Zero(t, limitAIReviewProblems(specs, few))
		require.Equal(t, names(few), posted(few))
	})

	t.Run("posts every problem that keeps an earlier result", func(t *testing.T) {
		t.Parallel()

		problems := append([]*aiReviewProblem{
			kept(problem("accepted", aireview.SeverityP1, "spec-1", 1), true),
			kept(problem("reported again", aireview.SeverityP0, "spec-1", 30), false),
		}, onLines(1, 11)...)
		problems = append(problems, kept(problem("open below the limit", aireview.SeverityP2, "spec-1", 20), false))
		require.Equal(t, 2, limitAIReviewProblems(specs, problems))
		want := append([]string{"accepted", "reported again"}, names(onLines(1, 9))...)
		require.Equal(t, append(want, "open below the limit"), posted(problems), "a resolved thread takes no place, and an open one takes a place but posts below the limit too")
	})

	t.Run("gives a problem on a rule finding no place", func(t *testing.T) {
		t.Parallel()

		rule := dropRule()
		onRule := problem("on the rule's operation", aireview.SeverityP2, "spec-1", 40)
		onRule.Rule = &ruleMember{result: rule, finding: rule.members[0]}
		problems := append([]*aiReviewProblem{onRule}, onLines(1, 10)...)
		require.Zero(t, limitAIReviewProblems(specs, problems), "it posts no thread of its own, so it is never held")
		require.False(t, onRule.Held)
	})
}

// recordingSummary records the summary requests and answers each with a
// summary titled after its first finding, or fails or panics on the titles
// named.
type recordingSummary struct {
	mu       sync.Mutex
	requests []*aireview.SummaryRequest
	fail     string
	panic    string
}

func (m *recordingSummary) summarize(_ context.Context, request *aireview.SummaryRequest) (*aireview.Summary, error) {
	m.mu.Lock()
	m.requests = append(m.requests, request)
	m.mu.Unlock()
	switch title := request.Findings[0].Finding.Title; title {
	case m.fail:
		return nil, errors.New("provider down")
	case m.panic:
		panic("summary bug")
	default:
		return &aireview.Summary{Title: "summary of " + title}, nil
	}
}

func TestSummarizeAIReviewProblems(t *testing.T) {
	t.Parallel()

	finding := func(target, title string) *aiReviewFinding {
		return &aiReviewFinding{
			Unit:    &aiReviewUnit{Check: &plancheck.CheckTarget{SpecID: "spec-1", Target: target}, Statement: mergeStatement},
			Finding: aireview.Finding{Title: title, Severity: aireview.SeverityP1, Line: 1},
		}
	}
	problems := func() []*aiReviewProblem {
		return []*aiReviewProblem{
			{Findings: []*aiReviewFinding{finding("instances/i/databases/b", "drop"), finding("instances/i/databases/a", "drop")}, Block: aireview.LineRange{Start: 1, End: 2}},
			{Findings: []*aiReviewFinding{finding("instances/i/databases/a", "index"), finding("instances/i/databases/a", "index again")}},
			{Findings: []*aiReviewFinding{finding("instances/i/databases/a", "failing"), finding("instances/i/databases/b", "failing")}},
			{Findings: []*aiReviewFinding{finding("instances/i/databases/a", "panicking"), finding("instances/i/databases/b", "panicking")}},
			{Findings: []*aiReviewFinding{finding("instances/i/databases/a", "kept"), finding("instances/i/databases/b", "kept")}, Earlier: &aiReviewEarlier{Fingerprint: "kept"}},
			{Findings: []*aiReviewFinding{finding("instances/i/databases/a", "held"), finding("instances/i/databases/b", "held")}, Held: true},
			{Findings: []*aiReviewFinding{finding("instances/i/databases/a", "unchanged"), finding("instances/i/databases/b", "unchanged")}, Unchanged: true},
		}
	}

	t.Run("summarizes the problems of more than one database", func(t *testing.T) {
		t.Parallel()

		summary := &recordingSummary{fail: "failing", panic: "panicking"}
		problems := problems()
		require.NoError(t, summarizeAIReviewProblems(context.Background(), summary.summarize, problems))
		require.Equal(t, "summary of drop", problems[0].Summary.Title)
		require.Nil(t, problems[1].Summary, "one database's problem is not summarized")
		require.Nil(t, problems[2].Summary, "a failed summary leaves the lead finding")
		require.Nil(t, problems[3].Summary, "a panic is that problem's failure")
		require.Nil(t, problems[4].Summary, "a problem that keeps an earlier result keeps its text")
		require.Nil(t, problems[5].Summary, "a held problem posts no comment to summarize")
		require.Nil(t, problems[6].Summary, "a problem its rule result's earlier comment already states is not written again")

		require.Len(t, summary.requests, 3)
		var request *aireview.SummaryRequest
		for _, r := range summary.requests {
			if r.Findings[0].Finding.Title == "drop" {
				request = r
			}
		}
		require.NotNil(t, request)
		require.Equal(t, mergeStatement, request.Statement)
		require.Equal(t, aireview.LineRange{Start: 1, End: 2}, request.Block)
		require.Equal(t, []string{"instances/i/databases/a", "instances/i/databases/b"}, []string{request.Findings[0].Database, request.Findings[1].Database}, "ordered by database")
	})

	t.Run("stops when the run ends", func(t *testing.T) {
		t.Parallel()

		ctx, cancel := context.WithCancel(context.Background())
		summarize := func(context.Context, *aireview.SummaryRequest) (*aireview.Summary, error) {
			cancel()
			return nil, context.Canceled
		}
		require.ErrorIs(t, summarizeAIReviewProblems(ctx, summarize, problems()), context.Canceled)
	})
}

func TestStatementBlocks(t *testing.T) {
	t.Parallel()

	blocks, split := statementBlocks(storepb.Engine_POSTGRES, mergeStatement)
	require.True(t, split)
	require.Equal(t, []aireview.LineRange{{Start: 1, End: 2}, {Start: 3, End: 3}}, blocks)
	blocks, split = statementBlocks(storepb.Engine_POSTGRES, "-- c\n\nSELECT 1;\n\n-- d\nSELECT 2;")
	require.True(t, split)
	require.Equal(t, []aireview.LineRange{{Start: 1, End: 3}, {Start: 5, End: 6}}, blocks, "a statement keeps its comments and leaves out the blank lines before it")
	blocks, split = statementBlocks(storepb.Engine_ENGINE_UNSPECIFIED, mergeStatement)
	require.False(t, split)
	require.Equal(t, []aireview.LineRange{{Start: 1, End: 4}}, blocks)

	blocks, _ = statementBlocks(storepb.Engine_POSTGRES, "SELECT 1;\n\nSELECT 2;")
	require.Equal(t, 0, findBlock(&blocks, 1))
	require.Equal(t, 2, findBlock(&blocks, 2), "a line between statements is a block of its own")
	require.Equal(t, 2, findBlock(&blocks, 2))
	require.Equal(t, []aireview.LineRange{{Start: 1, End: 1}, {Start: 3, End: 3}, {Start: 2, End: 2}}, blocks)
}

func TestUniqueCheckTargets(t *testing.T) {
	t.Parallel()

	targets := []*plancheck.CheckTarget{
		{SpecID: "spec-1", Target: "instances/i/databases/a"},
		{SpecID: "spec-1", Target: "instances/i/databases/b"},
		{SpecID: "spec-1", Target: "instances/i/databases/a"},
		{SpecID: "spec-2", Target: "instances/i/databases/a"},
	}
	unique := uniqueCheckTargets(targets)
	require.Equal(t, []*plancheck.CheckTarget{targets[0], targets[1], targets[3]}, unique, "the same database under another spec is another review")
	require.Empty(t, uniqueCheckTargets(nil))
}

func TestCheckSheetSize(t *testing.T) {
	t.Parallel()

	require.NoError(t, checkSheetSize(strings.Repeat("s", maxAIReviewSheetBytes)))
	err := checkSheetSize(strings.Repeat("s", maxAIReviewSheetBytes+1))
	require.ErrorContains(t, err, "not reviewed")
	require.ErrorContains(t, err, "262145 bytes")
}

// TestReviewUnits pins the executor's concurrency guarantees: at most
// aiReviewRunConcurrency reviews in flight, every unit attempted whatever the
// others do, a panicking review recorded as that unit's error, and every slot
// returned whatever the review did.
func TestReviewUnits(t *testing.T) {
	t.Parallel()

	units := make([]*aiReviewUnit, 20)
	for i := range units {
		units[i] = &aiReviewUnit{Check: &plancheck.CheckTarget{SpecID: "spec-1", Target: "instances/i/databases/db" + string(rune('a'+i))}}
	}
	var inFlight, maxInFlight atomic.Int32
	review := func(_ context.Context, unit *aiReviewUnit) (*aireview.Result, error) {
		now := inFlight.Add(1)
		defer inFlight.Add(-1)
		for {
			seen := maxInFlight.Load()
			if now <= seen || maxInFlight.CompareAndSwap(seen, now) {
				break
			}
		}
		time.Sleep(20 * time.Millisecond)
		switch unit.Check.Target {
		case units[3].Check.Target:
			return nil, errors.New("model call 1 failed")
		case units[5].Check.Target:
			panic("boom")
		default:
			return &aireview.Result{Findings: []aireview.Finding{{Title: "finding on " + unit.Check.Target, Line: 1}}}, nil
		}
	}

	slots := semaphore.NewWeighted(aiReviewServerConcurrency)
	results, errs := reviewUnits(context.Background(), slots, units, review)
	require.Len(t, results, 20)
	require.Len(t, errs, 20)
	require.LessOrEqual(t, maxInFlight.Load(), int32(aiReviewRunConcurrency))
	require.Greater(t, maxInFlight.Load(), int32(1), "the reviews run concurrently")
	for i, result := range results {
		switch i {
		case 3:
			require.Nil(t, result)
			require.ErrorContains(t, errs[i], units[3].Check.Target+": model call 1 failed")
		case 5:
			require.Nil(t, result)
			require.ErrorContains(t, errs[i], units[5].Check.Target+": review panic: boom")
		default:
			require.NoError(t, errs[i])
			require.Same(t, units[i], result.Unit)
			require.Equal(t, "finding on "+units[i].Check.Target, result.Findings[0].Title)
		}
	}
	require.Zero(t, inFlight.Load())
	require.True(t, slots.TryAcquire(aiReviewServerConcurrency), "the failed and the panicking reviews return their slots too")
}

func TestReviewUnitsStopWaitingForASlotWhenTheRunEnds(t *testing.T) {
	t.Parallel()

	slots := semaphore.NewWeighted(1)
	require.True(t, slots.TryAcquire(1), "another run holds the only slot")
	units := []*aiReviewUnit{{Check: &plancheck.CheckTarget{SpecID: "spec-1", Target: "instances/i/databases/a"}}}
	var reviews atomic.Int32
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan []error)
	go func() {
		_, errs := reviewUnits(ctx, slots, units, func(context.Context, *aiReviewUnit) (*aireview.Result, error) {
			reviews.Add(1)
			return &aireview.Result{}, nil
		})
		done <- errs
	}()
	cancel()
	errs := receive(t, done)
	require.ErrorIs(t, errs[0], context.Canceled)
	require.Zero(t, reviews.Load(), "no review starts without a slot")
}
