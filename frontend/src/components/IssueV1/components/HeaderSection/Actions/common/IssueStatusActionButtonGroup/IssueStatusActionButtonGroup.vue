<template>
  <div v-if="displayMode === 'BUTTON'" class="flex items-center gap-x-2">
    <IssueStatusActionButton
      v-for="(action, index) in issueStatusActionList"
      :key="index"
      :action="action"
      @perform-action="
        (action) => events.emit('perform-issue-status-action', { action })
      "
    />

    <NDropdown
      v-if="extraActionList.length > 0"
      trigger="click"
      placement="bottom-end"
      :options="extraActionList"
      :render-option="renderDropdownOption"
      @select="handleDropdownSelect"
    >
      <NButton :quaternary="true" size="medium" style="--n-padding: 0 4px">
        <heroicons:ellipsis-vertical class="w-6 h-6" />
      </NButton>
    </NDropdown>
  </div>

  <NDropdown
    v-if="displayMode === 'DROPDOWN' && mergedDropdownActionList.length > 0"
    trigger="click"
    placement="bottom-end"
    :options="mergedDropdownActionList"
    :render-option="renderDropdownOption"
    @select="handleDropdownSelect"
  >
    <NButton :quaternary="true" size="medium" style="--n-padding: 0 4px">
      <heroicons:ellipsis-vertical class="w-6 h-6" />
    </NButton>
  </NDropdown>
</template>

<script setup lang="ts">
import { type DropdownOption, NButton, NDropdown } from "naive-ui";
import type { VNode } from "vue";
import { computed, h } from "vue";
import { useI18n } from "vue-i18n";
import { DropdownItemWithErrorList } from "@/components/IssueV1/components/common";
import type {
  IssueStatusAction,
  TaskRolloutAction,
} from "@/components/IssueV1/logic";
import {
  allowUserToApplyIssueStatusAction,
  issueStatusActionDisplayName,
  useIssueContext,
} from "@/components/IssueV1/logic";
import type { Task } from "@/types/proto-es/v1/rollout_service_pb";
import type { ExtraActionOption } from "../types";
import IssueStatusActionButton from "./IssueStatusActionButton.vue";

type IssueActionDropdownOption = ExtraActionOption & {
  description?: string;
};

const props = defineProps<{
  displayMode: "BUTTON" | "DROPDOWN";
  issueStatusActionList: IssueStatusAction[];
  extraActionList: ExtraActionOption[];
}>();

const { t } = useI18n();
const { issue, events } = useIssueContext();

const issueStatusActionDropdownOptions = computed(() => {
  return props.issueStatusActionList.map<IssueActionDropdownOption>(
    (action) => {
      const [ok, reason] = allowUserToApplyIssueStatusAction(
        issue.value,
        action
      );
      return {
        key: action,
        label: issueStatusActionDisplayName(action),
        type: "ISSUE",
        action: action,
        target: issue.value,
        disabled: !ok,
        description: reason,
      };
    }
  );
});

const mergedDropdownActionList = computed(() => {
  return [...issueStatusActionDropdownOptions.value, ...props.extraActionList];
});

const renderDropdownOption = ({
  node,
  option,
}: {
  node: VNode;
  option: DropdownOption;
}) => {
  const { disabled, description } = option as IssueActionDropdownOption;
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

const handleDropdownSelect = (key: string, dropdownOption: DropdownOption) => {
  const option = dropdownOption as ExtraActionOption;
  if (option.type === "ISSUE") {
    events.emit("perform-issue-status-action", {
      action: option.action as IssueStatusAction,
    });
  }
  if (option.type === "TASK-BATCH") {
    events.emit("perform-task-rollout-action", {
      action: option.action as TaskRolloutAction,
      tasks: option.target as Task[],
    });
  }
  if (option.type === "TASK") {
    events.emit("perform-task-rollout-action", {
      action: option.action as TaskRolloutAction,
      tasks: [option.target as Task],
    });
  }
};
</script>
