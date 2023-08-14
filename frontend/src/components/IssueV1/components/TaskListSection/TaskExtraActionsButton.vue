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
    >
      <template #icon>
        <heroicons:ellipsis-vertical-solid class="w-5 h-5" />
      </template>
    </NButton>
  </NDropdown>
</template>

<script setup lang="ts">
import { asyncComputed } from "@vueuse/core";
import { DropdownOption, NDropdown, NButton } from "naive-ui";
import { VNode, computed, h } from "vue";
import { useI18n } from "vue-i18n";
import {
  TaskRolloutAction,
  allowUserToApplyTaskRolloutAction,
  getApplicableTaskRolloutActionList,
  useIssueContext,
} from "@/components/IssueV1/logic";
import { useCurrentUserV1 } from "@/store";
import { Task } from "@/types/proto/v1/rollout_service";
import { DropdownItemWithErrorList } from "../common";

type ExtraTaskRolloutActionDropdownOption = DropdownOption & {
  action: TaskRolloutAction;
  tasks: Task[];
};

const props = defineProps<{
  task: Task;
}>();

const { t } = useI18n();
const currentUser = useCurrentUserV1();
const { events, isCreating, activeTask, issue } = useIssueContext();

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
    currentUser.value,
    "SKIP"
  );
});

const options = computed((): ExtraTaskRolloutActionDropdownOption[] => {
  if (isCreating.value) {
    return [];
  }
  const { task } = props;
  if (task.uid !== activeTask.value.uid) {
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
    ? ["You are not allowed to perform this action"]
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
