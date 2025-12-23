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

    <!-- Rollout Action Panel -->
    <template v-if="rollout && rolloutStage">
      <TaskRolloutActionPanel
        :show="Boolean(pendingRolloutAction)"
        :action="
          pendingRolloutAction === 'ROLLOUT_START'
            ? 'RUN'
            : pendingRolloutAction === 'ROLLOUT_CANCEL'
              ? 'CANCEL'
              : 'RUN'
        "
        :target="{ type: 'tasks', stage: rolloutStage }"
        @close="pendingRolloutAction = undefined"
        @confirm="handleRolloutActionConfirm"
      />
    </template>
  </template>
</template>

<script setup lang="ts">
import { create } from "@bufbuild/protobuf";
import { useDialog } from "naive-ui";
import { computed, nextTick, ref } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import { useSpecsValidation } from "@/components/Plan/components/common";
import { usePlanCheckStatus, usePlanContext } from "@/components/Plan/logic";
import { useResourcePoller } from "@/components/Plan/logic/poller";
import { useEditorState } from "@/components/Plan/logic/useEditorState";
import {
  issueServiceClientConnect,
  rolloutServiceClientConnect,
} from "@/grpcweb";
import { PROJECT_V1_ROUTE_ISSUE_DETAIL_V1 } from "@/router/dashboard/projectV1";
import {
  pushNotification,
  useCurrentProjectV1,
  useCurrentUserV1,
} from "@/store";
import { usePlanStore } from "@/store/modules/v1/plan";
import { State } from "@/types/proto-es/v1/common_pb";
import {
  CreateIssueRequestSchema,
  Issue_Type,
  IssueSchema,
  IssueStatus,
} from "@/types/proto-es/v1/issue_service_pb";
import type { Plan, Plan_Spec } from "@/types/proto-es/v1/plan_service_pb";
import {
  CreateRolloutRequestSchema,
  Task_Type,
} from "@/types/proto-es/v1/rollout_service_pb";
import { Advice_Level } from "@/types/proto-es/v1/sql_service_pb";
import {
  extractIssueUID,
  extractProjectResourceName,
  isValidIssueName,
  isValidPlanName,
} from "@/utils";
import TaskRolloutActionPanel from "../../RolloutView/TaskRolloutActionPanel.vue";
import { CreateButton } from "./create";
import { IssueReviewActionPanel, IssueStatusActionPanel } from "./panels";
import {
  type ActionConfig,
  type IssueReviewAction,
  type IssueStatusAction,
  type PlanAction,
  type RolloutAction,
  type UnifiedAction,
  UnifiedActionGroup,
} from "./unified";
import { usePlanAction } from "./unified/action";

const { t } = useI18n();
const router = useRouter();
const dialog = useDialog();
const resourcePoller = useResourcePoller();
const { isCreating, plan, issueLabels, issue, rollout, events } =
  usePlanContext();
const currentUser = useCurrentUserV1();
const { project } = useCurrentProjectV1();
const planStore = usePlanStore();
const editorState = useEditorState();
const { availableActions } = usePlanAction();

// Use the validation hook for all specs
const { isSpecEmpty } = useSpecsValidation(computed(() => plan.value.specs));

// Use plan check status for issue creation validation
const {
  getOverallStatus: planCheckSummaryStatus,
  hasRunning: hasRunningPlanChecks,
} = usePlanCheckStatus(plan);

// Policy for restricting issue creation when plan checks fail

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
const pendingRolloutAction = ref<RolloutAction | undefined>(undefined);

// Loading state for issue creation to prevent race conditions
const isCreatingIssue = ref(false);

// Get the stage that contains database creation or export tasks
const rolloutStage = computed(() => {
  if (!rollout.value) return undefined;

  // Find the first stage with database creation or export tasks
  return rollout.value.stages.find((stage) =>
    stage.tasks.some(
      (task) =>
        task.type === Task_Type.DATABASE_CREATE ||
        task.type === Task_Type.DATABASE_EXPORT
    )
  );
});

