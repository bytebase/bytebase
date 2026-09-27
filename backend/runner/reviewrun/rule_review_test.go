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
		target("spec-1", "instances/pg/databases/a", storepb.Engine_POSTGRES, false),
		target("spec-1", "instances/pg/databases/b", storepb.Engine_POSTGRES, false),
	})[0]
	err = aggregateUnitErrors(2, unit.failures(errors.New("engine POSTGRES has no standard rule reviewer")))
	require.EqualError(t, err, "2 of 2 review units failed: instances/pg/databases/a: engine POSTGRES has no standard rule reviewer; instances/pg/databases/b: engine POSTGRES has no standard rule reviewer")
}
