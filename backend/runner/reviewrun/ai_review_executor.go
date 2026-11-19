package reviewrun

import (
	"context"
	"log/slog"

	"github.com/pkg/errors"
	"golang.org/x/sync/semaphore"

	"github.com/bytebase/bytebase/backend/common/log"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
	"github.com/bytebase/bytebase/backend/plugin/aireview"
	"github.com/bytebase/bytebase/backend/runner/plancheck"
	"github.com/bytebase/bytebase/backend/store"
)

// AIReviewExecutor is the AI review: a model judges each spec's sheet against
// the workspace and project AI review policy, once per database of the
// project's CI sample, with that database's facts in the prompt and its synced
// schema behind the tools. The reviews of one run share the model and run
// concurrently under aiReviewRunConcurrency. A further call to the model
// merges the findings that more than one database reports on the same
// statement, tells which result an earlier run posted each problem continues,
// whose thread the problem keeps, and tells which finding of the standard
// rules each problem is on, whose result the problem joins. A run holds back
// the new problems past the maxAIReviewProblems most important open ones. One
// more call for each other merged problem it posts writes its comment.
type AIReviewExecutor struct {
	store    *store.Store
	license  LicenseChecker
	newModel func(*storepb.AISetting, ...aireview.ModelOption) aireview.Model
	// slots bounds the reviews in flight across every run of the executor.
	slots reviewSlots
	// ruleReviewCovers reports whether the standard rule review runs on an
	// engine. It is nil where SQL Review V2 is off: an AI review can still
	// run there, and no standard rule review runs beside it.
	ruleReviewCovers func(storepb.Engine) bool
}

// NewAIReviewExecutor creates the AI review executor. The server creates one,
// so aiReviewServerConcurrency bounds the reviews of every run on the server.
func NewAIReviewExecutor(s *store.Store, license LicenseChecker, ruleReviewCovers func(storepb.Engine) bool) *AIReviewExecutor {
	return &AIReviewExecutor{
		store:            s,
		license:          license,
		newModel:         aireview.NewModel,
		slots:            semaphore.NewWeighted(aiReviewServerConcurrency),
		ruleReviewCovers: ruleReviewCovers,
	}
}

