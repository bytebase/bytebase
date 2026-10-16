package reviewrun

import (
	"context"
	"fmt"
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
		newModel: func(*storepb.AISetting) aireview.Model { return model },
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
	require.Equal(t, int32(2), model.calls.Load())

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
		newModel: func(*storepb.AISetting) aireview.Model { return model },
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
	_, err = stores.UpsertSetting(ctx, &store.SettingMessage{Name: storepb.SettingName_AI, Workspace: "ws", Value: &storepb.AISetting{Enabled: true}})
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
		require.NoError(t, stores.UpsertDBSchema(ctx, "prod", name, schema, &storepb.DatabaseConfig{}, nil))
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

// findingModel answers every review with the same finding, so a run's one
// comment names every database the run reviewed.
type findingModel struct {
	calls atomic.Int32
}

func (m *findingModel) Chat(context.Context, *v1pb.AIChatRequest) (*v1pb.AIChatResponse, error) {
	m.calls.Add(1)
	content := `{"findings": [{"title": "Backfill note", "severity": "P2", "line": 1, "rule": "r", "evidence": "e", "fix": "f"}]}`
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