const primaryAction = computed((): ActionConfig | undefined => {
  const actions = availableActions.value;
  const isIssueOnly =
    !isValidPlanName(plan.value.name) && isValidIssueName(issue.value?.name);

  // Skip plan-specific actions for issue-only cases
  if (isIssueOnly) {
    // Skip ISSUE_CREATE, PLAN_REOPEN checks and go directly to issue actions
  }
  // ISSUE_CREATE is the highest priority when no issue exists
  else if (actions.includes("ISSUE_CREATE")) {
    // Check if all specs have valid statements (not empty)
    const hasValidSpecs = !plan.value.specs.some((spec) => isSpecEmpty(spec));

    // Check plan check status and policy restrictions
    const planChecksFailed =
      planCheckSummaryStatus.value === Advice_Level.ERROR;
    const isRestrictedByPolicy =
      planChecksFailed && (project.value.enforceSqlReview || false);
    const hasRunningChecks = hasRunningPlanChecks.value;

    const errors: string[] = [];
    if (!hasValidSpecs) {
      errors.push("Missing statement");
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
    if (project.value.forceIssueLabels && issueLabels.value.length === 0) {
      errors.push("Missing issue labels");
    }

    return {
      action: "ISSUE_CREATE",
      disabled: errors.length > 0,
      description: errors.length > 0 ? errors.join(", ") : undefined,
    };
  }

  // PLAN_REOPEN is primary when plan is deleted (skip for issue-only)
  if (!isIssueOnly && actions.includes("PLAN_REOPEN")) {
    return { action: "PLAN_REOPEN" };
  }

  // ISSUE_STATUS_REOPEN is primary when issue is closed or done
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

  // ISSUE_STATUS_RESOLVE is primary when issue can be resolved
  if (actions.includes("ISSUE_STATUS_RESOLVE")) {
    return { action: "ISSUE_STATUS_RESOLVE" };
  }

  // Rollout actions as primary actions (high priority for force rollout scenarios)
  if (actions.includes("ROLLOUT_START")) {
    return { action: "ROLLOUT_START" };
  }

  return undefined;
});

const secondaryActions = computed((): ActionConfig[] => {
  const actions = availableActions.value;
  const isIssueOnly =
    !isValidPlanName(plan.value.name) && isValidIssueName(issue.value?.name);
  const secondary: ActionConfig[] = [];

  if (actions.includes("ISSUE_REVIEW_REJECT")) {
    secondary.push({ action: "ISSUE_REVIEW_REJECT" });
  }
  if (actions.includes("ROLLOUT_CANCEL")) {
    secondary.push({ action: "ROLLOUT_CANCEL" });
  }
  if (actions.includes("ISSUE_STATUS_CLOSE")) {
    secondary.push({ action: "ISSUE_STATUS_CLOSE" });
  }
  // Skip PLAN_CLOSE for issue-only cases
  if (!isIssueOnly && actions.includes("PLAN_CLOSE")) {
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
    case "ISSUE_STATUS_RESOLVE":
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
    case "ROLLOUT_START":
    case "ROLLOUT_CANCEL":
      pendingRolloutAction.value = action as RolloutAction;
      break;
  }
};

const handleIssueCreate = async () => {
  if (!plan?.value) return;

  // Prevent race condition: check if already creating issue
  if (isCreatingIssue.value) {
    return;
  }
  isCreatingIssue.value = true;

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
  } finally {
    isCreatingIssue.value = false;
  }
};

// Helper function to determine issue type from plan specs
const getIssueTypeFromPlan = (planValue: Plan): Issue_Type => {
  // Check if any spec is for data export
  const hasExportDataSpec = planValue.specs.some(
    (spec: Plan_Spec) => spec.config?.case === "exportDataConfig"
  );

  if (hasExportDataSpec) {
    return Issue_Type.DATABASE_EXPORT;
  }

  // For both database creation and changes, use DATABASE_CHANGE
  return Issue_Type.DATABASE_CHANGE;
};

const doCreateIssue = async () => {
  const createIssueRequest = create(CreateIssueRequestSchema, {
    parent: project.value.name,
    issue: create(IssueSchema, {
      creator: `users/${currentUser.value.email}`,
      labels: [], // No labels for direct creation
      plan: plan.value.name,
      status: IssueStatus.OPEN,
      type: getIssueTypeFromPlan(plan.value),
      rollout: "",
    }),
  });

  // Create issue first
  const createdIssue =
    await issueServiceClientConnect.createIssue(createIssueRequest);

  const createRolloutRequest = create(CreateRolloutRequestSchema, {
    parent: project.value.name,
    rollout: {
      plan: plan.value.name,
    },
  });

  // Then create rollout
  await rolloutServiceClientConnect.createRollout(createRolloutRequest);

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

const handleRolloutActionConfirm = () => {
  // The TaskRolloutActionPanel handles the actual execution
  // We just need to emit status change to refresh the UI
  events.emit("status-changed", { eager: true });
  pendingRolloutAction.value = undefined;
};
</script>
