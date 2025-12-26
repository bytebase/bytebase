# Event-Driven Rollout Creation Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Replace manual "Create Rollout" button with event-driven auto-creation when approval and plan checks complete.

**Architecture:** Add centralized `TryCreateRollout()` function that checks conditions (approval + plan checks). Hook into `ApproveIssue()` and plan check completion to call `TryCreateRollout()`. Make `CreateRollout()` idempotent to handle race conditions.

**Tech Stack:** Go, gRPC/Connect, PostgreSQL

---

### Task 1: Add Idempotency Check to CreateRollout

**Files:**
- Modify: `backend/api/v1/rollout_service.go:203-332`

**Step 1: Read current CreateRollout implementation**

Read `backend/api/v1/rollout_service.go` lines 203-234 to understand the current flow.

**Step 2: Add idempotency check**

Add this check immediately after fetching the plan (after line 233):

```go
// Idempotency check: prevent duplicate rollout creation
if plan.Config != nil && plan.Config.HasRollout {
	return nil, connect.NewError(connect.CodeAlreadyExists, errors.Errorf("rollout already exists for plan %s", request.GetRollout().GetPlan()))
}
```

**Step 3: Test manually**

Run: `go build -o ./bytebase-build/bytebase ./backend/bin/server/main.go`
Expected: Builds successfully

**Step 4: Commit**

```bash
git add backend/api/v1/rollout_service.go
git commit -m "feat: add idempotency check to CreateRollout

Prevent duplicate rollout creation by checking HasRollout flag.
Returns AlreadyExists error if rollout already exists.

🤖 Generated with [Claude Code](https://claude.com/claude-code)

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>"
```

---

### Task 2: Implement TryCreateRollout Function

**Files:**
- Modify: `backend/api/v1/rollout_service.go` (add new function after CreateRollout)

**Step 1: Add TryCreateRollout function**

Add this function after the `CreateRollout()` function (after line 332):

```go
// TryCreateRollout attempts to create a rollout if all conditions are met.
// This function is called asynchronously after approval or plan check completion.
// It checks approval and plan check conditions before calling CreateRollout.
func (s *RolloutService) TryCreateRollout(ctx context.Context, issueID int) {
	issue, err := s.store.GetIssue(ctx, &store.FindIssueMessage{UID: &issueID})
	if err != nil {
		slog.Error("failed to get issue for rollout creation",
			slog.Int("issue_id", issueID),
			log.BBError(err))
		return
	}
	if issue == nil {
		slog.Debug("issue not found for rollout creation", slog.Int("issue_id", issueID))
		return
	}

	if issue.PlanUID == nil {
		slog.Debug("issue has no plan, skipping rollout creation", slog.Int("issue_id", issueID))
		return
	}

	plan, err := s.store.GetPlan(ctx, &store.FindPlanMessage{UID: issue.PlanUID})
	if err != nil {
		slog.Error("failed to get plan for rollout creation",
			slog.Int("plan_id", int(*issue.PlanUID)),
			log.BBError(err))
		return
	}
	if plan == nil {
		slog.Debug("plan not found for rollout creation", slog.Int("plan_id", int(*issue.PlanUID)))
		return
	}

	// Idempotency: skip if rollout already exists
	if plan.Config != nil && plan.Config.HasRollout {
		slog.Debug("rollout already exists, skipping creation",
			slog.Int("issue_id", issueID),
			slog.Int("plan_id", int(*issue.PlanUID)))
		return
	}

	project, err := s.store.GetProject(ctx, &store.FindProjectMessage{ResourceID: &plan.ProjectID})
	if err != nil {
		slog.Error("failed to get project for rollout creation",
			slog.String("project_id", plan.ProjectID),
			log.BBError(err))
		return
	}
	if project == nil {
		slog.Error("project not found for rollout creation", slog.String("project_id", plan.ProjectID))
		return
	}

	// Check approval condition
	if project.Setting != nil && project.Setting.RequireIssueApproval {
		if issue.ApprovalStatus != storepb.Issue_APPROVED {
			slog.Debug("issue not approved yet, skipping rollout creation",
				slog.Int("issue_id", issueID),
				slog.String("approval_status", issue.ApprovalStatus.String()))
			return
		}
	}

	// Check plan check condition
	if project.Setting != nil && project.Setting.RequirePlanCheckNoError {
		planCheckRun, err := s.store.GetPlanCheckRun(ctx, *issue.PlanUID)
		if err != nil {
			slog.Error("failed to get plan check run for rollout creation",
				slog.Int("plan_id", int(*issue.PlanUID)),
				log.BBError(err))
			return
		}

		// If no plan checks exist, treat as passing (same as old behavior)
		if planCheckRun != nil {
			// Check if plan checks are still running
			if planCheckRun.Status == store.PlanCheckRunStatusRunning {
				slog.Debug("plan checks still running, skipping rollout creation",
					slog.Int("issue_id", issueID))
				return
			}

			// Check for ERROR-level results
			if planCheckRun.Result != nil {
				for _, result := range planCheckRun.Result.Results {
					if result.Status == storepb.Advice_ERROR {
						slog.Debug("plan checks have errors, skipping rollout creation",
							slog.Int("issue_id", issueID))
						return
					}
				}
			}
		}
	}

	// All conditions met - create the rollout
	slog.Info("auto-creating rollout",
		slog.Int("issue_id", issueID),
		slog.Int("plan_id", int(*issue.PlanUID)))

	projectID := common.FormatProject(plan.ProjectID)
	planName := common.FormatPlan(project.Name, *issue.PlanUID)

	_, err = s.CreateRollout(ctx, connect.NewRequest(&v1pb.CreateRolloutRequest{
		Parent: projectID,
		Rollout: &v1pb.Rollout{
			Plan: planName,
		},
	}))

	if err != nil {
		// If rollout already exists, this is not an error (race condition handled)
		if connect.CodeOf(err) == connect.CodeAlreadyExists {
			slog.Debug("rollout already exists (race condition), ignoring",
				slog.Int("issue_id", issueID))
			return
		}
		slog.Error("failed to auto-create rollout",
			slog.Int("issue_id", issueID),
			slog.Int("plan_id", int(*issue.PlanUID)),
			log.BBError(err))
		return
	}

	slog.Info("successfully auto-created rollout",
		slog.Int("issue_id", issueID),
		slog.Int("plan_id", int(*issue.PlanUID)))
}
```

