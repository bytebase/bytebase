# bytebase-e2e-testing Skill Validation Report

**Date:** 2026-04-13
**Skill version:** v1.5.0 (team-skills PR #28)
**Target pages:** Plan Detail + Rollout (unified page)
**Approach:** D — Blind run first, then gap-fill with BYT-9158 status audit

## Summary

The `bytebase-e2e-testing` skill was run against the plan detail / rollout pages on a fresh `--demo` Bytebase instance. The validation assessed:
1. Whether the skill's phases (0–5) work end-to-end
2. Whether Phase 3 exploratory QA catches the bugs the skill was designed to catch (BYT-9158 class)
3. What gaps exist in the skill's instructions

**Result: The skill works as designed, but has two critical gaps:**
1. **It doesn't instruct the tester to adapt the environment** (change settings, create data, switch users) to test CUJs that require specific preconditions.
2. **It doesn't instruct the tester to create data with multiplicity and overlap** — cross-view bugs like BYT-9160 are invisible with minimal (N=1) data.

## Phases Executed

| Phase | Status | Notes |
|---|---|---|
| 0: Environment setup | ✅ | playwright-cli detected, test runner ready, framework docs read |
| 1: Understand the feature | ✅ | Component tree, state model, polling, shared data sources mapped via Explore subagent |
| 2: Discover and triage CUJs | ✅ | 5 CUJs agreed (happy path, checks-failed blocker, approval-pending, task failure, detail panel) |
| 3: Exploratory QA (blind) | ⚠️ | See "Blind Run vs. Redo" below |
| 4: Write tests | ✅ | 3 spec files + 1 POM written in `frontend/tests/e2e/plan-detail/` |
| 5: Offer to fix bugs | ✅ | BYT-9160 reproduced (NOT FIXED); BYT-9159, 9161 verified fixed |

## Blind Run vs. Redo

### Blind run (first attempt)
Only CUJ 1 (happy path) and CUJ 6 (detail panel toggle) were testable. CUJs 2, 3, 4 were skipped because the demo project had both `requireIssueApproval` and `requirePlanCheckNoError` set to false (Optional), so the blocker states couldn't be reached.

**Findings from blind run:**
- F1 (Minor): Small click targets (18×14 external-link icons) — cosmetic, not functional
- F2 (Minor): GetPlanCheckRun/GetIamPolicy 404 polling — backend data issue, not UI bug
- F3, F4: Withdrawn after triage — FAILED/CANCELED are intentionally recoverable, not terminal

### Redo (with environment adaptation principle)
After adding the principle "adapt the environment to test the CUJ, don't skip the CUJ," all 5 CUJs were tested:

| CUJ | Environment Adaptation | Result |
|---|---|---|
| 1: Happy path | None (demo plan 101) | ✅ Pass |
| 2: Approved + checks failed | Set `requirePlanCheckNoError=true`, upgraded `COLUMN_NO_NULL` to ERROR, approved via 2-step API | ✅ Pass |
| 3: Approval pending | Created ALTER TABLE plan targeting prod, ran checks to trigger approval flow classification | ✅ Pass |
| 4: Task failure | Created plan with `ALTER TABLE nonexistent_table_xyz` | ✅ Pass |
| 6: Detail panel toggle | Clicked "View details" after rollout | ✅ Pass |

## BYT-9158 Bug Status

| Bug | Description | Status | How Verified |
|---|---|---|---|
| BYT-9159 | Rollout button missing when approved + failed checks | **FIXED** | Tested all 4 combinations of requireApproval × requireChecks with approved+failed state. Button correctly appears when checks optional, hidden when required. |
| BYT-9160 | Plan checks inconsistent across sections | **NOT FIXED — reproduced** | Created plan with 2 specs: spec #1 targets hr_prod (3 success results), spec #2 targets hr_prod+hr_test (5 success results). Sidebar shows spec #2's counts (W:1 S:5) even when spec #1 is selected (inline shows W:1 S:3). Sidebar `PlanCheckStatusCount` does not re-scope to the selected spec. |
| BYT-9161 | Section collapses on panel open | **FIXED** | Sections remained expanded after rollout creation, task detail open, and navigation away/back. |
| BYT-9162 | Auto-refresh continues after DONE | **Code review item** (not E2E) | `isPlanDone` correctly checks DONE/SKIPPED. FAILED/CANCELED are intentionally not terminal (user can retry). Verified during triage that this is correct behavior. |
| BYT-9164 | Click target too small | **Minor** | 18×14 external-link icons still exist. Cosmetic issue on secondary elements, not functional. |

## Skill Gaps

### Gap 1: Environment Adaptation

**The skill doesn't instruct the tester to adapt the environment to reach required states.**

The skill's `checks.md` Step 1.3 says "set up the state (via API, direct URL params, or UI actions)" but does not instruct the tester to:
1. Identify project/workspace **settings** that gate the feature under test
2. **Toggle those settings** and re-test the same flows
3. Test the **cross-product** of interacting settings
4. **Create data** that triggers specific states (plans with specific SQL, approval flows via specific user roles)
5. **Switch users** to test role-dependent behavior

Without this principle, the skill produces a false "all tests passed" on a demo instance that happens to have permissive defaults.

### Proposed Fix

Add to `checks.md` Section 1 (State Combination Testing):

> **Step 1.5: Configuration-driven state testing.** Many UI behaviors are gated by project or workspace settings. Before declaring state combination testing complete:
> 1. Read the component code to identify settings that affect the feature (grep for `project.value.require*`, `workspace.*setting`, permission checks)
> 2. For each setting, test with the setting both enabled and disabled
> 3. Test the cross-product of settings that interact (e.g., approval required × checks required = 4 combinations)
> 4. Use the API to change settings, create data in specific states, and authenticate as different user roles
> 5. The principle: **adapt the environment to test the CUJ, don't skip the CUJ because the environment doesn't match**

### Environment Adaptations Used in This Validation

| Adaptation | API Call | Purpose |
|---|---|---|
| Change project settings | `PATCH /v1/projects/{project}?update_mask=require_issue_approval,require_plan_check_no_error` | Enable/disable approval and check requirements |
| Change SQL review rule severity | `PATCH /v1/reviewConfigs/{config}?update_mask=rules` | Upgrade COLUMN_NO_NULL from WARNING to ERROR |
| Create sheets with specific SQL | `POST /v1/projects/{project}/sheets` | Provide ALTER TABLE, DROP TABLE, nonexistent table SQL |
| Create plans via API | `POST /v1/projects/{project}/plans` | Set up specific test scenarios |
| Create issues via API | `POST /v1/projects/{project}/issues` | Trigger approval workflow |
| Run plan checks via API | `POST /v1/{plan}:runPlanChecks` | Trigger SQL classification and approval flow generation |
| Approve as different users | `POST /v1/{issue}:approve` with different tokens | Complete multi-step approval (Project Owner + Workspace DBA) |

### Gap 2: Data Multiplicity and Overlap

**The skill doesn't instruct the tester to create data with sufficient complexity to surface cross-view bugs.**

The blind run used the demo plan 101 — a single spec targeting a single database (N=1). BYT-9160 only appears with N=2+ specs sharing overlapping database targets. At N=1, the inline and sidebar views trivially agree because there's only one data scope.

**Proposed fix — add to `checks.md` Section 2 (Cross-View Data Consistency):**

> **Step 2.0: Create data with multiplicity and overlap.** Cross-view bugs are invisible with minimal data. Before testing data consistency:
> 1. Ensure the feature has **3+ items** in any repeated section (specs, stages, tasks, rules, labels)
> 2. Create **overlapping references** between items — e.g., two specs targeting the same database, two roles assigned to the same user, two rules matching the same table
> 3. When items have sub-counts (check results, task counts, affected rows), make the counts **different per item** so scoping bugs produce visible mismatches
> 4. Switch between items and verify that ALL views (inline section, sidebar, panel, badges, tooltips) update to reflect the selected item — not the previous or aggregated data

### BYT-9160 Reproduction Details

**Conditions:**
1. Plan with 2 specs
2. Spec #1: targets `instances/prod-sample-instance/databases/hr_prod` (ALTER TABLE ADD COLUMN col1 TEXT)
3. Spec #2: targets `instances/prod-sample-instance/databases/hr_prod` + `instances/test-sample-instance/databases/hr_test` (ALTER TABLE ADD COLUMN col2 INTEGER)
4. Plan checks run → spec #1 gets 4 results (W:1 S:3), spec #2 gets 6 results (W:1 S:5)

**Observed behavior:**
| Selected spec | Inline checks | Sidebar checks | Match? |
|---|---|---|---|
| Spec #1 | W:1, S:3 | W:1, S:5 | **MISMATCH** |
| Spec #2 | W:1, S:5 | W:1, S:5 | Match |

**Root cause (likely):** Sidebar `PlanCheckStatusCount` reads plan-level check data without filtering by the currently selected spec. The inline component correctly scopes per-spec.

## Other Observations

### What worked well
- Phase 0 tool detection (playwright-cli vs MCP) worked correctly
- Phase 1 code reading via Explore subagent produced a thorough map
- Phase 2 CUJ discovery was comprehensive for the feature
- Phase 3 check procedures (section preservation, cross-view data, responsive layout) all produced clean results
- Phase 4 test generation patterns (POM, shared browser, serial mode) are well-documented

### What needs improvement
- The skill should explicitly state the environment adaptation principle in Phase 3 (Gap 1)
- The skill should instruct testers to create data with multiplicity and overlap for cross-view testing (Gap 2)
- The skill should provide example API calls for common Bytebase setup operations (project settings, sheet creation, plan creation, issue approval)
- Plan check running needs to happen BEFORE the approval flow is classified — this sequencing isn't obvious

### Triage process
The initial blind run produced 4 findings (F1–F4). After proper triage:
- F1: Downgraded to Minor (cosmetic)
- F2: Downgraded to Minor (backend data issue)
- F3: Withdrawn (correct behavior — FAILED is recoverable)
- F4: Withdrawn (correct behavior — consistent with F3)

This demonstrates the importance of triage before writing lock tests. Without triage, we would have written lock tests for F3/F4 that encode incorrect expectations.
