import { computed, watch, type ComputedRef, unref } from "vue";
import type Emittery from "emittery";

import type { Issue, MaybeRef, ReviewFlow, WrappedReviewStep } from "@/types";
import {
  ApprovalTemplate,
  Review,
  Review_Approver_Status,
} from "@/types/proto/v1/review_service";
import {
  candidatesOfApprovalStep,
  useAuthStore,
  useReviewStore,
  useUserStore,
} from "@/store";
import { IssueReviewContext, provideIssueReviewContext } from "./context";
import { useProgressivePoll } from "@/composables/useProgressivePoll";
import { extractUserResourceName } from "@/utils";

export type ReviewEvents = {
  "issue-status-changed": boolean;
};

export const extractIssueReviewContext = (
  issue: ComputedRef<Issue | undefined>,
  review: ComputedRef<Review>
): IssueReviewContext => {
  const ready = computed(() => {
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
  const status = computed(() => {
    if (!ready.value) {
      return Review_Approver_Status.PENDING;
    }
    if (review.value.approvalFindingError) {
      return Review_Approver_Status.PENDING;
    }

    const { template, approvers } = flow.value;
    const steps = template.flow?.steps ?? [];

    if (steps.length === 0) {
      // No review flow steps. That means need not manual review.
      return Review_Approver_Status.APPROVED;
    }

    if (
      approvers.some((app) => app.status === Review_Approver_Status.REJECTED)
    ) {
      // If any of the approvers down voted, the overall status should be 'REJECTED'
      return Review_Approver_Status.REJECTED;
    }

    // For an N-steps approval flow, we need exactly N upvote approvals to
    // pass the entire flow.
    const upVotes = approvers.filter(
      (app) => app.status === Review_Approver_Status.APPROVED
    );
    if (upVotes.length === steps.length) {
      return Review_Approver_Status.APPROVED;
    }
    return Review_Approver_Status.PENDING;
  });
  const done = computed(() => {
    return status.value === Review_Approver_Status.APPROVED;
  });
  const error = computed(() => {
    return review.value.approvalFindingError;
  });

  return {
    review,
    ready,
    flow,
    status,
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
    const approvers = flow.value.approvers;
    const currentStepIndex = flow.value.currentStepIndex ?? -1;

    const statusOfStep = (index: number) => {
      if (done.value) {
        return "APPROVED";
      }
      if (index >= (steps?.length ?? 0)) {
        // Out of index
        return "PENDING";
      }
      const approval = approvers[index];
      if (approval && approval.status === Review_Approver_Status.REJECTED) {
        return "REJECTED";
      }
      if (index < currentStepIndex) {
        return "APPROVED";
      }
      if (index === currentStepIndex) {
        return "CURRENT";
      }
      return "PENDING";
    };
    const approverOfStep = (index: number) => {
      const principal = approvers[index]?.principal;
      if (!principal) return undefined;
      const email = extractUserResourceName(principal);
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
