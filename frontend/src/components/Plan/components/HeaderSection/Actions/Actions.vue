<template>
  <CreateButton v-if="isCreating" />
  <div v-else class="flex items-center gap-x-2">
    <!-- Rollout ready link for database change issues -->
    <RolloutReadyLink v-if="shouldShowRolloutReadyLink" />

    <!-- Primary action: special components for specific actions -->
    <CreateIssueButton v-if="primaryAction?.id === 'ISSUE_CREATE'" />
    <IssueReviewButton
      v-else-if="primaryAction?.id === 'ISSUE_REVIEW'"
      :can-approve="context.permissions.canApprove"
      :can-reject="context.permissions.canReject"
      :disabled="isActionDisabled(primaryAction)"
    />
    <ExportArchiveDownloadAction
      v-else-if="primaryAction?.id === 'EXPORT_DOWNLOAD'"
    />
    <ActionButton
      v-else-if="primaryAction"
      :action="primaryAction"
      :context="context"
      :disabled="isActionDisabled(primaryAction)"
      :disabled-reason="getDisabledReason(primaryAction)"
      @execute="handlePerformAction"
    />

    <!-- Secondary actions dropdown -->
    <ActionDropdown
      v-if="secondaryActions.length > 0"
      :actions="secondaryActions"
      :context="context"
      :global-disabled="globalDisabled"
      :global-disabled-reason="globalDisabledReason"
      :is-action-disabled="isActionDisabled"
      :get-disabled-reason="getDisabledReason"
      @execute="handlePerformAction"
    />

    <!-- Rollout Action Panel (only when rollout exists) -->
    <TaskRolloutActionPanel
      v-if="pendingRolloutAction && rolloutStage"
      :show="true"
      :action="pendingRolloutAction === 'ROLLOUT_CANCEL' ? 'CANCEL' : 'RUN'"
      :target="{ type: 'tasks', stage: rolloutStage }"
      @close="pendingRolloutAction = undefined"
      @confirm="handleRolloutActionConfirm"
    />

    <!-- Rollout Create Panel -->
    <RolloutCreatePanel
      :show="showRolloutCreatePanel"
      :context="context"
      @close="showRolloutCreatePanel = false"
      @confirm="showRolloutCreatePanel = false"
    />
  </div>
</template>

<script setup lang="ts">
import { create } from "@bufbuild/protobuf";
import { useDialog } from "naive-ui";
import { computed, ref } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import { usePlanContext, useRolloutReadyLink } from "@/components/Plan/logic";
import { useResourcePoller } from "@/components/Plan/logic/poller";
import TaskRolloutActionPanel from "@/components/RolloutV1/components/TaskRolloutActionPanel.vue";
import {
  issueServiceClientConnect,
  rolloutServiceClientConnect,
} from "@/connect";
import { PROJECT_V1_ROUTE_PLAN_ROLLOUT } from "@/router/dashboard/projectV1";
import { pushNotification, useCurrentProjectV1 } from "@/store";
import { usePlanStore } from "@/store/modules/v1/plan";
import { State } from "@/types/proto-es/v1/common_pb";
import {
  BatchUpdateIssuesStatusRequestSchema,
  IssueStatus,
} from "@/types/proto-es/v1/issue_service_pb";
import {
  BatchRunTasksRequestSchema,
  CreateRolloutRequestSchema,
} from "@/types/proto-es/v1/rollout_service_pb";
import {
  extractPlanUIDFromRolloutName,
  extractProjectResourceName,
} from "@/utils";
import { CreateButton, CreateIssueButton, RolloutCreatePanel } from "./create";
import { ExportArchiveDownloadAction } from "./export";
import RolloutReadyLink from "./RolloutReadyLink.vue";
import {
  ActionButton,
  ActionDropdown,
  IssueReviewButton,
  useActionRegistry,
} from "./registry";
import type {
  IssueStatusAction,
  RolloutAction,
  UnifiedAction,
} from "./registry/types";

const { t } = useI18n();
const router = useRouter();
const dialog = useDialog();
const resourcePoller = useResourcePoller();
const { project } = useCurrentProjectV1();
const { isCreating, plan, issue, rollout, events } = usePlanContext();
const planStore = usePlanStore();
const creatingRollout = ref(false);
const { shouldShow: shouldShowRolloutReadyLink } = useRolloutReadyLink();

// Use the action registry
const {
  context,
  primaryAction,
  secondaryActions,
  isActionDisabled,
  getDisabledReason,
} = useActionRegistry();

