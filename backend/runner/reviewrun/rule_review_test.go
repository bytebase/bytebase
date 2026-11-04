package reviewrun

import (
	"context"
	"errors"
	"testing"

	metadatapb "github.com/bytebase/omni/metadata"
	"github.com/bytebase/omni/review"
	"github.com/stretchr/testify/require"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/runner/plancheck"
	"github.com/bytebase/bytebase/backend/store"
)

const testSheet = "0be1f01d6ee8e6f6c6a2ce9b418ba10ea9d16c9b9bfae5548b8fa0e26c04a5e0"

func TestReviewRules(t *testing.T) {
	t.Parallel()
	require.Equal(t, []review.Rule{review.Syntax, review.RequireWhere}, reviewRules(&storepb.ReviewRulePolicy{
		Rules: []storepb.ReviewRuleType{storepb.ReviewRuleType_SYNTAX, storepb.ReviewRuleType_REQUIRE_WHERE},
	}))
	require.Empty(t, reviewRules(nil))

	// Every default rule is one the omni contract names.
	known := map[review.Rule]bool{}
	for _, rule := range []review.Rule{
		review.Syntax, review.WalkThrough, review.OnlineMigration, review.PriorBackup, review.RequireIsNull,
		review.RequireWhere, review.DisallowDropObject, review.DisallowTruncate, review.DisallowDropConstraint,
		review.DisallowRename, review.RequirePrimaryKey,
	} {
		known[rule] = true
	}
	for _, rule := range reviewRules(store.GetDefaultReviewRulePolicy()) {
		require.True(t, known[rule], "rule %s is not in the omni contract", rule)
	}
}

// TestRulePriorityCoversEveryRule fails when a rule is added to
// ReviewRuleType without a priority.
func TestRulePriorityCoversEveryRule(t *testing.T) {
	t.Parallel()
	for number, name := range storepb.ReviewRuleType_name {
		rule := storepb.ReviewRuleType(number)
		if rule == storepb.ReviewRuleType_REVIEW_RULE_TYPE_UNSPECIFIED {
			require.Equal(t, storepb.IssueCommentPayload_ReviewMetadata_PRIORITY_UNSPECIFIED, rulePriority(rule))
			continue
		}
		require.NotEqual(t, storepb.IssueCommentPayload_ReviewMetadata_PRIORITY_UNSPECIFIED, rulePriority(rule), name)
	}
	require.Equal(t, storepb.IssueCommentPayload_ReviewMetadata_P0, rulePriority(storepb.ReviewRuleType_SYNTAX))
	require.Equal(t, storepb.IssueCommentPayload_ReviewMetadata_P1, rulePriority(storepb.ReviewRuleType_DISALLOW_DROP_OBJECT))
}

func target(specID, name string, engine storepb.Engine, ghost bool) *reviewTarget {
	return &reviewTarget{
		Check:  &plancheck.CheckTarget{SpecID: specID, Target: name, SheetSha256: testSheet, EnableGhost: ghost, EnablePriorBackup: !ghost},
		Engine: engine,
		Input:  review.Target{SessionUser: name},
	}
}

func TestGroupReviewUnits(t *testing.T) {
	t.Parallel()
	rules := []review.Rule{review.Syntax}
	units := groupReviewUnits(rules, []*reviewTarget{
		target("spec-1", "instances/pg/databases/a", storepb.Engine_POSTGRES, true),
		target("spec-1", "instances/my/databases/b", storepb.Engine_MYSQL, true),
		target("spec-2", "instances/pg/databases/c", storepb.Engine_POSTGRES, false),
		target("spec-1", "instances/pg/databases/d", storepb.Engine_POSTGRES, true),
	})
	require.Len(t, units, 3, "one unit per (spec, engine), in first-seen order")

	require.Equal(t, "spec-1", units[0].SpecID)
	require.Equal(t, storepb.Engine_POSTGRES, units[0].Engine)
	require.Equal(t, testSheet, units[0].SheetSha256)
	require.Equal(t, review.Options{Rules: rules, Change: review.Change{OnlineMigration: true, MaxBackupSize: maxBackupSize}}, units[0].Options)
	require.Equal(t, []string{"instances/pg/databases/a", "instances/pg/databases/d"}, targetNames(units[0]))

	require.Equal(t, storepb.Engine_MYSQL, units[1].Engine)
	require.Equal(t, []string{"instances/my/databases/b"}, targetNames(units[1]))

	require.Equal(t, "spec-2", units[2].SpecID)
	require.Equal(t, review.Change{PriorBackup: true, MaxBackupSize: maxBackupSize}, units[2].Options.Change)
	require.Equal(t, []string{"instances/pg/databases/c"}, targetNames(units[2]))
}