// reviewWith runs the AI review beside rules, the results of the standard
// rules on the same change, and returns the results of both to post: each
// rule result with the AI review's problems on its findings' operations, and
// one result for every other problem, in no order. Collect-all, no fail-fast:
// every database of the sample is attempted, failures aggregate into one
// message, and the results are returned only when every one of them was
// reviewed.
func (e *AIReviewExecutor) reviewWith(ctx context.Context, projectID string, issueUID int64, rules []*ruleResult) ([]*store.IssueCommentMessage, error) {
	plan, project, err := loadReviewPlan(ctx, e.store, projectID, issueUID)
	if err != nil {
		return nil, err
	}
	// The review skips an AI review that is off, but the switch can change
	// before this read. Switched off, or on a plan without it, the AI review
	// reaches no provider and reports no finding, so the rules' results post
	// as they are.
	aiSetting, err := e.store.GetAISettingUncached(ctx, project.Workspace)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get AI setting")
	}
	allowed := aireview.CheckSetting(aiSetting) == nil &&
		e.license.IsFeatureEnabled(ctx, project.Workspace, v1pb.PlanFeature_FEATURE_AI_SQL_REVIEW) == nil
	if !allowed {
		return ruleResultComments(projectID, issueUID, rules), nil
	}
	policy, err := e.store.GetEffectiveReviewAIPolicy(ctx, project.Workspace, projectID)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get AI review policy")
	}
	// Without a standard rule review the policy of its rules means nothing,
	// so the AI review states no standard rule.
	var rulePolicy *storepb.ReviewRulePolicy
	if e.ruleReviewCovers != nil {
		rulePolicy, err = e.store.GetEffectiveReviewRulePolicy(ctx, project.Workspace, projectID)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get review rule policy")
		}
	}

	databaseGroup, err := plancheck.GetDatabaseGroupForPlan(ctx, e.store, plan, nil)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get database group for plan")
	}
	// Every database costs model calls, so the review covers the databases
	// plan checks sample when the project sets ci_sampling_size, and every
	// database otherwise.
	checkTargets, err := plancheck.DeriveCheckTargets(ctx, e.store, project, plan, databaseGroup)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to derive review targets")
	}
	checkTargets = uniqueCheckTargets(checkTargets)

	var unitErrs []error
	var units []*aiReviewUnit
	sheets := make(map[string]string)
	for _, checkTarget := range checkTargets {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		statement, err := e.sheetStatement(ctx, sheets, checkTarget.SheetSha256)
		if err != nil {
			unitErrs = append(unitErrs, errors.Wrapf(err, "%s", checkTarget.Target))
			continue
		}
		units = append(units, &aiReviewUnit{Check: checkTarget, Statement: statement, Policies: []string{policy.Workspace, policy.Project}})
	}

	// Each review is told the rule findings on its database, to add facts to.
	standard := standardFindings(rules)

	// The setting is read again before every provider call, retries included,
	// so switching the review off stops a run in flight at its next call.
	reviewer := aireview.NewReviewer(e.newModel(aiSetting, aireview.CheckBeforeEachCall(func(ctx context.Context) error {
		return e.checkAISetting(ctx, project.Workspace)
	})))
	results, reviewErrs := reviewUnits(ctx, e.slots, units, func(ctx context.Context, unit *aiReviewUnit) (*aireview.Result, error) {
		// The schema is loaded once the unit holds its slot and dropped when
		// the review ends, so the runs together hold at most one schema per
		// slot however many databases they review. The store's schema cache
		// keeps its own bounded copies.
		request := &aireview.Request{
			WorkspacePolicy:  policy.Workspace,
			ProjectPolicy:    policy.Project,
			Statement:        unit.Statement,
			StandardFindings: standardFindingsOn(standard[aiReviewSheet{specID: unit.Check.SpecID, sheetSha256: unit.Check.SheetSha256}], unit.Check.Target),
		}
		tools, engine, err := e.resolveAIReviewTarget(ctx, unit.Check, rulePolicy, request)
		if err != nil {
			return nil, err
		}
		unit.Engine = engine
		result, err := reviewer.Review(ctx, request, tools)
		if err != nil {
			return nil, err
		}
		// The notes name the facts the model could not get. They are for
		// the operator, never a finding.
		slog.Info("AI review completed",
			slog.String("project", projectID),
			slog.Int64("issue_id", issueUID),
			slog.String("spec", unit.Check.SpecID),
			slog.String("target", unit.Check.Target),
			slog.Int("model_calls", result.Calls),
			slog.Int("tokens", result.TotalTokens),
			slog.Int("findings", len(result.Findings)),
			slog.Int("facts", len(result.Facts)),
			slog.Any("notes", result.Notes))
		return result, nil
	})
	// A canceled context is shutdown: the scheduler leaves the run to the
	// reaper instead of failing it.
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	for _, err := range reviewErrs {
		if err != nil {
			unitErrs = append(unitErrs, err)
		}
	}
	if err := aggregateUnitErrors(len(checkTargets), unitErrs); err != nil {
		return nil, err
	}
	earlier, threads, err := e.earlierResults(ctx, projectID, issueUID, results, sheets, len(rules) > 0)
	if err != nil {
		return nil, err
	}
	for _, rule := range rules {
		rule.earlier = threads[rule.payload().GetReviewMetadata().GetFingerprint()]
	}
	problems, err := mergeAIReviewFindings(ctx, func(ctx context.Context, request *aireview.MergeRequest) (*aireview.MergeResult, error) {
		// The merge is a model call like a review, so it takes a slot too.
		if err := e.slots.Acquire(ctx, 1); err != nil {
			return nil, err
		}
		defer e.slots.Release(1)
		result, err := reviewer.MergeFindings(ctx, request)
		if err != nil {
			return nil, err
		}
		slog.Info("AI review merged findings",
			slog.String("project", projectID),
			slog.Int64("issue_id", issueUID),
			slog.Int("findings", len(request.Candidates)),
			slog.Int("problems", len(result.Groups)),
			slog.Int("model_calls", result.Calls),
			slog.Int("tokens", result.TotalTokens))
		return result, nil
	}, results, earlier, rules)
	if err != nil {
		return nil, err
	}
	// The limit comes before the summaries, so a held problem costs no model
	// call.
	if held := limitAIReviewProblems(plan.Config.GetSpecs(), problems); held > 0 {
		slog.Info("AI review held back the problems over the limit",
			slog.String("project", projectID),
			slog.Int64("issue_id", issueUID),
			slog.Int("problems", len(problems)),
			slog.Int("held", held))
	}
	if err := summarizeAIReviewProblems(ctx, func(ctx context.Context, request *aireview.SummaryRequest) (*aireview.Summary, error) {
		// A summary is a model call like a review, so it takes a slot too.
		if err := e.slots.Acquire(ctx, 1); err != nil {
			return nil, err
		}
		defer e.slots.Release(1)
		summary, err := reviewer.Summarize(ctx, request)
		if err != nil {
			return nil, err
		}
		slog.Info("AI review summarized a problem",
			slog.String("project", projectID),
			slog.Int64("issue_id", issueUID),
			slog.Int("findings", len(request.Findings)),
			slog.Int("groups", len(summary.Groups)),
			slog.Int("model_calls", summary.Calls),
			slog.Int("tokens", summary.TotalTokens))
		return summary, nil
	}, problems); err != nil {
		return nil, err
	}
	for _, problem := range problems {
		if problem.Rule == nil {
			continue
		}
		// The thread shows neither the severity nor, most times, the rule the
		// model gave the problem, so the log keeps them.
		lead := leadFinding(findingsByTarget(problem.Findings))
		slog.Info("AI review put a problem on a standard rule finding",
			slog.String("project", projectID),
			slog.Int64("issue_id", issueUID),
			slog.String("rule_type", problem.Rule.finding.rule.String()),
			slog.Int("findings", len(problem.Findings)),
			slog.String("severity", string(lead.Finding.Severity)),
			slog.String("cited_rule", lead.Finding.Rule),
			slog.Bool("unchanged", problem.Unchanged))
	}
	foldAIReviewProblems(problems)
	keepFactsOnlyComments(rules, problems)
	err = addRuleFacts(ctx, func(ctx context.Context, request *aireview.FactsRequest) (*aireview.FactsSummary, error) {
		// A summary is a model call like a review, so it takes a slot too.
		if err := e.slots.Acquire(ctx, 1); err != nil {
			return nil, err
		}
		defer e.slots.Release(1)
		summary, err := reviewer.SummarizeFacts(ctx, request)
		if err != nil {
			return nil, err
		}
		slog.Info("AI review summarized the facts of a rule finding",
			slog.String("project", projectID),
			slog.Int64("issue_id", issueUID),
			slog.Int("databases", len(request.Facts)),
			slog.Int("groups", len(summary.Groups)),
			slog.Int("model_calls", summary.Calls),
			slog.Int("tokens", summary.TotalTokens))
		return summary, nil
	}, standard, results, sheets)
	// The merge and the summaries fall back when a model call fails, the call
	// that a switch-off refused included, so the setting is read again before
	// the results post.
	if err == nil {
		err = e.checkAISetting(ctx, project.Workspace)
	}
	if err != nil {
		// The caller posts the rule results alone when the AI review is off.
		dropAIReview(rules)
		return nil, err
	}
	return append(aiReviewComments(projectID, issueUID, problems), ruleResultComments(projectID, issueUID, rules)...), nil
}

