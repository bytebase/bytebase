package reviewrun

import (
	"context"
	"fmt"
	"log/slog"
	"slices"
	"strings"

	metadatapb "github.com/bytebase/omni/metadata"
	"github.com/pkg/errors"
	"golang.org/x/sync/errgroup"

	"github.com/bytebase/bytebase/backend/common/log"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/plugin/aireview"
	"github.com/bytebase/bytebase/backend/runner/plancheck"
	"github.com/bytebase/bytebase/backend/store"
)

const (
	// maxAIReviewSheetBytes bounds the sheet one review reads whole.
	maxAIReviewSheetBytes = 256 << 10
	// aiReviewConcurrency caps the reviews of one run in flight at the model.
	aiReviewConcurrency = 8
)

// aiReviewUnit is one AI review: a spec's sheet against one database, with
// the facts the prompt states about that database.
type aiReviewUnit struct {
	Check     *plancheck.CheckTarget
	Target    aireview.Target
	Statement string
}

// aiReviewResult is the findings of one completed unit.
type aiReviewResult struct {
	Unit     *aiReviewUnit
	Findings []aireview.Finding
}

// aiReviewTarget states what the prompt says about a database: the engine,
// its version, the environment, and every schema with its object count.
func aiReviewTarget(instance *store.InstanceMessage, database *store.DatabaseMessage, schema *metadatapb.DatabaseSchemaMetadata) aireview.Target {
	target := aireview.Target{
		Engine:  instance.Metadata.GetEngine().String(),
		Version: instance.Metadata.GetVersion(),
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

// aiReviewComments converts the findings of the completed units into the
// review result comments to post: one OPEN root per finding, anchored to the
// line it starts on in the spec's sheet, with the priority of its severity and
// every database it applies to. The same title on the same line of a spec
// across databases is one finding: its comment lists every database, carries
// the highest priority among them, and keeps the evidence of the first.
func aiReviewComments(projectID string, issueUID int64, results []*aiReviewResult) []*store.IssueCommentMessage {
	type key struct {
		specID string
		line   int
		title  string
	}
	merged := make(map[key]*store.IssueCommentMessage)
	var comments []*store.IssueCommentMessage
	for _, result := range results {
		if result == nil {
			continue
		}
		for _, finding := range result.Findings {
			k := key{specID: result.Unit.Check.SpecID, line: finding.Line, title: finding.Title}
			comment, ok := merged[k]
			if !ok {
				comment = &store.IssueCommentMessage{
					ProjectID: projectID,
					IssueUID:  issueUID,
					Payload: &storepb.IssueCommentPayload{
						Comment: aiReviewCommentText(finding),
						StatementAnchor: &storepb.IssueCommentPayload_StatementAnchor{
							SpecId:        result.Unit.Check.SpecID,
							SheetSha256:   result.Unit.Check.SheetSha256,
							StartPosition: &storepb.Position{Line: int32(finding.Line)},
							EndPosition:   &storepb.Position{Line: int32(finding.Line)},
						},
						ReviewMetadata: &storepb.IssueCommentPayload_ReviewMetadata{
							RunType:  storepb.ReviewRun_AI,
							Priority: aiReviewPriority(finding.Severity),
						},
					},
				}
				merged[k] = comment
				comments = append(comments, comment)
			}
			metadata := comment.Payload.ReviewMetadata
			if !slices.Contains(metadata.Targets, result.Unit.Check.Target) {
				metadata.Targets = append(metadata.Targets, result.Unit.Check.Target)
			}
			// The enum orders P0 before P1 before P2, so the smaller value is
			// the higher priority.
			if priority := aiReviewPriority(finding.Severity); priority < metadata.Priority {
				metadata.Priority = priority
			}
		}
	}
	for _, comment := range comments {
		slices.Sort(comment.Payload.ReviewMetadata.Targets)
	}
	return comments
}

// aiReviewCommentText renders a finding as the body of its thread.
func aiReviewCommentText(finding aireview.Finding) string {
	var b strings.Builder
	_, _ = b.WriteString(finding.Title)
	_, _ = fmt.Fprintf(&b, "\n\n%s", finding.Evidence)
	_, _ = fmt.Fprintf(&b, "\n\nRule: %s", finding.Rule)
	_, _ = fmt.Fprintf(&b, "\n\nSuggested fix: %s", finding.Fix)
	return b.String()
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

// reviewUnits runs review over the units, at most aiReviewConcurrency at a
// time, and returns each unit's result or error by index. A panic in one
// review is that unit's error, so the run fails instead of the process: the
// scheduler's recovery covers only the goroutine that called RunOnce.
func reviewUnits(ctx context.Context, units []*aiReviewUnit, review func(context.Context, *aiReviewUnit) (*aireview.Result, error)) ([]*aiReviewResult, []error) {
	results := make([]*aiReviewResult, len(units))
	errs := make([]error, len(units))
	var group errgroup.Group
	group.SetLimit(aiReviewConcurrency)
	for i, unit := range units {
		group.Go(func() error {
			defer func() {
				if r := recover(); r != nil {
					slog.Error("AI review PANIC RECOVER", slog.String("target", unit.Check.Target), slog.Any("panic", r), log.BBStack("panic-stack"))
					errs[i] = errors.Errorf("%s: review panic: %v", unit.Check.Target, r)
				}
			}()
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