func TestMarkOtherSpecs(t *testing.T) {
	t.Parallel()
	shared1 := target("spec-1", "instances/pg/databases/a", storepb.Engine_POSTGRES, false)
	own := target("spec-1", "instances/pg/databases/b", storepb.Engine_POSTGRES, false)
	shared2 := target("spec-2", "instances/pg/databases/a", storepb.Engine_POSTGRES, false)
	repeated1 := target("spec-3", "instances/pg/databases/c", storepb.Engine_POSTGRES, false)
	repeated2 := target("spec-3", "instances/pg/databases/c", storepb.Engine_POSTGRES, false)
	markOtherSpecs([]*reviewTarget{shared1, own, shared2, repeated1, repeated2})

	require.True(t, shared1.Input.OtherSpecs)
	require.True(t, shared2.Input.OtherSpecs)
	require.False(t, own.Input.OtherSpecs)
	require.False(t, repeated1.Input.OtherSpecs, "one spec naming a database twice is still one spec")
	require.False(t, repeated2.Input.OtherSpecs)
}

func targetNames(unit *reviewUnit) []string {
	var names []string
	for _, target := range unit.Targets {
		names = append(names, target.Check.Target)
	}
	return names
}

func TestReviewResultComments(t *testing.T) {
	t.Parallel()
	const sql = "UPDATE t SET a = 1;\nDROP TABLE 你好;\nSELECT 1"
	unit := groupReviewUnits(nil, []*reviewTarget{
		target("spec-1", "instances/pg/databases/b", storepb.Engine_POSTGRES, false),
		target("spec-1", "instances/pg/databases/a", storepb.Engine_POSTGRES, false),
	})[0]
	result := &review.Result{
		Statements: []review.Range{{Start: 0, End: 19}, {Start: 20, End: 38}, {Start: 39, End: 47}},
		Findings: []review.Finding{
			{Rule: review.RequireWhere, Statement: 0, Message: "UPDATE without WHERE", Targets: []int{0, 1}},
			{Rule: review.DisallowDropObject, Statement: 1, Range: review.Range{Start: 31, End: 37}, Message: "DROP TABLE 你好", Targets: []int{1}},
			{Rule: review.PriorBackup, Statement: -1, Message: "backup database missing", Targets: []int{0}},
			{Rule: "FUTURE_RULE", Statement: 2, Message: "dropped: unknown to this server", Targets: []int{0}},
			{Rule: review.Syntax, Statement: 2, Range: review.Range{Start: 40, End: 40}, Message: "empty range anchors the line", Targets: []int{0, 7}},
		},
	}

	comments := reviewResultComments("p", 101, unit, sql, result)
	require.Len(t, comments, 4, "the unknown rule's finding is dropped")
	for _, c := range comments {
		require.Equal(t, "p", c.ProjectID)
		require.Equal(t, int64(101), c.IssueUID)
		require.Empty(t, c.CreatorEmail)
		require.Equal(t, storepb.ReviewRun_RULE, c.Payload.ReviewMetadata.RunType)
	}

	whole := comments[0]
	require.Equal(t, "UPDATE without WHERE", whole.Payload.Comment)
	require.Equal(t, storepb.ReviewRuleType_REQUIRE_WHERE, whole.Payload.ReviewMetadata.RuleType)
	require.Equal(t, storepb.IssueCommentPayload_ReviewMetadata_P1, whole.Payload.ReviewMetadata.Priority)
	require.Equal(t, []string{"instances/pg/databases/a", "instances/pg/databases/b"}, whole.Payload.ReviewMetadata.Targets, "targets sorted")
	require.Equal(t, &storepb.IssueCommentPayload_StatementAnchor{
		SpecId: "spec-1", SheetSha256: testSheet,
		StartPosition: &storepb.Position{Line: 1, Column: 1},
		EndPosition:   &storepb.Position{Line: 1, Column: 20},
	}, whole.Payload.StatementAnchor, "a zero range anchors the whole statement")

	ranged := comments[1]
	require.Equal(t, []string{"instances/pg/databases/a"}, ranged.Payload.ReviewMetadata.Targets)
	require.Equal(t, &storepb.Position{Line: 2, Column: 12}, ranged.Payload.StatementAnchor.StartPosition, "columns count code points")
	require.Equal(t, &storepb.Position{Line: 2, Column: 14}, ranged.Payload.StatementAnchor.EndPosition)

	change := comments[2]
	require.Nil(t, change.Payload.StatementAnchor, "a whole-change finding has no anchor")
	require.Equal(t, storepb.IssueCommentPayload_ReviewMetadata_P0, change.Payload.ReviewMetadata.Priority)

	line := comments[3]
	require.Equal(t, []string{"instances/pg/databases/b"}, line.Payload.ReviewMetadata.Targets, "an out-of-range target index is dropped")
	require.Equal(t, &storepb.Position{Line: 3}, line.Payload.StatementAnchor.StartPosition)
	require.Equal(t, &storepb.Position{Line: 3}, line.Payload.StatementAnchor.EndPosition)
}