**Step 2: Add required imports**

Add to imports at the top if not already present:
```go
"github.com/bytebase/bytebase/backend/common"
```

**Step 3: Build and verify**

Run: `go build -o ./bytebase-build/bytebase ./backend/bin/server/main.go`
Expected: Builds successfully

**Step 4: Commit**

```bash
git add backend/api/v1/rollout_service.go
git commit -m "feat: implement TryCreateRollout function

Add centralized function that checks all conditions (approval + plan
checks) and creates rollout if ready. Designed to be called
asynchronously from event handlers.

🤖 Generated with [Claude Code](https://claude.com/claude-code)

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>"
```

---

### Task 3: Hook TryCreateRollout into ApproveIssue

**Files:**
- Modify: `backend/api/v1/issue_service.go:32-40` (add rolloutService field)
- Modify: `backend/api/v1/issue_service.go:49-60` (update NewIssueService)
- Modify: `backend/api/v1/issue_service.go:634-822` (add trigger in ApproveIssue)

**Step 1: Add rolloutService field to IssueService struct**

Modify the IssueService struct (around line 32-40):

```go
// IssueService implements the issue service.
type IssueService struct {
	v1connect.UnimplementedIssueServiceHandler
	store          *store.Store
	webhookManager *webhook.Manager
	stateCfg       *state.State
	licenseService *enterprise.LicenseService
	profile        *config.Profile
	iamManager     *iam.Manager
	rolloutService *RolloutService  // Add this line
}
```

**Step 2: Update NewIssueService constructor**

Find the NewIssueService function and:
1. Add `rolloutService *RolloutService` parameter
2. Add `rolloutService: rolloutService` to the returned struct

**Step 3: Find where ApproveIssue function is defined**

Run: `grep -n "func.*ApproveIssue" backend/api/v1/issue_service.go`
Expected: Shows line number (around 634)

**Step 4: Read ApproveIssue to find where approval status changes**

Read lines 634-822 to find where the issue is updated with new approval status.

