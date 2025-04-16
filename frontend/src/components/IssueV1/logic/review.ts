import { computed, unref, watchEffect } from "vue";
import {
  candidatesOfApprovalStepV1,
  batchGetOrFetchUsers,
  useCurrentUserV1,
  userNamePrefix,
} from "@/store";
import type {
  ReviewFlow,
  MaybeRef,
  ComposedIssue,
  WrappedReviewStep,
} from "@/types";
import { emptyFlow } from "@/types";
import type { ApprovalNode, Issue } from "@/types/proto/v1/issue_service";
import {
  ApprovalNode_Type,
  Issue_Approver_Status,
} from "@/types/proto/v1/issue_service";
import { displayRoleTitle } from "@/utils";
import { isUserIncludedInList } from "@/utils";
import type { ReviewContext } from "./context";

export const extractReviewContext = (issue: MaybeRef<Issue>): ReviewContext => {
  const ready = computed(() => {
    return unref(issue).approvalFindingDone ?? false;
  });
  const flow = computed((): ReviewFlow => {
    if (!ready.value) return emptyFlow();
    const { approvalTemplates, approvers } = unref(issue);
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
    if (unref(issue).approvalFindingError) {
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
    return unref(issue).approvalFindingError;
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
  const currentUser = useCurrentUserV1();
  // Format: users/{email}
  const currentUserName = computed(
    () => `${userNamePrefix}${currentUser.value.email}`
  );

  watchEffect(async () => {
    const { flow } = context;
    const approvers = flow.value.approvers;
    const steps = flow.value.template.flow?.steps;
    await batchGetOrFetchUsers(
      steps?.map((_, i) => approvers[i]?.principal) ?? []
    );
  });

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
      return approvers[index]?.principal;
    };

    const candidatesOfStep = (index: number) => {
      const step = steps?.[index];
      if (!step) return [];
      const users = candidatesOfApprovalStepV1(unref(issue), step);
      if (isUserIncludedInList(currentUserName.value, users)) {
        const idx = users.indexOf(currentUserName.value);
        if (idx >= 0) {
          users.splice(idx, 1);
        }
        users.unshift(currentUserName.value);
      }
      return users;
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

export const displayReviewRoleTitle = (node: ApprovalNode) => {
  const { type, role } = node;
  if (type !== ApprovalNode_Type.ANY_IN_GROUP) {
    return "";
  }

  if (role) {
    return displayRoleTitle(role);
  }
  return "";
};
