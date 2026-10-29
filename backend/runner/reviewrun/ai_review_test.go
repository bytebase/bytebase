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
	"golang.org/x/sync/semaphore"

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

	defaults := &metadatapb.DatabaseSchemaMetadata{CharacterSet: "utf8mb4", Collation: "utf8mb4_bin", SearchPath: "app, public", Owner: "app_owner"}
	require.Equal(t, aireview.Target{Engine: "POSTGRES", Version: "16.2", CharacterSet: "utf8mb4", Collation: "utf8mb4_bin", SearchPath: "app, public", Owner: "app_owner"}, aiReviewTarget(instance, &store.DatabaseMessage{}, defaults))
}

func aiReviewResultOn(target string, findings ...aireview.Finding) *aiReviewResult {
	return &aiReviewResult{
		Unit:     &aiReviewUnit{Check: &plancheck.CheckTarget{SpecID: "spec-1", SheetSha256: "sha", Target: target}},
		Findings: findings,
	}
}

func TestAIReviewComments(t *testing.T) {
	t.Parallel()

	cascade := func(title string, severity aireview.Severity, reason string, evidence ...string) aireview.Finding {
		return aireview.Finding{Title: title, Severity: severity, Line: 1, Rule: "cascades", Reason: reason, Evidence: evidence, Fix: "Archive the churned accounts instead."}
	}
	invoices := aireview.Finding{
		Title:    "Keep the invoices of churned accounts",
		Severity: aireview.SeverityP0,
		Line:     1,
		Rule:     "workspace-1",
		Reason:   "The cascade into orders reaches invoices, which the policy keeps.",
		Evidence: []string{"invoices_order_id_fkey: NO ACTION"},
		Fix:      "Delete no account that has invoices.",
	}
	comments := aiReviewComments("p", 7, []*aiReviewResult{
		aiReviewResultOn("instances/i/databases/tenant_c", cascade("Do not hard-delete churned accounts", aireview.SeverityP0, "Orders go with their accounts.", "orders: 8,661 rows")),
		aiReviewResultOn("instances/i/databases/tenant_a", cascade("Avoid deleting churned accounts", aireview.SeverityP0, "Deleting accounts cascades into orders and users.", "orders_account_id_fkey: ON DELETE CASCADE", "orders: 5,523 rows")),
		nil,
		aiReviewResultOn("instances/i/databases/tenant_b", cascade("Prevent the cascade", aireview.SeverityP0, "The cascade removes orders.", "orders: 5,523 rows"), invoices),
		aiReviewResultOn("instances/i/databases/tenant_d", cascade("Avoid the cascade", aireview.SeverityP1, "A small cascade.", "orders: 12 rows")),
		aiReviewResultOn("instances/i/databases/tenant_e"),
	}, aireview.Rules("Never delete rows from invoices.", ""))
	require.Len(t, comments, 2, "the same rule on a line merges whatever the titles; another rule on the line stays apart")

	merged := comments[0]
	require.Equal(t, "p", merged.ProjectID)
	require.Equal(t, int64(7), merged.IssueUID)
	require.Equal(t, `Avoid deleting churned accounts

Deleting accounts cascades into orders and users.

**Evidence** (`+"`tenant_a`"+`):

- orders_account_id_fkey: ON DELETE CASCADE
- orders: 5,523 rows

**Severity:** P0 on `+"`tenant_a`, `tenant_b`, `tenant_c`"+`; P1 on `+"`tenant_d`"+`.

**Not flagged on:** `+"`tenant_e`"+`.

**Fix:** Archive the churned accounts instead.

**Rule:** The statement deletes or changes rows in other tables through foreign key actions or triggers.`, merged.Payload.Comment, "the most severe database, first by name, leads, and only its evidence shows")
	anchor := merged.Payload.StatementAnchor
	require.Equal(t, "spec-1", anchor.SpecId)
	require.Equal(t, "sha", anchor.SheetSha256)
	require.Equal(t, int32(1), anchor.StartPosition.Line)
	require.Equal(t, int32(1), anchor.EndPosition.Line)
	metadata := merged.Payload.ReviewMetadata
	require.Equal(t, storepb.ReviewRun_AI, metadata.RunType)
	require.Equal(t, storepb.ReviewRuleType_REVIEW_RULE_TYPE_UNSPECIFIED, metadata.RuleType)
	require.Equal(t, storepb.IssueCommentPayload_ReviewMetadata_P0, metadata.Priority)
	require.Equal(t, []string{"instances/i/databases/tenant_a", "instances/i/databases/tenant_b", "instances/i/databases/tenant_c", "instances/i/databases/tenant_d"}, metadata.Targets)

	policy := comments[1]
	require.Equal(t, `Keep the invoices of churned accounts

The cascade into orders reaches invoices, which the policy keeps.

**Evidence** (`+"`tenant_b`"+`):

- invoices_order_id_fkey: NO ACTION

**Not flagged on:** `+"`tenant_a`, `tenant_c`, `tenant_d`, `tenant_e`"+`.

**Fix:** Delete no account that has invoices.

**Rule:** Never delete rows from invoices.`, policy.Payload.Comment)
	require.Equal(t, []string{"instances/i/databases/tenant_b"}, policy.Payload.ReviewMetadata.Targets)

	require.Empty(t, aiReviewComments("p", 7, nil, nil))
}

