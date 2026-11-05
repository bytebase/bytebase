package reviewrun

import (
	"context"
	"fmt"
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
	require.Equal(t, "Drop orders_summary and orders_report before legacy_status"+
		"\n\nprod-1/db: orders_summary and orders_report select legacy_status.\n"+
		"\n- **P0** on prod-1/db"+
		"\n- **P1** on app, prod-2/db"+
		"\n- Not flagged on billing"+
		"\n\nRule: No column is dropped while a view reads it."+
		"\n\nSuggested fix: Drop both views first.", merged.Payload.Comment, "without a summary, the P0 finding leads and the databases group by severity")
	anchor := merged.Payload.StatementAnchor
	require.Equal(t, "spec-1", anchor.SpecId)
	require.Equal(t, "sha", anchor.SheetSha256)
	require.Equal(t, int32(4), anchor.StartPosition.Line, "the line most of the findings start on")
	require.Equal(t, int32(4), anchor.EndPosition.Line)
	metadata := merged.Payload.ReviewMetadata
	require.Equal(t, storepb.ReviewRun_AI, metadata.RunType)
	require.Equal(t, storepb.ReviewRuleType_REVIEW_RULE_TYPE_UNSPECIFIED, metadata.RuleType)
	require.Equal(t, storepb.IssueCommentPayload_ReviewMetadata_P0, metadata.Priority)
	require.Equal(t, []string{"instances/prod-1/databases/db", "instances/prod-2/databases/db", "instances/staging/databases/app"}, metadata.Targets)

	single := comments[1]
	require.Equal(t, "Create the index CONCURRENTLY\n\norders holds 52,000,000 rows.\n\nRule: An index on a table over 1 million rows is created CONCURRENTLY.\n\nSuggested fix: CREATE INDEX CONCURRENTLY idx_orders_created_at ON orders (created_at);", single.Payload.Comment)
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
	require.Equal(t, "Drop the views that read legacy_status first"+
		"\n\nViews select legacy_status, so the DROP fails.\n"+
		"\n- **P0** on prod-1/db: orders_summary and orders_report read it."+
		"\n- **P1** on app, prod-2/db: orders_summary reads it."+
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
	require.Equal(t, "t\n\ne\n\n- **P0–P1** on app, prod-1/db, prod-2/db\n\nRule: No column is dropped while a view reads it.\n\nSuggested fix: f", comment.Payload.Comment, "a group shows the range of its severities, and every database flagged the problem")

	// billing reports a different problem on the drop's statement.
	other := drop
	other.Title = "Archive legacy_status first"
	withOther := func(blockIsStatement bool) string {
		problem := dropProblem()
		problem.BlockIsStatement = blockIsStatement
		otherProblem := &aiReviewProblem{Findings: []*aiReviewFinding{{Unit: unit("instances/staging/databases/billing"), Finding: other}}, Reviewed: reviewed, BlockIsStatement: blockIsStatement}
		return aiReviewComments("p", 7, []*aiReviewProblem{problem, otherProblem})[0].Payload.Comment
	}
	require.Contains(t, withOther(true), "\n- **P1** on app, prod-2/db\n- Flagged differently on billing\n\nRule:")
	require.NotContains(t, withOther(true), "Not flagged")
	require.Contains(t, withOther(false), "\n- **P1** on app, prod-2/db\n\nRule:", "on a sheet that could not be split, the other finding can be on any statement")
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
	require.True(t, strings.HasPrefix(comments[0].Payload.Comment, "instances/i/databases/a\n\na: e\n\n- **P1** on a, b, c, d\n"), "the first database leads on a tie")
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
	err      error
}

func (m *recordingMerge) merge(_ context.Context, request *aireview.MergeRequest) (*aireview.MergeResult, error) {
	m.requests = append(m.requests, request)
	if m.err != nil {
		return nil, m.err
	}
	return &aireview.MergeResult{Groups: m.groups}, nil
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
		problems, err := mergeAIReviewFindings(context.Background(), merge.merge, results(storepb.Engine_POSTGRES), nil)
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
		problems, err := mergeAIReviewFindings(context.Background(), merge.merge, results(storepb.Engine_POSTGRES), nil)
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
		_, err := mergeAIReviewFindings(ctx, merge, results(storepb.Engine_POSTGRES), nil)
		require.ErrorIs(t, err, context.Canceled)
	})

	t.Run("treats the sheet as one block when the engines differ", func(t *testing.T) {
		t.Parallel()

		mixed := results(storepb.Engine_POSTGRES)
		mixed[3].Unit.Engine = storepb.Engine_MYSQL
		merge := &recordingMerge{groups: [][]int{{0, 2}, {1}, {3}}}
		problems, err := mergeAIReviewFindings(context.Background(), merge.merge, mixed, nil)
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
		problems, err := mergeAIReviewFindings(context.Background(), merge.merge, results(storepb.Engine_POSTGRES)[:1], nil)
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
			Comment:     name + "\n\nevidence on db1\n\nRule: r\n\nSuggested fix: f",
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
		})
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
		_, err := mergeAIReviewFindings(context.Background(), merge.merge, results(), map[aiReviewSheet][]*aiReviewEarlier{sheet: {moved}})
		require.NoError(t, err)
		require.Equal(t, 1, merge.requests[0].Candidates[1].Finding.Line, "the statement's first line, where its body starts now")
	})

	t.Run("keeps a result on a line outside every statement", func(t *testing.T) {
		t.Parallel()

		const gap = "SELECT 1;\n\nSELECT 2;\n"
		result := mergeResult("instances/prod/databases/db1", storepb.Engine_POSTGRES, finding("between", aireview.SeverityP2, 2))
		result.Unit.Statement = gap
		first, err := mergeAIReviewFindings(context.Background(), (&recordingMerge{}).merge, []*aiReviewResult{result}, nil)
		require.NoError(t, err)
		posted := &aiReviewEarlier{Fingerprint: first[0].Segment + "/between", Priority: p2, Comment: "between", Sheet: gap, SheetSha256: "sha", Line: 2}
		merge := &recordingMerge{groups: [][]int{{0, 1}}}
		problems, err := mergeAIReviewFindings(context.Background(), merge.merge, []*aiReviewResult{result}, map[aiReviewSheet][]*aiReviewEarlier{sheet: {posted}})
		require.NoError(t, err)
		require.Equal(t, []string{"between"}, kept(problems))
	})

	t.Run("keeps a resolved result only for a problem no more severe", func(t *testing.T) {
		t.Parallel()

		merge := &recordingMerge{groups: [][]int{{0, 2}, {1, 3}}}
		problems, err := mergeAIReviewFindings(context.Background(), merge.merge, results(), map[aiReviewSheet][]*aiReviewEarlier{
			sheet: {earlier("accepted as P2", 0, 1, true, p2), earlier("open as P2", 1, 3, false, p2)},
		})
		require.NoError(t, err)
		require.Equal(t, []string{"", "open as P2"}, kept(problems), "a P1 problem does not keep a P2 a person resolved, but keeps an open one")

		merge = &recordingMerge{groups: [][]int{{0, 1}}}
		problems, err = mergeAIReviewFindings(context.Background(), merge.merge, results(), map[aiReviewSheet][]*aiReviewEarlier{
			sheet: {earlier("accepted as P1", 0, 1, true, p1)},
		})
		require.NoError(t, err)
		require.Equal(t, []string{"accepted as P1", ""}, kept(problems))
	})

	t.Run("keeps nothing when the merge fails or groups two earlier results", func(t *testing.T) {
		t.Parallel()

		earlierResults := map[aiReviewSheet][]*aiReviewEarlier{sheet: {earlier("a", 0, 1, false, p1), earlier("b", 0, 2, false, p1)}}
		merge := &recordingMerge{err: errors.New("provider down")}
		problems, err := mergeAIReviewFindings(context.Background(), merge.merge, results(), earlierResults)
		require.NoError(t, err)
		require.Equal(t, []string{"", ""}, kept(problems))

		merge = &recordingMerge{groups: [][]int{{0, 1, 2}}}
		problems, err = mergeAIReviewFindings(context.Background(), merge.merge, results(), earlierResults)
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
		title, rule, fix := aiReviewCommentParts(aiReviewComments("p", 7, []*aiReviewProblem{problem})[0].Payload.Comment)
		require.Equal(t, []string{finding.Title, finding.Rule, finding.Fix}, []string{title, rule, fix})
	}
	title, rule, fix := aiReviewCommentParts("only a title")
	require.Equal(t, []string{"only a title", "", ""}, []string{title, rule, fix})
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
		problem(&aiReviewEarlier{Fingerprint: "kept", Comment: "first words", Priority: storepb.IssueCommentPayload_ReviewMetadata_P1}),
		problem(nil),
		problem(nil),
	})
	require.Equal(t, "first words", comments[0].Payload.Comment, "a kept result keeps its text")
	require.Equal(t, "kept", comments[0].Payload.ReviewMetadata.Fingerprint)
	require.Equal(t, storepb.IssueCommentPayload_ReviewMetadata_P2, comments[0].Payload.ReviewMetadata.Priority, "and takes this run's priority")
	require.Equal(t, "new", comments[0].Payload.StatementAnchor.SheetSha256)
	require.Equal(t, int32(4), comments[0].Payload.StatementAnchor.StartPosition.Line)
	require.True(t, strings.HasPrefix(comments[1].Payload.Comment, "new words"))
	require.True(t, strings.HasPrefix(comments[1].Payload.ReviewMetadata.Fingerprint, "segment/"), "a new fingerprint starts with its statement's segment")
	require.NotEqual(t, comments[1].Payload.ReviewMetadata.Fingerprint, comments[2].Payload.ReviewMetadata.Fingerprint)
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