**Step 5: Add TryCreateRollout trigger after approval**

After the issue is updated and approval status becomes APPROVED, add:

```go
// Auto-create rollout if this approval completes the approval flow
if updatedIssue.ApprovalStatus == storepb.Issue_APPROVED {
	go func() {
		s.rolloutService.TryCreateRollout(ctx, updatedIssue.UID)
	}()
}
```

**Step 6: Build and verify**

Run: `go build -o ./bytebase-build/bytebase ./backend/bin/server/main.go`
Expected: May fail due to NewIssueService call sites not updated yet

**Step 7: Find all NewIssueService call sites**

Run: `grep -rn "NewIssueService" backend/ --include="*.go"`
Expected: Shows all places where NewIssueService is called

**Step 8: Update all NewIssueService call sites to pass rolloutService**

Update each call site to pass the rolloutService parameter.

**Step 9: Build and verify**

Run: `go build -o ./bytebase-build/bytebase ./backend/bin/server/main.go`
Expected: Builds successfully

**Step 10: Commit**

```bash
git add backend/api/v1/issue_service.go
git add <other files with NewIssueService calls>
git commit -m "feat: trigger rollout creation on approval

Call TryCreateRollout when issue becomes fully approved.
Add rolloutService dependency to IssueService.

🤖 Generated with [Claude Code](https://claude.com/claude-code)

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>"
```

---

### Task 4: Hook TryCreateRollout into Plan Check Completion

**Files:**
- Modify: `backend/runner/plancheck/scheduler.go:24-32` (add rolloutService field to Scheduler)
- Modify: `backend/runner/plancheck/scheduler.go:116-127` (add trigger in markPlanCheckRunDone)

**Step 1: Add rolloutService to plan check Scheduler**

Modify the Scheduler struct around line 34-40:

```go
// Scheduler is the plan check run scheduler.
type Scheduler struct {
	store          *store.Store
	licenseService *enterprise.LicenseService
	stateCfg       *state.State
	executor       *CombinedExecutor
	rolloutService interface {  // Add this field
		TryCreateRollout(ctx context.Context, issueID int)
	}
}
```

**Step 2: Update NewScheduler function**

Modify NewScheduler (around line 25-32):

```go
// NewScheduler creates a new plan check scheduler.
func NewScheduler(s *store.Store, licenseService *enterprise.LicenseService, stateCfg *state.State, executor *CombinedExecutor, rolloutService interface {
	TryCreateRollout(ctx context.Context, issueID int)
}) *Scheduler {
	return &Scheduler{
		store:          s,
		licenseService: licenseService,
		stateCfg:       stateCfg,
		executor:       executor,
		rolloutService: rolloutService,
	}
}
```

**Step 3: Add TryCreateRollout trigger in markPlanCheckRunDone**

Modify markPlanCheckRunDone function (around line 116-127):

```go
func (s *Scheduler) markPlanCheckRunDone(ctx context.Context, planCheckRun *store.PlanCheckRunMessage, results []*storepb.PlanCheckRunResult_Result) {
	result := &storepb.PlanCheckRunResult{
		Results: results,
	}
	if err := s.store.UpdatePlanCheckRun(ctx,
		store.PlanCheckRunStatusDone,
		result,
		planCheckRun.UID,
	); err != nil {
		slog.Error("failed to mark plan check run done", log.BBError(err))
		return  // Add return here
	}

	// Auto-create rollout if plan checks pass
	issue, err := s.store.GetIssue(ctx, &store.FindIssueMessage{PlanUID: &planCheckRun.PlanUID})
	if err != nil {
		slog.Error("failed to get issue for rollout creation after plan check",
			slog.Int("plan_id", int(planCheckRun.PlanUID)),
			log.BBError(err))
		return
	}
	if issue != nil {
		go func() {
			s.rolloutService.TryCreateRollout(ctx, issue.UID)
		}()
	}
}
```

**Step 4: Add required imports**

Add to imports at top:
```go
"log/slog"
"github.com/bytebase/bytebase/backend/common/log"
```

**Step 5: Find all NewScheduler call sites**

Run: `grep -rn "plancheck.NewScheduler" backend/ --include="*.go"`
Expected: Shows where plancheck.NewScheduler is called

