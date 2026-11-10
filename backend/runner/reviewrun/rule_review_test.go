package reviewrun

import (
	"context"
	"errors"
	"fmt"
	"strings"
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
			{Rule: review.RequireWhere, Statement: 0, Message: "UPDATE without WHERE", Targets: []int{0, 1}, Codes: []review.FindingCode{review.MissingWhere}},
			{Rule: review.DisallowDropObject, Statement: 1, Range: review.Range{Start: 31, End: 37}, Message: "DROP TABLE 你好", Targets: []int{1}},
			{Rule: review.PriorBackup, Statement: -1, Message: "backup database missing", Targets: []int{0}, Codes: []review.FindingCode{review.BackupLocationMissing, review.ChangeTooLarge}},
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
	require.Equal(t, []string{"WH101"}, whole.Payload.ReviewMetadata.Codes)
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
	require.Equal(t, []string{"PB103", "PB102"}, change.Payload.ReviewMetadata.Codes, "a finding with several reasons keeps every code, in order")
	require.Empty(t, comments[1].Payload.ReviewMetadata.Codes)

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
// where they stand, and identical statements by their order. A finding on
// the whole change is new once any of the SQL changes.
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

	// A finding on the whole change stands on all of the SQL.
	wholeChange := func(sql string) []string {
		return fingerprintsOf(unit, sql, &review.Result{Findings: []review.Finding{{
			Rule: review.PriorBackup, Statement: -1, Codes: []review.FindingCode{review.BackupLocationMissing}, Targets: []int{0},
		}}})
	}
	require.Equal(t, wholeChange(update), wholeChange(update))
	require.NotEqual(t, wholeChange(update), wholeChange("DELETE FROM orders WHERE id = 2;"), "an edit to the SQL makes a new finding")
}

// lineStatements reviews one statement per line of sql: the statements and
// bodies a finding refers to, and nothing found yet.
func lineStatements(sql string) *review.Result {
	result := &review.Result{}
	start := 0
	for _, line := range strings.SplitAfter(sql, "\n") {
		r := review.Range{Start: start, End: start + len(strings.TrimSuffix(line, "\n"))}
		result.Statements = append(result.Statements, r)
		result.Bodies = append(result.Bodies, r)
		start += len(line)
	}
	return result
}

func TestReviewResultCommentsMergesARepeatedRule(t *testing.T) {
	t.Parallel()
	const sql = "DROP TABLE a;\nDROP TABLE b;\nDROP TABLE c;\nDROP TABLE d;\nUPDATE t SET x = 1;\nUPDATE u SET y = NULL WHERE y = NULL OR z = NULL;"
	unit := groupReviewUnits(nil, []*reviewTarget{
		target("spec-1", "instances/pg/databases/east", storepb.Engine_POSTGRES, false),
		target("spec-1", "instances/pg/databases/west", storepb.Engine_POSTGRES, false),
	})[0]
	drop := func(statement int, message string, code review.FindingCode, targets ...int) review.Finding {
		return review.Finding{Rule: review.DisallowDropObject, Statement: statement, Message: message, Targets: targets, Codes: []review.FindingCode{code}}
	}
	result := lineStatements(sql)
	result.Findings = []review.Finding{
		drop(0, "Drops table `a`.", review.DropsData, 0),
		drop(1, "Drops table `b`.\n\nIt has dependent views.", review.DropsData, 0),
		drop(2, "Drops function `c`.", review.DropsDefinition, 0),
		drop(3, "Drops table `d`.", review.DropsData, 0, 1),
		{Rule: review.RequireWhere, Statement: 4, Message: "No WHERE.", Targets: []int{0}},
		{Rule: review.RequireIsNull, Statement: 5, Message: "`y = NULL` is never true.", Targets: []int{0}},
		{Rule: review.RequireIsNull, Statement: 5, Message: "`z = NULL` is never true.", Targets: []int{0}},
		{Rule: review.RequireIsNull, Statement: 4, Message: "Another statement.", Targets: []int{0}},
		{Rule: review.PriorBackup, Statement: -1, Message: "Backup database missing.", Targets: []int{0}, Codes: []review.FindingCode{review.BackupLocationMissing}},
		{Rule: review.PriorBackup, Statement: -1, Message: "Change too large.", Targets: []int{0}, Codes: []review.FindingCode{review.ChangeTooLarge}},
		{Rule: review.PriorBackup, Statement: -1, Message: "Nothing to back up.", Targets: []int{0}, Codes: []review.FindingCode{review.NothingToBackUp}},
	}

	comments := reviewResultComments("p", 101, unit, sql, result)
	var messages []string
	for _, c := range comments {
		messages = append(messages, c.Payload.Comment)
	}
	require.Equal(t, []string{
		"4 statements, on lines 1 to 4.\n\n- Line 1: Drops table `a`. Only on east.\n- Line 2: Drops table `b`. Only on east.\n\n  It has dependent views.\n- Line 3: Drops function `c`. Only on east.\n- Line 4: Drops table `d`.",
		"No WHERE.",
		"`y = NULL` is never true.",
		"`z = NULL` is never true.",
		"Another statement.",
		"Backup database missing.",
		"Change too large.",
		"Nothing to back up.",
	}, messages, "a rule merges on three statements whatever their databases; two statements, or a finding on the whole change, stay apart")

	merged := comments[0].Payload
	require.Equal(t, storepb.ReviewRuleType_DISALLOW_DROP_OBJECT, merged.ReviewMetadata.RuleType)
	require.Equal(t, storepb.IssueCommentPayload_ReviewMetadata_P1, merged.ReviewMetadata.Priority)
	require.Equal(t, []string{"instances/pg/databases/east", "instances/pg/databases/west"}, merged.ReviewMetadata.Targets, "every database of any finding")
	require.Equal(t, []string{"DO101", "DO102"}, merged.ReviewMetadata.Codes, "every code once, in order")
	require.Equal(t, &storepb.Position{Line: 1, Column: 1}, merged.StatementAnchor.StartPosition, "anchored to the first finding")
	require.NotContains(t, merged.ReviewMetadata.Fingerprint, "/")
}

