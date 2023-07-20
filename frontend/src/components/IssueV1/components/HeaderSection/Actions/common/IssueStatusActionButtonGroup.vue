<template>
  <div v-if="displayMode === 'BUTTON'" class="flex items-center gap-x-2">
    <NButton
      v-for="(action, index) in issueStatusActionList"
      :key="index"
      :disabled="!allowApplyIssueStatusAction(action)"
      size="large"
      v-bind="issueStatusActionButtonProps(action)"
      @click.prevent="$emit('apply-issue-action', action)"
    >
      {{ issueStatusActionDisplayName(action) }}
    </NButton>
    <NDropdown
      v-if="extraActionList.length > 0"
      trigger="click"
      placement="bottom-end"
      :options="extraActionList"
      @select="handleDropdownSelect"
    >
      <NButton :quaternary="true" size="large" style="--n-padding: 0 4px">
        <heroicons:ellipsis-vertical class="w-6 h-6" />
      </NButton>
    </NDropdown>
  </div>

  <NDropdown
    v-if="displayMode === 'DROPDOWN' && mergedDropdownActionList.length > 0"
    trigger="click"
    placement="bottom-end"
    :options="mergedDropdownActionList"
    @select="handleDropdownSelect"
  >
    <NButton :quaternary="true" size="large" style="--n-padding: 0 4px">
      <heroicons:ellipsis-vertical class="w-6 h-6" />
    </NButton>
  </NDropdown>
</template>

<script setup lang="ts">
import { computed } from "vue";
import { NButton, DropdownOption } from "naive-ui";

import { Task } from "@/types/proto/v1/rollout_service";
import { ExtraActionOption } from "./types";
import {
  IssueStatusAction,
  TaskRolloutAction,
  issueStatusActionDisplayName,
  issueStatusActionButtonProps,
  useIssueContext,
} from "@/components/IssueV1/logic";

const props = defineProps<{
  displayMode: "BUTTON" | "DROPDOWN";
  issueStatusActionList: IssueStatusAction[];
  extraActionList: ExtraActionOption[];
}>();

const emit = defineEmits<{
  (event: "apply-issue-action", action: IssueStatusAction): void;
  (
    event: "apply-batch-task-action",
    action: TaskRolloutAction,
    targets: Task[]
  ): void;
}>();

const { issue } = useIssueContext();

const issueStatusActionDropdownOptions = computed(() => {
  return props.issueStatusActionList.map<ExtraActionOption>((action) => {
    return {
      key: action,
      label: issueStatusActionDisplayName(action),
      type: "ISSUE",
      action: action,
      target: issue.value,
    };
  });
});
const mergedDropdownActionList = computed(() => {
  if (issueStatusActionDropdownOptions.value.length > 0) {
    // When there are something to do with tasks, they will be shown as big
    // buttons.
    // Now we display issue-level actions as a dropdown together with "extra"
    // actions.
    return [
      ...issueStatusActionDropdownOptions.value,
      ...props.extraActionList,
    ];
  } else {
    // When we have nothing to do with tasks, show issue-level actions as big
    // buttons. And show only "extra" actions as a dropdown.
    return [...props.extraActionList];
  }
});

const allowApplyIssueStatusAction = (action: IssueStatusAction): boolean => {
  // TODO: permission check
  return true;
};

const handleDropdownSelect = (key: string, dropdownOption: DropdownOption) => {
  const option = dropdownOption as ExtraActionOption;
  if (option.type === "ISSUE") {
    emit("apply-issue-action", option.action as IssueStatusAction);
  }
  if (option.type === "TASK-BATCH") {
    emit(
      "apply-batch-task-action",
      option.action as TaskRolloutAction,
      option.target as Task[]
    );
  }
};
</script>
