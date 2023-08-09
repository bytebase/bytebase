<template>
  <ContextMenuButton
    v-if="actionList.length > 0"
    :action-list="actionList"
    :disabled="hasRunningTaskCheck"
    preference-key="issue.task.run-checks"
    default-action-key="RUN-CHECKS"
    @click="$emit('run-checks', ($event as ButtonAction).params.taskList)"
  >
    <template #icon>
      <BBSpin v-if="hasRunningTaskCheck" class="w-4 h-4" />
      <heroicons-outline:play v-else class="w-4 h-4" />
    </template>
    <template #default="{ action }">
      <template v-if="hasRunningTaskCheck">
        {{ $t("task.checking") }}
      </template>
      <template v-else>
        {{ action.text }}
      </template>
    </template>
  </ContextMenuButton>
</template>

<script setup lang="ts">
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import { Issue, Task } from "@/types";
import { ContextMenuButton, ContextMenuButtonAction } from "../v2";
import { useIssueLogic } from "./logic";

type ButtonAction = ContextMenuButtonAction<{
  taskList: Task[];
}>;

defineEmits<{
  (event: "run-checks", taskList: Task[]): void;
}>();

const { t } = useI18n();
const context = useIssueLogic();

const allowRunCheckForIssue = computed(() => {
  if (context.create.value) {
    return false;
  }
  const issue = context.issue.value as Issue;
  if (issue.status !== "OPEN") {
    return false;
  }
  return true;
});

const actionList = computed(() => {
  if (!allowRunCheckForIssue.value) return [];

  const actionList: ButtonAction[] = [];
  const selectedTask = context.selectedTask.value as Task;
  if (allowRunChecksForTask(selectedTask)) {
    actionList.push({
      key: "RUN-CHECKS",
      text: t("task.run-checks"),
      params: {
        taskList: [selectedTask],
      },
    });

    // Don't only show 'run checks in current stage' if we don't show 'run checks'
    // since that might be weird.
    const taskListInStage = context.selectedStage.value.taskList as Task[];
    const runnableTaskList = taskListInStage.filter((task) =>
      allowRunChecksForTask(task)
    );
    if (runnableTaskList.length > 1) {
      actionList.push({
        key: "RUN-CHECKS-IN-CURRENT-STAGE",
        text: t("task.run-checks-in-current-stage"),
        params: {
          taskList: runnableTaskList,
        },
      });
    }
  }
  return actionList;
});

const hasRunningTaskCheck = computed((): boolean => {
  if (context.create.value) return false;

  const selectedTask = context.selectedTask.value as Task;

  return selectedTask.taskCheckRunList.some(
    (checkRun) => checkRun.status === "RUNNING"
  );
});

const allowRunChecksForTask = (task: Task) => {
  return (
    task.status == "PENDING" ||
    task.status == "PENDING_APPROVAL" ||
    task.status == "RUNNING" ||
    task.status == "FAILED"
  );
};
</script>
