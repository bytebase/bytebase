<template>
  <div class="flex gap-x-3">
    <RolloutActionButtonGroup
      v-if="primaryTaskRolloutActionList.length > 0"
      :task-rollout-action-list="primaryTaskRolloutActionList"
      :stage-rollout-action-list="
        stageRolloutActionList.map((item) => item.action)
      "
      @perform-action="performRolloutAction"
    />

    <IssueStatusActionButtonGroup
      :display-mode="issueStatusButtonsDisplayMode"
      :issue-status-action-list="issueStatusActionList"
      :extra-action-list="extraActionList"
    />
  </div>
</template>

<script setup lang="ts">
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import type { TaskRolloutAction } from "@/components/IssueV1/logic";
import {
  getApplicableIssueStatusActionList,
  getApplicableStageRolloutActionList,
  getApplicableTaskRolloutActionList,
  PrimaryTaskRolloutActionList,
  useIssueContext,
} from "@/components/IssueV1/logic";
import { canRolloutTasks } from "@/components/RolloutV1/components/taskPermissions";
import type { Task } from "@/types/proto-es/v1/rollout_service_pb";
import type { ExtraActionOption } from "../common";
import { IssueStatusActionButtonGroup } from "../common";
import type { RolloutAction } from "./common";
import RolloutActionButtonGroup from "./RolloutActionButtonGroup.vue";

const { t } = useI18n();
const { issue, selectedStage, selectedTask, events } = useIssueContext();

const issueStatusActionList = computed(() => {
  return getApplicableIssueStatusActionList(issue.value);
});

const taskRolloutActionList = computed(() => {
  return getApplicableTaskRolloutActionList(issue.value, selectedTask.value);
});

const stageRolloutActionList = computed(() => {
  return getApplicableStageRolloutActionList(
    issue.value,
    selectedStage.value,
    false /* !allowSkipPendingTasks */
  );
});

const primaryTaskRolloutActionList = computed(() => {
  return taskRolloutActionList.value.filter((action) =>
    PrimaryTaskRolloutActionList.includes(action)
  );
});

const allowUserToSkipTask = computed(() => {
  const skip = stageRolloutActionList.value.find(
    (item) => item.action === "SKIP"
  );
  if (!skip) return false;
  return canRolloutTasks(skip.tasks, issue.value);
});

const extraActionList = computed(() => {
  const list: ExtraActionOption[] = [];
  const skip = stageRolloutActionList.value.find(
    (item) => item.action === "SKIP"
  );
  if (skip) {
    list.push({
      label: t("task.skip-failed-in-current-stage"),
      key: "skip-failed-tasks-in-current-stage",
      type: "TASK-BATCH",
      action: "SKIP",
      target: skip.tasks,
      disabled: !allowUserToSkipTask.value,
    });
  }
  return list;
});

const issueStatusButtonsDisplayMode = computed(() => {
  return primaryTaskRolloutActionList.value.length > 0 ? "DROPDOWN" : "BUTTON";
});

const performRolloutAction = async (params: RolloutAction) => {
  const { action, target } = params;
  if (target === "TASK") {
    return performBatchTaskAction(action, [selectedTask.value]);
  }
  if (target === "STAGE") {
    return performBatchTaskAction(action, selectedStage.value.tasks);
  }
};

const performBatchTaskAction = async (
  action: TaskRolloutAction,
  tasks: Task[]
) => {
  events.emit("perform-task-rollout-action", { action, tasks });
};
</script>
