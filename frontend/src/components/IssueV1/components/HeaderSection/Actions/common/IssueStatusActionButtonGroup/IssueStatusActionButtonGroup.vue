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
import type { DropdownOption } from "naive-ui";
import { NButton, NDropdown } from "naive-ui";
import type { VNode } from "vue";
import { computed, h } from "vue";
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
import { useCurrentUserV1 } from "@/store";
import type { Task } from "@/types/proto/v1/rollout_service";
import type { ExtraActionOption } from "../types";
import IssueStatusActionButton from "./IssueStatusActionButton.vue";

const props = defineProps<{
  displayMode: "BUTTON" | "DROPDOWN";
  issueStatusActionList: IssueStatusAction[];
  extraActionList: ExtraActionOption[];
}>();

const { issue, events } = useIssueContext();
const currentUser = useCurrentUserV1();

const issueStatusActionDropdownOptions = computed(() => {
  return props.issueStatusActionList.map<ExtraActionOption>((action) => {
    return {
      key: action,
      label: issueStatusActionDisplayName(action),
      type: "ISSUE",
      action: action,
      target: issue.value,
      disabled: !allowUserToApplyIssueStatusAction(
        issue.value,
        currentUser.value,
        action
      ),
    };
  });
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
  const errors = option.disabled
    ? ["You are not allowed to perform this action"]
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