// Global disabled state (for dropdown)
const globalDisabled = computed(() => {
  if (!primaryAction.value) return false;
  return isActionDisabled(primaryAction.value);
});

const globalDisabledReason = computed(() => {
  if (!primaryAction.value) return undefined;
  return getDisabledReason(primaryAction.value);
});

// Panel visibility state
const pendingRolloutAction = ref<RolloutAction | undefined>(undefined);
const showRolloutCreatePanel = ref(false);

// Get the first stage for database creation/export rollouts
// (the panel handles all stages internally for these task types)
const rolloutStage = computed(() => rollout.value?.stages[0]);

const handlePerformAction = async (action: UnifiedAction) => {
  switch (action) {
    case "ISSUE_STATUS_CLOSE":
    case "ISSUE_STATUS_REOPEN":
    case "ISSUE_STATUS_RESOLVE":
      await handleIssueStatusChange(action as IssueStatusAction);
      break;
    case "PLAN_CLOSE":
      await handlePlanStateChange("PLAN_CLOSE");
      break;
    case "PLAN_REOPEN":
      await handlePlanStateChange("PLAN_REOPEN");
      break;
    case "ROLLOUT_CREATE":
      showRolloutCreatePanel.value = true;
      break;
    case "ROLLOUT_START":
      // For deferred rollout plans without rollout, create and run all tasks
      if (context.value.hasDeferredRollout && !rollout.value) {
        await handleCreateRollout({ runAllTasks: true });
        return;
      }
      pendingRolloutAction.value = action as RolloutAction;
      break;
    case "ROLLOUT_CANCEL":
      pendingRolloutAction.value = action as RolloutAction;
      break;
  }
};

const handleIssueStatusChange = async (action: IssueStatusAction) => {
  const issueValue = issue?.value;
  if (!issueValue) return;

  const actionConfig = {
    ISSUE_STATUS_CLOSE: {
      title: t("common.close"),
      content: t("issue.status-transition.modal.close"),
      status: IssueStatus.CANCELED,
    },
    ISSUE_STATUS_REOPEN: {
      title: t("common.reopen"),
      content: t("issue.status-transition.modal.reopen"),
      status: IssueStatus.OPEN,
    },
    ISSUE_STATUS_RESOLVE: {
      title: t("issue.batch-transition.resolve"),
      content: t("issue.status-transition.modal.resolve"),
      status: IssueStatus.DONE,
    },
  }[action];

  const d = dialog.warning({
    title: actionConfig.title,
    content: actionConfig.content,
    positiveText: actionConfig.title,
    negativeText: t("common.cancel"),
    onPositiveClick: async () => {
      d.loading = true;
      try {
        const request = create(BatchUpdateIssuesStatusRequestSchema, {
          parent: project.value.name,
          issues: [issueValue.name],
          status: actionConfig.status,
        });
        await issueServiceClientConnect.batchUpdateIssuesStatus(request);
        events.emit("perform-issue-status-action", { action });
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

const handlePlanStateChange = async (action: "PLAN_CLOSE" | "PLAN_REOPEN") => {
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

// Create rollout and optionally run all tasks (for export plans)
const handleCreateRollout = async (options?: { runAllTasks?: boolean }) => {
  if (creatingRollout.value) return;

  creatingRollout.value = true;
  try {
    const request = create(CreateRolloutRequestSchema, {
      parent: plan.value.name,
    });
    const createdRollout =
      await rolloutServiceClientConnect.createRollout(request);

    if (options?.runAllTasks) {
      // Run all tasks in each stage (for export/create plans)
      for (const stage of createdRollout.stages) {
        const runRequest = create(BatchRunTasksRequestSchema, {
          parent: stage.name,
          tasks: stage.tasks.map((task) => task.name),
        });
        await rolloutServiceClientConnect.batchRunTasks(runRequest);
      }
      await resourcePoller.refreshResources();
      events.emit("status-changed", { eager: true });
    } else {
      // Navigate to rollout page
      pushNotification({
        module: "bytebase",
        style: "SUCCESS",
        title: t("common.created"),
      });
      router.push({
        name: PROJECT_V1_ROUTE_PLAN_ROLLOUT,
        params: {
          projectId: extractProjectResourceName(project.value.name),
          planId: extractPlanUIDFromRolloutName(createdRollout.name),
        },
      });
    }
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
  events.emit("status-changed", { eager: true });
  pendingRolloutAction.value = undefined;
};
</script>
