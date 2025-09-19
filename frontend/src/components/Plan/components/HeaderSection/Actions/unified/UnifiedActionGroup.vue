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
import { last } from "lodash-es";
import { EllipsisVerticalIcon } from "lucide-vue-next";
import { NButton, NDropdown, type DropdownOption } from "naive-ui";
import { computed, h } from "vue";
import type { VNode } from "vue";
import { useI18n } from "vue-i18n";
import { DropdownItemWithErrorList } from "@/components/IssueV1/components/common";
import { useCurrentUserV1, extractUserId } from "@/store";
import { IssueStatus } from "@/types/proto-es/v1/issue_service_pb";
import {
  TaskRun_ExportArchiveStatus,
  Task_Type,
} from "@/types/proto-es/v1/rollout_service_pb";
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

// Export download logic
const hasExportDataTask = computed(() => {
  return (
    rollout.value?.stages.some((stage) =>
      stage.tasks.some((task) => task.type === Task_Type.DATABASE_EXPORT)
    ) || false
  );
});

const taskRun = computed(() => {
  if (!rollout.value || !taskRuns.value.length) return undefined;

  // Find export tasks first
  const exportTasks = rollout.value.stages
    .flatMap((stage) => stage.tasks)
    .filter((task) => {
      // Find tasks that belong to export data specs
      const exportSpec = plan.value.specs.find(
        (spec) =>
          spec.config?.case === "exportDataConfig" && spec.id === task.specId
      );
      return !!exportSpec;
    });

  // Get task runs for export tasks
  const exportTaskRuns = taskRuns.value.filter((taskRun) =>
    exportTasks.some((task) => taskRun.name.startsWith(task.name + "/"))
  );

  return last(exportTaskRuns);
});

const shouldShowExportDownload = computed(() => {
  // Must have export data tasks
  if (!hasExportDataTask.value) return false;

  // Must have an issue
  if (!issue?.value) return false;

  // Issue status must be OPEN or DONE
  if (![IssueStatus.OPEN, IssueStatus.DONE].includes(issue.value.status)) {
    return false;
  }

  // Current user must be the issue creator
  const currentUserEmail = currentUser.value.email;
  const issueCreator = extractUserId(issue.value.creator);
  if (currentUserEmail !== issueCreator) return false;

  // Check if export archive is ready
  if (
    taskRun.value?.exportArchiveStatus !== TaskRun_ExportArchiveStatus.READY
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
