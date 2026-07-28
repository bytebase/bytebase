// Gathers the async / IAM-derived inputs the pure resolver needs (whose turn it
// is to review, whether the user may run the frontier stage) and feeds them into
// resolvePlanLifecycleHeaderState. Mirrors how ReviewReadinessFooter computes its
// inputs before calling computeReadinessFooterState.
import { create } from "@bufbuild/protobuf";
import { useMemo } from "react";
import { ApprovalStatus } from "@/types/proto-es/v1/common_pb";
import type { Issue } from "@/types/proto-es/v1/issue_service_pb";
import { IssueSchema } from "@/types/proto-es/v1/issue_service_pb";
import type { PlanDetailPageState } from "../../shell/hooks/types";
import { getPlanCheckSummary } from "../../utils/phaseSummary";
import { isReleaseBackedPlan } from "../../utils/spec";
import { deriveSteps } from "../review/ReviewApprovalFlow";
import { useApprovalCandidates } from "../review/useApprovalCandidates";
import {
  type PlanLifecycleHeaderState,
  resolvePlanLifecycleHeaderState,
} from "./planLifecycleHeaderState";

// Stable placeholder so the candidate hook can run unconditionally (rules of
// hooks) when no issue exists yet; an empty role resolves to no candidates.
const EMPTY_ISSUE: Issue = create(IssueSchema, {});

export function usePlanLifecycleHeader(
  page: PlanDetailPageState
): PlanLifecycleHeaderState {
  const issue = page.issue;
  const currentRole = useMemo(
    () =>
      issue
        ? (deriveSteps(issue).find((step) => step.status === "current")?.role ??
          "")
        : "",
    [issue]
  );
  const { isCurrentUserCandidate } = useApprovalCandidates(
    issue ?? EMPTY_ISSUE,
    page.projectId,
    currentRole
  );

  const isGitOpsPlan = useMemo(
    () => isReleaseBackedPlan(page.plan.specs),
    [page.plan.specs]
  );
  const checks = useMemo(() => getPlanCheckSummary(page.plan), [page.plan]);

  // Depend on the primitives the resolver actually reads, not the whole
  // `issue` object — polling replaces `page.issue` with a fresh reference every
  // tick, which would re-run the resolver and re-render the header even when the
  // status/approval it consumes is unchanged.
  const hasIssue = !!page.issue;
  const issueStatus = page.issue?.status;
  const issueDraft = page.issue?.draft ?? false;
  const approvalStatus =
    page.issue?.approvalStatus ?? ApprovalStatus.APPROVAL_STATUS_UNSPECIFIED;

  return useMemo(
    () =>
      resolvePlanLifecycleHeaderState({
        isCreating: page.isCreating,
        isGitOpsPlan,
        readonly: page.readonly,
        planState: page.plan.state,
        hasIssue,
        issueStatus,
        issueDraft,
        approvalStatus,
        hasCurrentStep: currentRole !== "",
        isCurrentUserCandidate,
        checks,
        hasRollout: !!page.rollout,
        rollout: page.rollout,
      }),
    [
      approvalStatus,
      checks,
      currentRole,
      hasIssue,
      isCurrentUserCandidate,
      isGitOpsPlan,
      issueDraft,
      issueStatus,
      page.isCreating,
      page.plan.state,
      page.readonly,
      page.rollout,
    ]
  );
}
