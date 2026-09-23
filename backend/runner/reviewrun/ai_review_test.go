package reviewrun

import (
	"context"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	metadatapb "github.com/bytebase/omni/metadata"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/aireview"
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
	dropOnTheOtherDatabase := drop
	dropOnTheOtherDatabase.Severity = aireview.SeverityP0
	dropOnTheOtherDatabase.Evidence = "orders_summary and orders_report select legacy_status."
	index := aireview.Finding{
		Title:    "Create the index CONCURRENTLY",
		Severity: aireview.SeverityP2,
		Line:     2,
		Rule:     "An index on a table over 1 million rows is created CONCURRENTLY.",
		Evidence: "orders holds 52,000,000 rows.",
		Fix:      "CREATE INDEX CONCURRENTLY idx_orders_created_at ON orders (created_at);",
	}
	comments := aiReviewComments("p", 7, []*aiReviewResult{
		{Unit: unit("instances/prod-2/databases/db"), Findings: []aireview.Finding{drop}},
		nil,
		{Unit: unit("instances/prod-1/databases/db"), Findings: []aireview.Finding{index, dropOnTheOtherDatabase}},
	})
	require.Len(t, comments, 2)

	merged := comments[0]
	require.Equal(t, "p", merged.ProjectID)
	require.Equal(t, int64(7), merged.IssueUID)
	require.Equal(t, "Keep legacy_status while orders_summary reads it\n\norders_summary selects legacy_status.\n\nRule: No column is dropped while a view reads it.\n\nSuggested fix: Drop the view first.", merged.Payload.Comment)
	anchor := merged.Payload.StatementAnchor
	require.Equal(t, "spec-1", anchor.SpecId)
	require.Equal(t, "sha", anchor.SheetSha256)
	require.Equal(t, int32(4), anchor.StartPosition.Line)
	require.Equal(t, int32(4), anchor.EndPosition.Line)
	metadata := merged.Payload.ReviewMetadata
	require.Equal(t, storepb.ReviewRun_AI, metadata.RunType)
	require.Equal(t, storepb.ReviewRuleType_REVIEW_RULE_TYPE_UNSPECIFIED, metadata.RuleType)
	require.Equal(t, storepb.IssueCommentPayload_ReviewMetadata_P0, metadata.Priority, "the merged finding carries the highest priority among the databases")
	require.Equal(t, []string{"instances/prod-1/databases/db", "instances/prod-2/databases/db"}, metadata.Targets)

	single := comments[1]
	require.Equal(t, int32(2), single.Payload.StatementAnchor.StartPosition.Line)
	require.Equal(t, storepb.IssueCommentPayload_ReviewMetadata_P2, single.Payload.ReviewMetadata.Priority)
	require.Equal(t, []string{"instances/prod-1/databases/db"}, single.Payload.ReviewMetadata.Targets)

	require.Empty(t, aiReviewComments("p", 7, nil))
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
// aiReviewConcurrency reviews in flight, every unit attempted whatever the
// others do, and a panicking review recorded as that unit's error.
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

	results, errs := reviewUnits(context.Background(), units, review)
	require.Len(t, results, 20)
	require.Len(t, errs, 20)
	require.LessOrEqual(t, maxInFlight.Load(), int32(aiReviewConcurrency))
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
}