// TestReviewResultFingerprints pins what tells a finding apart across runs,
// on PostgreSQL's own review. An edit outside its statement, blank lines and
// comments above it included, or a reworded message keeps its fingerprint;
// an edit to the statement, another set of databases, or another spec makes
// it a new finding. Findings of one code in one statement are told apart by
// where they stand, and identical statements by their order.
func TestReviewResultFingerprints(t *testing.T) {
	t.Parallel()
	unitOf := func(specID string, names ...string) *reviewUnit {
		targets := make([]*reviewTarget, 0, len(names))
		for _, name := range names {
			targets = append(targets, target(specID, name, storepb.Engine_POSTGRES, false))
		}
		return groupReviewUnits(nil, targets)[0]
	}
	reviewOf := func(unit *reviewUnit, sql string) *review.Result {
		t.Helper()
		rules := []review.Rule{review.RequireWhere, review.RequireIsNull}
		result, err := reviewFuncs[storepb.Engine_POSTGRES](context.Background(), sql, review.Options{Rules: rules}, make([]review.Target, len(unit.Targets)))
		require.NoError(t, err)
		return result
	}
	fingerprintsOf := func(unit *reviewUnit, sql string, result *review.Result) []string {
		var out []string
		for _, comment := range reviewResultComments("p", 101, unit, sql, result) {
			out = append(out, comment.Payload.ReviewMetadata.Fingerprint)
		}
		return out
	}
	fingerprints := func(unit *reviewUnit, sql string) []string {
		return fingerprintsOf(unit, sql, reviewOf(unit, sql))
	}

	unit := unitOf("spec-1", "instances/pg/databases/a")
	const update = "UPDATE t SET a = 1;"
	base := fingerprints(unit, update)
	require.Len(t, base, 1)
	require.NotEmpty(t, base[0])
	require.Equal(t, base, fingerprints(unit, "SELECT 1;\n\n-- set every row\n"+update), "an edit outside the statement keeps the finding")
	require.NotEqual(t, base, fingerprints(unit, "UPDATE t SET a = 2;"), "an edited statement is a new finding")
	require.NotEqual(t, base, fingerprints(unitOf("spec-1", "instances/pg/databases/a", "instances/pg/databases/b"), update), "other databases make a new finding")
	require.NotEqual(t, base, fingerprints(unitOf("spec-2", "instances/pg/databases/a"), update), "another spec makes a new finding")

	reworded := reviewOf(unit, update)
	reworded.Findings[0].Message = "Reworded."
	require.Equal(t, base, fingerprintsOf(unit, update, reworded), "a reworded message keeps the finding")

	twice := fingerprints(unit, update+"\n"+update)
	require.Len(t, twice, 2)
	require.Equal(t, base[0], twice[0], "the first of identical statements keeps the finding")
	require.NotEqual(t, twice[0], twice[1])

	const nulls = "DELETE FROM t WHERE x = NULL OR y = NULL;"
	both := fingerprints(unit, nulls)
	require.Len(t, both, 2)
	require.NotEqual(t, both[0], both[1])
	// When the schema ends one of two such findings, as it can WALK_THROUGH's,
	// the other keeps its own.
	one := reviewOf(unit, nulls)
	one.Findings = one.Findings[1:]
	require.Equal(t, both[1:], fingerprintsOf(unit, nulls, one))
}

