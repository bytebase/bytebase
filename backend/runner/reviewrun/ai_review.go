package reviewrun

import (
	"cmp"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"log/slog"
	"maps"
	"slices"
	"strconv"
	"strings"

	metadatapb "github.com/bytebase/omni/metadata"
	"github.com/bytebase/omni/review"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"golang.org/x/sync/errgroup"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/log"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/aireview"
	"github.com/bytebase/bytebase/backend/plugin/parser/base"
	"github.com/bytebase/bytebase/backend/runner/plancheck"
	"github.com/bytebase/bytebase/backend/store"
)

const (
	// maxAIReviewSheetBytes bounds the sheet one review reads whole.
	maxAIReviewSheetBytes = 256 << 10
	// aiReviewRunConcurrency caps the reviews of one run in flight at the model.
	aiReviewRunConcurrency = 8
	// aiReviewServerConcurrency caps the reviews in flight across every run on
	// the server, so runs side by side cannot multiply the load on the provider.
	aiReviewServerConcurrency = 16
	// maxAIReviewProblems is how many open problems a run posts before it holds
	// back new ones, so that the ones that matter are not lost in a long list.
	maxAIReviewProblems = 10
)

// maxFactSummaries is how many rule findings of one run get a model call to
// write their facts across databases. A finding past it shows the facts of
// one database.
const maxFactSummaries = 20

// reviewSlots is the limit on reviews in flight that every run of an executor
// shares. The server uses a *semaphore.Weighted.
type reviewSlots interface {
	Acquire(ctx context.Context, n int64) error
	Release(n int64)
}

// aiReviewUnit is one AI review: a spec's sheet against one database. It
// holds no schema, which the review loads when it starts.
type aiReviewUnit struct {
	Check     *plancheck.CheckTarget
	Statement string
	// Policies are the AI review policy texts the unit is reviewed under.
	Policies []string
	// Engine is set by the review once it resolves the database.
	Engine storepb.Engine
}

// aiReviewResult is the findings of one completed unit, and the facts it adds
// to the unit's standard findings.
type aiReviewResult struct {
	Unit     *aiReviewUnit
	Findings []aireview.Finding
	Facts    []aireview.Fact
}

// aiReviewFinding is one finding of one database.
type aiReviewFinding struct {
	Unit    *aiReviewUnit
	Finding aireview.Finding
}

// aiReviewProblem is the findings that report one problem, which one comment
// posts.
type aiReviewProblem struct {
	Findings []*aiReviewFinding
	// Block is the lines of the statement the findings are on, or of the
	// whole sheet when BlockIsStatement is false because the sheet could not
	// be split.
	Block            aireview.LineRange
	BlockIsStatement bool
	// Reviewed is every database that reviewed the problem's sheet.
	Reviewed []string
	// Summary is the comment the model wrote for a problem that more than one
	// database reports, or nil.
	Summary *aireview.Summary
	// Segment identifies the statement the findings are on, and starts the
	// fingerprint of a problem that keeps no earlier result.
	Segment string
	// Earlier is the result an earlier run posted that the problem keeps, or
	// nil.
	Earlier *aiReviewEarlier
	// Held is true when the run holds the problem back. It posts no comment,
	// but its databases still flagged its statement.
	Held bool
	// Rule is the standard rule finding whose operation the merge put the
	// problem on, or nil. Such a problem posts no comment of its own: its text
	// joins the comment of the finding's result.
	Rule *ruleMember
	// Unchanged is true when the comment an earlier run left on the result of
	// Rule already states the problem, so the result keeps that comment.
	Unchanged bool
}

// ruleMember is one finding of a rule result, which the merge can put the AI
// review's problems with.
type ruleMember struct {
	result  *ruleResult
	finding *ruleFinding
}

// aiReviewBlock is a statement of a sheet under review.
type aiReviewBlock struct {
	specID      string
	sheetSha256 string
	block       aireview.LineRange
}

// aiReviewSheet is a spec's sheet under review.
type aiReviewSheet struct {
	specID      string
	sheetSha256 string
}

// aiReviewEarlier is a result an earlier run posted on a spec under review: a
// current root of the AI reviewer, which a problem on the same statement can
// keep.
type aiReviewEarlier struct {
	// Fingerprint starts with the segment of the statement the result was
	// posted on.
	Fingerprint string
	// Resolved is true when a person resolved the thread, so that keeping it
	// lets the rollout past the problem.
	Resolved bool
	Priority storepb.IssueCommentPayload_ReviewMetadata_Priority
	// Title and Comment are what the result was posted with, which a kept
	// result keeps.
	Title   string
	Comment string
	// Sheet is the text of the sheet the result is anchored to, SheetSha256
	// its hash, and Line the anchored line.
	Sheet       string
	SheetSha256 string
	Line        int
}

// aiReviewStatements is the statements of a sheet, as merge blocks and their
// segments.
type aiReviewStatements struct {
	Blocks []aireview.LineRange
	// Split is false when the sheet could not be split, and its one block is
	// the whole sheet.
	Split bool
	// BodyStart is the first line of each block's statement body.
	BodyStart []int
	// Segments identifies each block's statement across runs: the hash of the
	// spec, the statement's body, and its ordinal among equal bodies, so that
	// an edit to the statement changes it and an edit anywhere else does not.
	Segments []string
}

// mergeFunc asks the model which findings report the same problem.
type mergeFunc func(context.Context, *aireview.MergeRequest) (*aireview.MergeResult, error)

// summarizeFunc asks the model to write the comment of a problem that more
// than one database reports.
type summarizeFunc func(context.Context, *aireview.SummaryRequest) (*aireview.Summary, error)

// summarizeFactsFunc asks the model to write the note of a rule finding from
// the facts more than one database adds to it.
type summarizeFactsFunc func(context.Context, *aireview.FactsRequest) (*aireview.FactsSummary, error)

// standardFinding is a rule finding as the reviews of its sheet know it.
type standardFinding struct {
	ID      string
	Result  *ruleResult
	Finding *ruleFinding
}

// standardFindings gives each rule finding on a statement an id that holds
// for every review of its sheet, by sheet.
func standardFindings(rules []*ruleResult) map[aiReviewSheet][]*standardFinding {
	bySheet := make(map[aiReviewSheet][]*standardFinding)
	for _, rule := range rules {
		key := aiReviewSheet{specID: rule.specID, sheetSha256: rule.sheetSha256}
		for _, finding := range rule.members {
			// A finding on the whole change has no statement to look up.
			if finding.anchor == nil {
				continue
			}
			bySheet[key] = append(bySheet[key], &standardFinding{
				ID:      fmt.Sprintf("S%d", len(bySheet[key])+1),
				Result:  rule,
				Finding: finding,
			})
		}
	}
	return bySheet
}

// standardFindingsOn returns the findings of a sheet that are on target, as a
// review of that database is told them.
func standardFindingsOn(findings []*standardFinding, target string) []aireview.StandardFinding {
	var on []aireview.StandardFinding
	for _, f := range findings {
		if slices.Contains(f.Finding.targets, target) {
			on = append(on, f.prompt())
		}
	}
	return on
}

func (f *standardFinding) prompt() aireview.StandardFinding {
	return aireview.StandardFinding{
		ID:    f.ID,
		Line:  int(f.Finding.anchor.GetStartPosition().GetLine()),
		Rule:  f.Finding.rule.String(),
		Codes: f.Finding.codes,
		Text:  f.Finding.message,
	}
}

// statesProblem reports whether the comment of a rule result holds a problem
// of the AI review, which always ends in its fix. A merged result indents the
// fix under its item.
func statesProblem(comment string) bool {
	label := strings.TrimLeft(aiReviewFixLabel, "\n")
	for _, line := range strings.Split(comment, "\n") {
		if strings.HasPrefix(strings.TrimLeft(line, " "), label) {
			return true
		}
	}
	return false
}

