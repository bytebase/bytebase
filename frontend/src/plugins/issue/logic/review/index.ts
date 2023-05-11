import { computed, watch, type ComputedRef, unref } from "vue";
import type Emittery from "emittery";

import type { Issue, MaybeRef, ReviewFlow, WrappedReviewStep } from "@/types";
import { Review } from "@/types/proto/v1/review_service";
import {
  candidatesOfApprovalStep,
  extractUserEmail,
  useAuthStore,
  useReviewStore,
  useUserStore,
} from "@/store";
import { IssueReviewContext, provideIssueReviewContext } from "./context";
import { ApprovalTemplate } from "@/types/proto/store/approval";
import { useProgressivePoll } from "@/composables/useProgressivePoll";
import { isGrantRequestIssueType } from "@/utils";

export type ReviewEvents = {
  "issue-status-changed": boolean;
};

export const extractIssueReviewContext = (
  issue: ComputedRef<Issue | undefined>,
  review: ComputedRef<Review>
): IssueReviewContext => {
  const ready = computed(() => {
    if (issue.value && isGrantRequestIssueType(issue.value?.type)) {
      return true;
    }
    return review.value.approvalFindingDone ?? false;
  });
  const flow = computed((): ReviewFlow => {
    if (!ready.value) return emptyFlow();
    const { approvalTemplates, approvers } = review.value;
    if (approvalTemplates.length === 0) return emptyFlow();
    return {
      template: approvalTemplates[0],
      approvers,
      currentStepIndex: approvers.length,
    };
  });
  const done = computed(() => {
    if (!ready.value) return false;
    if (review.value.approvalFindingError) return false;
    if (issue.value && isGrantRequestIssueType(issue.value?.type)) {
      return issue.value.status === "DONE";
    }

    const { template, approvers } = flow.value;
    const steps = template.flow?.steps ?? [];
    if (steps.length === 0) return true;
    return approvers.length === steps.length;
  });
  const error = computed(() => {
    return review.value.approvalFindingError;
  });

  return {
    review,
    ready,
    flow,
    done,
    error,
  };
};

export const provideIssueReview = (
  issue: ComputedRef<Issue | undefined>,
  events: Emittery<ReviewEvents>
) => {
  const store = useReviewStore();
  const review = computed(() => {
    return issue.value
      ? store.getReviewByIssue(issue.value)
      : Review.fromJSON({});
  });

  const update = () => {
    if (issue.value) {
      store.fetchReviewByIssue(issue.value);
    }
  };

  const poller = useProgressivePoll(update, {
    interval: {
      min: 500,
      max: 10000,
      growth: 2,
      jitter: 500,
    },
  });

  watch(
    () => issue.value?.id,
    () => {
      if (issue.value) poller.start();
      else poller.stop();
    },
    {
      immediate: true,
    }
  );
  events.on("issue-status-changed", () => {
    update();
  });

  const context = extractIssueReviewContext(issue, review);

  provideIssueReviewContext(context);
};

export const useWrappedReviewSteps = (
  issue: MaybeRef<Issue>,
  context: IssueReviewContext
) => {
  const userStore = useUserStore();
  const currentUserName = computed(() => useAuthStore().currentUser.name);
  return computed(() => {
    const { flow, done } = context;
    const steps = flow.value.template.flow?.steps;
    const currentStepIndex = flow.value.currentStepIndex ?? -1;

    const statusOfStep = (index: number) => {
      if (done.value) return "DONE";
      if (index < currentStepIndex) return "DONE";
      if (index === currentStepIndex) return "CURRENT";
      return "PENDING";
    };
    const approverOfStep = (index: number) => {
      const principal = flow.value.approvers[index]?.principal;
      if (!principal) return undefined;
      const email = extractUserEmail(principal);
      return userStore.getUserByEmail(email);
    };
    const candidatesOfStep = (index: number) => {
      const step = steps?.[index];
      if (!step) return [];
      const users = candidatesOfApprovalStep(unref(issue), step);
      const idx = users.indexOf(currentUserName.value);
      if (idx > 0) {
        users.splice(idx, 1);
        users.unshift(currentUserName.value);
      }
      return users.map((user) => userStore.getUserByName(user)!);
    };

    return steps?.map<WrappedReviewStep>((step, index) => ({
      index,
      step,
      status: statusOfStep(index),
      approver: approverOfStep(index),
      candidates: candidatesOfStep(index),
    }));
  });
};

const emptyFlow = (): ReviewFlow => {
  return {
    template: ApprovalTemplate.fromJSON({}),
    approvers: [],
    currentStepIndex: -1,
  };
};
