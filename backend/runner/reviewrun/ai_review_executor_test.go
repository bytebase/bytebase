package reviewrun

import (
	"context"
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	metadatapb "github.com/bytebase/omni/metadata"
	"github.com/bytebase/omni/review"
	"github.com/stretchr/testify/require"
	"golang.org/x/sync/semaphore"
	"google.golang.org/genproto/googleapis/type/expr"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/testcontainer"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/plugin/aireview"
	"github.com/bytebase/bytebase/backend/store"
)

// fixtureDatabases is how many databases the fixture's database group
// matches.
const fixtureDatabases = 5

// testWaitLimit fails a test whose reviews stop making progress, instead of
// letting it hang.
const testWaitLimit = 30 * time.Second

func TestCISampleBoundsOnlyTheAIReview(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	stores, issues := setupReviewFixture(ctx, t, 1)
	model := &findingModel{}
	executor := &AIReviewExecutor{
		store:    stores,
		newModel: func(*storepb.AISetting, ...aireview.ModelOption) aireview.Model { return model },
		slots:    semaphore.NewWeighted(aiReviewServerConcurrency),
	}
	reviewedDatabases := func() []string {
		t.Helper()
		comments, err := executor.RunOnce(ctx, "p", issues[0])
		require.NoError(t, err)
		require.Len(t, comments, 1, "the same finding on every database is one comment")
		return comments[0].Payload.ReviewMetadata.Targets
	}

	setCISamplingSize(ctx, t, stores, 2)
	require.Len(t, reviewedDatabases(), 2)
	require.Equal(t, int32(4), model.calls.Load(), "a review per database, one merge, and one summary")
	require.Equal(t, int32(1), model.merges.Load())
	require.Equal(t, int32(1), model.summaries.Load())

	var ruleTargets int
	ruleExecutor := &RuleExecutor{store: stores, reviewFuncs: map[storepb.Engine]review.ReviewFunc{
		storepb.Engine_POSTGRES: func(_ context.Context, _ string, _ review.Options, targets []review.Target) (*review.Result, error) {
			ruleTargets += len(targets)
			return &review.Result{}, nil
		},
	}}
	_, err := ruleExecutor.RunOnce(ctx, "p", issues[0])
	require.NoError(t, err)
	require.Equal(t, fixtureDatabases, ruleTargets, "the rule review evaluates every database whatever the sample")

	setCISamplingSize(ctx, t, stores, 0)
	require.Len(t, reviewedDatabases(), fixtureDatabases)
}

func TestAIReviewRunsShareTheServerLimit(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	stores, issues := setupReviewFixture(ctx, t, 2)
	const limit = 3
	total := len(issues) * fixtureDatabases
	require.Less(t, limit, fixtureDatabases, "one run alone could exceed the limit")
	model := &blockingModel{entered: make(chan struct{}, total), release: make(chan struct{})}
	slots := &observedSlots{Weighted: semaphore.NewWeighted(limit), waiting: make(chan struct{}, total)}
	executor := &AIReviewExecutor{
		store:    stores,
		newModel: func(*storepb.AISetting, ...aireview.ModelOption) aireview.Model { return model },
		slots:    slots,
	}

	runErrs := make(chan error, len(issues))
	for _, issueUID := range issues {
		go func() {
			_, err := executor.RunOnce(ctx, "p", issueUID)
			runErrs <- err
		}()
	}
	// Nothing is released yet, so every review either reaches the model or
	// waits for a slot.
	var entered, waiting int
	for entered+waiting < total {
		select {
		case <-model.entered:
			entered++
		case <-slots.waiting:
			waiting++
		case <-time.After(testWaitLimit):
			t.Fatalf("timed out with %d reviews at the model and %d waiting", entered, waiting)
		}
	}
	require.Equal(t, limit, entered, "the reviews of both runs past the limit wait for a slot")
	for range total - limit {
		send(t, model.release)
		receive(t, model.entered)
	}
	for range limit {
		send(t, model.release)
	}
	for range issues {
		require.NoError(t, receive(t, runErrs))
	}
	require.Equal(t, int32(limit), model.maxInFlight.Load())
}