// dropAIReview leaves the rule results as the standard rules gave them, for a
// review that ends without the AI review's results.
func dropAIReview(rules []*ruleResult) {
	for _, rule := range rules {
		rule.keepsEarlier = false
		for _, member := range rule.members {
			member.facts, member.folded = "", nil
			member.raised = storepb.IssueCommentPayload_ReviewMetadata_PRIORITY_UNSPECIFIED
		}
	}
}

// holdsFactsOnly reports whether the comment an earlier run left on a rule
// result states facts and no problem, under every line of the rules' text as
// it is now. Unlike canKeepEarlier, it lets the facts name the one database
// they came from, since facts say nothing about which databases have a problem.
func holdsFactsOnly(rule *ruleResult) bool {
	if rule.earlier == nil || statesProblem(rule.earlier.Comment) {
		return false
	}
	text := rule.ruleText()
	return rule.earlier.Comment != text && holdsLines(rule.earlier.Comment, text)
}

// keepFactsOnlyComments keeps the comment an earlier run left on each rule
// result that no problem of this run is on, when that comment states facts
// and no problem: facts stay as they were first written, so an unchanged
// statement keeps its text and costs no model call. A comment that states a
// problem is written again, since the problem is gone.
func keepFactsOnlyComments(rules []*ruleResult, problems []*aiReviewProblem) {
	hasProblem := make(map[*ruleResult]bool)
	for _, problem := range problems {
		if problem.Rule != nil {
			hasProblem[problem.Rule.result] = true
		}
	}
	for _, rule := range rules {
		if !hasProblem[rule] && holdsFactsOnly(rule) {
			rule.keepsEarlier = true
		}
	}
}

// addRuleFacts sets the facts of each rule finding from what the reviews
// added to it, for the results that write their comment again. A finding
// that holds a problem of this run gets none, since the problem's text
// already says what its statement touches. One database's facts are the
// text. The facts of several go to summarize, at most aiReviewRunConcurrency
// at a time and maxFactSummaries in a run, and a finding whose summary fails
// or is over the limit shows the facts of its first database and counts the
// rest. Facts from only part of the finding's databases name the databases
// they are from, since they say nothing about the others. The error is
// ctx's, when it ends.
func addRuleFacts(ctx context.Context, summarize summarizeFactsFunc, findings map[aiReviewSheet][]*standardFinding, results []*aiReviewResult, sheets map[string]string) error {
	facts := make(map[*standardFinding][]aireview.DatabaseFact)
	for _, result := range results {
		if result == nil {
			continue
		}
		check := result.Unit.Check
		for _, fact := range result.Facts {
			for _, f := range findings[aiReviewSheet{specID: check.SpecID, sheetSha256: check.SheetSha256}] {
				if f.ID == fact.Standard && slices.Contains(f.Finding.targets, check.Target) {
					facts[f] = append(facts[f], aireview.DatabaseFact{Database: check.Target, Text: fact.Text})
				}
			}
		}
	}

	var group errgroup.Group
	group.SetLimit(aiReviewRunConcurrency)
	summaries := 0
	// The findings are walked in id order, so the same ones get a summary on
	// every run.
	for _, key := range slices.SortedFunc(maps.Keys(findings), func(a, b aiReviewSheet) int {
		return cmp.Or(cmp.Compare(a.specID, b.specID), cmp.Compare(a.sheetSha256, b.sheetSha256))
	}) {
		for _, f := range findings[key] {
			added := facts[f]
			if len(added) == 0 || f.Result.keepsEarlier || len(f.Finding.folded) > 0 {
				continue
			}
			slices.SortStableFunc(added, func(a, b aireview.DatabaseFact) int { return cmp.Compare(a.Database, b.Database) })
			labels := databaseLabels(f.Finding.targets)
			scope := ""
			if len(added) < len(f.Finding.targets) {
				from := make([]string, 0, len(added))
				for _, fact := range added {
					from = append(from, fact.Database)
				}
				scope = databaseList(from, labels) + ": "
			}
			if len(added) == 1 {
				f.Finding.facts = scope + added[0].Text
				continue
			}
			// The fallback stands unless the summary replaces it.
			f.Finding.facts = fmt.Sprintf("%s: %s\n\n%d more databases added facts.", labels[added[0].Database], added[0].Text, len(added)-1)
			if summaries >= maxFactSummaries {
				continue
			}
			summaries++
			statement := sheets[key.sheetSha256]
			line := int(f.Finding.anchor.GetStartPosition().GetLine())
			block := aireview.LineRange{Start: line, End: max(line, int(f.Finding.anchor.GetEndPosition().GetLine()))}
			if blocks, split := statementBlocks(f.Result.engine, statement); split {
				if i := slices.IndexFunc(blocks, func(b aireview.LineRange) bool { return line >= b.Start && line <= b.End }); i >= 0 {
					block = blocks[i]
				}
			}
			group.Go(func() error {
				// The scheduler's recovery covers only the goroutine that called
				// RunOnce.
				defer func() {
					if r := recover(); r != nil {
						slog.Error("AI review facts PANIC RECOVER", slog.Any("panic", r), log.BBStack("panic-stack"))
					}
				}()
				summary, err := summarize(ctx, &aireview.FactsRequest{Statement: statement, Block: block, Finding: f.prompt(), Facts: added})
				if err != nil {
					if ctx.Err() == nil {
						slog.Warn("AI review could not summarize the facts of a rule finding, posting those of one database",
							slog.String("spec", key.specID),
							slog.Int("databases", len(added)),
							log.BBError(err))
					}
					return nil
				}
				f.Finding.facts = factsText(summary, labels, scope)
				return nil
			})
		}
	}
	_ = group.Wait()
	return ctx.Err()
}

// factsText renders a facts summary as the note under a rule finding: what
// every database shares, then one line for each group of databases with the
// facts that set it apart. One group needs no line. scope names the
// summary's databases when they are only part of the finding's, and is empty
// otherwise, since the thread names the finding's databases.
func factsText(summary *aireview.FactsSummary, labels map[string]string, scope string) string {
	if len(summary.Groups) == 1 {
		return scope + strings.TrimSpace(summary.Shared+" "+summary.Groups[0].Facts)
	}
	var lines []string
	for _, group := range summary.Groups {
		line := "- " + databaseList(group.Databases, labels)
		if group.Facts != "" {
			line += ": " + group.Facts
		}
		lines = append(lines, line)
	}
	if summary.Shared == "" {
		return strings.Join(lines, "\n")
	}
	return scope + summary.Shared + "\n\n" + strings.Join(lines, "\n")
}

// aiReviewTarget states what the prompt says about a database: the engine,
// its version, the environment, the database's defaults and owner, and every
// schema with its object count.
func aiReviewTarget(instance *store.InstanceMessage, database *store.DatabaseMessage, schema *metadatapb.DatabaseSchemaMetadata) aireview.Target {
	target := aireview.Target{
		Engine:       instance.Metadata.GetEngine().String(),
		Version:      instance.Metadata.GetVersion(),
		CharacterSet: schema.GetCharacterSet(),
		Collation:    schema.GetCollation(),
		SearchPath:   schema.GetSearchPath(),
		Owner:        schema.GetOwner(),
	}
	if database.EffectiveEnvironmentID != nil {
		target.Environment = *database.EffectiveEnvironmentID
	}
	for _, s := range schema.GetSchemas() {
		target.Schemas = append(target.Schemas, aireview.SchemaSummary{Name: s.GetName(), ObjectCount: schemaObjectCount(s)})
	}
	return target
}

