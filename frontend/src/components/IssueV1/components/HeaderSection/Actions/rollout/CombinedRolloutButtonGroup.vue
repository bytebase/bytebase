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
import { asyncComputed } from "@vueuse/core";

import {
  PrimaryTaskRolloutActionList,
  TaskRolloutAction,
  allowUserToApplyTaskRolloutAction,
  getApplicableIssueStatusActionList,
  getApplicableStageRolloutActionList,
  getApplicableTaskRolloutActionList,
  useIssueContext,
} from "@/components/IssueV1/logic";
import { Task } from "@/types/proto/v1/rollout_service";
import { useCurrentUserV1 } from "@/store";
import RolloutActionButtonGroup from "./RolloutActionButtonGroup.vue";
import { ExtraActionOption, IssueStatusActionButtonGroup } from "../common";
import { RolloutAction } from "./common";

const { t } = useI18n();
const currentUser = useCurrentUserV1();
const { issue, activeStage, activeTask, events } = useIssueContext();

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

const allowUserToSkipTask = asyncComputed(async () => {
  const skip = stageRolloutActionList.value.find(
    (item) => item.action === "SKIP"
  );
  if (!skip) return false;
  const allowed = await Promise.all(
    skip.tasks.map((task) =>
      allowUserToApplyTaskRolloutAction(
        issue.value,
        task,
        currentUser.value,
        "SKIP"
      )
    )
  );
  return allowed.every((allow) => allow);
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
  events.emit("perform-task-rollout-action", { action, tasks });
};
</script>
