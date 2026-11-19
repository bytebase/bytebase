package reviewrun

import (
	"context"
	"slices"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/bytebase/omni/review"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	"golang.org/x/sync/semaphore"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/plugin/aireview"
	"github.com/bytebase/bytebase/backend/runner/plancheck"
	"github.com/bytebase/bytebase/backend/store"
)

// factsOf is the result of db's review of mergeStatement's sheet, which adds
// text to the standard finding id.
func factsOf(db, id, text string) *aiReviewResult {
	return &aiReviewResult{
		Unit:  &aiReviewUnit{Check: &plancheck.CheckTarget{SpecID: "spec-1", SheetSha256: "sha", Target: "instances/prod/databases/" + db}, Statement: mergeStatement, Engine: storepb.Engine_POSTGRES},
		Facts: []aireview.Fact{{Standard: id, Text: text}},
	}
}

func TestStandardFindings(t *testing.T) {
	t.Parallel()

	drop := dropRule()
	drop.members[0].targets = []string{"instances/prod/databases/db1", "instances/prod/databases/db2"}
	whole := dropRule()
	whole.members[0].anchor = nil
	other := dropRule()
	other.specID = "spec-2"
	bySheet := standardFindings([]*ruleResult{drop, whole, other, dropRule()})

	sheet := bySheet[aiReviewSheet{specID: "spec-1", sheetSha256: "sha"}]
	require.Len(t, sheet, 2, "a finding on the whole change has no statement to add facts to")
	require.Equal(t, []string{"S1", "S2"}, []string{sheet[0].ID, sheet[1].ID})
	require.Len(t, bySheet[aiReviewSheet{specID: "spec-2", sheetSha256: "sha"}], 1)

	require.Equal(t, []aireview.StandardFinding{
		{ID: "S1", Line: 2, Rule: "DISALLOW_DROP_OBJECT", Codes: []string{"DO101"}, Text: "Drops column `legacy_status`."},
	}, standardFindingsOn(sheet, "instances/prod/databases/db2"), "a review is told the findings on its own database")
	require.Len(t, standardFindingsOn(sheet, "instances/prod/databases/db1"), 2)
	require.Empty(t, standardFindingsOn(sheet, "instances/prod/databases/db3"))
}

// recordingFacts answers a facts summary with what db2 holds apart from db1.
type recordingFacts struct {
	requests []*aireview.FactsRequest
}

func (m *recordingFacts) summarize(_ context.Context, request *aireview.FactsRequest) (*aireview.FactsSummary, error) {
	m.requests = append(m.requests, request)
	return &aireview.FactsSummary{Shared: "Nothing else reads it.", Groups: []aireview.SummaryGroup{
		{Databases: []string{"instances/prod/databases/db2"}, Facts: "It holds 12,000 values."},
		{Databases: []string{"instances/prod/databases/db1"}},
	}}, nil
}