// checkAISetting reports why the workspace AI setting or the workspace's plan
// does not allow the review, if they do not. It reads past the setting cache,
// which only this process's writes refresh, so a switch-off written anywhere
// else stops the review too.
func (e *AIReviewExecutor) checkAISetting(ctx context.Context, workspace string) error {
	setting, err := e.store.GetAISettingUncached(ctx, workspace)
	if err != nil {
		return errors.Wrapf(err, "failed to get AI setting")
	}
	if err := aireview.CheckSetting(setting); err != nil {
		return err
	}
	return e.license.IsFeatureEnabled(ctx, workspace, v1pb.PlanFeature_FEATURE_AI_SQL_REVIEW)
}

// earlierResults returns, for each sheet with findings, the results earlier
// runs posted on its spec that a problem on it can keep: the AI reviewer's
// current roots on the issue that carry a fingerprint, with the sheet each is
// anchored to. A result whose sheet cannot be read is left out, so its problem
// is posted again. It also returns the thread of each current root of the
// standard rules, by fingerprint, when the run has rule results or any sheet
// has findings.
func (e *AIReviewExecutor) earlierResults(ctx context.Context, projectID string, issueUID int64, results []*aiReviewResult, sheets map[string]string, hasRules bool) (map[aiReviewSheet][]*aiReviewEarlier, map[string]*ruleThread, error) {
	bySpec := make(map[string]aiReviewSheet)
	for _, result := range results {
		if result != nil && len(result.Findings) > 0 {
			check := result.Unit.Check
			bySpec[check.SpecID] = aiReviewSheet{specID: check.SpecID, sheetSha256: check.SheetSha256}
		}
	}
	if len(bySpec) == 0 && !hasRules {
		return nil, nil, nil
	}

	roots, err := e.store.ListIssueComment(ctx, &store.FindIssueCommentMessage{ProjectID: projectID, IssueUID: &issueUID, TopLevelOnly: true})
	if err != nil {
		return nil, nil, errors.Wrapf(err, "failed to list earlier review results")
	}
	earlier := make(map[aiReviewSheet][]*aiReviewEarlier)
	threads := make(map[string]*ruleThread)
	for _, root := range roots {
		metadata, anchor := root.Payload.GetReviewMetadata(), root.Payload.GetStatementAnchor()
		if metadata.GetSuperseded() || metadata.GetFingerprint() == "" || root.ThreadState == nil {
			continue
		}
		// A result of a standard rule is the one that names its rule.
		if metadata.GetRuleType() != storepb.ReviewRuleType_REVIEW_RULE_TYPE_UNSPECIFIED {
			threads[metadata.GetFingerprint()] = &ruleThread{Comment: root.Payload.GetComment(), Priority: metadata.GetPriority()}
			continue
		}
		key, ok := bySpec[anchor.GetSpecId()]
		if !ok {
			continue
		}
		sheet, err := e.sheetStatement(ctx, sheets, anchor.GetSheetSha256())
		if err != nil {
			if ctx.Err() != nil {
				return nil, nil, ctx.Err()
			}
			slog.Warn("AI review could not read the sheet of an earlier result, posting its problem again",
				slog.String("project", projectID),
				slog.Int64("issue_id", issueUID),
				slog.String("sheet", anchor.GetSheetSha256()),
				log.BBError(err))
			continue
		}
		earlier[key] = append(earlier[key], &aiReviewEarlier{
			Fingerprint: metadata.GetFingerprint(),
			Resolved:    *root.ThreadState == store.ThreadStateResolved,
			Priority:    metadata.GetPriority(),
			Title:       metadata.GetTitle(),
			Comment:     root.Payload.GetComment(),
			Sheet:       sheet,
			SheetSha256: anchor.GetSheetSha256(),
			Line:        int(anchor.GetStartPosition().GetLine()),
		})
	}
	return earlier, threads, nil
}