func TestMergedComment(t *testing.T) {
	t.Parallel()
	unit := groupReviewUnits(nil, []*reviewTarget{target("spec-1", "instances/pg/databases/a", storepb.Engine_POSTGRES, false)})[0]
	commentOf := func(sql string, findings []review.Finding) string {
		result := lineStatements(sql)
		result.Findings = findings
		comments := reviewResultComments("p", 101, unit, sql, result)
		require.Len(t, comments, 1)
		return comments[0].Payload.Comment
	}
	isNull := func(statement int, name string) review.Finding {
		return review.Finding{Rule: review.RequireIsNull, Statement: statement, Message: fmt.Sprintf("`%s = NULL` is never true.\n\nUse `IS NULL` instead.", name), Targets: []int{0}}
	}

	require.Equal(t,
		"3 statements, on lines 1 to 3.\n\n- Line 1: `x = NULL` is never true.\n- Line 2: `y = NULL` is never true.\n- Line 3: `z = NULL` is never true.\n\nUse `IS NULL` instead.",
		commentOf("a;\nb;\nc;", []review.Finding{isNull(2, "z"), isNull(0, "x"), isNull(1, "y")}),
		"listed in statement order; what every message shares follows the list once")

	statements := []review.Range{{Start: 0, End: 13}, {Start: 14, End: 27}, {Start: 28, End: 41}}
	oneLine := &review.Result{Statements: statements, Bodies: statements}
	for i, name := range []string{"a", "b", "c"} {
		oneLine.Findings = append(oneLine.Findings, review.Finding{Rule: review.DisallowDropObject, Statement: i, Message: fmt.Sprintf("Drops table `%s`.", name), Targets: []int{0}})
	}
	comments := reviewResultComments("p", 101, unit, "DROP TABLE a; DROP TABLE b; DROP TABLE c;", oneLine)
	require.Len(t, comments, 1)
	require.True(t, strings.HasPrefix(comments[0].Payload.Comment, "3 statements, on line 1.\n\n"), comments[0].Payload.Comment)

	var lines []string
	var findings []review.Finding
	for i := range maxMergedItems + 10 {
		lines = append(lines, "x;")
		findings = append(findings, isNull(i, fmt.Sprintf("c%d", i)))
	}
	long := commentOf(strings.Join(lines, "\n"), findings)
	require.Equal(t, maxMergedItems, strings.Count(long, "\n- Line "))
	require.Contains(t, long, "- Line 50: `c49 = NULL` is never true.\n\nAnd 10 more.\n\nUse `IS NULL` instead.")
	require.True(t, strings.HasPrefix(long, "60 statements, on lines 1 to 60.\n\n"))
}

