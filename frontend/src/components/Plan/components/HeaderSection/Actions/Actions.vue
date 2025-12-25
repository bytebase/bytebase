<template>
  <CreateButton v-if="isCreating" />
  <template v-else>
    <!-- Show CreateIssueButton when ISSUE_CREATE is the primary action -->
    <template v-if="showCreateIssueButton">
      <div class="flex items-center gap-x-2">
        <CreateIssueButton />
        <!-- Secondary actions dropdown -->
        <UnifiedActionGroup
          v-if="secondaryActions.length > 0"
          :secondary-actions="secondaryActions"
          :disabled="actionsDisabled"
          :disabled-tooltip="disabledTooltip"
          @perform-action="handlePerformAction"
        />
      </div>
    </template>

    <!-- Show IssueReviewButton when review actions are available -->
    <template v-else-if="showReviewButton">
      <div class="flex items-center gap-x-2">
        <IssueReviewButton
          :can-approve="canApprove"
          :can-reject="canReject"
          :disabled="actionsDisabled"
          :disabled-tooltip="disabledTooltip"
        />
        <!-- Secondary actions dropdown (excluding review actions) -->
        <UnifiedActionGroup
          v-if="nonReviewSecondaryActions.length > 0"
          :secondary-actions="nonReviewSecondaryActions"
          :disabled="actionsDisabled"
          :disabled-tooltip="disabledTooltip"
          @perform-action="handlePerformAction"
        />
      </div>
    </template>

    <!-- Show unified actions for other plan states -->
    <template v-else>
      <UnifiedActionGroup
        :primary-action="primaryAction"
        :secondary-actions="secondaryActions"
        :disabled="actionsDisabled"
        :disabled-tooltip="disabledTooltip"
        @perform-action="handlePerformAction"
      />
    </template>

    <!-- Panels -->
    <template v-if="issue">
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
import { computed, ref } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import { usePlanContext } from "@/components/Plan/logic";
import { useResourcePoller } from "@/components/Plan/logic/poller";
import { useEditorState } from "@/components/Plan/logic/useEditorState";
import TaskRolloutActionPanel from "@/components/RolloutV1/components/TaskRolloutActionPanel.vue";
import { rolloutServiceClientConnect } from "@/grpcweb";
import { PROJECT_V1_ROUTE_ROLLOUT_DETAIL } from "@/router/dashboard/projectV1";
import { pushNotification, useCurrentProjectV1 } from "@/store";
import { usePlanStore } from "@/store/modules/v1/plan";
import { State } from "@/types/proto-es/v1/common_pb";
import {
  CreateRolloutRequestSchema,
  Task_Type,
} from "@/types/proto-es/v1/rollout_service_pb";
import {
  extractProjectResourceName,
  extractRolloutUID,
  isValidIssueName,
  isValidPlanName,
} from "@/utils";
import { CreateButton, CreateIssueButton } from "./create";
import { IssueStatusActionPanel } from "./panels";
import {
  type ActionConfig,
  IssueReviewButton,
  type IssueStatusAction,
  type PlanAction,
  type RolloutAction,
  type UnifiedAction,
  UnifiedActionGroup,
  usePlanAction,
} from "./unified";

const { t } = useI18n();
const router = useRouter();
const dialog = useDialog();
const resourcePoller = useResourcePoller();
const { project } = useCurrentProjectV1();
const { isCreating, plan, issue, rollout, events } = usePlanContext();
const planStore = usePlanStore();
const editorState = useEditorState();
const { availableActions, canApprove, canReject } = usePlanAction();
const creatingRollout = ref(false);

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
const pendingStatusAction = ref<IssueStatusAction | undefined>(undefined);
const pendingRolloutAction = ref<RolloutAction | undefined>(undefined);

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

// Issue-only mode: when viewing an issue without a valid plan (legacy issues)
const isIssueOnly = computed(() => {
  return (
    !isValidPlanName(plan.value.name) && isValidIssueName(issue.value?.name)
  );
});

// Check if we should show CreateIssueButton instead of UnifiedActionGroup
const showCreateIssueButton = computed(() => {
  // Show CreateIssueButton when ISSUE_CREATE is available and not issue-only
  return !isIssueOnly.value && availableActions.value.includes("ISSUE_CREATE");
});

// Show the unified review button when ISSUE_REVIEW is available
const showReviewButton = computed(() => {
  return availableActions.value.includes("ISSUE_REVIEW");
});

// Secondary actions excluding review action (for when review button is shown)
const nonReviewSecondaryActions = computed((): ActionConfig[] => {
  return secondaryActions.value.filter(
    (config) => config.action !== "ISSUE_REVIEW"
  );
});

