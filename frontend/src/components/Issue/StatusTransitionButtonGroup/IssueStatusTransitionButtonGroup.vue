<template>
  <div v-if="displayMode === 'BUTTON'" class="flex items-center gap-x-2">
    <button
      v-for="(transition, index) in issueStatusTransitionList"
      :key="index"
      type="button"
      :class="transition.buttonClass"
      :disabled="!allowIssueStatusTransition(transition)"
      @click.prevent="$emit('apply-issue-transition', transition)"
    >
      {{ $t(transition.buttonName) }}
    </button>
    <NDropdown
      v-if="extraActionList.length > 0"
      trigger="click"
      placement="bottom-end"
      :options="extraActionList"
      @select="handleDropdownSelect"
    >
      <button
        id="user-menu"
        type="button"
        class="text-control-light p-0.5 rounded hover:bg-control-bg-hover"
        aria-label="User menu"
        aria-haspopup="true"
      >
        <heroicons-solid:dots-vertical class="w-6 h-6" />
      </button>
    </NDropdown>
  </div>

  <NDropdown
    v-if="displayMode === 'DROPDOWN' && mergedDropdownActionList.length > 0"
    trigger="click"
    placement="bottom-end"
    :options="mergedDropdownActionList"
    @select="handleDropdownSelect"
  >
    <button
      id="user-menu"
      type="button"
      class="text-control-light p-0.5 rounded hover:bg-control-bg-hover"
      aria-label="User menu"
      aria-haspopup="true"
    >
      <heroicons-solid:dots-vertical class="w-6 h-6" />
    </button>
  </NDropdown>
</template>

<script setup lang="ts">
import { DropdownOption } from "naive-ui";
import { Ref, computed } from "vue";
import { useI18n } from "vue-i18n";
import { Issue, IssueStatusTransition, Task } from "@/types";
import { TaskStatusTransition } from "@/utils";
import { useIssueLogic } from "../logic";
import { ExtraActionOption, IssueContext } from "./common";

const props = defineProps<{
  issueContext: IssueContext;
  displayMode: "BUTTON" | "DROPDOWN";
  issueStatusTransitionList: IssueStatusTransition[];
  extraActionList: ExtraActionOption[];
}>();

const emit = defineEmits<{
  (event: "apply-issue-transition", transition: IssueStatusTransition): void;
  (
    event: "apply-batch-task-transition",
    transition: TaskStatusTransition,
    targets: Task[]
  ): void;
}>();

const { t } = useI18n();
const issueLogic = useIssueLogic();
const issue = issueLogic.issue as Ref<Issue>;

const issueStatusTransitionDropdownOptions = computed(() => {
  return props.issueStatusTransitionList.map<ExtraActionOption>(
    (transition) => {
      return {
        key: transition.type,
        label: t(transition.buttonName),
        type: "ISSUE",
        transition,
        target: issue.value,
      };
    }
  );
});
const mergedDropdownActionList = computed(() => {
  if (issueStatusTransitionDropdownOptions.value.length > 0) {
    // When there are something to do with tasks, they will be shown as big
    // buttons.
    // Now we display issue-level actions as a dropdown together with "extra"
    // actions.
    return [
      ...issueStatusTransitionDropdownOptions.value,
      ...props.extraActionList,
    ];
  } else {
    // When we have nothing to do with tasks, show issue-level actions as big
    // buttons. And show only "extra" actions as a dropdown.
    return [...props.extraActionList];
  }
});

const allowIssueStatusTransition = (
  transition: IssueStatusTransition
): boolean => {
  if (transition.type == "RESOLVE") {
    const template = issueLogic.template.value;
    // Returns false if any of the required output fields is not provided.
    for (let i = 0; i < template.outputFieldList.length; i++) {
      const field = template.outputFieldList[i];
      if (!field.resolved(props.issueContext)) {
        return false;
      }
    }
    return true;
  }
  return true;
};

const handleDropdownSelect = (key: string, dropdownOption: DropdownOption) => {
  const option = dropdownOption as ExtraActionOption;
  if (option.type === "ISSUE") {
    emit("apply-issue-transition", option.transition as IssueStatusTransition);
  }
  if (option.type === "TASK-BATCH") {
    emit(
      "apply-batch-task-transition",
      option.transition as TaskStatusTransition,
      option.target as Task[]
    );
  }
};
</script>
