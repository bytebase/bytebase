<template>
  <NDropdown
    v-if="options.length > 0"
    trigger="click"
    placement="bottom-end"
    :options="options"
    @select="handleSelect"
  >
    <NButton
      quaternary
      size="tiny"
      style="--n-padding: 0 1px; --n-icon-size: 20px"
    >
      <template #icon>
        <heroicons:ellipsis-vertical-solid class="w-5 h-5" />
      </template>
    </NButton>
  </NDropdown>
</template>

<script setup lang="ts">
import { computed } from "vue";
import { DropdownOption, NDropdown, NButton } from "naive-ui";
import { useI18n } from "vue-i18n";

import { Task } from "@/types/proto/v1/rollout_service";
import {
  TaskRolloutAction,
  getApplicableTaskRolloutActionList,
  useIssueContext,
} from "@/components/IssueV1/logic";

type ExtraTaskRolloutActionDropdownOption = DropdownOption & {
  action: TaskRolloutAction;
  tasks: Task[];
};

const props = defineProps<{
  task: Task;
}>();

const { t } = useI18n();
const { events, isCreating, activeTask, issue } = useIssueContext();

const actionList = computed(() => {
  return getApplicableTaskRolloutActionList(
    issue.value,
    props.task,
    true /* allowSkipPendingTask */
  );
});

const options = computed((): ExtraTaskRolloutActionDropdownOption[] => {
  if (isCreating.value) {
    return [];
  }
  if (props.task.uid !== activeTask.value.uid) {
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
      tasks: [props.task],
    },
  ];
});

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
