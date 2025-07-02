import { head } from "lodash-es";
import type { InjectionKey, MaybeRef } from "vue";
import { computed, inject, provide, unref } from "vue";
import {
  Issue_Approver_Status,
  type Issue,
} from "@/types/proto-es/v1/issue_service_pb";

export const KEY = Symbol(
  "bb.plan.issue.review.context"
) as InjectionKey<IssueReviewContext>;

export const useIssueReviewContext = () => {
  return inject(KEY)!;
};

export const provideIssueReviewContext = (
  issue: MaybeRef<Issue | undefined>
) => {
  const status = computed(() => {
    const tempIssue = unref(issue);
    if (!tempIssue) {
      return Issue_Approver_Status.PENDING;
    }
    if (!tempIssue?.approvalFindingDone) {
      return Issue_Approver_Status.PENDING;
    }
    if (tempIssue?.approvalFindingError) {
      return Issue_Approver_Status.PENDING;
    }

    const { approvalTemplates, approvers } = tempIssue;
    const steps = head(approvalTemplates)?.flow?.steps ?? [];
    // No review flow steps. That means need not manual review.
    if (steps.length === 0) {
      return Issue_Approver_Status.APPROVED;
    }
    // If any of the approvers rejected, the overall status should be 'REJECTED'
    if (
      approvers.some((app) => app.status === Issue_Approver_Status.REJECTED)
    ) {
      return Issue_Approver_Status.REJECTED;
    }

    if (
      approvers.length === steps.length &&
      approvers.every((app) => app.status === Issue_Approver_Status.APPROVED)
    ) {
      return Issue_Approver_Status.APPROVED;
    }
    return Issue_Approver_Status.PENDING;
  });

  const done = computed(() => {
    return status.value === Issue_Approver_Status.APPROVED;
  });

  const error = computed(() => {
    return unref(issue)?.approvalFindingError;
  });

  const context = {
    done,
    error,
  };

  provide(KEY, context);

  return context;
};

type IssueReviewContext = ReturnType<typeof provideIssueReviewContext>;
