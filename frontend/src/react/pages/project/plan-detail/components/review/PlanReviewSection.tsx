import { create } from "@bufbuild/protobuf";
import { useEffect } from "react";
import { useTranslation } from "react-i18next";
import { useAppStore } from "@/react/stores/app";
import { projectNamePrefix } from "@/store/modules/v1/common";
import { ApprovalStatus } from "@/types/proto-es/v1/common_pb";
import type { IssueComment } from "@/types/proto-es/v1/issue_service_pb";
import { ListIssueCommentsRequestSchema } from "@/types/proto-es/v1/issue_service_pb";
import { usePlanDetailContext } from "../../shell/PlanDetailContext";
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
  const issueUpdateKey = `${issue?.updateTime?.seconds ?? ""}:${issue?.updateTime?.nanos ?? ""}`;
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

  // Refetch comments whenever the issue changes server-side (polling bumps
  // updateTime) or after local actions refresh the issue.
  useEffect(() => {
    if (!issueName) return;
    void useAppStore
      .getState()
      .listIssueComments(
        create(ListIssueCommentsRequestSchema, {
          parent: issueName,
          pageSize: 1000,
        })
      )
      .catch(() => undefined);
  }, [issueName, issueUpdateKey]);

  const comments = useAppStore((state) =>
    issueName ? state.getIssueComments(issueName) : EMPTY_COMMENTS
  );

  if (!issue) return null;

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
      <ReviewActivityTimeline
        comments={comments}
        issue={issue}
        plan={page.plan}
      />
      <ReviewReadinessFooter issue={issue} plan={page.plan} />
    </div>
  );
}
