package reviewrun

import (
	"cmp"
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

// mergeAtStatements is how many statements of a unit one rule must report
// before its findings post as one result: a reviewer accepts a class of
// operation once, not one statement at a time.
const mergeAtStatements = 3

// maxMergedItems bounds the findings a merged result lists, so its comment
// stays readable and its payload small; the fingerprint covers every finding.
const maxMergedItems = 50

// ruleFinding is one finding of a unit, resolved to what its result carries.
type ruleFinding struct {
	rule        storepb.ReviewRuleType
	statement   int
	targets     []string
	codes       []string
	message     string
	anchor      *storepb.IssueCommentPayload_StatementAnchor
	fingerprint string
	// facts is what the AI review adds to the finding about its statement on
	// the databases, which the result's comment carries after the finding's
	// own text. It is empty when folded is not.
	facts string
	// folded is the AI review's problems on the finding's operation, each as
	// the text the result's comment carries after the finding's own.
	folded []string
	// raised is the highest priority among those problems that report more
	// than the rule's own condition, or unspecified.
	raised storepb.IssueCommentPayload_ReviewMetadata_Priority
}

// priority is the rule's, or the higher one a problem on the finding's
// operation raises it to.
func (f *ruleFinding) priority() storepb.IssueCommentPayload_ReviewMetadata_Priority {
	priority := rulePriority(f.rule)
	if priorityRank(f.raised) < priorityRank(priority) {
		return f.raised
	}
	return priority
}

// ruleThread is the thread an earlier run left for a rule result that this
// run reports again.
type ruleThread struct {
	Comment  string
	Priority storepb.IssueCommentPayload_ReviewMetadata_Priority
}

// ruleResult is one result the standard rules post: a finding on its own, or
// the findings of one rule that reports mergeAtStatements or more statements.
type ruleResult struct {
	specID      string
	sheetSha256 string
	engine      storepb.Engine
	// members are the result's findings, in statement order when merged.
	members []*ruleFinding
	// statements is how many statements a merged result covers, and zero for
	// a finding on its own.
	statements int
	// earlier is the thread an earlier run left for the result, or nil.
	earlier *ruleThread
	// keepsEarlier is true when the earlier thread's comment already states
	// what this run found, so the result keeps that comment and its priority.
	keepsEarlier bool
}

// payload is the result as it posts: the rule, its priority, every target it
// applies to, its codes, and its fingerprint, with the text of its findings
// and of the AI review's problems on their operations.
func (r *ruleResult) payload() *storepb.IssueCommentPayload {
	var payload *storepb.IssueCommentPayload
	if r.statements > 0 {
		payload = mergedPayload(r.members, r.statements)
	} else {
		payload = findingPayload(r.members[0])
	}
	if r.keepsEarlier && r.earlier != nil {
		payload.Comment, payload.ReviewMetadata.Priority = r.earlier.Comment, r.earlier.Priority
	}
	return payload
}

// ruleText is the result's comment with nothing the AI review adds to it.
func (r *ruleResult) ruleText() string {
	bare := &ruleResult{statements: r.statements}
	for _, member := range r.members {
		clone := *member
		clone.facts, clone.folded = "", nil
		bare.members = append(bare.members, &clone)
	}
	return bare.payload().Comment
}

// ruleResultComments converts results into the review result comments to
// post, each an OPEN root.
func ruleResultComments(projectID string, issueUID int64, results []*ruleResult) []*store.IssueCommentMessage {
	comments := make([]*store.IssueCommentMessage, 0, len(results))
	for _, result := range results {
		comments = append(comments, &store.IssueCommentMessage{
			ProjectID: projectID,
			IssueUID:  issueUID,
			Payload:   result.payload(),
		})
	}
	return comments
}

// reviewResults groups a unit's findings into the results to post. A finding
// is posted on its own, anchored to its range on the spec's sheet, unless its
// rule reports mergeAtStatements or more statements, whatever databases each
// applies to; those findings post as one merged result instead. A finding for
// a rule this server does not know is dropped, as the contract says.
func reviewResults(unit *reviewUnit, sql string, result *review.Result) []*ruleResult {
	findings := ruleFindings(unit, sql, result)

	groups := make(map[storepb.ReviewRuleType][]*ruleFinding)
	statements := make(map[storepb.ReviewRuleType]map[int]bool)
	for _, f := range findings {
		// A finding on the whole change has no statement to merge by.
		if f.statement < 0 {
			continue
		}
		groups[f.rule] = append(groups[f.rule], f)
		if statements[f.rule] == nil {
			statements[f.rule] = make(map[int]bool)
		}
		statements[f.rule][f.statement] = true
	}

	results := make([]*ruleResult, 0, len(findings))
	posted := make(map[storepb.ReviewRuleType]bool)
	for _, f := range findings {
		r := &ruleResult{specID: unit.SpecID, sheetSha256: unit.SheetSha256, engine: unit.Engine, members: []*ruleFinding{f}}
		if f.statement >= 0 && len(statements[f.rule]) >= mergeAtStatements {
			if posted[f.rule] {
				continue
			}
			posted[f.rule] = true
			r.members = slices.Clone(groups[f.rule])
			slices.SortStableFunc(r.members, func(a, b *ruleFinding) int { return cmp.Compare(a.statement, b.statement) })
			r.statements = len(statements[f.rule])
		}
		results = append(results, r)
	}
	return results
}

// ruleFindings resolves a unit's findings in the order the reviewer gave them,
// dropping those of an unknown rule.
func ruleFindings(unit *reviewUnit, sql string, result *review.Result) []*ruleFinding {
	text := review.Index(sql)
	occurrences := make(map[string]int)
	findings := make([]*ruleFinding, 0, len(result.Findings))
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
		findings = append(findings, &ruleFinding{
			rule:        rule,
			statement:   finding.Statement,
			targets:     targets,
			codes:       codes,
			message:     finding.Message,
			anchor:      findingAnchor(text, unit, result, finding),
			fingerprint: fmt.Sprintf("%s/%d", key, occurrences[key]),
		})
	}
	return findings
}

