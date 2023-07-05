import { computed, watch, type ComputedRef, unref } from "vue";
import type Emittery from "emittery";

import type {
  Issue as LegacyIssue,
  MaybeRef,
  ReviewFlow,
  WrappedReviewStep,
} from "@/types";
import {
  ApprovalTemplate,
  Issue,
  Issue_Approver_Status,
} from "@/types/proto/v1/issue_service";
import {
  candidatesOfApprovalStep,
  useAuthStore,
  useIssueV1Store,
  useUserStore,
} from "@/store";
import { IssueReviewContext, provideIssueReviewContext } from "./context";
import { useProgressivePoll } from "@/composables/useProgressivePoll";
import { extractUserResourceName } from "@/utils";

export type ReviewEvents = {
  "issue-status-changed": boolean;
};

export const extractIssueReviewContext = (
  legacyIssue: ComputedRef<LegacyIssue | undefined>,
  issue: ComputedRef<Issue>
): IssueReviewContext => {
  const ready = computed(() => {
    return issue.value.approvalFindingDone ?? false;
  });
  const flow = computed((): ReviewFlow => {
    if (!ready.value) return emptyFlow();
    const { approvalTemplates, approvers } = issue.value;
    if (approvalTemplates.length === 0) return emptyFlow();

    const rejectedIndex = approvers.findIndex(
      (ap) => ap.status === Issue_Approver_Status.REJECTED
    );
    const currentStepIndex =
      rejectedIndex >= 0 ? rejectedIndex : approvers.length;

    return {
      template: approvalTemplates[0],
      approvers,
      currentStepIndex,
    };
  });
  const status = computed(() => {
    if (!ready.value) {
      return Issue_Approver_Status.PENDING;
    }
    if (issue.value.approvalFindingError) {
      return Issue_Approver_Status.PENDING;
    }

    const { template, approvers } = flow.value;
    const steps = template.flow?.steps ?? [];

    if (steps.length === 0) {
      // No review flow steps. That means need not manual review.
      return Issue_Approver_Status.APPROVED;
    }

    if (
      approvers.some((app) => app.status === Issue_Approver_Status.REJECTED)
    ) {
      // If any of the approvers down voted, the overall status should be 'REJECTED'
      return Issue_Approver_Status.REJECTED;
    }

    // For an N-steps approval flow, we need exactly N upvote approvals to
    // pass the entire flow.
    const upVotes = approvers.filter(
      (app) => app.status === Issue_Approver_Status.APPROVED
    );
    if (upVotes.length === steps.length) {
      return Issue_Approver_Status.APPROVED;
    }
    return Issue_Approver_Status.PENDING;
  });
  const done = computed(() => {
    return status.value === Issue_Approver_Status.APPROVED;
  });
  const error = computed(() => {
    return issue.value.approvalFindingError;
  });

  return {
    issue: issue,
    ready,
    flow,
    status,
    done,
    error,
  };
};

export const provideIssueReview = (
  legacyIssue: ComputedRef<LegacyIssue | undefined>,
  events: Emittery<ReviewEvents>
) => {
  const store = useIssueV1Store();
  const issue = computed(() => {
    return legacyIssue.value
      ? store.getIssueByIssue(legacyIssue.value)
      : Issue.fromJSON({});
  });

  const update = () => {
    if (legacyIssue.value) {
      store.fetchReviewByIssue(legacyIssue.value);
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
    () => legacyIssue.value?.id,
    () => {
      if (legacyIssue.value) poller.start();
      else poller.stop();
    },
    {
      immediate: true,
    }
  );
  events.on("issue-status-changed", () => {
    update();
  });

  const context = extractIssueReviewContext(legacyIssue, issue);

  provideIssueReviewContext(context);
};

export const useWrappedReviewSteps = (
  issue: MaybeRef<LegacyIssue>,
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
      if (approval && approval.status === Issue_Approver_Status.REJECTED) {
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
