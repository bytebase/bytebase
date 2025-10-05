import { computed, unref, watchEffect } from "vue";
import {
  candidatesOfApprovalStepV1,
  useCurrentUserV1,
  userNamePrefix,
  useUserStore,
} from "@/store";
import type {
  ReviewFlow,
  MaybeRef,
  ComposedIssue,
  WrappedReviewStep,
} from "@/types";
import { emptyFlow } from "@/types";
import type {
  ApprovalNode,
  ApprovalStep,
  Issue,
} from "@/types/proto-es/v1/issue_service_pb";
import {
  ApprovalNode_Type,
  Issue_Approver_Status,
  Issue_ApprovalStatus,
} from "@/types/proto-es/v1/issue_service_pb";
import { displayRoleTitle } from "@/utils";
import { isUserIncludedInList } from "@/utils";
import type { ReviewContext } from "./context";

export const extractReviewContext = (issue: MaybeRef<Issue>): ReviewContext => {
  const flow = computed((): ReviewFlow => {
    const issueValue = unref(issue);
    const approvalStatus = issueValue.approvalStatus;
    if (
      approvalStatus === Issue_ApprovalStatus.CHECKING ||
      approvalStatus === Issue_ApprovalStatus.APPROVAL_STATUS_UNSPECIFIED
    ) {
      return emptyFlow();
    }

    const { approvalTemplates, approvers } = issueValue;
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

  return {
    flow,
  };
};

export const useWrappedReviewStepsV1 = (
  issue: MaybeRef<ComposedIssue>,
  context: ReviewContext
) => {
  const userStore = useUserStore();
  const currentUser = useCurrentUserV1();
  // Format: users/{email}
  const currentUserName = computed(
    () => `${userNamePrefix}${currentUser.value.email}`
  );

  watchEffect(async () => {
    const { flow } = context;
    const approvers = flow.value.approvers;
    const steps = flow.value.template.flow?.steps;
    const distinctUsers = steps?.map((_, i) => approvers[i]?.principal) ?? [];
    await userStore.batchGetUsers(distinctUsers);
  });

  return computed(() => {
    const { flow } = context;
    const steps = flow.value.template.flow?.steps || [];
    const approvers = flow.value.approvers;
    const currentStepIndex = flow.value.currentStepIndex ?? -1;
    const issueValue = unref(issue);
    const rolloutReady =
      issueValue.approvalStatus === Issue_ApprovalStatus.APPROVED ||
      issueValue.approvalStatus === Issue_ApprovalStatus.SKIPPED;

    const statusOfStep = (index: number) => {
      if (rolloutReady) {
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

    const candidatesOfStep = (step: ApprovalStep) => {
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

    return steps.map<WrappedReviewStep>((step, index) => ({
      index,
      step,
      status: statusOfStep(index),
      approver: approverOfStep(index),
      candidates: candidatesOfStep(step),
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
