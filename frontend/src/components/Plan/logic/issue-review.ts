import type { ComputedRef, InjectionKey } from "vue";
import { computed, inject, provide, unref } from "vue";
import {
  Issue_ApprovalStatus,
  type Issue,
} from "@/types/proto-es/v1/issue_service_pb";

export const KEY = Symbol(
  "bb.plan.issue.review.context"
) as InjectionKey<IssueReviewContext>;

export const useIssueReviewContext = () => {
  return inject(KEY)!;
};

export const provideIssueReviewContext = (
  issue: ComputedRef<Issue | undefined>
) => {
  const status = computed(() => {
    const tempIssue = issue.value;
    if (!tempIssue) {
      return Issue_ApprovalStatus.PENDING;
    }
    return tempIssue.approvalStatus;
  });

  const done = computed(() => {
    return (
      status.value === Issue_ApprovalStatus.APPROVED ||
      status.value === Issue_ApprovalStatus.SKIPPED
    );
  });

  const error = computed(() => {
    return unref(issue)?.approvalStatusError;
  });

  const context = {
    done,
    error,
  };

  provide(KEY, context);

  return context;
};

type IssueReviewContext = ReturnType<typeof provideIssueReviewContext>;