func TestAIReviewKeepsTheThreadOfAProblemReportedAgain(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	stores, issues := setupReviewFixture(ctx, t, 1)
	model := &findingModel{}
	executor := &AIReviewExecutor{
		store:    stores,
		newModel: func(*storepb.AISetting, ...aireview.ModelOption) aireview.Model { return model },
		slots:    semaphore.NewWeighted(aiReviewServerConcurrency),
	}
	run := func() []*store.IssueCommentMessage {
		t.Helper()
		results, err := executor.RunOnce(ctx, "p", issues[0])
		require.NoError(t, err)
		_, err = stores.CreateReviewRun(ctx, "p", issues[0], store.ReviewRunTypeAI)
		require.NoError(t, err)
		claimed, err := stores.ClaimAvailableReviewRuns(ctx, "replica-1")
		require.NoError(t, err)
		require.Len(t, claimed, 1)
		updated, err := stores.CompleteReviewRun(ctx, claimed[0], "replica-1", storepb.ReviewRun_DONE, nil, results)
		require.NoError(t, err)
		require.True(t, updated)
		return results
	}
	roots := func() []*store.IssueCommentMessage {
		t.Helper()
		comments, err := stores.ListIssueComment(ctx, &store.FindIssueCommentMessage{ProjectID: "p", IssueUID: &issues[0], TopLevelOnly: true})
		require.NoError(t, err)
		return slices.DeleteFunc(comments, func(c *store.IssueCommentMessage) bool { return c.Payload.GetReviewMetadata() == nil })
	}

	first := run()
	require.Len(t, first, 1)
	require.NotEmpty(t, first[0].Payload.ReviewMetadata.Fingerprint)
	posted := roots()
	require.Len(t, posted, 1)
	resolved := store.ThreadStateResolved
	require.NoError(t, stores.UpdateIssueComment(ctx, &store.UpdateIssueCommentMessage{ProjectID: "p", ResourceID: posted[0].ResourceID, ThreadState: &resolved}))
	merges, summaries := model.merges.Load(), model.summaries.Load()

	second := run()
	require.Len(t, second, 1)
	require.Equal(t, first[0].Payload.ReviewMetadata.Fingerprint, second[0].Payload.ReviewMetadata.Fingerprint, "the merge matched the problem to the earlier result")
	require.Equal(t, first[0].Payload.Comment, second[0].Payload.Comment)
	require.NotEmpty(t, second[0].Payload.ReviewMetadata.Title)
	require.Equal(t, first[0].Payload.ReviewMetadata.Title, second[0].Payload.ReviewMetadata.Title)
	require.Equal(t, merges+1, model.merges.Load())
	require.Equal(t, summaries, model.summaries.Load(), "a kept problem keeps its text, so nothing is summarized")
	kept := roots()
	require.Len(t, kept, 1, "nothing is posted again")
	require.Equal(t, posted[0].ResourceID, kept[0].ResourceID)
	require.Equal(t, store.ThreadStateResolved, *kept[0].ThreadState, "the person's resolution stays")
	require.False(t, kept[0].Payload.ReviewMetadata.Superseded)
}