// schemaObjectCount counts the objects of a schema that have a definition of
// their own.
func schemaObjectCount(s *metadatapb.SchemaMetadata) int {
	return len(s.GetTables()) + len(s.GetExternalTables()) + len(s.GetViews()) + len(s.GetMaterializedViews()) +
		len(s.GetFunctions()) + len(s.GetProcedures()) + len(s.GetSequences()) + len(s.GetPackages()) +
		len(s.GetEvents()) + len(s.GetStreams()) + len(s.GetTasks()) + len(s.GetEnumTypes()) + len(s.GetCompositeTypes())
}

// aiReviewStandardRules returns the standard rules the policy turns on, each
// with the severity the standard review gives its findings and whether the
// model leaves the rule to it on the engine, which needs the standard review
// to run there. A policy that switches the standard review off turns none on.
func aiReviewStandardRules(policy *storepb.ReviewRulePolicy, engine storepb.Engine, runs bool) []aireview.StandardRule {
	rules := make([]aireview.StandardRule, 0, len(policy.GetRules()))
	for _, rule := range policy.GetRules() {
		rules = append(rules, aireview.StandardRule{
			Type:     rule,
			Severity: aiReviewSeverity(rulePriority(rule)),
			// The walk stops at a statement it cannot follow, such as a DO
			// block, and says nothing of the statements after it, so the
			// model checks WALK_THROUGH's condition too.
			Reviewed: runs && rule != storepb.ReviewRuleType_WALK_THROUGH && reviewsRule(engine, rule),
		})
	}
	return rules
}

// mergeAIReviewFindings groups the findings of the completed units into
// problems. The findings that more than one database reported on one
// statement of a sheet go to merge, which decides which of them are the same
// problem, and so do the findings on a statement an earlier result or a
// finding of rules stands on, with that result or finding. A problem the
// merge puts with a rule finding joins that finding's result, and every other
// problem posts a comment of its own. Every other finding, and every finding
// of a merge that fails, is a problem of its own, so a failed merge posts
// duplicates and never drops a finding. The error is ctx's, when it ends.
func mergeAIReviewFindings(ctx context.Context, merge mergeFunc, results []*aiReviewResult, earlier map[aiReviewSheet][]*aiReviewEarlier, rules []*ruleResult) ([]*aiReviewProblem, error) {
	var order []aiReviewSheet
	bySheet := make(map[aiReviewSheet][]*aiReviewFinding)
	reviewed := make(map[aiReviewSheet][]string)
	for _, result := range results {
		if result == nil {
			continue
		}
		key := aiReviewSheet{specID: result.Unit.Check.SpecID, sheetSha256: result.Unit.Check.SheetSha256}
		if !slices.Contains(reviewed[key], result.Unit.Check.Target) {
			reviewed[key] = append(reviewed[key], result.Unit.Check.Target)
		}
		for _, finding := range result.Findings {
			if _, ok := bySheet[key]; !ok {
				order = append(order, key)
			}
			bySheet[key] = append(bySheet[key], &aiReviewFinding{Unit: result.Unit, Finding: finding})
		}
	}
	rulesOn := make(map[aiReviewSheet][]*ruleResult)
	for _, rule := range rules {
		key := aiReviewSheet{specID: rule.specID, sheetSha256: rule.sheetSha256}
		rulesOn[key] = append(rulesOn[key], rule)
	}
	var problems []*aiReviewProblem
	for _, key := range order {
		sheetProblems, err := mergeSheetFindings(ctx, merge, bySheet[key], earlier[key], rulesOn[key])
		if err != nil {
			return nil, err
		}
		slices.Sort(reviewed[key])
		for _, problem := range sheetProblems {
			problem.Reviewed = reviewed[key]
		}
		problems = append(problems, sheetProblems...)
	}
	return problems, nil
}

