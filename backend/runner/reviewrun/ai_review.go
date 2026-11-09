package reviewrun

import (
	"cmp"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"log/slog"
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
	// Engine is set by the review once it resolves the database.
	Engine storepb.Engine
}

// aiReviewResult is the findings of one completed unit.
type aiReviewResult struct {
	Unit     *aiReviewUnit
	Findings []aireview.Finding
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

// mergeAIReviewFindings groups the findings of the completed units into the
// problems to post, one comment each. The findings that more than one
// database reported on one statement of a sheet go to merge, which decides
// which of them are the same problem, and so do the findings on a statement
// an earlier result stands on, with that result. Every other finding, and
// every finding of a merge that fails, is a problem of its own, so a failed
// merge posts duplicates and never drops a finding. The error is ctx's, when
// it ends.
func mergeAIReviewFindings(ctx context.Context, merge mergeFunc, results []*aiReviewResult, earlier map[aiReviewSheet][]*aiReviewEarlier) ([]*aiReviewProblem, error) {
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
	var problems []*aiReviewProblem
	for _, key := range order {
		sheetProblems, err := mergeSheetFindings(ctx, merge, bySheet[key], earlier[key])
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
// finding of each problem, and gives each problem the earlier result it keeps.
// An earlier result is offered to the merge only while its statement, which
// its fingerprint names, is unchanged and has findings; one left out is
// superseded by the run.
func mergeSheetFindings(ctx context.Context, merge mergeFunc, findings []*aiReviewFinding, earlier []*aiReviewEarlier) ([]*aiReviewProblem, error) {
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
	// sources counts the databases with findings on a block, and one more when
	// an earlier result stands on it.
	sources := func(block int) int {
		if earlierOn[block] {
			return len(blockTargets[block]) + 1
		}
		return len(blockTargets[block])
	}

	request := &aireview.MergeRequest{Statement: statement, Blocks: blocks}
	// offered maps a merge candidate to its finding, or to its earlier result
	// at len(findings) and after.
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

	type group struct {
		findings []int
		earlier  []*aiReviewEarlier
	}
	var groups []*group
	merged := make([]bool, len(findings))
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
					if i := offered[candidate]; i < len(findings) {
						g.findings = append(g.findings, i)
						merged[i] = true
					} else {
						g.earlier = append(g.earlier, earlier[i-len(findings)])
					}
				}
				if len(g.findings) > 0 {
					groups = append(groups, g)
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
		problems = append(problems, problem)
	}
	return problems, nil
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
// that more than one database reports, and that keeps no earlier result, whose
// text it keeps instead, at most aiReviewRunConcurrency at a time. A problem
// whose summary fails keeps none, and its comment is built from its lead
// finding. The error is ctx's, when it ends.
func summarizeAIReviewProblems(ctx context.Context, summarize summarizeFunc, problems []*aiReviewProblem) error {
	var group errgroup.Group
	group.SetLimit(aiReviewRunConcurrency)
	for _, problem := range problems {
		findings := findingsByTarget(problem.Findings)
		if problem.Held || problem.Earlier != nil || len(problemTargets(findings)) < 2 {
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

// aiReviewComments converts the problems into the review result comments to
// post: one OPEN root per problem not held, anchored to the line most of its
// findings start on, with every database it applies to. The priority and rule
// are those of its lead finding. A problem that keeps an earlier result carries
// that result's fingerprint and text, so the run keeps its thread and moves it.
// Any other problem gets a new fingerprint: its statement's segment, then a
// random part, since nothing else about an AI finding holds across runs.
func aiReviewComments(projectID string, issueUID int64, problems []*aiReviewProblem) []*store.IssueCommentMessage {
	type blockKey struct {
		specID      string
		sheetSha256 string
		block       aireview.LineRange
	}
	keyOf := func(problem *aiReviewProblem) blockKey {
		check := problem.Findings[0].Unit.Check
		return blockKey{specID: check.SpecID, sheetSha256: check.SheetSha256, block: problem.Block}
	}
	// onBlock holds the databases with any finding on a statement.
	onBlock := make(map[blockKey][]string)
	for _, problem := range problems {
		key := keyOf(problem)
		for _, finding := range problem.Findings {
			if !slices.Contains(onBlock[key], finding.Unit.Check.Target) {
				onBlock[key] = append(onBlock[key], finding.Unit.Check.Target)
			}
		}
	}

	comments := make([]*store.IssueCommentMessage, 0, len(problems))
	for _, problem := range problems {
		if problem.Held {
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
			title, comment = aiReviewCommentText(problem, lead, findings, targets, onBlock[keyOf(problem)])
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
					RunType:     storepb.ReviewRun_AI,
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
func aiReviewCommentText(problem *aiReviewProblem, lead *aiReviewFinding, findings []*aiReviewFinding, targets []string, onBlock []string) (title, text string) {
	var b strings.Builder
	if len(targets) == 1 {
		for i, finding := range findings {
			if i > 0 {
				_, _ = b.WriteString("\n\n")
			}
			_, _ = b.WriteString(finding.Finding.Evidence)
		}
		_, _ = b.WriteString(aiReviewRuleLabel + lead.Finding.Rule)
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

	_, _ = fmt.Fprintf(&b, "%s\n", evidence)
	for _, group := range groups {
		_, _ = fmt.Fprintf(&b, "\n- %s", databaseList(group.targets, labels))
		if group.facts != "" {
			_, _ = fmt.Fprintf(&b, ": %s", group.facts)
		}
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
		_, _ = fmt.Fprintf(&b, "\n- Flagged differently on %s", databaseList(differently, labels))
	}
	if len(unflagged) > 0 {
		_, _ = fmt.Fprintf(&b, "\n- Not flagged on %s", databaseList(unflagged, labels))
	}
	_, _ = b.WriteString(aiReviewRuleLabel + lead.Finding.Rule)
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
			results[i] = &aiReviewResult{Unit: unit, Findings: result.Findings}
			return nil
		})
	}
	_ = group.Wait()
	return results, errs
}