// setupReviewFixture seeds project p with AI enabled, a PostgreSQL instance
// holding fixtureDatabases synced databases, a database group matching all of
// them, and the given number of issues, each with a plan whose one spec
// targets the group. It returns the issue UIDs.
func setupReviewFixture(ctx context.Context, t *testing.T, issueCount int) (*store.Store, []int64) {
	t.Helper()
	db, stores, _ := testcontainer.NewMetadataDB(t)
	_, err := db.ExecContext(ctx, `
		INSERT INTO workspace (resource_id) VALUES ('ws');
		INSERT INTO principal (name, email, password_hash) VALUES ('creator', 'creator@example.com', 'unused');
		INSERT INTO project (resource_id, workspace, name) VALUES ('p', 'ws', 'P');
	`)
	require.NoError(t, err)
	_, err = stores.UpsertSetting(ctx, &store.SettingMessage{Name: storepb.SettingName_AI, Workspace: "ws", Value: &storepb.AISetting{Enabled: true, SqlReviewEnabled: true}})
	require.NoError(t, err)
	rules, err := protojson.Marshal(&storepb.ReviewRulePolicy{Rules: []storepb.ReviewRuleType{storepb.ReviewRuleType_SYNTAX}})
	require.NoError(t, err)
	_, err = stores.CreatePolicy(ctx, &store.PolicyMessage{
		Workspace:    "ws",
		ResourceType: storepb.Policy_PROJECT,
		Resource:     common.FormatProject("p"),
		Type:         storepb.Policy_REVIEW_RULE,
		Payload:      string(rules),
		Enforce:      true,
	})
	require.NoError(t, err)

	_, err = stores.CreateInstance(ctx, &store.InstanceMessage{
		ResourceID: "prod",
		Workspace:  "ws",
		Metadata: &storepb.Instance{
			Engine:      storepb.Engine_POSTGRES,
			DataSources: []*storepb.DataSource{{Id: "admin", Type: storepb.DataSourceType_ADMIN}},
		},
	})
	require.NoError(t, err)
	for i := range fixtureDatabases {
		name := fmt.Sprintf("db%d", i+1)
		_, err = stores.UpsertDatabase(ctx, &store.DatabaseMessage{
			ProjectID:    "p",
			InstanceID:   "prod",
			DatabaseName: name,
			Metadata:     &storepb.DatabaseMetadata{Labels: map[string]string{}},
		})
		require.NoError(t, err)
		schema := &metadatapb.DatabaseSchemaMetadata{Name: name, Schemas: []*metadatapb.SchemaMetadata{{Name: "public"}}}
		_, err = stores.UpsertDBSchema(ctx, "prod", name, schema, nil, time.Now())
		require.NoError(t, err)
	}
	_, err = stores.CreateDatabaseGroup(ctx, &store.DatabaseGroupMessage{
		ProjectID:  "p",
		ResourceID: "all",
		Title:      "all",
		Expression: &expr.Expr{Expression: `resource.instance_id == "prod"`},
	})
	require.NoError(t, err)
	sheets, err := stores.CreateSheets(ctx, "p", &store.SheetMessage{Statement: "ALTER TABLE orders ADD COLUMN note text;"})
	require.NoError(t, err)

	var issues []int64
	for i := range issueCount {
		plan, err := stores.CreatePlan(ctx, &store.PlanMessage{
			ProjectID: "p",
			Name:      fmt.Sprintf("plan %d", i),
			Config: &storepb.PlanConfig{
				Specs: []*storepb.PlanConfig_Spec{{
					Id: "spec-1",
					Config: &storepb.PlanConfig_Spec_ChangeDatabaseConfig{
						ChangeDatabaseConfig: &storepb.PlanConfig_ChangeDatabaseConfig{
							Targets:     []string{"projects/p/databaseGroups/all"},
							SheetSha256: sheets[0].Sha256,
						},
					},
				}},
			},
		}, "creator@example.com")
		require.NoError(t, err)
		issue, err := stores.CreateIssue(ctx, &store.IssueMessage{
			ProjectID:    "p",
			CreatorEmail: "creator@example.com",
			Title:        fmt.Sprintf("issue %d", i),
			Type:         storepb.Issue_DATABASE_CHANGE,
			Payload:      &storepb.Issue{},
			PlanUID:      &plan.UID,
		})
		require.NoError(t, err)
		issues = append(issues, issue.UID)
	}
	return stores, issues
}

func setCISamplingSize(ctx context.Context, t *testing.T, stores *store.Store, size int32) {
	t.Helper()
	require.NoError(t, stores.UpdateProjects(ctx, &store.UpdateProjectMessage{
		ResourceID: "p",
		Workspace:  "ws",
		Setting:    &storepb.Project{CiSamplingSize: size},
	}))
}

// summaryDatabaseID finds the database ids of a summary prompt's findings.
var summaryDatabaseID = regexp.MustCompile(`"database":"([a-z2-7]+)"`)