// mergeSheetFindings groups the findings of one sheet, ordered by the first
// finding of each problem, and gives each problem the earlier result it keeps
// or the rule finding it joins. An earlier result is offered to the merge
// only while its statement, which its fingerprint names, is unchanged and has
// findings; one left out is superseded by the run. A finding of rules, the
// rule results on the sheet, is offered while its statement has findings.
func mergeSheetFindings(ctx context.Context, merge mergeFunc, findings []*aiReviewFinding, earlier []*aiReviewEarlier, rules []*ruleResult) ([]*aiReviewProblem, error) {
	statement := findings[0].Unit.Statement
	check := findings[0].Unit.Check
	engine := sheetEngine(findings)
	statements := sheetStatements(ctx, engine, check.SpecID, statement)
	blocks, split := slices.Clone(statements.Blocks), statements.Split
	blockOf := make([]int, len(findings))
	blockTargets := make(map[int]map[string]bool)
	for i, finding := range findings {
		blockOf[i] = findBlock(&blocks, finding.Finding.Line)
		if blockTargets[blockOf[i]] == nil {
			blockTargets[blockOf[i]] = make(map[string]bool)
		}
		blockTargets[blockOf[i]][finding.Unit.Check.Target] = true
	}
	// A line outside every statement, which findBlock made a block of its
	// own, is its own body. Its ordinal counts the equal lines outside every
	// statement before it, so it holds whichever lines have findings.
	segments, bodyStart := slices.Clone(statements.Segments), slices.Clone(statements.BodyStart)
	if extra := blocks[len(segments):]; len(extra) > 0 {
		lines := aireview.StatementLines(statement)
		outside := make([]bool, len(lines)+1)
		for line := range outside {
			outside[line] = true
		}
		for _, block := range statements.Blocks {
			for line := max(block.Start, 1); line <= min(block.End, len(lines)); line++ {
				outside[line] = false
			}
		}
		for _, block := range extra {
			at := min(block.Start, len(lines))
			text := strings.TrimSpace(lines[at-1])
			ordinal := 0
			for line := 1; line <= at; line++ {
				if outside[line] && strings.TrimSpace(lines[line-1]) == text {
					ordinal++
				}
			}
			segments = append(segments, statementSegment(check.SpecID, text, ordinal))
			bodyStart = append(bodyStart, block.Start)
		}
	}
	// An earlier result stands where its line stood in its statement's body,
	// which is the same text in both sheets.
	earlierBlockOf := make([]int, len(earlier))
	earlierLine := make([]int, len(earlier))
	earlierOn := make(map[int]bool)
	before := map[string]*aiReviewStatements{check.SheetSha256: statements}
	for i, result := range earlier {
		segment, _, _ := strings.Cut(result.Fingerprint, "/")
		earlierBlockOf[i] = slices.Index(segments, segment)
		if earlierBlockOf[i] < 0 {
			continue
		}
		earlierOn[earlierBlockOf[i]] = true
		old, ok := before[result.SheetSha256]
		if !ok {
			old = sheetStatements(ctx, engine, check.SpecID, result.Sheet)
			before[result.SheetSha256] = old
		}
		block := blocks[earlierBlockOf[i]]
		earlierLine[i] = bodyStart[earlierBlockOf[i]]
		if from := slices.Index(old.Segments, segment); from >= 0 {
			earlierLine[i] = min(max(earlierLine[i]+result.Line-old.BodyStart[from], block.Start), block.End)
		}
	}
	// A rule finding stands on the statement its anchor starts in. Only a
	// statement with findings has something to put with it.
	var members []*ruleMember
	var memberBlock []int
	ruleOn := make(map[int]bool)
	for _, rule := range rules {
		for _, finding := range rule.members {
			if finding.anchor == nil {
				continue
			}
			line := int(finding.anchor.GetStartPosition().GetLine())
			block := slices.IndexFunc(blocks, func(b aireview.LineRange) bool { return line >= b.Start && line <= b.End })
			if block < 0 || len(blockTargets[block]) == 0 {
				continue
			}
			members = append(members, &ruleMember{result: rule, finding: finding})
			memberBlock = append(memberBlock, block)
			ruleOn[block] = true
		}
	}
	// sources counts the databases with findings on a block, one more when an
	// earlier result stands on it, and one more when a rule finding does.
	sources := func(block int) int {
		n := len(blockTargets[block])
		if earlierOn[block] {
			n++
		}
		if ruleOn[block] {
			n++
		}
		return n
	}

	request := &aireview.MergeRequest{Statement: statement, Blocks: blocks}
	// offered maps a merge candidate to its finding, to its earlier result at
	// len(findings) and after, or to its rule finding after those.
	var offered []int
	for i, finding := range findings {
		if sources(blockOf[i]) > 1 {
			offered = append(offered, i)
			request.Candidates = append(request.Candidates, aireview.MergeCandidate{
				Block:    blockOf[i],
				Database: finding.Unit.Check.Target,
				Finding:  finding.Finding,
			})
		}
	}
	for i, result := range earlier {
		if earlierBlockOf[i] >= 0 && len(blockTargets[earlierBlockOf[i]]) > 0 {
			offered = append(offered, len(findings)+i)
			request.Candidates = append(request.Candidates, aireview.MergeCandidate{
				Block:   earlierBlockOf[i],
				Earlier: true,
				Finding: earlierFinding(result, earlierLine[i]),
			})
		}
	}

	// The merge is shown the comment an earlier run left on a rule result when
	// that comment can stay as it is.
	var threads []*ruleResult
	for i, member := range members {
		offered = append(offered, len(findings)+len(earlier)+i)
		request.Candidates = append(request.Candidates, aireview.MergeCandidate{
			Block:    memberBlock[i],
			Standard: true,
			Finding:  ruleCandidate(member.finding),
		})
		rule := member.result
		thread := slices.Index(threads, rule)
		if thread < 0 {
			if !offersEarlier(rule) {
				continue
			}
			thread = len(threads)
			threads = append(threads, rule)
			request.Threads = append(request.Threads, aireview.MergeThread{Comment: rule.earlier.Comment})
		}
		request.Threads[thread].Candidates = append(request.Threads[thread].Candidates, len(request.Candidates)-1)
	}

	type group struct {
		findings []int
		earlier  []*aiReviewEarlier
		rules    []*ruleMember
	}
	var groups []*group
	merged := make([]bool, len(findings))
	same := make(map[*ruleResult]bool)
	if len(offered) > 0 {
		result, err := merge(ctx, request)
		switch {
		case ctx.Err() != nil:
			return nil, ctx.Err()
		case err != nil:
			slog.Warn("AI review could not merge the findings of different databases, posting each on its own",
				slog.String("spec", findings[0].Unit.Check.SpecID),
				slog.Int("findings", len(offered)),
				log.BBError(err))
		default:
			for _, candidates := range result.Groups {
				g := &group{}
				for _, candidate := range candidates {
					switch i := offered[candidate]; {
					case i < len(findings):
						g.findings = append(g.findings, i)
						merged[i] = true
					case i < len(findings)+len(earlier):
						g.earlier = append(g.earlier, earlier[i-len(findings)])
					default:
						g.rules = append(g.rules, members[i-len(findings)-len(earlier)])
					}
				}
				if len(g.findings) > 0 {
					groups = append(groups, g)
				}
			}
			for _, thread := range result.Same {
				if thread >= 0 && thread < len(threads) {
					same[threads[thread]] = true
				}
			}
		}
	}
	for i := range findings {
		if !merged[i] {
			groups = append(groups, &group{findings: []int{i}})
		}
	}
	slices.SortFunc(groups, func(a, b *group) int { return cmp.Compare(a.findings[0], b.findings[0]) })

	problems := make([]*aiReviewProblem, 0, len(groups))
	for _, g := range groups {
		// A merged group holds the findings of one block.
		block := blockOf[g.findings[0]]
		problem := &aiReviewProblem{Block: blocks[block], BlockIsStatement: split, Segment: segments[block]}
		for _, i := range g.findings {
			problem.Findings = append(problem.Findings, findings[i])
		}
		problem.Earlier = keptEarlier(problem.Findings, g.earlier)
		// The problem's text joins the rule finding's result, so it keeps no
		// thread of its own. A rule finding is offered without its databases,
		// so the merge can put a problem with one that is on other databases:
		// such a problem stays on its own, since the result names the
		// databases it applies to.
		if len(g.rules) == 1 && !slices.ContainsFunc(problem.Findings, func(finding *aiReviewFinding) bool {
			return !slices.Contains(g.rules[0].finding.targets, finding.Unit.Check.Target)
		}) {
			problem.Rule, problem.Earlier = g.rules[0], nil
		}
		problems = append(problems, problem)
	}
	markUnchanged(problems, same)
	return problems, nil
}

// offersEarlier reports whether the merge is asked if the comment an earlier
// run left on a rule result already states this run's problems. A comment of
// facts is offered only when its facts name no database: facts that do are
// about that database, and the merge, which is shown no database name, cannot
// tell whether they state a problem on another one.
func offersEarlier(rule *ruleResult) bool {
	if rule.earlier == nil {
		return false
	}
	if statesProblem(rule.earlier.Comment) {
		return canKeepEarlier(rule)
	}
	if !holdsFactsOnly(rule) {
		return false
	}
	// The names are those addRuleFacts gives the databases of each finding.
	names := make(map[string]bool)
	for _, member := range rule.members {
		for _, label := range databaseLabels(member.targets) {
			names[label] = true
		}
	}
	own := strings.Split(rule.ruleText(), "\n")
	for _, line := range strings.Split(rule.earlier.Comment, "\n") {
		// The rules' own lines name databases in a merged result.
		if slices.Contains(own, line) {
			continue
		}
		for name := range names {
			if strings.Contains(line, name) {
				return false
			}
		}
	}
	return true
}

// canKeepEarlier reports whether the comment an earlier run left on a rule
// result can stay as it is. It holds more than the rules' own text, which is
// what the AI review added then. It holds every line of the rules' text as it
// is now, which the line numbers of a merged result's statements leave when an
// edit moves them. And it names no part of the result's databases as the ones
// a problem is on, since the model is shown no database name and cannot tell
// whether that part is the same now.
func canKeepEarlier(rule *ruleResult) bool {
	if rule.earlier == nil {
		return false
	}
	text := rule.ruleText()
	if rule.earlier.Comment == text || !holdsLines(rule.earlier.Comment, text) {
		return false
	}
	var targets []string
	for _, member := range rule.members {
		targets = append(targets, member.targets...)
	}
	var labels []string
	for target, label := range databaseLabels(targets) {
		labels = append(labels, label+": ")
		// A database is named with its instance where another shares its name.
		if i := strings.Index(target, common.InstanceNamePrefix); i >= 0 {
			if instance, database, err := common.GetInstanceDatabaseID(target[i:]); err == nil {
				labels = append(labels, database+": ", instance+"/"+database+": ")
			}
		}
	}
	for _, line := range strings.Split(rule.earlier.Comment, "\n") {
		line = strings.TrimLeft(line, " ")
		if strings.HasPrefix(line, flaggedDifferentlyOn) || strings.HasPrefix(line, notFlaggedOn) ||
			slices.ContainsFunc(labels, func(label string) bool { return strings.HasPrefix(line, label) }) {
			return false
		}
	}
	return true
}

