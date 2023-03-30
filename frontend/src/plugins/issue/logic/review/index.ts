import { computed, watch, type Ref } from "vue";
import type Emittery from "emittery";

import type { Issue, ReviewFlow } from "@/types";
import { Review } from "@/types/proto/v1/review_service";
import { useReviewStore } from "@/store";
import { provideIssueReviewContext } from "./context";
import { ApprovalTemplate } from "@/types/proto/store/approval";
import { useProgressivePoll } from "@/composables/useProgressivePoll";

export type ReviewEvents = {
  "issue-status-changed": boolean;
};

export const provideIssueReview = (
  issue: Ref<Issue | undefined>,
  events: Emittery<ReviewEvents>
) => {
  const store = useReviewStore();
  const review = computed(() => {
    return issue.value
      ? store.getReviewByIssue(issue.value)
      : Review.fromJSON({});
  });
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
  const done = computed(() => {
    if (!ready.value) return false;
    if (review.value.approvalFindingError) return false;

    const { template, approvers } = flow.value;
    const steps = template.flow?.steps ?? [];
    if (steps.length === 0) return true;
    return approvers.length === steps.length;
  });
  const error = computed(() => {
    return review.value.approvalFindingError;
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

  provideIssueReviewContext({
    review,
    ready,
    flow,
    done,
    error,
  });
};

const emptyFlow = (): ReviewFlow => {
  return {
    template: ApprovalTemplate.fromJSON({}),
    approvers: [],
    currentStepIndex: -1,
  };
};