// findingModel answers every review with the same finding, the merge with one
// group of every finding, and the summary with one group of every database,
// so a run's one comment names every database the run reviewed.
type findingModel struct {
	calls     atomic.Int32
	merges    atomic.Int32
	summaries atomic.Int32
	// onSummary runs when the summary call arrives.
	onSummary func()
}

func (m *findingModel) Chat(_ context.Context, request *v1pb.AIChatRequest) (*v1pb.AIChatResponse, error) {
	m.calls.Add(1)
	content := `{"findings": [{"title": "Backfill note", "severity": "P2", "line": 1, "rule": "r", "evidence": "e", "fix": "f"}]}`
	// A review offers the schema tools, and the merge and the summary offer
	// none. A merge numbers the findings, and a summary names the databases
	// by id.
	if len(request.GetToolDefinitions()) == 0 {
		user := request.GetMessages()[1].GetContent()
		var ids []string
		if count := strings.Count(user, `{"id":`); count > 0 {
			m.merges.Add(1)
			for i := range count {
				ids = append(ids, strconv.Itoa(i+1))
			}
			content = fmt.Sprintf(`{"groups": [[%s]]}`, strings.Join(ids, ", "))
		} else {
			m.summaries.Add(1)
			if m.onSummary != nil {
				m.onSummary()
			}
			for _, match := range summaryDatabaseID.FindAllStringSubmatch(user, -1) {
				if id := strconv.Quote(match[1]); !slices.Contains(ids, id) {
					ids = append(ids, id)
				}
			}
			content = fmt.Sprintf(`{"title": "Backfill note", "evidence": "e", "groups": [{"databases": [%s], "facts": ""}], "fix": "f"}`, strings.Join(ids, ", "))
		}
	}
	return &v1pb.AIChatResponse{Content: &content}, nil
}

func TestAIReviewStopsWhenSwitchedOffBeforeItPosts(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	stores, issues := setupReviewFixture(ctx, t, 1)
	var switchOffErr error
	model := &findingModel{onSummary: func() {
		_, switchOffErr = stores.UpsertSetting(ctx, &store.SettingMessage{Name: storepb.SettingName_AI, Workspace: "ws", Value: &storepb.AISetting{Enabled: true}})
	}}
	executor := &AIReviewExecutor{
		store:    stores,
		newModel: func(*storepb.AISetting, ...aireview.ModelOption) aireview.Model { return model },
		slots:    semaphore.NewWeighted(aiReviewServerConcurrency),
	}
	comments, err := executor.RunOnce(ctx, "p", issues[0])
	require.NoError(t, switchOffErr)
	require.Equal(t, int32(1), model.summaries.Load())
	require.ErrorContains(t, err, "AI SQL review is not enabled", "a switch-off during the summary stops the run before it posts")
	require.Empty(t, comments)
}

func TestAIReviewExecutorSortsItsResults(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	stores, issues := setupReviewFixture(ctx, t, 1)
	// One database: no merge and no summary, so each finding is a result.
	setCISamplingSize(ctx, t, stores, 1)
	executor := &AIReviewExecutor{
		store:    stores,
		newModel: func(*storepb.AISetting, ...aireview.ModelOption) aireview.Model { return advisoryFirstModel{} },
		slots:    semaphore.NewWeighted(aiReviewServerConcurrency),
	}
	comments, err := executor.RunOnce(ctx, "p", issues[0])
	require.NoError(t, err)
	var priorities []storepb.IssueCommentPayload_ReviewMetadata_Priority
	for _, comment := range comments {
		priorities = append(priorities, comment.Payload.ReviewMetadata.Priority)
	}
	require.Equal(t, []storepb.IssueCommentPayload_ReviewMetadata_Priority{
		storepb.IssueCommentPayload_ReviewMetadata_P0,
		storepb.IssueCommentPayload_ReviewMetadata_P2,
	}, priorities)
}