**Step 6: Update NewScheduler call sites to pass rolloutService**

Update each call site to pass the rolloutService parameter.

**Step 7: Build and verify**

Run: `go build -o ./bytebase-build/bytebase ./backend/bin/server/main.go`
Expected: Builds successfully

**Step 8: Commit**

```bash
git add backend/runner/plancheck/scheduler.go
git add <other files with NewScheduler calls>
git commit -m "feat: trigger rollout creation on plan check completion

Call TryCreateRollout when plan checks complete with DONE status.
Add rolloutService dependency to plan check Scheduler.

🤖 Generated with [Claude Code](https://claude.com/claude-code)

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>"
```

---

### Task 5: Run Tests and Lint

**Step 1: Run Go linter**

Run: `golangci-lint run --allow-parallel-runners`
Expected: No errors (or fixable errors)

**Step 2: Auto-fix linter issues**

Run: `golangci-lint run --fix --allow-parallel-runners`
Expected: Auto-fixes issues

**Step 3: Re-run linter until clean**

Run: `golangci-lint run --allow-parallel-runners`
Repeat until: No errors

**Step 4: Format Go code**

Run: `gofmt -w backend/api/v1/rollout_service.go backend/api/v1/issue_service.go backend/runner/plancheck/scheduler.go`
Expected: Files formatted

**Step 5: Run relevant tests**

Run: `go test -v -count=1 github.com/bytebase/bytebase/backend/api/v1 -run CreateRollout`
Expected: Tests pass

**Step 6: Build final binary**

Run: `go build -ldflags "-w -s" -p=16 -o ./bytebase-build/bytebase ./backend/bin/server/main.go`
Expected: Builds successfully

**Step 7: Commit any fixes**

```bash
git add .
git commit -m "chore: fix linting and formatting issues

🤖 Generated with [Claude Code](https://claude.com/claude-code)

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>"
```

---

### Task 6: Manual Integration Testing

**Step 1: Start development server**

Run: `PG_URL=postgresql://bbdev@localhost/bbdev ./bytebase-build/bytebase --port 8080 --data . --debug`
Expected: Server starts

**Step 2: Create test issue with approval required**

1. Create a project with `RequireIssueApproval = true`
2. Create a plan with database change
3. Create an issue from the plan
4. Verify no rollout exists yet

**Step 3: Approve the issue**

1. Approve the issue (if plan checks also configured, they should be passing)
2. Verify rollout is created automatically
3. Check logs for "auto-creating rollout" message

**Step 4: Test plan check trigger**

1. Create another issue with approval already done
2. Wait for plan checks to complete
3. Verify rollout is created automatically
4. Check logs for "auto-creating rollout" message

**Step 5: Test idempotency**

1. Try to manually create rollout again via UI button
2. Verify AlreadyExists error is returned
3. Verify no duplicate rollouts created

**Step 6: Test race condition**

1. Create issue where approval and plan checks complete simultaneously
2. Verify only one rollout is created
3. Check logs for any race condition messages

**Step 7: Document test results**

Create a test results document:

```markdown
# Event-Driven Rollout Testing Results

## Test 1: Approval Trigger
- Status: PASS/FAIL
- Notes: ...

## Test 2: Plan Check Trigger
- Status: PASS/FAIL
- Notes: ...

## Test 3: Idempotency
- Status: PASS/FAIL
- Notes: ...

## Test 4: Race Condition
- Status: PASS/FAIL
- Notes: ...
```

---

## Completion Checklist

- [ ] Task 1: Idempotency check added to CreateRollout
- [ ] Task 2: TryCreateRollout function implemented
- [ ] Task 3: ApproveIssue hook added
- [ ] Task 4: Plan check completion hook added
- [ ] Task 5: Tests and linting pass
- [ ] Task 6: Manual integration tests complete
- [ ] All code formatted with gofmt
- [ ] All linter issues resolved
- [ ] Build succeeds
- [ ] Feature tested in development environment

## Notes

- The manual "Create Rollout" button in the UI remains functional as a fallback
- No frontend changes are required
- All errors in TryCreateRollout are logged but not propagated (fire-and-forget design)
- Race conditions are handled via idempotency check and database transactions
