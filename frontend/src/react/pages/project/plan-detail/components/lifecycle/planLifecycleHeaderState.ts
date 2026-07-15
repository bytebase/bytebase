// Resolver for the plan-detail header's single "lifecycle slot" (BYT-9722).
//
// One pure function reads the plan's current state and decides the single safe
// next action / status the header shows. It is evaluated top to bottom: the
// first matching state owns the slot. Async, IAM-derived inputs (whether it is
// the user's turn to review, whether they may run the frontier stage) are
// computed by usePlanLifecycleHeader and passed in as plain values so this stays
// pure and unit-testable.
import { ApprovalStatus, State } from "@/types/proto-es/v1/common_pb";
import { IssueStatus } from "@/types/proto-es/v1/issue_service_pb";
import type { Rollout, Stage } from "@/types/proto-es/v1/rollout_service_pb";
import type { PlanCheckSummary } from "../../utils/phaseSummary";
import {
  getFrontierStage,
  stageHasRunnableTasks,
  stageHasRunningTasks,
} from "./frontierStage";

// The blocker/readiness label shown when the slot is a read-only status rather
// than an advance action. The component maps each reason to copy + tone.
// One gate at a time: review is the active gate first ("In review" regardless of
// checks); checks only become the headline once review has passed.
export type PlanStatusReason =
  | "checking" // approved, plan checks still running
  | "in-review" // review pending (the active gate)
  | "rejected" // review rejected
  | "checks-failing"; // review approved, but plan checks failed

export type PlanLifecycleHeaderState =
  // Nothing safe to surface (GitOps with no rollout, inconsistent backend
  // state). The slot renders empty and detail stays in the sections.
  | { kind: "none" }
  // Draft & review setup.
  | { kind: "create" }
  | { kind: "ready-for-review" }
  | { kind: "incomplete" }
  | { kind: "closed" } // terminal stamp
  // Review.
  | { kind: "review-generating" } // disabled + loading
  | { kind: "review-your-turn" }
  | { kind: "plan-status"; reason: PlanStatusReason }
  // Pre-deploy → rollout creation.
  | { kind: "preparing-rollout" }
  // Deploy (frontier stage). Run permission is enforced by the run confirmation
  // sheet (canRolloutTasks), so the resolver only decides which stage is the
  // frontier advance — it does not gate on IAM.
  | { kind: "run-stage"; stage: Stage }
  | { kind: "running-stage"; stage: Stage }
  | { kind: "deployed" }; // terminal stamp

export interface PlanLifecycleResolverInput {
  isCreating: boolean;
  isGitOpsPlan: boolean;
  readonly: boolean;
  planState: State;
  hasIssue: boolean;

  issueStatus: IssueStatus | undefined;
  issueDraft: boolean;
  approvalStatus: ApprovalStatus;
  hasCurrentStep: boolean;
  isCurrentUserCandidate: boolean;

  checks: PlanCheckSummary;

  hasRollout: boolean;
  rollout: Rollout | undefined;
}

function resolveDeploy(
  input: PlanLifecycleResolverInput
): PlanLifecycleHeaderState {
  const { rollout } = input;
  // A rollout without deployable stages is an inconsistent backend state, not a
  // user workflow — defer to the Deploy section rather than guess an action.
  if (!rollout || rollout.stages.length === 0) {
    return { kind: "none" };
  }
  const frontier = getFrontierStage(rollout);
  if (!frontier) {
    return { kind: "deployed" };
  }
  if (stageHasRunnableTasks(frontier)) {
    return { kind: "run-stage", stage: frontier };
  }
  if (stageHasRunningTasks(frontier)) {
    return { kind: "running-stage", stage: frontier };
  }
  // Non-complete frontier with nothing runnable or running: defer to Deploy.
  return { kind: "none" };
}

export function resolvePlanLifecycleHeaderState(
  input: PlanLifecycleResolverInput
): PlanLifecycleHeaderState {
  // Brand-new plan being created.
  if (input.isCreating) {
    return { kind: "create" };
  }

  // Terminal: closed plan. (Close-after-issue/rollout is future work, BYT-9204.)
  if (input.planState === State.DELETED) {
    return { kind: "closed" };
  }

  // Draft is the lifecycle boundary. Malformed stale rollout/approval data must
  // never surface governance or deploy controls before the draft is submitted.
  if (input.hasIssue && input.issueDraft) {
    return { kind: "ready-for-review" };
  }

  // A canceled review is terminal even after a rollout was created — surface the
  // closed stamp, not deploy actions (matches the issue detail page). This must
  // precede the rollout branch, which otherwise wins on a stale/orphaned rollout.
  if (input.issueStatus === IssueStatus.CANCELED) {
    return { kind: "closed" };
  }

  // Once a rollout exists, the review gates are settled and the lifecycle is in
  // deploy — the header summarizes the frontier stage.
  if (input.hasRollout) {
    return resolveDeploy(input);
  }

  // GitOps/release-backed plans bypass review; with no rollout yet there is no
  // safe header advance to offer.
  if (input.isGitOpsPlan) {
    return { kind: "none" };
  }

  // Persisted plans must have a linked Draft Review Issue. A missing or
  // unloadable issue is the durable partial-success state left when the second
  // create call fails; expose it instead of offering another create/retry.
  if (!input.hasIssue) {
    return { kind: "incomplete" };
  }

  // Approval flow still being generated — no safe action until it resolves.
  if (input.approvalStatus === ApprovalStatus.CHECKING) {
    return { kind: "review-generating" };
  }

  // Rejected review.
  if (input.approvalStatus === ApprovalStatus.REJECTED) {
    return { kind: "plan-status", reason: "rejected" };
  }

  // Approved / skipped: pre-deploy gate evaluation, then rollout creation.
  if (
    input.approvalStatus === ApprovalStatus.APPROVED ||
    input.approvalStatus === ApprovalStatus.SKIPPED
  ) {
    if (input.checks.error > 0) {
      return { kind: "plan-status", reason: "checks-failing" };
    }
    if (input.checks.running > 0) {
      return { kind: "plan-status", reason: "checking" };
    }
    // All required gates passed; the backend auto-creates the rollout. Show a
    // transient progress state until it arrives, then resolveDeploy takes over.
    return { kind: "preparing-rollout" };
  }

  // Pending review: the advance is the current reviewer's to take.
  if (input.hasCurrentStep && input.isCurrentUserCandidate && !input.readonly) {
    return { kind: "review-your-turn" };
  }
  // Review is the active gate — surface it plainly regardless of check state.
  return { kind: "plan-status", reason: "in-review" };
}

// Whether the right-hand lifecycle slot renders a primary action/status control.
// The terminal states (closed / deployed) render as a stamp left of the title,
// and "none" is empty — in those cases the header surfaces a secondary action
// directly instead of tucking it into the overflow menu. Kept next to the
// resolver so this stays in sync with the states it produces.
export function slotHasPrimaryControl(
  state: PlanLifecycleHeaderState
): boolean {
  return (
    state.kind !== "none" &&
    state.kind !== "closed" &&
    state.kind !== "deployed"
  );
}
