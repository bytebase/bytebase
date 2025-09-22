<template>
  <div class="flex items-center gap-x-2">
    <!-- Export Archive Download Action for export data issues (replaces primary action) -->
    <ExportArchiveDownloadAction v-if="shouldShowExportDownload" />

    <!-- Primary action button (hidden when export download is shown) -->
    <UnifiedActionButton
      v-else-if="primaryAction && primaryAction.action !== 'EXPORT_DOWNLOAD'"
      :action="primaryAction.action"
      :disabled="disabled || primaryAction.disabled"
      :disabled-tooltip="disabled ? disabledTooltip : primaryAction.description"
      @perform-action="handleAction"
    />

    <!-- Dropdown for secondary actions -->
    <NDropdown
      v-if="secondaryActions.length > 0"
      trigger="click"
      placement="bottom-end"
      :options="dropdownOptions"
      :render-option="renderDropdownOption"
      :disabled="disabled"
      @select="handleDropdownSelect"
    >
      <NButton
        class="!px-1"
        quaternary
        size="medium"
        :disabled="disabled"
        :title="disabled ? disabledTooltip : undefined"
      >
        <EllipsisVerticalIcon class="w-5 h-5" />
      </NButton>
    </NDropdown>
  </div>
</template>

<script setup lang="ts">
import { first, orderBy } from "lodash-es";
import { EllipsisVerticalIcon } from "lucide-vue-next";
import { NButton, NDropdown, type DropdownOption } from "naive-ui";
import { computed, h } from "vue";
import type { VNode } from "vue";
import { useI18n } from "vue-i18n";
import { DropdownItemWithErrorList } from "@/components/IssueV1/components/common";
import { useCurrentUserV1, extractUserId } from "@/store";
import { IssueStatus } from "@/types/proto-es/v1/issue_service_pb";
import {
  Task_Type,
  TaskRun_ExportArchiveStatus,
  TaskRun_Status,
  type TaskRun,
} from "@/types/proto-es/v1/rollout_service_pb";
import { extractTaskRunUID, extractTaskUID } from "@/utils";
import { usePlanContext } from "../../../../logic";
import { ExportArchiveDownloadAction } from "../export";
import UnifiedActionButton from "./UnifiedActionButton.vue";
import type { ActionConfig, UnifiedAction } from "./types";

const props = defineProps<{
  primaryAction?: ActionConfig;
  secondaryActions: ActionConfig[];
  disabled?: boolean;
  disabledTooltip?: string;
}>();

const emit = defineEmits<{
  (event: "perform-action", action: UnifiedAction): void;
}>();

const { t } = useI18n();
const { plan, issue, rollout, taskRuns } = usePlanContext();
const currentUser = useCurrentUserV1();

// Check if this is a database export plan
const isExportPlan = computed(() => {
  return plan.value.specs.every(
    (spec) => spec.config.case === "exportDataConfig"
  );
});

const shouldShowExportDownload = computed(() => {
  const exportTasks =
    rollout.value?.stages
      .flatMap((stage) => stage.tasks)
      .filter((task) => {
        return task.type === Task_Type.DATABASE_EXPORT;
      }) || [];

  // Must have export data tasks
  if (exportTasks.length === 0) return false;

  // Must have an issue
  if (!issue?.value) return false;

  // Issue status must be OPEN or DONE
  if (![IssueStatus.OPEN, IssueStatus.DONE].includes(issue.value.status)) {
    return false;
  }

  // Current user must be the issue creator
  if (currentUser.value.email !== extractUserId(issue.value.creator)) {
    return false;
  }

  // Get latest task run for each export task
  const exportTaskRuns = exportTasks
    .map((task) => {
      const taskRunsForTask = taskRuns.value.filter(
        (taskRun) => extractTaskUID(taskRun.name) === extractTaskUID(task.name)
      );
      return first(
        orderBy(
          taskRunsForTask,
          (taskRun) => Number(extractTaskRunUID(taskRun.name)),
          "desc"
        )
      );
    })
    .filter(Boolean) as TaskRun[];

  // Check if export archive is ready
  if (
    exportTaskRuns.length === 0 ||
    exportTaskRuns.some(
      (taskRun) =>
        taskRun.status !== TaskRun_Status.DONE ||
        (taskRun.exportArchiveStatus !== TaskRun_ExportArchiveStatus.READY &&
          taskRun.exportArchiveStatus !== TaskRun_ExportArchiveStatus.EXPORTED)
    )
  ) {
    return false;
  }

  return true;
});

const actionDisplayName = (action: UnifiedAction): string => {
  switch (action) {
    case "ISSUE_REVIEW_APPROVE":
      return t("common.approve");
    case "ISSUE_REVIEW_REJECT":
      return t("custom-approval.issue-review.send-back");
    case "ISSUE_REVIEW_RE_REQUEST":
      return t("custom-approval.issue-review.re-request-review");
    case "ISSUE_STATUS_CLOSE":
      return t("issue.batch-transition.close");
    case "ISSUE_STATUS_REOPEN":
      return t("issue.batch-transition.reopen");
    case "ISSUE_STATUS_RESOLVE":
      return t("issue.batch-transition.resolve");
    case "ISSUE_CREATE":
      return t("plan.ready-for-review");
    case "PLAN_CLOSE":
      return t("common.close");
    case "PLAN_REOPEN":
      return t("common.reopen");
    case "ROLLOUT_START":
      return isExportPlan.value ? t("common.export") : t("common.rollout");
    case "ROLLOUT_CANCEL":
      return t("common.cancel");
    case "EXPORT_DOWNLOAD":
      return t("common.download");
  }
};

const dropdownOptions = computed(() => {
  return props.secondaryActions.map((config) => ({
    key: config.action,
    label: actionDisplayName(config.action),
    action: config.action,
    disabled: props.disabled || config.disabled,
    description: props.disabled ? props.disabledTooltip : config.description,
  }));
});

const renderDropdownOption = ({
  node,
  option,
}: {
  node: VNode;
  option: DropdownOption;
}) => {
  const actionOption = props.secondaryActions.find(
    (config) => config.action === (option as any).key
  );
  const disabled = props.disabled || actionOption?.disabled;
  const description = props.disabled
    ? props.disabledTooltip
    : actionOption?.description;
  const errors = disabled
    ? [
        description ||
          t("issue.error.you-are-not-allowed-to-perform-this-action"),
      ]
    : [];
  return h(
    DropdownItemWithErrorList,
    {
      errors,
      placement: "left",
    },
    {
      default: () => node,
    }
  );
};

const handleAction = (action: UnifiedAction) => {
  emit("perform-action", action);
};

const handleDropdownSelect = (key: string) => {
  const option = dropdownOptions.value.find((opt) => opt.key === key);
  if (option && !option.disabled) {
    handleAction(option.action);
  }
};
</script>
