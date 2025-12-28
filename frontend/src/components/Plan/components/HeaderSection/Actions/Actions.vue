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
} from "@/grpcweb";
import { PROJECT_V1_ROUTE_ROLLOUT_DETAIL } from "@/router/dashboard/projectV1";
import { pushNotification, useCurrentProjectV1 } from "@/store";
import { usePlanStore } from "@/store/modules/v1/plan";
import { State } from "@/types/proto-es/v1/common_pb";
import {
  BatchUpdateIssuesStatusRequestSchema,
  IssueStatus,
} from "@/types/proto-es/v1/issue_service_pb";
import {
  CreateRolloutRequestSchema,
  Task_Type,
} from "@/types/proto-es/v1/rollout_service_pb";
import { extractProjectResourceName, extractRolloutUID } from "@/utils";
import { CreateButton, CreateIssueButton, RolloutCreatePanel } from "./create";
import { ExportArchiveDownloadAction } from "./export";
import RolloutReadyLink from "./RolloutReadyLink.vue";
import { ActionButton, ActionDropdown, useActionRegistry } from "./registry";
import type {
  IssueStatusAction,
  RolloutAction,
  UnifiedAction,
} from "./registry/types";
import { IssueReviewButton } from "./unified";

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

// Helper to check if rollout creation has warnings
const hasRolloutCreationWarnings = computed(
  () => context.value.rolloutCreationWarnings.hasAny
);

// Get the stage that contains database creation or export tasks
const rolloutStage = computed(() => {
  if (!rollout.value) return undefined;
  return rollout.value.stages.find((stage) =>
    stage.tasks.some(
      (task) =>
        task.type === Task_Type.DATABASE_CREATE ||
        task.type === Task_Type.DATABASE_EXPORT
    )
  );
});

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
      // Show panel if there are warnings; otherwise create immediately
      if (hasRolloutCreationWarnings.value) {
        showRolloutCreatePanel.value = true;
      } else {
        await handleCreateRollout();
      }
      break;
    case "ROLLOUT_START":
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

const handleCreateRollout = async () => {
  if (creatingRollout.value) return;

  creatingRollout.value = true;
  try {
    const request = create(CreateRolloutRequestSchema, {
      parent: plan.value.name,
    });
    const createdRollout =
      await rolloutServiceClientConnect.createRollout(request);

    pushNotification({
      module: "bytebase",
      style: "SUCCESS",
      title: t("common.created"),
    });

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
  events.emit("status-changed", { eager: true });
  pendingRolloutAction.value = undefined;
};
</script>
