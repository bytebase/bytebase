package reviewrun

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"slices"
	"strings"

	"github.com/bytebase/omni/review"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/runner/plancheck"
	"github.com/bytebase/bytebase/backend/store"
)

// reviewTarget is one database of the plan with the inputs its engine's
// Review needs, resolved by the executor.
type reviewTarget struct {
	Check  *plancheck.CheckTarget
	Engine storepb.Engine
	Input  review.Target
}

// reviewUnit is one call to an engine's Review: a spec's SQL against every
// database of one engine the spec addresses. Findings refer to Targets by
// index, in the order they are passed to Review.
type reviewUnit struct {
	SpecID      string
	SheetSha256 string
	Engine      storepb.Engine
	Options     review.Options
	Targets     []*reviewTarget
}

// reviewRules names the rules a policy switches on, by the enum name the
// omni contract maps on.
func reviewRules(policy *storepb.ReviewRulePolicy) []review.Rule {
	rules := make([]review.Rule, 0, len(policy.GetRules()))
	for _, rule := range policy.GetRules() {
		rules = append(rules, review.Rule(rule.String()))
	}
	return rules
}

// markOtherSpecs sets OtherSpecs on each target whose database another spec
// of the plan also changes. Such specs run in no fixed order (only release
// tasks are sequential), so neither may assume it meets the synced schema.
// Check.Target names a database in one form only: ResolveDatabaseTarget
// rejects the other.
func markOtherSpecs(targets []*reviewTarget) {
	specs := make(map[string]map[string]bool)
	for _, target := range targets {
		if specs[target.Check.Target] == nil {
			specs[target.Check.Target] = make(map[string]bool)
		}
		specs[target.Check.Target][target.Check.SpecID] = true
	}
	for _, target := range targets {
		target.Input.OtherSpecs = len(specs[target.Check.Target]) > 1
	}
}

// groupReviewUnits folds the targets into one unit per (spec, engine), in
// first-seen order. The change's requested behaviors are read from the first
// target: they come from the spec and its sheet, so every target of a spec
// agrees.
func groupReviewUnits(rules []review.Rule, targets []*reviewTarget) []*reviewUnit {
	type key struct {
		specID string
		engine storepb.Engine
	}
	units := make(map[key]*reviewUnit)
	var ordered []*reviewUnit
	for _, target := range targets {
		k := key{specID: target.Check.SpecID, engine: target.Engine}
		unit, ok := units[k]
		if !ok {
			unit = &reviewUnit{
				SpecID:      target.Check.SpecID,
				SheetSha256: target.Check.SheetSha256,
				Engine:      target.Engine,
				Options: review.Options{
					Rules: rules,
					Change: review.Change{
						OnlineMigration: target.Check.EnableGhost,
						PriorBackup:     target.Check.EnablePriorBackup,
						MaxBackupSize:   maxBackupSize,
					},
				},
			}
			units[k] = unit
			ordered = append(ordered, unit)
		}
		unit.Targets = append(unit.Targets, target)
	}
	return ordered
}

// rulePriority is what resolving a finding's thread means: P0 rules report
// SQL that is wrong, P1 rules report operations a person must accept. Every
// ReviewRuleType has a line here; TestRulePriorityCoversEveryRule holds it.
func rulePriority(rule storepb.ReviewRuleType) storepb.IssueCommentPayload_ReviewMetadata_Priority {
	switch rule {
	case storepb.ReviewRuleType_SYNTAX,
		storepb.ReviewRuleType_WALK_THROUGH,
		storepb.ReviewRuleType_ONLINE_MIGRATION,
		storepb.ReviewRuleType_PRIOR_BACKUP,
		storepb.ReviewRuleType_REQUIRE_IS_NULL:
		return storepb.IssueCommentPayload_ReviewMetadata_P0
	case storepb.ReviewRuleType_REQUIRE_WHERE,
		storepb.ReviewRuleType_DISALLOW_DROP_OBJECT,
		storepb.ReviewRuleType_DISALLOW_TRUNCATE,
		storepb.ReviewRuleType_DISALLOW_DROP_CONSTRAINT,
		storepb.ReviewRuleType_DISALLOW_RENAME,
		storepb.ReviewRuleType_REQUIRE_PRIMARY_KEY:
		return storepb.IssueCommentPayload_ReviewMetadata_P1
	default:
		return storepb.IssueCommentPayload_ReviewMetadata_PRIORITY_UNSPECIFIED
	}
}

