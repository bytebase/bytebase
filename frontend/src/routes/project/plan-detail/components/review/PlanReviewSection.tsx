import { useEffect } from "react";
import { useTranslation } from "react-i18next";
import { useAppStore } from "@/stores/app";
import { projectNamePrefix } from "@/stores/modules/v1/common";
import { ApprovalStatus } from "@/types/proto-es/v1/common_pb";
import type { IssueComment } from "@/types/proto-es/v1/issue_service_pb";
import { usePlanReviewState } from "../../hooks/usePlanReviewState";
import { usePlanDetailContext } from "../../shell/PlanDetailContext";
import { AutomatedReviewSection } from "./AutomatedReviewSection";
import { ReviewActivityTimeline } from "./ReviewActivityTimeline";
import { ReviewApprovalFlow } from "./ReviewApprovalFlow";
import { ReviewReadinessFooter } from "./ReviewReadinessFooter";
import { ReviewRejectionBanner } from "./ReviewRejectionBanner";
import { PlanReviewSectionHeader } from "./ReviewSectionHeader";

const EMPTY_COMMENTS: IssueComment[] = [];

export function PlanReviewSection() {
  const { t } = useTranslation();
  const page = usePlanDetailContext();
  const issue = page.issue;
  const issueName = issue?.name ?? "";
  const loadProjectIamPolicy = useAppStore(
    (state) => state.loadProjectIamPolicy
  );

  // Candidate computation needs project + IAM in the cache.
  useEffect(() => {
    const projectName = `${projectNamePrefix}${page.projectId}`;
    void useAppStore
      .getState()
      .getOrFetchProjectByName(projectName)
      .catch(() => undefined);
    void loadProjectIamPolicy(projectName).catch(() => undefined);
  }, [loadProjectIamPolicy, page.projectId]);

  const comments = useAppStore((state) =>
    issueName ? state.getIssueComments(issueName) : EMPTY_COMMENTS
  );
  // Under SQL Review V2 the machine reviews run from draft onward, so Review
  // shows them, and Deploy lists what the rollout waits on.
  const reviewState = usePlanReviewState();

  if (!issue) return null;

  if (issue.draft) {
    return (
      <div className="flex flex-col">
        {reviewState && (
          <div className="flex flex-col gap-y-1 px-4 pt-3">
            <h3 className="text-base font-medium text-main">
              {t("issue.approval-flow.self")}
            </h3>
            <p className="text-sm text-control-light">
              {t("plan.review.approval-after-submit")}
            </p>
          </div>
        )}
        {reviewState && <AutomatedReviewSection state={reviewState} />}
        <ReviewActivityTimeline
          comments={comments}
          issue={issue}
          plan={page.plan}
        />
      </div>
    );
  }

  if (issue.approvalStatus === ApprovalStatus.CHECKING) {
    return (
      <div className="flex items-center gap-x-2 p-4 text-sm text-control-placeholder">
        <div className="size-4 animate-spin rounded-full border-2 border-control-border border-t-accent" />
        <span>
          {t("custom-approval.issue-review.generating-approval-flow")}
        </span>
      </div>
    );
  }

  return (
    <div className="flex flex-col">
      <PlanReviewSectionHeader issue={issue} />
      {/* ReviewApprovalFlow renders the "no approval required" note itself when
          the approval is skipped / has no roles, so no guard is needed here. */}
      <ReviewApprovalFlow issue={issue} />
      <ReviewRejectionBanner comments={comments} issue={issue} />
      {reviewState && <AutomatedReviewSection state={reviewState} />}
      <ReviewActivityTimeline
        comments={comments}
        issue={issue}
        plan={page.plan}
      />
      {!reviewState && <ReviewReadinessFooter issue={issue} plan={page.plan} />}
    </div>
  );
}