func TestAIReviewPostsTheMostImportantProblems(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	stores, issues := setupReviewFixture(ctx, t, 1)
	setCISamplingSize(ctx, t, stores, 2)
	model := &overLimitModel{}
	executor := &AIReviewExecutor{
		store:    stores,
		newModel: func(*storepb.AISetting, ...aireview.ModelOption) aireview.Model { return model },
		slots:    semaphore.NewWeighted(aiReviewServerConcurrency),
	}
	comments, err := executor.RunOnce(ctx, "p", issues[0])
	require.NoError(t, err)
	require.Len(t, comments, maxAIReviewProblems)
	require.Equal(t, storepb.IssueCommentPayload_ReviewMetadata_P0, comments[0].Payload.ReviewMetadata.Priority, "the problem reported last posts, since it is P0")
	require.Len(t, comments[0].Payload.ReviewMetadata.Targets, 2, "a problem on two databases counts once")
	require.Equal(t, int32(maxAIReviewProblems), model.summaries.Load(), "a held problem is not summarized")
}

// overLimitModel reports one problem more than a run posts, the last one P0,
// merges each problem's findings across the databases, and counts the
// summaries.
type overLimitModel struct {
	summaries atomic.Int32
}

func (m *overLimitModel) Chat(_ context.Context, request *v1pb.AIChatRequest) (*v1pb.AIChatResponse, error) {
	if len(request.GetToolDefinitions()) > 0 {
		findings := make([]string, 0, maxAIReviewProblems+1)
		for i := range maxAIReviewProblems + 1 {
			severity := "P2"
			if i == maxAIReviewProblems {
				severity = "P0"
			}
			findings = append(findings, fmt.Sprintf(`{"title": "Problem %d", "severity": %q, "line": 1, "rule": "r", "evidence": "e", "fix": "f"}`, i, severity))
		}
		content := `{"findings": [` + strings.Join(findings, ", ") + `]}`
		return &v1pb.AIChatResponse{Content: &content}, nil
	}
	user := request.GetMessages()[1].GetContent()
	var content string
	if count := strings.Count(user, `{"id":`); count > 0 {
		// The merge numbers the first database's findings before the second's.
		n := count / 2
		groups := make([]string, 0, n)
		for i := 1; i <= n; i++ {
			groups = append(groups, fmt.Sprintf("[%d, %d]", i, i+n))
		}
		content = `{"groups": [` + strings.Join(groups, ", ") + `]}`
	} else {
		m.summaries.Add(1)
		var ids []string
		for _, match := range summaryDatabaseID.FindAllStringSubmatch(user, -1) {
			if id := strconv.Quote(match[1]); !slices.Contains(ids, id) {
				ids = append(ids, id)
			}
		}
		content = fmt.Sprintf(`{"title": "t", "evidence": "e", "groups": [{"databases": [%s], "facts": ""}], "fix": "f"}`, strings.Join(ids, ", "))
	}
	return &v1pb.AIChatResponse{Content: &content}, nil
}

// advisoryFirstModel reports a P2 finding before a P0 one on the same line.
type advisoryFirstModel struct{}

func (advisoryFirstModel) Chat(context.Context, *v1pb.AIChatRequest) (*v1pb.AIChatResponse, error) {
	content := `{"findings": [
		{"title": "Advisory", "severity": "P2", "line": 1, "rule": "r", "evidence": "e", "fix": "f"},
		{"title": "Blocking", "severity": "P0", "line": 1, "rule": "r", "evidence": "e", "fix": "f"}
	]}`
	return &v1pb.AIChatResponse{Content: &content}, nil
}

// observedSlots tells the test when a review has to wait for a slot.
type observedSlots struct {
	*semaphore.Weighted
	waiting chan struct{}
}

func (s *observedSlots) Acquire(ctx context.Context, n int64) error {
	if s.TryAcquire(n) {
		return nil
	}
	s.waiting <- struct{}{}
	return s.Weighted.Acquire(ctx, n)
}

// blockingModel holds every call until the test releases one, and records
// the most calls in flight at once.
type blockingModel struct {
	entered     chan struct{}
	release     chan struct{}
	inFlight    atomic.Int32
	maxInFlight atomic.Int32
}