// sheetStatement returns the sheet text, loaded once per run, and refuses a
// sheet over maxAIReviewSheetBytes.
func (e *AIReviewExecutor) sheetStatement(ctx context.Context, sheets map[string]string, sha256 string) (string, error) {
	if statement, ok := sheets[sha256]; ok {
		return statement, nil
	}
	sheet, err := e.store.GetSheetFull(ctx, sha256)
	if err != nil {
		return "", errors.Wrapf(err, "failed to get sheet")
	}
	if sheet == nil {
		return "", errors.Errorf("sheet %s not found", sha256)
	}
	if err := checkSheetSize(sheet.Statement); err != nil {
		return "", err
	}
	sheets[sha256] = sheet.Statement
	return sheet.Statement, nil
}

// resolveAIReviewTarget gathers what one review needs of a database: it sets
// the facts the prompt states as the request's target, and the standard rules
// the policy turns on with which of them the standard review evaluates on its
// engine, and returns the tools over the synced schema, sized to the
// request's prompt, and the database's engine. A database whose schema is not
// synced cannot be reviewed.
func (e *AIReviewExecutor) resolveAIReviewTarget(ctx context.Context, checkTarget *plancheck.CheckTarget, rulePolicy *storepb.ReviewRulePolicy, request *aireview.Request) (aireview.Tools, storepb.Engine, error) {
	instance, database, err := plancheck.ResolveDatabaseTarget(ctx, e.store, checkTarget.Target)
	if err != nil {
		return nil, storepb.Engine_ENGINE_UNSPECIFIED, err
	}
	dbSchema, err := e.store.GetDBSchema(ctx, &store.FindDBSchemaMessage{
		Workspace:    instance.Workspace,
		InstanceID:   database.InstanceID,
		DatabaseName: database.DatabaseName,
	})
	if err != nil {
		return nil, storepb.Engine_ENGINE_UNSPECIFIED, errors.Wrapf(err, "failed to get database schema")
	}
	if dbSchema == nil || dbSchema.GetProto() == nil {
		return nil, storepb.Engine_ENGINE_UNSPECIFIED, errMetadataNotSynced
	}
	schema := dbSchema.GetProto()
	request.Target = aiReviewTarget(instance, database, schema)
	engine := instance.Metadata.GetEngine()
	request.StandardReviewRuns = e.ruleReviewCovers != nil && e.ruleReviewCovers(engine)
	request.StandardRules = aiReviewStandardRules(rulePolicy, engine, request.StandardReviewRuns)
	return aireview.NewCatalogTools(engine, schema, aireview.WithPromptOf(request)), engine, nil
}