// added is what the AI review adds to a finding's text: each problem it found
// on the finding's operation, or else its facts.
func (f *ruleFinding) added() []string {
	if len(f.folded) > 0 || f.facts == "" {
		return f.folded
	}
	return []string{f.facts}
}

// findingPayload posts one finding as a result: its message, and then what
// the AI review adds to it.
func findingPayload(f *ruleFinding) *storepb.IssueCommentPayload {
	return &storepb.IssueCommentPayload{
		Comment:         strings.Join(append([]string{f.message}, f.added()...), "\n\n"),
		StatementAnchor: f.anchor,
		ReviewMetadata: &storepb.IssueCommentPayload_ReviewMetadata{
			RuleType:    f.rule,
			Priority:    f.priority(),
			Targets:     f.targets,
			Fingerprint: f.fingerprint,
			Codes:       f.codes,
		},
	}
}

// mergedPayload posts one rule's findings, in statement order, as a single
// result, anchored to the first of them, carrying every code they report,
// the highest priority among them, and applying to every database any of them
// applies to. Its fingerprint hashes the set of its findings' fingerprints, so
// a later run keeps the thread only while it reports exactly these findings on
// exactly their databases: accepting the result never covers a statement or a
// database nobody saw in it.
func mergedPayload(members []*ruleFinding, statements int) *storepb.IssueCommentPayload {
	first := members[0]

	priority := first.priority()
	var codes, targets, fingerprints []string
	for _, m := range members {
		if priorityRank(m.priority()) < priorityRank(priority) {
			priority = m.priority()
		}
		for _, code := range m.codes {
			if !slices.Contains(codes, code) {
				codes = append(codes, code)
			}
		}
		targets = append(targets, m.targets...)
		fingerprints = append(fingerprints, m.fingerprint)
	}
	slices.Sort(targets)
	targets = slices.Compact(targets)
	slices.Sort(fingerprints)

	return &storepb.IssueCommentPayload{
		Comment:         mergedComment(members, statements, targets),
		StatementAnchor: first.anchor,
		ReviewMetadata: &storepb.IssueCommentPayload_ReviewMetadata{
			RuleType:    first.rule,
			Priority:    priority,
			Targets:     targets,
			Fingerprint: hashFields(append([]string{"merged"}, fingerprints...)...),
			Codes:       codes,
		},
	}
}

// mergedComment lists the findings by the line each starts on, under a
// summary line that names how many statements they cover. A message's
// summary line goes on its item, followed by the databases the finding is on
// when that is only some of targets, the result's databases. The rest of a
// message, when every finding shares it, is said once after the list, else
// under its own item. What the AI review adds to a finding goes under its
// item too, and a finding with a problem is listed past maxMergedItems, so
// that the limit never hides a problem.
func mergedComment(members []*ruleFinding, statements int, targets []string) string {
	lineOf := func(m *ruleFinding) int32 { return m.anchor.GetStartPosition().GetLine() }
	labels := databaseLabels(targets)
	firstLine, lastLine := lineOf(members[0]), lineOf(members[len(members)-1])

	summaries := make([]string, len(members))
	details := make([]string, len(members))
	shared := true
	for i, m := range members {
		summaries[i], details[i], _ = strings.Cut(strings.TrimSpace(m.message), "\n\n")
		shared = shared && details[i] == details[0]
	}

	var b strings.Builder
	if firstLine == lastLine {
		fmt.Fprintf(&b, "%d statements, on line %d.\n\n", statements, firstLine)
	} else {
		fmt.Fprintf(&b, "%d statements, on lines %d to %d.\n\n", statements, firstLine, lastLine)
	}
	listed := 0
	for i, m := range members {
		if i >= maxMergedItems && len(m.folded) == 0 {
			continue
		}
		listed++
		item := fmt.Sprintf("Line %d: %s", lineOf(m), summaries[i])
		if len(m.targets) > 0 && len(m.targets) < len(targets) {
			item += " Only on " + databaseList(m.targets, labels) + "."
		}
		if !shared && details[i] != "" {
			item += "\n\n" + details[i]
		}
		for _, text := range m.added() {
			item += "\n\n" + text
		}
		// Continuation lines are indented to stay inside the item.
		lines := strings.Split(item, "\n")
		for j := 1; j < len(lines); j++ {
			if lines[j] != "" {
				lines[j] = "  " + lines[j]
			}
		}
		fmt.Fprintf(&b, "- %s\n", strings.Join(lines, "\n"))
	}
	if more := len(members) - listed; more > 0 {
		fmt.Fprintf(&b, "\nAnd %d more.\n", more)
	}
	if shared && details[0] != "" {
		fmt.Fprintf(&b, "\n%s\n", details[0])
	}
	return strings.TrimSuffix(b.String(), "\n")
}

// findingKey identifies a finding by what was judged: the unit, the rule, the
// codes of the conditions it reports, where it stands in its statement, and
// its databases. The message is left out, so rewording it keeps the finding.
func findingKey(unit *reviewUnit, rule storepb.ReviewRuleType, codes []string, body, at string, targets []string) string {
	return hashFields(append([]string{unit.SpecID, unit.Engine.String(), rule.String(), strings.Join(codes, ","), body, at}, targets...)...)
}

// hashFields hashes each field behind its length, so two different lists
// of fields never hash the same bytes.
func hashFields(fields ...string) string {
	h := sha256.New()
	for _, field := range fields {
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