// markUnchanged marks the problems on the rule results in same, whose earlier
// comment the merge says already states them. Two kinds of result are left
// out, and their comment is written again. One is a result its problems raise
// above the priority that comment was posted with: nobody read the comment at
// the higher priority. The other is a result with a problem that is not on
// every database of its rule finding, when the comment states a problem:
// canKeepEarlier offers only such a comment written for all of them. A comment
// of facts is offered only when it names no database, so its facts hold for
// whichever databases the problem is on, and it stays either way.
func markUnchanged(problems []*aiReviewProblem, same map[*ruleResult]bool) {
	priority := make(map[*ruleResult]storepb.IssueCommentPayload_ReviewMetadata_Priority)
	partial := make(map[*ruleResult]bool)
	for _, problem := range problems {
		if problem.Rule == nil || !same[problem.Rule.result] {
			continue
		}
		rule := problem.Rule.result
		if _, ok := priority[rule]; !ok {
			priority[rule] = rulePriority(problem.Rule.finding.rule)
		}
		if raised := raisedPriority(problem); priorityRank(raised) < priorityRank(priority[rule]) {
			priority[rule] = raised
		}
		// Without statement boundaries the comment cannot say which of the
		// other databases flagged the statement, so it names no part either.
		if !problem.BlockIsStatement || !slices.Equal(problemTargets(problem.Findings), problem.Rule.finding.targets) {
			partial[rule] = true
		}
	}
	for _, problem := range problems {
		if problem.Rule == nil {
			continue
		}
		rule := problem.Rule.result
		if p, ok := priority[rule]; ok && priorityRank(p) >= priorityRank(rule.earlier.Priority) && (!partial[rule] || !statesProblem(rule.earlier.Comment)) {
			problem.Unchanged = true
		}
	}
}

// holdsLines reports whether comment holds every line of text, in order.
func holdsLines(comment, text string) bool {
	lines := strings.Split(comment, "\n")
	for _, want := range strings.Split(text, "\n") {
		i := slices.Index(lines, want)
		if i < 0 {
			return false
		}
		lines = lines[i+1:]
	}
	return true
}

// raisedPriority is the priority a problem raises the result of its rule
// finding to: the highest among its findings that report more than the rule's
// own condition, whose priority the rule sets, or unspecified when none does.
func raisedPriority(problem *aiReviewProblem) storepb.IssueCommentPayload_ReviewMetadata_Priority {
	raised := storepb.IssueCommentPayload_ReviewMetadata_PRIORITY_UNSPECIFIED
	for _, finding := range problem.Findings {
		if restatesRule(finding, problem.Rule.finding.rule) {
			continue
		}
		if priority := aiReviewPriority(finding.Finding.Severity); priorityRank(priority) < priorityRank(raised) {
			raised = priority
		}
	}
	return raised
}

// restatesRule reports whether a finding cites the standard rule itself. A
// policy sentence can name the rule, and a finding that cites it is the
// policy's, not a restatement.
func restatesRule(finding *aiReviewFinding, rule storepb.ReviewRuleType) bool {
	return aireview.CitesStandardRule(finding.Finding.Rule, rule) && !aireview.CitesPolicy(finding.Finding.Rule, finding.Unit.Policies...)
}

// foldedRule is the rule a problem on a rule finding cites in that finding's
// comment, or empty for none. A policy sentence says why the problem is one,
// so the most severe finding that quotes the policy gives it. Otherwise the
// lead finding does, unless it quotes the review's own prompt or a standard
// rule, which tells the author nothing the thread does not.
func foldedRule(findings []*aiReviewFinding, lead *aiReviewFinding) string {
	var quoting []*aiReviewFinding
	for _, finding := range findings {
		if aireview.CitesPolicy(finding.Finding.Rule, finding.Unit.Policies...) {
			quoting = append(quoting, finding)
		}
	}
	if len(quoting) > 0 {
		return leadFinding(quoting).Finding.Rule
	}
	if aireview.CitesPrompt(lead.Finding.Rule) {
		return ""
	}
	return lead.Finding.Rule
}

// ruleCandidate is a rule finding as the merge shows it: its rule and codes
// as the title, the rule's priority, the line it starts on, and its message
// as the evidence.
func ruleCandidate(finding *ruleFinding) aireview.Finding {
	title := finding.rule.String()
	if len(finding.codes) > 0 {
		title += " (" + strings.Join(finding.codes, ", ") + ")"
	}
	return aireview.Finding{
		Title:    title,
		Severity: aiReviewSeverity(rulePriority(finding.rule)),
		Line:     int(finding.anchor.GetStartPosition().GetLine()),
		Evidence: finding.message,
	}
}

// keptEarlier returns the earlier result a problem keeps: the one the merge
// grouped with its findings, or nil when there is none or more than one. A
// resolved result is kept only while the problem is no more severe than the
// result was, since keeping it lets the rollout past the problem.
func keptEarlier(findings []*aiReviewFinding, earlier []*aiReviewEarlier) *aiReviewEarlier {
	if len(earlier) != 1 {
		return nil
	}
	result := earlier[0]
	// The enum orders P0 before P1 before P2, so the smaller value is the
	// higher priority.
	if result.Resolved && aiReviewPriority(leadFinding(findings).Finding.Severity) < result.Priority {
		return nil
	}
	return result
}

// earlierFinding is an earlier result as the merge shows it: its title, the
// rule and fix of its comment, its priority, and the line it stands on now.
// The evidence is left out, since a comment on more than one database names
// them there.
func earlierFinding(result *aiReviewEarlier, line int) aireview.Finding {
	rule, fix := aiReviewCommentParts(result.Comment)
	return aireview.Finding{Title: result.Title, Severity: aiReviewSeverity(result.Priority), Line: line, Rule: rule, Fix: fix}
}

// aiReviewCommentParts reads back the rule and fix that aiReviewCommentText
// wrote: the last "Rule:" and "Suggested fix:" paragraphs.
func aiReviewCommentParts(comment string) (rule, fix string) {
	if i := strings.LastIndex(comment, aiReviewFixLabel); i >= 0 {
		fix = comment[i+len(aiReviewFixLabel):]
		comment = comment[:i]
	}
	if i := strings.LastIndex(comment, aiReviewRuleLabel); i >= 0 {
		rule = comment[i+len(aiReviewRuleLabel):]
	}
	return rule, fix
}

// sheetStatements splits a sheet into the merge's blocks and gives each its
// segment. A block's body is its statement's as the standard rule review reads
// it, from the first token to the last, so that the comments and blank lines
// around a statement are not part of it. For an engine the rule review does
// not cover, the body is the block's text, comments included.
func sheetStatements(ctx context.Context, engine storepb.Engine, specID, sql string) *aiReviewStatements {
	blocks, split := statementBlocks(engine, sql)
	lines := aireview.StatementLines(sql)
	bodies := make([]string, len(blocks))
	bodyStart := make([]int, len(blocks))
	for i, block := range blocks {
		bodies[i] = strings.TrimSpace(strings.Join(lines[min(block.Start, len(lines))-1:min(block.End, len(lines))], "\n"))
		bodyStart[i] = block.Start
	}
	var ranges []review.Range
	if split {
		ranges, _ = statementBodies(ctx, engine, sql)
	}
	read := make([]bool, len(blocks))
	for _, r := range ranges {
		if r.Start < 0 || r.Start >= r.End || r.End > len(sql) {
			continue
		}
		line := strings.Count(sql[:r.Start], "\n") + 1
		i := slices.IndexFunc(blocks, func(block aireview.LineRange) bool { return line >= block.Start && line <= block.End })
		if i < 0 || read[i] {
			continue
		}
		read[i] = true
		bodies[i], bodyStart[i] = sql[r.Start:r.End], line
	}
	statements := &aiReviewStatements{Blocks: blocks, Split: split, BodyStart: bodyStart}
	ordinals := make(map[string]int)
	for _, body := range bodies {
		ordinals[body]++
		statements.Segments = append(statements.Segments, statementSegment(specID, body, ordinals[body]))
	}
	return statements
}