// TestBackupDatabaseExistsPostgres reads the PostgreSQL archive schema from
// the synced metadata; no store lookup is involved, so the executor has none.
func TestBackupDatabaseExistsPostgres(t *testing.T) {
	t.Parallel()
	e := &RuleExecutor{}
	pg := &store.InstanceMessage{Metadata: &storepb.Instance{Engine: storepb.Engine_POSTGRES}}
	withArchive := &metadatapb.DatabaseSchemaMetadata{Schemas: []*metadatapb.SchemaMetadata{{Name: "public"}, {Name: "bbdataarchive"}}}
	withoutArchive := &metadatapb.DatabaseSchemaMetadata{Schemas: []*metadatapb.SchemaMetadata{{Name: "public"}}}

	exists, err := e.backupDatabaseExists(context.Background(), pg, withArchive)
	require.NoError(t, err)
	require.True(t, exists)
	exists, err = e.backupDatabaseExists(context.Background(), pg, withoutArchive)
	require.NoError(t, err)
	require.False(t, exists)

	// An engine without prior backup never has a backup location.
	exists, err = e.backupDatabaseExists(context.Background(), &store.InstanceMessage{Metadata: &storepb.Instance{Engine: storepb.Engine_SNOWFLAKE}}, withArchive)
	require.NoError(t, err)
	require.False(t, exists)
}

func TestSessionUser(t *testing.T) {
	t.Parallel()
	pg := &store.InstanceMessage{Metadata: &storepb.Instance{
		Engine:      storepb.Engine_POSTGRES,
		DataSources: []*storepb.DataSource{{Type: storepb.DataSourceType_READ_ONLY, Username: "ro"}, {Type: storepb.DataSourceType_ADMIN, Username: "admin"}},
	}}
	mysql := &store.InstanceMessage{Metadata: &storepb.Instance{Engine: storepb.Engine_MYSQL, DataSources: pg.Metadata.DataSources}}
	plain := &store.ProjectMessage{Setting: &storepb.Project{}}
	tenant := &store.ProjectMessage{Setting: &storepb.Project{PostgresDatabaseTenantMode: true}}

	require.Equal(t, "admin", sessionUser(plain, pg, "owner"))
	require.Equal(t, "owner", sessionUser(tenant, pg, "owner"), "tenant mode runs the change as the database owner")
	require.Empty(t, sessionUser(plain, mysql, "owner"), "only PostgreSQL has ownership checks")
}

func TestAggregateUnitErrors(t *testing.T) {
	t.Parallel()
	require.NoError(t, aggregateUnitErrors(3, nil))
	err := aggregateUnitErrors(5, []error{errors.New("a"), errors.New("b"), errors.New("c"), errors.New("d")})
	require.EqualError(t, err, "4 of 5 review units failed: a; b; c (+1 more)")

	unit := groupReviewUnits(nil, []*reviewTarget{
		target("spec-1", "instances/ch/databases/a", storepb.Engine_CLICKHOUSE, false),
		target("spec-1", "instances/ch/databases/b", storepb.Engine_CLICKHOUSE, false),
	})[0]
	err = aggregateUnitErrors(2, unit.failures(errors.New("engine CLICKHOUSE has no standard rule reviewer")))
	require.EqualError(t, err, "2 of 2 review units failed: instances/ch/databases/a: engine CLICKHOUSE has no standard rule reviewer; instances/ch/databases/b: engine CLICKHOUSE has no standard rule reviewer")
}