func (m *blockingModel) Chat(ctx context.Context, _ *v1pb.AIChatRequest) (*v1pb.AIChatResponse, error) {
	now := m.inFlight.Add(1)
	defer m.inFlight.Add(-1)
	for {
		seen := m.maxInFlight.Load()
		if now <= seen || m.maxInFlight.CompareAndSwap(seen, now) {
			break
		}
	}
	m.entered <- struct{}{}
	select {
	case <-m.release:
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	content := `{"findings": []}`
	return &v1pb.AIChatResponse{Content: &content}, nil
}

func receive[T any](t *testing.T, ch <-chan T) T {
	t.Helper()
	select {
	case v := <-ch:
		return v
	case <-time.After(testWaitLimit):
		t.Fatal("timed out waiting for the reviews")
		var zero T
		return zero
	}
}

func send(t *testing.T, ch chan<- struct{}) {
	t.Helper()
	select {
	case ch <- struct{}{}:
	case <-time.After(testWaitLimit):
		t.Fatal("timed out waiting for a model call to release")
	}
}

func TestAIReviewExecutorChecksTheSettingAgain(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	db, stores, _ := testcontainer.NewMetadataDBWithCache(t, true)
	_, err := db.ExecContext(ctx, `
		INSERT INTO workspace (resource_id) VALUES ('ws');
		INSERT INTO principal (name, email, password_hash) VALUES ('creator', 'creator@example.com', 'unused');
		INSERT INTO project (resource_id, workspace, name) VALUES ('p', 'ws', 'P');
	`)
	require.NoError(t, err)

	plan, err := stores.CreatePlan(ctx, &store.PlanMessage{
		ProjectID: "p",
		Name:      "plan",
		Config: &storepb.PlanConfig{
			Specs: []*storepb.PlanConfig_Spec{{
				Id: "spec-1",
				Config: &storepb.PlanConfig_Spec_ChangeDatabaseConfig{
					ChangeDatabaseConfig: &storepb.PlanConfig_ChangeDatabaseConfig{
						Targets: []string{"instances/missing/databases/app"},
					},
				},
			}},
		},
	}, "creator@example.com")
	require.NoError(t, err)
	issue, err := stores.CreateIssue(ctx, &store.IssueMessage{
		ProjectID:    "p",
		CreatorEmail: "creator@example.com",
		Title:        "issue",
		Type:         storepb.Issue_DATABASE_CHANGE,
		Payload:      &storepb.Issue{},
		PlanUID:      &plan.UID,
	})
	require.NoError(t, err)

	setAISetting := func(setting *storepb.AISetting) {
		t.Helper()
		_, err := stores.UpsertSetting(ctx, &store.SettingMessage{
			Name:      storepb.SettingName_AI,
			Workspace: "ws",
			Value:     setting,
		})
		require.NoError(t, err)
	}
	executor := NewAIReviewExecutor(stores)

	setAISetting(&storepb.AISetting{Enabled: true})
	_, err = executor.RunOnce(ctx, "p", issue.UID)
	require.ErrorContains(t, err, "AI SQL review is not enabled")

	// With the switch on the executor goes on to the target, which does not
	// exist in this fixture.
	setAISetting(&storepb.AISetting{Enabled: true, SqlReviewEnabled: true})
	_, err = executor.RunOnce(ctx, "p", issue.UID)
	require.Error(t, err)
	require.NotContains(t, err.Error(), "is not enabled")

	// A switch-off written outside this process leaves the cached copy on.
	_, err = db.ExecContext(ctx, `UPDATE setting SET value = '{"enabled": true}' WHERE workspace = 'ws' AND name = 'AI'`)
	require.NoError(t, err)
	cached, err := stores.GetAISetting(ctx, "ws")
	require.NoError(t, err)
	require.True(t, cached.SqlReviewEnabled)
	_, err = executor.RunOnce(ctx, "p", issue.UID)
	require.ErrorContains(t, err, "AI SQL review is not enabled")
}