func TestAddRuleFacts(t *testing.T) {
	t.Parallel()

	sheets := map[string]string{"sha": mergeStatement}
	onBoth := func() *ruleResult {
		rule := dropRule()
		rule.engine = storepb.Engine_POSTGRES
		rule.members[0].targets = []string{"instances/prod/databases/db1", "instances/prod/databases/db2"}
		return rule
	}
	failing := func(context.Context, *aireview.FactsRequest) (*aireview.FactsSummary, error) {
		return nil, errors.New("provider down")
	}

	t.Run("one database's facts are the text", func(t *testing.T) {
		t.Parallel()
		rule := dropRule()
		require.NoError(t, addRuleFacts(context.Background(), failing, standardFindings([]*ruleResult{rule}), []*aiReviewResult{factsOf("db1", "S1", "The column is empty."), nil}, sheets))
		require.Equal(t, "Drops column `legacy_status`.\n\nThe column is empty.", rule.payload().Comment)

		both := onBoth()
		require.NoError(t, addRuleFacts(context.Background(), failing, standardFindings([]*ruleResult{both}), []*aiReviewResult{factsOf("db2", "S1", "The column is empty.")}, sheets))
		require.Equal(t, "db2: The column is empty.", both.members[0].facts, "named, since the fact says nothing about db1")
	})

	t.Run("the facts of several databases are summarized", func(t *testing.T) {
		t.Parallel()
		summary := &recordingFacts{}
		rule := onBoth()
		results := []*aiReviewResult{factsOf("db2", "S1", "It holds 12,000 values."), factsOf("db1", "S1", "It is empty."), factsOf("db3", "S1", "not a database of the finding"), factsOf("db1", "S7", "no such finding")}
		require.NoError(t, addRuleFacts(context.Background(), summary.summarize, standardFindings([]*ruleResult{rule}), results, sheets))
		require.Equal(t, "Drops column `legacy_status`.\n\nNothing else reads it.\n\n- db2: It holds 12,000 values.\n- db1", rule.payload().Comment)
		requests := summary.requests
		require.Len(t, requests, 1)
		require.Equal(t, aireview.LineRange{Start: 1, End: 2}, requests[0].Block, "the whole statement, not the clause the finding is anchored on")
		require.Equal(t, []aireview.DatabaseFact{
			{Database: "instances/prod/databases/db1", Text: "It is empty."},
			{Database: "instances/prod/databases/db2", Text: "It holds 12,000 values."},
		}, requests[0].Facts)
		require.Equal(t, "S1", requests[0].Finding.ID)
	})

	t.Run("facts from part of the finding's databases name them", func(t *testing.T) {
		t.Parallel()
		summary := &recordingFacts{}
		rule := onBoth()
		rule.members[0].targets = append(rule.members[0].targets, "instances/prod/databases/db3")
		results := []*aiReviewResult{factsOf("db2", "S1", "It holds 12,000 values."), factsOf("db1", "S1", "It is empty.")}
		require.NoError(t, addRuleFacts(context.Background(), summary.summarize, standardFindings([]*ruleResult{rule}), results, sheets))
		require.Equal(t, "db1, db2: Nothing else reads it.\n\n- db2: It holds 12,000 values.\n- db1", rule.members[0].facts, "db3 added no fact, so the shared sentence is not about it")
	})

	t.Run("a failed summary shows one database's facts and counts the rest", func(t *testing.T) {
		t.Parallel()
		rule := onBoth()
		results := []*aiReviewResult{factsOf("db2", "S1", "It holds 12,000 values."), factsOf("db1", "S1", "It is empty.")}
		require.NoError(t, addRuleFacts(context.Background(), failing, standardFindings([]*ruleResult{rule}), results, sheets))
		require.Equal(t, "db1: It is empty.\n\n1 more databases added facts.", rule.members[0].facts)
	})

	t.Run("a result that keeps its earlier comment takes none", func(t *testing.T) {
		t.Parallel()
		rule := dropRule()
		rule.earlier = &ruleThread{Comment: "Drops column `legacy_status`.\n\nThe column held 3 values."}
		rule.keepsEarlier = true
		require.NoError(t, addRuleFacts(context.Background(), failing, standardFindings([]*ruleResult{rule}), []*aiReviewResult{factsOf("db1", "S1", "The column is empty.")}, sheets))
		require.Equal(t, "Drops column `legacy_status`.\n\nThe column held 3 values.", rule.payload().Comment)
	})

	t.Run("a finding that holds a problem of this run takes none", func(t *testing.T) {
		t.Parallel()
		rule := dropRule()
		rule.members[0].folded = []string{"v_orders reads it."}
		require.NoError(t, addRuleFacts(context.Background(), failing, standardFindings([]*ruleResult{rule}), []*aiReviewResult{factsOf("db1", "S1", "The column is empty.")}, sheets))
		require.Empty(t, rule.members[0].facts)
		require.Equal(t, "Drops column `legacy_status`.\n\nv_orders reads it.", rule.payload().Comment)
	})

	t.Run("one group needs no list", func(t *testing.T) {
		t.Parallel()
		one := &aireview.FactsSummary{Shared: "Nothing reads it.", Groups: []aireview.SummaryGroup{{Databases: []string{"a", "b"}, Facts: "It is empty."}}}
		require.Equal(t, "Nothing reads it. It is empty.", factsText(one, nil, ""))
		require.Equal(t, "a, b: Nothing reads it. It is empty.", factsText(one, nil, "a, b: "), "named when the finding is on more databases")
		require.Equal(t, "- a: x\n- b: y", factsText(&aireview.FactsSummary{Groups: []aireview.SummaryGroup{{Databases: []string{"a"}, Facts: "x"}, {Databases: []string{"b"}, Facts: "y"}}}, map[string]string{"a": "a", "b": "b"}, "a, b: "), "every line names its databases already")
	})

	t.Run("stops when the run ends", func(t *testing.T) {
		t.Parallel()
		ctx, cancel := context.WithCancel(context.Background())
		canceling := func(context.Context, *aireview.FactsRequest) (*aireview.FactsSummary, error) {
			cancel()
			return nil, context.Canceled
		}
		rule := onBoth()
		results := []*aiReviewResult{factsOf("db2", "S1", "a"), factsOf("db1", "S1", "b")}
		require.ErrorIs(t, addRuleFacts(ctx, canceling, standardFindings([]*ruleResult{rule}), results, sheets), context.Canceled)
	})
}