// statementSegment identifies a statement across runs by its spec, its body,
// and its ordinal among the equal bodies of its sheet.
func statementSegment(specID, body string, ordinal int) string {
	h := sha256.New()
	for _, field := range []string{specID, body, strconv.Itoa(ordinal)} {
		_, _ = h.Write(binary.BigEndian.AppendUint64(nil, uint64(len(field))))
		_, _ = h.Write([]byte(field))
	}
	return hex.EncodeToString(h.Sum(nil))
}

// limitAIReviewProblems marks Held every problem past the maxAIReviewProblems
// most important open ones, by priority and then in reading order, and returns
// how many it marked. A problem that keeps an earlier result is never held,
// since holding it would supersede the thread of a problem the run still
// reports, and a resolved one takes no place, since nobody has to act on it.
// Neither does a problem on a rule finding, which posts no thread of its own.
func limitAIReviewProblems(specs []*storepb.PlanConfig_Spec, problems []*aiReviewProblem) int {
	specIndex := make(map[string]int, len(specs))
	for i, spec := range specs {
		specIndex[spec.GetId()] = i
	}
	type place struct{ priority, spec, line int }
	placeOf := func(problem *aiReviewProblem) place {
		lead := leadFinding(problem.Findings)
		p := place{priority: priorityRank(aiReviewPriority(lead.Finding.Severity)), spec: len(specs), line: commonLine(problem.Findings)}
		if i, ok := specIndex[lead.Unit.Check.SpecID]; ok {
			p.spec = i
		}
		return p
	}
	ranked := slices.Clone(problems)
	slices.SortStableFunc(ranked, func(x, y *aiReviewProblem) int {
		a, b := placeOf(x), placeOf(y)
		return cmp.Or(cmp.Compare(a.priority, b.priority), cmp.Compare(a.spec, b.spec), cmp.Compare(a.line, b.line))
	})

	places, held := 0, 0
	for _, problem := range ranked {
		switch {
		case problem.Rule != nil:
		case problem.Earlier != nil && problem.Earlier.Resolved:
		case places < maxAIReviewProblems:
			places++
		case problem.Earlier != nil:
		default:
			problem.Held = true
			held++
		}
	}
	return held
}

// summarizeAIReviewProblems sets the summary of each problem that is not held,
// that more than one database reports, and that keeps no earlier text, its
// own earlier result's or its rule result's, at most aiReviewRunConcurrency at
// a time. A problem
// whose summary fails keeps none, and its comment is built from its lead
// finding. The error is ctx's, when it ends.
func summarizeAIReviewProblems(ctx context.Context, summarize summarizeFunc, problems []*aiReviewProblem) error {
	var group errgroup.Group
	group.SetLimit(aiReviewRunConcurrency)
	for _, problem := range problems {
		findings := findingsByTarget(problem.Findings)
		if problem.Held || problem.Earlier != nil || problem.Unchanged || len(problemTargets(findings)) < 2 {
			continue
		}
		group.Go(func() error {
			// The scheduler's recovery covers only the goroutine that called
			// RunOnce.
			defer func() {
				if r := recover(); r != nil {
					slog.Error("AI review summary PANIC RECOVER", slog.Any("panic", r), log.BBStack("panic-stack"))
				}
			}()
			request := &aireview.SummaryRequest{Statement: findings[0].Unit.Statement, Block: problem.Block}
			for _, finding := range findings {
				request.Findings = append(request.Findings, aireview.SummaryFinding{Database: finding.Unit.Check.Target, Finding: finding.Finding})
			}
			summary, err := summarize(ctx, request)
			if err != nil {
				if ctx.Err() == nil {
					slog.Warn("AI review could not summarize a problem, posting its lead finding",
						slog.String("spec", findings[0].Unit.Check.SpecID),
						slog.Int("findings", len(findings)),
						log.BBError(err))
				}
				return nil
			}
			problem.Summary = summary
			return nil
		})
	}
	_ = group.Wait()
	return ctx.Err()
}

// sheetEngine returns the engine of the databases a sheet's findings come
// from, or ENGINE_UNSPECIFIED when they run different engines.
func sheetEngine(findings []*aiReviewFinding) storepb.Engine {
	engine := findings[0].Unit.Engine
	for _, finding := range findings[1:] {
		if finding.Unit.Engine != engine {
			return storepb.Engine_ENGINE_UNSPECIFIED
		}
	}
	return engine
}

// statementBlocks returns the line range of each statement of a sheet, the
// blocks inside which findings can merge, and true. Without a splitter for the
// engine, or when the split does not give ordered ranges, the sheet is one
// block and the result is false.
func statementBlocks(engine storepb.Engine, statement string) ([]aireview.LineRange, bool) {
	// The count is one too many when the sheet ends with a line break, which
	// is harmless at the end of the last block.
	whole := []aireview.LineRange{{Start: 1, End: strings.Count(statement, "\n") + 1}}
	if engine == storepb.Engine_ENGINE_UNSPECIFIED {
		return whole, false
	}
	statements, err := base.SplitMultiSQL(engine, statement)
	if err != nil {
		return whole, false
	}
	var blocks []aireview.LineRange
	for _, s := range statements {
		if s.Empty {
			continue
		}
		if s.Start == nil || s.End == nil {
			return whole, false
		}
		// A statement starts where the one before it ended, so its text can
		// open with the line break and blank lines between the two.
		leading := s.Text[:len(s.Text)-len(strings.TrimLeft(s.Text, " \t\r\n"))]
		start := int(s.Start.Line) + strings.Count(leading, "\n")
		end := int(s.End.Line)
		// End is the position after the last character, so a statement that
		// ends with a line break ends at the start of the next line.
		if s.End.Column == 1 && end > start {
			end--
		}
		if end < start || (len(blocks) > 0 && start <= blocks[len(blocks)-1].End) {
			return whole, false
		}
		blocks = append(blocks, aireview.LineRange{Start: start, End: end})
	}
	if len(blocks) == 0 {
		return whole, false
	}
	return blocks, true
}

// findBlock returns the index of the block that holds line. A line outside
// every statement, such as a blank line between two, becomes a block of its
// own.
func findBlock(blocks *[]aireview.LineRange, line int) int {
	for i, block := range *blocks {
		if line >= block.Start && line <= block.End {
			return i
		}
	}
	*blocks = append(*blocks, aireview.LineRange{Start: line, End: line})
	return len(*blocks) - 1
}

// flaggedOnBlocks returns, for each statement, the databases with any finding
// on it.
func flaggedOnBlocks(problems []*aiReviewProblem) map[aiReviewBlock][]string {
	onBlock := make(map[aiReviewBlock][]string)
	for _, problem := range problems {
		key := problemBlock(problem)
		for _, finding := range problem.Findings {
			if !slices.Contains(onBlock[key], finding.Unit.Check.Target) {
				onBlock[key] = append(onBlock[key], finding.Unit.Check.Target)
			}
		}
	}
	return onBlock
}

func problemBlock(problem *aiReviewProblem) aiReviewBlock {
	check := problem.Findings[0].Unit.Check
	return aiReviewBlock{specID: check.SpecID, sheetSha256: check.SheetSha256, block: problem.Block}
}