func TestMergedResultNamesTargets(t *testing.T) {
	t.Parallel()
	var targets []*reviewTarget
	for _, name := range []string{"tenant_1", "tenant_7", "tenant_9"} {
		targets = append(targets, target("spec-1", "instances/prod/databases/"+name, storepb.Engine_POSTGRES, false))
	}
	unit := groupReviewUnits(nil, targets)[0]
	missing := func(statement int, table string, targets ...int) review.Finding {
		return review.Finding{Rule: review.WalkThrough, Statement: statement, Message: fmt.Sprintf("Table `%s` does not exist.", table), Targets: targets}
	}
	const sql = "CREATE INDEX ON t1 (a);\nCREATE INDEX ON t2 (a);\nCREATE INDEX ON t3 (a);\nCREATE INDEX ON t4 (a);"
	result := lineStatements(sql)
	result.Findings = []review.Finding{missing(0, "t1", 1), missing(1, "t2", 1, 2), missing(2, "t3", 1, 2), missing(3, "t4", 2, 9)}
	comments := reviewResultComments("p", 101, unit, sql, result)
	require.Len(t, comments, 1, "findings on different databases still merge")
	require.Equal(t, []string{"instances/prod/databases/tenant_7", "instances/prod/databases/tenant_9"}, comments[0].Payload.ReviewMetadata.Targets)
	require.Equal(t,
		"4 statements, on lines 1 to 4.\n\n- Line 1: Table `t1` does not exist. Only on tenant_7.\n- Line 2: Table `t2` does not exist.\n- Line 3: Table `t3` does not exist.\n- Line 4: Table `t4` does not exist. Only on tenant_9.",
		comments[0].Payload.Comment, "a finding on every database of the result names none")

	result.Findings = []review.Finding{missing(0, "t1", 1), missing(1, "t2", 1), missing(2, "t3", 9)}
	require.NotContains(t, reviewResultComments("p", 101, unit, sql, result)[0].Payload.Comment, "Line 3: Table `t3` does not exist. Only", "a finding with no database names none")
}

// TestMergedResultFingerprints pins what keeps a merged result's thread
// across runs, on PostgreSQL's own review: exactly the same findings, whatever
// their order or the SQL around them. Any finding added, dropped, or changed
// makes a new result, and so does falling below the merge threshold.
func TestMergedResultFingerprints(t *testing.T) {
	t.Parallel()
	unitOf := func(names ...string) *reviewUnit {
		targets := make([]*reviewTarget, 0, len(names))
		for _, name := range names {
			targets = append(targets, target("spec-1", name, storepb.Engine_POSTGRES, false))
		}
		return groupReviewUnits(nil, targets)[0]
	}
	resultsOf := func(unit *reviewUnit, sql string) []*storepb.IssueCommentPayload {
		t.Helper()
		result, err := reviewFuncs[storepb.Engine_POSTGRES](context.Background(), sql, review.Options{Rules: []review.Rule{review.DisallowDropObject}}, make([]review.Target, len(unit.Targets)))
		require.NoError(t, err)
		var out []*storepb.IssueCommentPayload
		for _, c := range reviewResultComments("p", 101, unit, sql, result) {
			out = append(out, c.Payload)
		}
		return out
	}
	fingerprintsOf := func(unit *reviewUnit, sql string) []string {
		var out []string
		for _, p := range resultsOf(unit, sql) {
			out = append(out, p.ReviewMetadata.Fingerprint)
		}
		return out
	}
	unit := unitOf("instances/pg/databases/a")
	fingerprints := func(sql string) []string { return fingerprintsOf(unit, sql) }

	const drops = "DROP TABLE a;\nDROP TABLE b;\nDROP TABLE c;"
	base := resultsOf(unit, drops)
	require.Len(t, base, 1)
	want := []string{base[0].ReviewMetadata.Fingerprint}

	moved := resultsOf(unit, "SELECT 1;\n\n-- clean up\n"+drops)
	require.Len(t, moved, 1)
	require.Equal(t, want[0], moved[0].ReviewMetadata.Fingerprint, "an edit outside the findings keeps the result")
	require.Equal(t, int32(4), moved[0].StatementAnchor.StartPosition.Line)
	require.True(t, strings.HasPrefix(moved[0].Comment, "3 statements, on lines 4 to 6."), "the kept result takes the new lines")
	require.Equal(t, want, fingerprints("DROP TABLE c;\nDROP TABLE a;\nDROP TABLE b;"), "reordering the statements keeps the result")

	require.NotEqual(t, want, fingerprints(drops+"\nDROP TABLE d;"), "an added finding makes a new result")
	require.NotEqual(t, want, fingerprints("DROP TABLE b;\nDROP TABLE c;\nDROP TABLE d;"), "swapping one finding for another makes a new result")
	require.NotEqual(t, want, fingerprints("DROP TABLE a;\nDROP TABLE b CASCADE;\nDROP TABLE c;"), "an edited statement makes a new result")
	require.NotEqual(t, want, fingerprintsOf(unitOf("instances/pg/databases/a", "instances/pg/databases/b"), drops), "other databases make a new result")

	apart := fingerprints("DROP TABLE a;\nDROP TABLE b;")
	require.Len(t, apart, 2, "below the threshold each finding stands alone")
	require.NotContains(t, apart, want[0])

	repeated := fingerprints("DROP TABLE a;\nDROP TABLE a;\nDROP TABLE a;")
	require.Len(t, repeated, 1, "identical statements count apart")
	require.NotEqual(t, repeated, fingerprints("DROP TABLE a;\nDROP TABLE a;\nDROP TABLE a;\nDROP TABLE a;"))
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