func TestKeepFactsOnlyComments(t *testing.T) {
	t.Parallel()

	const rules = "Drops column `legacy_status`."
	keeps := func(comment string, problems ...*aiReviewProblem) bool {
		rule := dropRule()
		rule.earlier = &ruleThread{Comment: comment}
		for _, problem := range problems {
			problem.Rule = &ruleMember{result: rule, finding: rule.members[0]}
		}
		keepFactsOnlyComments([]*ruleResult{rule}, problems)
		return rule.keepsEarlier
	}
	require.True(t, keeps(rules+"\n\nThe column is empty."), "facts stay as they were first written")
	require.False(t, keeps(rules), "the rules' text alone: this run's facts are new")
	require.False(t, keeps(rules+"\n\nv_orders reads it.\n\nSuggested fix: Drop v_orders first."), "the problem it states is gone")
	require.False(t, keeps(rules+"\n\nThe column is empty.", &aiReviewProblem{}), "a problem of this run is not in it")
	require.False(t, keeps("Drops the column `legacy_status`.\n\nThe column is empty."), "the rules' text changed")
	require.True(t, keeps(rules+"\n\ndb1: The column is empty."), "facts that name their one database stay too")
}

func TestRuleResultShowsFactsOrProblems(t *testing.T) {
	t.Parallel()

	rule := dropRule()
	rule.members[0].facts = "The column holds 12,000 values."
	require.Equal(t, "Drops column `legacy_status`.\n\nThe column holds 12,000 values.", rule.payload().Comment)
	require.Equal(t, "Drops column `legacy_status`.", rule.ruleText(), "the rules' text carries nothing of the AI review")
	require.Equal(t, "The column holds 12,000 values.", rule.members[0].facts, "reading the rules' text leaves the result as it is")

	rule.members[0].folded = []string{"v_orders reads it.\n\nSuggested fix: Drop v_orders first."}
	require.Equal(t, "Drops column `legacy_status`.\n\nv_orders reads it.\n\nSuggested fix: Drop v_orders first.", rule.payload().Comment, "a problem says what the statement touches")
}

// factsModel adds one fact to the first standard finding a review lists, and
// answers a facts summary with one group of every database.
type factsModel struct {
	reviews   atomic.Int32
	summaries atomic.Int32
	// listed counts the reviews that were told a standard finding.
	listed atomic.Int32
	// onSummary runs during a facts summary, when set.
	onSummary func()
}

func (m *factsModel) Chat(_ context.Context, request *v1pb.AIChatRequest) (*v1pb.AIChatResponse, error) {
	user := request.GetMessages()[1].GetContent()
	content := `{"findings": [], "notes": []}`
	switch {
	case len(request.GetToolDefinitions()) > 0:
		m.reviews.Add(1)
		if strings.Contains(user, `{"id":"S1",`) {
			m.listed.Add(1)
			content = `{"findings": [], "facts": [{"standard": "S1", "text": "` + "`orders`" + ` holds 5,000 rows."}], "notes": []}`
		}
	case strings.Contains(user, "\n# Facts\n"):
		m.summaries.Add(1)
		if m.onSummary != nil {
			m.onSummary()
		}
		var ids []string
		for _, match := range summaryDatabaseID.FindAllStringSubmatch(user, -1) {
			if id := `"` + match[1] + `"`; !slices.Contains(ids, id) {
				ids = append(ids, id)
			}
		}
		content = `{"shared": "` + "`orders`" + ` holds about 5,000 rows.", "groups": [{"databases": [` + strings.Join(ids, ", ") + `], "facts": ""}]}`
	default:
	}
	return &v1pb.AIChatResponse{Content: &content}, nil
}