const primaryAction = computed((): ActionConfig | undefined => {
  const actions = availableActions.value;

  // PLAN_REOPEN is primary when plan is deleted (skip for issue-only)
  if (!isIssueOnly.value && actions.includes("PLAN_REOPEN")) {
    return { action: "PLAN_REOPEN" };
  }

  // ISSUE_STATUS_REOPEN is primary when issue is closed or done
  if (actions.includes("ISSUE_STATUS_REOPEN")) {
    return { action: "ISSUE_STATUS_REOPEN" };
  }

  // ISSUE_REVIEW is handled by IssueReviewButton, not UnifiedActionGroup
  // So we skip it here and let showReviewButton handle it

  // ISSUE_STATUS_RESOLVE is primary when issue can be resolved
  if (actions.includes("ISSUE_STATUS_RESOLVE")) {
    return { action: "ISSUE_STATUS_RESOLVE" };
  }

  // ROLLOUT_CREATE is primary when issue exists but rollout doesn't
  if (actions.includes("ROLLOUT_CREATE")) {
    return { action: "ROLLOUT_CREATE" };
  }

  // Rollout actions as primary actions (high priority for force rollout scenarios)
  if (actions.includes("ROLLOUT_START")) {
    return { action: "ROLLOUT_START" };
  }

  return undefined;
});

const secondaryActions = computed((): ActionConfig[] => {
  const actions = availableActions.value;
  const secondary: ActionConfig[] = [];

  // ISSUE_REVIEW is handled by IssueReviewButton, not in secondary actions
  if (actions.includes("ROLLOUT_CREATE")) {
    secondary.push({ action: "ROLLOUT_CREATE" });
  }
  if (actions.includes("ROLLOUT_CANCEL")) {
    secondary.push({ action: "ROLLOUT_CANCEL" });
  }
  if (actions.includes("ISSUE_STATUS_CLOSE")) {
    secondary.push({ action: "ISSUE_STATUS_CLOSE" });
  }
  // Skip PLAN_CLOSE for issue-only cases
  if (!isIssueOnly.value && actions.includes("PLAN_CLOSE")) {
    secondary.push({ action: "PLAN_CLOSE" });
  }

  return secondary;
});

const handlePerformAction = async (action: UnifiedAction) => {
  switch (action) {
    // ISSUE_REVIEW is handled directly by IssueReviewButton
    case "ISSUE_STATUS_CLOSE":
    case "ISSUE_STATUS_REOPEN":
    case "ISSUE_STATUS_RESOLVE":
      pendingStatusAction.value = action as IssueStatusAction;
      break;
    case "PLAN_CLOSE":
      await handlePlanStateChange("PLAN_CLOSE");
      break;
    case "PLAN_REOPEN":
      await handlePlanStateChange("PLAN_REOPEN");
      break;
    case "ROLLOUT_CREATE":
      await handleCreateRollout();
      break;
    case "ROLLOUT_START":
    case "ROLLOUT_CANCEL":
      pendingRolloutAction.value = action as RolloutAction;
      break;
  }
};

const handlePlanStateChange = async (action: PlanAction) => {
  if (!plan?.value) return;

  const isClosing = action === "PLAN_CLOSE";
  const title = isClosing ? t("common.close") : t("common.reopen");
  const content = isClosing
    ? t("plan.state.close-confirm")
    : t("plan.state.reopen-confirm");
  const newState = isClosing ? State.DELETED : State.ACTIVE;

  const d = dialog.warning({
    title,
    content,
    positiveText: title,
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
        pushNotification({
          module: "bytebase",
          style: "CRITICAL",
          title: t("common.failed"),
          description: String(error),
        });
      }
    },
  });
};

const handleCreateRollout = async () => {
  if (creatingRollout.value) return;

  creatingRollout.value = true;
  try {
    const request = create(CreateRolloutRequestSchema, {
      parent: project.value.name,
      rollout: {
        plan: plan.value.name,
      },
    });
    const createdRollout =
      await rolloutServiceClientConnect.createRollout(request);

    pushNotification({
      module: "bytebase",
      style: "SUCCESS",
      title: t("common.created"),
    });

    // Redirect to rollout page
    router.push({
      name: PROJECT_V1_ROUTE_ROLLOUT_DETAIL,
      params: {
        projectId: extractProjectResourceName(project.value.name),
        rolloutId: extractRolloutUID(createdRollout.name),
      },
    });
  } catch (error) {
    pushNotification({
      module: "bytebase",
      style: "CRITICAL",
      title: t("common.failed"),
      description: String(error),
    });
  } finally {
    creatingRollout.value = false;
  }
};

const handleRolloutActionConfirm = () => {
  // The TaskRolloutActionPanel handles the actual execution
  // We just need to emit status change to refresh the UI
  events.emit("status-changed", { eager: true });
  pendingRolloutAction.value = undefined;
};
</script>