// aiReviewComments converts the problems into the review result comments to
// post: one OPEN root per problem that is not held and joins no rule finding,
// anchored to the line most of its findings start on, with every database it
// applies to. The priority and rule are those of its lead finding. A problem
// that keeps an earlier result carries that result's fingerprint and text, so
// the run keeps its thread and moves it. Any other problem gets a new
// fingerprint: its statement's segment, then a random part, since nothing else
// about an AI finding holds across runs.
func aiReviewComments(projectID string, issueUID int64, problems []*aiReviewProblem) []*store.IssueCommentMessage {
	onBlock := flaggedOnBlocks(problems)
	comments := make([]*store.IssueCommentMessage, 0, len(problems))
	for _, problem := range problems {
		if problem.Held || problem.Rule != nil {
			continue
		}
		findings := findingsByTarget(problem.Findings)
		lead := leadFinding(findings)
		line := int32(commonLine(findings))
		targets := problemTargets(findings)
		var title, comment, fingerprint string
		if problem.Earlier != nil {
			title, comment, fingerprint = problem.Earlier.Title, problem.Earlier.Comment, problem.Earlier.Fingerprint
		} else {
			title, comment = aiReviewCommentText(problem, lead, findings, targets, onBlock[problemBlock(problem)], lead.Finding.Rule)
			fingerprint = problem.Segment + "/" + uuid.NewString()
		}
		comments = append(comments, &store.IssueCommentMessage{
			ProjectID: projectID,
			IssueUID:  issueUID,
			Payload: &storepb.IssueCommentPayload{
				Comment: comment,
				StatementAnchor: &storepb.IssueCommentPayload_StatementAnchor{
					SpecId:        lead.Unit.Check.SpecID,
					SheetSha256:   lead.Unit.Check.SheetSha256,
					StartPosition: &storepb.Position{Line: line},
					EndPosition:   &storepb.Position{Line: line},
				},
				ReviewMetadata: &storepb.IssueCommentPayload_ReviewMetadata{
					Priority:    aiReviewPriority(lead.Finding.Severity),
					Targets:     targets,
					Fingerprint: fingerprint,
					Title:       title,
				},
			},
		})
	}
	return comments
}

// foldAIReviewProblems adds each problem on a rule finding to the result of
// that finding: its text after the finding's own, and its priority when it is
// the higher one and the problem reports more than the rule's condition. A
// result whose problems are all unchanged keeps the comment an earlier run
// left instead. A problem adds its evidence and its fix as plain text, with no
// title of its own, so that the thread reads as one finding, and with the
// rule foldedRule gives it.
func foldAIReviewProblems(problems []*aiReviewProblem) {
	onBlock := flaggedOnBlocks(problems)
	for _, problem := range problems {
		if problem.Rule == nil {
			continue
		}
		rule, finding := problem.Rule.result, problem.Rule.finding
		if problem.Unchanged {
			rule.keepsEarlier = true
			continue
		}
		findings := findingsByTarget(problem.Findings)
		lead := leadFinding(findings)
		_, text := aiReviewCommentText(problem, lead, findings, problemTargets(findings), onBlock[problemBlock(problem)], foldedRule(findings, lead))
		finding.folded = append(finding.folded, text)
		if raised := raisedPriority(problem); priorityRank(raised) < priorityRank(finding.raised) {
			finding.raised = raised
		}
	}
}

// findingsByTarget orders findings by database, keeping the order of one
// database's findings.
func findingsByTarget(findings []*aiReviewFinding) []*aiReviewFinding {
	byTarget := slices.Clone(findings)
	slices.SortStableFunc(byTarget, func(x, y *aiReviewFinding) int { return cmp.Compare(x.Unit.Check.Target, y.Unit.Check.Target) })
	return byTarget
}

// problemTargets returns the databases of findings ordered by database.
func problemTargets(findings []*aiReviewFinding) []string {
	var targets []string
	for _, finding := range findings {
		if !slices.Contains(targets, finding.Unit.Check.Target) {
			targets = append(targets, finding.Unit.Check.Target)
		}
	}
	slices.Sort(targets)
	return targets
}

// leadFinding returns the finding a problem is posted as: the one with the
// highest priority, on the first database by name among those.
func leadFinding(findings []*aiReviewFinding) *aiReviewFinding {
	lead := findings[0]
	for _, finding := range findings[1:] {
		// The enum orders P0 before P1 before P2, so the smaller value is the
		// higher priority.
		priority, leadPriority := aiReviewPriority(finding.Finding.Severity), aiReviewPriority(lead.Finding.Severity)
		if priority < leadPriority || (priority == leadPriority && finding.Unit.Check.Target < lead.Unit.Check.Target) {
			lead = finding
		}
	}
	return lead
}

// commonLine returns the line most of the findings start on, the first such
// line on a tie.
func commonLine(findings []*aiReviewFinding) int {
	counts := make(map[int]int)
	for _, finding := range findings {
		counts[finding.Finding.Line]++
	}
	best := 0
	for line, count := range counts {
		if count > counts[best] || (count == counts[best] && line < best) {
			best = line
		}
	}
	return best
}

// maxNamedDatabases is how many databases one line of a comment names; the
// rest are counted.
const maxNamedDatabases = 10

// The paragraphs that end every comment, which aiReviewCommentParts reads back.
const (
	aiReviewRuleLabel = "\n\nRule: "
	aiReviewFixLabel  = "\n\nSuggested fix: "
)

// The lines of a comment that name the reviewed databases a problem is not
// on, which canKeepEarlier reads back.
const (
	flaggedDifferentlyOn = "- Flagged differently on "
	notFlaggedOn         = "- Not flagged on "
)

// aiReviewGroup is databases of a problem that one line of its comment covers,
// with the highest severity their findings carry.
type aiReviewGroup struct {
	highest aireview.Severity
	targets []string
	facts   string
}