// factsExecutor reviews with a rule that flags the whole sheet as "no WHERE"
// on every database, and with model as the AI review.
func factsExecutor(stores *store.Store, model aireview.Model) *ReviewExecutor {
	return NewReviewExecutor(stores, licenseStub{},
		&RuleExecutor{store: stores, reviewFuncs: map[storepb.Engine]review.ReviewFunc{
			storepb.Engine_POSTGRES: func(_ context.Context, sql string, _ review.Options, targets []review.Target) (*review.Result, error) {
				every := make([]int, len(targets))
				for i := range targets {
					every[i] = i
				}
				whole := review.Range{Start: 0, End: len(sql)}
				return &review.Result{
					Statements: []review.Range{whole},
					Bodies:     []review.Range{whole},
					Findings:   []review.Finding{{Rule: review.RequireWhere, Statement: 0, Message: "no WHERE", Targets: every}},
				}, nil
			},
		}},
		&AIReviewExecutor{
			license:  licenseStub{},
			store:    stores,
			newModel: func(*storepb.AISetting, ...aireview.ModelOption) aireview.Model { return model },
			slots:    semaphore.NewWeighted(aiReviewServerConcurrency),
		},
	)
}

// TestReviewAddsFactsToTheRulesThread drives the real reviewers through runs
// in which the AI review finds no problem: each review is told the rule
// finding on its database, the facts they add post in the rule's thread, and
// an unchanged re-run keeps the comment without another summary.
func TestReviewAddsFactsToTheRulesThread(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	stores, issues := setupReviewFixture(ctx, t, 1)
	model := &factsModel{}
	executor := factsExecutor(stores, model)
	run := func() []*store.IssueCommentMessage {
		t.Helper()
		results, err := executor.RunOnce(ctx, "p", issues[0])
		require.NoError(t, err)
		_, err = stores.CreateReviewRun(ctx, "p", issues[0])
		require.NoError(t, err)
		claimed, err := stores.ClaimAvailableReviewRuns(ctx, "replica-1")
		require.NoError(t, err)
		require.Len(t, claimed, 1)
		updated, err := stores.CompleteReviewRun(ctx, claimed[0], "replica-1", storepb.ReviewRun_DONE, nil, results)
		require.NoError(t, err)
		require.True(t, updated)
		comments, err := stores.ListIssueComment(ctx, &store.FindIssueCommentMessage{ProjectID: "p", IssueUID: &issues[0], TopLevelOnly: true})
		require.NoError(t, err)
		return slices.DeleteFunc(comments, func(c *store.IssueCommentMessage) bool {
			return c.Payload.GetReviewMetadata() == nil || c.Payload.GetReviewMetadata().GetSuperseded()
		})
	}
	const withFacts = "no WHERE\n\n`orders` holds about 5,000 rows."

	first := run()
	require.Len(t, first, 1)
	require.Equal(t, withFacts, first[0].Payload.Comment)
	require.Equal(t, storepb.ReviewRuleType_REQUIRE_WHERE, first[0].Payload.ReviewMetadata.RuleType)
	require.Equal(t, storepb.IssueCommentPayload_ReviewMetadata_P1, first[0].Payload.ReviewMetadata.Priority, "a fact leaves the rule's priority")
	require.Equal(t, int32(fixtureDatabases), model.listed.Load(), "every review was told the rule finding")
	require.Equal(t, int32(1), model.summaries.Load())

	second := run()
	require.Len(t, second, 1)
	require.Equal(t, first[0].ResourceID, second[0].ResourceID)
	require.Equal(t, withFacts, second[0].Payload.Comment)
	require.Equal(t, int32(1), model.summaries.Load(), "facts stay as first written, so nothing is summarized again")

	setAIReview(ctx, t, stores, false)
	third := run()
	require.Len(t, third, 1)
	require.Equal(t, "no WHERE", third[0].Payload.Comment, "a run without the AI review posts the rule's text alone")
}