func TestAIReviewCommentsOnOneDatabase(t *testing.T) {
	t.Parallel()

	finding := aireview.Finding{
		Title:    "Create the index CONCURRENTLY",
		Severity: aireview.SeverityP2,
		Line:     2,
		Rule:     "blocks",
		Reason:   "CREATE INDEX blocks writes to orders until the index is built.",
		Evidence: []string{"orders:  52,000,000\nrows", "no index covers created_at", "c", "d", "e", "past the limit"},
		Fix:      "CREATE INDEX CONCURRENTLY idx_orders_created_at ON orders (created_at);",
	}
	comments := aiReviewComments("p", 7, []*aiReviewResult{aiReviewResultOn("instances/i/databases/db", finding)}, aireview.Rules("", ""))
	require.Len(t, comments, 1)
	require.Equal(t, `Create the index CONCURRENTLY

CREATE INDEX blocks writes to orders until the index is built.

**Evidence**:

- orders: 52,000,000 rows
- no index covers created_at
- c
- d
- e

**Fix:** CREATE INDEX CONCURRENTLY idx_orders_created_at ON orders (created_at);

**Rule:** The statement blocks reads or writes on a large table for a long time, for example with a lock held while the table is rewritten, scanned, or validated, or with one UPDATE or DELETE over many rows.`, comments[0].Payload.Comment, "one database needs no name; a fact keeps to one line, and at most five show")
	require.Equal(t, storepb.IssueCommentPayload_ReviewMetadata_P2, comments[0].Payload.ReviewMetadata.Priority)
}

func TestAIReviewCommentsKeepARepeatedRuleApart(t *testing.T) {
	t.Parallel()

	fails := func(evidence string) aireview.Finding {
		return aireview.Finding{Title: "t", Severity: aireview.SeverityP0, Line: 2, Rule: "fails", Reason: "r", Evidence: []string{evidence}, Fix: "f"}
	}
	comments := aiReviewComments("p", 7, []*aiReviewResult{
		aiReviewResultOn("instances/i/databases/a", fails("invoices references orders."), fails("refunds references orders.")),
		aiReviewResultOn("instances/i/databases/b", fails("invoices references orders.")),
	}, aireview.Rules("", ""))
	require.Len(t, comments, 2, "one database's two findings never merge with each other")
	require.Equal(t, []string{"instances/i/databases/a", "instances/i/databases/b"}, comments[0].Payload.ReviewMetadata.Targets)
	require.Equal(t, []string{"instances/i/databases/a"}, comments[1].Payload.ReviewMetadata.Targets)
}

func TestAIReviewCommentTextCountsManyDatabases(t *testing.T) {
	t.Parallel()

	target := func(name string) string { return "instances/i/databases/" + name }
	var reviewed []string
	occurrence := func(name string, severity aireview.Severity) aiReviewOccurrence {
		return aiReviewOccurrence{Target: target(name), Finding: aireview.Finding{Title: "t", Severity: severity, Reason: "r", Evidence: []string{"e"}, Fix: "f"}}
	}
	occurrences := []aiReviewOccurrence{occurrence("a", aireview.SeverityP0)}
	for _, name := range []string{"b", "c", "d", "e", "f", "g"} {
		occurrences = append(occurrences, occurrence(name, aireview.SeverityP1))
	}
	for _, name := range []string{"a", "b", "c", "d", "e", "f", "g", "h", "i", "j", "k", "l", "m"} {
		reviewed = append(reviewed, target(name))
	}
	require.Equal(t, "t\n\nr\n\n**Evidence** (`a`):\n\n- e\n\n**Severity:** P0 on `a`; P1 on 6 databases.\n\n**Not flagged on:** 6 databases.\n\n**Fix:** f\n\n**Rule:** rule text", aiReviewCommentText(occurrences, "rule text", reviewed))
}

func TestCodeSpan(t *testing.T) {
	t.Parallel()

	require.Equal(t, "`tenant_001`", codeSpan("tenant_001"))
	require.Equal(t, "odd`name", codeSpan("odd`name"), "a backtick would end the span")
}

func TestDatabaseLabels(t *testing.T) {
	t.Parallel()

	require.Equal(t, map[string]string{
		"instances/prod-1/databases/orders": "prod-1/orders",
		"instances/prod-2/databases/orders": "prod-2/orders",
		"instances/prod-1/databases/users":  "users",
		"not a database":                    "not a database",
	}, databaseLabels([]string{
		"instances/prod-1/databases/orders",
		"instances/prod-2/databases/orders",
		"instances/prod-1/databases/users",
		"not a database",
	}))
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
