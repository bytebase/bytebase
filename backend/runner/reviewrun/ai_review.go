package reviewrun

import (
	"cmp"
	"context"
	"fmt"
	"log/slog"
	"slices"
	"strings"

	metadatapb "github.com/bytebase/omni/metadata"
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

// mergeFunc asks the model which findings report the same problem.
type mergeFunc func(context.Context, *aireview.MergeRequest) (*aireview.MergeResult, error)

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
// which of them are the same problem. Every other finding, and every finding
// of a merge that fails, is a problem of its own, so a failed merge posts
// duplicates and never drops a finding. The error is ctx's, when it ends.
func mergeAIReviewFindings(ctx context.Context, merge mergeFunc, results []*aiReviewResult) ([][]*aiReviewFinding, error) {
	type sheetKey struct {
		specID      string
		sheetSha256 string
	}
	var order []sheetKey
	bySheet := make(map[sheetKey][]*aiReviewFinding)
	for _, result := range results {
		if result == nil {
			continue
		}
		key := sheetKey{specID: result.Unit.Check.SpecID, sheetSha256: result.Unit.Check.SheetSha256}
		for _, finding := range result.Findings {
			if _, ok := bySheet[key]; !ok {
				order = append(order, key)
			}
			bySheet[key] = append(bySheet[key], &aiReviewFinding{Unit: result.Unit, Finding: finding})
		}
	}
	var problems [][]*aiReviewFinding
	for _, key := range order {
		sheetProblems, err := mergeSheetFindings(ctx, merge, bySheet[key])
		if err != nil {
			return nil, err
		}
		problems = append(problems, sheetProblems...)
	}
	return problems, nil
}

// mergeSheetFindings groups the findings of one sheet, ordered by the first
// finding of each problem.
func mergeSheetFindings(ctx context.Context, merge mergeFunc, findings []*aiReviewFinding) ([][]*aiReviewFinding, error) {
	statement := findings[0].Unit.Statement
	blocks := statementBlocks(sheetEngine(findings), statement)
	blockOf := make([]int, len(findings))
	blockTargets := make(map[int]map[string]bool)
	for i, finding := range findings {
		blockOf[i] = findBlock(&blocks, finding.Finding.Line)
		if blockTargets[blockOf[i]] == nil {
			blockTargets[blockOf[i]] = make(map[string]bool)
		}
		blockTargets[blockOf[i]][finding.Unit.Check.Target] = true
	}

	request := &aireview.MergeRequest{Statement: statement, Blocks: blocks}
	// offered maps a merge candidate to its finding.
	var offered []int
	for i, finding := range findings {
		if len(blockTargets[blockOf[i]]) > 1 {
			offered = append(offered, i)
			request.Candidates = append(request.Candidates, aireview.MergeCandidate{
				Block:    blockOf[i],
				Database: finding.Unit.Check.Target,
				Finding:  finding.Finding,
			})
		}
	}

	var groups [][]int
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
			for _, group := range result.Groups {
				members := make([]int, 0, len(group))
				for _, candidate := range group {
					members = append(members, offered[candidate])
					merged[offered[candidate]] = true
				}
				groups = append(groups, members)
			}
		}
	}
	for i := range findings {
		if !merged[i] {
			groups = append(groups, []int{i})
		}
	}
	slices.SortFunc(groups, func(a, b []int) int { return cmp.Compare(a[0], b[0]) })

	problems := make([][]*aiReviewFinding, 0, len(groups))
	for _, group := range groups {
		problem := make([]*aiReviewFinding, 0, len(group))
		for _, i := range group {
			problem = append(problem, findings[i])
		}
		problems = append(problems, problem)
	}
	return problems, nil
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
// blocks inside which findings can merge. Without a splitter for the engine,
// or when the split does not give ordered ranges, the sheet is one block.
func statementBlocks(engine storepb.Engine, statement string) []aireview.LineRange {
	// The count is one too many when the sheet ends with a line break, which
	// is harmless at the end of the last block.
	whole := []aireview.LineRange{{Start: 1, End: strings.Count(statement, "\n") + 1}}
	if engine == storepb.Engine_ENGINE_UNSPECIFIED {
		return whole
	}
	statements, err := base.SplitMultiSQL(engine, statement)
	if err != nil {
		return whole
	}
	var blocks []aireview.LineRange
	for _, s := range statements {
		if s.Empty {
			continue
		}
		if s.Start == nil || s.End == nil {
			return whole
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
			return whole
		}
		blocks = append(blocks, aireview.LineRange{Start: start, End: end})
	}
	if len(blocks) == 0 {
		return whole
	}
	return blocks
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
// post: one OPEN root per problem, anchored to the line most of its findings
// start on, with every database it applies to. The title, priority, rule, and
// fix are those of its lead finding, and the evidence of every finding is
// kept, since each database has facts of its own.
func aiReviewComments(projectID string, issueUID int64, problems [][]*aiReviewFinding) []*store.IssueCommentMessage {
	comments := make([]*store.IssueCommentMessage, 0, len(problems))
	for _, findings := range problems {
		lead := leadFinding(findings)
		line := int32(commonLine(findings))
		var targets []string
		for _, finding := range findings {
			if !slices.Contains(targets, finding.Unit.Check.Target) {
				targets = append(targets, finding.Unit.Check.Target)
			}
		}
		slices.Sort(targets)
		comments = append(comments, &store.IssueCommentMessage{
			ProjectID: projectID,
			IssueUID:  issueUID,
			Payload: &storepb.IssueCommentPayload{
				Comment: aiReviewCommentText(lead, findings, targets),
				StatementAnchor: &storepb.IssueCommentPayload_StatementAnchor{
					SpecId:        lead.Unit.Check.SpecID,
					SheetSha256:   lead.Unit.Check.SheetSha256,
					StartPosition: &storepb.Position{Line: line},
					EndPosition:   &storepb.Position{Line: line},
				},
				ReviewMetadata: &storepb.IssueCommentPayload_ReviewMetadata{
					RunType:  storepb.ReviewRun_AI,
					Priority: aiReviewPriority(lead.Finding.Severity),
					Targets:  targets,
				},
			},
		})
	}
	return comments
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

// aiReviewCommentText renders a problem as the body of its thread. Evidence of
// more than one database is labeled with the database it comes from.
func aiReviewCommentText(lead *aiReviewFinding, findings []*aiReviewFinding, targets []string) string {
	var b strings.Builder
	_, _ = b.WriteString(lead.Finding.Title)
	byTarget := slices.Clone(findings)
	slices.SortStableFunc(byTarget, func(x, y *aiReviewFinding) int { return cmp.Compare(x.Unit.Check.Target, y.Unit.Check.Target) })
	labels := databaseLabels(targets)
	for _, finding := range byTarget {
		if len(targets) > 1 {
			_, _ = fmt.Fprintf(&b, "\n\n%s: %s", labels[finding.Unit.Check.Target], finding.Finding.Evidence)
		} else {
			_, _ = fmt.Fprintf(&b, "\n\n%s", finding.Finding.Evidence)
		}
	}
	_, _ = fmt.Fprintf(&b, "\n\nRule: %s", lead.Finding.Rule)
	_, _ = fmt.Fprintf(&b, "\n\nSuggested fix: %s", lead.Finding.Fix)
	return b.String()
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