func TestReviewPostsTheRulesTextWhenSwitchedOffDuringTheFacts(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	stores, issues := setupReviewFixture(ctx, t, 1)
	var switchOffErr error
	model := &factsModel{}
	model.onSummary = func() {
		_, switchOffErr = stores.UpsertSetting(ctx, &store.SettingMessage{Name: storepb.SettingName_AI, Workspace: "ws", Value: &storepb.AISetting{Enabled: true}})
	}
	results, err := factsExecutor(stores, model).RunOnce(ctx, "p", issues[0])
	require.NoError(t, err)
	require.NoError(t, switchOffErr)
	require.Equal(t, int32(1), model.summaries.Load())
	require.Len(t, results, 1)
	require.Equal(t, "no WHERE", results[0].Payload.Comment, "the review ends with the rules' results, which carry nothing of the AI review")
}

func TestDropAIReview(t *testing.T) {
	t.Parallel()

	rule := dropRule()
	bare := rule.payload()
	rule.earlier = &ruleThread{Comment: "Drops column `legacy_status`.\n\nThe column held 3 values.", Priority: storepb.IssueCommentPayload_ReviewMetadata_P0}
	rule.keepsEarlier = true
	rule.members[0].facts = "The column is empty."
	rule.members[0].folded = []string{"v_orders reads it."}
	rule.members[0].raised = storepb.IssueCommentPayload_ReviewMetadata_P0
	dropAIReview([]*ruleResult{rule})
	require.Equal(t, bare, rule.payload())
}

func TestStatesProblem(t *testing.T) {
	t.Parallel()

	require.False(t, statesProblem("Drops column `legacy_status`.\n\nThe column is empty."))
	require.True(t, statesProblem("Drops column `legacy_status`.\n\nv_orders reads it.\n\nSuggested fix: Drop v_orders first."))

	// A merged result lists its findings, with what the AI review adds
	// indented under each item.
	merged := &ruleResult{statements: mergeAtStatements}
	for line := int32(1); line <= mergeAtStatements; line++ {
		member := *dropRule().members[0]
		member.statement = int(line)
		member.anchor = &storepb.IssueCommentPayload_StatementAnchor{StartPosition: &storepb.Position{Line: line}, EndPosition: &storepb.Position{Line: line}}
		merged.members = append(merged.members, &member)
	}
	merged.members[1].folded = []string{"v_orders reads it." + aiReviewFixLabel + "Drop v_orders first."}
	comment := merged.payload().Comment
	require.NotContains(t, comment, aiReviewFixLabel, "the fix is indented under its item")
	require.True(t, statesProblem(comment))
	merged.earlier = &ruleThread{Comment: comment}
	merged.members[1].folded = nil
	require.False(t, holdsFactsOnly(merged), "a comment that states a problem is written again once the problem is gone")
}

func TestOffersEarlierFactsThatNameNoDatabase(t *testing.T) {
	t.Parallel()

	// Three statements of one rule on the same two databases post as one
	// merged result, with each finding's facts under its item.
	merged := func(facts string) *ruleResult {
		rule := &ruleResult{statements: mergeAtStatements}
		for line := int32(1); line <= mergeAtStatements; line++ {
			member := *dropRule().members[0]
			member.statement = int(line)
			member.targets = []string{"instances/prod/databases/db1", "instances/prod/databases/db2"}
			member.anchor = &storepb.IssueCommentPayload_StatementAnchor{StartPosition: &storepb.Position{Line: line}, EndPosition: &storepb.Position{Line: line}}
			rule.members = append(rule.members, &member)
		}
		rule.members[1].facts = facts
		rule.earlier = &ruleThread{Comment: rule.payload().Comment}
		rule.members[1].facts = ""
		return rule
	}
	require.True(t, offersEarlier(merged("The column is empty.")))
	require.False(t, offersEarlier(merged("db1: The column is empty.")), "facts of db1 say nothing about a problem on db2")
	require.False(t, offersEarlier(merged("")), "the rules' text alone has nothing to compare")
}
