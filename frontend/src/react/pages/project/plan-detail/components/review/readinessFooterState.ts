// Footer state machine + bypass gating (spec: "Rollout readiness footer").
// The backend only enforces bb.rollouts.create on CreateRollout; the project
// "require issue approval" / "require no failed checks" settings are
// client-side gates, so the action hides whenever clicking it would violate
// one of them in the current state. The status line renders for everyone.
import { ApprovalStatus } from "@/types/proto-es/v1/common_pb";
import { IssueStatus } from "@/types/proto-es/v1/issue_service_pb";
import type { PlanCheckSummary } from "../../utils/phaseSummary";

export type ReadinessFooterStateKind =
  | "hidden"
  | "waiting-review"
  | "all-gates-passed"
  | "approved-checks-failed"
  | "rejected";

export interface ReadinessFooterState {
  kind: ReadinessFooterStateKind;
  checks: PlanCheckSummary;
}

export function computeReadinessFooterState(input: {
  hasRollout: boolean;
  issueStatus: IssueStatus | undefined;
  approvalStatus: ApprovalStatus;
  checks: PlanCheckSummary;
}): ReadinessFooterState {
  const { checks } = input;
  if (input.hasRollout || input.issueStatus !== IssueStatus.OPEN) {
    return { kind: "hidden", checks };
  }
  if (input.approvalStatus === ApprovalStatus.REJECTED) {
    return { kind: "rejected", checks };
  }
  if (
    input.approvalStatus === ApprovalStatus.APPROVED ||
    input.approvalStatus === ApprovalStatus.SKIPPED
  ) {
    return {
      kind: checks.error > 0 ? "approved-checks-failed" : "all-gates-passed",
      checks,
    };
  }
  return { kind: "waiting-review", checks };
}

export type BypassActionWeight = "none" | "link" | "button";

export function computeBypassActionWeight(input: {
  state: ReadinessFooterStateKind;
  canCreateRollout: boolean;
  requireIssueApproval: boolean;
}): BypassActionWeight {
  if (input.state === "hidden") {
    return "none";
  }
  if (!input.canCreateRollout) {
    return "none";
  }
  // A rejected review is an explicit, deliberate override: show the action
  // whenever the user can create a rollout. Kept as a muted link so it stays
  // quiet next to the "blocked" message. The confirm sheet still blocks deploy
  // when a mandatory project gate is unmet.
  if (input.state === "rejected") {
    return "link";
  }
  // Review still in progress: only offer the bypass when approval is optional —
  // a mandatory-approval project must not be bypassed before a decision.
  if (input.state === "waiting-review") {
    return input.requireIssueApproval ? "none" : "link";
  }
  // Review passed (approved/skipped): always offer a manual deploy when the user
  // can create a rollout, regardless of plan-check state. Prominent button when
  // checks failed (the call to action), muted link otherwise. The confirm sheet
  // still blocks deploy if the project mandates no failed checks.
  return input.state === "approved-checks-failed" ? "button" : "link";
}
