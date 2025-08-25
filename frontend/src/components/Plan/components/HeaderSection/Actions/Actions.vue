<template>
  <CreateButton v-if="isCreating" />
  <template v-else>
    <!-- Show unified actions for all plan states -->
    <UnifiedActionGroup
      :primary-action="primaryAction"
      :secondary-actions="secondaryActions"
      :disabled="actionsDisabled"
      :disabled-tooltip="disabledTooltip"
      @perform-action="handlePerformAction"
    />

    <!-- Panels -->

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
import { create } from "@bufbuild/protobuf";
import { head } from "lodash-es";
import { useDialog } from "naive-ui";
import { computed, nextTick, ref, watchEffect } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import { useSpecsValidation } from "@/components/Plan/components/common";
import { usePlanContext, usePlanCheckStatus } from "@/components/Plan/logic";
import { useIssueReviewContext } from "@/components/Plan/logic/issue-review";
import { useResourcePoller } from "@/components/Plan/logic/poller";
import { useEditorState } from "@/components/Plan/logic/useEditorState";
import {
  issueServiceClientConnect,
  rolloutServiceClientConnect,
} from "@/grpcweb";
import { PROJECT_V1_ROUTE_ISSUE_DETAIL_V1 } from "@/router/dashboard/projectV1";
import {
  useCurrentUserV1,
  candidatesOfApprovalStepV1,
  extractUserId,
  useCurrentProjectV1,
  pushNotification,
  usePolicyV1Store,
} from "@/store";
import { usePlanStore } from "@/store/modules/v1/plan";
import { State } from "@/types/proto-es/v1/common_pb";
import {
  CreateIssueRequestSchema,
  IssueSchema,
  IssueStatus,
  Issue_Approver_Status,
  Issue_Type,
} from "@/types/proto-es/v1/issue_service_pb";
import { PolicyType } from "@/types/proto-es/v1/org_policy_service_pb";
import { PlanCheckRun_Result_Status } from "@/types/proto-es/v1/plan_service_pb";
import { CreateRolloutRequestSchema } from "@/types/proto-es/v1/rollout_service_pb";
import {
  isUserIncludedInList,
  hasProjectPermissionV2,
  extractProjectResourceName,
  extractIssueUID,
} from "@/utils";
import { CreateButton } from "./create";
import { IssueReviewActionPanel, IssueStatusActionPanel } from "./panels";
import {
  UnifiedActionGroup,
  type ActionConfig,
  type IssueReviewAction,
  type IssueStatusAction,
  type PlanAction,
  type UnifiedAction,
} from "./unified";

const { t } = useI18n();
const router = useRouter();
const dialog = useDialog();
const resourcePoller = useResourcePoller();
const { isCreating, plan, issue, events } = usePlanContext();
const currentUser = useCurrentUserV1();
const { project } = useCurrentProjectV1();
const planStore = usePlanStore();
const policyV1Store = usePolicyV1Store();
const editorState = useEditorState();

// Use the validation hook for all specs
const { isSpecEmpty } = useSpecsValidation(computed(() => plan.value.specs));

// Use plan check status for issue creation validation
const {
  getOverallStatus: planCheckSummaryStatus,
  hasRunning: hasRunningPlanChecks,
} = usePlanCheckStatus(plan);

// Provide issue review context when an issue exists
const reviewContext = useIssueReviewContext();

// Policy for restricting issue creation when plan checks fail
const restrictIssueCreationForSqlReviewPolicy = ref(false);

// Computed property for actions disabled state.
const actionsDisabled = computed(() => {
  return editorState.isEditing.value;
});

// Tooltip message for disabled state.
const disabledTooltip = computed(() => {
  if (editorState.isEditing.value) {
    return t("plan.editor.save-changes-before-continuing");
  }
  return "";
});

// Panel visibility state
const pendingReviewAction = ref<IssueReviewAction | undefined>(undefined);
const pendingStatusAction = ref<IssueStatusAction | undefined>(undefined);

// Watch for policy changes to determine if issue creation is restricted
watchEffect(async () => {
  const workspaceLevelPolicy =
    await policyV1Store.getOrFetchPolicyByParentAndType({
      parentPath: "",
      policyType: PolicyType.RESTRICT_ISSUE_CREATION_FOR_SQL_REVIEW,
    });
  if (
    workspaceLevelPolicy?.policy?.case ===
      "restrictIssueCreationForSqlReviewPolicy" &&
    workspaceLevelPolicy.policy.value.disallow
  ) {
    restrictIssueCreationForSqlReviewPolicy.value = true;
    return;
  }

  const projectLevelPolicy =
    await policyV1Store.getOrFetchPolicyByParentAndType({
      parentPath: project.value.name,
      policyType: PolicyType.RESTRICT_ISSUE_CREATION_FOR_SQL_REVIEW,
    });
  if (
    projectLevelPolicy?.policy?.case ===
      "restrictIssueCreationForSqlReviewPolicy" &&
    projectLevelPolicy.policy.value.disallow
  ) {
    restrictIssueCreationForSqlReviewPolicy.value = true;
    return;
  }

  // Fall back to default value.
  restrictIssueCreationForSqlReviewPolicy.value = false;
});

