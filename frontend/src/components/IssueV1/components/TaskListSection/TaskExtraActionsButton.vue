<template>
  <NDropdown
    v-if="options.length > 0"
    trigger="click"
    placement="bottom-end"
    :options="options"
    :render-option="renderOption"
    @select="handleSelect"
  >
    <NButton
      quaternary
      size="tiny"
      style="--n-padding: 0 1px; --n-icon-size: 20px"
      @click="($event) => $event.stopPropagation()"
    >
      <template #icon>
        <heroicons:ellipsis-vertical-solid class="w-5 h-5" />
      </template>
    </NButton>
  </NDropdown>
</template>

<script setup lang="ts">
import { asyncComputed } from "@vueuse/core";
import type { DropdownOption } from "naive-ui";
import { NDropdown, NButton } from "naive-ui";
import type { VNode } from "vue";
import { computed, h } from "vue";
import { useI18n } from "vue-i18n";
import type { TaskRolloutAction } from "@/components/IssueV1/logic";
import {
  allowUserToApplyTaskRolloutAction,
  getApplicableTaskRolloutActionList,
  useIssueContext,
} from "@/components/IssueV1/logic";
import type { Task } from "@/types/proto/v1/rollout_service";
import { DropdownItemWithErrorList } from "../common";

type ExtraTaskRolloutActionDropdownOption = DropdownOption & {
  action: TaskRolloutAction;
  tasks: Task[];
};

const props = defineProps<{
  task: Task;
}>();

const { t } = useI18n();
const { events, isCreating, selectedTask, issue, releaserCandidates } =
  useIssueContext();

const actionList = computed(() => {
  return getApplicableTaskRolloutActionList(
    issue.value,
    props.task,
    true /* allowSkipPendingTask */
  );
});

const allowUserToSkipTask = asyncComputed(() => {
  return allowUserToApplyTaskRolloutAction(
    issue.value,
    props.task,
    "SKIP",
    releaserCandidates.value
  );
});

const options = computed((): ExtraTaskRolloutActionDropdownOption[] => {
  if (isCreating.value) {
    return [];
  }
  const { task } = props;
  if (task.name !== selectedTask.value.name) {
    return [];
  }
  const SKIP = actionList.value.includes("SKIP");
  if (!SKIP) {
    return [];
  }
  return [
    {
      key: "skip",
      label: t("task.skip"),
      action: "SKIP",
      tasks: [task],
      disabled: !allowUserToSkipTask.value,
    },
  ];
});

const renderOption = ({
  node,
  option,
}: {
  node: VNode;
  option: DropdownOption;
}) => {
  const errors = option.disabled
    ? [t("issue.error.you-are-not-allowed-to-perform-this-action")]
    : [];
  return h(
    DropdownItemWithErrorList,
    { errors },
    {
      default: () => node,
    }
  );
};

const handleSelect = (key: string) => {
  const option = options.value.find((opt) => opt.key === key);
  if (!option) return;
  const { action, tasks } = option;
  events.emit("perform-task-rollout-action", {
    action,
    tasks,
  });
};
</script>
