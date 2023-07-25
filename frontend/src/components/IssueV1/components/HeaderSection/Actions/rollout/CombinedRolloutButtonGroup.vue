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
      :display-mode="
        primaryTaskRolloutActionList.length > 0 ? 'DROPDOWN' : 'BUTTON'
      "
      :issue-status-action-list="issueStatusActionList"
      :extra-action-list="extraActionList"
      @perform-batch-task-action="performBatchTaskAction"
    />
  </div>

  <div class="issue-debug">
    <div>taskRolloutActionList: {{ taskRolloutActionList }}</div>
    <div>
      stageRolloutActionList:
      {{
        stageRolloutActionList.map(
          ({ action, tasks }) => `${action}(${tasks.map((t) => t.uid)})`
        )
      }}
    </div>
    <div>primaryTaskRolloutActionList: {{ primaryTaskRolloutActionList }}</div>
    <div>issueStatusActionList: {{ issueStatusActionList }}</div>
  </div>
</template>

<script setup lang="ts">
import { computed } from "vue";
import { useI18n } from "vue-i18n";

import {
  PrimaryTaskRolloutActionList,
  TaskRolloutAction,
  getApplicableIssueStatusActionList,
  getApplicableStageRolloutActionList,
  getApplicableTaskRolloutActionList,
  useIssueContext,
} from "@/components/IssueV1/logic";
import RolloutActionButtonGroup from "./RolloutActionButtonGroup.vue";
import { ExtraActionOption, IssueStatusActionButtonGroup } from "../common";
import { RolloutAction } from "./common";
import { Task } from "@/types/proto/v1/rollout_service";

const { t } = useI18n();
const { issue, activeStage, activeTask } = useIssueContext();

const issueStatusActionList = computed(() => {
  return getApplicableIssueStatusActionList(issue.value);
});

const taskRolloutActionList = computed(() => {
  return getApplicableTaskRolloutActionList(issue.value, activeTask.value);
});

const stageRolloutActionList = computed(() => {
  return getApplicableStageRolloutActionList(
    issue.value,
    activeStage.value,
    false /* !allowSkipPendingTasks */
  );
});

const primaryTaskRolloutActionList = computed(() => {
  return taskRolloutActionList.value.filter((action) =>
    PrimaryTaskRolloutActionList.includes(action)
  );
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
    });
  }
  return list;
});

const performRolloutAction = async (params: RolloutAction) => {
  const { action, target } = params;
  if (target === "TASK") {
    return performBatchTaskAction(action, [activeTask.value]);
  }
  if (target === "STAGE") {
    const actionItem = stageRolloutActionList.value.find(
      (item) => item.action === action
    );
    if (actionItem) {
      return performBatchTaskAction(action, actionItem.tasks);
    }
  }
};

const performBatchTaskAction = async (
  action: TaskRolloutAction,
  tasks: Task[]
) => {
  alert(
    `performBatchTaskAction: action=${action}, tasks=${tasks.map((t) => t.uid)}`
  );
};
</script>
