<template>
  <CreateButton v-if="isCreating" />
  <template v-else>
    <!-- Show unified actions for all plan states -->
    <UnifiedActionGroup
      :primary-action="primaryAction"
      :secondary-actions="secondaryActions"
      @perform-action="handlePerformAction"
    />

    <!-- Panels -->
    <IssueCreationActionPanel
      :show="pendingIssueCreationAction !== undefined"
      @close="pendingIssueCreationAction = undefined"
    />

    <template v-if="issue">
      <IssueReviewActionPanel
        :action="pendingReviewAction"
        @close="pendingReviewAction = undefined"
      />
      <IssueStatusActionPanel
        :action="pendingStatusAction"
        @close="pendingStatusAction = undefined"
      />
    </template>
  </template>
</template>

<script setup lang="ts">
import { head } from "lodash-es";
import { computed, ref } from "vue";
import { usePlanContext } from "@/components/Plan/logic";
import { provideIssueReviewContext } from "@/components/Plan/logic/issue-review";
import {
  useCurrentUserV1,
  candidatesOfApprovalStepV1,
  extractUserId,
  useCurrentProjectV1,
} from "@/store";
import {
  IssueStatus,
  Issue_Approver_Status,
} from "@/types/proto-es/v1/issue_service_pb";
import { isUserIncludedInList, hasProjectPermissionV2 } from "@/utils";
import { CreateButton } from "./create";
import {
  IssueReviewActionPanel,
  IssueStatusActionPanel,
  IssueCreationActionPanel,
} from "./panels";
import {
  UnifiedActionGroup,
  type ActionConfig,
  type IssueReviewAction,
  type IssueStatusAction,
  type UnifiedAction,
} from "./unified";

const { isCreating, issue, rollout } = usePlanContext();
const currentUser = useCurrentUserV1();
const { project } = useCurrentProjectV1();

// Provide issue review context when an issue exists
const reviewContext = provideIssueReviewContext(issue);

// Panel visibility state
const pendingReviewAction = ref<IssueReviewAction | undefined>(undefined);
const pendingStatusAction = ref<IssueStatusAction | undefined>(undefined);
const pendingIssueCreationAction = ref<"CREATE" | undefined>(undefined);

// Compute available actions based on issue state and user permissions
const availableActions = computed(() => {
  const actions: UnifiedAction[] = [];

  if (isCreating.value) return actions;

  // If no issue exists, show create issue action
  if (!issue?.value) {
    // If rollout exists, no actions are available.
    if (rollout?.value) {
      return actions;
    }
    const canCreateIssue = hasProjectPermissionV2(
      project.value,
      "bb.plans.create"
    );
    if (canCreateIssue) {
      actions.push("CREATE_ISSUE");
    }
    return actions;
  }

  const issueValue = issue.value;
  const isCanceled = issueValue.status === IssueStatus.CANCELED;
  const isDone = issueValue.status === IssueStatus.DONE;

  // If issue is canceled, check for re-open action
  if (isCanceled) {
    const currentUserEmail = currentUser.value.email;
    const issueCreator = extractUserId(issueValue.creator);
    const canReopen =
      currentUserEmail === issueCreator ||
      hasProjectPermissionV2(project.value, "bb.issues.update");

    if (canReopen) {
      actions.push("REOPEN");
    }
    return actions;
  }

  // If issue is done, no actions available
  if (isDone) return actions;

  // Check for review actions
  if (issueValue.approvalFindingDone && !reviewContext.done.value) {
    const currentUserEmail = currentUser.value.email;
    const issueCreator = extractUserId(issueValue.creator);
    const { approvers, approvalTemplates } = issueValue;

    // Check if issue has been rejected
    const hasRejection = approvers.some(
      (app) => app.status === Issue_Approver_Status.REJECTED
    );

    // RE_REQUEST is only available to the issue creator when rejected
    if (hasRejection && currentUserEmail === issueCreator) {
      actions.push("RE_REQUEST");
    } else {
      // Check if user can approve/reject
      const steps = head(approvalTemplates)?.flow?.steps ?? [];
      if (steps.length > 0) {
        const rejectedIndex = approvers.findIndex(
          (ap) => ap.status === Issue_Approver_Status.REJECTED
        );
        const currentStepIndex =
          rejectedIndex >= 0 ? rejectedIndex : approvers.length;
        const currentStep = steps[currentStepIndex];

        if (currentStep) {
          const candidates = candidatesOfApprovalStepV1(
            issueValue,
            currentStep
          );
          if (isUserIncludedInList(currentUserEmail, candidates)) {
            if (hasRejection) {
              actions.push("APPROVE");
            } else {
              actions.push("APPROVE", "REJECT");
            }
          }
        }
      }
    }
  }

  // Check for close action
  const currentUserEmail = currentUser.value.email;
  const issueCreator = extractUserId(issueValue.creator);
  const canClose =
    currentUserEmail === issueCreator ||
    hasProjectPermissionV2(project.value, "bb.issues.update");

  if (canClose) {
    actions.push("CLOSE");
  }

  return actions;
});

const primaryAction = computed((): ActionConfig | undefined => {
  const actions = availableActions.value;

  // CREATE_ISSUE is the highest priority when no issue exists
  if (actions.includes("CREATE_ISSUE")) {
    return { action: "CREATE_ISSUE" };
  }

  // REOPEN is primary when issue is closed
  if (actions.includes("REOPEN")) {
    return { action: "REOPEN" };
  }

  // APPROVE and RE_REQUEST are primary actions
  if (actions.includes("APPROVE")) {
    return { action: "APPROVE" };
  }
  if (actions.includes("RE_REQUEST")) {
    return { action: "RE_REQUEST" };
  }

  return undefined;
});

const secondaryActions = computed((): ActionConfig[] => {
  const actions = availableActions.value;
  const secondary: ActionConfig[] = [];

  // REJECT and CLOSE go in the dropdown
  if (actions.includes("REJECT")) {
    secondary.push({ action: "REJECT" });
  }
  if (actions.includes("CLOSE")) {
    secondary.push({ action: "CLOSE" });
  }

  return secondary;
});

const handlePerformAction = (action: UnifiedAction) => {
  switch (action) {
    case "APPROVE":
    case "REJECT":
    case "RE_REQUEST":
      pendingReviewAction.value = action as IssueReviewAction;
      break;
    case "CLOSE":
    case "REOPEN":
      pendingStatusAction.value = action as IssueStatusAction;
      break;
    case "CREATE_ISSUE":
      pendingIssueCreationAction.value = "CREATE";
      break;
  }
};
</script>