// Compute available actions based on issue state and user permissions
const availableActions = computed(() => {
  const actions: UnifiedAction[] = [];

  if (isCreating.value) return actions;

  const currentUserEmail = currentUser.value.email;
  // If no issue exists, show create issue action or close plan action.
  if (plan.value.issue === "") {
    // If rollout exists, no actions are available.
    if (plan.value.rollout !== "") {
      return actions;
    }

    // Check if user can close the plan
    const canClosePlan =
      plan.value.state === State.ACTIVE &&
      (currentUserEmail === extractUserId(plan.value.creator || "") ||
        hasProjectPermissionV2(project.value, "bb.plans.update"));

    if (canClosePlan) {
      actions.push("PLAN_CLOSE");
    }

    // Check if user can reopen the plan
    const canReopenPlan =
      plan.value.state === State.DELETED &&
      currentUserEmail === extractUserId(plan.value.creator || "") &&
      hasProjectPermissionV2(project.value, "bb.plans.update");

    if (canReopenPlan) {
      actions.push("PLAN_REOPEN");
      return actions; // For deleted plans, only show reopen action
    }

    const canCreateIssue =
      plan.value.state === State.ACTIVE &&
      currentUserEmail === extractUserId(plan.value.creator || "") &&
      hasProjectPermissionV2(project.value, "bb.issues.create");

    if (canCreateIssue) {
      actions.push("ISSUE_CREATE");
    }
    return actions;
  }

  const issueValue = issue?.value;
  // Should not reach here.
  if (!issueValue) return actions;
  const isCanceled = issueValue.status === IssueStatus.CANCELED;
  const isDone = issueValue.status === IssueStatus.DONE;

  // If issue is canceled, check for re-open action
  if (isCanceled) {
    const issueCreator = extractUserId(issueValue.creator);
    const canReopen =
      currentUserEmail === issueCreator &&
      hasProjectPermissionV2(project.value, "bb.issues.update");

    if (canReopen) {
      actions.push("ISSUE_STATUS_REOPEN");
    }
    return actions;
  }

  // If issue is done, no actions available
  if (isDone) return actions;

  // Check for review actions
  if (issueValue.approvalFindingDone && !reviewContext.done.value) {
    const issueCreator = extractUserId(issueValue.creator);
    const { approvers, approvalTemplates } = issueValue;

    // Check if issue has been rejected
    const hasRejection = approvers.some(
      (app) => app.status === Issue_Approver_Status.REJECTED
    );

    // RE_REQUEST is only available to the issue creator when rejected
    if (hasRejection && currentUserEmail === issueCreator) {
      actions.push("ISSUE_REVIEW_RE_REQUEST");
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
              actions.push("ISSUE_REVIEW_APPROVE");
            } else {
              actions.push("ISSUE_REVIEW_APPROVE", "ISSUE_REVIEW_REJECT");
            }
          }
        }
      }
    }
  }

  // Check for close action
  const issueCreator = extractUserId(issueValue.creator);
  const canClose =
    currentUserEmail === issueCreator &&
    hasProjectPermissionV2(project.value, "bb.issues.update");

  if (canClose) {
    actions.push("ISSUE_STATUS_CLOSE");
  }

  return actions;
});

const primaryAction = computed((): ActionConfig | undefined => {
  const actions = availableActions.value;

  // ISSUE_CREATE is the highest priority when no issue exists
  if (actions.includes("ISSUE_CREATE")) {
    // Check if all specs have valid statements (not empty)
    const hasValidSpecs = !plan.value.specs.some((spec) => isSpecEmpty(spec));

    // Check permissions
    const hasPermission = hasProjectPermissionV2(
      project.value,
      "bb.issues.create"
    );

    // Check plan check status and policy restrictions
    const planChecksFailed =
      planCheckSummaryStatus.value === PlanCheckRun_Result_Status.ERROR;
    const isRestrictedByPolicy =
      planChecksFailed && restrictIssueCreationForSqlReviewPolicy.value;
    const hasRunningChecks = hasRunningPlanChecks.value;

    const errors: string[] = [];
    if (!hasValidSpecs) {
      errors.push("Missing statement");
    }
    if (!hasPermission) {
      errors.push(t("common.missing-required-permission"));
    }
    if (hasRunningChecks) {
      errors.push("Plan checks are running");
    }
    if (isRestrictedByPolicy) {
      errors.push(
        t(
          "custom-approval.issue-review.disallow-approve-reason.some-task-checks-didnt-pass"
        )
      );
    }

    const isDisabled = errors.length > 0;
    const description = isDisabled ? errors.join(", ") : undefined;

    return {
      action: "ISSUE_CREATE",
      disabled: isDisabled,
      description,
    };
  }

  // PLAN_REOPEN is primary when plan is deleted
  if (actions.includes("PLAN_REOPEN")) {
    return { action: "PLAN_REOPEN" };
  }

  // ISSUE_STATUS_REOPEN is primary when issue is closed
  if (actions.includes("ISSUE_STATUS_REOPEN")) {
    return { action: "ISSUE_STATUS_REOPEN" };
  }

  // ISSUE_REVIEW_APPROVE and ISSUE_REVIEW_RE_REQUEST are primary actions
  if (actions.includes("ISSUE_REVIEW_APPROVE")) {
    return { action: "ISSUE_REVIEW_APPROVE" };
  }
  if (actions.includes("ISSUE_REVIEW_RE_REQUEST")) {
    return { action: "ISSUE_REVIEW_RE_REQUEST" };
  }

  return undefined;
});