// aiReviewCommentText renders a problem as the title and the body of its
// thread. A problem of more than one database states the shared evidence
// once, then one line for each group of databases with the facts that set it
// apart, then the reviewed databases that reported a different problem on the
// statement, onBlock, or nothing. Without a summary, the shared evidence is the
// lead finding's, labeled with its database.
//
// The comment cites rule, or no rule when it is empty.
//
// A problem on a rule finding is read inside that finding's comment, beside
// the databases the thread already names. So it leaves out the list of its
// databases when they are the rule finding's, and it names its one database
// when the rule finding is on others too.
func aiReviewCommentText(problem *aiReviewProblem, lead *aiReviewFinding, findings []*aiReviewFinding, targets []string, onBlock []string, rule string) (title, text string) {
	var ruleTargets []string
	if problem.Rule != nil {
		ruleTargets = problem.Rule.finding.targets
	}
	var b strings.Builder
	if len(targets) == 1 {
		if len(ruleTargets) > 1 {
			named := slices.Clone(ruleTargets)
			if !slices.Contains(named, targets[0]) {
				named = append(named, targets[0])
			}
			_, _ = b.WriteString(databaseLabels(named)[targets[0]] + ": ")
		}
		for i, finding := range findings {
			if i > 0 {
				_, _ = b.WriteString("\n\n")
			}
			_, _ = b.WriteString(finding.Finding.Evidence)
		}
		if rule != "" {
			_, _ = b.WriteString(aiReviewRuleLabel + rule)
		}
		_, _ = b.WriteString(aiReviewFixLabel + lead.Finding.Fix)
		return lead.Finding.Title, b.String()
	}

	named := slices.Clone(problem.Reviewed)
	for _, target := range targets {
		if !slices.Contains(named, target) {
			named = append(named, target)
		}
	}
	labels := databaseLabels(named)
	severities := make(map[string]aireview.Severity)
	for _, finding := range findings {
		if severity, ok := severities[finding.Unit.Check.Target]; !ok || aiReviewPriority(finding.Finding.Severity) < aiReviewPriority(severity) {
			severities[finding.Unit.Check.Target] = finding.Finding.Severity
		}
	}
	var evidence, fix string
	var groups []*aiReviewGroup
	if summary := problem.Summary; summary != nil {
		title, evidence, fix = summary.Title, summary.Evidence, summary.Fix
		for _, summaryGroup := range summary.Groups {
			group := &aiReviewGroup{highest: aireview.SeverityP2, targets: summaryGroup.Databases, facts: summaryGroup.Facts}
			for _, target := range summaryGroup.Databases {
				priority := aiReviewPriority(severities[target])
				if priority < aiReviewPriority(group.highest) {
					group.highest = severities[target]
				}
			}
			groups = append(groups, group)
		}
	} else {
		title, fix = lead.Finding.Title, lead.Finding.Fix
		evidence = fmt.Sprintf("%s: %s", labels[lead.Unit.Check.Target], lead.Finding.Evidence)
		groups = []*aiReviewGroup{{targets: targets}}
	}
	// The most severe group comes first.
	slices.SortStableFunc(groups, func(x, y *aiReviewGroup) int {
		return cmp.Compare(aiReviewPriority(x.highest), aiReviewPriority(y.highest))
	})

	_, _ = b.WriteString(evidence)
	var lines []string
	for _, group := range groups {
		line := "- " + databaseList(group.targets, labels)
		if group.facts != "" {
			line += ": " + group.facts
		} else if len(groups) == 1 && len(group.targets) == len(ruleTargets) && !slices.ContainsFunc(group.targets, func(target string) bool { return !slices.Contains(ruleTargets, target) }) {
			continue
		}
		lines = append(lines, line)
	}
	var differently, unflagged []string
	for _, target := range problem.Reviewed {
		switch {
		case slices.Contains(targets, target):
		case slices.Contains(onBlock, target):
			// Without statement boundaries, the other finding can be on any
			// statement of the sheet, so neither list names the database.
			if problem.BlockIsStatement {
				differently = append(differently, target)
			}
		default:
			unflagged = append(unflagged, target)
		}
	}
	if len(differently) > 0 {
		lines = append(lines, flaggedDifferentlyOn+databaseList(differently, labels))
	}
	if len(unflagged) > 0 {
		lines = append(lines, notFlaggedOn+databaseList(unflagged, labels))
	}
	if len(lines) > 0 {
		_, _ = b.WriteString("\n\n" + strings.Join(lines, "\n"))
	}
	if rule != "" {
		_, _ = b.WriteString(aiReviewRuleLabel + rule)
	}
	_, _ = b.WriteString(aiReviewFixLabel + fix)
	return title, b.String()
}

// databaseList names databases by their labels in order, up to
// maxNamedDatabases, and counts the rest.
func databaseList(targets []string, labels map[string]string) string {
	names := make([]string, 0, len(targets))
	for _, target := range targets {
		names = append(names, labels[target])
	}
	slices.Sort(names)
	if len(names) > maxNamedDatabases {
		return fmt.Sprintf("%s, and %d more", strings.Join(names[:maxNamedDatabases], ", "), len(names)-maxNamedDatabases)
	}
	return strings.Join(names, ", ")
}

// databaseLabels names each target by its database name, or by instance and
// database name when two targets share the database name.
func databaseLabels(targets []string) map[string]string {
	type name struct {
		instance string
		database string
	}
	names := make(map[string]name, len(targets))
	databases := make(map[string]int)
	for _, target := range targets {
		n := name{database: target}
		if i := strings.Index(target, common.InstanceNamePrefix); i >= 0 {
			if instance, database, err := common.GetInstanceDatabaseID(target[i:]); err == nil {
				n = name{instance: instance, database: database}
			}
		}
		names[target] = n
		databases[n.database]++
	}
	labels := make(map[string]string, len(targets))
	for target, n := range names {
		labels[target] = n.database
		if databases[n.database] > 1 && n.instance != "" {
			labels[target] = n.instance + "/" + n.database
		}
	}
	return labels
}

// aiReviewSeverity maps a thread priority back to the severity of a finding.
func aiReviewSeverity(priority storepb.IssueCommentPayload_ReviewMetadata_Priority) aireview.Severity {
	switch priority {
	case storepb.IssueCommentPayload_ReviewMetadata_P0:
		return aireview.SeverityP0
	case storepb.IssueCommentPayload_ReviewMetadata_P1:
		return aireview.SeverityP1
	default:
		return aireview.SeverityP2
	}
}

// aiReviewPriority maps a finding's severity to the thread priority. The
// reviewer validates the severity, so every finding maps to one.
func aiReviewPriority(severity aireview.Severity) storepb.IssueCommentPayload_ReviewMetadata_Priority {
	switch severity {
	case aireview.SeverityP0:
		return storepb.IssueCommentPayload_ReviewMetadata_P0
	case aireview.SeverityP1:
		return storepb.IssueCommentPayload_ReviewMetadata_P1
	default:
		return storepb.IssueCommentPayload_ReviewMetadata_P2
	}
}

// uniqueCheckTargets drops a database a spec names twice, so that each
// database is reviewed once per spec.
func uniqueCheckTargets(targets []*plancheck.CheckTarget) []*plancheck.CheckTarget {
	type key struct {
		specID string
		target string
	}
	seen := make(map[key]bool, len(targets))
	unique := make([]*plancheck.CheckTarget, 0, len(targets))
	for _, target := range targets {
		k := key{specID: target.SpecID, target: target.Target}
		if seen[k] {
			continue
		}
		seen[k] = true
		unique = append(unique, target)
	}
	return unique
}

// checkSheetSize refuses a sheet over maxAIReviewSheetBytes. A larger sheet
// is reported as not reviewed, never truncated: the tail would pass
// unreviewed.
func checkSheetSize(statement string) error {
	if len(statement) > maxAIReviewSheetBytes {
		return errors.Errorf("not reviewed: the sheet is %d bytes, over the %d byte limit", len(statement), maxAIReviewSheetBytes)
	}
	return nil
}

// reviewUnits runs review over the units, at most aiReviewRunConcurrency at a
// time, and returns each unit's result or error by index. Each review holds
// one of slots, which every run shares. A unit takes its slot before review
// starts, so the wait does not count against the reviewer's timeout, and
// gives up the wait when ctx is done. A panic in one review is that unit's
// error, so the run fails instead of the process: the scheduler's recovery
// covers only the goroutine that called RunOnce.
func reviewUnits(ctx context.Context, slots reviewSlots, units []*aiReviewUnit, review func(context.Context, *aiReviewUnit) (*aireview.Result, error)) ([]*aiReviewResult, []error) {
	results := make([]*aiReviewResult, len(units))
	errs := make([]error, len(units))
	var group errgroup.Group
	group.SetLimit(aiReviewRunConcurrency)
	for i, unit := range units {
		group.Go(func() error {
			defer func() {
				if r := recover(); r != nil {
					slog.Error("AI review PANIC RECOVER", slog.String("target", unit.Check.Target), slog.Any("panic", r), log.BBStack("panic-stack"))
					errs[i] = errors.Errorf("%s: review panic: %v", unit.Check.Target, r)
				}
			}()
			if err := slots.Acquire(ctx, 1); err != nil {
				errs[i] = errors.Wrapf(err, "%s", unit.Check.Target)
				return nil
			}
			defer slots.Release(1)
			result, err := review(ctx, unit)
			if err != nil {
				errs[i] = errors.Wrapf(err, "%s", unit.Check.Target)
				return nil
			}
			results[i] = &aiReviewResult{Unit: unit, Findings: result.Findings, Facts: result.Facts}
			return nil
		})
	}
	_ = group.Wait()
	return results, errs
}