// reviewResultComments converts a unit's findings into the review result
// comments to post: one OPEN root per finding, anchored to the finding's
// range on the spec's sheet, naming the rule, its priority, every target the
// finding applies to, its codes, and its fingerprint. A finding for a rule
// this server does not know is dropped, as the contract says.
func reviewResultComments(projectID string, issueUID int64, unit *reviewUnit, sql string, result *review.Result) []*store.IssueCommentMessage {
	text := review.Index(sql)
	occurrences := make(map[string]int)
	comments := make([]*store.IssueCommentMessage, 0, len(result.Findings))
	for _, finding := range result.Findings {
		rule := storepb.ReviewRuleType(storepb.ReviewRuleType_value[string(finding.Rule)])
		if rule == storepb.ReviewRuleType_REVIEW_RULE_TYPE_UNSPECIFIED {
			continue
		}
		targets := make([]string, 0, len(finding.Targets))
		for _, i := range finding.Targets {
			if i < 0 || i >= len(unit.Targets) {
				continue
			}
			targets = append(targets, unit.Targets[i].Check.Target)
		}
		slices.Sort(targets)
		codes := make([]string, 0, len(finding.Codes))
		for _, code := range finding.Codes {
			codes = append(codes, string(code))
		}
		// Identical statements give equal keys; their order tells them apart.
		body, at := findingPlace(sql, result, finding)
		key := findingKey(unit, rule, codes, body, at, targets)
		occurrences[key]++
		comments = append(comments, &store.IssueCommentMessage{
			ProjectID: projectID,
			IssueUID:  issueUID,
			Payload: &storepb.IssueCommentPayload{
				Comment:         finding.Message,
				StatementAnchor: findingAnchor(text, unit, result, finding),
				ReviewMetadata: &storepb.IssueCommentPayload_ReviewMetadata{
					RunType:     storepb.ReviewRun_RULE,
					RuleType:    rule,
					Priority:    rulePriority(rule),
					Targets:     targets,
					Fingerprint: fmt.Sprintf("%s/%d", key, occurrences[key]),
					Codes:       codes,
				},
			},
		})
	}
	return comments
}

// findingKey identifies a finding by what was judged: the unit, the rule, the
// codes of the conditions it reports, where it stands in its statement, and
// its databases. The message is left out, so rewording it keeps the finding.
func findingKey(unit *reviewUnit, rule storepb.ReviewRuleType, codes []string, body, at string, targets []string) string {
	h := sha256.New()
	for _, field := range append([]string{unit.SpecID, unit.Engine.String(), rule.String(), strings.Join(codes, ","), body, at}, targets...) {
		_, _ = h.Write(binary.BigEndian.AppendUint64(nil, uint64(len(field))))
		_, _ = h.Write([]byte(field))
	}
	return hex.EncodeToString(h.Sum(nil))
}

// findingPlace is where a finding stands: the body of its statement, and its
// range within that body. A body leaves out the blank lines and comments
// around the statement, so an edit to them, or anywhere else outside the
// statement, keeps the finding, while an edit to the statement makes it a new
// one. A finding on the whole change stands on all of the SQL, so any edit to
// it makes a new one; a finding on the whole statement has no range.
func findingPlace(sql string, result *review.Result, finding review.Finding) (body, at string) {
	if finding.Statement < 0 {
		return sql, ""
	}
	if finding.Statement >= len(result.Bodies) {
		return "", ""
	}
	b := result.Bodies[finding.Statement]
	if b.Start < 0 || b.Start > b.End || b.End > len(sql) {
		return "", ""
	}
	if finding.Range == (review.Range{}) {
		return sql[b.Start:b.End], ""
	}
	return sql[b.Start:b.End], fmt.Sprintf("%d-%d", finding.Range.Start-b.Start, finding.Range.End-b.Start)
}

// findingAnchor anchors a finding on the spec's sheet: its own range, else
// its statement's range, else nothing for a finding that addresses the whole
// change. An empty range anchors the whole line it starts on.
func findingAnchor(text *review.Text, unit *reviewUnit, result *review.Result, finding review.Finding) *storepb.IssueCommentPayload_StatementAnchor {
	if finding.Statement < 0 {
		return nil
	}
	r := finding.Range
	if r == (review.Range{}) && finding.Statement < len(result.Statements) {
		r = result.Statements[finding.Statement]
	}
	startLine, startColumn := text.Position(r.Start)
	endLine, endColumn := text.Position(r.End)
	anchor := &storepb.IssueCommentPayload_StatementAnchor{
		SpecId:      unit.SpecID,
		SheetSha256: unit.SheetSha256,
	}
	if endLine < startLine || (endLine == startLine && endColumn <= startColumn) {
		anchor.StartPosition = &storepb.Position{Line: int32(startLine)}
		anchor.EndPosition = &storepb.Position{Line: int32(startLine)}
		return anchor
	}
	anchor.StartPosition = &storepb.Position{Line: int32(startLine), Column: int32(startColumn)}
	anchor.EndPosition = &storepb.Position{Line: int32(endLine), Column: int32(endColumn)}
	return anchor
}