const secondaryActions = computed((): ActionConfig[] => {
  const actions = availableActions.value;
  const secondary: ActionConfig[] = [];

  // ISSUE_REVIEW_REJECT, ISSUE_STATUS_CLOSE, and PLAN_CLOSE go in the dropdown
  if (actions.includes("ISSUE_REVIEW_REJECT")) {
    secondary.push({ action: "ISSUE_REVIEW_REJECT" });
  }
  if (actions.includes("ISSUE_STATUS_CLOSE")) {
    secondary.push({ action: "ISSUE_STATUS_CLOSE" });
  }
  if (actions.includes("PLAN_CLOSE")) {
    secondary.push({ action: "PLAN_CLOSE" });
  }

  return secondary;
});

const handlePerformAction = async (action: UnifiedAction) => {
  switch (action) {
    case "ISSUE_REVIEW_APPROVE":
    case "ISSUE_REVIEW_REJECT":
    case "ISSUE_REVIEW_RE_REQUEST":
      pendingReviewAction.value = action as IssueReviewAction;
      break;
    case "ISSUE_STATUS_CLOSE":
    case "ISSUE_STATUS_REOPEN":
      pendingStatusAction.value = action as IssueStatusAction;
      break;
    case "ISSUE_CREATE":
      await handleIssueCreate();
      break;
    case "PLAN_CLOSE":
      await handlePlanStateChange("PLAN_CLOSE");
      break;
    case "PLAN_REOPEN":
      await handlePlanStateChange("PLAN_REOPEN");
      break;
  }
};

const handleIssueCreate = async () => {
  if (!plan?.value) return;

  try {
    await doCreateIssue();
  } catch (error) {
    console.error("Failed to create issue:", error);
    pushNotification({
      module: "bytebase",
      style: "CRITICAL",
      title: "Failed to create issue",
      description: String(error),
    });
  }
};

const doCreateIssue = async () => {
  const request = create(CreateIssueRequestSchema, {
    parent: project.value.name,
    issue: create(IssueSchema, {
      creator: `users/${currentUser.value.email}`,
      labels: [], // No labels for direct creation
      plan: plan.value.name,
      status: IssueStatus.OPEN,
      type: Issue_Type.DATABASE_CHANGE,
      rollout: "",
    }),
  });
  const createdIssue = await issueServiceClientConnect.createIssue(request);

  // Then create the rollout from the plan.
  const rolloutRequest = create(CreateRolloutRequestSchema, {
    parent: project.value.name,
    rollout: {
      plan: plan.value.name,
    },
  });
  await rolloutServiceClientConnect.createRollout(rolloutRequest);

  // Emit status changed to refresh the UI
  events.emit("status-changed", { eager: true });

  nextTick(() => {
    router.push({
      name: PROJECT_V1_ROUTE_ISSUE_DETAIL_V1,
      params: {
        projectId: extractProjectResourceName(plan.value.name),
        issueId: extractIssueUID(createdIssue.name),
      },
    });
  });
};

const handlePlanStateChange = async (action: PlanAction) => {
  if (!plan?.value) return;

  const isClosing = action === "PLAN_CLOSE";
  const title = isClosing ? t("common.close") : t("common.reopen");
  const content = isClosing
    ? t("plan.state.close-confirm")
    : t("plan.state.reopen-confirm");
  const positiveText = title;
  const newState = isClosing ? State.DELETED : State.ACTIVE;
  const errorMessage = isClosing
    ? "Failed to close plan:"
    : "Failed to reopen plan:";

  const d = dialog.warning({
    title,
    content,
    positiveText,
    negativeText: t("common.cancel"),
    onPositiveClick: async () => {
      d.loading = true;
      try {
        await planStore.updatePlan({ ...plan.value, state: newState }, [
          "state",
        ]);
        // The plan context should automatically refresh and redirect or update the UI.
        await resourcePoller.refreshResources();
      } catch (error) {
        console.error(errorMessage, error);
        pushNotification({
          module: "bytebase",
          style: "CRITICAL",
          title: "Failed to update plan",
          description: String(error),
        });
      }
    },
  });
};
</script>
