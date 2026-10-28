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
}

// aiReviewResult is the findings of one completed unit.
type aiReviewResult struct {
	Unit     *aiReviewUnit
	Findings []aireview.Finding
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

// aiReviewOccurrence is one database's finding of a problem.
type aiReviewOccurrence struct {
	Target  string
	Finding aireview.Finding
}

// aiReviewComments converts the findings of the completed units into the
// review result comments to post: one OPEN root per problem, anchored to the
// line it starts on in the spec's sheet. Findings that cite the same rule on
// the same line of a spec are one problem across databases, since a rule
// holds one kind of fix. Its comment lists every database it applies to, and
// carries the highest priority among them and each database's evidence.
func aiReviewComments(projectID string, issueUID int64, results []*aiReviewResult, rules []aireview.Rule) []*store.IssueCommentMessage {
	type key struct {
		specID string
		line   int
		rule   string
		// repeat tells apart the findings of one database that cite the same
		// rule on the same line, which the reviewer accepts only when the
		// model keeps them after a correction.
		repeat int
	}
	type problem struct {
		check       *plancheck.CheckTarget
		line        int
		rule        string
		occurrences []aiReviewOccurrence
	}
	problems := make(map[key]*problem)
	var ordered []*problem
	reviewed := make(map[string][]string)
	for _, result := range results {
		if result == nil {
			continue
		}
		check := result.Unit.Check
		reviewed[check.SpecID] = append(reviewed[check.SpecID], check.Target)
		repeats := make(map[key]int)
		for _, finding := range result.Findings {
			k := key{specID: check.SpecID, line: finding.Line, rule: finding.Rule}
			repeat := repeats[k]
			repeats[k]++
			k.repeat = repeat
			p, ok := problems[k]
			if !ok {
				p = &problem{check: check, line: finding.Line, rule: finding.Rule}
				problems[k] = p
				ordered = append(ordered, p)
			}
			p.occurrences = append(p.occurrences, aiReviewOccurrence{Target: check.Target, Finding: finding})
		}
	}

	ruleTexts := make(map[string]string, len(rules))
	for _, rule := range rules {
		ruleTexts[rule.ID] = rule.Text
	}
	comments := make([]*store.IssueCommentMessage, 0, len(ordered))
	for _, p := range ordered {
		// The most severe finding leads, and among equals the first database
		// by name.
		slices.SortFunc(p.occurrences, func(a, b aiReviewOccurrence) int {
			return cmp.Or(
				cmp.Compare(aiReviewPriority(a.Finding.Severity), aiReviewPriority(b.Finding.Severity)),
				cmp.Compare(a.Target, b.Target),
			)
		})
		targets := make([]string, 0, len(p.occurrences))
		for _, occurrence := range p.occurrences {
			targets = append(targets, occurrence.Target)
		}
		slices.Sort(targets)
		ruleText, ok := ruleTexts[p.rule]
		if !ok {
			ruleText = p.rule
		}
		comments = append(comments, &store.IssueCommentMessage{
			ProjectID: projectID,
			IssueUID:  issueUID,
			Payload: &storepb.IssueCommentPayload{
				Comment: aiReviewCommentText(p.occurrences, ruleText, reviewed[p.check.SpecID]),
				StatementAnchor: &storepb.IssueCommentPayload_StatementAnchor{
					SpecId:        p.check.SpecID,
					SheetSha256:   p.check.SheetSha256,
					StartPosition: &storepb.Position{Line: int32(p.line)},
					EndPosition:   &storepb.Position{Line: int32(p.line)},
				},
				ReviewMetadata: &storepb.IssueCommentPayload_ReviewMetadata{
					RunType:  storepb.ReviewRun_AI,
					Priority: aiReviewPriority(p.occurrences[0].Finding.Severity),
					Targets:  targets,
				},
			},
		})
	}
	return comments
}

// maxNamedUnflaggedDatabases is how many databases that did not report a
// problem its comment names; more are counted instead.
const maxNamedUnflaggedDatabases = 5

// aiReviewCommentText renders a problem as the body of its thread, from its
// occurrences with the leading one first. On a spec reviewed against one
// database it reads as that database's finding. Otherwise each database's
// evidence gets a line, as the facts differ between databases, and the
// databases that did not report the problem are named.
func aiReviewCommentText(occurrences []aiReviewOccurrence, ruleText string, reviewed []string) string {
	lead := occurrences[0].Finding
	var b strings.Builder
	_, _ = b.WriteString(lead.Title)
	if len(reviewed) <= 1 {
		_, _ = fmt.Fprintf(&b, "\n\n%s", lead.Evidence)
	} else {
		labels := databaseLabels(reviewed)
		_, _ = b.WriteString("\n\nEvidence by database:\n")
		writeEvidence(&b, occurrences, labels)
		var unflagged []string
		for _, target := range reviewed {
			if !slices.ContainsFunc(occurrences, func(o aiReviewOccurrence) bool { return o.Target == target }) {
				unflagged = append(unflagged, labels[target])
			}
		}
		switch {
		case len(unflagged) == 0:
		case len(unflagged) <= maxNamedUnflaggedDatabases:
			slices.Sort(unflagged)
			_, _ = fmt.Fprintf(&b, "\n\nNot flagged on %s.", strings.Join(unflagged, ", "))
		default:
			_, _ = fmt.Fprintf(&b, "\n\nNot flagged on %d other databases.", len(unflagged))
		}
	}
	_, _ = fmt.Fprintf(&b, "\n\nRule: %s", ruleText)
	_, _ = fmt.Fprintf(&b, "\n\nSuggested fix: %s", lead.Fix)
	return b.String()
}

// writeEvidence writes one list item per distinct evidence, naming the
// databases that reported it, and their severity when it differs from the
// leading one's.
func writeEvidence(b *strings.Builder, occurrences []aiReviewOccurrence, labels map[string]string) {
	type entry struct {
		severity aireview.Severity
		evidence string
	}
	databases := make(map[entry][]string)
	var entries []entry
	for _, occurrence := range occurrences {
		// A list item holds one line.
		e := entry{severity: occurrence.Finding.Severity, evidence: strings.Join(strings.Fields(occurrence.Finding.Evidence), " ")}
		if _, ok := databases[e]; !ok {
			entries = append(entries, e)
		}
		databases[e] = append(databases[e], labels[occurrence.Target])
	}
	for i, e := range entries {
		if i > 0 {
			_ = b.WriteByte('\n')
		}
		_, _ = fmt.Fprintf(b, "- %s", strings.Join(databases[e], ", "))
		if e.severity != occurrences[0].Finding.Severity {
			_, _ = fmt.Fprintf(b, " (%s)", e.severity)
		}
		_, _ = fmt.Fprintf(b, ": %s", e.evidence)
	}
}

// databaseLabels names each target by its database, and by instance and
// database when two targets share a database name.
func databaseLabels(targets []string) map[string]string {
	names := make(map[string]string, len(targets))
	counts := make(map[string]int, len(targets))
	for _, target := range targets {
		names[target] = target
		if _, database, err := common.GetInstanceDatabaseID(target); err == nil {
			names[target] = database
		}
		counts[names[target]]++
	}
	labels := make(map[string]string, len(targets))
	for _, target := range targets {
		labels[target] = names[target]
		if counts[names[target]] > 1 {
			if instance, database, err := common.GetInstanceDatabaseID(target); err == nil {
				labels[target] = instance + "/" + database
			}
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
