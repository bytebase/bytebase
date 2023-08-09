import { Ref, computed, unref } from "vue";

import {
  ReviewFlow,
  emptyFlow,
  MaybeRef,
  ComposedIssue,
  WrappedReviewStep,
} from "@/types";
import { Issue, Issue_Approver_Status } from "@/types/proto/v1/issue_service";
import { ReviewContext } from "./context";
import {
  candidatesOfApprovalStepV1,
  useAuthStore,
  useUserStore,
} from "@/store";
import { extractUserResourceName } from "@/utils";

export const extractReviewContext = (issue: Ref<Issue>): ReviewContext => {
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
    ready,
    flow,
    status,
    done,
    error,
  };
};

export const useWrappedReviewStepsV1 = (
  issue: MaybeRef<ComposedIssue>,
  context: ReviewContext
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
      const users = candidatesOfApprovalStepV1(unref(issue), step);
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
